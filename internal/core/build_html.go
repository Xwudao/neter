package core

import (
	"os"
	"path/filepath"
)

type BuildHtml struct {
	rootDir string

	buildDir  string
	frontDir  string
	staticDir string

	appName []string
}

func NewBuildHtml(rootDir string, appName []string) *BuildHtml {
	return &BuildHtml{
		rootDir:   rootDir,
		appName:   appName,
		buildDir:  filepath.Join(rootDir, "build"),
		staticDir: filepath.Join(rootDir, "web", "static"),
		frontDir:  filepath.Join(rootDir, "web", "front"),
	}
}

func (b *BuildHtml) Check() error {
	// clear buildDir
	if err := os.RemoveAll(b.buildDir); err != nil {
		return err
	}

	// if buildDir not exist create it
	if err := os.MkdirAll(b.buildDir, 0644); err != nil {
		return err
	}

	return nil
}

func (b *BuildHtml) Copy() error {

	var aimDirs = []string{"front", "static", "public"}

	// copy aimDirs to build/
	for _, aim := range aimDirs {
		var srcDir = filepath.Join(b.rootDir, "web", aim)
		var destDir = filepath.Join(b.buildDir, "web", aim)
		if err := os.CopyFS(destDir, os.DirFS(srcDir)); err != nil {
			return err
		}
	}

	//// copy web/front to build/web
	//if err := os.CopyFS(filepath.Join(finalDir, "front"), os.DirFS(b.frontDir)); err != nil {
	//	return err
	//}
	//
	//// copy web/static to build/web
	//if err := os.CopyFS(filepath.Join(finalDir, "static"), os.DirFS(b.staticDir)); err != nil {
	//	return err
	//}

	// copy appName to build/
	for _, app := range b.appName {
		appDir := filepath.Join(b.rootDir, app)
		if err := os.Rename(appDir, filepath.Join(b.buildDir, app)); err != nil {
			return err
		}
	}

	return nil
}
