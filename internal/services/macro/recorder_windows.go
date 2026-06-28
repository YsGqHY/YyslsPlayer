//go:build windows && completion

package macro

import (
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Low-level hook constants.
const (
	whKeyboardLL = 13
	whMouseLL    = 14
	hcAction     = 0
	llkhfUp      = 0x80

	wmKeyDown    = 0x0100
	wmKeyUp      = 0x0101
	wmSysKeyDown = 0x0104
	wmSysKeyUp   = 0x0105

	wmLButtonDown = 0x0201
	wmLButtonUp   = 0x0202
	wmRButtonDown = 0x0204
	wmRButtonUp   = 0x0205
	wmMButtonDown = 0x0207
	wmMButtonUp   = 0x0208
	wmXButtonDown = 0x020B
	wmXButtonUp   = 0x020C
	wmMouseWheel  = 0x020A
	wmMouseHWheel = 0x020E
	wmMouseMove   = 0x0200

	xButton1Hi = 0x0001
	xButton2Hi = 0x0002

	wmQuitRecorder = 0x0012 // WM_QUIT
)

var (
	recUser32                  = windows.NewLazySystemDLL("user32.dll")
	recKernel32                = windows.NewLazySystemDLL("kernel32.dll")
	recProcSetWindowsHookExW   = recUser32.NewProc("SetWindowsHookExW")
	recProcUnhookWindowsHookEx = recUser32.NewProc("UnhookWindowsHookEx")
	recProcCallNextHookEx      = recUser32.NewProc("CallNextHookEx")
	recProcGetMessageW         = recUser32.NewProc("GetMessageW")
	recProcPeekMessageW        = recUser32.NewProc("PeekMessageW")
	recProcPostThreadMessageW  = recUser32.NewProc("PostThreadMessageW")
	recProcGetModuleHandleW    = recKernel32.NewProc("GetModuleHandleW")
	recProcGetCurrentThreadId  = recKernel32.NewProc("GetCurrentThreadId")
)

// recKbd mirrors KBDLLHOOKSTRUCT.
type recKbd struct {
	vkCode      uint32
	scanCode    uint32
	flags       uint32
	time        uint32
	dwExtraInfo uintptr
}

// recMouse mirrors MSLLHOOKSTRUCT.
type recMouse struct {
	pt          struct{ x, y int32 }
	mouseData   uint32
	flags       uint32
	time        uint32
	dwExtraInfo uintptr
}

type recMsg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

// windowsRecorder installs WH_KEYBOARD_LL + WH_MOUSE_LL on a dedicated thread.
type windowsRecorder struct {
	mu       sync.Mutex
	threadID uint32
	kbHook   uintptr
	msHook   uintptr
	onEvent  func(capturedEvent)
	start    time.Time
	ready    chan struct{}
	stopped  chan struct{}
	downKeys map[uint32]struct{} // vks currently held, to suppress auto-repeat
	// lastPt tracks the previous cursor position so WM_MOUSEMOVE can be turned
	// into relative deltas. hasPt guards the first sample (no delta yet).
	lastX  int32
	lastY  int32
	hasPt  bool
}

func newKeyRecorder() keyRecorder {
	return &windowsRecorder{downKeys: make(map[uint32]struct{})}
}

// activeRecorder is the single in-flight recorder the C callbacks dispatch to.
// Low-level hooks are global and the callbacks are package functions, so we
// route through this pointer guarded by a mutex.
var (
	activeRecorderMu sync.Mutex
	activeRecorder   *windowsRecorder
	recKbCallback    uintptr
	recMsCallback    uintptr
)

func (r *windowsRecorder) Start(onEvent func(capturedEvent)) error {
	r.mu.Lock()
	r.onEvent = onEvent
	r.start = time.Now()
	r.ready = make(chan struct{})
	r.stopped = make(chan struct{})
	r.mu.Unlock()

	activeRecorderMu.Lock()
	if activeRecorder != nil {
		activeRecorderMu.Unlock()
		return fmt.Errorf("recorder already active")
	}
	activeRecorder = r
	if recKbCallback == 0 {
		recKbCallback = windows.NewCallback(recordKeyboardProc)
		recMsCallback = windows.NewCallback(recordMouseProc)
	}
	activeRecorderMu.Unlock()

	go r.pump()
	<-r.ready

	r.mu.Lock()
	ok := r.kbHook != 0
	r.mu.Unlock()
	if !ok {
		activeRecorderMu.Lock()
		activeRecorder = nil
		activeRecorderMu.Unlock()
		return fmt.Errorf("failed to install recording hooks")
	}
	return nil
}

func (r *windowsRecorder) Stop() {
	r.mu.Lock()
	threadID := r.threadID
	stopped := r.stopped
	r.mu.Unlock()
	if threadID != 0 {
		recProcPostThreadMessageW.Call(uintptr(threadID), uintptr(wmQuitRecorder), 0, 0)
	}
	if stopped != nil {
		<-stopped
	}
	activeRecorderMu.Lock()
	if activeRecorder == r {
		activeRecorder = nil
	}
	activeRecorderMu.Unlock()
}

func (r *windowsRecorder) pump() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(r.stopped)

	threadID, _, _ := recProcGetCurrentThreadId.Call()
	r.mu.Lock()
	r.threadID = uint32(threadID)
	r.mu.Unlock()

	var probe recMsg
	recProcPeekMessageW.Call(uintptr(unsafe.Pointer(&probe)), 0, 0, 0, 0)

	hInst, _, _ := recProcGetModuleHandleW.Call(0)
	kbHook, _, _ := recProcSetWindowsHookExW.Call(uintptr(whKeyboardLL), recKbCallback, hInst, 0)
	msHook, _, _ := recProcSetWindowsHookExW.Call(uintptr(whMouseLL), recMsCallback, hInst, 0)

	r.mu.Lock()
	r.kbHook = kbHook
	r.msHook = msHook
	r.mu.Unlock()
	close(r.ready)

	if kbHook == 0 {
		if msHook != 0 {
			recProcUnhookWindowsHookEx.Call(msHook)
		}
		return
	}
	defer func() {
		r.mu.Lock()
		kb, ms := r.kbHook, r.msHook
		r.kbHook, r.msHook = 0, 0
		r.mu.Unlock()
		if kb != 0 {
			recProcUnhookWindowsHookEx.Call(kb)
		}
		if ms != 0 {
			recProcUnhookWindowsHookEx.Call(ms)
		}
	}()

	var m recMsg
	for {
		ret, _, _ := recProcGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if ret == 0 || ret == ^uintptr(0) {
			return
		}
	}
}

