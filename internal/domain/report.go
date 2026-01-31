package domain

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type ForensicReport struct {
	ID          string
	Crash       PodCrash
	Logs        []string
	PreviousLog []string
	Events      []Event
	EnvVars     map[string]string
	Warnings    []string
	CollectedAt time.Time
}

func NewForensicReport(crash PodCrash) *ForensicReport {
	return &ForensicReport{
		ID:          generateID(),
		Crash:       crash,
		EnvVars:     make(map[string]string),
		Events:      make([]Event, 0),
		Warnings:    make([]string, 0),
		CollectedAt: time.Now(),
	}
}

func generateID() string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (r *ForensicReport) AddEvent(event Event) {
	r.Events = append(r.Events, event)
}

func (r *ForensicReport) SetLogs(logs []string) {
	r.Logs = logs
}

func (r *ForensicReport) SetPreviousLogs(logs []string) {
	r.PreviousLog = logs
}

func (r *ForensicReport) SetEnvVar(key, value string) {
	r.EnvVars[key] = value
}

func (r *ForensicReport) AddWarning(msg string) {
	if msg == "" {
		return
	}
	r.Warnings = append(r.Warnings, msg)
}

func (r *ForensicReport) WarningCount() int {
	count := 0
	for _, e := range r.Events {
		if e.IsWarning() {
			count++
		}
	}
	return count
}

func (r *ForensicReport) Summary() string {
	return r.Crash.FullName() + " - " + r.Crash.Reason
}
