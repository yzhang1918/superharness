package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yzhang1918/superharness/internal/plan"
	"github.com/yzhang1918/superharness/internal/status"
)

type App struct {
	Stdout io.Writer
	Stderr io.Writer
	Now    func() time.Time
	Getwd  func() (string, error)
}

func New(stdout, stderr io.Writer) *App {
	return &App{
		Stdout: stdout,
		Stderr: stderr,
		Now:    time.Now,
		Getwd:  os.Getwd,
	}
}

func (a *App) Run(args []string) int {
	if len(args) == 0 {
		a.printRootUsage()
		return 2
	}

	switch args[0] {
	case "plan":
		return a.runPlan(args[1:])
	case "status":
		return a.runStatus(args[1:])
	case "-h", "--help", "help":
		a.printRootUsage()
		return 0
	default:
		fmt.Fprintf(a.Stderr, "unknown command %q\n\n", args[0])
		a.printRootUsage()
		return 2
	}
}

func (a *App) runPlan(args []string) int {
	if len(args) == 0 {
		a.printPlanUsage()
		return 2
	}
	switch args[0] {
	case "template":
		return a.runPlanTemplate(args[1:])
	case "lint":
		return a.runPlanLint(args[1:])
	case "-h", "--help", "help":
		a.printPlanUsage()
		return 0
	default:
		fmt.Fprintf(a.Stderr, "unknown plan subcommand %q\n\n", args[0])
		a.printPlanUsage()
		return 2
	}
}

func (a *App) runPlanTemplate(args []string) int {
	fs := flag.NewFlagSet("harness plan template", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)

	var refs stringListFlag
	title := fs.String("title", "", "Seed the H1 title.")
	output := fs.String("output", "", "Write the rendered template to this file instead of stdout.")
	dateValue := fs.String("date", "", "Seed timestamps using this YYYY-MM-DD date at local midnight.")
	timestampValue := fs.String("timestamp", "", "Seed timestamps using this RFC3339 timestamp.")
	sourceType := fs.String("source-type", "direct_request", "Seed the frontmatter source_type field.")
	fs.Var(&refs, "source-ref", "Seed one source_refs entry. Repeat to add multiple refs.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness plan template [flags]")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Render the packaged plan template with seeded title, timestamp, and source metadata.")
		fmt.Fprintln(a.Stderr)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(a.Stderr, "harness plan template does not accept positional arguments")
		return 2
	}

	ts, err := a.resolveTimestamp(*timestampValue, *dateValue)
	if err != nil {
		fmt.Fprintln(a.Stderr, err.Error())
		return 2
	}

	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      *title,
		Timestamp:  ts,
		SourceType: *sourceType,
		SourceRefs: refs,
	})
	if err != nil {
		fmt.Fprintf(a.Stderr, "render template: %v\n", err)
		return 1
	}

	if *output == "" {
		_, _ = io.WriteString(a.Stdout, rendered)
		return 0
	}

	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintf(a.Stderr, "create parent directory: %v\n", err)
		return 1
	}
	if err := os.WriteFile(*output, []byte(rendered), 0o644); err != nil {
		fmt.Fprintf(a.Stderr, "write template: %v\n", err)
		return 1
	}
	fmt.Fprintf(a.Stdout, "Wrote plan template to %s\n", *output)
	return 0
}

func (a *App) runPlanLint(args []string) int {
	fs := flag.NewFlagSet("harness plan lint", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness plan lint <plan-path>")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Validate a tracked plan and emit compact machine-readable results.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}

	result := plan.LintFile(fs.Arg(0))
	encoder := json.NewEncoder(a.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(a.Stderr, "encode lint result: %v\n", err)
		return 1
	}
	if result.OK {
		return 0
	}
	return 1
}

func (a *App) runStatus(args []string) int {
	fs := flag.NewFlagSet("harness status", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness status")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Summarize the current plan plus local execution state for the current worktree.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return 2
	}

	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}

	result := status.Service{Workdir: workdir}.Read()
	encoder := json.NewEncoder(a.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(a.Stderr, "encode status result: %v\n", err)
		return 1
	}
	if result.OK {
		return 0
	}
	return 1
}

func (a *App) resolveTimestamp(timestampValue, dateValue string) (time.Time, error) {
	if strings.TrimSpace(timestampValue) != "" {
		ts, err := time.Parse(time.RFC3339, timestampValue)
		if err != nil {
			return time.Time{}, fmt.Errorf("--timestamp must be RFC3339: %w", err)
		}
		return ts, nil
	}
	if strings.TrimSpace(dateValue) != "" {
		day, err := time.ParseInLocation("2006-01-02", dateValue, time.Local)
		if err != nil {
			return time.Time{}, fmt.Errorf("--date must be YYYY-MM-DD: %w", err)
		}
		return day, nil
	}
	return a.Now(), nil
}

func (a *App) printRootUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness <command> [subcommand] [flags]")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Commands:")
	fmt.Fprintln(a.Stderr, "  plan template   Render the packaged plan template")
	fmt.Fprintln(a.Stderr, "  plan lint       Validate a tracked plan")
	fmt.Fprintln(a.Stderr, "  status          Summarize the current plan and local execution state")
}

func (a *App) printPlanUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness plan <subcommand> [flags]")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Subcommands:")
	fmt.Fprintln(a.Stderr, "  template   Render the packaged plan template")
	fmt.Fprintln(a.Stderr, "  lint       Validate a tracked plan")
}

type stringListFlag []string

func (f *stringListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}
