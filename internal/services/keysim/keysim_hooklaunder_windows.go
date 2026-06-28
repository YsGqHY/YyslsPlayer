//go:build windows

package keysim

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// WH_KEYBOARD_LL hook laundering constants.
const (
	whKeyboardLL         = 13 // WH_KEYBOARD_LL
	hcAction             = 0  // HC_ACTION
	llkhfExtended        = 0x01
	llkhfInjected        = 0x10
	llkhfLowerILInjected = 0x02
	llkhfUp              = 0x80
	wmHookCommand        = 0x8000 + 0x59
	pmNoRemove           = 0
)

const (
	hookRefreshInterval = time.Second
	hookRefreshTimeout  = 500 * time.Millisecond
)

// kbdllhookstruct mirrors the Windows KBDLLHOOKSTRUCT (x64 size = 24 bytes).
type kbdllhookstruct struct {
	vkCode      uint32
	scanCode    uint32
	flags       uint32
	time        uint32
	dwExtraInfo uintptr
}

// hook API bindings (separate LazyDLL to avoid init-order dependency with keysim_windows.go).
var (
	kernel32Hook            = windows.NewLazySystemDLL("kernel32.dll")
	procGetModuleHandleW    = kernel32Hook.NewProc("GetModuleHandleW")
	procGetCurrentThreadId  = kernel32Hook.NewProc("GetCurrentThreadId")
	user32Hook              = windows.NewLazySystemDLL("user32.dll")
	procSetWindowsHookExW   = user32Hook.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx = user32Hook.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx      = user32Hook.NewProc("CallNextHookEx")
	procGetMessageW         = user32Hook.NewProc("GetMessageW")
	procPeekMessageW        = user32Hook.NewProc("PeekMessageW")
	procPostThreadMessageW  = user32Hook.NewProc("PostThreadMessageW")
	procTranslateMessage    = user32Hook.NewProc("TranslateMessage")
	procDispatchMessageW    = user32Hook.NewProc("DispatchMessageW")
)

// msg mirrors the Windows MSG struct for the GetMessageW pump.
type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

type hookCommandKind int

const (
	hookCommandRefresh hookCommandKind = iota + 1
)

type hookCommand struct {
	kind hookCommandKind
	done chan error
}

type pendingHookEvent struct {
	vkCode   uint32
	scanCode uint32
	keyUp    bool
	expires  time.Time
}

type pendingMatchResult struct {
	matched        bool
	before         int
	after          int
	scanned        int
	expired        int
	mismatchReason string
}

// launderCallbackRef holds a reference to the Go callback to prevent GC.
var launderCallbackRef uintptr

var pendingHookEvents = struct {
	sync.Mutex
	events []pendingHookEvent
}{events: make([]pendingHookEvent, 0, 64)}

// hookLaunderDriver implements Driver by pairing SendInput with a
// WH_KEYBOARD_LL hook that strips LLKHF_INJECTED before downstream
// hooks (including the game's) see the event.
type hookLaunderDriver struct {
	mu          sync.Mutex
	hHook       uintptr
	threadID    uint32
	lastRefresh time.Time
	ready       chan struct{}
	commands    chan hookCommand
}

// NewHookLaunderDriver creates a hook laundering driver. Returns nil
// if the hook could not be installed (e.g. on a system without a
// working message pump thread).
func NewHookLaunderDriver() Driver {
	d := &hookLaunderDriver{
		ready:    make(chan struct{}),
		commands: make(chan hookCommand, 4),
	}
	go d.messagePump()
	<-d.ready
	d.mu.Lock()
	hHook := d.hHook
	d.mu.Unlock()
	if hHook == 0 {
		return nil
	}
	return d
}

func (d *hookLaunderDriver) messagePump() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	threadID, _, _ := procGetCurrentThreadId.Call()
	d.mu.Lock()
	d.threadID = uint32(threadID)
	d.mu.Unlock()
	// Create this thread's message queue before other goroutines post commands.
	var queueProbe msg
	procPeekMessageW.Call(uintptr(unsafe.Pointer(&queueProbe)), 0, 0, 0, uintptr(pmNoRemove))

	hInst, _, _ := procGetModuleHandleW.Call(0) // GetModuleHandleW(NULL)
	if hInst == 0 {
		close(d.ready)
		return
	}

	cb := windows.NewCallback(launderProc)
	launderCallbackRef = cb

	hHook, _ := installLaunderHook(hInst, cb)

	d.mu.Lock()
	d.hHook = hHook
	d.lastRefresh = time.Now()
	d.mu.Unlock()
	close(d.ready)

	if hHook == 0 {
		return
	}

	defer func() {
		if hHook != 0 {
			procUnhookWindowsHookEx.Call(hHook)
		}
		d.mu.Lock()
		d.hHook = 0
		d.mu.Unlock()
	}()

	// Message loop blocks the OS thread until a message arrives.
	var m msg
	for {
		ret, _, _ := procGetMessageW.Call(
			uintptr(unsafe.Pointer(&m)),
			0, 0, 0,
		)
		if ret == 0 || ret == ^uintptr(0) { // WM_QUIT (0) or error (-1)
			return
		}
		if m.message == wmHookCommand {
			d.handleHookCommands(hInst, cb, &hHook)
			continue
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func installLaunderHook(hInst, cb uintptr) (uintptr, error) {
	hHook, _, callErr := procSetWindowsHookExW.Call(
		uintptr(whKeyboardLL),
		cb,
		uintptr(hInst),
		0, // global hook
	)
	if hHook == 0 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return 0, callErr
		}
		return 0, fmt.Errorf("SetWindowsHookExW returned 0")
	}
	return hHook, nil
}

