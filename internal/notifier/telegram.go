package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

const (
	telegramAPIBaseURL             = "https://api.telegram.org"
	maxTelegramResponseBytes int64 = 1 << 20
)

type TelegramNotifier struct {
	baseURL string
	token   string
	chatID  string
	client  *http.Client
}

type telegramSendMessageRequest struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

type telegramResponse struct {
	OK          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code,omitempty"`
	Description string `json:"description,omitempty"`
	Parameters  struct {
		RetryAfter int `json:"retry_after,omitempty"`
	} `json:"parameters,omitempty"`
}

func NewTelegramNotifier(baseURL *string, token, chatID string) *TelegramNotifier {
	if baseURL == nil || strings.TrimSpace(*baseURL) == "" {
		baseURL = new(string)
		*baseURL = telegramAPIBaseURL
	}
	return &TelegramNotifier{
		baseURL: strings.TrimRight(*baseURL, "/"),
		chatID:  chatID,
		token:   token,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *TelegramNotifier) Notify(report domain.ForensicReport) error {
	msg := telegramSendMessageRequest{
		ChatID: s.chatID,
		Text: fmt.Sprintf(
			"Pod crash detected: %s\nNamespace: %s\nPod: %s\nContainer: %s\nReason: %s\nExit code: %d\nRestart count: %d\nReport ID: %s\nCollected: %s",
			report.Summary(),
			report.Crash.Namespace,
			report.Crash.PodName,
			report.Crash.ContainerName,
			report.Crash.Reason,
			report.Crash.ExitCode,
			report.Crash.RestartCount,
			report.ID,
			report.CollectedAt.Format("2006-01-02 15:04:05"),
		),
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram message: %w", err)
	}

	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", s.baseURL, s.token)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send telegram notification: %w", err)
		} else {
			respBody, readErr := readAndClose(resp)
			if readErr != nil {
				lastErr = fmt.Errorf("failed to read telegram response: %w", readErr)
			} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				var tr telegramResponse
				if len(respBody) > 0 && json.Unmarshal(respBody, &tr) == nil && !tr.OK {
					return fmt.Errorf("telegram rejected request: %s (code=%d)", strings.TrimSpace(tr.Description), tr.ErrorCode)
				}
				return nil
			} else {
				desc, retryAfter := parseTelegramError(resp, respBody)
				lastErr = fmt.Errorf("telegram returned status %d: %s", resp.StatusCode, desc)

				if resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
					return lastErr
				}

				if attempt < 2 {
					if resp.StatusCode == http.StatusTooManyRequests && retryAfter > 0 {
						time.Sleep(retryAfter)
					} else {
						time.Sleep(backoff(attempt))
					}
				}
				continue
			}
		}

		if attempt < 2 {
			time.Sleep(backoff(attempt))
		}
	}

	return lastErr
}

func (s *TelegramNotifier) Name() string {
	return "telegram"
}

func readAndClose(resp *http.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, nil
	}
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxTelegramResponseBytes))
	closeErr := resp.Body.Close()
	if readErr != nil && closeErr != nil {
		return nil, fmt.Errorf("read: %v; close: %v", readErr, closeErr)
	}
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return body, nil
}

func parseTelegramError(resp *http.Response, body []byte) (string, time.Duration) {
	desc := "unknown error"
	var retryAfter time.Duration

	if resp != nil {
		if h := strings.TrimSpace(resp.Header.Get("Retry-After")); h != "" {
			if seconds, err := strconv.Atoi(h); err == nil && seconds > 0 {
				retryAfter = time.Duration(seconds) * time.Second
			}
		}
	}

	var tr telegramResponse
	if len(body) > 0 && json.Unmarshal(body, &tr) == nil {
		if strings.TrimSpace(tr.Description) != "" {
			desc = strings.TrimSpace(tr.Description)
		}
		if tr.Parameters.RetryAfter > 0 {
			retryAfter = time.Duration(tr.Parameters.RetryAfter) * time.Second
		}
		return desc, retryAfter
	}

	if len(body) > 0 {
		desc = strings.TrimSpace(string(body))
		if desc == "" {
			desc = "unknown error"
		}
	}

	return desc, retryAfter
}
