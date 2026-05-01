package dashboard

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
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
	RecentRuns             []recentRunRow
}

type statsPageData struct {
	Title                  string
	SuccessfulRuns         int
	FailedRuns             int
	TotalBytesSaved        int
	TotalWordsSaved        int
	TotalApproxTokensSaved int
	TrendRows              []trendDisplayRow
	CommandInsights        []commandInsight
	OutcomeContext         []statsContextItem
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
	TrendRows              []trendRow         `json:"trendRows"`
	CommandUsage           []commandUsage     `json:"commandUsage"`
	CommandInsights        []commandInsight   `json:"commandInsights"`
	OutcomeContext         []statsContextItem `json:"outcomeContext"`
	SuccessfulRuns         int                `json:"successfulRuns"`
	FailedRuns             int                `json:"failedRuns"`
	TotalBytesSaved        int                `json:"totalBytesSaved"`
	TotalWordsSaved        int                `json:"totalWordsSaved"`
	TotalApproxTokensSaved int                `json:"totalApproxTokensSaved"`
}

type commandUsage struct {
	Command string `json:"command"`
	Count   int    `json:"count"`
}

type recentRunRow struct {
	ID          string
	Command     string
	StatusClass string
	StatusLabel string
	TimeLabel   string
	Summary     string
	HasDetails  bool
}

type trendRow struct {
	Month             string `json:"month"`
	BytesSaved        int    `json:"bytesSaved"`
	WordsSaved        int    `json:"wordsSaved"`
	ApproxTokensSaved int    `json:"approxTokensSaved"`
}

type trendDisplayRow struct {
	Month             string
	BytesSaved        int
	WordsSaved        int
	ApproxTokensSaved int
	TokenBarWidth     int
	OutcomeNote       string
}

type commandInsight struct {
	Command           string `json:"command"`
	Runs              int    `json:"runs"`
	SuccessfulRuns    int    `json:"successfulRuns"`
	FailedRuns        int    `json:"failedRuns"`
	BytesSaved        int    `json:"bytesSaved"`
	WordsSaved        int    `json:"wordsSaved"`
	ApproxTokensSaved int    `json:"approxTokensSaved"`
	SuccessRate       int    `json:"successRate"`
}

type statsContextItem struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Note  string `json:"note"`
	Tone  string `json:"tone"`
}

type detailPageData struct {
	Title       string
	Command     string
	StatusClass string
	StatusLabel string
	StartedAt   string
	FinishedAt  string
	PayloadNote string
	Summary     string
	Error       string
	InputPanel  detailPanel
	OutputPanel detailPanel
}

type detailPanel struct {
	Title      string
	StateClass string
	StateLabel string
	Message    string
	Content    string
}

var dashboardTemplateFuncs = template.FuncMap{
	"runDetailHref": runDetailHref,
}

var dashboardPageTemplate = template.Must(template.New("dashboard").Funcs(dashboardTemplateFuncs).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ .Title }}</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f6f4ef;
      --surface: #fcfbf8;
      --surface-strong: #f2eee6;
      --border: #dad5ca;
      --text: #1f2933;
      --muted: #5b6776;
      --quiet: #708090;
      --accent: #275fe4;
      --success-bg: #edf7ef;
      --success-text: #1f6b3b;
      --active-bg: #edf3ff;
      --active-text: #2350b9;
      --failed-bg: #feefef;
      --failed-text: #a02424;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background: var(--bg);
      color: var(--text);
      font: 15px/1.5 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    a { color: var(--accent); text-decoration: none; }
    a:hover { text-decoration: underline; }
    .shell {
      max-width: 76rem;
      margin: 0 auto;
      padding: 2rem 1.25rem 3rem;
    }
    .page-header {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      gap: 1rem;
      margin-bottom: 1.5rem;
    }
    .page-title {
      margin: 0;
      font-size: 1.1rem;
      font-weight: 650;
      letter-spacing: 0.01em;
      text-transform: lowercase;
    }
    .page-intro {
      margin: 0.35rem 0 0;
      max-width: 42rem;
      color: var(--muted);
    }
    .page-link {
      display: inline-flex;
      align-items: center;
      gap: 0.35rem;
      padding: 0.4rem 0.7rem;
      border: 1px solid var(--border);
      border-radius: 999px;
      background: rgba(252, 251, 248, 0.92);
      color: var(--text);
      font-size: 0.95rem;
      white-space: nowrap;
    }
    #activity-layout {
      display: grid;
      gap: 1.1rem;
    }
    .activity-hero {
      padding-bottom: 1rem;
      border-bottom: 1px solid var(--border);
    }
    .section-kicker {
      margin: 0 0 0.35rem;
      font-size: 0.76rem;
      font-weight: 700;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      color: var(--quiet);
    }
    .activity-heading {
      margin: 0;
      font-size: clamp(1.55rem, 2.8vw, 2.2rem);
      line-height: 1.2;
      font-weight: 650;
    }
    .activity-copy {
      margin: 0.55rem 0 0;
      max-width: 44rem;
      color: var(--muted);
    }
    .activity-meta {
      display: flex;
      flex-wrap: wrap;
      gap: 0.55rem;
      margin-top: 1rem;
    }
    .meta-pill {
      display: inline-flex;
      align-items: center;
      gap: 0.4rem;
      padding: 0.35rem 0.65rem;
      border-radius: 999px;
      border: 1px solid var(--border);
      background: var(--surface);
      color: var(--muted);
      font-size: 0.92rem;
    }
    #savings-band {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 0.75rem;
      padding: 0.85rem 0 1rem;
      border-bottom: 1px solid var(--border);
    }
    .band-card {
      padding: 0.8rem 0.95rem;
      border: 1px solid var(--border);
      border-radius: 1rem;
      background: var(--surface);
    }
    .band-card h2 {
      margin: 0;
      font-size: 0.82rem;
      font-weight: 650;
      color: var(--quiet);
      letter-spacing: 0.04em;
      text-transform: uppercase;
    }
    .band-card p {
      margin: 0.3rem 0 0;
      font-size: clamp(1.3rem, 3vw, 1.75rem);
      line-height: 1.1;
      font-weight: 640;
    }
    .band-card span {
      display: block;
      margin-top: 0.35rem;
      color: var(--muted);
      font-size: 0.9rem;
    }
    .section-head {
      display: flex;
      justify-content: space-between;
      align-items: end;
      gap: 1rem;
      margin-top: 0.15rem;
    }
    .section-head h2 {
      margin: 0;
      font-size: 1.15rem;
    }
    .section-head p {
      margin: 0.2rem 0 0;
      color: var(--muted);
    }
    .recent-runs-head {
      display: grid;
      grid-template-columns: minmax(12rem, 1.3fr) 8.5rem 12.5rem minmax(0, 1.9fr);
      gap: 0.85rem;
      padding: 0.8rem 0;
      border-bottom: 1px solid var(--border);
      color: var(--quiet);
      font-size: 0.78rem;
      font-weight: 700;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }
    #recent-runs-list {
      list-style: none;
      margin: 0;
      padding: 0;
    }
    .recent-run-row {
      display: grid;
      grid-template-columns: minmax(12rem, 1.3fr) 8.5rem 12.5rem minmax(0, 1.9fr);
      gap: 0.85rem;
      align-items: start;
      padding: 0.95rem 0;
      border-bottom: 1px solid rgba(218, 213, 202, 0.8);
    }
    .run-command {
      min-width: 0;
      display: flex;
      flex-direction: column;
      gap: 0.3rem;
    }
    .run-command strong {
      font-size: 0.98rem;
      font-weight: 650;
      word-break: break-word;
    }
    .run-command a {
      color: var(--muted);
      font-size: 0.88rem;
    }
    .status-pill {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-height: 2rem;
      padding: 0.2rem 0.7rem;
      border-radius: 999px;
      font-size: 0.86rem;
      font-weight: 650;
      white-space: nowrap;
    }
    .status-finished { background: var(--success-bg); color: var(--success-text); }
    .status-started { background: var(--active-bg); color: var(--active-text); }
    .status-failed { background: var(--failed-bg); color: var(--failed-text); }
    .status-stale { background: var(--border); color: var(--muted); }
    .run-time {
      color: var(--muted);
      font-size: 0.92rem;
      white-space: nowrap;
    }
    .run-summary {
      min-width: 0;
      color: var(--text);
      word-break: break-word;
    }
    .empty-state {
      padding: 1rem 0;
      color: var(--muted);
    }
    @media (max-width: 56rem) {
      #savings-band { grid-template-columns: 1fr; }
      .recent-runs-head { display: none; }
      .recent-run-row {
        grid-template-columns: 1fr;
        gap: 0.45rem;
      }
      .run-command a { margin-top: -0.05rem; }
      .status-pill, .run-time { width: fit-content; }
    }
  </style>
