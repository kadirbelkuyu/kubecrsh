package notifier

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const maxDrainBytes int64 = 1 << 20

func backoff(attempt int) time.Duration {
	d := 200 * time.Millisecond
	for i := 0; i < attempt; i++ {
		d *= 2
		if d >= 2*time.Second {
			return 2 * time.Second
		}
	}
	return d
}

func drainAndClose(resp *http.Response) error {
	if resp == nil || resp.Body == nil {
		return nil
	}
	_, copyErr := io.Copy(io.Discard, io.LimitReader(resp.Body, maxDrainBytes))
	closeErr := resp.Body.Close()
	if copyErr != nil && closeErr != nil {
		return fmt.Errorf("copy: %v; close: %v", copyErr, closeErr)
	}
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
