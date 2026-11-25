package router

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/pyjeebz/ports/internal/inspect"
)

// Diagnose is what bare `ports doctor` says about names: whether a router
// runs and where, and whether this machine's own resolver finds
// *.localhost names — browsers and curl always do, but Node's fetch,
// Python and Go ask the system, and on macOS or a Linux whose
// resolv.conf skips systemd-resolved's stub they get nothing. Every
// finding is info: none of it stops a dev server from working, and a
// machine that never uses names shouldn't fail doctor over them. named is
// how many dev servers have a name right now.
func Diagnose(named int) []inspect.Finding {
	var f []inspect.Finding
	info, running := Running()
	switch {
	case running && info.Port == DefaultPort:
		f = append(f, inspect.Finding{Check: "router", Status: inspect.CheckOK,
			Message: fmt.Sprintf("routing *.localhost on :%d (pid %d), %d dev server(s) by name", info.Port, info.Pid, named)})
	case running:
		f = append(f, inspect.Finding{Check: "router", Status: inspect.CheckInfo,
			Message: fmt.Sprintf("routing *.localhost on :%d (pid %d) — port 80 wasn't available, so names carry :%d", info.Port, info.Pid, info.Port),
			Fix:     port80Hint()})
	case named > 0:
		f = append(f, inspect.Finding{Check: "router", Status: inspect.CheckInfo,
			Message: fmt.Sprintf("not running — %d dev server(s) have names but no router serves them", named),
			Fix:     "`ports proxy` serves them at http://<name>.localhost"})
	default:
		return f // no router, nothing named: names aren't in use here
	}
	if !resolvesLocalhost() {
		f = append(f, inspect.Finding{Check: "names", Status: inspect.CheckInfo,
			Message: "this machine's resolver doesn't find *.localhost — browsers and curl reach names, but Node's fetch, Python and Go clients don't",
			Fix:     "scripts and agents: `ports url --local <name>` prints an address every client reaches"})
	}
	return f
}

// resolvesLocalhost asks the system resolver for a name under .localhost,
// the way a non-browser client would.
func resolvesLocalhost() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(ctx, "ports-doctor.localhost")
	if err != nil {
		return false
	}
	for _, a := range addrs {
		if ip := net.ParseIP(a); ip != nil && ip.IsLoopback() {
			return true
		}
	}
	return false
}

// port80Hint is the one-time step to give the router port 80, where
// there is one: on Linux, ports below 1024 need a capability.
func port80Hint() string {
	return "on Linux: `sudo setcap cap_net_bind_service=+ep \"$(command -v ports)\"`, then restart the router; elsewhere, `ports 80` shows what holds it"
}
