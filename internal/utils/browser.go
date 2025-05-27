package utils

import (
	"log/slog"
	"os"
	"os/exec"
	"runtime"
)

func OpenBrowserURL(url string) {
	operatingSystem := runtime.GOOS
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		operatingSystem = "wsl"
	}

	slog.Info("If browser doesn't open automatically, visit this URL", "url", url)
	switch operatingSystem {
	case "linux":
		_ = exec.Command("xdg-open", url).Start()
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "wsl":
		_ = exec.Command("wslview", url).Start()
	case "darwin":
		_ = exec.Command("open", url).Start()
	}
}
