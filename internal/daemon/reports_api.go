package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

type reportSummary struct {
	ID          string    `json:"id"`
	Namespace   string    `json:"namespace"`
	PodName     string    `json:"podName"`
	Container   string    `json:"container"`
	Reason      string    `json:"reason"`
	CollectedAt time.Time `json:"collectedAt"`
	Warnings    int       `json:"warnings"`
	HasLogs     bool      `json:"hasLogs"`
	HasPrevLogs bool      `json:"hasPreviousLogs"`
	HasEvents   bool      `json:"hasEvents"`
}

func (s *Server) reportsListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !s.authorize(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	limit := parseIntQuery(r, "limit", 200)
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}

	offset := parseIntQuery(r, "offset", 0)
	if offset < 0 {
		offset = 0
	}

	reports, err := s.store.List()
	if err != nil {
		http.Error(w, "failed to list reports", http.StatusInternalServerError)
		fmt.Printf("Failed to list reports: %v\n", err)
		return
	}

	sort.Slice(reports, func(i, j int) bool {
		return reports[i].CollectedAt.After(reports[j].CollectedAt)
	})

	if offset >= len(reports) {
		writeJSON(w, http.StatusOK, map[string]any{"items": []reportSummary{}, "total": len(reports)})
		return
	}

	end := offset + limit
	if end > len(reports) {
		end = len(reports)
	}

	items := make([]reportSummary, 0, end-offset)
	for _, rep := range reports[offset:end] {
		items = append(items, summarizeReport(rep))
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(reports)})
}

func (s *Server) reportGetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !s.authorize(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/reports/")
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	rep, err := s.store.Load(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	full := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("full")), "true") || r.URL.Query().Get("full") == "1"
	if full && !s.apiAllowFull {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if full {
		writeJSON(w, http.StatusOK, rep)
		return
	}

	writeJSON(w, http.StatusOK, summarizeReport(rep))
}

func (s *Server) authorize(r *http.Request) bool {
	if strings.TrimSpace(s.apiToken) == "" {
		return true
	}

	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		return false
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return false
	}

	return strings.TrimPrefix(auth, prefix) == s.apiToken
}

func summarizeReport(r *domain.ForensicReport) reportSummary {
	if r == nil {
		return reportSummary{}
	}

	return reportSummary{
		ID:          r.ID,
		Namespace:   r.Crash.Namespace,
		PodName:     r.Crash.PodName,
		Container:   r.Crash.ContainerName,
		Reason:      r.Crash.Reason,
		CollectedAt: r.CollectedAt,
		Warnings:    len(r.Warnings),
		HasLogs:     len(r.Logs) > 0,
		HasPrevLogs: len(r.PreviousLog) > 0,
		HasEvents:   len(r.Events) > 0,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	if err := enc.Encode(v); err != nil {
		fmt.Printf("Failed to write JSON response: %v\n", err)
	}
}

func parseIntQuery(r *http.Request, key string, def int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return def
	}
	i, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return i
}
