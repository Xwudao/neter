package core

import (
	"log"
	"os"
	"path/filepath"

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
		log.Println("\n" + res)
		log.Fatalf("npm install error: %v", err)
		return err
	}
	log.Println("\n" + res)

	if res, err = RunWithDir(b.pm, b.frontRoot, nil, "run", "build"); err != nil {
		log.Println("\n" + res)
		log.Fatalf("npm build error: %v", err)
		return err
	}
	log.Println("\n" + res)

	return nil
}

// Copy generated dist/ to ./assets/dist/, will delete assets/dist/ first
func (b *BuildWeb) Copy() error {
	oldAssetsPath := filepath.Join(b.assetsDir, "dist")
	if err := os.RemoveAll(oldAssetsPath); err != nil {
		return err
	}

	webDistPath := filepath.Join(b.frontRoot, "dist")
	//if err := utils.CopyDir(webDistPath, oldAssetsPath); err != nil {
	//	return err
	//}

	if err := os.CopyFS(oldAssetsPath, os.DirFS(webDistPath)); err != nil {
		return err
	}

	return nil
}
