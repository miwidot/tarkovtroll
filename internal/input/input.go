package input

import (
	"strings"
	"syscall"
	"time"
	"unsafe"

	"bitbotgo/internal/debuglog"
)

var (
	user32            = syscall.NewLazyDLL("user32.dll")
	procSendInput     = user32.NewProc("SendInput")
	procMapVirtualKey = user32.NewProc("MapVirtualKeyW")
)

const (
	INPUT_KEYBOARD     = 1
	INPUT_MOUSE_VAL    = 0
	KEYEVENTF_KEYUP    = 0x0002
	KEYEVENTF_SCANCODE = 0x0008

	MOUSEEVENTF_LEFTDOWN   = 0x0002
	MOUSEEVENTF_LEFTUP     = 0x0004
	MOUSEEVENTF_RIGHTDOWN  = 0x0008
	MOUSEEVENTF_RIGHTUP    = 0x0010
	MOUSEEVENTF_MIDDLEDOWN = 0x0020
	MOUSEEVENTF_MIDDLEUP   = 0x0040
	MOUSEEVENTF_XDOWN      = 0x0080
	MOUSEEVENTF_XUP        = 0x0100
	MOUSEEVENTF_MOVE       = 0x0001
	XBUTTON1               = 0x0001
	XBUTTON2               = 0x0002
)

type KEYBDINPUT struct {
	Vk        uint16
	Scan      uint16
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type MOUSEINPUT struct {
	Dx        int32
	Dy        int32
	MouseData uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type INPUT_KB struct {
	Type uint32
	Ki   KEYBDINPUT
	_    [8]byte // pad to 40 bytes (same as Windows INPUT struct on 64-bit)
}

type INPUT_MOUSE struct {
	Type uint32
	Mi   MOUSEINPUT
	// No extra padding needed — already 40 bytes (4 type + 4 alignment + 32 MOUSEINPUT)
}

var keyMap = map[string]uint16{
	"a": 0x41, "b": 0x42, "c": 0x43, "d": 0x44, "e": 0x45, "f": 0x46,
	"g": 0x47, "h": 0x48, "i": 0x49, "j": 0x4A, "k": 0x4B, "l": 0x4C,
	"m": 0x4D, "n": 0x4E, "o": 0x4F, "p": 0x50, "q": 0x51, "r": 0x52,
	"s": 0x53, "t": 0x54, "u": 0x55, "v": 0x56, "w": 0x57, "x": 0x58,
	"y": 0x59, "z": 0x5A,
	"0": 0x30, "1": 0x31, "2": 0x32, "3": 0x33, "4": 0x34,
	"5": 0x35, "6": 0x36, "7": 0x37, "8": 0x38, "9": 0x39,
	"f1": 0x70, "f2": 0x71, "f3": 0x72, "f4": 0x73, "f5": 0x74,
	"f6": 0x75, "f7": 0x76, "f8": 0x77, "f9": 0x78, "f10": 0x79,
	"f11": 0x7A, "f12": 0x7B,
	"space": 0x20, "enter": 0x0D, "tab": 0x09, "escape": 0x1B,
	"shift": 0xA0, "ctrl": 0xA2, "alt": 0xA4,
	"up": 0x26, "down": 0x28, "left": 0x25, "right": 0x27,
	"backspace": 0x08, "delete": 0x2E, "insert": 0x2D,
	"home": 0x24, "end": 0x23, "pageup": 0x21, "pagedown": 0x22,
	"capslock": 0x14, "numlock": 0x90,
	"`": 0xC0, "printscreen": 0x2C,
}

// Mouse button definitions: down flag, up flag, mouseData
type mouseButton struct {
	downFlag uint32
	upFlag   uint32
	xButton  uint32
}

var mouseMap = map[string]mouseButton{
	"mouse0": {MOUSEEVENTF_LEFTDOWN, MOUSEEVENTF_LEFTUP, 0},
	"mouse1": {MOUSEEVENTF_RIGHTDOWN, MOUSEEVENTF_RIGHTUP, 0},
	"mouse2": {MOUSEEVENTF_MIDDLEDOWN, MOUSEEVENTF_MIDDLEUP, 0},
	"mouse3": {MOUSEEVENTF_XDOWN, MOUSEEVENTF_XUP, XBUTTON1},
	"mouse4": {MOUSEEVENTF_XDOWN, MOUSEEVENTF_XUP, XBUTTON2},
	"lmouse": {MOUSEEVENTF_LEFTDOWN, MOUSEEVENTF_LEFTUP, 0},
	"rmouse": {MOUSEEVENTF_RIGHTDOWN, MOUSEEVENTF_RIGHTUP, 0},
	"mmouse": {MOUSEEVENTF_MIDDLEDOWN, MOUSEEVENTF_MIDDLEUP, 0},
}

func isMouseKey(key string) bool {
	_, ok := mouseMap[strings.ToLower(key)]
	return ok
}

func sendMouseDown(btn mouseButton) {
	inp := INPUT_MOUSE{
		Type: INPUT_MOUSE_VAL,
		Mi: MOUSEINPUT{
			Flags:     btn.downFlag,
			MouseData: btn.xButton,
		},
	}
	ret, _, err := procSendInput.Call(1, uintptr(unsafe.Pointer(&inp)), unsafe.Sizeof(inp))
	debuglog.Log("sendMouseDown: flags=0x%X ret=%d size=%d err=%v", btn.downFlag, ret, unsafe.Sizeof(inp), err)
}

func sendMouseUp(btn mouseButton) {
	inp := INPUT_MOUSE{
		Type: INPUT_MOUSE_VAL,
		Mi: MOUSEINPUT{
			Flags:     btn.upFlag,
			MouseData: btn.xButton,
		},
	}
	ret, _, err := procSendInput.Call(1, uintptr(unsafe.Pointer(&inp)), unsafe.Sizeof(inp))
	debuglog.Log("sendMouseUp: flags=0x%X ret=%d size=%d err=%v", btn.upFlag, ret, unsafe.Sizeof(inp), err)
}

// INPUT_MOUSE_TYPE_VAL is the SendInput type value for mouse events

func vkToScanCode(vk uint16) uint16 {
	ret, _, _ := procMapVirtualKey.Call(uintptr(vk), 0)
	return uint16(ret)
}

func sendKeyDown(vk uint16) {
	scan := vkToScanCode(vk)
	input := INPUT_KB{
		Type: INPUT_KEYBOARD,
		Ki: KEYBDINPUT{
			Vk:    vk,
			Scan:  scan,
			Flags: KEYEVENTF_SCANCODE,
		},
	}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&input)), unsafe.Sizeof(input))
}

