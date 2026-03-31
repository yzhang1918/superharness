package bootstrap

import (
	"embed"
	"io/fs"
	"path"
	"sort"
	"strings"
)

var (
	//go:embed agents-managed-block.md skills
	embeddedAssets embed.FS
)

func AgentsManagedBlock() string {
	data, err := fs.ReadFile(embeddedAssets, "agents-managed-block.md")
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(data)) + "\n"
}

func SkillFiles() (map[string]string, error) {
	files := map[string]string{}
	err := fs.WalkDir(embeddedAssets, "skills", func(filePath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(embeddedAssets, filePath)
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(filePath, "skills/")
		files[path.Clean(rel)] = string(data)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func SortedSkillPaths() ([]string, error) {
	files, err := SkillFiles()
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(files))
	for filePath := range files {
		paths = append(paths, filePath)
	}
	sort.Strings(paths)
	return paths, nil
}
