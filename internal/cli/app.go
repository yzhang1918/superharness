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

	"github.com/catu-ai/easyharness/internal/evidence"
	"github.com/catu-ai/easyharness/internal/install"
	"github.com/catu-ai/easyharness/internal/lifecycle"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/review"
	"github.com/catu-ai/easyharness/internal/status"
	versioninfo "github.com/catu-ai/easyharness/internal/version"
)

type App struct {
	Stdout  io.Writer
	Stderr  io.Writer
	Stdin   io.Reader
	Now     func() time.Time
	Getwd   func() (string, error)
	Version func() versioninfo.Info
}

func New(stdout, stderr io.Writer) *App {
	return &App{
		Stdout:  stdout,
		Stderr:  stderr,
		Stdin:   os.Stdin,
		Now:     time.Now,
		Getwd:   os.Getwd,
		Version: versioninfo.Current,
	}
}

func (a *App) Run(args []string) int {
	if len(args) == 0 {
		a.printRootUsage()
		return 2
	}

	switch args[0] {
	case "--version":
		return a.runVersion(args[1:])
	case "plan":
		return a.runPlan(args[1:])
	case "execute":
		return a.runExecute(args[1:])
	case "evidence":
		return a.runEvidence(args[1:])
	case "review":
		return a.runReview(args[1:])
	case "land":
		return a.runLand(args[1:])
	case "archive":
		return a.runArchive(args[1:])
	case "reopen":
		return a.runReopen(args[1:])
	case "status":
		return a.runStatus(args[1:])
	case "install":
		return a.runInstall(args[1:])
	case "-h", "--help", "help":
		a.printRootUsage()
		return 0
	default:
		fmt.Fprintf(a.Stderr, "unknown command %q\n\n", args[0])
		a.printRootUsage()
		return 2
	}
}

func (a *App) runVersion(args []string) int {
	fs := flag.NewFlagSet("harness --version", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness --version")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Print concise debug information for the running harness binary.")
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
	if a.Version == nil {
		a.Version = versioninfo.Current
	}
	_, err := io.WriteString(a.Stdout, a.Version().String())
	if err != nil {
		fmt.Fprintf(a.Stderr, "write version output: %v\n", err)
		return 1
	}
	return 0
}

func (a *App) runReview(args []string) int {
	if len(args) == 0 {
		a.printReviewUsage()
		return 2
	}
	switch args[0] {
	case "start":
		return a.runReviewStart(args[1:])
	case "submit":
		return a.runReviewSubmit(args[1:])
	case "aggregate":
		return a.runReviewAggregate(args[1:])
	case "-h", "--help", "help":
		a.printReviewUsage()
		return 0
	default:
		fmt.Fprintf(a.Stderr, "unknown review subcommand %q\n\n", args[0])
		a.printReviewUsage()
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

func (a *App) runExecute(args []string) int {
	if len(args) == 0 {
		a.printExecuteUsage()
		return 2
	}
	switch args[0] {
	case "start":
		return a.runExecuteStart(args[1:])
	case "-h", "--help", "help":
		a.printExecuteUsage()
		return 0
	default:
		fmt.Fprintf(a.Stderr, "unknown execute subcommand %q\n\n", args[0])
		a.printExecuteUsage()
		return 2
	}
}

func (a *App) runEvidence(args []string) int {
	if len(args) == 0 {
		a.printEvidenceUsage()
		return 2
	}
	switch args[0] {
	case "submit":
		return a.runEvidenceSubmit(args[1:])
	case "-h", "--help", "help":
		a.printEvidenceUsage()
		return 0
	default:
		fmt.Fprintf(a.Stderr, "unknown evidence subcommand %q\n\n", args[0])
		a.printEvidenceUsage()
		return 2
	}
}

func (a *App) runLand(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "complete":
			return a.runLandComplete(args[1:])
		case "-h", "--help", "help":
			a.printLandUsage()
			return 0
		}
	}
	return a.runLandEntry(args)
}

