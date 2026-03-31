package install

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	bootstrapassets "github.com/catu-ai/easyharness/assets/bootstrap"
)

const (
	ScopeAll    = "all"
	ScopeAgents = "agents"
	ScopeSkills = "skills"

	ActionCreate = "create"
	ActionUpdate = "update"
	ActionNoop   = "noop"

	agentsManagedBlockBegin = "<!-- easyharness:begin -->"
	agentsManagedBlockEnd   = "<!-- easyharness:end -->"
)

var (
	agentsManagedBlockBeginPattern = regexp.MustCompile(`(?m)^<!-- easyharness:begin -->[ \t]*\r?$`)
	agentsManagedBlockEndPattern   = regexp.MustCompile(`(?m)^<!-- easyharness:end -->[ \t]*\r?$`)
)

type Service struct {
	Workdir string
}

type Options struct {
	Scope  string
	DryRun bool
}

type Result struct {
	OK         bool           `json:"ok"`
	Command    string         `json:"command"`
	Summary    string         `json:"summary"`
	Mode       string         `json:"mode"`
	Scope      string         `json:"scope"`
	Actions    []Action       `json:"actions"`
	NextAction []NextAction   `json:"next_actions"`
	Errors     []CommandError `json:"errors,omitempty"`
}

type Action struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Details string `json:"details"`
}

type NextAction struct {
	Command     *string `json:"command"`
	Description string  `json:"description"`
}

type CommandError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type plannedWrite struct {
	relPath string
	absPath string
	kind    string
	details string
	content []byte
}

func (s Service) Install(opts Options) Result {
	scope := normalizeScope(opts.Scope)
	if !isValidScope(scope) {
		return Result{
			OK:      false,
			Command: "install",
			Summary: "Unsupported install scope.",
			Mode:    modeName(opts.DryRun),
			Scope:   scope,
			Actions: []Action{},
			Errors: []CommandError{{
				Path:    "scope",
				Message: fmt.Sprintf("supported scopes are %q, %q, and %q", ScopeAgents, ScopeSkills, ScopeAll),
			}},
			NextAction: []NextAction{},
		}
	}

	var writes []plannedWrite
	var errs []CommandError

	if scope == ScopeAll || scope == ScopeAgents {
		planned, err := s.planAgents()
		if err != nil {
			errs = append(errs, *err)
		} else {
			writes = append(writes, planned)
		}
	}
	if scope == ScopeAll || scope == ScopeSkills {
		planned, skillErrs := s.planSkills()
		writes = append(writes, planned...)
		errs = append(errs, skillErrs...)
	}
	if len(errs) > 0 {
		return Result{
			OK:         false,
			Command:    "install",
			Summary:    "Unable to prepare the harness-managed repository install.",
			Mode:       modeName(opts.DryRun),
			Scope:      scope,
			Actions:    toActions(writes),
			Errors:     errs,
			NextAction: []NextAction{},
		}
	}

	if !opts.DryRun {
		for _, write := range writes {
			if write.kind == ActionNoop {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(write.absPath), 0o755); err != nil {
				return Result{
					OK:      false,
					Command: "install",
					Summary: "Unable to write the harness-managed repository install.",
					Mode:    modeName(opts.DryRun),
					Scope:   scope,
					Actions: toActions(writes),
					Errors: []CommandError{{
						Path:    write.relPath,
						Message: fmt.Sprintf("create parent directory: %v", err),
					}},
					NextAction: []NextAction{},
				}
			}
			if err := os.WriteFile(write.absPath, write.content, 0o644); err != nil {
				return Result{
					OK:      false,
					Command: "install",
					Summary: "Unable to write the harness-managed repository install.",
					Mode:    modeName(opts.DryRun),
					Scope:   scope,
					Actions: toActions(writes),
					Errors: []CommandError{{
						Path:    write.relPath,
						Message: fmt.Sprintf("write file: %v", err),
					}},
					NextAction: []NextAction{},
				}
			}
		}
	}

	summary := summarizeWrites(writes, opts.DryRun)
	creates, updates := countWrites(writes)
	nextActions := []NextAction{}
	if opts.DryRun && (creates > 0 || updates > 0) {
		runCommand := "harness install"
		if scope != ScopeAll {
			runCommand = fmt.Sprintf("harness install --scope %s", scope)
		}
		nextActions = append(nextActions, NextAction{
			Command:     &runCommand,
			Description: "Run without --dry-run to apply the planned harness-managed repository changes.",
		})
	}

	return Result{
		OK:         true,
		Command:    "install",
		Summary:    summary,
		Mode:       modeName(opts.DryRun),
		Scope:      scope,
		Actions:    toActions(writes),
		NextAction: nextActions,
	}
}

