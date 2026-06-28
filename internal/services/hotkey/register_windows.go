//go:build windows

package hotkey

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Win32 消息 / 错误常量。
const (
	wmHotkey = 0x0312 // WM_HOTKEY
	wmApp    = 0x8000 // WM_APP（用于唤醒消息循环处理命令）

	errorHotkeyAlreadyRegistered syscall.Errno = 1409 // ERROR_HOTKEY_ALREADY_REGISTERED
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procRegisterHotKey     = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey   = user32.NewProc("UnregisterHotKey")
	procGetMessageW        = user32.NewProc("GetMessageW")
	procPeekMessageW       = user32.NewProc("PeekMessageW")
	procPostThreadMessageW = user32.NewProc("PostThreadMessageW")
	procGetCurrentThreadId = kernel32.NewProc("GetCurrentThreadId")
)

// msg 对应 Win32 MSG 结构。
type msg struct {
	Hwnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       struct{ X, Y int32 }
	LPrivate uint32
}

const (
	cmdApply = iota
	cmdStop
)

// winCommand 是投递给消息循环线程执行的命令。
type winCommand struct {
	kind     int
	bindings []resolved
	resultCh chan []registerResult
	doneCh   chan struct{}
}

// winManager 在独立的、锁定到 OS 线程的消息循环里管理 RegisterHotKey。
//
// 为什么要独立线程：RegisterHotKey 把 WM_HOTKEY 投递到"调用线程"的消息队列，
// 因此注册与 GetMessage 取消息必须在同一个 OS 线程上。Go 的 goroutine 会在
// 多个 OS 线程间漂移，所以用 runtime.LockOSThread 钉住。
type winManager struct {
	mu       sync.Mutex
	threadID uint32
	started  bool
	trigger  triggerFunc

	cmdCh   chan winCommand
	readyCh chan struct{}

	// 以下仅在消息循环线程访问，无需加锁。
	idToTarget map[int32]target
	nextID     int32
}

func newManager() manager {
	return &winManager{
		cmdCh:      make(chan winCommand, 8),
		readyCh:    make(chan struct{}),
		idToTarget: make(map[int32]target),
	}
}

func (m *winManager) Supported() bool { return true }

func (m *winManager) Start(trigger triggerFunc) error {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return nil
	}
	m.trigger = trigger
	m.mu.Unlock()

	go m.loop()
	<-m.readyCh // 等待消息队列就绪后再允许 Apply / Stop

	m.mu.Lock()
	m.started = true
	m.mu.Unlock()
	return nil
}

func (m *winManager) Apply(bindings []resolved) []registerResult {
	m.mu.Lock()
	tid := m.threadID
	started := m.started
	m.mu.Unlock()
	if !started || tid == 0 {
		return nil
	}
	resCh := make(chan []registerResult, 1)
	m.cmdCh <- winCommand{kind: cmdApply, bindings: bindings, resultCh: resCh}
	procPostThreadMessageW.Call(uintptr(tid), wmApp, 0, 0)
	return <-resCh
}

func (m *winManager) Stop() {
	m.mu.Lock()
	tid := m.threadID
	started := m.started
	m.started = false
	m.mu.Unlock()
	if !started || tid == 0 {
		return
	}
	done := make(chan struct{})
	m.cmdCh <- winCommand{kind: cmdStop, doneCh: done}
	procPostThreadMessageW.Call(uintptr(tid), wmApp, 0, 0)
	<-done
}

// loop 是钉在单一 OS 线程上的消息循环。
func (m *winManager) loop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// PeekMessage 强制创建该线程的消息队列，之后 PostThreadMessageW 才不会失败。
	var probe msg
	procPeekMessageW.Call(uintptr(unsafe.Pointer(&probe)), 0, 0, 0, 0)

	tid, _, _ := procGetCurrentThreadId.Call()
	m.mu.Lock()
	m.threadID = uint32(tid)
	m.mu.Unlock()
	close(m.readyCh)

	running := true
	for running {
		var mm msg
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&mm)), 0, 0, 0)
		switch int32(ret) {
		case -1:
			// GetMessage 出错，退出循环避免空转。
			running = false
		case 0:
			// WM_QUIT
			running = false
		default:
			switch mm.Message {
			case wmHotkey:
				m.dispatch(int32(mm.WParam))
			case wmApp:
				if !m.drainCommands() {
					running = false
				}
			}
		}
	}
	m.unregisterAll()
}

// drainCommands 处理已投递的命令；收到 stop 返回 false 以结束循环。
func (m *winManager) drainCommands() bool {
	for {
		select {
		case cmd := <-m.cmdCh:
			switch cmd.kind {
			case cmdApply:
				cmd.resultCh <- m.applyLocked(cmd.bindings)
			case cmdStop:
				m.unregisterAll()
				close(cmd.doneCh)
				return false
			}
		default:
			return true
		}
	}
}

// applyLocked 在消息循环线程上全量重注册热键，返回 per-binding 结果。
func (m *winManager) applyLocked(bindings []resolved) []registerResult {
	for id := range m.idToTarget {
		procUnregisterHotKey.Call(0, uintptr(id))
	}
	m.idToTarget = make(map[int32]target, len(bindings))

	results := make([]registerResult, 0, len(bindings))
	for _, b := range bindings {
		m.nextID++
		id := m.nextID
		mods := uintptr(b.modifiers | ModNoRepeat)
		ret, _, callErr := procRegisterHotKey.Call(0, uintptr(id), mods, uintptr(b.vk))
		if ret == 0 {
			code := CodeRegisterFailed
			if errno, ok := callErr.(syscall.Errno); ok && errno == errorHotkeyAlreadyRegistered {
				code = CodeAlreadyRegistered
			}
			results = append(results, registerResult{target: b.target, ok: false, errorCode: code})
			continue
		}
		m.idToTarget[id] = b.target
		results = append(results, registerResult{target: b.target, ok: true})
	}
	return results
}

func (m *winManager) unregisterAll() {
	for id := range m.idToTarget {
		procUnregisterHotKey.Call(0, uintptr(id))
	}
	m.idToTarget = make(map[int32]target)
}

// dispatch 把热键 id 映射回目标并异步回调（不阻塞消息循环）。
func (m *winManager) dispatch(id int32) {
	tgt, ok := m.idToTarget[id]
	if !ok {
		return
	}
	trigger := m.trigger
	if trigger == nil {
		return
	}
	go trigger(tgt)
}
