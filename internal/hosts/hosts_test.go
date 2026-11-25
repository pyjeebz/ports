package hosts

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

const system = "127.0.0.1\tlocalhost\n::1\tlocalhost ip6-localhost\n# a comment the user wrote\n10.0.0.5 nas.home\n"

func TestReplaceAddsABlock(t *testing.T) {
	out := string(Replace([]byte(system), []string{"api.shop", "shop"}))
	if !strings.HasPrefix(out, system) {
		t.Errorf("the lines outside the block changed:\n%s", out)
	}
	for _, want := range []string{"127.0.0.1\tshop.localhost\n", "::1\tshop.localhost\n", "127.0.0.1\tapi.shop.localhost\n", begin, end} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if got := Names([]byte(out)); !slices.Equal(got, []string{"api.shop", "shop"}) {
		t.Errorf("Names = %v", got)
	}
}

func TestReplaceRewritesInPlace(t *testing.T) {
	first := Replace([]byte(system+"# after\n"), []string{"shop"})
	// a line the user adds after the block stays after it
	withTail := strings.Replace(string(first), end+"\n", end+"\n192.168.1.9 printer\n", 1)
	second := string(Replace([]byte(withTail), []string{"blog", "shop"}))
	if strings.Count(second, begin) != 1 {
		t.Fatalf("two blocks:\n%s", second)
	}
	if !strings.Contains(second, end+"\n192.168.1.9 printer\n") {
		t.Errorf("the line after the block moved:\n%s", second)
	}
	if got := Names([]byte(second)); !slices.Equal(got, []string{"blog", "shop"}) {
		t.Errorf("Names = %v", got)
	}
}

func TestReplaceRemoves(t *testing.T) {
	with := Replace([]byte(system), []string{"shop"})
	if out := string(Replace(with, nil)); out != system {
		t.Errorf("clean left:\n%q\nwant the file as it was:\n%q", out, system)
	}
}

func TestReplaceNoTrailingNewline(t *testing.T) {
	out := string(Replace([]byte("127.0.0.1 localhost"), []string{"shop"}))
	if !strings.HasPrefix(out, "127.0.0.1 localhost\n") {
		t.Errorf("the last line ran into the block:\n%s", out)
	}
}

func TestWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hosts")
	if err := os.WriteFile(path, []byte(system), 0o644); err != nil {
		t.Fatal(err)
	}
	if !Writable(path) {
		t.Fatal("a temp file isn't writable")
	}
	if err := Write(path, []string{"shop"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if got := Names(data); !slices.Equal(got, []string{"shop"}) {
		t.Errorf("after Write: %v\n%s", got, data)
	}
	// Windows keeps no Unix permission bits to carry over
	if st, _ := os.Stat(path); runtime.GOOS != "windows" && st.Mode().Perm() != 0o644 {
		t.Errorf("mode %v, want the file's own 0644", st.Mode().Perm())
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("left temp files behind: %v", entries)
	}
}
