package hook

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

/*
hook_run.yml

app:
	enabled: true
	hooks:
		- event: "on_start"
		  action: "xxx cmd to run, eg:　scripts/updatexx.bat"
		- event: "on_stop"
		  action: "stop_app"

*/

type HookConfig struct {
	App AppConfig `yaml:"app"`
}

type AppConfig struct {
	Enabled bool       `yaml:"enabled"`
	Hooks   []HookItem `yaml:"hooks"`
}

type HookItem struct {
	Event  string `yaml:"event"`
	Action string `yaml:"action"`
}

type HookManager struct {
	config HookConfig
	loaded bool
}

func NewHookManager() *HookManager {
	return &HookManager{}
}

func (h *HookManager) LoadConfig() error {
	configPath := "hook_run.yml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Println("[hook] hook_run.yml not found, skipping hooks")
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read hook_run.yml: %v", err)
	}

	if err := yaml.Unmarshal(data, &h.config); err != nil {
		return fmt.Errorf("failed to parse hook_run.yml: %v", err)
	}

	h.loaded = true
	return nil
}

func (h *HookManager) ExecuteHooks(event string) error {
	if !h.loaded || !h.config.App.Enabled {
		return nil
	}

	for _, hook := range h.config.App.Hooks {
		if hook.Event == event {
			log.Printf("[hook] executing %s hook: %s", event, hook.Action)
			if err := h.executeCommand(hook.Action); err != nil {
				return fmt.Errorf("failed to execute %s hook: %v", event, err)
			}
		}
	}

	return nil
}

func (h *HookManager) executeCommand(action string) error {
	if action == "" {
		return nil
	}

	// Split command and arguments
	parts := strings.Fields(action)
	if len(parts) == 0 {
		return nil
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir, _ = os.Getwd()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Handle different file extensions for Windows
	if filepath.Ext(parts[0]) == ".bat" || filepath.Ext(parts[0]) == ".cmd" {
		cmd = exec.Command("cmd", append([]string{"/C"}, parts...)...)
		cmd.Dir, _ = os.Getwd()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}
