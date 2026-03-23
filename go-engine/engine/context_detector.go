package engine

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// WindowInfo chứa thông tin về cửa sổ đang active
type WindowInfo struct {
	AppName     string
	WindowTitle string
	IsTerminal  bool
	PID         int
}

// terminalApps là danh sách tên process của terminal emulators
var terminalApps = map[string]bool{
	"alacritty":       true,
	"kitty":           true,
	"gnome-terminal":  true,
	"gnome-terminal-": true,
	"xterm":           true,
	"xfce4-terminal":  true,
	"konsole":         true,
	"tilix":           true,
	"wt":              true, // Windows Terminal
	"WindowsTerminal": true,
	"iTerm2":          true,
	"Terminal":        true,
	"bash":            true,
	"zsh":             true,
	"fish":            true,
}

// IsTerminalApp kiểm tra xem appName có phải là terminal emulator không
func IsTerminalApp(appName string) bool {
	if appName == "" {
		return false
	}
	return terminalApps[appName]
}

// GetActiveWindow trả về thông tin cửa sổ đang active
// Trên Linux: dùng xprop qua _NET_ACTIVE_WINDOW
// Trên macOS/Windows: trả về WindowInfo rỗng (stub cho v1)
func GetActiveWindow() *WindowInfo {
	switch runtime.GOOS {
	case "linux":
		return getActiveWindowLinux()
	default:
		// Stub cho macOS và Windows — implement ở phase sau
		return &WindowInfo{}
	}
}

// getActiveWindowLinux detect active window trên Linux qua xprop
func getActiveWindowLinux() *WindowInfo {
	info := &WindowInfo{}

	// Lấy window ID đang active
	out, err := exec.Command("xprop", "-root", "_NET_ACTIVE_WINDOW").Output()
	if err != nil {
		// Không có display hoặc xprop không cài — trả về struct rỗng (headless CI)
		return info
	}

	// Parse "window id # 0x123456"
	line := strings.TrimSpace(string(out))
	parts := strings.Fields(line)
	if len(parts) < 5 {
		return info
	}
	windowIDHex := parts[len(parts)-1]

	// Lấy WM_CLASS (app name) và _NET_WM_NAME (window title)
	wmClass, err := exec.Command("xprop", "-id", windowIDHex, "WM_CLASS").Output()
	if err == nil {
		// Format: WM_CLASS(STRING) = "alacritty", "Alacritty"
		s := string(wmClass)
		if idx := strings.Index(s, "= "); idx >= 0 {
			fields := strings.Split(s[idx+2:], ",")
			if len(fields) >= 1 {
				info.AppName = strings.Trim(strings.TrimSpace(fields[0]), `"`)
			}
		}
	}

	wmName, err := exec.Command("xprop", "-id", windowIDHex, "_NET_WM_NAME").Output()
	if err == nil {
		s := string(wmName)
		if idx := strings.Index(s, "= "); idx >= 0 {
			info.WindowTitle = strings.Trim(strings.TrimSpace(s[idx+2:]), `"`)
		}
	}

	// Lấy PID
	pidOut, err := exec.Command("xprop", "-id", windowIDHex, "_NET_WM_PID").Output()
	if err == nil {
		s := string(pidOut)
		if idx := strings.Index(s, "= "); idx >= 0 {
			pidStr := strings.TrimSpace(s[idx+2:])
			if pid, err := strconv.Atoi(pidStr); err == nil {
				info.PID = pid
			}
		}
	}

	info.IsTerminal = IsTerminalApp(info.AppName)
	return info
}
