package dashboard

import (
	"sort"
	"sync"
	"time"

	"github.com/spw-m-riley/ma/internal/app"
)

const defaultRecentRunLimit = 32

type RunsSnapshot struct {
	Runs []RunView `json:"runs"`
}

type RunView struct {
	ID            string     `json:"id"`
	Command       string     `json:"command"`
	Status        string     `json:"status"`
	StartedAt     time.Time  `json:"startedAt"`
	FinishedAt    *time.Time `json:"finishedAt,omitempty"`
	Success       bool       `json:"success"`
	Changed       bool       `json:"changed"`
	Error         string     `json:"error,omitempty"`
	PayloadStatus string     `json:"payloadStatus,omitempty"`
	ResultSummary string     `json:"resultSummary,omitempty"`
	HasDetails    bool       `json:"hasDetails"`
}

type RunDetail struct {
	RunView
	Input  string     `json:"input,omitempty"`
	Result app.Result `json:"result,omitempty"`
}

type runTracker struct {
	mu       sync.Mutex
	runs     map[string]RunDetail
	capacity int
}

func newRunTracker(capacity int) *runTracker {
	if capacity <= 0 {
		capacity = defaultRecentRunLimit
	}
	return &runTracker{
		runs:     make(map[string]RunDetail),
		capacity: capacity,
	}
}

func (t *runTracker) Apply(event RunEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	current := t.runs[event.ID]
	current.ID = event.ID
	current.Command = event.Command
	current.Status = event.Kind
	current.StartedAt = event.StartedAt
	current.FinishedAt = event.FinishedAt
	current.Success = event.Success
	current.Changed = event.Changed
	current.Error = event.Error
	current.PayloadStatus = event.PayloadStatus
	current.ResultSummary = event.ResultSummary
	current.Input = event.Input
	current.Result = event.Result
	current.HasDetails = event.PayloadStatus == payloadStatusObserved || event.PayloadStatus == payloadStatusRedacted || event.Result.Output != "" || event.Error != ""

	t.runs[event.ID] = current
	t.trim()
}

func (t *runTracker) Snapshot() RunsSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()

	runs := make([]RunView, 0, len(t.runs))
	for _, detail := range t.runs {
		runs = append(runs, detail.RunView)
	}
	sort.Slice(runs, func(i, j int) bool {
		if runs[i].StartedAt.Equal(runs[j].StartedAt) {
			return runs[i].ID > runs[j].ID
		}
		return runs[i].StartedAt.After(runs[j].StartedAt)
	})
	return RunsSnapshot{Runs: runs}
}

func (t *runTracker) Detail(id string) (RunDetail, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	detail, ok := t.runs[id]
	return detail, ok
}

func (t *runTracker) trim() {
	if len(t.runs) <= t.capacity {
		return
	}

	type item struct {
		id        string
		startedAt time.Time
	}
	items := make([]item, 0, len(t.runs))
	for id, detail := range t.runs {
		items = append(items, item{id: id, startedAt: detail.StartedAt})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].startedAt.Before(items[j].startedAt)
	})
	for len(items) > t.capacity {
		delete(t.runs, items[0].id)
		items = items[1:]
	}
}