func (r *windowsRecorder) emit(ev capturedEvent) {
	r.mu.Lock()
	fn := r.onEvent
	start := r.start
	r.mu.Unlock()
	if fn == nil {
		return
	}
	ev.atMs = time.Since(start).Milliseconds()
	fn(ev)
}

func recordKeyboardProc(nCode int32, wParam, lParam uintptr) uintptr {
	if nCode == hcAction {
		activeRecorderMu.Lock()
		r := activeRecorder
		activeRecorderMu.Unlock()
		if r != nil {
			kbd := (*recKbd)(unsafe.Pointer(lParam))
			// Ignore synthetic events produced by our own keysim driver.
			if kbd.dwExtraInfo != yyslsRecorderExtraInfo {
				up := wParam == wmKeyUp || wParam == wmSysKeyUp || kbd.flags&llkhfUp != 0
				if r.acceptKey(kbd.vkCode, up) {
					r.emit(capturedEvent{vk: int(kbd.vkCode), scan: int(kbd.scanCode), keyUp: up})
				}
			}
		}
	}
	ret, _, _ := recProcCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}

// acceptKey suppresses OS auto-repeat: a held key emits repeated WM_KEYDOWN
// messages, but a macro only wants one keyDown per physical press until the
// matching keyUp. Returns false for repeat-downs and for stray ups whose down
// was never recorded.
func (r *windowsRecorder) acceptKey(vk uint32, up bool) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.downKeys == nil {
		r.downKeys = make(map[uint32]struct{})
	}
	if up {
		if _, ok := r.downKeys[vk]; !ok {
			return false
		}
		delete(r.downKeys, vk)
		return true
	}
	if _, ok := r.downKeys[vk]; ok {
		return false // auto-repeat
	}
	r.downKeys[vk] = struct{}{}
	return true
}

