//go:build linux

package waylandbackend

import "go.hasen.dev/shirei/internal/wayland/xdgdecoration"

// Server-side decorations via xdg-decoration-unstable-v1 (added in the Flasher
// fork). Where the compositor offers to draw the titlebar and borders (KDE,
// sway, Hyprland, ...), ask it to, and switch the client-side titlebar off once
// it agrees: the window then gets the desktop's own controls, theme and window
// menu. GNOME does not offer the protocol, so it keeps client-side decorations.

var (
	decorationMgr *xdgdecoration.Manager
	decoration    *xdgdecoration.ToplevelDecoration
)

func bindDecorationManager(name, version uint32) {
	if mgr, err := xdgdecoration.BindManager(registry, name, version); err == nil {
		decorationMgr = mgr
	}
}

// requestServerSideDecorations runs once the toplevel exists and before its
// first commit, so the compositor's answer arrives in the first configure
// sequence, before anything is drawn.
func requestServerSideDecorations() {
	if decorationMgr == nil {
		return
	}
	d, err := decorationMgr.GetToplevelDecoration(xdgToplevel)
	if err != nil {
		return
	}
	decoration = d
	d.OnConfigure = setDecorationMode
	d.SetMode(xdgdecoration.ModeServerSide)
}

// setDecorationMode applies the mode the compositor settled on. The content
// keeps its size: drawing our own titlebar makes the surface titlebarHeight
// taller, and handing the titlebar to the compositor takes that back off.
func setDecorationMode(mode uint32) {
	csd := mode != xdgdecoration.ModeServerSide
	if csd == csdEnabled {
		return
	}
	csdEnabled = csd
	if logicalW > 0 && logicalH > 0 {
		if csd {
			logicalH += titlebarHeight
		} else {
			logicalH -= titlebarHeight
		}
		recomputeDeviceSize()
	}
	dirty = true
}
