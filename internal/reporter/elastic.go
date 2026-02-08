package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

var _ Storage = (*ElasticStore)(nil)

type ElasticConfig struct {
	Addresses []string
	Username  string
	Password  string
	CloudID   string
	APIKey    string
	Index     string
}

type ElasticStore struct {
	client    *elasticsearch.Client
	indexName string
	mu        sync.RWMutex
}

func NewElasticStore(cfg ElasticConfig) (*ElasticStore, error) {
	esCfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
		CloudID:   cfg.CloudID,
		APIKey:    cfg.APIKey,
	}

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	res, err := client.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch ping failed: %s", res.Status())
	}

	indexName := cfg.Index
	if indexName == "" {
		indexName = "kubecrsh-reports"
	}

	store := &ElasticStore{
		client:    client,
		indexName: indexName,
	}

	if err := store.ensureIndex(); err != nil {
		return nil, fmt.Errorf("failed to ensure index: %w", err)
	}

	return store, nil
}

func (s *ElasticStore) ensureIndex() error {
	res, err := s.client.Indices.Exists([]string{s.indexName})
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		return nil
	}

	mapping := `{
		"mappings": {
			"properties": {
				"id": {"type": "keyword"},
				"crash": {
					"properties": {
						"namespace": {"type": "keyword"},
						"pod_name": {"type": "keyword"},
						"container_name": {"type": "keyword"},
						"exit_code": {"type": "integer"},
						"reason": {"type": "keyword"},
						"signal": {"type": "integer"},
						"restart_count": {"type": "integer"},
						"started_at": {"type": "date"},
						"finished_at": {"type": "date"}
					}
				},
				"logs": {"type": "text"},
				"previous_log": {"type": "text"},
				"events": {
					"type": "nested",
					"properties": {
						"type": {"type": "keyword"},
						"reason": {"type": "keyword"},
						"message": {"type": "text"},
						"count": {"type": "integer"},
						"first_seen": {"type": "date"},
						"last_seen": {"type": "date"},
						"source": {"type": "keyword"}
					}
				},
				"env_vars": {"type": "object", "enabled": false},
				"warnings": {"type": "text"},
				"collected_at": {"type": "date"}
			}
		}
	}`

	createRes, err := s.client.Indices.Create(
		s.indexName,
		s.client.Indices.Create.WithBody(strings.NewReader(mapping)),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		return fmt.Errorf("failed to create index: %s", createRes.Status())
	}

	return nil
}

func (s *ElasticStore) Save(report *domain.ForensicReport) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc := s.toDocument(report)

	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	req := esapi.IndexRequest{
		Index:      s.indexName,
		DocumentID: report.ID,
		Body:       bytes.NewReader(data),
		Refresh:    "false",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := req.Do(ctx, s.client)
	if err != nil {
		return fmt.Errorf("failed to index report: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to index report: %s", res.Status())
	}

	return nil
}

func (s *ElasticStore) Load(id string) (*domain.ForensicReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := s.client.Get(s.indexName, id, s.client.Get.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get report: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, fmt.Errorf("report not found: %s", id)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get report: %s", res.Status())
	}

	var result struct {
		Source elasticDocument `json:"_source"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return s.fromDocument(&result.Source), nil
}

func (s *ElasticStore) List() ([]*domain.ForensicReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `{
		"query": {"match_all": {}},
		"size": 1000,
		"sort": [{"collected_at": {"order": "desc"}}]
	}`

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := s.client.Search(
		s.client.Search.WithContext(ctx),
		s.client.Search.WithIndex(s.indexName),
		s.client.Search.WithBody(strings.NewReader(query)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search reports: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search failed: %s", res.Status())
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source elasticDocument `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	reports := make([]*domain.ForensicReport, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		reports = append(reports, s.fromDocument(&hit.Source))
	}

	return reports, nil
}

type elasticDocument struct {
	ID          string            `json:"id"`
	Crash       elasticCrash      `json:"crash"`
	Logs        []string          `json:"logs"`
	PreviousLog []string          `json:"previous_log"`
	Events      []elasticEvent    `json:"events"`
	EnvVars     map[string]string `json:"env_vars"`
	Warnings    []string          `json:"warnings"`
	CollectedAt time.Time         `json:"collected_at"`
}

type elasticCrash struct {
	Namespace     string    `json:"namespace"`
	PodName       string    `json:"pod_name"`
	ContainerName string    `json:"container_name"`
	ExitCode      int32     `json:"exit_code"`
	Reason        string    `json:"reason"`
	Signal        int32     `json:"signal"`
	RestartCount  int32     `json:"restart_count"`
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    time.Time `json:"finished_at"`
}

type elasticEvent struct {
	Type      string    `json:"type"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Count     int32     `json:"count"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	Source    string    `json:"source"`
}

func (s *ElasticStore) toDocument(report *domain.ForensicReport) *elasticDocument {
	events := make([]elasticEvent, 0, len(report.Events))
	for _, e := range report.Events {
		events = append(events, elasticEvent{
			Type:      e.Type,
			Reason:    e.Reason,
			Message:   e.Message,
			Count:     e.Count,
			FirstSeen: e.FirstSeen,
			LastSeen:  e.LastSeen,
			Source:    e.Source,
		})
	}

	return &elasticDocument{
		ID: report.ID,
		Crash: elasticCrash{
			Namespace:     report.Crash.Namespace,
			PodName:       report.Crash.PodName,
			ContainerName: report.Crash.ContainerName,
			ExitCode:      report.Crash.ExitCode,
			Reason:        report.Crash.Reason,
			Signal:        report.Crash.Signal,
			RestartCount:  report.Crash.RestartCount,
			StartedAt:     report.Crash.StartedAt,
			FinishedAt:    report.Crash.FinishedAt,
		},
		Logs:        report.Logs,
		PreviousLog: report.PreviousLog,
		Events:      events,
		EnvVars:     report.EnvVars,
		Warnings:    report.Warnings,
		CollectedAt: report.CollectedAt,
	}
}

func (s *ElasticStore) fromDocument(doc *elasticDocument) *domain.ForensicReport {
	events := make([]domain.Event, 0, len(doc.Events))
	for _, e := range doc.Events {
		events = append(events, domain.Event{
			Type:      e.Type,
			Reason:    e.Reason,
			Message:   e.Message,
			Count:     e.Count,
			FirstSeen: e.FirstSeen,
			LastSeen:  e.LastSeen,
			Source:    e.Source,
		})
	}

	return &domain.ForensicReport{
		ID: doc.ID,
		Crash: domain.PodCrash{
			Namespace:     doc.Crash.Namespace,
			PodName:       doc.Crash.PodName,
			ContainerName: doc.Crash.ContainerName,
			ExitCode:      doc.Crash.ExitCode,
			Reason:        doc.Crash.Reason,
			Signal:        doc.Crash.Signal,
			RestartCount:  doc.Crash.RestartCount,
			StartedAt:     doc.Crash.StartedAt,
			FinishedAt:    doc.Crash.FinishedAt,
		},
		Logs:        doc.Logs,
		PreviousLog: doc.PreviousLog,
		Events:      events,
		EnvVars:     doc.EnvVars,
		Warnings:    doc.Warnings,
		CollectedAt: doc.CollectedAt,
	}
}
