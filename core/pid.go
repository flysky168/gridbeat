package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// createPidFile checks whether a process recorded in pid file exists.
// If exists -> return error.
// If not exists (or stale pid file) -> write current pid to the file.
//
// Notes:
//   - On Unix, it uses kill(pid, 0) to probe existence/permission.
//   - On Windows, syscall-based probing isn't provided here; it falls back to "best effort":
//     if pid file exists and is non-empty, it returns an error to avoid double-run.
//     (You can implement Windows-specific process checks if needed.)
func CreatePidFile(path string) error {
	if path == "" {
		return errors.New("pid file path is empty")
	}

	// Ensure parent dir exists
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create pid dir: %w", err)
		}
	}

	// Read existing pid if file exists
	if b, err := os.ReadFile(path); err == nil {
		s := strings.TrimSpace(string(b))
		if s != "" {
			oldPID, err := strconv.Atoi(s)
			if err != nil || oldPID <= 0 {
				// Corrupt pidfile, treat as stale; overwrite
			} else {
				exists, probeErr := processExists(oldPID)
				if probeErr != nil {
					return fmt.Errorf("probe existing pid %d: %w", oldPID, probeErr)
				}
				if exists {
					return fmt.Errorf("process already running (pid=%d) from pidfile %s", oldPID, path)
				}
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read pidfile: %w", err)
	}

	// Write current pid (atomic replace)
	pid := os.Getpid()
	tmp := fmt.Sprintf("%s.tmp.%d", path, pid)
	if err := os.WriteFile(tmp, []byte(strconv.Itoa(pid)+"\n"), 0o644); err != nil {
		return fmt.Errorf("write temp pidfile: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename pidfile: %w", err)
	}
	return nil
}

func RemovePidFile(path string) {
	_ = os.Remove(path)
}

func processExists(pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}

	// Best-effort fallback for Windows in this snippet.
	if runtime.GOOS == "windows" {
		// Safer default: if pidfile says someone is running, refuse.
		// Replace with a real OpenProcess/GetExitCodeProcess check if you need it.
		return true, nil
	}

	// On Unix: kill(pid, 0) checks existence without sending a signal.
	err := syscall.Kill(pid, 0)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, syscall.ESRCH):
		return false, nil // no such process
	case errors.Is(err, syscall.EPERM):
		return true, nil // exists but no permission
	default:
		return false, err
	}
}
