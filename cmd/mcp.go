package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Serve ports to coding agents over MCP (stdio)",
	Long: `ports mcp is a Model Context Protocol server on stdin and stdout, for
coding agents: the tools are ports's own commands with their JSON output
and exit codes, so an agent sees exactly what a script would.

  list     what's listening, with names and URLs
  here     what's running for a project directory — ask before starting one
  find     filter by framework, process or project
  url      a dev server's URL by name
  wait     block until a port or name is up (or, with gone, free)
  stop     stop by port or name — expect is required, and must match all
  doctor   explain a port, or the machine

It starts nothing: an agent starts a dev server in a shell, with
ports run. Add it to an agent's MCP config as the command ports with
the argument mcp.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return serveMCP(os.Stdin, os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

// mcpProtocol is the protocol version this server speaks; a client that
// asks for another is answered with this one, as the spec has it.
const mcpProtocol = "2025-06-18"

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// serveMCP answers newline-delimited JSON-RPC until stdin closes. Tool
// calls run concurrently — a wait can block for a while — and replies
// are written whole, one per line.
func serveMCP(in io.Reader, out io.Writer) error {
	var mu sync.Mutex
	write := func(r rpcResponse) {
		r.JSONRPC = "2.0"
		data, _ := json.Marshal(r)
		mu.Lock()
		defer mu.Unlock()
		_, _ = out.Write(append(data, '\n'))
	}
	var wg sync.WaitGroup
	defer wg.Wait()
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			write(rpcResponse{ID: json.RawMessage("null"), Error: &rpcError{-32700, "parse error"}})
			continue
		}
		if req.ID == nil {
			continue // a notification: nothing to answer
		}
		switch req.Method {
		case "initialize":
			var p struct {
				ProtocolVersion string `json:"protocolVersion"`
			}
			_ = json.Unmarshal(req.Params, &p)
			version := mcpProtocol
			if p.ProtocolVersion != "" && p.ProtocolVersion <= mcpProtocol {
				version = p.ProtocolVersion
			}
			write(rpcResponse{ID: req.ID, Result: map[string]any{
				"protocolVersion": version,
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "ports", "version": rootCmd.Version},
				"instructions": "ports sees every dev server on this machine by project and name. " +
					"Before starting a dev server, call here for the project directory and reuse what's running. " +
					"Start servers in a shell with `ports run -- <command>` — it picks a free port and names the server. " +
					"Reach them by url, not by guessing ports. stop needs expect, and stops nothing unless every target matches it.",
			}})
		case "ping":
			write(rpcResponse{ID: req.ID, Result: map[string]any{}})
		case "tools/list":
			write(rpcResponse{ID: req.ID, Result: map[string]any{"tools": mcpTools}})
		case "tools/call":
			wg.Add(1)
			go func(req rpcRequest) {
				defer wg.Done()
				result, rerr := callTool(req.Params)
				if rerr != nil {
					write(rpcResponse{ID: req.ID, Error: rerr})
					return
				}
				write(rpcResponse{ID: req.ID, Result: result})
			}(req)
		default:
			write(rpcResponse{ID: req.ID, Error: &rpcError{-32601, "method not found: " + req.Method}})
		}
	}
	return sc.Err()
}

func schema(props map[string]any, required ...string) map[string]any {
	s := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
func boolean(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}

var mcpTools = []map[string]any{
	{
		"name":        "list",
		"description": "Every listening port on this machine: framework, project, uptime, and for web dev servers their name and URL. Read-only.",
		"inputSchema": schema(map[string]any{
			"project":   str("Keep rows of these projects (name or path, comma-separated)"),
			"framework": str("Keep rows of these frameworks (nextjs, vite, …, comma-separated)"),
		}),
		"annotations": map[string]any{"readOnlyHint": true},
	},
	{
		"name":        "here",
		"description": "What's running for a project directory. Call this before starting a dev server: reuse what's up instead of starting a second copy on another port. Returns [] when nothing is running.",
		"inputSchema": schema(map[string]any{"dir": str("The project directory (absolute path); any subdirectory works")}, "dir"),
		"annotations": map[string]any{"readOnlyHint": true},
	},
	{
		"name":        "find",
		"description": "Listening ports whose framework, process, project or name contains the query. Read-only.",
		"inputSchema": schema(map[string]any{"query": str("What to look for: postgres, vite, shop, …")}, "query"),
		"annotations": map[string]any{"readOnlyHint": true},
	},
	{
		"name":        "url",
		"description": "A dev server's URL by name (shop, api.shop), or the app in dir. Use local for an address every HTTP client resolves (http://localhost:PORT) — some systems' resolvers don't find *.localhost names.",
		"inputSchema": schema(map[string]any{
			"name":  str("The name: shop, api.shop, …"),
			"dir":   str("With no name: the project directory whose app to give"),
			"local": boolean("Give http://localhost:PORT instead of the name's URL"),
		}),
		"annotations": map[string]any{"readOnlyHint": true},
	},
	{
		"name":        "wait",
		"description": "Block until a port or name is listening (http: until it answers HTTP), or with gone until it's free. Returns the rows once ready.",
		"inputSchema": schema(map[string]any{
			"target":          str("A port (3000) or a name (shop, api.shop, nextjs)"),
			"http":            boolean("Wait until it answers HTTP, not just accepts connections"),
			"gone":            boolean("Wait until nothing holds it instead"),
			"timeout_seconds": map[string]any{"type": "integer", "minimum": 1, "maximum": 300, "description": "Give up after this long (default 30)"},
		}, "target"),
		"annotations": map[string]any{"readOnlyHint": true},
	},
	{
		"name":        "stop",
		"description": "Stop what holds a port or name (SIGTERM, then SIGKILL). expect is required: everything the stop would reach must be that project, framework, container or process, or nothing is stopped. Never stops other users' processes.",
		"inputSchema": schema(map[string]any{
			"target": str("A port (3000) or a name (shop, api.shop)"),
			"expect": str("What it must be — the project, framework (nextjs), container or process — or nothing is stopped"),
		}, "target", "expect"),
		"annotations": map[string]any{"destructiveHint": true, "idempotentHint": true},
	},
	{
		"name":        "doctor",
		"description": "Explain why a port won't free or looks odd, or check the whole machine (no port): findings and the command that fixes each. Read-only.",
		"inputSchema": schema(map[string]any{"port": map[string]any{"type": "integer", "minimum": 1, "maximum": 65535}}),
		"annotations": map[string]any{"readOnlyHint": true},
	},
}

// callTool runs one tool as the ports command it stands for and returns
// what that command printed: its JSON as the result, its stderr as the
// message, exit 1 (nothing found, timed out, refused) as a result an
// agent reads rather than a failure, exit 2 as an error.
func callTool(raw json.RawMessage) (any, *rpcError) {
	var p struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpcError{-32602, "bad params"}
	}
	argv, dir, timeout, err := toolArgs(p.Name, p.Arguments)
	if err != nil {
		var unknown unknownTool
		if errors.As(err, &unknown) {
			return nil, &rpcError{-32602, err.Error()}
		}
		return toolResult("", err.Error(), 2), nil
	}
	stdout, stderr, code := runSelf(argv, dir, timeout)
	return toolResult(stdout, stderr, code), nil
}

type unknownTool string

func (u unknownTool) Error() string { return "unknown tool: " + string(u) }

func toolResult(stdout, stderr string, code int) map[string]any {
	var content []map[string]any
	if s := strings.TrimSpace(stdout); s != "" {
		content = append(content, map[string]any{"type": "text", "text": s})
	}
	if s := strings.TrimSpace(stderr); s != "" {
		content = append(content, map[string]any{"type": "text", "text": s})
	}
	if len(content) == 0 {
		content = append(content, map[string]any{"type": "text", "text": fmt.Sprintf("exit %d", code)})
	}
	out := map[string]any{"content": content, "isError": code != 0 && code != 1}
	var parsed any
	if json.Unmarshal([]byte(stdout), &parsed) == nil {
		out["structuredContent"] = map[string]any{"result": parsed, "exit_code": code, "message": strings.TrimSpace(stderr)}
	}
	return out
}

// toolArgs is the ports command line a tool call stands for, the
// directory to run it in, and how long it may take.
func toolArgs(name string, a map[string]any) (argv []string, dir string, timeout time.Duration, err error) {
	s := func(k string) string { v, _ := a[k].(string); return strings.TrimSpace(v) }
	b := func(k string) bool { v, _ := a[k].(bool); return v }
	timeout = 30 * time.Second
	switch name {
	case "list":
		argv = []string{"--json"}
		if v := s("project"); v != "" {
			argv = append(argv, "--project", v)
		}
		if v := s("framework"); v != "" {
			argv = append(argv, "--framework", v)
		}
	case "here":
		if s("dir") == "" {
			return nil, "", 0, errors.New("here needs dir")
		}
		argv = []string{"here", s("dir"), "--json"}
	case "find":
		if s("query") == "" {
			return nil, "", 0, errors.New("find needs query")
		}
		argv = []string{"find", s("query"), "--json"}
	case "url":
		argv = []string{"url", "--json"}
		if v := s("name"); v != "" {
			argv = append(argv, v)
		} else if s("dir") == "" {
			return nil, "", 0, errors.New("url needs name, or dir for the app in a project")
		}
		dir = s("dir")
		if b("local") {
			argv = append(argv, "--local")
		}
	case "wait":
		if s("target") == "" {
			return nil, "", 0, errors.New("wait needs target")
		}
		secs := 30
		if v, ok := a["timeout_seconds"].(float64); ok && v >= 1 && v <= 300 {
			secs = int(v)
		}
		argv = []string{"wait", s("target"), "--json", "--timeout", strconv.Itoa(secs) + "s"}
		if b("http") {
			argv = append(argv, "--http")
		}
		if b("gone") {
			argv = append(argv, "--gone")
		}
		timeout = time.Duration(secs+10) * time.Second
	case "stop":
		if s("target") == "" || s("expect") == "" {
			return nil, "", 0, errors.New("stop needs target and expect — nothing was stopped")
		}
		argv = []string{"stop", s("target"), "--expect", s("expect"), "--json"}
	case "doctor":
		argv = []string{"doctor", "--json"}
		if v, ok := a["port"].(float64); ok && v >= 1 && v <= 65535 {
			argv = append(argv, strconv.Itoa(int(v)))
		}
	default:
		return nil, "", 0, unknownTool(name)
	}
	return argv, dir, timeout, nil
}

// runSelf runs this ports binary with argv and no terminal, so it never
// opens a picker or colours its output.
func runSelf(argv []string, dir string, timeout time.Duration) (stdout, stderr string, code int) {
	exe, err := os.Executable()
	if err != nil {
		return "", err.Error(), 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c := exec.CommandContext(ctx, exe, argv...)
	c.Dir = dir
	c.Env = append(os.Environ(), "NO_COLOR=1")
	var out, errb bytes.Buffer
	c.Stdout, c.Stderr = &out, &errb
	err = c.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return out.String(), errb.String(), 0
	case errors.As(err, &exit):
		return out.String(), strings.TrimPrefix(errb.String(), "ports: "), exit.ExitCode()
	default:
		return out.String(), err.Error(), 2
	}
}
