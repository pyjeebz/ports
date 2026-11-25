package cmd

import (
	"bufio"
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// mcpExchange feeds lines to the server and returns its replies by id.
func mcpExchange(t *testing.T, lines ...string) map[string]map[string]any {
	t.Helper()
	var out strings.Builder
	if err := serveMCP(strings.NewReader(strings.Join(lines, "\n")+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	replies := map[string]map[string]any{}
	sc := bufio.NewScanner(strings.NewReader(out.String()))
	for sc.Scan() {
		var r map[string]any
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatalf("reply isn't JSON: %q", sc.Text())
		}
		id, _ := json.Marshal(r["id"])
		replies[string(id)] = r
	}
	return replies
}

func TestMCPHandshake(t *testing.T) {
	r := mcpExchange(t,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":"p","method":"ping"}`,
		`{"jsonrpc":"2.0","id":3,"method":"resources/list"}`,
		`not json`,
	)
	init, _ := r["1"]["result"].(map[string]any)
	if init["protocolVersion"] != "2025-03-26" {
		t.Errorf("an older protocol the client asked for = %v, want it echoed", init["protocolVersion"])
	}
	if _, ok := init["instructions"].(string); !ok {
		t.Error("no instructions for the agent")
	}
	tools, _ := r["2"]["result"].(map[string]any)["tools"].([]any)
	var names []string
	for _, tool := range tools {
		names = append(names, tool.(map[string]any)["name"].(string))
	}
	for _, want := range []string{"list", "here", "find", "url", "wait", "stop", "doctor"} {
		if !slices.Contains(names, want) {
			t.Errorf("tools/list lacks %s: %v", want, names)
		}
	}
	if slices.Contains(names, "run") {
		t.Error("the MCP server starts nothing, so it has no run tool")
	}
	if _, ok := r[`"p"`]["result"]; !ok {
		t.Errorf("ping: %v", r[`"p"`])
	}
	if e, _ := r["3"]["error"].(map[string]any); e["code"] != float64(-32601) {
		t.Errorf("unknown method: %v", r["3"])
	}
	if e, _ := r["null"]["error"].(map[string]any); e["code"] != float64(-32700) {
		t.Errorf("a line that isn't JSON: %v", r["null"])
	}
	if len(r) != 5 {
		t.Errorf("%d replies, want 5 — a notification gets none", len(r))
	}
}

func TestMCPStopNeedsExpect(t *testing.T) {
	r := mcpExchange(t, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"stop","arguments":{"target":"3000"}}}`)
	res, _ := r["1"]["result"].(map[string]any)
	if res["isError"] != true {
		t.Fatalf("stop without expect: %v", r["1"])
	}
	text := res["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "nothing was stopped") {
		t.Errorf("message %q should say nothing was stopped", text)
	}
}

func TestToolArgs(t *testing.T) {
	cases := []struct {
		tool string
		args map[string]any
		want string
	}{
		{"list", map[string]any{"framework": "vite"}, "--json --framework vite"},
		{"here", map[string]any{"dir": "/p/shop"}, "here /p/shop --json"},
		{"url", map[string]any{"name": "api.shop", "local": true}, "url --json api.shop --local"},
		{"wait", map[string]any{"target": "shop", "http": true, "timeout_seconds": 5.0}, "wait shop --json --timeout 5s --http"},
		{"wait", map[string]any{"target": "3000", "timeout_seconds": 9999.0}, "wait 3000 --json --timeout 30s"},
		{"stop", map[string]any{"target": "shop", "expect": "nextjs"}, "stop shop --expect nextjs --json"},
		{"doctor", map[string]any{"port": 3000.0}, "doctor --json 3000"},
	}
	for _, c := range cases {
		argv, _, _, err := toolArgs(c.tool, c.args)
		if err != nil || strings.Join(argv, " ") != c.want {
			t.Errorf("%s %v = %q, %v; want %q", c.tool, c.args, strings.Join(argv, " "), err, c.want)
		}
	}
	if _, _, _, err := toolArgs("run", nil); err == nil {
		t.Error("an unknown tool was accepted")
	}
}
