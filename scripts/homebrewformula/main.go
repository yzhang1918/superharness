package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	formulaClassName = "Easyharness"
	formulaName      = "easyharness"
	formulaDesc      = "Thin, git-native harness CLI for human-steered, agent-executed work"
	formulaHomepage  = "https://github.com/catu-ai/easyharness"
	formulaLicense   = "MIT"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	repo := flag.String("repo", "", "GitHub repo in owner/name form")
	tag := flag.String("tag", "", "Release tag in v-prefixed form")
	checksumsPath := flag.String("checksums", "", "Path to SHA256SUMS")
	outputPath := flag.String("output", "", "Optional output path; stdout when omitted")
	flag.Parse()

	if *repo == "" || *tag == "" || *checksumsPath == "" {
		return fmt.Errorf("repo, tag, and checksums are required")
	}
	if strings.Count(*repo, "/") != 1 {
		return fmt.Errorf("repo must be in owner/name form, got %s", *repo)
	}

	version, err := versionFromTag(*tag)
	if err != nil {
		return err
	}

	checksumData, err := os.ReadFile(*checksumsPath)
	if err != nil {
		return fmt.Errorf("read SHA256SUMS %s: %w", *checksumsPath, err)
	}

	checksums, err := parseChecksums(string(checksumData))
	if err != nil {
		return err
	}

	formula, err := renderFormula(*repo, *tag, version, checksums)
	if err != nil {
		return err
	}

	if *outputPath == "" {
		fmt.Print(formula)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		return fmt.Errorf("create formula output directory for %s: %w", *outputPath, err)
	}
	if err := os.WriteFile(*outputPath, []byte(formula), 0o644); err != nil {
		return fmt.Errorf("write formula %s: %w", *outputPath, err)
	}
	return nil
}

func versionFromTag(tag string) (string, error) {
	if !strings.HasPrefix(tag, "v") || tag == "v" {
		return "", fmt.Errorf("tag must be v-prefixed, got %s", tag)
	}
	return strings.TrimPrefix(tag, "v"), nil
}

func renderFormula(repo, tag, version string, checksums map[string]string) (string, error) {
	darwinArm64Asset, err := formulaAsset(tag, "darwin", "arm64", checksums)
	if err != nil {
		return "", err
	}
	darwinAMD64Asset, err := formulaAsset(tag, "darwin", "amd64", checksums)
	if err != nil {
		return "", err
	}
	linuxArm64Asset, err := formulaAsset(tag, "linux", "arm64", checksums)
	if err != nil {
		return "", err
	}
	linuxAMD64Asset, err := formulaAsset(tag, "linux", "amd64", checksums)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "class %s < Formula\n", formulaClassName)
	fmt.Fprintf(&builder, "  desc %q\n", formulaDesc)
	fmt.Fprintf(&builder, "  homepage %q\n", formulaHomepage)
	fmt.Fprintf(&builder, "  license %q\n", formulaLicense)
	fmt.Fprintf(&builder, "  version %q\n", version)
	builder.WriteString("\n")
	builder.WriteString("  on_macos do\n")
	builder.WriteString("    if Hardware::CPU.arm?\n")
	fmt.Fprintf(&builder, "      url %q\n", formulaURL(repo, tag, darwinArm64Asset.name))
	fmt.Fprintf(&builder, "      sha256 %q\n", darwinArm64Asset.checksum)
	builder.WriteString("    else\n")
	fmt.Fprintf(&builder, "      url %q\n", formulaURL(repo, tag, darwinAMD64Asset.name))
	fmt.Fprintf(&builder, "      sha256 %q\n", darwinAMD64Asset.checksum)
	builder.WriteString("    end\n")
	builder.WriteString("  end\n")
	builder.WriteString("\n")
	builder.WriteString("  on_linux do\n")
	builder.WriteString("    if Hardware::CPU.arm?\n")
	fmt.Fprintf(&builder, "      url %q\n", formulaURL(repo, tag, linuxArm64Asset.name))
	fmt.Fprintf(&builder, "      sha256 %q\n", linuxArm64Asset.checksum)
	builder.WriteString("    else\n")
	fmt.Fprintf(&builder, "      url %q\n", formulaURL(repo, tag, linuxAMD64Asset.name))
	fmt.Fprintf(&builder, "      sha256 %q\n", linuxAMD64Asset.checksum)
	builder.WriteString("    end\n")
	builder.WriteString("  end\n")
	builder.WriteString("\n")
	builder.WriteString("  def install\n")
	builder.WriteString("    bin.install Dir[\"**/harness\"].fetch(0) => \"harness\"\n")
	builder.WriteString("  end\n")
	builder.WriteString("\n")
	builder.WriteString("  test do\n")
	builder.WriteString("    output = shell_output(\"#{bin}/harness --version\")\n")
	builder.WriteString("    assert_match \"version: v#{version}\", output\n")
	builder.WriteString("    assert_match \"mode: release\", output\n")
	builder.WriteString("  end\n")
	builder.WriteString("end\n")

	return builder.String(), nil
}

type assetSpec struct {
	name     string
	checksum string
}

func formulaAsset(tag, goos, goarch string, checksums map[string]string) (*assetSpec, error) {
	name := fmt.Sprintf("%s_%s_%s_%s.zip", formulaName, tag, goos, goarch)
	checksum, ok := checksums[name]
	if !ok {
		return nil, fmt.Errorf("SHA256SUMS is missing required checksum entry for %s", name)
	}
	return &assetSpec{name: name, checksum: checksum}, nil
}

func formulaURL(repo, tag, assetName string) string {
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, tag, assetName)
}

func parseChecksums(contents string) (map[string]string, error) {
	result := map[string]string{}
	for _, line := range strings.Split(contents, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("invalid SHA256SUMS line: %s", line)
		}
		result[fields[1]] = fields[0]
	}
	return result, nil
}
