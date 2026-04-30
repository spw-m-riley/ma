package dashboard

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"
)

type Server struct {
	store *Store
	runs  *runTracker
}

type pageData struct {
	Title                  string
	TotalRuns              int
	SuccessfulRuns         int
	FailedRuns             int
	ActiveRuns             int
	TotalBytesSaved        int
	TotalWordsSaved        int
	TotalApproxTokensSaved int
	CommandUsage           []commandUsage
	RecentRuns             []RunView
}

type statsPageData struct {
	Title          string
	TrendRows      []trendRow
	CommandUsage   []commandUsage
	SuccessfulRuns int
	FailedRuns     int
}

type summarySnapshot struct {
	TotalRuns              int `json:"totalRuns"`
	SuccessfulRuns         int `json:"successfulRuns"`
	FailedRuns             int `json:"failedRuns"`
	TotalBytesSaved        int `json:"totalBytesSaved"`
	TotalWordsSaved        int `json:"totalWordsSaved"`
	TotalApproxTokensSaved int `json:"totalApproxTokensSaved"`
}

type overviewSnapshot struct {
	Summary      summarySnapshot `json:"summary"`
	CommandUsage []commandUsage  `json:"commandUsage"`
	ActiveRuns   int             `json:"activeRuns"`
	Runs         []RunView       `json:"runs"`
}

type statsSnapshot struct {
	TrendRows      []trendRow     `json:"trendRows"`
	CommandUsage   []commandUsage `json:"commandUsage"`
	SuccessfulRuns int            `json:"successfulRuns"`
	FailedRuns     int            `json:"failedRuns"`
}

type commandUsage struct {
	Command string `json:"command"`
	Count   int    `json:"count"`
}

type trendRow struct {
	Day               string `json:"day"`
	BytesSaved        int    `json:"bytesSaved"`
	WordsSaved        int    `json:"wordsSaved"`
	ApproxTokensSaved int    `json:"approxTokensSaved"`
}

