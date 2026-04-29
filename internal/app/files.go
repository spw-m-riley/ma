package app

import (
	"os"
	"path/filepath"
)

func WriteWithBackup(path string, original string, output string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	backupPath := path + ".ma.bak"
	tempPath := path + ".ma.tmp"

	if err := os.WriteFile(backupPath, []byte(original), info.Mode().Perm()); err != nil {
		return err
	}

	restore := func(writeErr error) error {
		_ = os.Remove(tempPath)
		_ = os.WriteFile(path, []byte(original), info.Mode().Perm())
		return writeErr
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return restore(err)
	}
	if err := os.WriteFile(tempPath, []byte(output), info.Mode().Perm()); err != nil {
		return restore(err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return restore(err)
	}

	return nil
}
