package router

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	gproc "github.com/shirou/gopsutil/v4/process"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/pyjeebz/ports/internal/state"
)

// DefaultPort is where the router prefers to listen: on 80 a name needs
// no port in its URL — http://shop.localhost.
const DefaultPort = 80

// FallbackPort is where it listens when 80 is taken or not allowed. One
// fixed number, never a free one picked at random, so a URL written down
// once keeps working. 8480: ends in 80 so it reads as HTTP, sits on no
// browser's blocked-port list, clear of the ports dev servers reach for
// (3000s, 5173, 8000, 8080, 8888) and of ports run's 4000-4999, and of
// portless, which takes 443.
const FallbackPort = 8480

// Listen opens the router's sockets on both loopbacks, and nowhere else:
// nothing off this machine can reach a dev server through it. A machine
// with no IPv6 loopback gets IPv4 alone.
func Listen(port int) ([]net.Listener, error) {
	v4, err := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	v6, err := net.Listen("tcp6", net.JoinHostPort("::1", strconv.Itoa(port)))
	if err != nil {
		if errors.Is(err, os.ErrPermission) || isAddrInUse(err) {
			_ = v4.Close()
			return nil, err
		}
		return []net.Listener{v4}, nil // no ::1 here
	}
	return []net.Listener{v4, v6}, nil
}

// ListenDefault listens on DefaultPort, falling back to FallbackPort,
// and says why when it had to.
func ListenDefault() (ls []net.Listener, port int, why error, err error) {
	ls, err = Listen(DefaultPort)
	if err == nil {
		return ls, DefaultPort, nil, nil
	}
	why = err
	ls, err = Listen(FallbackPort)
	if err != nil {
		return nil, 0, why, fmt.Errorf("port %d: %w; port %d: %w", DefaultPort, why, FallbackPort, err)
	}
	return ls, FallbackPort, why, nil
}

func isAddrInUse(err error) bool {
	var op *net.OpError
	if errors.As(err, &op) {
		var sys *os.SyscallError
		if errors.As(op.Err, &sys) {
			return sys.Syscall == "bind" && !errors.Is(sys.Err, os.ErrPermission)
		}
	}
	return false
}

// Info is the running router's whereabouts, written to the state
// directory while it runs, so the table and ports url can show names as
// URLs that work.
type Info struct {
	Pid     int       `json:"pid"`
	Port    int       `json:"port"`
	Started time.Time `json:"started"`
}

const infoFile = "router.json"

// Announce records this process as the router on port.
func Announce(port int) error {
	return state.Write(infoFile, Info{Pid: os.Getpid(), Port: port, Started: time.Now()})
}

// Withdraw removes the record Announce wrote, if it's still this
// process's.
func Withdraw() {
	var info Info
	if state.Read(infoFile, &info) == nil && info.Pid == os.Getpid() {
		_ = state.Remove(infoFile)
	}
}

// Running is the router that runs now: the one recorded, if its process
// is alive and its port is still held. A record left by a router that
// died is ignored.
func Running() (Info, bool) {
	var info Info
	if err := state.Read(infoFile, &info); err != nil || info.Pid == 0 || info.Port == 0 {
		return Info{}, false
	}
	if alive, err := gproc.PidExists(int32(info.Pid)); err != nil || !alive {
		return Info{}, false
	}
	if !inspect.Listening(uint32(info.Port)) {
		return Info{}, false
	}
	return info, true
}
