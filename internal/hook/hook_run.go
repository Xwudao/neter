package hook

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
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
		  depends:
		    - flags: ["--web"]
		- event: "before_binary"
		  action: "scripts/pre_build.bat"
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
	Event   string       `yaml:"event"`
	Action  string       `yaml:"action"`
	Depends *HookDepends `yaml:"depends,omitempty"`
}

type HookDepends struct {
	Flags []string `yaml:"flags"`
}

type HookManager struct {
	config      HookConfig
	loaded      bool
	activeFlags []string
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

func (h *HookManager) SetActiveFlags(flags []string) {
	h.activeFlags = flags
}

func (h *HookManager) ExecuteHooks(event string) error {
	if !h.loaded || !h.config.App.Enabled {
		return nil
	}

	for _, hook := range h.config.App.Hooks {
		if hook.Event == event {
			// Check dependencies
			if hook.Depends != nil && len(hook.Depends.Flags) > 0 {
				if !h.checkFlagDependencies(hook.Depends.Flags) {
					log.Printf("[hook] skipping %s hook due to unmet flag dependencies: %v", event, hook.Depends.Flags)
					continue
				}
			}

			log.Printf("[hook] executing %s hook: %s", event, hook.Action)
			if err := h.executeCommand(hook.Action); err != nil {
				return fmt.Errorf("failed to execute %s hook: %v", event, err)
			}
		}
	}

	return nil
}

func (h *HookManager) checkFlagDependencies(requiredFlags []string) bool {
	for _, required := range requiredFlags {
		found := slices.Contains(h.activeFlags, required)
		if !found {
			return false
		}
	}
	return true
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

	// Default: execute directly by splitting the action into program + args
	// Special-case platform-specific script runners:
	// - On Windows: .bat/.cmd should be run via cmd /C
	// - On Unix-like: .sh scripts can be executed via sh (so they don't need +x)

	ext := filepath.Ext(parts[0])

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// If it's a batch/cmd file, run via cmd /C <action>
		if ext == ".bat" || ext == ".cmd" {
			// Use the full action string so that arguments are preserved
			cmd = exec.Command("cmd", "/C", action)
		} else {
			cmd = exec.Command(parts[0], parts[1:]...)
		}
	} else {
		// Unix-like systems
		if ext == ".sh" {
			// Run shell script with sh. Use the raw action so args are preserved.
			// This allows running scripts even if they are not executable.
			cmd = exec.Command("sh", "-c", action)
		} else {
			cmd = exec.Command(parts[0], parts[1:]...)
		}
	}

	cmd.Dir, _ = os.Getwd()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