</head>
<body>
  <main class="shell">
    <header class="page-header">
      <div>
        <p class="page-title">{{ .Title }}</p>
        <p class="page-intro">Recent activity stays in the foreground while the savings band keeps long-term value visible without turning the dashboard into a control room.</p>
      </div>
      <a class="page-link" href="/stats">View stats</a>
    </header>
    <div id="activity-layout">
      <section class="activity-hero" aria-labelledby="recent-activity-title">
        <p class="section-kicker">Overview</p>
        <h1 id="recent-activity-title" class="activity-heading">Recent runs lead, aggregate savings stay quiet.</h1>
        <p class="activity-copy">Keep the last few runs easy to compare at a glance, surface live work without shouting, and leave heavier analysis to the dedicated stats view.</p>
        <div class="activity-meta">
          <span class="meta-pill">Total runs: <strong id="total-runs">{{ .TotalRuns }}</strong></span>
          <span class="meta-pill" id="run-outcomes">{{ .SuccessfulRuns }} successful / {{ .FailedRuns }} failed</span>
          <span class="meta-pill" id="running-now">Running now: {{ .ActiveRuns }}</span>
        </div>
      </section>
      <section id="savings-band" aria-label="Savings band">
        <article class="band-card">
          <h2>Bytes saved</h2>
          <p id="total-bytes-saved">{{ .TotalBytesSaved }} bytes</p>
          <span>Net byte reduction across completed runs.</span>
        </article>
        <article class="band-card">
          <h2>Words saved</h2>
          <p id="total-words-saved">{{ .TotalWordsSaved }} words</p>
          <span>Compression visible without competing with live activity.</span>
        </article>
        <article class="band-card">
          <h2>Approx tokens saved</h2>
          <p id="total-approx-tokens-saved">{{ .TotalApproxTokensSaved }} approx tokens</p>
          <span>Useful context for long-running prompt hygiene.</span>
        </article>
      </section>
      <section aria-labelledby="recent-runs-title">
        <div class="section-head">
          <div>
            <h2 id="recent-runs-title">Recent runs</h2>
            <p>Command, status, time, and short summary stay aligned so failures stand out and quiet wins fade into the background.</p>
          </div>
        </div>
        <div class="recent-runs-head" id="recent-runs-head">
          <span>Command</span>
          <span>Status</span>
          <span>Time</span>
          <span>Summary</span>
        </div>
        <ol id="recent-runs-list">
          {{ range .RecentRuns }}
          <li class="recent-run-row">
            <div class="run-command">
              <strong>{{ .Command }}</strong>
              {{ if .HasDetails }}<a href="{{ runDetailHref .ID }}">Open detail view</a>{{ end }}
            </div>
            <div><span class="status-pill {{ .StatusClass }}">{{ .StatusLabel }}</span></div>
            <time class="run-time">{{ .TimeLabel }}</time>
            <div class="run-summary">{{ .Summary }}</div>
          </li>
          {{ else }}
          <li class="empty-state">No recent runs yet.</li>
          {{ end }}
        </ol>
      </section>
    </div>
    <script>
      function escapeHTML(value) {
        return String(value || '').replace(/[&<>"']/g, function(char) {
          return {
            '&': '&amp;',
            '<': '&lt;',
            '>': '&gt;',
            '"': '&quot;',
            "'": '&#39;'
          }[char];
        });
      }

      function statusMeta(status) {
        if (status === 'failed') {
          return { className: 'status-failed', label: 'Failed' };
        }
        if (status === 'started') {
          return { className: 'status-started', label: 'Active' };
        }
        if (status === 'stale') {
          return { className: 'status-stale', label: 'Stale' };
        }
        return { className: 'status-finished', label: 'Completed' };
      }

      function formatTimestamp(value) {
        if (!value) {
          return 'Awaiting timestamp';
        }
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
          return 'Awaiting timestamp';
        }
        return date.toISOString().replace('T', ' ').replace(/\.\d+Z$/, ' UTC');
      }

      function runSummary(run) {
        if (run.resultSummary) {
          return run.resultSummary;
        }
        if (run.error) {
          return run.error;
        }
        if (run.payloadStatus === 'redacted') {
          return 'Details withheld for a protected path.';
        }
        if (run.payloadStatus === 'unavailable') {
          return 'Input payload unavailable for this run.';
        }
        if (run.status === 'started') {
          return 'Run in progress.';
        }
        if (run.status === 'failed') {
          return 'Run failed before a summary was captured.';
        }
        return 'Run completed without a recorded summary.';
      }

      function renderRecentRuns(runs) {
        if (!runs || runs.length === 0) {
          return '<li class="empty-state">No recent runs yet.</li>';
        }
        return runs.map(function(run) {
          const meta = statusMeta(run.status);
          const detail = run.hasDetails
            ? '<a href="/runs/' + encodeURIComponent(run.id) + '">Open detail view</a>'
            : '';
          const time = run.finishedAt || run.startedAt;
          return '' +
            '<li class="recent-run-row">' +
              '<div class="run-command">' +
                '<strong>' + escapeHTML(run.command) + '</strong>' +
                detail +
              '</div>' +
              '<div><span class="status-pill ' + meta.className + '">' + meta.label + '</span></div>' +
              '<time class="run-time">' + escapeHTML(formatTimestamp(time)) + '</time>' +
              '<div class="run-summary">' + escapeHTML(runSummary(run)) + '</div>' +
            '</li>';
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
        const recentRuns = document.getElementById('recent-runs-list');

        if (totalRuns) totalRuns.textContent = String(payload.summary.totalRuns);
        if (outcomes) outcomes.textContent = payload.summary.successfulRuns + ' successful / ' + payload.summary.failedRuns + ' failed';
        if (runningNow) runningNow.textContent = 'Running now: ' + payload.activeRuns;
        if (totalBytesSaved) totalBytesSaved.textContent = payload.summary.totalBytesSaved + ' bytes';
        if (totalWordsSaved) totalWordsSaved.textContent = payload.summary.totalWordsSaved + ' words';
        if (totalApproxTokensSaved) totalApproxTokensSaved.textContent = payload.summary.totalApproxTokensSaved + ' approx tokens';
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
  <style>
    :root {
      color-scheme: light;
      --bg: #f6f4ef;
      --surface: #fcfbf8;
      --border: #dad5ca;
      --text: #1f2933;
      --muted: #5b6776;
      --quiet: #708090;
      --accent: #275fe4;
      --alert-bg: #feefef;
      --alert-text: #a02424;
      --accent-bg: #edf3ff;
      --accent-text: #234fbe;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background: var(--bg);
      color: var(--text);
      font: 15px/1.5 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    a { color: var(--accent); text-decoration: none; }
    a:hover { text-decoration: underline; }
    .shell {
      max-width: 76rem;
      margin: 0 auto;
      padding: 2rem 1.25rem 3rem;
    }
    .page-header {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      gap: 1rem;
      margin-bottom: 1.4rem;
    }
    .page-title {
      margin: 0;
      font-size: 1.1rem;
      font-weight: 650;
      text-transform: lowercase;
    }
    .page-intro {
      margin: 0.35rem 0 0;
      max-width: 44rem;
      color: var(--muted);
    }
    .page-link {
      display: inline-flex;
      align-items: center;
      gap: 0.35rem;
      padding: 0.4rem 0.7rem;
      border: 1px solid var(--border);
      border-radius: 999px;
      background: rgba(252, 251, 248, 0.92);
      color: var(--text);
      font-size: 0.95rem;
      white-space: nowrap;
    }
    #stats-overview-band {
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      gap: 0.75rem;
      padding-bottom: 1.1rem;
      border-bottom: 1px solid var(--border);
    }
    .metric-card {
      padding: 0.85rem 0.95rem;
      border: 1px solid var(--border);
      border-radius: 1rem;
      background: var(--surface);
    }
    .metric-card h2 {
      margin: 0;
      font-size: 0.8rem;
      font-weight: 700;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      color: var(--quiet);
    }
    .metric-card p {
      margin: 0.28rem 0 0;
      font-size: 1.55rem;
      line-height: 1.1;
      font-weight: 650;
    }
    .metric-card span {
      display: block;
      margin-top: 0.35rem;
      color: var(--muted);
      font-size: 0.9rem;
    }
    .stats-grid {
      display: grid;
      grid-template-columns: minmax(0, 1.5fr) minmax(18rem, 1fr);
      gap: 1rem;
      margin-top: 1.1rem;
    }
    .panel {
      padding: 1rem 1.05rem;
      border: 1px solid var(--border);
      border-radius: 1rem;
      background: var(--surface);
    }
    .panel h2 {
      margin: 0;
      font-size: 1.08rem;
    }
    .panel p {
      margin: 0.3rem 0 0.95rem;
      color: var(--muted);
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 0.94rem;
    }
    th, td {
      padding: 0.7rem 0.2rem;
      border-bottom: 1px solid rgba(218, 213, 202, 0.85);
      text-align: left;
      vertical-align: top;
    }
    th {
      color: var(--quiet);
      font-size: 0.78rem;
      font-weight: 700;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }
    .sparkbar {
      display: inline-flex;
      align-items: center;
      width: 6.5rem;
      height: 0.55rem;
      border-radius: 999px;
      background: #ebe7dc;
      overflow: hidden;
      vertical-align: middle;
      margin-right: 0.5rem;
    }
    .sparkbar > span {
      display: block;
      height: 100%;
      border-radius: 999px;
      background: #2f6af0;
    }
    .trend-note {
      color: var(--muted);
      font-size: 0.86rem;
    }
    .success-meter {
      display: inline-flex;
      min-width: 3rem;
      padding: 0.18rem 0.55rem;
      border-radius: 999px;
      background: #eef3fb;
      color: #35506f;
      font-weight: 650;
      font-size: 0.84rem;
      justify-content: center;
    }
    #stats-outcome-context {
      list-style: none;
      margin: 0;
      padding: 0;
      display: grid;
      gap: 0.7rem;
    }
    #stats-outcome-context li {
      padding: 0.8rem 0.9rem;
      border-radius: 0.95rem;
      border: 1px solid var(--border);
      background: #faf8f3;
    }
    #stats-outcome-context strong {
      display: block;
      margin-bottom: 0.1rem;
      font-size: 1rem;
    }
    #stats-outcome-context .tone-alert { background: var(--alert-bg); color: var(--alert-text); }
    #stats-outcome-context .tone-accent { background: var(--accent-bg); color: var(--accent-text); }
    #stats-outcome-context .tone-quiet { color: var(--text); }
    @media (max-width: 68rem) {
      #stats-overview-band { grid-template-columns: repeat(2, minmax(0, 1fr)); }
      .stats-grid { grid-template-columns: 1fr; }
    }
    @media (max-width: 42rem) {
      #stats-overview-band { grid-template-columns: 1fr; }
      th:nth-child(4), td:nth-child(4), th:nth-child(5), td:nth-child(5) { display: none; }
    }
  </style>
