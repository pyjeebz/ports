package naming

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/pyjeebz/ports/internal/state"
)

// claimsFile holds which repository owns which base name.
const claimsFile = "names.json"

// Claims remember which repository a base name belongs to, so two
// projects that are both called shop never swap names: the first one
// seen keeps shop for good, the second becomes shop-<its parent folder>,
// and a restart, a reboot or a new scan order changes neither.
type Claims struct {
	// Repos maps a repository's root to the base name it was given.
	Repos map[string]string `json:"repos"`
	dirty bool
}

// LoadClaims reads the remembered claims. An unreadable file starts
// fresh rather than failing a listing: the worst case is a project
// claiming its name again.
func LoadClaims() *Claims {
	c := &Claims{}
	if err := state.Read(claimsFile, c); err != nil {
		c = &Claims{}
	}
	if c.Repos == nil {
		c.Repos = map[string]string{}
	}
	return c
}

// Save writes the claims back when Claim handed out something new.
func (c *Claims) Save() error {
	if !c.dirty {
		return nil
	}
	if err := state.Write(claimsFile, c); err != nil {
		return err
	}
	c.dirty = false
	return nil
}

// Claim is the base name repo goes by. A repository keeps the name it
// was given before; a new one gets want, or — when another repository
// already holds want — want with its parent folder added (shop-client),
// then a count. A holder whose folder is gone gives its name up: a moved
// or deleted project doesn't keep a name from the next one.
func (c *Claims) Claim(repo, want string) string {
	if want == "" {
		return ""
	}
	if c.Repos == nil {
		c.Repos = map[string]string{}
	}
	if name, ok := c.Repos[repo]; ok {
		return name
	}
	name := want
	if holder := c.holder(name); holder != "" {
		name = join2(want, Label(filepath.Base(filepath.Dir(repo))))
		for n := 2; c.holder(name) != ""; n++ {
			name = join2(want, Label(filepath.Base(filepath.Dir(repo))), strconv.Itoa(n))
		}
	}
	c.Repos[repo] = name
	c.dirty = true
	return name
}

// holder is the live repository holding name, releasing any claim whose
// folder no longer exists.
func (c *Claims) holder(name string) string {
	var repos []string
	for repo, n := range c.Repos {
		if n == name {
			repos = append(repos, repo)
		}
	}
	slices.Sort(repos)
	for _, repo := range repos {
		if _, err := os.Stat(repo); err != nil {
			delete(c.Repos, repo)
			c.dirty = true
			continue
		}
		return repo
	}
	return ""
}

func join2(parts ...string) string {
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		if out != "" {
			out += "-"
		}
		out += p
	}
	return out
}