var dashboardPageTemplate = template.Must(template.New("dashboard").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ .Title }}</title>
  <style>
    body { font-family: sans-serif; margin: 2rem; }
    .status-started { color: #1d4ed8; }
    .status-finished { color: #166534; }
    .status-failed { color: #b91c1c; }
  </style>
</head>
<body>
  <main>
    <h1>{{ .Title }}</h1>
    <section id="overview-summary">
      <h2>Total runs</h2>
      <p id="total-runs">{{ .TotalRuns }}</p>
      <p id="run-outcomes">{{ .SuccessfulRuns }} successful / {{ .FailedRuns }} failed</p>
      <p id="running-now">Running now: {{ .ActiveRuns }}</p>
      <p id="total-bytes-saved">{{ .TotalBytesSaved }} bytes saved</p>
      <p id="total-words-saved">{{ .TotalWordsSaved }} words saved</p>
      <p id="total-approx-tokens-saved">{{ .TotalApproxTokensSaved }} approx tokens saved</p>
    </section>
    <section>
      <h2>Command usage</h2>
      <ul id="command-usage">
        {{ range .CommandUsage }}
        <li>{{ .Command }}: {{ .Count }}</li>
        {{ else }}
        <li>No runs recorded yet.</li>
        {{ end }}
      </ul>
    </section>
    <section>
      <h2>Recent runs</h2>
      <ul id="recent-runs">
        {{ range .RecentRuns }}
        <li class="status-{{ .Status }}">
          <strong>{{ .Command }}</strong> — {{ .Status }}
          {{ if and .PayloadStatus (ne .PayloadStatus "none") }} ({{ .PayloadStatus }}){{ end }}
          {{ if .ResultSummary }} — {{ .ResultSummary }}{{ end }}
          {{ if .HasDetails }} — <a href="/runs/{{ .ID }}">details</a>{{ end }}
        </li>
        {{ else }}
        <li>No recent runs yet.</li>
        {{ end }}
      </ul>
    </section>
    <script>
      function renderCommandUsage(items) {
        if (!items || items.length === 0) {
          return '<li>No runs recorded yet.</li>';
        }
        return items.map(function(item) {
          return '<li>' + item.command + ': ' + item.count + '</li>';
        }).join('');
      }

      function renderRecentRuns(runs) {
        if (!runs || runs.length === 0) {
          return '<li>No recent runs yet.</li>';
        }
        return runs.map(function(run) {
          const payloadState = run.payloadStatus && run.payloadStatus !== 'none' ? ' (' + run.payloadStatus + ')' : '';
          const summary = run.resultSummary ? ' — ' + run.resultSummary : '';
          const detail = run.hasDetails ? ' — <a href="/runs/' + run.id + '">details</a>' : '';
          return '<li class="status-' + run.status + '"><strong>' + run.command + '</strong> — ' + run.status + payloadState + summary + detail + '</li>';
        }).join('');
      }

      async function refreshOverview() {
        const response = await fetch('/api/overview');
        if (!response.ok) return;
        const payload = await response.json();
        if (!payload.summary) return;

        const totalRuns = document.getElementById('total-runs');
        const outcomes = document.getElementById('run-outcomes');
        const runningNow = document.getElementById('running-now');
        const totalBytesSaved = document.getElementById('total-bytes-saved');
        const totalWordsSaved = document.getElementById('total-words-saved');
        const totalApproxTokensSaved = document.getElementById('total-approx-tokens-saved');
        const commandUsage = document.getElementById('command-usage');
        const recentRuns = document.getElementById('recent-runs');

        if (totalRuns) totalRuns.textContent = String(payload.summary.totalRuns);
        if (outcomes) outcomes.textContent = payload.summary.successfulRuns + ' successful / ' + payload.summary.failedRuns + ' failed';
        if (runningNow) runningNow.textContent = 'Running now: ' + payload.activeRuns;
        if (totalBytesSaved) totalBytesSaved.textContent = payload.summary.totalBytesSaved + ' bytes saved';
        if (totalWordsSaved) totalWordsSaved.textContent = payload.summary.totalWordsSaved + ' words saved';
        if (totalApproxTokensSaved) totalApproxTokensSaved.textContent = payload.summary.totalApproxTokensSaved + ' approx tokens saved';
        if (commandUsage) commandUsage.innerHTML = renderCommandUsage(payload.commandUsage);
        if (recentRuns) recentRuns.innerHTML = renderRecentRuns(payload.runs);
      }
      setInterval(refreshOverview, 1000);
    </script>
  </main>
</body>
</html>`))

var statsPageTemplate = template.Must(template.New("stats").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ .Title }}</title>
</head>
<body>
  <main>
    <h1>{{ .Title }}</h1>
    <p id="stats-outcomes">{{ .SuccessfulRuns }} successful / {{ .FailedRuns }} failed</p>
    <section>
      <h2>Usage trends</h2>
      <table>
        <thead>
          <tr><th>Day</th><th>Bytes saved</th><th>Words saved</th><th>Approx tokens saved</th></tr>
        </thead>
        <tbody id="usage-trend-rows">
          {{ range .TrendRows }}
          <tr>
            <td>{{ .Day }}</td>
            <td>{{ .BytesSaved }}</td>
            <td>{{ .WordsSaved }}</td>
            <td>{{ .ApproxTokensSaved }}</td>
          </tr>
          {{ else }}
          <tr><td colspan="4">No history yet.</td></tr>
          {{ end }}
        </tbody>
      </table>
    </section>
    <section>
      <h2>Top commands</h2>
      <ul id="stats-command-usage">
        {{ range .CommandUsage }}
        <li>{{ .Command }}: {{ .Count }}</li>
        {{ else }}
        <li>No runs recorded yet.</li>
        {{ end }}
      </ul>
    </section>
    <script>
      function renderStatsCommandUsage(items) {
        if (!items || items.length === 0) {
          return '<li>No runs recorded yet.</li>';
        }
        return items.map(function(item) {
          return '<li>' + item.command + ': ' + item.count + '</li>';
        }).join('');
      }

      function renderTrendRows(rows) {
        if (!rows || rows.length === 0) {
          return '<tr><td colspan="4">No history yet.</td></tr>';
        }
        return rows.map(function(row) {
          return '<tr><td>' + row.day + '</td><td>' + row.bytesSaved + '</td><td>' + row.wordsSaved + '</td><td>' + row.approxTokensSaved + '</td></tr>';
        }).join('');
      }

      async function refreshStats() {
        const response = await fetch('/api/stats');
        if (!response.ok) return;
        const payload = await response.json();

        const outcomes = document.getElementById('stats-outcomes');
        const trendRows = document.getElementById('usage-trend-rows');
        const commandUsage = document.getElementById('stats-command-usage');

        if (outcomes) outcomes.textContent = payload.successfulRuns + ' successful / ' + payload.failedRuns + ' failed';
        if (trendRows) trendRows.innerHTML = renderTrendRows(payload.trendRows);
        if (commandUsage) commandUsage.innerHTML = renderStatsCommandUsage(payload.commandUsage);
      }
      setInterval(refreshStats, 1000);
    </script>
  </main>
</body>
</html>`))

func NewServer(store *Store) *Server {
	return &Server{
		store: store,
		runs:  newRunTracker(defaultRecentRunLimit),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleOverview)
	mux.HandleFunc("/stats", s.handleStats)
	mux.HandleFunc("/api/events", s.handleEvent)
	mux.HandleFunc("/api/overview", s.handleOverviewSnapshot)
	mux.HandleFunc("/api/stats", s.handleStatsSnapshot)
	mux.HandleFunc("/api/runs", s.handleRuns)
	mux.HandleFunc("/runs/", s.handleRunDetail)
	return mux
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	summary, err := s.store.Summary()
	if err != nil {
		http.Error(w, fmt.Sprintf("load dashboard summary: %v", err), http.StatusInternalServerError)
		return
	}

	data := pageData{
		Title:                  "ma dashboard",
		TotalRuns:              summary.TotalRuns,
		SuccessfulRuns:         summary.SuccessfulRuns,
		FailedRuns:             summary.FailedRuns,
		ActiveRuns:             countActiveRuns(s.runs.Snapshot().Runs),
		TotalBytesSaved:        summary.TotalBytesSaved,
		TotalWordsSaved:        summary.TotalWordsSaved,
		TotalApproxTokensSaved: summary.TotalApproxTokensSaved,
		CommandUsage:           sortedCommandUsage(summary.CommandUsage),
		RecentRuns:             s.runs.Snapshot().Runs,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := dashboardPageTemplate.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("render dashboard: %v", err), http.StatusInternalServerError)
	}
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/stats" {
		http.NotFound(w, r)
		return
	}

	summary, err := s.store.Summary()
	if err != nil {
		http.Error(w, fmt.Sprintf("load dashboard summary: %v", err), http.StatusInternalServerError)
		return
	}
	history, err := s.store.History()
	if err != nil {
		http.Error(w, fmt.Sprintf("load dashboard history: %v", err), http.StatusInternalServerError)
		return
	}

	data := statsPageData{
		Title:          "ma dashboard stats",
		TrendRows:      buildTrendRows(history),
		CommandUsage:   sortedCommandUsage(summary.CommandUsage),
		SuccessfulRuns: summary.SuccessfulRuns,
		FailedRuns:     summary.FailedRuns,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := statsPageTemplate.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("render dashboard stats: %v", err), http.StatusInternalServerError)
	}
}