</head>
<body>
  <main class="shell">
    <header class="page-header">
      <div>
        <p class="page-title">{{ .Title }}</p>
        <p class="page-intro">The stats view carries the longer arc: trends, rankings, and outcome context, all with restrained indicators so the page stays readable before it tries to be impressive.</p>
      </div>
      <a class="page-link" href="/">Back to overview</a>
    </header>
    <section id="stats-overview-band" aria-label="Stats overview">
      <article class="metric-card">
        <h2>Run outcomes</h2>
        <p id="stats-outcomes">{{ .SuccessfulRuns }} successful / {{ .FailedRuns }} failed</p>
        <span>Successful vs failed completed runs.</span>
      </article>
      <article class="metric-card">
        <h2>Bytes saved</h2>
        <p id="stats-total-bytes">{{ .TotalBytesSaved }}</p>
        <span>Net byte reduction across durable history.</span>
      </article>
      <article class="metric-card">
        <h2>Words saved</h2>
        <p id="stats-total-words">{{ .TotalWordsSaved }}</p>
        <span>Word-level compression staying visible but calm.</span>
      </article>
      <article class="metric-card">
        <h2>Approx tokens saved</h2>
        <p id="stats-total-approx-tokens">{{ .TotalApproxTokensSaved }}</p>
        <span>Prompt-weight context without big-chart noise.</span>
      </article>
    </section>
    <div class="stats-grid">
      <section class="panel">
        <h2>Usage trends</h2>
        <p>Monthly savings stay text-first, with a small token accent to help comparisons without taking over the page.</p>
        <table>
          <thead>
            <tr><th>Month</th><th>Bytes saved</th><th>Words saved</th><th>Approx tokens saved</th><th>Accent</th></tr>
          </thead>
          <tbody id="usage-trend-rows">
            {{ range .TrendRows }}
            <tr>
              <td>{{ .Month }}</td>
              <td>{{ .BytesSaved }}</td>
              <td>{{ .WordsSaved }}</td>
              <td>{{ .ApproxTokensSaved }}</td>
              <td><span class="sparkbar"><span style="width: {{ .TokenBarWidth }}%"></span></span><span class="trend-note">{{ .OutcomeNote }}</span></td>
            </tr>
            {{ else }}
            <tr><td colspan="5">No history yet.</td></tr>
            {{ end }}
          </tbody>
        </table>
      </section>
      <section class="panel">
        <h2>Outcome context</h2>
        <p>Quiet context helps the homepage stay focused on what just happened.</p>
        <ul id="stats-outcome-context">
          {{ range .OutcomeContext }}
          <li class="{{ .Tone }}">
            <strong>{{ .Label }}: {{ .Value }}</strong>
            <span>{{ .Note }}</span>
          </li>
          {{ else }}
          <li class="tone-quiet"><strong>No history yet</strong><span>Run a few commands to build durable context here.</span></li>
          {{ end }}
        </ul>
      </section>
      <section class="panel" style="grid-column: 1 / -1;">
        <h2>Command rankings</h2>
        <p>High-level command performance belongs here so the overview can stay centered on recent activity.</p>
        <table>
          <thead>
            <tr><th>Command</th><th>Runs</th><th>Success</th><th>Bytes saved</th><th>Approx tokens saved</th></tr>
          </thead>
          <tbody id="stats-command-usage">
            {{ range .CommandInsights }}
            <tr>
              <td>{{ .Command }}</td>
              <td>{{ .Runs }}</td>
              <td><span class="success-meter">{{ .SuccessRate }}%</span></td>
              <td>{{ .BytesSaved }}</td>
              <td>{{ .ApproxTokensSaved }}</td>
            </tr>
            {{ else }}
            <tr><td colspan="5">No runs recorded yet.</td></tr>
            {{ end }}
          </tbody>
        </table>
      </section>
    </div>
    <script>
      function escapeHTML(value) {
        return String(value || '').replace(/[&<>"']/g, function(char) {
          return {
            '&': '&amp;',
            '<': '&lt;',
            '>': '&gt;',
            '"': '&quot;',
            "'": '&#39;'
          }[char];
        });
      }

      function barWidth(rows, tokens) {
        const max = (rows || []).reduce(function(current, row) {
          return Math.max(current, Math.abs(row.approxTokensSaved || 0));
        }, 0);
        if (max === 0 || !tokens) {
          return 0;
        }
        const width = Math.round(Math.abs(tokens) * 100 / max);
        return Math.max(width, 12);
      }

      function trendNote(tokens) {
        if (tokens < 0) {
          return 'negative net month';
        }
        if (tokens === 0) {
          return 'flat month';
        }
        return 'steady savings';
      }

      function renderTrendRows(rows) {
        if (!rows || rows.length === 0) {
          return '<tr><td colspan="5">No history yet.</td></tr>';
        }
        return rows.map(function(row) {
          const width = barWidth(rows, row.approxTokensSaved);
          return '<tr>' +
            '<td>' + escapeHTML(row.month) + '</td>' +
            '<td>' + escapeHTML(row.bytesSaved) + '</td>' +
            '<td>' + escapeHTML(row.wordsSaved) + '</td>' +
            '<td>' + escapeHTML(row.approxTokensSaved) + '</td>' +
            '<td><span class="sparkbar"><span style="width: ' + width + '%"></span></span><span class="trend-note">' + trendNote(row.approxTokensSaved) + '</span></td>' +
            '</tr>';
        }).join('');
      }

      function renderCommandInsights(items) {
        if (!items || items.length === 0) {
          return '<tr><td colspan="5">No runs recorded yet.</td></tr>';
        }
        return items.map(function(item) {
          return '<tr>' +
            '<td>' + escapeHTML(item.command) + '</td>' +
            '<td>' + escapeHTML(item.runs) + '</td>' +
            '<td><span class="success-meter">' + escapeHTML(item.successRate) + '%</span></td>' +
            '<td>' + escapeHTML(item.bytesSaved) + '</td>' +
            '<td>' + escapeHTML(item.approxTokensSaved) + '</td>' +
            '</tr>';
        }).join('');
      }

      function renderOutcomeContext(items) {
        if (!items || items.length === 0) {
          return '<li class="tone-quiet"><strong>No history yet</strong><span>Run a few commands to build durable context here.</span></li>';
        }
        return items.map(function(item) {
          return '<li class="' + escapeHTML(item.tone) + '"><strong>' + escapeHTML(item.label) + ': ' + escapeHTML(item.value) + '</strong><span>' + escapeHTML(item.note) + '</span></li>';
        }).join('');
      }

      async function refreshStats() {
        const response = await fetch('/api/stats');
        if (!response.ok) return;
        const payload = await response.json();

        const outcomes = document.getElementById('stats-outcomes');
        const totalBytes = document.getElementById('stats-total-bytes');
        const totalWords = document.getElementById('stats-total-words');
        const totalApproxTokens = document.getElementById('stats-total-approx-tokens');
        const trendRows = document.getElementById('usage-trend-rows');
        const commandUsage = document.getElementById('stats-command-usage');
        const outcomeContext = document.getElementById('stats-outcome-context');

        if (outcomes) outcomes.textContent = payload.successfulRuns + ' successful / ' + payload.failedRuns + ' failed';
        if (totalBytes) totalBytes.textContent = String(payload.totalBytesSaved || 0);
        if (totalWords) totalWords.textContent = String(payload.totalWordsSaved || 0);
        if (totalApproxTokens) totalApproxTokens.textContent = String(payload.totalApproxTokensSaved || 0);
        if (trendRows) trendRows.innerHTML = renderTrendRows(payload.trendRows);
        if (commandUsage) commandUsage.innerHTML = renderCommandInsights(payload.commandInsights);
        if (outcomeContext) outcomeContext.innerHTML = renderOutcomeContext(payload.outcomeContext);
      }
      setInterval(refreshStats, 1000);
    </script>
  </main>
