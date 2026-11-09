package inspect

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"

	gnet "github.com/shirou/gopsutil/v4/net"
	gproc "github.com/shirou/gopsutil/v4/process"
)

// Finding statuses — what WriteFindings draws for each line. Info is an
// observation with nothing to do about it: shown, never a problem.
const (
	CheckOK   = "ok"
	CheckInfo = "info"
	CheckWarn = "warn"
	CheckFail = "fail"
)

// Finding is one doctor observation: what was checked, what was found,
// and — when something is wrong — the way out. Doctor never acts; Fix is
// the user's next command, never something ports runs.
type Finding struct {
	Check   string `json:"check"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Fix     string `json:"fix,omitempty"`
}

func ok(check, msg string) Finding {
	return Finding{Check: check, Status: CheckOK, Message: msg}
}

func info(check, msg string) Finding {
	return Finding{Check: check, Status: CheckInfo, Message: msg}
}

func warn(check, msg, fix string) Finding {
	return Finding{Check: check, Status: CheckWarn, Message: msg, Fix: fix}
}

func fail(check, msg, fix string) Finding {
	return Finding{Check: check, Status: CheckFail, Message: msg, Fix: fix}
}

// HasProblems reports whether any finding is a warning or failure — the
// grep-style exit code: 0 when nothing needs doing, so a healthy server
// with TIME_WAIT sockets still passes `ports doctor 3000 && ./deploy.sh`.
func HasProblems(findings []Finding) bool {
	for _, f := range findings {
		if f.Status == CheckWarn || f.Status == CheckFail {
			return true
		}
	}
	return false
}

// rowFacts is the live-process material the per-row checks read, gathered
// once in Diagnose so the checks themselves stay pure over rows and
// facts — and so the tests can drive them with synthetic inputs.
type rowFacts struct {
	states []string // process state letters (R, S, D, Z, …), nil when unknown
	fresh  bool     // CreateTime still matches the scan's row
	exists bool     // still running when doctor looked
}

func factsFor(s Service) rowFacts {
	f := rowFacts{fresh: true}
	if !s.Known() {
		return f
	}
	p, err := gproc.NewProcess(s.Pid)
	if err != nil {
		return f // gone — exists stays false, fresh is moot
	}
	f.exists = true
	if st, err := p.Status(); err == nil {
		for _, w := range st {
			f.states = append(f.states, stateLetter(w))
		}
	}
	f.fresh = sameProcess(p, s.Started)
	return f
}

// stateLetter maps gopsutil's ps-style status words back to the kernel
// letters stateFindings reads — the words are platform-flavored, the
// letters are the kernel's own and stable across the Unixes. Anything
// unrecognized passes through and is reported as-is.
func stateLetter(w string) string {
	switch w {
	case gproc.Running:
		return "R"
	case gproc.Sleep:
		return "S"
	case gproc.Blocked, "disk sleep":
		return "D" // uninterruptible sleep
	case gproc.Zombie:
		return "Z"
	case gproc.Stop:
		return "T"
	default:
		return w
	}
}

// Diagnose answers "why is this port like this?" — without touching it.
// It re-scans, because the row the user is looking at may already be
// stale, then runs the check sequence for the port. The rows come back
// too: they are the header of the report.
func Diagnose(cfg *Config, port uint32) ([]Service, []Finding, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	services, err := Scan(cfg)
	if err != nil {
		return nil, nil, err
	}
	var rows []Service
	for _, s := range services {
		if s.Port == port {
			rows = append(rows, s)
		}
	}
	facts := make([]rowFacts, len(rows))
	for i, s := range rows {
		facts[i] = factsFor(s)
	}
	dialOpen := len(rows) == 0 && portAnswers(port)
	winMirror := onWSL() && mirrorHeld(rows, port)
	servesHTTP := len(rows) > 0 && httpAnswers(port)

	probe := RuntimeProbe{Runtime: containerRuntime, Socket: dockerSocket, Status: RuntimeOK}
	for _, s := range rows {
		if s.ContainerID != "" || s.Process == "docker-proxy" {
			probe = probeRuntime()
			break
		}
	}
	return rows, diagnosePort(port, rows, facts, connectionsOnPort(port), dialOpen, winMirror, servesHTTP, probe), nil
}

// DiagnoseMachine is bare `ports doctor`: the machine-wide subset — the
// engine's reachability, every proxy row that didn't resolve, and every
// port that stop can't see or shouldn't share.
func DiagnoseMachine(cfg *Config) ([]Finding, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	services, err := Scan(cfg)
	if err != nil {
		return nil, err
	}
	return diagnoseMachine(services, probeRuntime(), onWSL()), nil
}

// diagnosePort is the check sequence for one port, pure over its inputs.
// rows and facts are parallel slices; conns is every socket on the port,
// in every state — the LISTEN-only filter is Scan's, and doctor needs
// the whole picture to explain a port.
func diagnosePort(port uint32, rows []Service, facts []rowFacts, conns []gnet.ConnectionStat, dialOpen, winMirror, servesHTTP bool, probe RuntimeProbe) []Finding {
	var f []Finding

	// — the listener itself ————————————————————————————————
	if len(rows) == 0 {
		f = append(f, fail("listener", fmt.Sprintf("nothing is listening on :%d", port), ""))
		if dialOpen {
			f = append(f, warn("listener", fmt.Sprintf("a connection to :%d succeeds anyway", port),
				"something outside the scan's view holds it — another user's process, or a listener in another namespace; try `sudo ports`"))
		}
	} else {
		f = append(f, ok("listener", fmt.Sprintf("listening on :%d", port)))
		if servesHTTP {
			// the one honest home for a clickable URL: doctor doesn't
			// hand one over until it verified the port speaks HTTP
			f = append(f, ok("http", fmt.Sprintf("responds to HTTP → http://localhost:%d", port)))
		}
	}

	// — who owns it, how many of them ————————————————————————
	pids := map[int32]bool{}
	for _, s := range rows {
		pids[s.Pid] = true
	}
	if len(pids) > 1 {
		f = append(f, warn("multiplicity", fmt.Sprintf("%d processes listen on :%d (SO_REUSEPORT)", len(pids), port),
			fmt.Sprintf("`ports stop %d` stops all of them — `--dry-run` previews the full set", port)))
	}

	// — the primary row, checked live ————————————————————————
	if len(rows) > 0 {
		s := rows[0]
		ft := rowFacts{fresh: true}
		if len(facts) > 0 {
			ft = facts[0]
		}
		switch {
		case s.Windows:
			f = append(f, fail("windows",
				fmt.Sprintf("the listener is on the Windows side (%s, pid %d) — signals from here cannot reach it", s.Process, s.WindowsPid),
				fmt.Sprintf("stop it from Windows: taskkill /PID %d /F", s.WindowsPid)))
		case !s.Known():
			f = append(f, ownerFinding(s))
		case s.Process == "docker-proxy" && s.ContainerID == "":
			f = append(f, fail("container",
				fmt.Sprintf("docker-proxy holds :%d but no matching container appears in the engine", port),
				"check `docker ps` — the port mapping may be stale"))
		default:
			f = append(f, stateFindings(s, ft)...)
			if !ft.fresh {
				f = append(f, fail("pid-freshness",
					fmt.Sprintf("pid %d now belongs to another process — the scan row is stale", s.Pid),
					"run `ports` again to refresh"))
			} else if ft.exists {
				f = append(f, ok("pid-freshness", fmt.Sprintf("pid %d still matches the scan — no pid reuse", s.Pid)))
			}
			if ft.exists && s.Cwd == "" {
				f = append(f, info("cwd",
					"the process exposes no working directory — likely a system daemon or kernel thread, so no project context"))
			}
		}
	}

	// — everything else holding the port —————————————————————
	byStatus := map[string]int{}
	for _, c := range conns {
		if c.Status != "LISTEN" {
			byStatus[c.Status]++
		}
	}
	if n := byStatus["TIME_WAIT"]; n > 0 {
		f = append(f, info("sockets",
			fmt.Sprintf("%d sockets in TIME_WAIT on :%d — they age out in ~60s, and rebinds usually succeed anyway", n, port)))
	}
	if n := byStatus["CLOSE_WAIT"]; n > 0 {
		f = append(f, info("sockets",
			fmt.Sprintf("%d sockets in CLOSE_WAIT on :%d — clients hung up and the process hasn't closed its end: a leak hint in the app, not a ports problem", n, port)))
	}
	if n := byStatus["ESTABLISHED"]; n > 0 {
		f = append(f, ok("sockets", fmt.Sprintf("%d live client connections", n)))
	}
	if len(byStatus) == 0 && len(rows) > 0 {
		f = append(f, ok("sockets", fmt.Sprintf("nothing else holds :%d — no stray sockets", port)))
	}

	// — the boundary and the engine —————————————————————————
	if winMirror {
		f = append(f, warn("mirror",
			fmt.Sprintf(":%d also has a listener on the Windows side — mirrored networking forwards between them", port),
			"check both sides before assuming a kill freed the port"))
	}
	if probe.Status != RuntimeOK && probeNeeded(rows) {
		f = append(f, runtimeFinding(probe))
	}
	return f
}

// stateFindings reads the process state letters. The letters are the
// kernel's, stable across the Unixes ports runs on.
func stateFindings(s Service, ft rowFacts) []Finding {
	if !ft.exists {
		return []Finding{warn("state",
			fmt.Sprintf("pid %d is no longer running — the scan row is stale", s.Pid),
			"run `ports` again")}
	}
	if ft.states == nil {
		return nil // state unreadable (permissions, race) — say nothing rather than guess
	}
	st := ft.states[0]
	switch st {
	case "D":
		return []Finding{fail("state",
			fmt.Sprintf("pid %d is in uninterruptible sleep (state D) — signals, SIGKILL included, are held until its I/O returns", s.Pid),
			"wait out the I/O, or reboot — this process cannot be killed")}
	case "Z":
		parent := s.ParentPid
		fix := "the row disappears when its parent reaps it or exits"
		if parent > 0 {
			fix = fmt.Sprintf("the row disappears when the parent (pid %d) reaps it or exits", parent)
		}
		return []Finding{warn("state",
			fmt.Sprintf("pid %d is a zombie — it has already exited; :%d is effectively free", s.Pid, s.Port),
			fix)}
	case "T":
		return []Finding{warn("state",
			fmt.Sprintf("pid %d is suspended (state T) — SIGTERM is held until it resumes; SIGKILL still lands", s.Pid),
			fmt.Sprintf("`ports stop %d --force` kills a suspended process outright", s.Port))}
	default:
		return []Finding{ok("state", fmt.Sprintf("process state %s — signals land", st))}
	}
}

// probeNeeded reports whether any row depends on the container engine —
// the runtime is only worth reporting when a row's story needs it.
func probeNeeded(rows []Service) bool {
	for _, s := range rows {
		if s.ContainerID != "" || s.Process == "docker-proxy" {
			return true
		}
	}
	return false
}

// runtimeFinding says why the engine can't be reached — the scan stays
// silent about this by design; doctor is where it surfaces.
func runtimeFinding(p RuntimeProbe) Finding {
	// an engine ports doesn't know has no well-known path to name
	if p.Socket == "" {
		return warn("runtime",
			fmt.Sprintf("no %s socket to talk to — ports speaks the Docker API over docker and podman sockets", p.Runtime),
			"point CONTAINER_RUNTIME at docker or podman, or unset it")
	}
	switch p.Status {
	case RuntimePermission:
		return warn("runtime",
			fmt.Sprintf("permission denied on the %s socket (%s)", p.Runtime, p.Socket),
			"add yourself to the socket's group (usually `docker`)")
	case RuntimeTimeout:
		return warn("runtime",
			fmt.Sprintf("the %s daemon did not answer within 2s (%s)", p.Runtime, p.Socket),
			"the daemon may be hung — restart it")
	case RuntimeUnreachable:
		return warn("runtime",
			fmt.Sprintf("no answer on the %s socket (%s)", p.Runtime, p.Socket),
			"is the engine running?")
	default:
		return warn("runtime",
			fmt.Sprintf("no %s socket found at %s", p.Runtime, p.Socket),
			"start the engine to resolve container-backed ports")
	}
}

// diagnoseMachine is the machine-wide subset, pure over its inputs.
func diagnoseMachine(services []Service, probe RuntimeProbe, wsl bool) []Finding {
	var f []Finding
	switch probe.Status {
	case RuntimeOK:
		f = append(f, ok("runtime", fmt.Sprintf("%s engine reachable via %s", probe.Runtime, probe.Socket)))
	case RuntimeNoSocket:
		// no engine installed is not a problem — plenty of machines
		// are fine without one; nothing to say
	default:
		f = append(f, runtimeFinding(probe))
	}

	var proxy, unowned, winrows int
	shared := map[uint32]int{}
	for _, s := range services {
		if s.Process == "docker-proxy" && s.ContainerID == "" {
			proxy++
		}
		if s.Known() {
			shared[s.Port]++
		}
		if s.Anonymous() {
			unowned++
		}
		if s.Windows {
			winrows++
		}
	}
	var ports []uint32
	for p, n := range shared {
		if n > 1 {
			ports = append(ports, p)
		}
	}
	slices.Sort(ports)
	for _, p := range ports {
		f = append(f, warn("multiplicity",
			fmt.Sprintf(":%d is shared by %d processes (SO_REUSEPORT)", p, shared[p]),
			fmt.Sprintf("`ports stop %d` stops all of them — `--dry-run` previews", p)))
	}
	if proxy > 0 {
		f = append(f, fail("container",
			fmt.Sprintf("%d docker-proxy port(s) did not resolve to a container", proxy),
			"check `docker ps` — the mappings may be stale"))
	}
	if unowned > 0 {
		f = append(f, warn("owner",
			fmt.Sprintf("%d listener(s) belong to other users' processes — shown as ?", unowned),
			"run `sudo ports` to see and stop them"))
	}
	if wsl && winrows > 0 {
		f = append(f, warn("windows",
			fmt.Sprintf("%d listener(s) sit on the Windows side", winrows),
			"stop those from Windows (Task Manager or taskkill)"))
	}
	if !HasProblems(f) {
		f = append(f, ok("machine", "nothing suspicious — every listener is attributable and unshared"))
	}
	return f
}

// connectionsOnPort returns every socket whose local port matches, in
// every state. The LISTEN-only filter is Scan's; doctor needs the whole
// picture to explain a port.
func connectionsOnPort(port uint32) []gnet.ConnectionStat {
	conns, err := gnet.Connections("tcp")
	if err != nil {
		return nil
	}
	var out []gnet.ConnectionStat
	for _, c := range conns {
		if c.Laddr.Port == port {
			out = append(out, c)
		}
	}
	return out
}

// portAnswers reports whether a TCP connection to the port succeeds —
// the "listening but invisible to the scan" case.
func portAnswers(port uint32) bool { return dialable(port) }

// httpAnswers reports whether the listener speaks HTTP; doctor stays
// silent about a URL it hasn't verified. The finding says localhost, the
// form people type.
func httpAnswers(port uint32) bool { return ServesHTTP(port) }

// windowsHolds reports whether the Windows side also listens on the port
// (WSL2 mirrored mode can show the same port from both sides).
func windowsHolds(port uint32) bool {
	for _, w := range windowsListeners() {
		if w.Port == port {
			return true
		}
	}
	return false
}

// mirrorHeld reports whether a mirror is real: the port must be held on
// *both* sides. A Windows row alone is not a mirror — the Linux twin is
// what makes it one, so a scan that only sees the Windows side says
// nothing.
func mirrorHeld(rows []Service, port uint32) bool {
	local := slices.ContainsFunc(rows, func(s Service) bool { return !s.Windows })
	return local && windowsHolds(port)
}

// ownerFinding explains a listener another user owns: what it was
// traced to, if anything, and the command that stops it — systemctl for
// a service, since a signal from ports would only race its restart
// policy.
func ownerFinding(s Service) Finding {
	who := "another user's process"
	if s.User != "" {
		who = "a process of user " + s.User
	}
	switch {
	case s.Unit != "":
		return warn("owner",
			fmt.Sprintf(":%d is %s, run by systemd unit %s.service as %s — its pid is hidden from this scan", s.Port, ProcessName(s), s.Unit, s.User),
			fmt.Sprintf("stop it with `sudo systemctl stop %s`", s.Unit))
	case !s.Anonymous():
		return warn("owner",
			fmt.Sprintf(":%d is %s, running as %s — its pid is hidden from this scan", s.Port, ProcessName(s), s.User),
			fmt.Sprintf("run `sudo ports stop %d`", s.Port))
	default:
		return warn("owner",
			fmt.Sprintf("the listener on :%d belongs to %s — its pid is hidden from this scan", s.Port, who),
			fmt.Sprintf("run `sudo ports stop %d`", s.Port))
	}
}

// PortHeader is the one-line idiom of the table, for doctor's header:
// the port, the process as a developer knows it, who owns it, where.
func PortHeader(s Service) string {
	switch {
	case s.Container != "":
		return fmt.Sprintf(":%d  %s  container %s  up %s", s.Port, ProcessName(s), DockerName(s), uptime(s))
	case s.Windows:
		return fmt.Sprintf(":%d  %s  windows pid %d  up %s", s.Port, ProcessName(s), s.WindowsPid, uptime(s))
	case s.Anonymous():
		return fmt.Sprintf(":%d  ?  another user's process  %s  up %s", s.Port, project(s), uptime(s))
	case !s.Known():
		return fmt.Sprintf(":%d  %s  %s  up %s", s.Port, ProcessName(s), project(s), uptime(s))
	default:
		return fmt.Sprintf(":%d  %s  pid %d  %s  up %s", s.Port, ProcessName(s), s.Pid, project(s), uptime(s))
	}
}

// WriteDiagnosis renders one port's diagnosis: the row header when the
// port is listening, then one line per finding.
func WriteDiagnosis(rows []Service, findings []Finding) {
	if len(rows) > 0 {
		fmt.Println(PortHeader(rows[0]))
	}
	writeFindings(findings)
}

// WriteMachineFindings renders the bare machine-wide diagnosis.
func WriteMachineFindings(findings []Finding) {
	writeFindings(findings)
}

func writeFindings(findings []Finding) {
	for _, f := range findings {
		switch f.Status {
		case CheckInfo:
			fmt.Println(Mark("· "+f.Message, false))
		case CheckWarn:
			fmt.Println(Mark("⚠ "+f.Message, false))
		case CheckFail:
			fmt.Println(Mark("✗ "+f.Message, false))
		default:
			fmt.Println(Mark("✓ "+f.Message, false))
		}
		if f.Fix != "" {
			fmt.Printf("  → %s\n", f.Fix)
		}
	}
}

// WriteFindingsJSON renders findings for --format json.
func WriteFindingsJSON(findings []Finding) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(findings)
}
