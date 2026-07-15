package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Xwudao/neter/pkg/filex"
	"github.com/Xwudao/neter/pkg/utils"
)

type BuildWeb struct {
	webDir    string
	assetsDir string

	frontRoot string

	pm string // package manager
}

func NewBuildWeb(pm string) *BuildWeb {
	return &BuildWeb{
		webDir:    "./web/",
		assetsDir: "./assets/",
		pm:        pm,
	}
}

func (b *BuildWeb) Check() error {
	wd, _ := os.Getwd()
	fullPath := filepath.Join(wd, b.webDir)
	err := utils.CheckFolder(fullPath)
	if err != nil {
		return err
	}
	b.frontRoot = fullPath

	return nil
}
func (b *BuildWeb) Build() error {
	var res string
	var err error
	if res, err = RunWithDir(b.pm, b.frontRoot, nil, "install"); err != nil {
		return fmt.Errorf("%s install failed: %s\n%s", b.pm, err, res)
	}

	if res, err = RunWithDir(b.pm, b.frontRoot, nil, "run", "build"); err != nil {
		return fmt.Errorf("%s run build failed: %s\n%s", b.pm, err, res)
	}
	_ = res

	return nil
}

// Copy synchronises the generated web/dist/ to ./assets/dist/.
// Only changed/new files are copied, and stale files in assets/dist/ are removed.
func (b *BuildWeb) Copy() error {
	src := filepath.Join(b.frontRoot, "dist")
	dst := filepath.Join(b.assetsDir, "dist")
	return filex.SyncDir(src, dst)
}
