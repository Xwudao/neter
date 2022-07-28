package utils

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func SaveToFile(p string, cnt []byte, cover bool) (err error) {
	if string(cnt) == "" {
		return errors.New("write file: empty content")
	}
	_, err = os.Stat(p)
	if err == nil {
		if !cover {
			return fmt.Errorf("file [%s] existed, please rename or remove it", p)
		}
	}
	err = ioutil.WriteFile(p, cnt, os.ModePerm)
	if err != nil {
		return
	}
	return nil
}
func RemoveExt(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(filename)

	return strings.Replace(base, ext, "", 1)
}
