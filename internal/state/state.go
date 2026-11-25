// Package state is where ports keeps what outlives one run: the names
// it has handed out and the router's whereabouts. Nothing in it is
// configuration — deleting the directory loses nothing a user wrote.
package state

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// Dir is the per-user state directory: $PORTS_STATE_DIR when set (tests,
// unusual setups), else $XDG_STATE_HOME/ports, else the platform's own —
// ~/.local/state/ports, ~/Library/Application Support/ports on macOS,
// %LocalAppData%\ports on Windows.
func Dir() (string, error) {
	if dir := os.Getenv("PORTS_STATE_DIR"); dir != "" {
		return dir, nil
	}
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" && runtime.GOOS != "windows" {
		return filepath.Join(dir, "ports"), nil
	}
	switch runtime.GOOS {
	case "windows":
		if dir := os.Getenv("LocalAppData"); dir != "" {
			return filepath.Join(dir, "ports"), nil
		}
	case "darwin":
		if dir, err := os.UserConfigDir(); err == nil {
			return filepath.Join(dir, "ports"), nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "ports"), nil
}

// Path is a file inside Dir.
func Path(name string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

// Read decodes a JSON state file into v. A file that doesn't exist yet
// is not an error: v keeps its zero state.
func Read(name string, v any) error {
	path, err := Path(name)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// Write stores v as a JSON state file, user-only, replacing the old one
// in one rename so a reader never sees half a file.
func Write(name string, v any) error {
	path, err := Path(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), name+".*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Remove deletes a state file; one already gone is not an error.
func Remove(name string) error {
	path, err := Path(name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// Glob is the state files whose names match pattern (filepath.Match
// syntax), by name.
func Glob(pattern string) ([]string, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	paths, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return nil, err
	}
	names := make([]string, len(paths))
	for i, p := range paths {
		names[i] = filepath.Base(p)
	}
	return names, nil
}
