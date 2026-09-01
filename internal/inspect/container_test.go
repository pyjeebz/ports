package inspect

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// fakeDocker serves handler on a unix socket and points the package's
// docker client at it for the duration of the test.
func fakeDocker(t *testing.T, handler http.Handler) {
	t.Helper()
	// t.TempDir embeds the test name; on darwin that can push the socket
	// path past the ~104-char sockaddr limit, so use a short temp dir.
	dir, err := os.MkdirTemp("", "ports")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sock := dir + "/docker.sock"
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewUnstartedServer(handler)
	_ = srv.Listener.Close()
	srv.Listener = l
	srv.Start()
	old := dockerSocket
	dockerSocket = sock
	t.Cleanup(func() {
		dockerSocket = old
		srv.Close()
	})
}

const containersJSON = `[
  {
    "Id": "aaaabbbbccccdddd",
    "Names": ["/shop-db-1"],
    "Image": "postgres:16-alpine",
    "Created": 1752300000,
    "Labels": {"com.docker.compose.project": "shop", "com.docker.compose.service": "db"},
    "Ports": [{"PrivatePort": 5432, "PublicPort": 5432, "Type": "tcp"}]
  },
  {
    "Id": "eeeeffff00001111",
    "Names": ["/lonely-redis"],
    "Image": "redis:7",
    "Created": 1752300000,
    "Labels": {},
    "Ports": [{"PrivatePort": 6379, "PublicPort": 16379, "Type": "tcp"}]
  }
]`

func TestEnrichDocker(t *testing.T) {
	fakeDocker(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/containers/json" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(containersJSON))
	}))

	services := []Service{
		{Port: 5432, Pid: 0, Process: "?"},                     // root-owned docker-proxy, non-root scan
		{Port: 16379, Pid: 42, Process: "docker-proxy"},        // visible docker-proxy
		{Port: 3000, Pid: 7, Process: "node", Cmdline: "node"}, // host process: untouched
	}
	enrichDocker(services, listContainers())

	db := services[0]
	if db.Container != "shop-db-1" || db.ContainerID != "aaaabbbbccccdddd" {
		t.Errorf("compose row not filled: %+v", db)
	}
	if db.Framework != "PostgreSQL" {
		t.Errorf("image detection: got framework %q, want PostgreSQL", db.Framework)
	}
	if db.ProjectName != "shop" || db.ComposeService != "db" {
		t.Errorf("compose labels: got project %q service %q", db.ProjectName, db.ComposeService)
	}
	if db.Started.IsZero() {
		t.Error("container Created should fill Started")
	}

	redis := services[1]
	if redis.Framework != "Redis" || redis.ProjectName != "lonely-redis" {
		t.Errorf("plain container: got framework %q project %q", redis.Framework, redis.ProjectName)
	}

	if host := services[2]; host.Container != "" || host.ProjectName != "" {
		t.Errorf("host process row was touched: %+v", host)
	}
}

func TestStopContainer(t *testing.T) {
	var gotPath, gotGrace, gotMethod string
	fakeDocker(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotGrace = r.Method, r.URL.Path, r.URL.Query().Get("t")
		w.WriteHeader(http.StatusNoContent)
	}))

	if err := StopContainer("aaaabbbbccccdddd", false); err != nil {
		t.Fatal(err)
	}
	if gotMethod != "POST" || gotPath != "/containers/aaaabbbbccccdddd/stop" || gotGrace != "2" {
		t.Errorf("got %s %s?t=%s", gotMethod, gotPath, gotGrace)
	}
	if err := StopContainer("aaaabbbbccccdddd", true); err != nil || gotGrace != "0" {
		t.Errorf("force: err=%v grace=%s, want t=0", err, gotGrace)
	}
}

func TestStopContainerAlreadyStopped(t *testing.T) {
	fakeDocker(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotModified)
	}))
	if err := StopContainer("aaaabbbbccccdddd", false); err != nil {
		t.Fatalf("304 must count as success: %v", err)
	}
}

func TestStopContainerError(t *testing.T) {
	fakeDocker(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no such container", http.StatusNotFound)
	}))
	if err := StopContainer("aaaabbbbccccdddd", false); err == nil {
		t.Fatal("expected error on 404")
	}
}

// Stop must route docker-backed services to the daemon even when the
// proxy pid is unknown, and must never signal anything.
func TestStopRoutesToDocker(t *testing.T) {
	stopped := false
	fakeDocker(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stopped = true
		w.WriteHeader(http.StatusNoContent)
	}))
	s := Service{Port: 5432, Pid: 0, Process: "?", ContainerID: "aaaabbbbccccdddd", Started: time.Now()}
	if err := Stop(s, false); err != nil {
		t.Fatal(err)
	}
	if !stopped {
		t.Fatal("Stop did not call the docker API")
	}
}

func TestListContainersNoDaemon(t *testing.T) {
	old := dockerSocket
	dockerSocket = t.TempDir() + "/absent.sock"
	t.Cleanup(func() { dockerSocket = old })
	if cs := listContainers(); cs != nil {
		t.Fatalf("expected nil without a daemon, got %d containers", len(cs))
	}
}

func TestImageBase(t *testing.T) {
	cases := map[string]string{
		"postgres":                "postgres",
		"postgres:16-alpine":      "postgres",
		"ghcr.io/acme/api:1.2":    "api",
		"localhost:5000/team/web": "web",
		"redis@sha256:deadbeef":   "redis",
		"mysql:8.4":               "mysql",
	}
	for ref, want := range cases {
		if got := ImageBase(ref); got != want {
			t.Errorf("ImageBase(%q) = %q, want %q", ref, got, want)
		}
	}
}