func (d *hookLaunderDriver) handleHookCommands(hInst, cb uintptr, hHook *uintptr) {
	for {
		select {
		case cmd := <-d.commands:
			var err error
			switch cmd.kind {
			case hookCommandRefresh:
				err = d.refreshHookOnThread(hInst, cb, hHook)
			default:
				err = fmt.Errorf("unknown hook command: %d", cmd.kind)
			}
			cmd.done <- err
		default:
			return
		}
	}
}

func (d *hookLaunderDriver) refreshHookOnThread(hInst, cb uintptr, hHook *uintptr) error {
	if *hHook != 0 {
		procUnhookWindowsHookEx.Call(*hHook)
		*hHook = 0
	}
	nextHook, err := installLaunderHook(hInst, cb)
	d.mu.Lock()
	d.hHook = nextHook
	d.lastRefresh = time.Now()
	d.mu.Unlock()
	*hHook = nextHook
	if err != nil {
		return fmt.Errorf("refresh hook launder chain position: %w", err)
	}
	return nil
}

func rememberPendingHookEvent(ki keyboardInput, kind PhysicalKind) int {
	now := time.Now()
	entry := pendingHookEvent{
		vkCode:   uint32(ki.VirtualKey),
		scanCode: uint32(ki.ScanCode),
		keyUp:    kind == PhysicalUp,
		expires:  now.Add(2 * time.Second),
	}

	pendingHookEvents.Lock()
	defer pendingHookEvents.Unlock()
	write := pendingHookEvents.events[:0]
	for _, item := range pendingHookEvents.events {
		if item.expires.After(now) {
			write = append(write, item)
		}
	}
	pendingHookEvents.events = write
	if len(pendingHookEvents.events) >= 128 {
		copy(pendingHookEvents.events, pendingHookEvents.events[len(pendingHookEvents.events)-64:])
		pendingHookEvents.events = pendingHookEvents.events[:64]
	}
	pendingHookEvents.events = append(pendingHookEvents.events, entry)
	return len(pendingHookEvents.events)
}

func consumePendingHookEvent(kbd *kbdllhookstruct) pendingMatchResult {
	now := time.Now()
	keyUp := kbd.flags&llkhfUp != 0

	pendingHookEvents.Lock()
	defer pendingHookEvents.Unlock()
	result := pendingMatchResult{before: len(pendingHookEvents.events)}
	write := pendingHookEvents.events[:0]
	for _, item := range pendingHookEvents.events {
		if !item.expires.After(now) {
			result.expired++
			continue
		}
		result.scanned++
		if !result.matched {
			switch {
			case item.keyUp != keyUp:
				result.mismatchReason = "kind"
			case item.scanCode != kbd.scanCode:
				result.mismatchReason = "scan"
			case item.vkCode != 0 && item.vkCode != kbd.vkCode:
				result.mismatchReason = "vk"
			default:
				result.matched = true
				result.mismatchReason = ""
				continue
			}
		}
		write = append(write, item)
	}
	pendingHookEvents.events = write
	result.after = len(pendingHookEvents.events)
	if !result.matched && result.mismatchReason == "" {
		if result.before == 0 {
			result.mismatchReason = "empty"
		} else if result.scanned == 0 {
			result.mismatchReason = "expired"
		} else {
			result.mismatchReason = "unknown"
		}
	}
	return result
}

func (d *hookLaunderDriver) RefreshChainHead(_ context.Context) error {
	return d.refreshChainHead(true)
}

func (d *hookLaunderDriver) ensureChainHead() error {
	return d.refreshChainHead(false)
}

