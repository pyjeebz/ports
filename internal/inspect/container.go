package inspect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// dockerSocket and containerRuntime name the engine the enrichment and
// container stops talk to. Both are vars so tests can point the client
// at a fake daemon; detectRuntime fills the real values.
var (
	dockerSocket     = "/var/run/docker.sock"
	containerRuntime = "docker"
)

func init() { detectRuntime() }

// detectRuntime picks the socket to talk to. A DOCKER_HOST unix path is
// authoritative, as the docker CLI treats it; otherwise the well-known
// Podman sockets are probed before the Docker default. CONTAINER_RUNTIME
// narrows the probe to one engine — an explicit choice is honored even
// when nothing is found, and never silently falls through to docker's
// socket. Nothing found leaves the defaults, and scanning silently skips
// container enrichment, as it already does without a daemon.
func detectRuntime() {
	dockerSocket, containerRuntime = "/var/run/docker.sock", "docker"

	if prefer := strings.ToLower(os.Getenv("CONTAINER_RUNTIME")); prefer != "" {
		for _, c := range candidateSockets(prefer) {
			if socketExists(c) {
				containerRuntime, dockerSocket = prefer, c
				return
			}
		}
		// named engine, nothing found: keep the choice anyway. Docker
		// keeps its one well-known path so doctor can name where it
		// looked; podman's depends on the XDG dir and an unknown engine
		// has none at all, so both report a socket-less runtime rather
		// than naming a path that was only a guess.
		containerRuntime = prefer
		if prefer != "docker" {
			dockerSocket = ""
		}
		return
	}

	// nerdctl is not here on purpose: it fronts containerd, whose raw
	// API has no /containers/json — nothing to talk to without a proxy.
	if dh := unixPath(os.Getenv("DOCKER_HOST")); dh != "" {
		dockerSocket = dh // authoritative, even if currently unreachable
		return
	}
	for _, c := range candidateSockets("podman") {
		if socketExists(c) {
			containerRuntime, dockerSocket = "podman", c
			return
		}
	}
}

// candidateSockets lists the well-known Docker-API socket paths for one
// engine, XDG-relative first.
func candidateSockets(runtime string) []string {
	switch runtime {
	case "podman":
		var paths []string
		if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
			paths = append(paths, filepath.Join(dir, "podman", "podman.sock"))
		}
		return append(paths, "/var/run/podman/podman.sock")
	case "docker":
		return []string{unixPath(os.Getenv("DOCKER_HOST")), "/var/run/docker.sock"}
	default:
		return nil
	}
}

func socketExists(path string) bool {
	if path == "" {
		return false
	}
	st, err := os.Stat(path)
	return err == nil && st.Mode()&os.ModeSocket != 0
}

// unixPath turns a Docker-CLI-style DOCKER_HOST into a unix socket path,
// empty for anything else (tcp, npipe) — those engines are reachable
// over the network and this scanner only speaks sockets.
func unixPath(dockerHost string) string {
	if before, ok := strings.CutPrefix(dockerHost, "unix://"); ok {
		return before
	}
	return ""
}

// Container is the slice of the engine's /containers/json response that
// enrichment needs (docker, podman, or any Docker-API-compatible peer).
type Container struct {
	ID      string            `json:"Id"`
	Names   []string          `json:"Names"`
	Image   string            `json:"Image"`
	Created int64             `json:"Created"`
	Labels  map[string]string `json:"Labels"`
	Ports   []ContainerPort   `json:"Ports"`
}

type ContainerPort struct {
	PublicPort uint32 `json:"PublicPort"`
	Type       string `json:"Type"`
}

// dockerTransports pools one keep-alive transport per socket, so repeated
// calls — a --watch tick above all — reuse one connection instead of
// stranding a fresh transport, its socket, and its read loop per call.
var dockerTransports sync.Map // socket path → *http.Transport

func dockerHTTP(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: dockerTransport(dockerSocket),
	}
}

func dockerTransport(socket string) *http.Transport {
	if t, ok := dockerTransports.Load(socket); ok {
		return t.(*http.Transport)
	}
	t := &http.Transport{
		IdleConnTimeout: time.Minute,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socket)
		},
	}
	actual, _ := dockerTransports.LoadOrStore(socket, t)
	return actual.(*http.Transport)
}

// RuntimeStatus classifies the container engine's reachability. The scan
// itself is silent-fail — no daemon just means no enrichment — but doctor
// has to say *why* the engine is unreachable, and "no socket installed"
// and "socket there, permission denied" are different problems.
type RuntimeStatus string