</body>
</html>`))

var detailPageTemplate = template.Must(template.New("detail").Funcs(dashboardTemplateFuncs).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ .Title }}</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f6f4ef;
      --surface: #fcfbf8;
      --border: #dad5ca;
      --text: #1f2933;
      --muted: #5b6776;
      --quiet: #708090;
      --success-bg: #edf7ef;
      --success-text: #1f6b3b;
      --active-bg: #edf3ff;
      --active-text: #2350b9;
      --failed-bg: #feefef;
      --failed-text: #a02424;
      --state-bg: #f7f3ea;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background: var(--bg);
      color: var(--text);
      font: 15px/1.55 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    a { color: #275fe4; text-decoration: none; }
    a:hover { text-decoration: underline; }
    .shell {
      max-width: 82rem;
      margin: 0 auto;
      padding: 2rem 1.25rem 3rem;
    }
    .page-header {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      gap: 1rem;
      margin-bottom: 1.2rem;
    }
    .page-header p {
      margin: 0;
      color: var(--muted);
    }
    .status-pill {
      display: inline-flex;
      align-items: center;
      min-height: 2rem;
      padding: 0.2rem 0.7rem;
      border-radius: 999px;
      font-size: 0.86rem;
      font-weight: 650;
      white-space: nowrap;
    }
    .status-finished { background: var(--success-bg); color: var(--success-text); }
    .status-started { background: var(--active-bg); color: var(--active-text); }
    .status-failed { background: var(--failed-bg); color: var(--failed-text); }
    .status-stale { background: var(--border); color: var(--muted); }
    .detail-header {
      padding-bottom: 1rem;
      border-bottom: 1px solid var(--border);
    }
    .detail-header h1 {
      margin: 0.3rem 0 0;
      font-size: clamp(1.5rem, 2.8vw, 2.15rem);
      line-height: 1.2;
      font-weight: 650;
    }
    .detail-header p {
      margin: 0.55rem 0 0;
      max-width: 46rem;
      color: var(--muted);
    }
    .detail-meta {
      display: flex;
      flex-wrap: wrap;
      gap: 0.6rem;
      margin-top: 1rem;
    }
    .meta-chip {
      display: inline-flex;
      align-items: center;
      gap: 0.4rem;
      padding: 0.35rem 0.65rem;
      border: 1px solid var(--border);
      border-radius: 999px;
      background: var(--surface);
      color: var(--muted);
      font-size: 0.92rem;
    }
    .detail-grid {
      display: grid;
      gap: 1rem;
      margin-top: 1rem;
    }
    .detail-card {
      padding: 1rem 1.05rem;
      border: 1px solid var(--border);
      border-radius: 1rem;
      background: var(--surface);
    }
    .detail-card h2 {
      margin: 0;
      font-size: 1.08rem;
    }
    .detail-card p {
      margin: 0.35rem 0 0;
      color: var(--muted);
    }
    .comparison-grid {
      display: grid;
      gap: 1rem;
    }
    .comparison-panel {
      padding: 1rem 1.05rem;
      border: 1px solid var(--border);
      border-radius: 1rem;
      background: var(--surface);
    }
    .comparison-panel h2 {
      margin: 0;
      font-size: 1.08rem;
    }
    .panel-state {
      display: inline-flex;
      align-items: center;
      width: fit-content;
      margin-top: 0.55rem;
      padding: 0.18rem 0.58rem;
      border-radius: 999px;
      background: var(--state-bg);
      color: var(--muted);
      font-size: 0.84rem;
      font-weight: 650;
    }
    .state-redacted { background: #f5e7d9; color: #8a4b12; }
    .state-unavailable { background: #f2edf9; color: #6b40a9; }
    .state-empty { background: #eef2f7; color: #5a6573; }
    .state-ready { background: #edf7ef; color: #1f6b3b; }
    .comparison-panel p {
      margin: 0.55rem 0 0;
      color: var(--muted);
    }
    pre {
      margin: 0.85rem 0 0;
      padding: 0.95rem;
      border-radius: 0.85rem;
      background: #f4f2eb;
      overflow-x: auto;
      white-space: pre-wrap;
      word-break: break-word;
      font: 0.93rem/1.55 ui-monospace, SFMono-Regular, SFMono-Regular, Menlo, monospace;
      color: #1d2730;
    }
    @media (min-width: 72rem) {
      .comparison-grid {
        grid-template-columns: repeat(2, minmax(0, 1fr));
      }
    }
  </style>
</head>
<body>
  <main class="shell">
    <header class="page-header">
      <div>
        <a href="/">Back to overview</a>
      </div>
      <p><a href="/stats">Open stats</a></p>
    </header>
    <section class="detail-header">
      <span class="status-pill {{ .StatusClass }}">{{ .StatusLabel }}</span>
      <h1>{{ .Command }}</h1>
      <p>Comparisons stay stacked by default for readability, then expand into a wider side-by-side layout when the viewport can afford it.</p>
      <div class="detail-meta">
        <span class="meta-chip">Started: {{ .StartedAt }}</span>
        <span class="meta-chip">Finished: {{ .FinishedAt }}</span>
        <span class="meta-chip">Payload: {{ .PayloadNote }}</span>
      </div>
    </section>
    <section class="detail-grid">
      <article class="detail-card">
        <h2>Result summary</h2>
        <p>{{ .Summary }}</p>
      </article>
      {{ if .Error }}
      <article class="detail-card">
        <h2>Error</h2>
        <pre>{{ .Error }}</pre>
      </article>
      {{ end }}
      <div class="comparison-grid">
        <section class="comparison-panel">
          <h2>{{ .InputPanel.Title }}</h2>
          <span class="panel-state {{ .InputPanel.StateClass }}">{{ .InputPanel.StateLabel }}</span>
          <p>{{ .InputPanel.Message }}</p>
          {{ if .InputPanel.Content }}<pre>{{ .InputPanel.Content }}</pre>{{ end }}
        </section>
        <section class="comparison-panel">
          <h2>{{ .OutputPanel.Title }}</h2>
          <span class="panel-state {{ .OutputPanel.StateClass }}">{{ .OutputPanel.StateLabel }}</span>
          <p>{{ .OutputPanel.Message }}</p>
          {{ if .OutputPanel.Content }}<pre>{{ .OutputPanel.Content }}</pre>{{ end }}
        </section>
      </div>
    </section>
  </main>
</body>
</html>`))