func (d *hookLaunderDriver) refreshChainHead(force bool) error {
	d.mu.Lock()
	hHook := d.hHook
	threadID := d.threadID
	lastRefresh := d.lastRefresh
	d.mu.Unlock()

	if hHook == 0 || threadID == 0 {
		return fmt.Errorf("hook launder driver is not active")
	}
	if !force && time.Since(lastRefresh) < hookRefreshInterval {
		return nil
	}

	done := make(chan error, 1)
	cmd := hookCommand{kind: hookCommandRefresh, done: done}
	select {
	case d.commands <- cmd:
	default:
		return fmt.Errorf("hook launder command queue is full")
	}

	ret, _, callErr := procPostThreadMessageW.Call(uintptr(threadID), uintptr(wmHookCommand), 0, 0)
	if ret == 0 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return fmt.Errorf("wake hook thread: %w", callErr)
		}
		return fmt.Errorf("wake hook thread: PostThreadMessageW returned 0")
	}

	select {
	case err := <-done:
		return err
	case <-time.After(hookRefreshTimeout):
		return fmt.Errorf("refresh hook launder chain position timed out")
	}
}

// Send dispatches a key event via SendInput. The WH_KEYBOARD_LL hook
// automatically strips LLKHF_INJECTED from events carrying yyslsExtraInfo.
func (d *hookLaunderDriver) Send(ctx context.Context, event KeyEvent, opts RunOptions) error {
	if opts.DryRun {
		return nil
	}
	if event.Key.IsMouse() {
		return sendMouseEventCtx(ctx, event)
	}
	ki, err := keyboardInputFromEvent(event)
	if err != nil {
		return err
	}
	if err := d.ensureChainHead(); err != nil {
		return err
	}
	rememberPendingHookEvent(ki, event.Kind)
	in := input{Type: inputKeyboard, Ki: ki}
	ret, _, callErr := procSendInput.Call(
		uintptr(1),
		uintptr(unsafe.Pointer(&in)),
		unsafe.Sizeof(in),
	)
	if ret != 1 {
		errMsg := fmt.Sprintf("sent=%d", ret)
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			errMsg = callErr.Error()
		}
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return fmt.Errorf("%w: %v", ErrSendFailed, callErr)
		}
		return fmt.Errorf("%w: %s", ErrSendFailed, errMsg)
	}
	return nil
}

// MoveMouse moves the cursor by a relative (dx, dy) pixel offset via SendInput.
// Mouse movement does not depend on the keyboard laundering hook.
func (d *hookLaunderDriver) MoveMouse(_ context.Context, dx, dy int, opts RunOptions) error {
	if opts.DryRun {
		return nil
	}
	return sendMouseMove(dx, dy)
}

// SendText injects a Unicode string via SendInput using KEYEVENTF_UNICODE.
// Each rune is encoded to UTF-16 and emitted as keydown/keyup pairs carrying
// yyslsExtraInfo so the launder hook strips the injected flag. Surrogate pairs
// (runes outside the BMP) are sent as two consecutive code units.
func (d *hookLaunderDriver) SendText(ctx context.Context, text string, opts RunOptions) error {
	if opts.DryRun || text == "" {
		return nil
	}
	if err := d.ensureChainHead(); err != nil {
		return err
	}
	delay := time.Duration(opts.InterKeyDelayMs) * time.Millisecond
	units := utf16.Encode([]rune(text))
	for i, unit := range units {
		if err := ctx.Err(); err != nil {
			return err
		}
		for _, up := range [2]bool{false, true} {
			ki := unicodeInput(unit, up)
			in := input{Type: inputKeyboard, Ki: ki}
			ret, _, callErr := procSendInput.Call(
				uintptr(1),
				uintptr(unsafe.Pointer(&in)),
				unsafe.Sizeof(in),
			)
			if ret != 1 {
				if callErr != nil && callErr != windows.ERROR_SUCCESS {
					return fmt.Errorf("%w: %v", ErrSendFailed, callErr)
				}
				return fmt.Errorf("%w: sent=%d", ErrSendFailed, ret)
			}
		}
		if delay > 0 && i < len(units)-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return nil
}

// launderProc is the WH_KEYBOARD_LL callback. It strips LLKHF_INJECTED
// from keyboard events carrying yyslsExtraInfo, making them appear as
// genuine hardware input to downstream hooks.
func launderProc(nCode int32, wParam uintptr, lParam uintptr) uintptr {
	if nCode < 0 {
		ret, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
		return ret
	}
	if nCode == hcAction {
		kbd := (*kbdllhookstruct)(unsafe.Pointer(lParam))
		matched := kbd.dwExtraInfo == yyslsExtraInfo
		// ExtraInfo may be stripped by some paths before this hook sees it;
		// pending matching keeps laundering tied to events sent by this driver.
		pending := consumePendingHookEvent(kbd)
		if !matched && pending.matched {
			matched = true
		}
		if matched {
			kbd.flags &^= llkhfInjected | llkhfLowerILInjected
			kbd.dwExtraInfo = 0
		}
	}
	ret, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}
