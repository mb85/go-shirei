// Package xdgforeign implements the exporting half of the
// xdg-foreign-unstable-v2 protocol, hand-written in the style of
// cursorshape. Exporting a toplevel gives a handle string that another
// client can use to place its own window over ours, which is how
// xdg-desktop-portal dialogs (file choosers and the like) attach to the app
// window: the portal takes the handle as "wayland:<handle>".
//
// Protocol:
//
//	zxdg_exporter_v2.export_toplevel(new_id exported, wl_surface) -> opcode 1
//	zxdg_exported_v2.handle(string)                                <- event 0
package xdgforeign

import (
	"errors"

	"go.hasen.dev/shirei/internal/wayland/wl"
)

// Interface is the global's advertised interface name in the registry.
const Interface = "zxdg_exporter_v2"

type Exporter struct{ wl.BaseProxy }

func (*Exporter) Dispatch(*wl.Event) {}

// Exported is an exported toplevel. Its handle stays valid for as long as
// the object exists. OnHandle, when set, receives the handle once the
// compositor sends it.
type Exported struct {
	wl.BaseProxy
	OnHandle func(handle string)
}

func (e *Exported) Dispatch(ev *wl.Event) {
	if ev.Opcode == 0 && e.OnHandle != nil {
		e.OnHandle(ev.String())
	}
}

var errNoContext = errors.New("xdgforeign: proxy has no context")

// BindExporter binds zxdg_exporter_v2 from the registry.
func BindExporter(registry *wl.Registry, name, version uint32) (*Exporter, error) {
	ctx := registry.Context()
	if ctx == nil {
		return nil, errNoContext
	}
	ex := &Exporter{}
	ctx.Register(ex)
	if err := registry.Bind(name, Interface, version, ex); err != nil {
		return nil, err
	}
	return ex, nil
}

// ExportToplevel exports surface, which must have the xdg_toplevel role.
func (x *Exporter) ExportToplevel(surface *wl.Surface) (*Exported, error) {
	ctx := x.Context()
	if ctx == nil {
		return nil, errNoContext
	}
	e := &Exported{}
	ctx.Register(e)
	if err := ctx.SendRequest(x, 1, e, surface); err != nil {
		return nil, err
	}
	return e, nil
}
