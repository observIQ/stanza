package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var agentName = "bplogagent"

func defaultPluginDir() string {
	if stat, err := os.Stat("./plugins"); err == nil {
		if stat.IsDir() {
			return "./plugins"
		}
	}

	return filepath.Join(agentHome(), "plugins")
}

func defaultConfig() string {
	if _, err := os.Stat("./config.yaml"); err == nil {
		return "./config.yaml"
	}

	return filepath.Join(agentHome(), "config.yaml")
}

func defaultDatabaseFile() string {
	if _, err := os.Stat("./offsets.db"); err == nil {
		return "./offsets.db"
	}

	return filepath.Join(agentHome(), "offsets.db")
}

func agentHome() string {
	if home := os.Getenv(strings.ToUpper(agentName) + "_HOME"); home != "" {
		return home
	}

	switch runtime.GOOS {
	case "windows":
		return filepath.Join(`C:\`, agentName)
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, agentName)
	case "linux":
		return filepath.Join("/opt", agentName)
	default:
		panic(fmt.Sprintf("Unsupported GOOS %s", runtime.GOOS))
	}
}
