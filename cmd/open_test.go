package cmd

import (
	"os"
	"slices"
	"testing"
)

// $BROWSER wins: its first entry, with %s standing for the URL, or the
// URL appended when there's no %s.
func TestBrowserCommandHonoursBROWSER(t *testing.T) {
	for _, c := range []struct {
		env  string
		want []string
	}{
		{"firefox", []string{"firefox", "http://localhost:3000/"}},
		{"firefox --new-tab", []string{"firefox", "--new-tab", "http://localhost:3000/"}},
		{"w3m %s -dump", []string{"w3m", "http://localhost:3000/", "-dump"}},
		// a list, in the platform's separator — ; on Windows, where : is in paths
		{"lynx" + string(os.PathListSeparator) + "firefox", []string{"lynx", "http://localhost:3000/"}},
	} {
		t.Setenv("BROWSER", c.env)
		cmd, err := browserCommand("http://localhost:3000/")
		if err != nil || !slices.Equal(cmd.Args, c.want) {
			t.Errorf("BROWSER=%q: got %v, %v — want %v", c.env, cmd.Args, err, c.want)
		}
	}
}

// A browser that can't start is an error with the URL kept, not a
// silent success.
func TestOpenReportsALaunchFailure(t *testing.T) {
	t.Setenv("BROWSER", "/nonexistent/browser")
	if err := launchBrowser("http://localhost:3000/"); err == nil {
		t.Error("a missing $BROWSER should fail")
	}
}
