package output

import "testing"

func TestDarkTheme_Defaults(t *testing.T) {
	th := DarkTheme
	if th.UserColor == "" || th.TimeColor == "" || th.ChannelColor == "" {
		t.Error("dark theme must have all colors set")
	}
}

func TestThemeResolve_PrefersThemeOverEmptyColor(t *testing.T) {
	th := resolveTheme("", "dark")
	if th != DarkTheme {
		t.Error("empty color should resolve to theme dark")
	}
	th = resolveTheme("never", "light")
	if th != LightTheme || th.IsNone() {
		t.Error("color=never should not force theme=None")
	}
	th = resolveTheme("", "minimal")
	if th != MinimalTheme || !th.IsNone() {
		t.Error("minimal theme should have no ANSI colors; IsNone should return true")
	}
}
