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

	"github.com/Xwudao/neter/internal/core"
	"gopkg.in/yaml.v3"
)

/*
Primary config now lives in neter.yml:

hooks:
	enabled: true
	items:
		- event: "on_start"
		  action: "scripts/pre_build.sh"
		  depends:
		    flags: ["--web"]

Legacy hook_run.yml is still supported for migration:

app:
	enabled: true
	hooks:
		- event: "on_start"
		  action: "scripts/pre_build.sh"
*/

type HookConfig struct {
	App AppConfig `yaml:"app"`
}

type AppConfig struct {
	Enabled bool       `yaml:"enabled"`
	Hooks   []HookItem `yaml:"hooks"`
}

type HookItem struct {
	Event     string       `yaml:"event"`
	Action    string       `yaml:"action"`
	Platforms []string     `yaml:"platforms,omitempty"`
	Depends   *HookDepends `yaml:"depends,omitempty"`
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
	neterCfg, err := core.LoadOptionalNeterConfig()
	if err != nil {
		return fmt.Errorf("load neter.yml: %w", err)
	}

	if neterCfg != nil && hasHookItems(neterCfg.Hooks) {
		h.config = hookConfigFromNeter(neterCfg.Hooks)
		h.loaded = true
		if legacyPath, ok := findLegacyHookConfig(); ok {
			log.Printf("[hook] detected legacy %s; please move hook config into neter.yml -> hooks", legacyPath)
		}
		return nil
	}

	legacyPath, ok := findLegacyHookConfig()
	if !ok {
		log.Println("[hook] hooks not configured in neter.yml, skipping hooks")
		return nil
	}

	log.Printf("[hook] detected legacy %s; please move hook config into neter.yml -> hooks", legacyPath)
	if err := h.loadLegacyConfig(legacyPath); err != nil {
		return err
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
			if !shouldRunOnCurrentPlatform(hook) {
				log.Printf("[hook] skipping %s hook on %s due to platform restriction or incompatible script: %s", event, runtime.GOOS, hook.Action)
				continue
			}

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

func shouldRunOnCurrentPlatform(hook HookItem) bool {
	return shouldRunOnPlatform(runtime.GOOS, hook.Platforms, hook.Action)
}

func shouldRunOnPlatform(goos string, platforms []string, action string) bool {
	// Check explicit platform restriction
	if len(platforms) > 0 {
		return matchPlatforms(goos, platforms)
	}

	// Fall back to extension-based check
	command := strings.TrimSpace(action)
	if command == "" {
		return false
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	ext := strings.ToLower(filepath.Ext(parts[0]))
	switch goos {
	case "windows":
		return ext != ".sh"
	default:
		return ext != ".bat" && ext != ".cmd"
	}
}

// matchPlatforms checks whether the current OS matches any of the configured platforms.
// Supported platform values: "linux", "macos" (or "mac", maps to darwin), "windows".
func matchPlatforms(goos string, platforms []string) bool {
	for _, p := range platforms {
		switch strings.ToLower(p) {
		case "linux":
			if goos == "linux" {
				return true
			}
		case "macos", "mac":
			if goos == "darwin" {
				return true
			}
		case "windows":
			if goos == "windows" {
				return true
			}
		}
	}
	return false
}

func hasHookItems(cfg core.HooksConfig) bool {
	return cfg.Enabled || len(cfg.Items) > 0
}

func hookConfigFromNeter(cfg core.HooksConfig) HookConfig {
	items := make([]HookItem, 0, len(cfg.Items))
	for _, item := range cfg.Items {
		var depends *HookDepends
		if item.Depends != nil {
			depends = &HookDepends{Flags: append([]string(nil), item.Depends.Flags...)}
		}
		items = append(items, HookItem{
			Event:     item.Event,
			Action:    item.Action,
			Platforms: item.Platforms,
			Depends:   depends,
		})
	}

	return HookConfig{
		App: AppConfig{
			Enabled: cfg.Enabled,
			Hooks:   items,
		},
	}
}

func findLegacyHookConfig() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}

	for range 10 {
		p := filepath.Join(dir, "hook_run.yml")
		if _, statErr := os.Stat(p); statErr == nil {
			return p, true
		}
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return "", false
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", false
}

func (h *HookManager) loadLegacyConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read hook_run.yml: %v", err)
	}

	if err := yaml.Unmarshal(data, &h.config); err != nil {
		return fmt.Errorf("failed to parse hook_run.yml: %v", err)
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
