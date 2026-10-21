// Package xdgdecoration implements the xdg-decoration-unstable-v1 protocol,
// hand-written in the style of cursorshape. It lets a client ask the
// compositor to draw the window's titlebar and borders (server-side
// decorations), so the window gets the desktop's own controls and theme.
// Compositors that never draw decorations, notably GNOME, do not advertise
// it, and the client keeps decorating itself.
//
// Protocol:
//
//	zxdg_decoration_manager_v1.get_toplevel_decoration(new_id, xdg_toplevel) -> opcode 1
//	zxdg_toplevel_decoration_v1.set_mode(mode)                               -> opcode 1
//	zxdg_toplevel_decoration_v1.configure(mode)                              <- event 0
//
// The compositor answers set_mode with configure, as part of the surface's
// configure sequence, and may choose a different mode than requested.
package xdgdecoration

import (
	"errors"

	"go.hasen.dev/shirei/internal/wayland/wl"
)

// Interface is the global's advertised interface name in the registry.
const Interface = "zxdg_decoration_manager_v1"

// Decoration modes.
const (
	ModeClientSide = 1
	ModeServerSide = 2
)

type Manager struct{ wl.BaseProxy }

func (*Manager) Dispatch(*wl.Event) {}

// ToplevelDecoration is a toplevel's decoration object. OnConfigure, when
// set, receives each mode the compositor settles on.
type ToplevelDecoration struct {
	wl.BaseProxy
	OnConfigure func(mode uint32)
}

func (d *ToplevelDecoration) Dispatch(ev *wl.Event) {
	if ev.Opcode == 0 && d.OnConfigure != nil {
		d.OnConfigure(ev.Uint32())
	}
}

var errNoContext = errors.New("xdgdecoration: proxy has no context")

// BindManager binds zxdg_decoration_manager_v1 from the registry.
func BindManager(registry *wl.Registry, name, version uint32) (*Manager, error) {
	ctx := registry.Context()
	if ctx == nil {
		return nil, errNoContext
	}
	mgr := &Manager{}
	ctx.Register(mgr)
	if err := registry.Bind(name, Interface, version, mgr); err != nil {
		return nil, err
	}
	return mgr, nil
}

// GetToplevelDecoration creates the decoration object for toplevel, which
// must be an xdg_toplevel that has not been mapped yet.
func (m *Manager) GetToplevelDecoration(toplevel wl.Proxy) (*ToplevelDecoration, error) {
	ctx := m.Context()
	if ctx == nil {
		return nil, errNoContext
	}
	d := &ToplevelDecoration{}
	ctx.Register(d)
	if err := ctx.SendRequest(m, 1, d, toplevel); err != nil {
		return nil, err
	}
	return d, nil
}

// SetMode asks the compositor for a decoration mode.
func (d *ToplevelDecoration) SetMode(mode uint32) error {
	ctx := d.Context()
	if ctx == nil {
		return errNoContext
	}
	return ctx.SendRequest(d, 1, mode)
}
