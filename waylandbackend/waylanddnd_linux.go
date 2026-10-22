//go:build linux

package waylandbackend

import (
	"io"
	"os"
	"time"

	"go.hasen.dev/shirei"
	"go.hasen.dev/shirei/internal/urilist"
	"go.hasen.dev/shirei/internal/wayland/wl"
	"go.hasen.dev/shirei/internal/wayland/wlclient"
)

// File drops via wl_data_device.
//
// Every data_offer (drag or selection) announces its mime types right after
// data_device.data_offer introduces it, so each offer gets a small listener
// that notes whether text/uri-list is among them. On enter we accept the
// drag only if it carries a uri-list; motion moves the pointer so hover
// follows the drag; on drop we read the list over a pipe, like paste, and
// hand the local paths to the next frame as FrameInput.DroppedFiles.

const uriListMime = "text/uri-list"

var (
	offers    = map[*wl.DataOffer]*offerMimes{}
	dragOffer *wl.DataOffer // the offer under the current drag, until drop or leave
)

// offerMimes listens to one data_offer's mime announcements.
type offerMimes struct{ uriList bool }

func (o *offerMimes) HandleDataOfferOffer(ev wl.DataOfferOfferEvent) {
	if ev.MimeType == uriListMime {
		o.uriList = true
	}
}
func (*offerMimes) HandleDataOfferSourceActions(wl.DataOfferSourceActionsEvent) {}
func (*offerMimes) HandleDataOfferAction(wl.DataOfferActionEvent)               {}

func destroyOffer(o *wl.DataOffer) {
	delete(offers, o)
	o.Destroy()
}

func (*handler) HandleDataDeviceDataOffer(ev wl.DataDeviceDataOfferEvent) {
	if ev.Id == nil {
		return
	}
	m := new(offerMimes)
	offers[ev.Id] = m
	wlclient.DataOfferAddListener(ev.Id, m)
}

func (*handler) HandleDataDeviceEnter(ev wl.DataDeviceEnterEvent) {
	dragOffer = ev.Id
	if ev.Id == nil {
		return // a drag from this client, which never offers files
	}
	if m := offers[ev.Id]; m != nil && m.uriList {
		ev.Id.Accept(ev.Serial, uriListMime)
		if dataDevVersion >= 3 {
			ev.Id.SetActions(wl.DataDeviceManagerDndActionCopy, wl.DataDeviceManagerDndActionCopy)
		}
		shirei.GetInputState().FileDragOver = true
	} else {
		ev.Id.Accept(ev.Serial, "") // nothing we can take
	}
	setMouse(ev.X, ev.Y)
	dirty = true
}

func (*handler) HandleDataDeviceMotion(ev wl.DataDeviceMotionEvent) {
	setMouse(ev.X, ev.Y)
	dirty = true
}

// HandleDataDeviceLeave ends a drag that left the window or was cancelled.
// After a drop the offer is already gone and the pointer stays where the
// files landed, so the drop frame still sees the target as hovered.
func (*handler) HandleDataDeviceLeave(wl.DataDeviceLeaveEvent) {
	shirei.GetInputState().FileDragOver = false
	if dragOffer != nil {
		destroyOffer(dragOffer)
		dragOffer = nil
		shirei.GetInputState().MousePoint = shirei.Vec2{-1, -1}
	}
	dirty = true
}

func (*handler) HandleDataDeviceDrop(wl.DataDeviceDropEvent) {
	o := dragOffer
	dragOffer = nil
	shirei.GetInputState().FileDragOver = false
	dirty = true
	if o == nil {
		return
	}
	defer destroyOffer(o)
	if m := offers[o]; m == nil || !m.uriList {
		return
	}
	r, w, err := os.Pipe()
	if err != nil {
		return
	}
	o.Receive(uriListMime, w.Fd())
	w.Close()
	// The source is another process, so a synchronous read is safe; the
	// deadline guards against one that never writes.
	r.SetReadDeadline(time.Now().Add(time.Second))
	data, _ := io.ReadAll(r)
	r.Close()
	if dataDevVersion >= 3 {
		o.Finish()
	}
	fi := shirei.GetFrameInput()
	fi.DroppedFiles = append(fi.DroppedFiles, urilist.Paths(string(data))...)
}
