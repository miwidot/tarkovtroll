package actions

import (
	"fmt"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"bitbotgo/internal/config"
	"bitbotgo/internal/debuglog"
	"bitbotgo/internal/input"
	"bitbotgo/internal/keylock"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetWindowText       = user32.NewProc("GetWindowTextW")
	procGetWindowThreadPID  = user32.NewProc("GetWindowThreadProcessId")
	procOpenProcess         = kernel32.NewProc("OpenProcess")
	procCloseHandle         = kernel32.NewProc("CloseHandle")
	procQueryFullProcessImageName = kernel32.NewProc("QueryFullProcessImageNameW")
)

type Executor struct {
	mu         sync.Mutex
	cfg        *config.Config
	keyLocker  *keylock.KeyLocker
	cooldowns  map[string]time.Time
	onAction   func(actionID, userName string)
	onCooldown func(actionID string, remainingMs int)
}

func NewExecutor(cfg *config.Config, kl *keylock.KeyLocker) *Executor {
	return &Executor{
		cfg:       cfg,
		keyLocker: kl,
		cooldowns: make(map[string]time.Time),
	}
}

func (e *Executor) SetOnAction(fn func(actionID, userName string)) {
	e.onAction = fn
}

func (e *Executor) SetOnCooldown(fn func(actionID string, remainingMs int)) {
	e.onCooldown = fn
}

func (e *Executor) Execute(actionID string, userName string) error {
	if !e.cfg.GlobalEnable {
		return fmt.Errorf("TarkovTroll is disabled")
	}

	actions := e.cfg.GetActions()
	var action *config.Action
	for i, a := range actions {
		if a.ID == actionID {
			action = &actions[i]
			break
		}
	}
	if action == nil {
		return fmt.Errorf("action not found: %s", actionID)
	}
	if !action.Enabled {
		return fmt.Errorf("action disabled: %s", actionID)
	}

	// Check cooldown
	e.mu.Lock()
	if cd, ok := e.cooldowns[actionID]; ok {
		remaining := time.Until(cd)
		if remaining > 0 {
			e.mu.Unlock()
			if e.onCooldown != nil {
				e.onCooldown(actionID, int(remaining.Milliseconds()))
			}
			return fmt.Errorf("action on cooldown: %dms remaining", int(remaining.Milliseconds()))
		}
	}
	e.cooldowns[actionID] = time.Now().Add(time.Duration(action.Cooldown) * time.Millisecond)
	e.mu.Unlock()

	// Check target window (by process exe name + window title)
	if !e.isTargetWindowActive() {
		activeInfo := e.getActiveWindowTitle()
		debuglog.Log("Execute: target window not active. Active: %q, looking for: %q",
			activeInfo, e.cfg.TargetWindow)
		return fmt.Errorf("target window not active (active: %s)", activeInfo)
	}

	if e.onAction != nil {
		e.onAction(actionID, userName)
	}

	go e.executeAction(action)
	return nil
}

