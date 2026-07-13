package inspect

import (
	"strings"
	"time"
)

// Service is one listening TCP socket with the context a developer
// recognizes: what the process is, which project it belongs to, and
// how long it has been running.
type Service struct {
	Port        uint32    `json:"port"`
	Pid         int32     `json:"pid"`
	Process     string    `json:"process"`
	Framework   string    `json:"framework,omitempty"`
	Cmdline     string    `json:"cmdline,omitempty"`
	Addr        string    `json:"addr"`
	Cwd         string    `json:"cwd,omitempty"`
	Project     string    `json:"project,omitempty"`
	ProjectName string    `json:"project_name,omitempty"`
	Started     time.Time `json:"started,omitzero"`
}

// Known reports whether the owning process could be identified.
// Without root, sockets of other users' processes come back with Pid 0.
func (s Service) Known() bool { return s.Pid > 0 }

// MatchesName reports an exact, case-insensitive match on the handles a
// developer stops things by: project name, framework, or process name.
func (s Service) MatchesName(q string) bool {
	return (s.ProjectName != "" && strings.EqualFold(q, s.ProjectName)) ||
		(s.Framework != "" && strings.EqualFold(q, s.Framework)) ||
		strings.EqualFold(q, s.Process)
}

// MatchesQuery reports a case-insensitive substring match across the
// recognizable context, for `ports find`.
func (s Service) MatchesQuery(q string) bool {
	q = strings.ToLower(q)
	for _, hay := range []string{s.Framework, s.Process, s.ProjectName, s.Project} {
		if hay != "" && strings.Contains(strings.ToLower(hay), q) {
			return true
		}
	}
	return false
}
