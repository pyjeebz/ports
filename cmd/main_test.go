package cmd

import (
	"os"
	"testing"
)

// TestMain keeps the names the commands hand out — and any router record
// — in a throwaway state directory, never the developer's own.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "ports-state-")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("PORTS_STATE_DIR", dir)
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
