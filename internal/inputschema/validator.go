package inputschema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/catu-ai/easyharness/internal/contracts"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

var quotedPropertyPattern = regexp.MustCompile(`'([^']+)'`)

const (
	SchemaReviewSpec       = "inputs.review.spec"
	SchemaReviewSubmission = "inputs.review.submission"
	SchemaEvidenceCI       = "inputs.evidence.ci"
	SchemaEvidencePublish  = "inputs.evidence.publish"
	SchemaEvidenceSync     = "inputs.evidence.sync"
)

type compiledSchema struct {
	once   sync.Once
	schema *jsonschema.Schema
	err    error
}

var compiledByKey sync.Map

// Validate validates raw JSON bytes against one generated command-input schema
// and returns normalized harness error details.
func Validate(schemaKey, rootLabel string, input []byte) []contracts.ErrorDetail {
	decoded, err := jsonschema.UnmarshalJSON(bytes.NewReader(input))
	if err != nil {
		return []contracts.ErrorDetail{{
			Path:    rootLabel,
			Message: fmt.Sprintf("parse input JSON: %v", err),
		}}
	}
	schema, err := loadCompiledSchema(schemaKey)
	if err != nil {
		return []contracts.ErrorDetail{{
			Path:    rootLabel,
			Message: fmt.Sprintf("load generated input schema: %v", err),
		}}
	}
	if err := schema.Validate(decoded); err != nil {
		var validationErr *jsonschema.ValidationError
		if !errors.As(err, &validationErr) {
			return []contracts.ErrorDetail{{
				Path:    rootLabel,
				Message: fmt.Sprintf("validate input against schema: %v", err),
			}}
		}
		issues := flattenValidationErrors(rootLabel, validationErr.DetailedOutput())
		issues = pruneParentIssues(issues)
		if len(issues) > 0 {
			return issues
		}
		return []contracts.ErrorDetail{{
			Path:    rootLabel,
			Message: validationErr.Error(),
		}}
	}
	return nil
}

// DecodeAndValidate decodes raw JSON bytes into the target Go type only after
// they pass schema validation.
func DecodeAndValidate[T any](schemaKey, rootLabel string, input []byte, out *T) []contracts.ErrorDetail {
	if issues := Validate(schemaKey, rootLabel, input); len(issues) > 0 {
		return issues
	}
	if err := json.Unmarshal(input, out); err != nil {
		return []contracts.ErrorDetail{{
			Path:    rootLabel,
			Message: fmt.Sprintf("decode validated input JSON: %v", err),
		}}
	}
	return nil
}

func loadCompiledSchema(schemaKey string) (*jsonschema.Schema, error) {
	entry, ok := schemaEntry(schemaKey)
	if !ok {
		return nil, fmt.Errorf("unknown input schema key %q", schemaKey)
	}
	cacheValue, _ := compiledByKey.LoadOrStore(schemaKey, &compiledSchema{})
	cached := cacheValue.(*compiledSchema)
	cached.once.Do(func() {
		cached.schema, cached.err = compileSchema(entry)
	})
	return cached.schema, cached.err
}

func compileSchema(entry contracts.SchemaEntry) (*jsonschema.Schema, error) {
	rendered, ok := generatedInputSchemas[entry.Path]
	if !ok {
		return nil, fmt.Errorf("schema bytes missing for %s", entry.Path)
	}
	var document any
	if err := json.Unmarshal(rendered, &document); err != nil {
		return nil, fmt.Errorf("reparse %s: %w", entry.Path, err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(entry.Path, document); err != nil {
		return nil, fmt.Errorf("register %s: %w", entry.Path, err)
	}
	schema, err := compiler.Compile(entry.Path)
	if err != nil {
		return nil, fmt.Errorf("compile %s: %w", entry.Path, err)
	}
	return schema, nil
}

func schemaEntry(schemaKey string) (contracts.SchemaEntry, bool) {
	for _, entry := range contracts.SchemaRegistry() {
		if entry.Key == schemaKey && entry.Group == "command_inputs" {
			return entry, true
		}
	}
	return contracts.SchemaEntry{}, false
}

func flattenValidationErrors(rootLabel string, unit *jsonschema.OutputUnit) []contracts.ErrorDetail {
	if unit == nil {
		return nil
	}
	if len(unit.Errors) == 0 {
		return renderIssueDetails(rootLabel, unit.InstanceLocation, unit.Error.String())
	}
	issues := make([]contracts.ErrorDetail, 0)
	for i := range unit.Errors {
		issues = append(issues, flattenValidationErrors(rootLabel, &unit.Errors[i])...)
	}
	return pruneParentIssues(issues)
}

func pruneParentIssues(issues []contracts.ErrorDetail) []contracts.ErrorDetail {
	if len(issues) < 2 {
		return issues
	}
	filtered := make([]contracts.ErrorDetail, 0, len(issues))
	for i, issue := range issues {
		if shouldDropParentIssue(i, issues) {
			continue
		}
		filtered = append(filtered, issue)
	}
	return filtered
}

func shouldDropParentIssue(index int, issues []contracts.ErrorDetail) bool {
	path := issues[index].Path
	if strings.TrimSpace(path) == "" {
		return false
	}
	for i, other := range issues {
		if i == index {
			continue
		}
		if strings.HasPrefix(other.Path, path+".") || strings.HasPrefix(other.Path, path+"[") {
			return true
		}
	}
	return false
}

func renderIssueDetails(rootLabel, instanceLocation, message string) []contracts.ErrorDetail {
	base := rootLabel + renderInstanceLocation(instanceLocation)
	properties := propertiesFromValidationMessage(message)
	if len(properties) == 0 {
		return []contracts.ErrorDetail{{
			Path:    base,
			Message: message,
		}}
	}
	issues := make([]contracts.ErrorDetail, 0, len(properties))
	for _, property := range properties {
		issues = append(issues, contracts.ErrorDetail{
			Path:    appendPropertyPath(base, property),
			Message: message,
		})
	}
	return issues
}

func renderInstanceLocation(pointer string) string {
	if strings.TrimSpace(pointer) == "" || pointer == "/" {
		return ""
	}
	segments := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	var b strings.Builder
	for _, segment := range segments {
		segment = decodePointerToken(segment)
		if segment == "" {
			continue
		}
		if isDigits(segment) {
			b.WriteString("[")
			b.WriteString(segment)
			b.WriteString("]")
			continue
		}
		b.WriteString(".")
		b.WriteString(segment)
	}
	return b.String()
}

func appendPropertyPath(base, property string) string {
	if base == "" {
		return property
	}
	return base + "." + property
}

func propertiesFromValidationMessage(message string) []string {
	matches := quotedPropertyPattern.FindAllStringSubmatch(message, -1)
	if len(matches) == 0 {
		return nil
	}
	properties := make([]string, 0, len(matches))
	seen := make(map[string]bool, len(matches))
	for _, match := range matches {
		property := match[1]
		if seen[property] {
			continue
		}
		seen[property] = true
		properties = append(properties, property)
	}
	return properties
}

func decodePointerToken(token string) string {
	token = strings.ReplaceAll(token, "~1", "/")
	token = strings.ReplaceAll(token, "~0", "~")
	return token
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
