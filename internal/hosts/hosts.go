// Package hosts keeps the names ports hands out in the system hosts file,
// for the clients that don't resolve *.localhost on their own: browsers
// and curl send every .localhost name to loopback, but Node, Python and Go
// ask the system resolver, which on Windows — and on a Linux whose
// resolv.conf skips systemd-resolved's stub — knows nothing of them. The
// names live in one marked block; nothing outside it is ever touched.
package hosts

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

const (
	begin = "# >>> ports: dev server names (ports hosts sync writes this block, ports hosts clean removes it) >>>"
	end   = "# <<< ports <<<"
)

// Path is the system hosts file, or $PORTS_HOSTS_FILE when set (tests).
func Path() string {
	if p := os.Getenv("PORTS_HOSTS_FILE"); p != "" {
		return p
	}
	if runtime.GOOS == "windows" {
		root := os.Getenv("SystemRoot")
		if root == "" {
			root = `C:\Windows`
		}
		return filepath.Join(root, "System32", "drivers", "etc", "hosts")
	}
	return "/etc/hosts"
}

// Names is what the block in data holds, sorted: the names, without
// .localhost.
func Names(data []byte) []string {
	var names []string
	in := false
	for line := range strings.Lines(string(data)) {
		line = strings.TrimSpace(line)
		switch {
		case line == begin:
			in = true
		case line == end:
			in = false
		case in && line != "" && !strings.HasPrefix(line, "#"):
			fields := strings.Fields(line)
			for _, h := range fields[1:] {
				if name, ok := strings.CutSuffix(h, ".localhost"); ok && !slices.Contains(names, name) {
					names = append(names, name)
				}
			}
		}
	}
	slices.Sort(names)
	return names
}

// Replace is data with its block holding exactly names — added at the
// end when there was none, removed when names is empty. Every line
// outside the block passes through untouched.
func Replace(data []byte, names []string) []byte {
	var out bytes.Buffer
	in, wrote := false, false
	block := func() {
		if len(names) == 0 {
			return
		}
		out.WriteString(begin + "\n")
		for _, n := range names {
			for _, ip := range []string{"127.0.0.1", "::1"} {
				out.WriteString(ip)
				out.WriteString("\t")
				out.WriteString(n)
				out.WriteString(".localhost\n")
			}
		}
		out.WriteString(end + "\n")
	}
	for line := range strings.Lines(string(data)) {
		switch strings.TrimSpace(line) {
		case begin:
			in = true
			if len(names) == 0 && bytes.HasSuffix(out.Bytes(), []byte("\n\n")) {
				out.Truncate(out.Len() - 1) // the blank line sync put before the block
			}
			continue
		case end:
			if in {
				in = false
				if !wrote {
					block()
					wrote = true
				}
				continue
			}
		}
		if in {
			continue
		}
		out.WriteString(line)
		if !strings.HasSuffix(line, "\n") {
			out.WriteString("\n")
		}
	}
	if !wrote && len(names) > 0 {
		if out.Len() > 0 && !bytes.HasSuffix(out.Bytes(), []byte("\n\n")) {
			out.WriteString("\n")
		}
		block()
	}
	return out.Bytes()
}

// Write replaces the block in the hosts file at path with names. It
// writes a new file beside the old and renames it into place, so a reader
// never sees half a hosts file; where the file can't be replaced — a
// bind mount, as in a container — it's rewritten in place.
func Write(path string, names []string) error {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	next := Replace(data, names)
	if bytes.Equal(next, data) {
		return nil
	}
	mode := fs.FileMode(0o644)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".hosts-ports-*")
	if err == nil {
		name := tmp.Name()
		_, werr := tmp.Write(next)
		cerr := tmp.Close()
		if werr == nil && cerr == nil && os.Chmod(name, mode) == nil && os.Rename(name, path) == nil {
			return nil
		}
		_ = os.Remove(name)
	}
	return os.WriteFile(path, next, mode)
}

// Writable reports whether this process may change the hosts file.
func Writable(path string) bool {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return false
	}
	_ = f.Close()
	dir, err := os.CreateTemp(filepath.Dir(path), ".hosts-ports-check-*")
	if err != nil {
		return false
	}
	_ = dir.Close()
	_ = os.Remove(dir.Name())
	return true
}