func NewServer(store *Store) *Server {
	tracker := newRunTracker(recentRunLimit())
	if store != nil {
		if recentRuns, err := store.RecentRuns(); err == nil {
			tracker.Seed(recentRuns)
		}
	}

	return &Server{
		store: store,
		runs:  tracker,
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
		ActiveRuns:             countActiveRuns(s.runs.Snapshot().Runs, time.Now().UTC()),
		TotalBytesSaved:        summary.TotalBytesSaved,
		TotalWordsSaved:        summary.TotalWordsSaved,
		TotalApproxTokensSaved: summary.TotalApproxTokensSaved,
		RecentRuns:             buildRecentRunRows(s.runs.Snapshot().Runs),
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
		Title:                  "ma dashboard stats",
		SuccessfulRuns:         summary.SuccessfulRuns,
		FailedRuns:             summary.FailedRuns,
		TotalBytesSaved:        summary.TotalBytesSaved,
		TotalWordsSaved:        summary.TotalWordsSaved,
		TotalApproxTokensSaved: summary.TotalApproxTokensSaved,
		TrendRows:              buildTrendDisplayRows(buildTrendRows(history)),
		CommandInsights:        buildCommandInsights(history),
		OutcomeContext:         buildOutcomeContext(summary, buildTrendRows(history), buildCommandInsights(history)),
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
	now := time.Now().UTC()
	deriveEffectiveStatus(runs, now)
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
		ActiveRuns:   countActiveRuns(runs, now),
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
		TrendRows:              buildTrendRows(history),
		CommandUsage:           sortedCommandUsage(summary.CommandUsage),
		CommandInsights:        buildCommandInsights(history),
		OutcomeContext:         buildOutcomeContext(summary, buildTrendRows(history), buildCommandInsights(history)),
		SuccessfulRuns:         summary.SuccessfulRuns,
		FailedRuns:             summary.FailedRuns,
		TotalBytesSaved:        summary.TotalBytesSaved,
		TotalWordsSaved:        summary.TotalWordsSaved,
		TotalApproxTokensSaved: summary.TotalApproxTokensSaved,
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

	idPath := strings.TrimPrefix(r.URL.Path, "/runs/")
	if idPath == "" || idPath == r.URL.Path {
		http.NotFound(w, r)
		return
	}
	id, err := url.PathUnescape(idPath)
	if err != nil || id == "" {
		http.NotFound(w, r)
		return
	}

	detail, ok := s.runs.Detail(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := detailPageTemplate.Execute(w, buildDetailPageData(detail)); err != nil {
		http.Error(w, fmt.Sprintf("render run detail: %v", err), http.StatusInternalServerError)
	}
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
		month := entry.StartedAt.In(time.UTC).Format("2006-01")
		row := rows[month]
		if row == nil {
			row = &trendRow{Month: month}
			rows[month] = row
		}
		row.BytesSaved += entry.Stats.InputBytes - entry.Stats.OutputBytes
		row.WordsSaved += entry.Stats.InputWords - entry.Stats.OutputWords
		row.ApproxTokensSaved += entry.Stats.InputApproxTokens - entry.Stats.OutputApproxTokens
	}

	keys := make([]string, 0, len(rows))
	for month := range rows {
		keys = append(keys, month)
	}
	sort.Strings(keys)

	result := make([]trendRow, 0, len(keys))
	for _, month := range keys {
		result = append(result, *rows[month])
	}
	return result
}

const staleRunThreshold = 60 * time.Second

// deriveEffectiveStatus replaces "started" with "stale" in the status field
// for runs that have been active longer than the stale threshold. This keeps
// stale detection in one place so the JS polling path renders what the server
// computes.
func deriveEffectiveStatus(runs []RunView, now time.Time) {
	cutoff := now.Add(-staleRunThreshold)
	for i := range runs {
		if runs[i].Status == eventKindStarted && !runs[i].StartedAt.After(cutoff) {
			runs[i].Status = "stale"
		}
	}
}

func countActiveRuns(runs []RunView, now time.Time) int {
	activeRuns := 0
	cutoff := now.Add(-staleRunThreshold)
	for _, run := range runs {
		if run.Status == eventKindStarted && run.StartedAt.After(cutoff) {
			activeRuns++
		}
	}
	return activeRuns
}

func buildRecentRunRows(runs []RunView) []recentRunRow {
	rows := make([]recentRunRow, 0, len(runs))
	for _, run := range runs {
		statusLabel, statusClass := statusPresentation(run.Status, run.StartedAt)
		rows = append(rows, recentRunRow{
			ID:          run.ID,
			Command:     run.Command,
			StatusClass: statusClass,
			StatusLabel: statusLabel,
			TimeLabel:   runTimestampLabel(run),
			Summary:     recentRunSummary(run),
			HasDetails:  run.HasDetails,
		})
	}
	return rows
}

func buildTrendDisplayRows(rows []trendRow) []trendDisplayRow {
	maxTokens := 0
	for _, row := range rows {
		if tokens := absInt(row.ApproxTokensSaved); tokens > maxTokens {
			maxTokens = tokens
		}
	}

	displayRows := make([]trendDisplayRow, 0, len(rows))
	for _, row := range rows {
		displayRows = append(displayRows, trendDisplayRow{
			Month:             row.Month,
			BytesSaved:        row.BytesSaved,
			WordsSaved:        row.WordsSaved,
			ApproxTokensSaved: row.ApproxTokensSaved,
			TokenBarWidth:     scaledWidth(absInt(row.ApproxTokensSaved), maxTokens),
			OutcomeNote:       trendNote(row.ApproxTokensSaved),
		})
	}
	return displayRows
}

func buildCommandInsights(history []HistoryEntry) []commandInsight {
	rows := make(map[string]*commandInsight)
	for _, entry := range history {
		row := rows[entry.Command]
		if row == nil {
			row = &commandInsight{Command: entry.Command}
			rows[entry.Command] = row
		}
		row.Runs++
		if entry.Success {
			row.SuccessfulRuns++
		} else {
			row.FailedRuns++
		}
		row.BytesSaved += entry.Stats.InputBytes - entry.Stats.OutputBytes
		row.WordsSaved += entry.Stats.InputWords - entry.Stats.OutputWords
		row.ApproxTokensSaved += entry.Stats.InputApproxTokens - entry.Stats.OutputApproxTokens
	}

	items := make([]commandInsight, 0, len(rows))
	for _, row := range rows {
		if row.Runs > 0 {
			row.SuccessRate = (row.SuccessfulRuns*100 + row.Runs/2) / row.Runs
		}
		items = append(items, *row)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Runs == items[j].Runs {
			if items[i].ApproxTokensSaved == items[j].ApproxTokensSaved {
				return items[i].Command < items[j].Command
			}
			return items[i].ApproxTokensSaved > items[j].ApproxTokensSaved
		}
		return items[i].Runs > items[j].Runs
	})
	return items
}