func (a *App) runEvidenceSubmit(args []string) int {
	fs := flag.NewFlagSet("harness evidence submit", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	kind := fs.String("kind", "", "Evidence kind: ci, publish, or sync.")
	inputPath := fs.String("input", "", "Read the evidence payload JSON from this path. Defaults to stdin.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness evidence submit --kind <ci|publish|sync> [--input <path>]")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Record append-only CI, publish, or sync evidence for the current archived candidate.")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Schemas:")
		fmt.Fprintln(a.Stderr, `  ci:      {"status":"pending|success|failed|not_applied","provider":"optional","url":"optional","reason":"required when status=not_applied"}`)
		fmt.Fprintln(a.Stderr, `  publish: {"status":"recorded|not_applied","pr_url":"required when status=recorded","branch":"optional","base":"optional","commit":"optional","reason":"required when status=not_applied"}`)
		fmt.Fprintln(a.Stderr, `  sync:    {"status":"fresh|stale|conflicted|not_applied","base_ref":"optional","head_ref":"optional","reason":"required when status=not_applied"}`)
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Examples:")
		fmt.Fprintln(a.Stderr, `  harness evidence submit --kind ci <<'EOF'`)
		fmt.Fprintln(a.Stderr, `  {"status":"success","provider":"github-actions","url":"https://github.com/org/repo/actions/runs/123"}`)
		fmt.Fprintln(a.Stderr, `  EOF`)
		fmt.Fprintln(a.Stderr, `  harness evidence submit --kind sync <<'EOF'`)
		fmt.Fprintln(a.Stderr, `  {"status":"not_applied","reason":"repository has no shared merge target in this environment"}`)
		fmt.Fprintln(a.Stderr, `  EOF`)
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 || strings.TrimSpace(*kind) == "" {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	inputBytes, err := a.readInput(*inputPath)
	if err != nil {
		fmt.Fprintf(a.Stderr, "read evidence input: %v\n", err)
		return 1
	}
	result := evidence.Service{Workdir: workdir, Now: a.Now}.Submit(*kind, inputBytes)
	return a.writeJSONResult(result)
}

func (a *App) runLandEntry(args []string) int {
	fs := flag.NewFlagSet("harness land", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	prURL := fs.String("pr", "", "Merged PR URL.")
	commit := fs.String("commit", "", "Optional landed commit SHA.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness land --pr <url> [--commit <sha>]")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Record merge confirmation for the current archived candidate and enter land cleanup.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 || strings.TrimSpace(*prURL) == "" {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	result := lifecycle.Service{Workdir: workdir, Now: a.Now}.Land(*prURL, *commit)
	return a.writeJSONResult(result)
}

func (a *App) runInstall(args []string) int {
	fs := flag.NewFlagSet("harness install", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	scope := fs.String("scope", install.ScopeAll, "Install scope: agents, skills, or all.")
	dryRun := fs.Bool("dry-run", false, "Show the planned repository changes without writing files.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness install [--scope <agents|skills|all>] [--dry-run]")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Install or refresh the harness-managed repository bootstrap for the current repo.")
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
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	result := install.Service{Workdir: workdir}.Install(install.Options{
		Scope:  *scope,
		DryRun: *dryRun,
	})
	return a.writeJSONResult(result)
}

func (a *App) runPlanTemplate(args []string) int {
	fs := flag.NewFlagSet("harness plan template", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)

	var refs stringListFlag
	title := fs.String("title", "", "Seed the H1 title.")
	output := fs.String("output", "", "Write the rendered template to this file instead of stdout.")
	lightweight := fs.Bool("lightweight", false, "Render the lightweight variant and seed workflow_profile: lightweight.")
	dateValue := fs.String("date", "", "Seed timestamps using this YYYY-MM-DD date with the current local time-of-day.")
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
		WorkflowProfile: func() string {
			if *lightweight {
				return plan.WorkflowProfileLightweight
			}
			return ""
		}(),
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

func (a *App) runReviewStart(args []string) int {
	fs := flag.NewFlagSet("harness review start", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	specPath := fs.String("spec", "", "Read the review spec JSON from this path. Defaults to stdin.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness review start [--spec <path>]")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Create a deterministic review round from a minimal review spec.")
		fmt.Fprintln(a.Stderr, "The spec must include `kind` and `dimensions`, and may include optional `review_title` or `step`.")
		fmt.Fprintln(a.Stderr, "Harness infers whether the round is step-bound or finalize-bound from the current workflow state.")
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
	specBytes, err := a.readInput(*specPath)
	if err != nil {
		fmt.Fprintf(a.Stderr, "read review spec: %v\n", err)
		return 1
	}
	result := review.Service{Workdir: workdir, Now: a.Now}.Start(specBytes)
	return a.writeJSONResult(result)
}

func (a *App) runReviewSubmit(args []string) int {
	fs := flag.NewFlagSet("harness review submit", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	roundID := fs.String("round", "", "Review round ID.")
	slot := fs.String("slot", "", "Reviewer slot name.")
	inputPath := fs.String("input", "", "Read the reviewer submission JSON from this path. Defaults to stdin.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness review submit --round <round-id> --slot <slot> [--input <path>]")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Record one reviewer submission for the selected review round and slot.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 || strings.TrimSpace(*roundID) == "" || strings.TrimSpace(*slot) == "" {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	inputBytes, err := a.readInput(*inputPath)
	if err != nil {
		fmt.Fprintf(a.Stderr, "read reviewer submission: %v\n", err)
		return 1
	}
	result := review.Service{Workdir: workdir, Now: a.Now}.Submit(*roundID, *slot, inputBytes)
	return a.writeJSONResult(result)
}

func (a *App) runReviewAggregate(args []string) int {
	fs := flag.NewFlagSet("harness review aggregate", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	roundID := fs.String("round", "", "Review round ID.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness review aggregate --round <round-id>")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Aggregate reviewer submissions into a decision surface for the controller agent.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 || strings.TrimSpace(*roundID) == "" {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	result := review.Service{Workdir: workdir, Now: a.Now}.Aggregate(*roundID)
	return a.writeJSONResult(result)
}

func (a *App) runExecuteStart(args []string) int {
	fs := flag.NewFlagSet("harness execute start", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness execute start")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Record the explicit execution-start milestone for the current active plan.")
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
	result := lifecycle.Service{Workdir: workdir, Now: a.Now}.ExecuteStart()
	return a.writeJSONResult(result)
}

func (a *App) runLandComplete(args []string) int {
	fs := flag.NewFlagSet("harness land complete", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness land complete")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Record that post-merge cleanup is complete and restore idle worktree state.")
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
	result := lifecycle.Service{Workdir: workdir, Now: a.Now}.LandComplete()
	return a.writeJSONResult(result)
}

func (a *App) runArchive(args []string) int {
	fs := flag.NewFlagSet("harness archive", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness archive")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Freeze the current active plan for merge handoff.")
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
	result := lifecycle.Service{Workdir: workdir, Now: a.Now}.Archive()
	return a.writeJSONResult(result)
}

func (a *App) runReopen(args []string) int {
	fs := flag.NewFlagSet("harness reopen", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	mode := fs.String("mode", "", "Reopen mode: finalize-fix or new-step.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness reopen --mode <finalize-fix|new-step>")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Restore the current archived plan to active execution.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 || strings.TrimSpace(*mode) == "" {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	result := lifecycle.Service{Workdir: workdir, Now: a.Now}.Reopen(*mode)
	return a.writeJSONResult(result)
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
		now := a.Now()
		location := now.Location()
		day, err := time.ParseInLocation("2006-01-02", dateValue, location)
		if err != nil {
			return time.Time{}, fmt.Errorf("--date must be YYYY-MM-DD: %w", err)
		}
		return time.Date(day.Year(), day.Month(), day.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), location), nil
	}
	return a.Now(), nil
}

func (a *App) printRootUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness <command> [subcommand] [flags]")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Flags:")
	fmt.Fprintln(a.Stderr, "  --version       Print concise debug information for the running harness binary")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Commands:")
	fmt.Fprintln(a.Stderr, "  plan template   Render the packaged plan template")
	fmt.Fprintln(a.Stderr, "  plan lint       Validate a tracked plan")
	fmt.Fprintln(a.Stderr, "  execute start   Record the execution-start milestone")
	fmt.Fprintln(a.Stderr, "  evidence submit Record append-only CI, publish, or sync evidence")
	fmt.Fprintln(a.Stderr, "  review start    Create a deterministic review round")
	fmt.Fprintln(a.Stderr, "  review submit   Record one reviewer submission")
	fmt.Fprintln(a.Stderr, "  review aggregate Aggregate reviewer submissions")
	fmt.Fprintln(a.Stderr, "  land            Record merge confirmation for the archived candidate")
	fmt.Fprintln(a.Stderr, "  land complete   Record post-merge cleanup completion")
	fmt.Fprintln(a.Stderr, "  archive         Freeze the current active plan")
	fmt.Fprintln(a.Stderr, "  reopen          Restore the current archived plan")
	fmt.Fprintln(a.Stderr, "  status          Summarize the current plan and local execution state")
	fmt.Fprintln(a.Stderr, "  install         Install or refresh the harness-managed repository bootstrap")
}

func (a *App) printPlanUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness plan <subcommand> [flags]")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Subcommands:")
	fmt.Fprintln(a.Stderr, "  template   Render the packaged plan template")
	fmt.Fprintln(a.Stderr, "  lint       Validate a tracked plan")
}

func (a *App) printReviewUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness review <subcommand> [flags]")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Subcommands:")
	fmt.Fprintln(a.Stderr, "  start      Create a deterministic review round")
	fmt.Fprintln(a.Stderr, "  submit     Record one reviewer submission")
	fmt.Fprintln(a.Stderr, "  aggregate  Aggregate reviewer submissions")
}

func (a *App) printExecuteUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness execute <subcommand> [flags]")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Subcommands:")
	fmt.Fprintln(a.Stderr, "  start      Record the explicit execution-start milestone")
}

func (a *App) printEvidenceUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness evidence <subcommand> [flags]")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Subcommands:")
	fmt.Fprintln(a.Stderr, "  submit     Record append-only CI, publish, or sync evidence")
}

