//go:build linux || (darwin && x11darwin)

package x11backend

import (
	"github.com/jezek/xgb/xproto"

	"go.hasen.dev/shirei"
	"go.hasen.dev/shirei/internal/urilist"
)

// File drops via XDND (version 5), the drag-and-drop protocol X file
// managers speak. The window advertises XdndAware; the drag source then
// sends client messages: XdndEnter with the offered types, XdndPosition as
// the pointer moves (answered with XdndStatus, accepting only
// text/uri-list), and XdndLeave or XdndDrop. On a drop we convert the
// XdndSelection to text/uri-list onto our own property, read it when the
// SelectionNotify arrives, hand the paths to the next frame, and tell the
// source with XdndFinished.

const xdndVersion = 5

var (
	atomXdndAware, atomXdndEnter, atomXdndPosition, atomXdndStatus   xproto.Atom
	atomXdndLeave, atomXdndDrop, atomXdndFinished, atomXdndSelection xproto.Atom
	atomXdndTypeList, atomXdndActionCopy, atomURIList, atomDropProp  xproto.Atom

	dragSource xproto.Window // source of the drag over the window, or 0
	dragFiles  bool          // the drag offers text/uri-list
	dropSource xproto.Window // source whose drop we are reading, or 0
)

// setupDrops interns the XDND atoms and marks the window as a drop target.
func setupDrops() {
	atoms := map[string]*xproto.Atom{
		"XdndAware": &atomXdndAware, "XdndEnter": &atomXdndEnter,
		"XdndPosition": &atomXdndPosition, "XdndStatus": &atomXdndStatus,
		"XdndLeave": &atomXdndLeave, "XdndDrop": &atomXdndDrop,
		"XdndFinished": &atomXdndFinished, "XdndSelection": &atomXdndSelection,
		"XdndTypeList": &atomXdndTypeList, "XdndActionCopy": &atomXdndActionCopy,
		"text/uri-list": &atomURIList, "SHIREI_DROP": &atomDropProp,
	}
	for name, a := range atoms {
		if *a = internAtom(name); *a == 0 {
			return
		}
	}
	version := []byte{xdndVersion, 0, 0, 0}
	xproto.ChangeProperty(X, xproto.PropModeReplace, win,
		atomXdndAware, xproto.AtomAtom, 32, 1, version)
	shirei.GetHost().FileDrops = true
}

// handleXdnd handles one XDND client message, reporting whether it was one
// and whether a frame should run.
func handleXdnd(e xproto.ClientMessageEvent) (handled, frame bool) {
	if atomXdndAware == 0 || e.Format != 32 {
		return false, false
	}
	d := e.Data.Data32
	source := xproto.Window(d[0])
	switch e.Type {
	case atomXdndEnter:
		dragSource = source
		dragFiles = offersURIList(source, d)
		shirei.GetInputState().FileDragOver = dragFiles
		return true, true

	case atomXdndPosition:
		if source != dragSource {
			return true, false
		}
		// The position is in root coordinates, packed as x<<16 | y.
		x, y := int16(d[2]>>16), int16(d[2]&0xFFFF)
		if r, err := xproto.TranslateCoordinates(X, screen.Root, win, x, y).Reply(); err == nil && r != nil {
			setMouse(float32(r.DstX), float32(r.DstY))
		}
		var accept, action uint32
		if dragFiles {
			accept, action = 1, uint32(atomXdndActionCopy)
		}
		sendXdnd(source, atomXdndStatus, [5]uint32{uint32(win), accept, 0, 0, action})
		return true, true

	case atomXdndLeave:
		if source == dragSource {
			endDrag()
			shirei.GetInputState().MousePoint = shirei.Vec2{-1, -1}
		}
		return true, true

	case atomXdndDrop:
		if source != dragSource {
			return true, false
		}
		files := dragFiles
		endDrag()
		if !files {
			sendXdnd(source, atomXdndFinished, [5]uint32{uint32(win), 0, 0, 0, 0})
			return true, true
		}
		// The pointer stays where the files landed, so the drop frame still
		// sees its target hovered. The paths arrive with SelectionNotify.
		dropSource = source
		xproto.ConvertSelection(X, win, atomXdndSelection, atomURIList, atomDropProp, xproto.Timestamp(d[2]))
		return true, true
	}
	return false, false
}

func endDrag() {
	dragSource, dragFiles = 0, false
	shirei.GetInputState().FileDragOver = false
}

// offersURIList reports whether an XdndEnter offers text/uri-list. Up to
// three types travel in the message; a source with more lists them all in
// its XdndTypeList property.
func offersURIList(source xproto.Window, d []uint32) bool {
	if d[1]&1 == 0 {
		for _, t := range d[2:5] {
			if xproto.Atom(t) == atomURIList {
				return true
			}
		}
		return false
	}
	r, err := xproto.GetProperty(X, false, source, atomXdndTypeList,
		xproto.AtomAtom, 0, 1<<16).Reply()
	if err != nil || r == nil || r.Format != 32 {
		return false
	}
	for i := 0; i+4 <= len(r.Value); i += 4 {
		v := r.Value[i:]
		if xproto.Atom(uint32(v[0])|uint32(v[1])<<8|uint32(v[2])<<16|uint32(v[3])<<24) == atomURIList {
			return true
		}
	}
	return false
}

// handleDropData reads a dropped uri-list off our property, queues its local
// paths for the next frame and tells the source the drop is done. It
// reports whether the event was the drop's and whether a frame should run.
func handleDropData(e xproto.SelectionNotifyEvent) (handled, frame bool) {
	if atomXdndSelection == 0 || e.Selection != atomXdndSelection || dropSource == 0 {
		return false, false
	}
	source := dropSource
	dropSource = 0
	var paths []string
	if e.Property != 0 {
		r, err := xproto.GetProperty(X, true, win, e.Property,
			xproto.GetPropertyTypeAny, 0, 1<<22).Reply()
		if err == nil && r != nil {
			paths = urilist.Paths(string(r.Value))
		}
	}
	var accepted, action uint32
	if len(paths) > 0 {
		accepted, action = 1, uint32(atomXdndActionCopy)
		fi := shirei.GetFrameInput()
		fi.DroppedFiles = append(fi.DroppedFiles, paths...)
	}
	sendXdnd(source, atomXdndFinished, [5]uint32{uint32(win), accepted, action, 0, 0})
	return true, len(paths) > 0
}

func sendXdnd(to xproto.Window, msg xproto.Atom, data [5]uint32) {
	ev := xproto.ClientMessageEvent{
		Format: 32,
		Window: to,
		Type:   msg,
		Data:   xproto.ClientMessageDataUnionData32New(data[:]),
	}
	xproto.SendEvent(X, false, to, xproto.EventMaskNoEvent, string(ev.Bytes()))
}
