package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func defaultPluginDir() string {
	if stat, err := os.Stat("./plugins"); err == nil {
		if stat.IsDir() {
			return "./plugins"
		}
	}

	return filepath.Join(bplogagentHome(), "plugins")
}

func defaultConfig() string {
	if _, err := os.Stat("./config.yaml"); err == nil {
		return "./config.yaml"
	}

	return filepath.Join(bplogagentHome(), "config.yaml")
}

func defaultDatabaseFile() string {
	if _, err := os.Stat("./offsets.db"); err == nil {
		return "./offsets.db"
	}

	return filepath.Join(bplogagentHome(), "offsets.db")
}

func bplogagentHome() string {
	switch runtime.GOOS {
	case "windows":
		return `C:\bplogagent`
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "bplogagent")
	case "linux":
		return `/opt/bplogagent`
	default:
		panic(fmt.Sprintf("Unsupported GOOS %s", runtime.GOOS))
	}
}