func buildOutcomeContext(summary Summary, trends []trendRow, insights []commandInsight) []statsContextItem {
	if summary.TotalRuns == 0 {
		return nil
	}

	items := make([]statsContextItem, 0, 3)
	successRate := 0
	if summary.TotalRuns > 0 {
		successRate = (summary.SuccessfulRuns*100 + summary.TotalRuns/2) / summary.TotalRuns
	}
	tone := "tone-quiet"
	if summary.FailedRuns > 0 {
		tone = "tone-alert"
	}
	items = append(items, statsContextItem{
		Label: "Success rate",
		Value: fmt.Sprintf("%d%%", successRate),
		Note:  fmt.Sprintf("%d successful and %d failed runs in durable history.", summary.SuccessfulRuns, summary.FailedRuns),
		Tone:  tone,
	})
	if len(insights) > 0 {
		items = append(items, statsContextItem{
			Label: "Most active command",
			Value: insights[0].Command,
			Note:  fmt.Sprintf("%d runs with %d%% success and %d approx tokens saved.", insights[0].Runs, insights[0].SuccessRate, insights[0].ApproxTokensSaved),
			Tone:  "tone-accent",
		})
	}
	if bestMonth, ok := strongestSavingsMonth(trends); ok {
		items = append(items, statsContextItem{
			Label: "Strongest savings month",
			Value: bestMonth.Month,
			Note:  fmt.Sprintf("%d approx tokens and %d bytes saved.", bestMonth.ApproxTokensSaved, bestMonth.BytesSaved),
			Tone:  "tone-quiet",
		})
	}
	return items
}

