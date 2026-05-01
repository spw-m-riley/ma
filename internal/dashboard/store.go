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
const recentFileName = "recent.json"

type Store struct {
	root        string
	historyPath string
	recentPath  string
	mu          sync.Mutex
	history     []persistedRun
	recentRuns  []RunDetail
}

type FinishedRun struct {
	ID            string
	Command       string
	Source        string
	StartedAt     time.Time
	FinishedAt    time.Time
	Success       bool
	Changed       bool
	Error         string
	PayloadStatus string
	Input         string
	ResultSummary string
	Result        app.Result
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
	store := &Store{
		root:        root,
		historyPath: filepath.Join(root, historyFileName),
		recentPath:  filepath.Join(root, recentFileName),
	}
	history, err := store.loadRuns()
	if err != nil {
		return nil, err
	}
	recentRuns, err := store.loadRecentRuns()
	if err != nil {
		return nil, err
	}
	store.history = history
	store.recentRuns = recentRuns
	return store, nil
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

	s.history = append(s.history, record)
	if limit := historyLimit(); len(s.history) > limit {
		s.history = append([]persistedRun(nil), s.history[len(s.history)-limit:]...)
	}

	detail := RunDetail{
		RunView: RunView{
			ID:            run.ID,
			Command:       run.Command,
			Source:        run.Source,
			Status:        statusFromFinishedRun(run),
			StartedAt:     run.StartedAt,
			FinishedAt:    finishedAtPtr(run.FinishedAt),
			Success:       run.Success,
			Changed:       run.Changed || run.Result.Changed,
			Error:         run.Error,
			PayloadStatus: run.PayloadStatus,
			ResultSummary: run.ResultSummary,
			HasDetails:    run.PayloadStatus == payloadStatusObserved || run.PayloadStatus == payloadStatusRedacted || run.Result.Output != "" || run.Error != "",
		},
		Input:  run.Input,
		Result: run.Result,
	}
	s.recentRuns = append(s.recentRuns, detail)
	if limit := recentRunLimit(); len(s.recentRuns) > limit {
		s.recentRuns = append([]RunDetail(nil), s.recentRuns[len(s.recentRuns)-limit:]...)
	}

	if err := s.persistHistoryLocked(); err != nil {
		return err
	}
	if err := s.persistRecentRunsLocked(); err != nil {
		return err
	}
	return nil
}

func (s *Store) Summary() (Summary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	summary := Summary{
		CommandUsage: make(map[string]int),
	}

	for _, record := range s.history {
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

	entries := make([]HistoryEntry, 0, len(s.history))
	for _, record := range s.history {
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

func (s *Store) RecentRuns() ([]RunDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	recent := make([]RunDetail, len(s.recentRuns))
	copy(recent, s.recentRuns)
	return recent, nil
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

func (s *Store) loadRecentRuns() ([]RunDetail, error) {
	content, err := os.ReadFile(s.recentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read dashboard recent runs: %w", err)
	}
	if len(content) == 0 {
		return nil, nil
	}

	var runs []RunDetail
	if err := json.Unmarshal(content, &runs); err != nil {
		return nil, fmt.Errorf("decode dashboard recent runs: %w", err)
	}
	return runs, nil
}

func (s *Store) persistHistoryLocked() error {
	file, err := os.OpenFile(s.historyPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("open dashboard history: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, record := range s.history {
		if err := encoder.Encode(record); err != nil {
			return fmt.Errorf("write dashboard history: %w", err)
		}
	}
	return nil
}

func (s *Store) persistRecentRunsLocked() error {
	content, err := json.Marshal(s.recentRuns)
	if err != nil {
		return fmt.Errorf("marshal dashboard recent runs: %w", err)
	}
	if err := os.WriteFile(s.recentPath, append(content, '\n'), 0o644); err != nil {
		return fmt.Errorf("write dashboard recent runs: %w", err)
	}
	return nil
}

func statusFromFinishedRun(run FinishedRun) string {
	if !run.Success {
		return eventKindFailed
	}
	return eventKindFinished
}

func finishedAtPtr(value time.Time) *time.Time {
	return &value
}
