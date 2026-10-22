package win32backend

import (
	"unsafe"

	"golang.org/x/sys/windows"

	"go.hasen.dev/shirei/widgets"
)

// DWMWA_USE_IMMERSIVE_DARK_MODE is 20 from Windows 10 20H1 on; earlier
// Windows 10 builds took the same flag as attribute 19.
const (
	dwmwaUseImmersiveDarkMode       = 20
	dwmwaUseImmersiveDarkModeBefore = 19
)

var (
	dwmapi                    = windows.NewLazySystemDLL("dwmapi.dll")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")

	titleBarDark, titleBarSynced bool
)

// syncTitleBarTheme gives the window a dark title bar while the widgets use
// the dark scheme, so the frame Windows draws matches the content. It only
// calls into DWM when the scheme changes.
func syncTitleBarTheme() {
	dark := widgets.IsDarkMode()
	if hwnd == 0 || (titleBarSynced && dark == titleBarDark) {
		return
	}
	titleBarDark, titleBarSynced = dark, true
	if procDwmSetWindowAttribute.Find() != nil {
		return
	}
	var v int32
	if dark {
		v = 1
	}
	for _, attr := range []uintptr{dwmwaUseImmersiveDarkMode, dwmwaUseImmersiveDarkModeBefore} {
		r, _, _ := procDwmSetWindowAttribute.Call(uintptr(hwnd), attr, uintptr(unsafe.Pointer(&v)), unsafe.Sizeof(v))
		if r == 0 {
			return
		}
	}
}
