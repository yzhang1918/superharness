package version

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
)

var (
	BuildCommit = ""
	BuildMode   = ""
)

type Info struct {
	Commit string
	Mode   string
	Path   string
}

func (i Info) String() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("mode: %s", i.Mode))
	lines = append(lines, fmt.Sprintf("commit: %s", i.Commit))
	if i.Mode == "dev" && strings.TrimSpace(i.Path) != "" {
		lines = append(lines, fmt.Sprintf("path: %s", i.Path))
	}
	return strings.Join(lines, "\n") + "\n"
}

func Current() Info {
	return current(debug.ReadBuildInfo, os.Executable)
}

func current(readBuildInfo func() (*debug.BuildInfo, bool), executablePath func() (string, error)) Info {
	info := Info{
		Commit: resolveCommit(readBuildInfo),
		Mode:   resolveMode(),
	}
	if info.Mode == "dev" {
		if path, err := executablePath(); err == nil {
			info.Path = strings.TrimSpace(path)
		}
	}
	return info
}

func resolveCommit(readBuildInfo func() (*debug.BuildInfo, bool)) string {
	if commit := strings.TrimSpace(BuildCommit); commit != "" {
		return commit
	}
	if readBuildInfo != nil {
		if buildInfo, ok := readBuildInfo(); ok {
			for _, setting := range buildInfo.Settings {
				if setting.Key == "vcs.revision" {
					if commit := strings.TrimSpace(setting.Value); commit != "" {
						return commit
					}
				}
			}
		}
	}
	return "unknown"
}

func resolveMode() string {
	if mode := strings.TrimSpace(BuildMode); mode != "" {
		return mode
	}
	return "release"
}
