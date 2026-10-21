//go:build linux

package waylandbackend

import "go.hasen.dev/shirei/internal/wayland/xdgforeign"

// Exporting the main window with xdg-foreign-unstable-v2 (added in the Flasher
// fork), so portal dialogs can attach to it. The window is exported once,
// right after it gets its toplevel role; the compositor's handle arrives a
// dispatch or two later and stays valid for the life of the window.

var (
	exporter       *xdgforeign.Exporter
	exported       *xdgforeign.Exported
	exportedHandle string
)

func bindExporter(name, version uint32) {
	if ex, err := xdgforeign.BindExporter(registry, name, version); err == nil {
		exporter = ex
	}
}

func exportToplevel() {
	if exporter == nil || surface == nil {
		return
	}
	e, err := exporter.ExportToplevel(surface)
	if err != nil {
		return
	}
	exported = e
	e.OnHandle = func(h string) { exportedHandle = h }
}
