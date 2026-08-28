package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/pyjeebz/ports/internal/inspect"
)

func match(services []inspect.Service, keep func(inspect.Service) bool) []inspect.Service {
	var out []inspect.Service
	for _, s := range services {
		if keep(s) {
			out = append(out, s)
		}
	}
	return out
}

// stopAll stops one process per pid — or one container per id, however
// many ports it publishes — and prints a line per stop. Returns the exit
// code. If dryRun is true, only shows what would be stopped without doing it.
func stopAll(matched []inspect.Service, force bool, dryRun bool, okMsg func(inspect.Service, string) string) int {
	type group struct {
		s     inspect.Service
		ports []string
	}
	var groups []*group
	byKey := map[string]*group{}
	for _, s := range matched {
		key := ""
		switch {
		case s.ContainerID != "":
			key = "c:" + s.ContainerID
		case s.Known():
			key = fmt.Sprintf("p:%d", s.Pid)
		}
		if g, ok := byKey[key]; ok && key != "" {
			g.ports = append(g.ports, fmt.Sprintf(":%d", s.Port))
			continue
		}
		g := &group{s: s, ports: []string{fmt.Sprintf(":%d", s.Port)}}
		groups = append(groups, g)
		if key != "" {
			byKey[key] = g
		}
	}
	code := 0
	for _, g := range groups {
		ports := strings.Join(g.ports, ", ")
		if dryRun {
			fmt.Printf("would stop %s (%s)\n", process(g.s), ports)
			continue
		}
		if err := inspect.Stop(g.s, force); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s (%s): %v\n", process(g.s), ports, err)
			code = 1
			continue
		}
		fmt.Println(okMsg(g.s, ports))
	}
	return code
}

func process(s inspect.Service) string {
	switch {
	case s.Framework != "":
		return s.Framework
	case s.Image != "":
		return inspect.ImageBase(s.Image)
	default:
		return s.Process
	}
}

func dockerName(s inspect.Service) string {
	if s.ComposeProject != "" && s.ComposeService != "" {
		return s.ComposeProject + "-" + s.ComposeService
	}
	return s.Container
}