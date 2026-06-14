package utils

import "github.com/Xwudao/neter/internal/core"

func run(name string, args ...string) (string, error) {
	return core.RunWithDir(name, "", nil, args...)
}