const (
	RuntimeOK          RuntimeStatus = "ok"
	RuntimeNoSocket    RuntimeStatus = "no-socket"    // no engine installed
	RuntimePermission  RuntimeStatus = "permission"   // socket exists, access denied
	RuntimeTimeout     RuntimeStatus = "timeout"      // daemon hung
	RuntimeUnreachable RuntimeStatus = "unreachable"  // socket exists, no answer
)

// RuntimeProbe is the classified answer to "can ports reach the engine?".
type RuntimeProbe struct {
	Runtime string // docker, podman, … — the engine this machine prefers
	Socket  string
	Status  RuntimeStatus
}

// probeRuntime pings the engine over its socket and classifies the
// outcome. It follows detectRuntime's pick, including an authoritative
// DOCKER_HOST: a named socket that isn't there is worth reporting.
func probeRuntime() RuntimeProbe {
	p := RuntimeProbe{Runtime: containerRuntime, Socket: dockerSocket}
	if !socketExists(dockerSocket) {
		p.Status = RuntimeNoSocket
		return p
	}
	resp, err := dockerHTTP(2 * time.Second).Get("http://docker/_ping")
	if err == nil {
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == http.StatusOK {
			p.Status = RuntimeOK
		} else {
			p.Status = RuntimeUnreachable
		}
		return p
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		p.Status = RuntimeTimeout
		return p
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && errors.Is(opErr.Err, os.ErrPermission) {
		p.Status = RuntimePermission
		return p
	}
	p.Status = RuntimeUnreachable
	return p
}

// listContainers returns the running containers, or nil when no daemon
// is reachable — not an error, the enrichment just doesn't apply.
func listContainers() []Container {
	resp, err := dockerHTTP(2 * time.Second).Get("http://docker/containers/json")
	if err != nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var cs []Container
	if err := json.NewDecoder(resp.Body).Decode(&cs); err != nil {
		return nil
	}
	return cs
}

// StopContainer asks the engine to stop a container — engine-side
// SIGTERM, grace period, then SIGKILL, the same contract as Stop. force
// cuts the grace period to zero.
func StopContainer(id string, force bool) error {
	grace := 2
	if force {
		grace = 0
	}
	url := fmt.Sprintf("http://docker/containers/%s/stop?t=%d", id, grace)
	resp, err := dockerHTTP(15*time.Second).Post(url, "", nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	// 304 = already stopped, which is what the user wanted anyway
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotModified {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	detail := strings.TrimSpace(string(body))
	if detail == "" {
		detail = resp.Status // a body-less refusal still deserves a cause
	}
	return fmt.Errorf("%s: stop %s: %s", containerRuntime, shortID(id), detail)
}

// enrichDocker fills docker-proxy and owner-unknown rows with the
// container really behind the port. It asks the daemon, not the process
// table, so it works without sudo even though docker-proxy is root-owned.
func enrichDocker(services []Service, containers []Container) {
	if len(containers) == 0 {
		return
	}
	byPort := map[uint32]*Container{}
	for i := range containers {
		for _, p := range containers[i].Ports {
			if p.PublicPort != 0 && p.Type == "tcp" {
				byPort[p.PublicPort] = &containers[i]
			}
		}
	}
	for i := range services {
		s := &services[i]
		if s.Known() && s.Process != "docker-proxy" {
			continue // a real host process owns this port
		}
		c := byPort[s.Port]
		if c == nil {
			continue
		}
		s.Runtime = containerRuntime
		s.Container = containerName(c)
		s.ContainerID = c.ID
		s.Image = c.Image
		s.ComposeProject = c.Labels["com.docker.compose.project"]
		s.ComposeService = c.Labels["com.docker.compose.service"]
		if s.ComposeProject != "" {
			s.ProjectName = s.ComposeProject
		} else {
			s.ProjectName = s.Container
		}
		if s.Framework == "" {
			s.Framework = DetectFramework(ImageBase(c.Image), c.Image)
		}
		if s.Started.IsZero() && c.Created > 0 {
			s.Started = time.Unix(c.Created, 0)
		}
	}
}

func containerName(c *Container) string {
	if len(c.Names) == 0 {
		return shortID(c.ID)
	}
	return strings.TrimPrefix(c.Names[0], "/")
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// ImageBase reduces an image ref to the name a developer recognizes:
// ghcr.io/acme/api:1.2 → api, postgres@sha256:… → postgres.
func ImageBase(ref string) string {
	base := ref
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	if i := strings.IndexAny(base, ":@"); i >= 0 {
		base = base[:i]
	}
	return base
}
