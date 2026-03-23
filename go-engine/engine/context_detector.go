package engine

import (
	"os"
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
	"wt":              true,
	"WindowsTerminal": true,
	"iTerm2":          true,
	"Terminal":        true,
	"wezterm":         true,
	"foot":            true,
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
		// Thử Wayland trước (nếu WAYLAND_DISPLAY có set)
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			if info := getActiveWindowWayland(); info.AppName != "" {
				return info
			}
		}
		return getActiveWindowLinux()
	default:
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

// getActiveWindowWayland detect active window trên Wayland qua swaymsg hoặc hyprctl
func getActiveWindowWayland() *WindowInfo {
	info := &WindowInfo{}

	// Thử swaymsg (Sway compositor)
	if out, err := exec.Command("swaymsg", "-t", "get_tree").Output(); err == nil {
		s := string(out)
		// Parse focused window từ JSON tree — tìm "focused": true
		if idx := strings.Index(s, `"focused":true`); idx >= 0 {
			// Tìm "app_id" gần nhất trước focused
			before := s[:idx]
			if appIdx := strings.LastIndex(before, `"app_id":"`); appIdx >= 0 {
				start := appIdx + len(`"app_id":"`)
				end := strings.Index(before[start:], `"`)
				if end > 0 {
					info.AppName = before[start : start+end]
				}
			}
			// Tìm "name" gần nhất trước focused
			if nameIdx := strings.LastIndex(before, `"name":"`); nameIdx >= 0 {
				start := nameIdx + len(`"name":"`)
				end := strings.Index(before[start:], `"`)
				if end > 0 {
					info.WindowTitle = before[start : start+end]
				}
			}
		}
		info.IsTerminal = IsTerminalApp(info.AppName)
		return info
	}

	// Thử hyprctl (Hyprland compositor)
	if out, err := exec.Command("hyprctl", "activewindow", "-j").Output(); err == nil {
		s := string(out)
		// Parse "class": "appname" từ JSON
		if classIdx := strings.Index(s, `"class":"`); classIdx >= 0 {
			start := classIdx + len(`"class":"`)
			end := strings.Index(s[start:], `"`)
			if end > 0 {
				info.AppName = s[start : start+end]
			}
		}
		if titleIdx := strings.Index(s, `"title":"`); titleIdx >= 0 {
			start := titleIdx + len(`"title":"`)
			end := strings.Index(s[start:], `"`)
			if end > 0 {
				info.WindowTitle = s[start : start+end]
			}
		}
		info.IsTerminal = IsTerminalApp(info.AppName)
		return info
	}

	return info
}
