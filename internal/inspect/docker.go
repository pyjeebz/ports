package inspect

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// dockerSocket is a var so tests can point the client at a fake daemon.
var dockerSocket = "/var/run/docker.sock"

// Container is the slice of dockerd's /containers/json response that
// enrichment needs.
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

// StopContainer asks dockerd to stop a container — engine-side SIGTERM,
// grace period, then SIGKILL, the same contract as Stop. force cuts the
// grace period to zero.
func StopContainer(id string, force bool) error {
	grace := 2
	if force {
		grace = 0
	}
	url := fmt.Sprintf("http://docker/containers/%s/stop?t=%d", id, grace)
	resp, err := dockerHTTP(15 * time.Second).Post(url, "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// 304 = already stopped, which is what the user wanted anyway
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotModified {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("docker: stop %s: %s", shortID(id), strings.TrimSpace(string(body)))
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
