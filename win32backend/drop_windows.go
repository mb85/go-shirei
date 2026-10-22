package win32backend

import (
	"unsafe"

	"golang.org/x/sys/windows"

	"go.hasen.dev/shirei"
)

// File drops from Explorer and other apps arrive as WM_DROPFILES once the
// window has called DragAcceptFiles. That message reports only the drop,
// so Input.FileDragOver stays false here.

const (
	wmDropfiles      = 0x0233
	wmCopydata       = 0x004A
	wmCopyglobaldata = 0x0049
	msgfltAllow      = 1
)

var (
	shell32         = windows.NewLazySystemDLL("shell32.dll")
	procDragAccept  = shell32.NewProc("DragAcceptFiles")
	procDragQuery   = shell32.NewProc("DragQueryFileW")
	procDragPoint   = shell32.NewProc("DragQueryPoint")
	procDragFinish  = shell32.NewProc("DragFinish")
	procAllowWindow = user32.NewProc("ChangeWindowMessageFilterEx")
)

// acceptFileDrops lets files be dropped on the window. An elevated app
// does not receive messages from lower-integrity windows such as Explorer
// unless it lets the drop messages through its message filter.
func acceptFileDrops() {
	procDragAccept.Call(uintptr(hwnd), 1)
	if procAllowWindow.Find() != nil {
		return
	}
	for _, msg := range []uintptr{wmDropfiles, wmCopydata, wmCopyglobaldata} {
		procAllowWindow.Call(uintptr(hwnd), msg, msgfltAllow, 0)
	}
}

// onDropFiles hands the dropped paths to the next frame, with the pointer
// at the drop point, and releases the drop handle.
func onDropFiles(hdrop uintptr) {
	defer procDragFinish.Call(hdrop)
	fi := shirei.GetFrameInput()
	count, _, _ := procDragQuery.Call(hdrop, 0xFFFFFFFF, 0, 0)
	for i := uintptr(0); i < count; i++ {
		n, _, _ := procDragQuery.Call(hdrop, i, 0, 0)
		buf := make([]uint16, n+1)
		procDragQuery.Call(hdrop, i, uintptr(unsafe.Pointer(&buf[0])), n+1)
		fi.DroppedFiles = append(fi.DroppedFiles, windows.UTF16ToString(buf))
	}
	var pt struct{ X, Y int32 }
	procDragPoint.Call(hdrop, uintptr(unsafe.Pointer(&pt)))
	scale := shirei.GetHost().WindowScale
	if scale <= 0 {
		scale = 1
	}
	shirei.GetInputState().MousePoint = shirei.Vec2{float32(pt.X) / scale, float32(pt.Y) / scale}
}
