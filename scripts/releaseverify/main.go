package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type assetFlags []string

func (f *assetFlags) String() string {
	return strings.Join(*f, ",")
}

func (f *assetFlags) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("asset names must not be empty")
	}
	*f = append(*f, value)
	return nil
}

type repoView struct {
	NameWithOwner string `json:"nameWithOwner"`
	URL           string `json:"url"`
}

type releaseView struct {
	URL     string `json:"url"`
	TagName string `json:"tagName"`
	Assets  []struct {
		Name string `json:"name"`
	} `json:"assets"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	repo := flag.String("repo", "", "GitHub repo in owner/name form")
	tag := flag.String("tag", "", "Release tag to verify")
	downloadDir := flag.String("download-dir", "", "Optional directory for downloaded release assets")
	var assets assetFlags
	flag.Var(&assets, "asset", "Release asset name to require (repeatable)")
	flag.Parse()

	if *repo == "" || *tag == "" {
		return fmt.Errorf("repo and tag are required")
	}
	if len(assets) == 0 {
		return fmt.Errorf("at least one --asset is required")
	}

	repoPayload, err := readRepo(*repo)
	if err != nil {
		return err
	}
	releasePayload, err := readRelease(*repo, *tag)
	if err != nil {
		return err
	}

	assetSet := map[string]bool{}
	for _, asset := range releasePayload.Assets {
		assetSet[asset.Name] = true
	}
	for _, asset := range assets {
		if !assetSet[asset] {
			return fmt.Errorf("release %s in %s is missing required asset %s", *tag, *repo, asset)
		}
	}

	fmt.Printf("Verified repo: %s (%s)\n", repoPayload.NameWithOwner, repoPayload.URL)
	fmt.Printf("Verified release: %s (%s)\n", releasePayload.TagName, releasePayload.URL)
	fmt.Printf("Verified assets: %s\n", strings.Join([]string(assets), ", "))

	if *downloadDir != "" {
		if err := downloadAssets(*repo, *tag, *downloadDir, assets); err != nil {
			return err
		}
		if err := verifyDownloadedAssets(*downloadDir, assets); err != nil {
			return err
		}
		fmt.Printf("Verified downloaded assets in %s\n", *downloadDir)
	}

	return nil
}

func readRepo(repo string) (*repoView, error) {
	output, err := runGH("repo", "view", repo, "--json", "nameWithOwner,url")
	if err != nil {
		return nil, fmt.Errorf("load repo metadata for %s: %w", repo, err)
	}
	var payload repoView
	if err := json.Unmarshal(output, &payload); err != nil {
		return nil, fmt.Errorf("decode repo metadata for %s: %w", repo, err)
	}
	if payload.NameWithOwner != repo {
		return nil, fmt.Errorf("expected repo %s, got %s", repo, payload.NameWithOwner)
	}
	return &payload, nil
}

func readRelease(repo, tag string) (*releaseView, error) {
	output, err := runGH("release", "view", tag, "-R", repo, "--json", "url,tagName,assets")
	if err != nil {
		return nil, fmt.Errorf("load release metadata for %s in %s: %w", tag, repo, err)
	}
	var payload releaseView
	if err := json.Unmarshal(output, &payload); err != nil {
		return nil, fmt.Errorf("decode release metadata for %s in %s: %w", tag, repo, err)
	}
	if payload.TagName != tag {
		return nil, fmt.Errorf("expected release tag %s, got %s", tag, payload.TagName)
	}
	return &payload, nil
}

func downloadAssets(repo, tag, downloadDir string, assets []string) error {
	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		return fmt.Errorf("create download dir %s: %w", downloadDir, err)
	}
	args := []string{"release", "download", tag, "-R", repo, "-D", downloadDir, "--clobber"}
	for _, asset := range assets {
		args = append(args, "-p", asset)
	}
	if _, err := runGH(args...); err != nil {
		return fmt.Errorf("download assets for %s in %s: %w", tag, repo, err)
	}
	for _, asset := range assets {
		path := filepath.Join(downloadDir, asset)
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("expected downloaded asset %s: %w", path, err)
		}
	}
	return nil
}

func verifyDownloadedAssets(downloadDir string, assets []string) error {
	hasChecksums := false
	for _, asset := range assets {
		if asset == "SHA256SUMS" {
			hasChecksums = true
			break
		}
	}
	if !hasChecksums {
		return nil
	}

	checksumPath := filepath.Join(downloadDir, "SHA256SUMS")
	checksumData, err := os.ReadFile(checksumPath)
	if err != nil {
		return fmt.Errorf("read downloaded SHA256SUMS: %w", err)
	}
	checksums, err := parseChecksums(string(checksumData))
	if err != nil {
		return err
	}

	for _, asset := range assets {
		if asset == "SHA256SUMS" {
			continue
		}
		expected, ok := checksums[asset]
		if !ok {
			return fmt.Errorf("SHA256SUMS is missing an entry for %s", asset)
		}
		actual, err := sha256File(filepath.Join(downloadDir, asset))
		if err != nil {
			return err
		}
		if actual != expected {
			return fmt.Errorf("checksum mismatch for %s: got %s want %s", asset, actual, expected)
		}
	}
	return nil
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

func sha256File(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func runGH(args ...string) ([]byte, error) {
	cmd := exec.Command("gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%v failed: %w\n%s", args, err, strings.TrimSpace(string(output)))
	}
	return output, nil
}
