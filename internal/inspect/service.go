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
	ParentPid   int32     `json:"ppid,omitempty"`
	Process     string    `json:"process"`
	Framework   string    `json:"framework,omitempty"`
	Cmdline     string    `json:"cmdline,omitempty"`
	Addr        string    `json:"addr"`
	Cwd         string    `json:"cwd,omitempty"`
	Project     string    `json:"project,omitempty"`
	ProjectName string    `json:"project_name,omitempty"`
	Started     time.Time `json:"started,omitzero"`

	// container context, filled from the engine when the port is really
	// a container behind a proxy; Runtime names the engine (docker,
	// podman, …) the row came from
	Runtime        string `json:"runtime,omitempty"`
	Container      string `json:"container,omitempty"`
	ContainerID    string `json:"container_id,omitempty"`
	Image          string `json:"image,omitempty"`
	ComposeProject string `json:"compose_project,omitempty"`
	ComposeService string `json:"compose_service,omitempty"`

	// windows-side listener, seen across the WSL2 boundary; WindowsPid
	// is a Windows pid — never a target for kill
	Windows    bool  `json:"windows,omitempty"`
	WindowsPid int32 `json:"windows_pid,omitempty"`

	// another user's listener, which a scan without root sees with no
	// pid: User owns the socket, and Unit is the systemd service it was
	// traced to through that user's processes. Pid stays 0 — the row
	// names the service, not which of its processes holds the port.
	User string `json:"user,omitempty"`
	Unit string `json:"unit,omitempty"`

	ownerUID uint32 // the socket's owner, from /proc/net/tcp
	hasOwner bool
}

// Known reports whether the owning process could be identified.
// Without root, sockets of other users' processes come back with Pid 0.
func (s Service) Known() bool { return s.Pid > 0 }

// unowned reports a host row no pid backs — another user's listener,
// named or not. Container and Windows rows have their own owners.
func (s Service) unowned() bool {
	return !s.Known() && s.Container == "" && !s.Windows
}

// Anonymous reports a row nothing could name: shown as ?, stoppable
// only after sudo reveals it.
func (s Service) Anonymous() bool { return s.unowned() && s.Process == "?" }

// MatchesName reports an exact, case-insensitive match on the handles a
// developer stops things by: project name, framework, container or
// compose-service name, or process name. On docker rows the process
// (docker-proxy, ?) is deliberately not a handle — matching it would
// select every published container at once.
func (s Service) MatchesName(q string) bool {
	if (s.ProjectName != "" && strings.EqualFold(q, s.ProjectName)) ||
		(s.Framework != "" && FrameworkKey(q) == FrameworkKey(s.Framework)) ||
		(s.Unit != "" && strings.EqualFold(q, s.Unit)) {
		return true
	}
	if s.Container != "" {
		return strings.EqualFold(q, s.Container) ||
			(s.ComposeService != "" && strings.EqualFold(q, s.ComposeService))
	}
	return strings.EqualFold(q, s.Process)
}

// MatchesQuery reports a case-insensitive substring match across the
// recognizable context, for `ports find`. The framework also matches in
// its typed form, so `ports find nextjs` finds Next.js.
func (s Service) MatchesQuery(q string) bool {
	if key := FrameworkKey(q); key != "" && strings.Contains(FrameworkKey(s.Framework), key) {
		return true
	}
	q = strings.ToLower(q)
	for _, hay := range []string{
		s.Framework, s.Process, s.ProjectName, s.Project,
		s.Container, s.Image, s.ComposeProject, s.ComposeService, s.Unit,
	} {
		if hay != "" && strings.Contains(strings.ToLower(hay), q) {
			return true
		}
	}
	return false
}
