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

	// docker context, filled from the daemon when the port is really a
	// container behind docker-proxy
	Container      string `json:"container,omitempty"`
	ContainerID    string `json:"container_id,omitempty"`
	Image          string `json:"image,omitempty"`
	ComposeProject string `json:"compose_project,omitempty"`
	ComposeService string `json:"compose_service,omitempty"`

	// windows-side listener, seen across the WSL2 boundary; WindowsPid
	// is a Windows pid — never a target for kill
	Windows    bool  `json:"windows,omitempty"`
	WindowsPid int32 `json:"windows_pid,omitempty"`
}

// Known reports whether the owning process could be identified.
// Without root, sockets of other users' processes come back with Pid 0.
func (s Service) Known() bool { return s.Pid > 0 }

// MatchesName reports an exact, case-insensitive match on the handles a
// developer stops things by: project name, framework, container or
// compose-service name, or process name. On docker rows the process
// (docker-proxy, ?) is deliberately not a handle — matching it would
// select every published container at once.
func (s Service) MatchesName(q string) bool {
	if (s.ProjectName != "" && strings.EqualFold(q, s.ProjectName)) ||
		(s.Framework != "" && strings.EqualFold(q, s.Framework)) {
		return true
	}
	if s.Container != "" {
		return strings.EqualFold(q, s.Container) ||
			(s.ComposeService != "" && strings.EqualFold(q, s.ComposeService))
	}
	return strings.EqualFold(q, s.Process)
}

// MatchesQuery reports a case-insensitive substring match across the
// recognizable context, for `ports find`.
func (s Service) MatchesQuery(q string) bool {
	q = strings.ToLower(q)
	for _, hay := range []string{
		s.Framework, s.Process, s.ProjectName, s.Project,
		s.Container, s.Image, s.ComposeProject, s.ComposeService,
	} {
		if hay != "" && strings.Contains(strings.ToLower(hay), q) {
			return true
		}
	}
	return false
}