func strongestSavingsMonth(rows []trendRow) (trendRow, bool) {
	if len(rows) == 0 {
		return trendRow{}, false
	}
	best := rows[0]
	for _, row := range rows[1:] {
		if row.ApproxTokensSaved > best.ApproxTokensSaved {
			best = row
		}
	}
	return best, true
}

func buildDetailPageData(detail RunDetail) detailPageData {
	statusLabel, statusClass := statusPresentation(detail.Status, detail.StartedAt)
	summary := detail.ResultSummary
	if summary == "" {
		summary = detailSummary(detail)
	}
	return detailPageData{
		Title:       detail.Command + " run detail",
		Command:     detail.Command,
		StatusClass: statusClass,
		StatusLabel: statusLabel,
		StartedAt:   formatTimestamp(detail.StartedAt),
		FinishedAt:  detailFinishedLabel(detail),
		PayloadNote: payloadLabel(detail.PayloadStatus),
		Summary:     summary,
		Error:       detail.Error,
		InputPanel:  buildInputPanel(detail),
		OutputPanel: buildOutputPanel(detail),
	}
}

func buildInputPanel(detail RunDetail) detailPanel {
	panel := detailPanel{Title: "Input"}
	switch detail.PayloadStatus {
	case payloadStatusRedacted:
		panel.StateClass = "state-redacted"
		panel.StateLabel = "Withheld"
		panel.Message = "This run involved a sensitive or protected path, so the dashboard intentionally withholds the captured input."
	case payloadStatusUnavailable:
		panel.StateClass = "state-unavailable"
		panel.StateLabel = "Unavailable"
		panel.Message = "The dashboard could not read the input payload for this run."
	case payloadStatusObserved:
		if detail.Input == "" {
			panel.StateClass = "state-empty"
			panel.StateLabel = "Empty"
			panel.Message = "The observed input body was empty."
			return panel
		}
		panel.StateClass = "state-ready"
		panel.StateLabel = "Observed"
		panel.Message = "Captured input is shown here for before-and-after inspection."
		panel.Content = detail.Input
	default:
		panel.StateClass = "state-empty"
		panel.StateLabel = "Not captured"
		panel.Message = "This run did not provide a readable file input for the dashboard."
	}
	return panel
}

