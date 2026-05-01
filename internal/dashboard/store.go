package dashboard

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spw-m-riley/ma/internal/app"
)

const historyFileName = "history.jsonl"

type Store struct {
	root        string
	historyPath string
	mu          sync.Mutex
}

type FinishedRun struct {
	ID         string
	Command    string
	Source     string
	StartedAt  time.Time
	FinishedAt time.Time
	Success    bool
	Changed    bool
	Error      string
	Result     app.Result
}

type Summary struct {
	TotalRuns              int
	SuccessfulRuns         int
	FailedRuns             int
	TotalBytesSaved        int
	TotalWordsSaved        int
	TotalApproxTokensSaved int
	CommandUsage           map[string]int
}

type HistoryEntry struct {
	ID         string
	Command    string
	Source     string
	StartedAt  time.Time
	FinishedAt time.Time
	Success    bool
	Changed    bool
	Error      string
	Stats      app.Stats
}

type persistedRun struct {
	ID         string    `json:"id"`
	Command    string    `json:"command"`
	Source     string    `json:"source,omitempty"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
	Success    bool      `json:"success"`
	Changed    bool      `json:"changed"`
	Error      string    `json:"error,omitempty"`
	Stats      app.Stats `json:"stats"`
}

func OpenStore(root string) (*Store, error) {
	if root == "" {
		return nil, fmt.Errorf("dashboard store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create dashboard store root: %w", err)
	}
	return &Store{
		root:        root,
		historyPath: filepath.Join(root, historyFileName),
	}, nil
}

func (s *Store) RecordFinished(run FinishedRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := persistedRun{
		ID:         run.ID,
		Command:    run.Command,
		Source:     run.Source,
		StartedAt:  run.StartedAt,
		FinishedAt: run.FinishedAt,
		Success:    run.Success,
		Changed:    run.Changed || run.Result.Changed,
		Error:      run.Error,
		Stats:      run.Result.Stats,
	}

	file, err := os.OpenFile(s.historyPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open dashboard history: %w", err)
	}
	defer file.Close()

	encoded, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal dashboard history entry: %w", err)
	}
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("append dashboard history entry: %w", err)
	}
	return nil
}

func (s *Store) Summary() (Summary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	summary := Summary{
		CommandUsage: make(map[string]int),
	}

	records, err := s.loadRuns()
	if err != nil {
		return Summary{}, err
	}
	for _, record := range records {
		summary.TotalRuns++
		if record.Success {
			summary.SuccessfulRuns++
		} else {
			summary.FailedRuns++
		}
		summary.TotalBytesSaved += record.Stats.InputBytes - record.Stats.OutputBytes
		summary.TotalWordsSaved += record.Stats.InputWords - record.Stats.OutputWords
		summary.TotalApproxTokensSaved += record.Stats.InputApproxTokens - record.Stats.OutputApproxTokens
		summary.CommandUsage[record.Command]++
	}

	return summary, nil
}

func (s *Store) History() ([]HistoryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.loadRuns()
	if err != nil {
		return nil, err
	}

	entries := make([]HistoryEntry, 0, len(records))
	for _, record := range records {
		entries = append(entries, HistoryEntry{
			ID:         record.ID,
			Command:    record.Command,
			Source:     record.Source,
			StartedAt:  record.StartedAt,
			FinishedAt: record.FinishedAt,
			Success:    record.Success,
			Changed:    record.Changed,
			Error:      record.Error,
			Stats:      record.Stats,
		})
	}
	return entries, nil
}

func (s *Store) loadRuns() ([]persistedRun, error) {
	file, err := os.Open(s.historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open dashboard history: %w", err)
	}
	defer file.Close()

	var records []persistedRun
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record persistedRun
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, fmt.Errorf("decode dashboard history entry: %w", err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan dashboard history: %w", err)
	}
	return records, nil
}
