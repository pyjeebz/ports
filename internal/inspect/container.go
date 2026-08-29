package inspect

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
// narrows the probe to one engine — an explicit choice should not
// silently fall through to docker's socket. Nothing found leaves the
// defaults, and scanning silently skips container enrichment, as it
// already does without a daemon.
func detectRuntime() {
	dockerSocket, containerRuntime = "/var/run/docker.sock", "docker"

	if prefer := strings.ToLower(os.Getenv("CONTAINER_RUNTIME")); prefer != "" {
		for _, c := range candidateSockets(prefer) {
			if socketExists(c) {
				containerRuntime, dockerSocket = prefer, c
				return
			}
		}
		return // named engine only; don't fall through
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

func dockerHTTP(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", dockerSocket)
			},
		},
	}
}

// listContainers returns the running containers, or nil when no daemon
// is reachable — not an error, the enrichment just doesn't apply.
func listContainers() []Container {
	resp, err := dockerHTTP(2 * time.Second).Get("http://docker/containers/json")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
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
	defer resp.Body.Close()
	// 304 = already stopped, which is what the user wanted anyway
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotModified {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("%s: stop %s: %s", containerRuntime, shortID(id), strings.TrimSpace(string(body)))
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