func buildOutputPanel(detail RunDetail) detailPanel {
	panel := detailPanel{Title: "Output"}
	switch {
	case detail.Result.Output != "":
		panel.StateClass = "state-ready"
		panel.StateLabel = "Observed"
		panel.Message = "Captured command output stays readable in place."
		panel.Content = detail.Result.Output
	case detail.Status == eventKindStarted:
		panel.StateClass = "state-unavailable"
		panel.StateLabel = "Pending"
		panel.Message = "This run is still active, so output content is not available yet."
	case detail.Status == eventKindFailed && detail.Error != "":
		panel.StateClass = "state-unavailable"
		panel.StateLabel = "Unavailable"
		panel.Message = "This run ended with an error before output text was captured."
	default:
		panel.StateClass = "state-empty"
		panel.StateLabel = "Empty"
		panel.Message = "This run completed without output text."
	}
	return panel
}

func detailSummary(detail RunDetail) string {
	switch {
	case detail.Status == eventKindStarted:
		return "The run is still in progress."
	case detail.Status == eventKindFailed && detail.Error != "":
		return "The run ended with an error before a dedicated result summary was captured."
	case detail.PayloadStatus == payloadStatusRedacted:
		return "Details are intentionally limited because the run involved a protected path."
	default:
		return "No additional result summary was captured for this run."
	}
}

func detailFinishedLabel(detail RunDetail) string {
	if detail.FinishedAt == nil {
		return "Still running"
	}
	return formatTimestamp(*detail.FinishedAt)
}

func payloadLabel(status string) string {
	switch status {
	case payloadStatusObserved:
		return "Observed"
	case payloadStatusRedacted:
		return "Withheld"
	case payloadStatusUnavailable:
		return "Unavailable"
	default:
		return "Not captured"
	}
}

func recentRunSummary(run RunView) string {
	switch {
	case run.ResultSummary != "":
		return run.ResultSummary
	case run.Error != "":
		return run.Error
	case run.PayloadStatus == payloadStatusRedacted:
		return "Details withheld for a protected path."
	case run.PayloadStatus == payloadStatusUnavailable:
		return "Input payload unavailable for this run."
	case run.Status == eventKindStarted:
		return "Run in progress."
	case run.Status == eventKindFailed:
		return "Run failed before a summary was captured."
	default:
		return "Run completed without a recorded summary."
	}
}

func runTimestampLabel(run RunView) string {
	if run.FinishedAt != nil {
		return formatTimestamp(*run.FinishedAt)
	}
	if !run.StartedAt.IsZero() {
		return formatTimestamp(run.StartedAt)
	}
	return "Awaiting timestamp"
}

func statusPresentation(status string, startedAt time.Time) (string, string) {
	switch status {
	case eventKindStarted:
		if time.Since(startedAt) > staleRunThreshold {
			return "Stale", "status-stale"
		}
		return "Active", "status-started"
	case "stale":
		return "Stale", "status-stale"
	case eventKindFailed:
		return "Failed", "status-failed"
	default:
		return "Completed", "status-finished"
	}
}

func runDetailHref(id string) string {
	return "/runs/" + url.PathEscape(id)
}

func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return "Awaiting timestamp"
	}
	return value.In(time.UTC).Format("2006-01-02 15:04:05 MST")
}

func trendNote(tokens int) string {
	switch {
	case tokens < 0:
		return "negative net month"
	case tokens == 0:
		return "flat month"
	default:
		return "steady savings"
	}
}

func scaledWidth(value int, maxValue int) int {
	if value <= 0 || maxValue <= 0 {
		return 0
	}
	width := (value * 100) / maxValue
	if width < 12 {
		return 12
	}
	return width
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
