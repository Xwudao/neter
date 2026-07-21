package core

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDeployBinaryRunsScpThenSsh(t *testing.T) {
	dir := t.TempDir()
	localPath := filepath.Join(dir, "app-linux")
	if err := os.WriteFile(localPath, []byte("binary"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg := DeployConfig{
		Alias:           "prod-app",
		RemoteUploadDir: "/srv/myapp",
		RemoteScript:    "/srv/myapp/deploy.sh",
	}

	var calls [][]string
	oldRunner := deployRunStreamWithDir
	deployRunStreamWithDir = func(name string, dir string, env []string, args ...string) error {
		call := append([]string{name}, args...)
		calls = append(calls, call)
		return nil
	}
	defer func() {
		deployRunStreamWithDir = oldRunner
	}()

	if err := DeployBinary(cfg, localPath); err != nil {
		t.Fatalf("DeployBinary() error = %v", err)
	}

	want := [][]string{
		{"scp", localPath, "prod-app:/srv/myapp/app-linux"},
		{"ssh", "prod-app", "sh", "/srv/myapp/deploy.sh", "/srv/myapp/app-linux"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("DeployBinary() calls = %#v, want %#v", calls, want)
	}
}

func TestDeployBinaryRequiresLocalArtifact(t *testing.T) {
	cfg := DeployConfig{
		Alias:           "prod-app",
		RemoteUploadDir: "/srv/myapp",
		RemoteScript:    "/srv/myapp/deploy.sh",
	}

	if err := DeployBinary(cfg, filepath.Join(t.TempDir(), "missing-app-linux")); err == nil {
		t.Fatal("DeployBinary() error = nil, want error")
	}
}
