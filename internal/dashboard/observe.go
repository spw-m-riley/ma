package dashboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spw-m-riley/ma/internal/app"
	"github.com/spw-m-riley/ma/internal/detect"
)

const stateDirEnv = "MA_DASHBOARD_STATE_DIR"
const sourceEnv = "MA_SOURCE"
const (
	eventKindStarted  = "started"
	eventKindFinished = "finished"
	eventKindFailed   = "failed"

	payloadStatusNone        = "none"
	payloadStatusObserved    = "observed"
	payloadStatusRedacted    = "redacted"
	payloadStatusUnavailable = "unavailable"
)

type RunEvent struct {
	Kind          string     `json:"kind"`
	ID            string     `json:"id"`
	Command       string     `json:"command"`
	Source        string     `json:"source,omitempty"`
	StartedAt     time.Time  `json:"startedAt"`
	FinishedAt    *time.Time `json:"finishedAt,omitempty"`
	Success       bool       `json:"success"`
	Changed       bool       `json:"changed"`
	Error         string     `json:"error,omitempty"`
	PayloadStatus string     `json:"payloadStatus,omitempty"`
	Input         string     `json:"input,omitempty"`
	Result        app.Result `json:"result,omitempty"`
	ResultSummary string     `json:"resultSummary,omitempty"`
}

func DefaultRoot() (string, error) {
	if root := os.Getenv(stateDirEnv); root != "" {
		return root, nil
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve user cache dir: %w", err)
	}

	return filepath.Join(cacheDir, "ma", "dashboard"), nil
}

func ObserveRun(commandName string, args []string, run func() (app.Result, error)) (app.Result, error) {
	startedAt := time.Now().UTC()
	runID := fmt.Sprintf("%d-%d", startedAt.UnixNano(), os.Getpid())
	input, payloadStatus := collectInputPayload(args)
	source := os.Getenv(sourceEnv)

	root, rootErr := DefaultRoot()
	if rootErr == nil {
		dispatchEvent(root, RunEvent{
			Kind:          eventKindStarted,
			ID:            runID,
			Command:       commandName,
			Source:        source,
			StartedAt:     startedAt,
			PayloadStatus: payloadStatus,
		})
	}

	result, runErr := run()
	finishedAt := time.Now().UTC()
	observedResult := trimObservedResult(withCommand(result, commandName))

	if rootErr != nil {
		return result, runErr
	}
	store, err := OpenStore(root)
	if err != nil {
		_ = recordDiagnostic(root, "history persistence unavailable: "+err.Error())
		dispatchEvent(root, finishedEvent(runID, commandName, source, startedAt, finishedAt, input, payloadStatus, observedResult, runErr))
		return result, runErr
	}

	if err := store.RecordFinished(FinishedRun{
		ID:            runID,
		Command:       commandName,
		Source:        source,
		StartedAt:     startedAt,
		FinishedAt:    finishedAt,
		Success:       runErr == nil,
		Changed:       observedResult.Changed,
		Error:         errorMessage(runErr),
		PayloadStatus: payloadStatus,
		Input:         input,
		ResultSummary: summarizeResult(observedResult, runErr),
		Result:        observedResult,
	}); err != nil {
		_ = recordDiagnostic(root, "history persistence failed: "+err.Error())
		dispatchEvent(root, finishedEvent(runID, commandName, source, startedAt, finishedAt, input, payloadStatus, observedResult, runErr))
		return result, runErr
	}

	dispatchEvent(root, finishedEvent(runID, commandName, source, startedAt, finishedAt, input, payloadStatus, observedResult, runErr))
	return result, runErr
}

func withCommand(result app.Result, commandName string) app.Result {
	if result.Command == "" {
		result.Command = commandName
	}
	return result
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func sessionPath(root string) string {
	return filepath.Join(root, sessionFileName)
}

func publishEvent(root string, event RunEvent) error {
	sessionBytes, err := os.ReadFile(sessionPath(root))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		_ = clearSession(root)
		return err
	}

	var session Session
	if err := json.Unmarshal(sessionBytes, &session); err != nil {
		_ = clearSession(root)
		return err
	}
	if session.Address == "" {
		_ = clearSession(root)
		return nil
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, "http://"+session.Address+"/api/events", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		_ = clearSession(root)
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_ = clearSession(root)
		return fmt.Errorf("event delivery returned status %d", response.StatusCode)
	}

	return nil
}

func collectInputPayload(args []string) (string, string) {
	if len(args) == 0 {
		return "", payloadStatusNone
	}

	var blocks []string
	for _, arg := range args {
		if arg == "" {
			continue
		}
		if detect.IsSensitivePathResolved(arg) {
			return "", payloadStatusRedacted
		}
		content, err := os.ReadFile(arg)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", payloadStatusUnavailable
		}
		blocks = append(blocks, formatInputBlock(arg, string(content)))
	}
	if len(blocks) == 0 {
		return "", payloadStatusNone
	}
	return trimPayload(strings.Join(blocks, "\n\n"), payloadLimit()), payloadStatusObserved
}

func formatInputBlock(path string, content string) string {
	return fmt.Sprintf("=== %s ===\n%s", path, content)
}

func finishedEvent(runID string, commandName string, source string, startedAt time.Time, finishedAt time.Time, input string, payloadStatus string, result app.Result, runErr error) RunEvent {
	kind := eventKindFinished
	if runErr != nil {
		kind = eventKindFailed
	}
	return RunEvent{
		Kind:          kind,
		ID:            runID,
		Command:       commandName,
		Source:        source,
		StartedAt:     startedAt,
		FinishedAt:    &finishedAt,
		Success:       runErr == nil,
		Changed:       result.Changed,
		Error:         errorMessage(runErr),
		PayloadStatus: payloadStatus,
		Input:         input,
		Result:        result,
		ResultSummary: summarizeResult(result, runErr),
	}
}

func summarizeResult(result app.Result, runErr error) string {
	if runErr != nil {
		return runErr.Error()
	}
	if result.ProducedOutput && !result.Changed && zeroSavings(result.Stats) {
		return "output generated without a tracked content delta"
	}
	stats := result.Stats
	return fmt.Sprintf(
		"changed=%t, saved %d bytes, %d words, %d approx tokens",
		result.Changed,
		stats.InputBytes-stats.OutputBytes,
		stats.InputWords-stats.OutputWords,
		stats.InputApproxTokens-stats.OutputApproxTokens,
	)
}

func trimObservedResult(result app.Result) app.Result {
	if result.Output == "" {
		return result
	}
	result.Output = trimPayload(result.Output, payloadLimit())
	return result
}

func trimPayload(payload string, limit int) string {
	if limit <= 0 || len(payload) <= limit {
		return payload
	}
	omitted := len(payload) - limit
	return payload[:limit] + fmt.Sprintf("\n... [truncated %d bytes] ...", omitted)
}

func zeroSavings(stats app.Stats) bool {
	return stats.InputBytes == stats.OutputBytes &&
		stats.InputWords == stats.OutputWords &&
		stats.InputApproxTokens == stats.OutputApproxTokens
}

func clearSession(root string) error {
	err := os.Remove(sessionPath(root))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