// detectRuntime fills the package vars from the environment and the
// well-known sockets; each case restores them after itself.
func TestDetectRuntime(t *testing.T) {
	oldSocket, oldRuntime := dockerSocket, containerRuntime
	t.Cleanup(func() { dockerSocket, containerRuntime = oldSocket, oldRuntime })
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("XDG_RUNTIME_DIR", "")

	podmanSock := mountSocket(t, "podman.sock")

	t.Run("docker default when no podman socket", func(t *testing.T) {
		t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
		detectRuntime()
		if containerRuntime != "docker" {
			t.Errorf("no podman socket present: got runtime %q", containerRuntime)
		}
	})

	t.Run("xdg podman socket detected", func(t *testing.T) {
		t.Setenv("XDG_RUNTIME_DIR", filepath.Dir(filepath.Dir(podmanSock)))
		detectRuntime()
		if containerRuntime != "podman" || dockerSocket != podmanSock {
			t.Errorf("got %q at %q, want podman at %q", containerRuntime, dockerSocket, podmanSock)
		}
	})

	t.Run("CONTAINER_RUNTIME narrows to docker even with podman present", func(t *testing.T) {
		t.Setenv("XDG_RUNTIME_DIR", filepath.Dir(filepath.Dir(podmanSock)))
		t.Setenv("CONTAINER_RUNTIME", "docker")
		detectRuntime()
		if containerRuntime != "docker" || dockerSocket != "/var/run/docker.sock" {
			t.Errorf("got %q at %q, want docker default", containerRuntime, dockerSocket)
		}
	})

	t.Run("DOCKER_HOST beats podman", func(t *testing.T) {
		t.Setenv("XDG_RUNTIME_DIR", filepath.Dir(filepath.Dir(podmanSock)))
		t.Setenv("DOCKER_HOST", "unix:///tmp/missing.sock")
		detectRuntime()
		if dockerSocket != "/tmp/missing.sock" {
			t.Errorf("got %q, want the DOCKER_HOST path", dockerSocket)
		}
	})

	t.Run("CONTAINER_RUNTIME names an engine ports doesn't know", func(t *testing.T) {
		t.Setenv("CONTAINER_RUNTIME", "nerdctl")
		detectRuntime()
		if containerRuntime != "nerdctl" || dockerSocket != "" {
			t.Errorf("got %q at %q, want nerdctl with no socket — an unknown engine has no well-known path", containerRuntime, dockerSocket)
		}
	})

	t.Run("CONTAINER_RUNTIME=podman absent has no path to name", func(t *testing.T) {
		t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
		t.Setenv("CONTAINER_RUNTIME", "podman")
		detectRuntime()
		if containerRuntime != "podman" || dockerSocket != "" {
			t.Errorf("got %q at %q, want podman with no socket — its path depends on the XDG dir, so naming one would be a guess", containerRuntime, dockerSocket)
		}
	})
}

// One transport per socket, for the socket's lifetime: every call — each
// --watch tick above all — must reuse it instead of stranding a fresh one.
func TestDockerTransportPooled(t *testing.T) {
	old := dockerSocket
	dockerSocket = "/pool-test.sock"
	t.Cleanup(func() { dockerSocket = old })

	same := dockerHTTP(2*time.Second).Transport
	if again := dockerHTTP(15 * time.Second).Transport; again != same {
		t.Error("dockerHTTP built a second transport for one socket")
	}
	other := "/pool-test-other.sock"
	if dockerTransport(other) == same {
		t.Error("different sockets must not share a transport")
	}
}

func TestStopContainerBodylessRefusal(t *testing.T) {
	fakeDocker(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // no body, no cause
	}))
	err := StopContainer("aaaabbbbccccdddd", false)
	if err == nil {
		t.Fatal("expected error on 500")
	}
	if !strings.Contains(err.Error(), "500 Internal Server Error") {
		t.Errorf("a body-less refusal should fall back to the status: %v", err)
	}
}

// probeRuntime classifies every way the engine can fail to answer, and
// doctor reports that classification — each case must survive a probe.
func TestProbeRuntime(t *testing.T) {
	old := dockerSocket
	t.Cleanup(func() { dockerSocket = old })

	t.Run("engine answers", func(t *testing.T) {
		fakeDocker(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_ping" {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.NotFound(w, r)
		}))
		if p := probeRuntime(); p.Status != RuntimeOK {
			t.Errorf("engine answering /_ping read as %q", p.Status)
		}
	})

	t.Run("no socket", func(t *testing.T) {
		dockerSocket = filepath.Join(t.TempDir(), "absent.sock")
		if p := probeRuntime(); p.Status != RuntimeNoSocket {
			t.Errorf("missing socket read as %q", p.Status)
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("unix socket permissions")
		}
		if os.Geteuid() == 0 {
			t.Skip("root ignores socket permissions")
		}
		sock := mountSocket(t, "denied.sock")
		if err := os.Chmod(sock, 0); err != nil {
			t.Fatal(err)
		}
		dockerSocket = sock
		if p := probeRuntime(); p.Status != RuntimePermission {
			t.Errorf("chmod-000 socket read as %q", p.Status)
		}
	})

	t.Run("daemon hung", func(t *testing.T) {
		if testing.Short() {
			t.Skip("waits out the 2s probe timeout")
		}
		dockerSocket = mountSocket(t, "hung.sock") // exists, accepts nothing, answers nothing
		if p := probeRuntime(); p.Status != RuntimeTimeout {
			t.Errorf("silent socket read as %q", p.Status)
		}
	})
}

// mountSocket listens on a fresh unix socket path so os.Stat reports a
// socket file; the listener stays open for the test's duration.
func mountSocket(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "podman")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	l, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return path
}