func sendKeyUp(vk uint16) {
	scan := vkToScanCode(vk)
	input := INPUT_KB{
		Type: INPUT_KEYBOARD,
		Ki: KEYBDINPUT{
			Vk:    vk,
			Scan:  scan,
			Flags: KEYEVENTF_SCANCODE | KEYEVENTF_KEYUP,
		},
	}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&input)), unsafe.Sizeof(input))
}

func ResolveKey(key string) (uint16, bool) {
	vk, ok := keyMap[strings.ToLower(strings.TrimSpace(key))]
	return vk, ok
}

// SendKeyUpVK sends a key-up event for a virtual key code.
// Used by KeyLocker to release keys that may be physically held down.
// Sends both scancode-based and VK-based key-up for maximum compatibility.
func SendKeyUpVK(vk uint16) {
	// Scancode-based key-up (works with most games)
	sendKeyUp(vk)
	// Also send VK-only key-up (some games only check VK)
	inp := INPUT_KB{
		Type: INPUT_KEYBOARD,
		Ki: KEYBDINPUT{
			Vk:    vk,
			Flags: KEYEVENTF_KEYUP,
		},
	}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&inp)), unsafe.Sizeof(inp))
	debuglog.Log("SendKeyUpVK: sent dual key-up for vk=0x%X", vk)
}

func IsMouseButton(key string) bool {
	return isMouseKey(key)
}

// SendMouseMove sends a relative mouse move event.
func SendMouseMove(dx, dy int32) {
	inp := INPUT_MOUSE{
		Type: INPUT_MOUSE_VAL,
		Mi: MOUSEINPUT{
			Dx:    dx,
			Dy:    dy,
			Flags: MOUSEEVENTF_MOVE,
		},
	}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&inp)), unsafe.Sizeof(inp))
}

// Spin360 performs a 360-degree horizontal spin by moving the mouse in steps.
func Spin360(totalPixels int, durationMs int, direction int) {
	steps := 60
	stepDelay := time.Duration(durationMs/steps) * time.Millisecond
	pixelsPerStep := int32((totalPixels / steps) * direction)

	for i := 0; i < steps; i++ {
		SendMouseMove(pixelsPerStep, 0)
		time.Sleep(stepDelay)
	}
}

// HoldKeyDown presses a key/mouse down and returns a release function.
func HoldKeyDown(keySpec string) func() {
	key := strings.ToLower(strings.TrimSpace(keySpec))
	if btn, ok := mouseMap[key]; ok {
		sendMouseDown(btn)
		return func() { sendMouseUp(btn) }
	}
	if vk, ok := ResolveKey(key); ok {
		sendKeyDown(vk)
		return func() { sendKeyUp(vk) }
	}
	return func() {}
}

// PressKey simulates pressing a key or mouse button for a given duration.
// keySpec can be "g", "alt+t" for key combos, or "mouse0" for mouse buttons.
func PressKey(keySpec string, holdMs int) {
	parts := strings.Split(strings.ToLower(keySpec), "+")

	// Press modifiers first
	var modifiers []uint16
	for _, p := range parts[:max(0, len(parts)-1)] {
		if vk, ok := ResolveKey(p); ok {
			modifiers = append(modifiers, vk)
			sendKeyDown(vk)
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Press main key (could be keyboard or mouse)
	mainKey := parts[len(parts)-1]
	if btn, ok := mouseMap[mainKey]; ok {
		sendMouseDown(btn)
		time.Sleep(time.Duration(holdMs) * time.Millisecond)
		sendMouseUp(btn)
	} else if vk, ok := ResolveKey(mainKey); ok {
		sendKeyDown(vk)
		time.Sleep(time.Duration(holdMs) * time.Millisecond)
		sendKeyUp(vk)
	}

	// Release modifiers in reverse
	for i := len(modifiers) - 1; i >= 0; i-- {
		sendKeyUp(modifiers[i])
		time.Sleep(10 * time.Millisecond)
	}
}
