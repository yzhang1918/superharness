package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	sourcePath := flag.String("source", "", "staged artifact to publish")
	expectedDir := flag.String("expected-dir", "", "prepared output directory path before publish")
	destName := flag.String("dest-name", "", "final output file name")
	flag.Parse()

	if err := publish(*sourcePath, *expectedDir, *destName); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func publish(sourcePath, expectedDir, destName string) error {
	if sourcePath == "" || expectedDir == "" || destName == "" {
		return fmt.Errorf("source, expected-dir, and dest-name are required")
	}
	if destName == "." || destName == ".." || strings.ContainsRune(destName, os.PathSeparator) {
		return fmt.Errorf("destination name must be a simple file name: %s", destName)
	}
	if err := requireCurrentDir(expectedDir); err != nil {
		return err
	}

	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open staged output %s: %w", sourcePath, err)
	}
	defer sourceFile.Close()

	stagedOutput, err := os.CreateTemp(".", "."+destName+".publish.*")
	if err != nil {
		return fmt.Errorf("create publish staging file in prepared output directory: %w", err)
	}
	stagedOutputPath := stagedOutput.Name()
	cleanupStaged := true
	defer func() {
		if cleanupStaged {
			_ = os.Remove(stagedOutputPath)
		}
	}()

	if _, err := io.Copy(stagedOutput, sourceFile); err != nil {
		stagedOutput.Close()
		return fmt.Errorf("copy staged output for %s: %w", destName, err)
	}
	if err := stagedOutput.Chmod(0o644); err != nil {
		stagedOutput.Close()
		return fmt.Errorf("chmod staged output for %s: %w", destName, err)
	}
	if err := stagedOutput.Close(); err != nil {
		return fmt.Errorf("close staged output for %s: %w", destName, err)
	}
	if err := requireCurrentDir(expectedDir); err != nil {
		return err
	}

	destinationPath := filepath.Join(".", destName)
	if err := os.Rename(stagedOutputPath, destinationPath); err != nil {
		return fmt.Errorf("publish %s into prepared output directory: %w", destName, err)
	}

	cleanupStaged = false
	return nil
}

func requireCurrentDir(expectedDir string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve prepared output directory before publish: %w", err)
	}
	if currentDir != expectedDir {
		return fmt.Errorf("prepared output directory changed unexpectedly during build: expected %s, got %s", expectedDir, currentDir)
	}
	return nil
}
