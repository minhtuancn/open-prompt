package engine_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/engine"
)

func TestGetActiveWindow_ReturnsStruct(t *testing.T) {
	// Kiểm tra hàm trả về struct hợp lệ (không panic, không nil)
	info := engine.GetActiveWindow()
	// AppName và WindowTitle có thể rỗng trong CI/headless, nhưng struct phải non-nil
	if info == nil {
		t.Fatal("GetActiveWindow() trả về nil, mong đợi *WindowInfo")
	}
}

func TestIsTerminalApp(t *testing.T) {
	cases := []struct {
		name     string
		appName  string
		expected bool
	}{
		{"alacritty là terminal", "alacritty", true},
		{"kitty là terminal", "kitty", true},
		{"gnome-terminal là terminal", "gnome-terminal", true},
		{"wt là terminal", "wt", true},
		{"WindowsTerminal là terminal", "WindowsTerminal", true},
		{"iTerm2 là terminal", "iTerm2", true},
		{"code không phải terminal", "code", false},
		{"chrome không phải terminal", "google-chrome", false},
		{"empty string", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := engine.IsTerminalApp(tc.appName)
			if got != tc.expected {
				t.Errorf("IsTerminalApp(%q) = %v, muốn %v", tc.appName, got, tc.expected)
			}
		})
	}
}

func TestGetActiveWindow_Fields(t *testing.T) {
	info := engine.GetActiveWindow()
	if info == nil {
		t.Fatal("trả về nil")
	}
	// Chỉ kiểm tra fields tồn tại và có type đúng
	_ = info.AppName
	_ = info.WindowTitle
	_ = info.IsTerminal
	_ = info.PID
}
