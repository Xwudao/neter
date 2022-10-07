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