func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	var event RunEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, fmt.Sprintf("decode event: %v", err), http.StatusBadRequest)
		return
	}

	s.runs.Apply(event)
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.runs.Snapshot()); err != nil {
		http.Error(w, fmt.Sprintf("encode runs snapshot: %v", err), http.StatusInternalServerError)
	}
}

func (s *Server) handleOverviewSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	summary, err := s.store.Summary()
	if err != nil {
		http.Error(w, fmt.Sprintf("load dashboard summary: %v", err), http.StatusInternalServerError)
		return
	}
	runs := s.runs.Snapshot().Runs
	payload := overviewSnapshot{
		Summary: summarySnapshot{
			TotalRuns:              summary.TotalRuns,
			SuccessfulRuns:         summary.SuccessfulRuns,
			FailedRuns:             summary.FailedRuns,
			TotalBytesSaved:        summary.TotalBytesSaved,
			TotalWordsSaved:        summary.TotalWordsSaved,
			TotalApproxTokensSaved: summary.TotalApproxTokensSaved,
		},
		CommandUsage: sortedCommandUsage(summary.CommandUsage),
		ActiveRuns:   countActiveRuns(runs),
		Runs:         runs,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, fmt.Sprintf("encode overview snapshot: %v", err), http.StatusInternalServerError)
	}
}