func (s Service) planAgents() (plannedWrite, *CommandError) {
	targetPath := filepath.Join(s.Workdir, "AGENTS.md")
	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			managedBlock := renderManagedBlock("\n")
			content := "# AGENTS.md\n\n" + managedBlock
			return plannedWrite{
				relPath: "AGENTS.md",
				absPath: targetPath,
				kind:    ActionCreate,
				details: "Create AGENTS.md with the harness-managed workflow block.",
				content: []byte(content),
			}, nil
		}
		return plannedWrite{}, &CommandError{Path: "AGENTS.md", Message: err.Error()}
	}

	existing := string(data)
	lineEnding := detectLineEnding(existing)
	managedBlock := renderManagedBlock(lineEnding)
	beginMatches := agentsManagedBlockBeginPattern.FindAllStringIndex(existing, -1)
	endMatches := agentsManagedBlockEndPattern.FindAllStringIndex(existing, -1)
	beginCount := len(beginMatches)
	endCount := len(endMatches)
	if beginCount != endCount || beginCount > 1 {
		return plannedWrite{}, &CommandError{
			Path:    "AGENTS.md",
			Message: "expected zero or one easyharness managed block; found an ambiguous marker layout",
		}
	}

	var next string
	var kind string
	var details string
	switch beginCount {
	case 0:
		trimmed := strings.TrimSpace(existing)
		if trimmed == "" {
			next = "# AGENTS.md" + lineEnding + lineEnding + managedBlock
			kind = ActionUpdate
			details = "Populate the empty AGENTS.md file with the harness-managed workflow block."
		} else {
			next = trimTrailingLineBreaks(existing) + lineEnding + lineEnding + managedBlock
			kind = ActionUpdate
			details = "Append the harness-managed workflow block to AGENTS.md."
		}
	default:
		begin := beginMatches[0][0]
		end := endMatches[0][0]
		if begin < 0 || end < 0 || begin > end {
			return plannedWrite{}, &CommandError{
				Path:    "AGENTS.md",
				Message: "unable to locate a valid easyharness managed block",
			}
		}
		end = endMatches[0][1]
		before := trimTrailingLineBreaks(existing[:begin])
		after := trimSurroundingLineBreaks(existing[end:])
		parts := []string{}
		if before != "" {
			parts = append(parts, before)
		}
		parts = append(parts, trimTrailingLineBreaks(managedBlock))
		if after != "" {
			parts = append(parts, after)
		}
		next = strings.Join(parts, lineEnding+lineEnding) + lineEnding
		kind = ActionUpdate
		details = "Refresh the existing harness-managed AGENTS.md block in place."
	}

	if next == existing {
		kind = ActionNoop
		details = "AGENTS.md already contains the current harness-managed workflow block."
	}

	return plannedWrite{
		relPath: "AGENTS.md",
		absPath: targetPath,
		kind:    kind,
		details: details,
		content: []byte(next),
	}, nil
}

func (s Service) planSkills() ([]plannedWrite, []CommandError) {
	files, err := bootstrapassets.SkillFiles()
	if err != nil {
		return nil, []CommandError{{Path: ".agents/skills", Message: err.Error()}}
	}
	relPaths := make([]string, 0, len(files))
	for relPath := range files {
		relPaths = append(relPaths, relPath)
	}
	sort.Strings(relPaths)

	writes := make([]plannedWrite, 0, len(relPaths))
	for _, relPath := range relPaths {
		rel := filepath.ToSlash(filepath.Join(".agents/skills", filepath.FromSlash(relPath)))
		abs := filepath.Join(s.Workdir, filepath.FromSlash(rel))
		content := []byte(files[relPath])
		existing, err := os.ReadFile(abs)
		if err != nil {
			if os.IsNotExist(err) {
				writes = append(writes, plannedWrite{
					relPath: rel,
					absPath: abs,
					kind:    ActionCreate,
					details: "Create the harness-managed repo-local skill file from packaged bootstrap assets.",
					content: content,
				})
				continue
			}
			return writes, []CommandError{{Path: rel, Message: err.Error()}}
		}
		kind := ActionUpdate
		details := "Refresh the harness-managed repo-local skill file from packaged bootstrap assets."
		if string(existing) == string(content) {
			kind = ActionNoop
			details = "Repo-local skill file already matches the packaged bootstrap asset."
		}
		writes = append(writes, plannedWrite{
			relPath: rel,
			absPath: abs,
			kind:    kind,
			details: details,
			content: content,
		})
	}
	return writes, nil
}

func renderManagedBlock(lineEnding string) string {
	body := normalizeLineEndings(strings.TrimSpace(bootstrapassets.AgentsManagedBlock()), lineEnding)
	return agentsManagedBlockBegin + lineEnding + body + lineEnding + agentsManagedBlockEnd + lineEnding
}

func normalizeScope(scope string) string {
	scope = strings.TrimSpace(strings.ToLower(scope))
	if scope == "" {
		return ScopeAll
	}
	return scope
}

func isValidScope(scope string) bool {
	switch scope {
	case ScopeAll, ScopeAgents, ScopeSkills:
		return true
	default:
		return false
	}
}

func modeName(dryRun bool) string {
	if dryRun {
		return "dry_run"
	}
	return "apply"
}

func toActions(writes []plannedWrite) []Action {
	actions := make([]Action, 0, len(writes))
	for _, write := range writes {
		actions = append(actions, Action{
			Path:    write.relPath,
			Kind:    write.kind,
			Details: write.details,
		})
	}
	return actions
}

func summarizeWrites(writes []plannedWrite, dryRun bool) string {
	creates, updates := countWrites(writes)
	if creates == 0 && updates == 0 {
		if dryRun {
			return "Dry run complete. Harness-managed repository assets are already up to date."
		}
		return "Harness-managed repository assets are already up to date."
	}
	if dryRun {
		return fmt.Sprintf("Dry run complete. %d file(s) would be created and %d file(s) would be updated.", creates, updates)
	}
	return fmt.Sprintf("Installed or refreshed harness-managed repository assets. %d file(s) created and %d file(s) updated.", creates, updates)
}

func detectLineEnding(content string) string {
	if strings.Contains(content, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

func normalizeLineEndings(content, lineEnding string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return strings.ReplaceAll(content, "\n", lineEnding)
}

func trimTrailingLineBreaks(content string) string {
	return strings.TrimRight(content, "\r\n")
}

func trimSurroundingLineBreaks(content string) string {
	return strings.Trim(content, "\r\n")
}

func countWrites(writes []plannedWrite) (int, int) {
	var creates, updates int
	for _, write := range writes {
		switch write.kind {
		case ActionCreate:
			creates++
		case ActionUpdate:
			updates++
		}
	}
	return creates, updates
}
