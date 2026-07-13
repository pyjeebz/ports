package inspect

import "time"

// Service is one listening TCP socket with the context a developer
// recognizes: what the process is, which project it belongs to, and
// how long it has been running.
type Service struct {
	Port    uint32    `json:"port"`
	Pid     int32     `json:"pid"`
	Process string    `json:"process"`
	Cmdline string    `json:"cmdline,omitempty"`
	Addr    string    `json:"addr"`
	Cwd     string    `json:"cwd,omitempty"`
	Project string    `json:"project,omitempty"`
	Started time.Time `json:"started,omitzero"`
}

// Known reports whether the owning process could be identified.
// Without root, sockets of other users' processes come back with Pid 0.
func (s Service) Known() bool { return s.Pid > 0 }
