package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"

	"github.com/Xwudao/neter/internal/core"
	"github.com/Xwudao/neter/pkg/utils"
)

func resolveAppRoot(dir string, dirChanged bool, action string) (string, error) {
	cmdPaths, err := findCommandDirs("cmd")
	if err != nil {
		return "", err
	}

	switch len(cmdPaths) {
	case 0:
		return "", fmt.Errorf("please run in root project directory")
	case 1:
		for _, appRoot := range cmdPaths {
			return appRoot, nil
		}
	}

	if dirChanged {
		appRoot, ok := cmdPaths[dir]
		if !ok {
			return "", fmt.Errorf("directory %q not found", dir)
		}
		return appRoot, nil
	}

	options := make([]string, 0, len(cmdPaths))
	for name := range cmdPaths {
		options = append(options, name)
	}
	sort.Strings(options)

	prompt := &survey.Select{
		Message:  fmt.Sprintf("Which directory do you want to %s?", action),
		Options:  options,
		PageSize: 10,
	}
	if err := survey.AskOne(prompt, &dir); err != nil || dir == "" {
		return "", nil
	}

	return cmdPaths[dir], nil
}

func findCommandDirs(base string) (map[string]string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	for range 5 {
		cmdPaths, err := scanCommandDirs(filepath.Join(currentDir, base), currentDir)
		if err != nil {
			return nil, err
		}
		if len(cmdPaths) > 0 {
			return cmdPaths, nil
		}
		if _, err := os.Stat(filepath.Join(currentDir, "go.mod")); err == nil {
			break
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}

	return map[string]string{}, nil
}

func scanCommandDirs(dir string, rootDir string) (map[string]string, error) {
	cmdPaths := make(map[string]string)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return cmdPaths, nil
		}
		return nil, err
	}

	err := filepath.Walk(dir, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil {
			return nil
		}

		if !info.IsDir() || filepath.Base(walkPath) != "cmd" {
			return nil
		}

		entries, err := os.ReadDir(walkPath)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			appRoot, err := filepath.Rel(rootDir, filepath.Join(walkPath, entry.Name()))
			if err != nil {
				return err
			}
			appRoot = filepath.ToSlash(appRoot)
			cmdPaths[appRoot] = appRoot
		}
		return nil
	})

	return cmdPaths, err
}

func buildWebAssets(pm string) error {
	log.Println("build with web assets")
	b := core.NewBuildWeb(pm)
	if err := b.Check(); err != nil {
		return err
	}
	if err := b.Build(); err != nil {
		return err
	}
	if err := b.Copy(); err != nil {
		return err
	}

	log.Println("build web assets success")
	return nil
}

func findInternalDataDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	schemaDir := filepath.Join(dir, "internal", "data")
	info, err := os.Stat(schemaDir)
	if err != nil {
		return "", fmt.Errorf("please run in project root")
	}
	if !info.IsDir() {
		return "", fmt.Errorf("internal/data directory not exist")
	}

	return schemaDir, nil
}

func runCommand(name string, args ...string) (string, error) {
	return core.RunWithDir(name, "", nil, args...)
}

func runCommandWithEnv(name string, env []string, args ...string) (string, error) {
	return core.RunWithDir(name, "", env, args...)
}

func checkErr(err error) {
	utils.CheckErrWithStatus(err)
}

func normalizeCommandPath(path string) string {
	return strings.TrimSuffix(filepath.ToSlash(path), "/")
}