func (s *Server) handleStatsSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	summary, err := s.store.Summary()
	if err != nil {
		http.Error(w, fmt.Sprintf("load dashboard summary: %v", err), http.StatusInternalServerError)
		return
	}
	history, err := s.store.History()
	if err != nil {
		http.Error(w, fmt.Sprintf("load dashboard history: %v", err), http.StatusInternalServerError)
		return
	}

	payload := statsSnapshot{
		TrendRows:      buildTrendRows(history),
		CommandUsage:   sortedCommandUsage(summary.CommandUsage),
		SuccessfulRuns: summary.SuccessfulRuns,
		FailedRuns:     summary.FailedRuns,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, fmt.Sprintf("encode stats snapshot: %v", err), http.StatusInternalServerError)
	}
}

func (s *Server) handleRunDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/runs/")
	if id == "" || id == r.URL.Path {
		http.NotFound(w, r)
		return
	}

	detail, ok := s.runs.Detail(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>%s</title></head><body>", template.HTMLEscapeString(detail.Command))
	fmt.Fprintf(w, "<h1>%s</h1><p>Status: %s</p>", template.HTMLEscapeString(detail.Command), template.HTMLEscapeString(detail.Status))
	if detail.PayloadStatus == payloadStatusRedacted {
		fmt.Fprint(w, "<p>Details withheld because the run involved a sensitive or protected path.</p>")
	} else {
		fmt.Fprintf(w, "<h2>Input</h2><pre>%s</pre>", template.HTMLEscapeString(detail.Input))
		fmt.Fprintf(w, "<h2>Output</h2><pre>%s</pre>", template.HTMLEscapeString(detail.Result.Output))
	}
	if detail.ResultSummary != "" {
		fmt.Fprintf(w, "<h2>Result summary</h2><p>%s</p>", template.HTMLEscapeString(detail.ResultSummary))
	}
	if detail.Error != "" {
		fmt.Fprintf(w, "<h2>Error</h2><pre>%s</pre>", template.HTMLEscapeString(detail.Error))
	}
	fmt.Fprint(w, "</body></html>")
}

func sortedCommandUsage(counts map[string]int) []commandUsage {
	items := make([]commandUsage, 0, len(counts))
	for command, count := range counts {
		items = append(items, commandUsage{Command: command, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Command < items[j].Command
		}
		return items[i].Count > items[j].Count
	})
	return items
}

func buildTrendRows(history []HistoryEntry) []trendRow {
	rows := make(map[string]*trendRow)
	for _, entry := range history {
		day := entry.StartedAt.In(time.UTC).Format("2006-01-02")
		row := rows[day]
		if row == nil {
			row = &trendRow{Day: day}
			rows[day] = row
		}
		row.BytesSaved += entry.Stats.InputBytes - entry.Stats.OutputBytes
		row.WordsSaved += entry.Stats.InputWords - entry.Stats.OutputWords
		row.ApproxTokensSaved += entry.Stats.InputApproxTokens - entry.Stats.OutputApproxTokens
	}

	keys := make([]string, 0, len(rows))
	for day := range rows {
		keys = append(keys, day)
	}
	sort.Strings(keys)

	result := make([]trendRow, 0, len(keys))
	for _, day := range keys {
		result = append(result, *rows[day])
	}
	return result
}

func countActiveRuns(runs []RunView) int {
	activeRuns := 0
	for _, run := range runs {
		if run.Status == eventKindStarted {
			activeRuns++
		}
	}
	return activeRuns
}
