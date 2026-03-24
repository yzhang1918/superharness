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
	BuildVersion = ""
)

type Info struct {
	Version string
	Commit  string
	Mode    string
	Path    string
}

func (i Info) String() string {
	var lines []string
	if strings.TrimSpace(i.Version) != "" {
		lines = append(lines, fmt.Sprintf("version: %s", i.Version))
	}
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
	var buildInfo *debug.BuildInfo
	var ok bool
	if readBuildInfo != nil {
		buildInfo, ok = readBuildInfo()
	}

	info := Info{
		Version: resolveVersion(buildInfo, ok),
		Commit:  resolveCommit(buildInfo, ok),
		Mode:    resolveMode(),
	}
	if info.Mode == "dev" {
		if path, err := executablePath(); err == nil {
			info.Path = strings.TrimSpace(path)
		}
	}
	return info
}

func resolveVersion(buildInfo *debug.BuildInfo, ok bool) string {
	if version := strings.TrimSpace(BuildVersion); version != "" {
		return version
	}
	if ok && buildInfo != nil {
		if version := strings.TrimSpace(buildInfo.Main.Version); version != "" && version != "(devel)" {
			return version
		}
	}
	return ""
}

func resolveCommit(buildInfo *debug.BuildInfo, ok bool) string {
	if commit := strings.TrimSpace(BuildCommit); commit != "" {
		return commit
	}
	if ok && buildInfo != nil {
		for _, setting := range buildInfo.Settings {
			if setting.Key == "vcs.revision" {
				if commit := strings.TrimSpace(setting.Value); commit != "" {
					return commit
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
