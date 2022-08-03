package filex

import (
	"os"
)

func Exists(f string) bool {
	_, err := os.Stat(f)
	return err == nil
}