func recordMouseProc(nCode int32, wParam, lParam uintptr) uintptr {
	if nCode == hcAction {
		activeRecorderMu.Lock()
		r := activeRecorder
		activeRecorderMu.Unlock()
		if r != nil {
			ms := (*recMouse)(unsafe.Pointer(lParam))
			// Skip synthetic events produced by our own keysim driver.
			if ms.dwExtraInfo == yyslsRecorderExtraInfo {
				ret, _, _ := recProcCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
				return ret
			}
			if uint32(wParam) == wmMouseMove {
				if dx, dy, ok := r.moveDelta(ms.pt.x, ms.pt.y); ok {
					r.emit(capturedEvent{mouse: true, move: true, dx: dx, dy: dy})
				}
			} else if dir, ok := wheelDirFromMsg(uint32(wParam), ms.mouseData); ok {
				// Scroll notches are one-shot events with no release.
				r.emit(capturedEvent{mouse: true, wheel: true, button: dir})
			} else if button, up, ok := mouseButtonFromMsg(uint32(wParam), ms.mouseData); ok {
				r.emit(capturedEvent{mouse: true, button: button, keyUp: up})
			}
		}
	}
	ret, _, _ := recProcCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}

// moveDelta converts an absolute cursor position from a WM_MOUSEMOVE hook into a
// relative delta against the previous sample. The first sample establishes the
// baseline and reports no delta.
func (r *windowsRecorder) moveDelta(x, y int32) (dx, dy int, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.hasPt {
		r.hasPt = true
		r.lastX, r.lastY = x, y
		return 0, 0, false
	}
	dx = int(x - r.lastX)
	dy = int(y - r.lastY)
	r.lastX, r.lastY = x, y
	if dx == 0 && dy == 0 {
		return 0, 0, false
	}
	return dx, dy, true
}

func mouseButtonFromMsg(message, mouseData uint32) (button int, up bool, ok bool) {
	switch message {
	case wmLButtonDown:
		return mouseButtonLeft, false, true
	case wmLButtonUp:
		return mouseButtonLeft, true, true
	case wmRButtonDown:
		return mouseButtonRight, false, true
	case wmRButtonUp:
		return mouseButtonRight, true, true
	case wmMButtonDown:
		return mouseButtonMiddle, false, true
	case wmMButtonUp:
		return mouseButtonMiddle, true, true
	case wmXButtonDown, wmXButtonUp:
		hi := (mouseData >> 16) & 0xFFFF
		btn := mouseButtonX1
		if hi == xButton2Hi {
			btn = mouseButtonX2
		}
		return btn, message == wmXButtonUp, true
	default:
		return 0, false, false
	}
}

// wheelDirFromMsg maps a wheel message + mouseData delta to a wheel direction
// id. The high word of mouseData is a signed WHEEL_DELTA multiple: positive
// scrolls up (vertical) or right (horizontal), negative the opposite.
func wheelDirFromMsg(message, mouseData uint32) (dir int, ok bool) {
	delta := int16(uint16(mouseData >> 16))
	switch message {
	case wmMouseWheel:
		if delta >= 0 {
			return mouseWheelUp, true
		}
		return mouseWheelDown, true
	case wmMouseHWheel:
		if delta >= 0 {
			return mouseWheelRight, true
		}
		return mouseWheelLeft, true
	default:
		return 0, false
	}
}

// yyslsRecorderExtraInfo matches the keysim driver ExtraInfo tag so the recorder
// can skip synthetic playback events. Kept in sync with keysim.yyslsExtraInfo.
const yyslsRecorderExtraInfo = 0x5959534C
