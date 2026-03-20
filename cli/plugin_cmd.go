package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"nanite/internal/plugin"

	fplugin "github.com/hollis-labs/fragments-engine/plugin"
)

const pluginGitOrg = "hollis-labs"

func cmdPlugin(ctx context.Context, args []string, env *commandEnv) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: nanite plugin <command>")
		fmt.Fprintln(os.Stderr, "commands: list, install, uninstall, disable, enable")
		os.Exit(1)
	}

	pluginsDir := resolveNanitePluginsDir()

	switch args[0] {
	case "list":
		nanitePluginList(pluginsDir)
	case "install":
		if len(args) < 2 {
			die("plugin install: requires <name>")
		}
		nanitePluginInstall(pluginsDir, args[1])
	case "uninstall":
		if len(args) < 2 {
			die("plugin uninstall: requires <name>")
		}
		nanitePluginUninstall(pluginsDir, args[1])
	case "disable":
		if len(args) < 2 {
			die("plugin disable: requires <name>")
		}
		nanitePluginDisable(pluginsDir, args[1])
	case "enable":
		if len(args) < 2 {
			die("plugin enable: requires <name>")
		}
		nanitePluginEnable(pluginsDir, args[1])
	default:
		die("plugin: unknown command %q", args[0])
	}
}

func resolveNanitePluginsDir() string {
	if d := os.Getenv("NANITE_PLUGINS_DIR"); d != "" {
		return d
	}
	return "./plugins"
}

func nanitePluginList(pluginsDir string) {
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No plugins installed.")
			return
		}
		die("plugin list: %v", err)
	}

	found := false
	fmt.Printf("%-25s %-10s %-10s %s\n", "PLUGIN", "VERSION", "STATUS", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 80))

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		status := plugin.PluginStatusString(pluginsDir, name)

		manifestPath := filepath.Join(pluginsDir, name, "plugin.yaml")
		if status == "disabled" {
			manifestPath = filepath.Join(pluginsDir, name, "plugin.yaml.disabled")
		}
		manifest, err := plugin.ParseManifest(manifestPath)
		if err != nil {
			continue
		}

		found = true
		fmt.Printf("%-25s %-10s %-10s %s\n",
			name, manifest.Version, status, manifest.Description)
	}

	if !found {
		fmt.Println("No plugins installed.")
	}
}

func nanitePluginInstall(pluginsDir, name string) {
	target := filepath.Join(pluginsDir, name)

	if _, err := os.Stat(filepath.Join(target, "plugin.yaml")); err == nil {
		die("plugin install: %q is already installed at %s", name, target)
	}

	os.MkdirAll(pluginsDir, 0755)

	repoURL := fmt.Sprintf("git@github.com:%s/%s.git", pluginGitOrg, name)
	fmt.Printf("Installing %s from %s...\n", name, repoURL)

	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		die("plugin install: failed to clone: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, "plugin.yaml")); err != nil {
		os.RemoveAll(target)
		die("plugin install: cloned repo does not contain plugin.yaml")
	}

	manifest, err := plugin.ParseManifest(filepath.Join(target, "plugin.yaml"))
	if err != nil {
		os.RemoveAll(target)
		die("plugin install: failed to parse plugin.yaml: %v", err)
	}

	if _, ok := plugin.LookupConstructor(manifest.Name); !ok {
		fmt.Printf("Warning: no compiled-in code for %q — plugin will need to be added to the binary\n", manifest.Name)
	} else {
		fmt.Printf("Found compiled-in code for %q\n", manifest.Name)
	}

	fmt.Printf("\nPlugin %q installed to %s\n", name, target)
}

func nanitePluginUninstall(pluginsDir, name string) {
	target := filepath.Join(pluginsDir, name)

	manifestPath := filepath.Join(target, "plugin.yaml")
	disabledPath := filepath.Join(target, "plugin.yaml.disabled")
	if _, err := os.Stat(manifestPath); err != nil {
		if _, err2 := os.Stat(disabledPath); err2 != nil {
			die("plugin uninstall: %q is not installed", name)
		}
		manifestPath = disabledPath
	}

	manifest, err := plugin.ParseManifest(manifestPath)
	if err != nil {
		die("plugin uninstall: failed to parse plugin.yaml: %v", err)
	}

	if constructor, ok := plugin.LookupConstructor(manifest.Name); ok {
		p := constructor()
		if uninstallable, ok := p.(fplugin.Uninstallable); ok {
			fmt.Printf("Running %s cleanup...\n", manifest.Name)
			host := plugin.NewHostMinimal()
			if err := uninstallable.Uninstall(host); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: cleanup error: %v\n", err)
			} else {
				fmt.Println("Cleanup complete.")
			}
		}
	}

	if err := os.RemoveAll(target); err != nil {
		die("plugin uninstall: failed to remove %s: %v", target, err)
	}

	fmt.Printf("\nPlugin %q uninstalled.\n", name)
}

func nanitePluginDisable(pluginsDir, name string) {
	if err := plugin.DisablePlugin(pluginsDir, name); err != nil {
		die("plugin disable: %v", err)
	}
	fmt.Printf("Plugin %q disabled.\n", name)
}

func nanitePluginEnable(pluginsDir, name string) {
	if err := plugin.EnablePlugin(pluginsDir, name); err != nil {
		die("plugin enable: %v", err)
	}
	fmt.Printf("Plugin %q enabled.\n", name)
}
