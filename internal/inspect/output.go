package inspect

import (
	"cmp"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"text/tabwriter"
)

// WriteJSON writes services as JSON to stdout.
func WriteJSON(services []Service) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(services)
}

// WriteCSV writes services as CSV to stdout.
func WriteCSV(services []Service) error {
	return writeDelimited(services, ',')
}

// WriteTSV writes services as TSV to stdout.
func WriteTSV(services []Service) error {
	return writeDelimited(services, '\t')
}

func writeDelimited(services []Service, delim rune) error {
	if len(services) == 0 {
		return nil
	}
	rows := slices.Clone(services)
	slices.SortStableFunc(rows, func(a, b Service) int {
		if (a.ProjectName == "") != (b.ProjectName == "") {
			if a.ProjectName != "" {
				return -1
			}
			return 1
		}
		if c := strings.Compare(a.ProjectName, b.ProjectName); c != 0 {
			return c
		}
		if c := strings.Compare(a.Project, b.Project); c != 0 {
			return c
		}
		return cmp.Compare(a.Port, b.Port)
	})

	w := csv.NewWriter(os.Stdout)
	w.Comma = delim
	defer w.Flush()

	// Header
	if err := w.Write([]string{"PORT", "PROCESS", "PROJECT", "UPTIME"}); err != nil {
		return err
	}
	for _, s := range rows {
		if err := w.Write([]string{
			fmt.Sprintf("%d", s.Port),
			ProcessName(s),
			project(s),
			uptime(s),
		}); err != nil {
			return err
		}
	}
	return nil
}

// WriteTable writes services as a formatted table to stdout.
func WriteTable(services []Service) {
	if len(services) == 0 {
		fmt.Println("no listening TCP ports")
		return
	}
	rows := slices.Clone(services)
	slices.SortStableFunc(rows, func(a, b Service) int {
		if (a.ProjectName == "") != (b.ProjectName == "") {
			if a.ProjectName != "" {
				return -1
			}
			return 1
		}
		if c := strings.Compare(a.ProjectName, b.ProjectName); c != 0 {
			return c
		}
		if c := strings.Compare(a.Project, b.Project); c != 0 {
			return c
		}
		return cmp.Compare(a.Port, b.Port)
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "PORT\tPROCESS\tPROJECT\tUPTIME")
	unknown := false
	for _, s := range rows {
		if !s.Known() && s.Container == "" && !s.Windows {
			unknown = true
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", s.Port, ProcessName(s), project(s), uptime(s))
	}
	w.Flush()
	if unknown {
		fmt.Fprintln(os.Stderr, "\nsome ports belong to other users' processes (shown as ?) — run with sudo to see them")
	}
}

// ProcessName is the process as a developer knows it: the framework
// when detected, the docker image base for containers, else the exe.
func ProcessName(s Service) string {
	switch {
	case s.Framework != "":
		return s.Framework
	case s.Image != "":
		return ImageBase(s.Image)
	default:
		return s.Process
	}
}

func project(s Service) string {
	if s.Windows {
		return "windows"
	}
	if s.Container != "" {
		return "docker: " + DockerName(s)
	}
	if s.Project == "" {
		return "—"
	}
	if home, err := os.UserHomeDir(); err == nil {
		if rel, ok := strings.CutPrefix(s.Project, home); ok {
			return "~" + rel
		}
	}
	return s.Project
}

func uptime(s Service) string {
	if s.Started.IsZero() {
		return "?"
	}
	d := time.Since(s.Started)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// DockerName is the container as a developer knows it: the compose
// project-service pair when labeled, else the container name.
func DockerName(s Service) string {
	if s.ComposeProject != "" && s.ComposeService != "" {
		return s.ComposeProject + "-" + s.ComposeService
	}
	return s.Container
}
