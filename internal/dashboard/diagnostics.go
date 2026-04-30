package dashboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const diagnosticsFileName = "diagnostics.jsonl"

type diagnosticEntry struct {
	RecordedAt time.Time `json:"recordedAt"`
	Message    string    `json:"message"`
}

func recordDiagnostic(root string, message string) error {
	baseDir := root
	if info, err := os.Stat(root); err == nil && !info.IsDir() {
		baseDir = filepath.Dir(root)
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(filepath.Join(baseDir, diagnosticsFileName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	payload, err := json.Marshal(diagnosticEntry{
		RecordedAt: time.Now().UTC(),
		Message:    message,
	})
	if err != nil {
		return err
	}
	_, err = file.Write(append(payload, '\n'))
	return err
}
