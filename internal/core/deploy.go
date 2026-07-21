package core

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var deployRunStreamWithDir = RunStreamWithDir

func (c DeployConfig) Validate() error {
	if strings.TrimSpace(c.Alias) == "" {
		return fmt.Errorf("deploy.alias is required")
	}
	if strings.TrimSpace(c.RemoteUploadDir) == "" {
		return fmt.Errorf("deploy.remote_upload_dir is required")
	}
	if strings.TrimSpace(c.RemoteScript) == "" {
		return fmt.Errorf("deploy.remote_script is required")
	}
	return nil
}

func (c DeployConfig) RemoteBinaryPath(localPath string) string {
	return path.Join(strings.TrimSpace(c.RemoteUploadDir), filepath.Base(localPath))
}

func DeployBinary(cfg DeployConfig, localPath string) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(localPath) == "" {
		return fmt.Errorf("local deploy artifact path is required")
	}
	if _, err := os.Stat(localPath); err != nil {
		return fmt.Errorf("stat local deploy artifact: %w", err)
	}

	remotePath := cfg.RemoteBinaryPath(localPath)
	target := fmt.Sprintf("%s:%s", strings.TrimSpace(cfg.Alias), remotePath)

	if err := deployRunStreamWithDir("scp", "", nil, localPath, target); err != nil {
		return fmt.Errorf("scp deploy artifact: %w", err)
	}

	if err := deployRunStreamWithDir("ssh", "", nil, strings.TrimSpace(cfg.Alias), "sh", strings.TrimSpace(cfg.RemoteScript), remotePath); err != nil {
		return fmt.Errorf("run remote deploy script: %w", err)
	}

	return nil
}
