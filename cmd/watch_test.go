package cmd

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestValidateWatch(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		format   string
		jsonOut  bool
		tty      bool
		wantErr  bool
		wantCode int
	}{
		{name: "table on a terminal", interval: time.Second, format: "table", tty: true},
		{name: "json anywhere", interval: time.Second, format: "json"},
		{name: "json in a pipe", interval: time.Second, format: "json"},
		{name: "--json in a pipe", interval: time.Second, jsonOut: true},

		{name: "interval below minimum", interval: 100 * time.Millisecond, format: "json", wantErr: true, wantCode: 2},
		{name: "interval at minimum", interval: minWatchInterval, format: "json"},

		{name: "csv was cut: an unknown format", interval: time.Second, format: "csv", wantErr: true, wantCode: 2},
		{name: "tsv was cut: an unknown format", interval: time.Second, format: "tsv", wantErr: true, wantCode: 2},
		{name: "table in a pipe", interval: time.Second, format: "table", tty: false, wantErr: true, wantCode: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWatch(tt.interval, tt.format, tt.jsonOut, tt.tty)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("validateWatch(%s, %q, tty=%v) = %v, want nil", tt.interval, tt.format, tt.tty, err)
				}
				return
			}
			var ee exitError
			if !errors.As(err, &ee) {
				t.Fatalf("validateWatch(%s, %q, tty=%v) = %v, want exitError", tt.interval, tt.format, tt.tty, err)
			}
			if ee.code != tt.wantCode {
				t.Errorf("exit code = %d, want %d (err: %v)", ee.code, tt.wantCode, err)
			}
		})
	}
}

func TestWatchLoopStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const want = 3
	var n int
	err := watchLoop(ctx, time.Millisecond, func() error {
		n++
		if n == want {
			cancel()
		}
		return nil
	})
	if err != nil {
		t.Fatalf("watchLoop = %v, want nil", err)
	}
	if n != want {
		t.Errorf("ticks = %d, want %d", n, want)
	}
}

func TestWatchLoopPropagatesTickError(t *testing.T) {
	sentinel := errors.New("scan failed")
	err := watchLoop(context.Background(), time.Millisecond, func() error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("watchLoop = %v, want %v", err, sentinel)
	}
}
