package support

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func RewritePlanPreservingFrontmatter(t *testing.T, path, title, body string) {
	t.Helper()

	content := readPlanFile(t, path)
	frontmatter := extractFrontmatter(t, content)
	rewritten := frontmatter + "\n\n# " + strings.TrimSpace(title) + "\n\n" + strings.TrimSpace(body) + "\n"
	writePlanFile(t, path, rewritten)
}

func CheckAllAcceptanceCriteria(t *testing.T, path string) {
	t.Helper()

	content := readPlanFile(t, path)
	updated, replaced := rewriteSection(content, "## Acceptance Criteria", func(section string) string {
		lines := strings.Split(section, "\n")
		count := 0
		for i, line := range lines {
			if strings.HasPrefix(line, "- [ ] ") {
				lines[i] = strings.Replace(line, "- [ ] ", "- [x] ", 1)
				count++
			}
		}
		if count == 0 {
			t.Fatalf("expected unchecked acceptance criteria in %s", path)
		}
		return strings.Join(lines, "\n")
	})
	if !replaced {
		t.Fatalf("acceptance criteria section not found in %s", path)
	}
	writePlanFile(t, path, updated)
}

func CompleteStep(t *testing.T, path string, stepNumber int, executionNotes, reviewNotes string) {
	t.Helper()

	content := readPlanFile(t, path)
	heading := fmt.Sprintf("### Step %d:", stepNumber)
	stepStart := strings.Index(content, heading)
	if stepStart < 0 {
		t.Fatalf("step heading %q not found in %s", heading, path)
	}

	stepEnd := len(content)
	for _, marker := range []string{"\n### Step ", "\n## Validation Strategy"} {
		if idx := strings.Index(content[stepStart+1:], marker); idx >= 0 {
			candidate := stepStart + 1 + idx
			if candidate < stepEnd {
				stepEnd = candidate
			}
		}
	}

	block := content[stepStart:stepEnd]
	if strings.Contains(block, "- Done: [ ]") {
		block = strings.Replace(block, "- Done: [ ]", "- Done: [x]", 1)
	} else if !strings.Contains(block, "- Done: [x]") {
		t.Fatalf("done marker not found in step %d for %s", stepNumber, path)
	}

	block = replaceSubsectionBody(t, path, block, "#### Execution Notes", executionNotes)
	block = replaceSubsectionBody(t, path, block, "#### Review Notes", reviewNotes)
	writePlanFile(t, path, content[:stepStart]+block+content[stepEnd:])
}

func AppendStepBeforeValidationStrategy(t *testing.T, path, stepMarkdown string) {
	t.Helper()

	content := readPlanFile(t, path)
	marker := "\n## Validation Strategy"
	index := strings.Index(content, marker)
	if index < 0 {
		t.Fatalf("validation strategy section not found in %s", path)
	}

	stepBlock := strings.TrimSpace(stepMarkdown)
	if stepBlock == "" {
		t.Fatalf("expected non-empty step markdown for %s", path)
	}

	updated := content[:index] + "\n\n" + stepBlock + content[index:]
	writePlanFile(t, path, updated)
}

func readPlanFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read plan %s: %v", path, err)
	}
	return string(data)
}

func writePlanFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write plan %s: %v", path, err)
	}
}

func extractFrontmatter(t *testing.T, content string) string {
	t.Helper()

	if !strings.HasPrefix(content, "---\n") {
		t.Fatalf("expected frontmatter block at start of plan")
	}
	rest := strings.TrimPrefix(content, "---\n")
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		t.Fatalf("expected closing frontmatter delimiter")
	}
	return "---\n" + rest[:end] + "\n---"
}

func rewriteSection(content, heading string, rewrite func(section string) string) (string, bool) {
	start := strings.Index(content, heading)
	if start < 0 {
		return content, false
	}

	bodyStart := start + len(heading)
	end := len(content)
	if idx := strings.Index(content[bodyStart:], "\n## "); idx >= 0 {
		end = bodyStart + idx
	}

	section := content[bodyStart:end]
	return content[:bodyStart] + rewrite(section) + content[end:], true
}

func replaceSubsectionBody(t *testing.T, path, block, heading, body string) string {
	t.Helper()

	start := strings.Index(block, heading)
	if start < 0 {
		t.Fatalf("subsection %q not found in %s", heading, path)
	}

	bodyStart := start + len(heading)
	if !strings.HasPrefix(block[bodyStart:], "\n") {
		t.Fatalf("expected newline after %q in %s", heading, path)
	}
	bodyStart++

	bodyEnd := len(block)
	for _, marker := range []string{"\n#### ", "\n### ", "\n## "} {
		if idx := strings.Index(block[bodyStart:], marker); idx >= 0 {
			candidate := bodyStart + idx
			if candidate < bodyEnd {
				bodyEnd = candidate
			}
		}
	}

	return block[:bodyStart] + strings.TrimSpace(body) + "\n" + block[bodyEnd:]
}