func (e *Executor) executeAction(action *config.Action) {
	debuglog.Log("executeAction: %s key=%q steps=%d keylock=%v",
		action.ID, action.Key, len(action.Steps), action.KeyLock.Enabled)

	// Multi-step action
	if len(action.Steps) > 0 {
		heldKeys := make(map[string]func())

		// Apply key lock BEFORE steps — this also sends key-up for locked keys
		// so held movement keys (W/A/S/D) get released in-game
		if action.KeyLock.Enabled && len(action.KeyLock.Keys) > 0 {
			debuglog.Log("executeAction: locking keys %v for %dms (before steps)", action.KeyLock.Keys, action.KeyLock.Duration)
			e.keyLocker.LockKeys(action.KeyLock.Keys, action.KeyLock.Duration)
			time.Sleep(200 * time.Millisecond) // wait for game to register key releases
		}

		for i, step := range action.Steps {
			debuglog.Log("executeAction: step %d key=%q holdMs=%d delayMs=%d holdDown=%v release=%q",
				i, step.Key, step.HoldMs, step.DelayMs, step.HoldDown, step.Release)

			if step.DelayMs > 0 {
				time.Sleep(time.Duration(step.DelayMs) * time.Millisecond)
			}
			if step.Release != "" {
				if relFn, ok := heldKeys[step.Release]; ok {
					relFn()
					delete(heldKeys, step.Release)
				}
				continue
			}
			if step.Key == "" {
				continue
			}
			if step.HoldDown {
				relFn := input.HoldKeyDown(step.Key)
				heldKeys[step.Key] = relFn
				continue
			}
			input.PressKey(step.Key, step.HoldMs)
		}
		for k, relFn := range heldKeys {
			debuglog.Log("executeAction: auto-releasing held key %q", k)
			relFn()
		}
		debuglog.Log("executeAction: %s done (multi-step)", action.ID)
		return
	}

	// Special actions
	if strings.HasPrefix(strings.ToLower(action.Key), "spin360") {
		if action.KeyLock.Enabled && len(action.KeyLock.Keys) > 0 {
			e.keyLocker.LockKeys(action.KeyLock.Keys, action.KeyLock.Duration)
			time.Sleep(50 * time.Millisecond)
		}
		direction := 1 // right
		if strings.Contains(action.Key, "left") {
			direction = -1
		}
		pixels := action.HoldMs // reuse hold_ms as pixel count for spin
		if pixels <= 0 {
			pixels = 8000 // default ~360 degrees
		}
		durationMs := 500
		if action.Cooldown > 0 && action.Cooldown < 2000 {
			durationMs = action.Cooldown
		}
		debuglog.Log("executeAction: %s spin360 pixels=%d dir=%d duration=%dms", action.ID, pixels, direction, durationMs)
		input.Spin360(pixels, durationMs, direction)
		debuglog.Log("executeAction: %s done (spin360)", action.ID)
		return
	}

	// Single-key: lock first then press
	if action.KeyLock.Enabled && len(action.KeyLock.Keys) > 0 {
		debuglog.Log("executeAction: locking keys %v for %dms", action.KeyLock.Keys, action.KeyLock.Duration)
		e.keyLocker.LockKeys(action.KeyLock.Keys, action.KeyLock.Duration)
		time.Sleep(50 * time.Millisecond)
	}

	// Repeat support
	repeatCount := action.Repeat
	if repeatCount <= 0 {
		repeatCount = 1
	}
	repeatDelay := action.RepeatDelayMs
	if repeatDelay <= 0 {
		repeatDelay = 100
	}

	for i := 0; i < repeatCount; i++ {
		if i > 0 {
			time.Sleep(time.Duration(repeatDelay) * time.Millisecond)
		}
		input.PressKey(action.Key, action.HoldMs)
		debuglog.Log("executeAction: %s press %d/%d", action.ID, i+1, repeatCount)
	}
	debuglog.Log("executeAction: %s done (repeat=%d)", action.ID, repeatCount)
}

const processQueryLimitedInformation = 0x1000

func (e *Executor) getForegroundProcessName() string {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return ""
	}

	var pid uint32
	procGetWindowThreadPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return ""
	}

	hProc, _, _ := procOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if hProc == 0 {
		return ""
	}
	defer procCloseHandle.Call(hProc)

	buf := make([]uint16, 1024)
	size := uint32(len(buf))
	ret, _, _ := procQueryFullProcessImageName.Call(hProc, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ret == 0 {
		return ""
	}

	fullPath := syscall.UTF16ToString(buf[:size])
	// Extract just the filename
	parts := strings.Split(fullPath, "\\")
	return parts[len(parts)-1]
}

func (e *Executor) isTargetWindowActive() bool {
	target := e.cfg.TargetWindow
	if target == "" {
		return true
	}

	// Check by process exe name (more reliable than window title)
	procName := e.getForegroundProcessName()
	if procName != "" && strings.Contains(strings.ToLower(procName), strings.ToLower(target)) {
		return true
	}

	// Fallback: check window title
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return false
	}
	buf := make([]uint16, 256)
	procGetWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 256)
	title := syscall.UTF16ToString(buf)

	return strings.Contains(strings.ToLower(title), strings.ToLower(target))
}

func (e *Executor) getActiveWindowTitle() string {
	// Show both process name and window title for debugging
	procName := e.getForegroundProcessName()

	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return procName
	}
	buf := make([]uint16, 256)
	procGetWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 256)
	title := syscall.UTF16ToString(buf)

	if procName != "" {
		return fmt.Sprintf("%s [%s]", title, procName)
	}
	return title
}

func (e *Executor) GetCooldownRemaining(actionID string) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	if cd, ok := e.cooldowns[actionID]; ok {
		remaining := time.Until(cd)
		if remaining > 0 {
			return int(remaining.Milliseconds())
		}
	}
	return 0
}