func (a *App) printLandUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness land --pr <url> [--commit <sha>]")
	fmt.Fprintln(a.Stderr, "   or: harness land complete")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Commands:")
	fmt.Fprintln(a.Stderr, "  land            Record merge confirmation and enter post-merge cleanup")
	fmt.Fprintln(a.Stderr, "  land complete   Record post-merge cleanup completion and restore idle")
}

type stringListFlag []string

func (f *stringListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func (a *App) readInput(path string) ([]byte, error) {
	if strings.TrimSpace(path) != "" {
		return os.ReadFile(path)
	}
	if a.Stdin == nil {
		return nil, fmt.Errorf("stdin is unavailable")
	}
	return io.ReadAll(a.Stdin)
}

func (a *App) writeJSONResult(value any) int {
	encoder := json.NewEncoder(a.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintf(a.Stderr, "encode JSON result: %v\n", err)
		return 1
	}

	switch result := value.(type) {
	case plan.LintResult:
		if result.OK {
			return 0
		}
	case status.Result:
		if result.OK {
			return 0
		}
	case review.StartResult:
		if result.OK {
			return 0
		}
	case review.SubmitResult:
		if result.OK {
			return 0
		}
	case review.AggregateResult:
		if result.OK {
			return 0
		}
	case evidence.Result:
		if result.OK {
			return 0
		}
	case lifecycle.Result:
		if result.OK {
			return 0
		}
	case install.Result:
		if result.OK {
			return 0
		}
	}
	return 1
}
