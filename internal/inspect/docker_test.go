package inspect

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeDocker serves handler on a unix socket and points the package's
// docker client at it for the duration of the test.
func fakeDocker(t *testing.T, handler http.Handler) {
	t.Helper()
	sock := t.TempDir() + "/docker.sock"
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewUnstartedServer(handler)
	srv.Listener.Close()
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
		w.Write([]byte(containersJSON))
	}))

	services := []Service{
		{Port: 5432, Pid: 0, Process: "?"},                    // root-owned docker-proxy, non-root scan
		{Port: 16379, Pid: 42, Process: "docker-proxy"},       // visible docker-proxy
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
		"postgres":                   "postgres",
		"postgres:16-alpine":         "postgres",
		"ghcr.io/acme/api:1.2":       "api",
		"localhost:5000/team/web":    "web",
		"redis@sha256:deadbeef":      "redis",
		"mysql:8.4":                  "mysql",
	}
	for ref, want := range cases {
		if got := ImageBase(ref); got != want {
			t.Errorf("ImageBase(%q) = %q, want %q", ref, got, want)
		}
	}
}
