package utils

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

var (
	ErrNotFoundMod = errors.New("not found mod file path")
)

// FindProjectRoot walks up the directory tree from cwd, looking for a go.mod
// file. Returns the directory that contains go.mod, or ErrNotFoundMod if not
// found within maxDepth parent steps.
func FindProjectRoot(maxDepth int) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for i := 0; i <= maxDepth; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Clean(dir), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}
	return "", ErrNotFoundMod
}

func GetModName() (mod string) {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	max := 4
	i := 0
	for i-1 < max {
		i++
		fp := filepath.Join(dir, "go.mod")
		_, err := os.Stat(fp)
		if err != nil {
			dir = filepath.Join(dir, "..")
			continue
		}

		cnt, err := ioutil.ReadFile(fp)
		if err != nil {
			dir = filepath.Join(dir, "..")
			continue
		}

		compile := regexp.MustCompile("(?m)module\\s([^\\s]+)")
		matches := compile.FindStringSubmatch(string(cnt))
		if len(matches) >= 2 {
			mod = matches[1]
			return
		}
		dir = filepath.Join(dir, "..")
	}

	return
}

func FindModPath(deep int) (string, error) {
	var (
		start = 0
		max   = deep
	)
	for start < max {
		start++
		dir, err := os.Getwd()
		if err != nil {
			return "", err
		}
		fp := filepath.Join(dir, "go.mod")
		_, err = os.Stat(fp)
		if err != nil {
			dir = filepath.Join(dir, "..")
			continue
		}
		return dir, nil
	}
	return "", ErrNotFoundMod
}
