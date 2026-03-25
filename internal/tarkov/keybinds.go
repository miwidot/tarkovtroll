package tarkov

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ControlINI struct {
	KeyBindings  []KeyBinding  `json:"keyBindings"`
	AxisBindings []AxisBinding `json:"axisBindings"`
}

type KeyBinding struct {
	KeyName   string    `json:"keyName"`
	Variants  []Variant `json:"variants"`
	PressType string    `json:"pressType"`
}

type AxisBinding struct {
	AxisName string     `json:"axisName"`
	Pairs    []AxisPair `json:"pairs"`
}

type AxisPair struct {
	Positive Variant `json:"positive"`
	Negative Variant `json:"negative"`
}

type Variant struct {
	KeyCode      []string `json:"keyCode"`
	IsAxis       bool     `json:"isAxis,omitempty"`
	AxisName     string   `json:"axisName,omitempty"`
	PositiveAxis bool     `json:"positiveAxis,omitempty"`
}

// Mapping from Tarkov keyCode names to our input.go key names
var tarkovKeyToOurs = map[string]string{
	"LeftAlt":      "alt",
	"RightAlt":     "alt",
	"LeftControl":  "ctrl",
	"RightControl": "ctrl",
	"LeftShift":    "shift",
	"RightShift":   "shift",
	"Space":        "space",
	"Tab":          "tab",
	"Return":       "enter",
	"Escape":       "escape",
	"BackQuote":    "`",
	"CapsLock":     "capslock",
	"Backspace":    "backspace",
	"Delete":       "delete",
	"Insert":       "insert",
	"Home":         "home",
	"End":          "end",
	"PageUp":       "pageup",
	"PageDown":     "pagedown",
	"UpArrow":      "up",
	"DownArrow":    "down",
	"LeftArrow":    "left",
	"RightArrow":   "right",
	"Mouse0":       "mouse0",
	"Mouse1":       "mouse1",
	"Mouse2":       "mouse2",
	"Mouse3":       "mouse3",
	"Mouse4":       "mouse4",
	"Alpha0":       "0",
	"Alpha1":       "1",
	"Alpha2":       "2",
	"Alpha3":       "3",
	"Alpha4":       "4",
	"Alpha5":       "5",
	"Alpha6":       "6",
	"Alpha7":       "7",
	"Alpha8":       "8",
	"Alpha9":       "9",
	"F1":           "f1",
	"F2":           "f2",
	"F3":           "f3",
	"F4":           "f4",
	"F5":           "f5",
	"F6":           "f6",
	"F7":           "f7",
	"F8":           "f8",
	"F9":           "f9",
	"F10":          "f10",
	"F11":          "f11",
	"F12":          "f12",
	"SysReq":       "printscreen",
	"Numlock":      "numlock",
}

// Mapping from our action IDs to Tarkov keyName(s) in control.ini
var ActionToTarkovKey = map[string]string{
	"grenade":        "ThrowGrenade",
	"reload":         "ReloadWeapon",
	"inventory":      "Inventory",
	"prone":          "Prone",
	"drop_mag":       "UnloadMagazine",
	"heal":           "Slot4",
	"check_mag":      "CheckAmmo",
	"mumble":         "MumbleQuick",
	"jump":           "Jump",
	"lean_left":      "LeanLockLeft",
	"lean_right":     "LeanLockRight",
	"flashlight":     "Tactical",
	"headlight":      "ToggleHeadLight",
	"shoot":          "Shoot",
	"fire_mode":      "ShootingMode",
	"nightvision":    "ToggleGoggles",
	"drop_backpack":  "DropBackpack",
	"duck":           "Duck",
	"check_chamber":  "CheckChamber",
}

func DefaultConfigPath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return ""
	}
	return filepath.Join(appData, "Battlestate Games", "Escape from Tarkov", "Settings", "control.ini")
}

func ReadKeybinds(path string) (*ControlINI, error) {
	if path == "" {
		path = DefaultConfigPath()
	}
	if path == "" {
		return nil, fmt.Errorf("Tarkov config path not found")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("control.ini nicht gefunden: %w", err)
	}

	var ctrl ControlINI
	if err := json.Unmarshal(data, &ctrl); err != nil {
		return nil, fmt.Errorf("control.ini parse error: %w", err)
	}
	return &ctrl, nil
}

// ConvertKeyCodes converts Tarkov keyCode array to our key format (e.g. "alt+r")
func ConvertKeyCodes(keyCodes []string) string {
	if len(keyCodes) == 0 {
		return ""
	}

	var modifiers []string
	var mainKey string

	for _, kc := range keyCodes {
		lower := strings.ToLower(kc)
		if ourKey, ok := tarkovKeyToOurs[kc]; ok {
			if ourKey == "alt" || ourKey == "ctrl" || ourKey == "shift" {
				modifiers = append(modifiers, ourKey)
			} else {
				mainKey = ourKey
			}
		} else if len(kc) == 1 {
			// Single letter
			mainKey = lower
		} else {
			// Unknown, try lowercase
			mainKey = lower
		}
	}

	if mainKey == "" && len(modifiers) > 0 {
		mainKey = modifiers[len(modifiers)-1]
		modifiers = modifiers[:len(modifiers)-1]
	}

	if mainKey == "" {
		return ""
	}

	if len(modifiers) > 0 {
		return strings.Join(modifiers, "+") + "+" + mainKey
	}
	return mainKey
}

// GetKeyForAction returns the key string for a given Tarkov action name
func (c *ControlINI) GetKeyForAction(tarkovKeyName string) string {
	for _, kb := range c.KeyBindings {
		if kb.KeyName == tarkovKeyName {
			if len(kb.Variants) > 0 && len(kb.Variants[0].KeyCode) > 0 {
				return ConvertKeyCodes(kb.Variants[0].KeyCode)
			}
		}
	}
	return ""
}

// GetMovementKeys returns the WASD keys from axis bindings
func (c *ControlINI) GetMovementKeys() []string {
	keys := make(map[string]bool)
	for _, ab := range c.AxisBindings {
		if ab.AxisName == "MoveX" || ab.AxisName == "MoveY" {
			for _, pair := range ab.Pairs {
				for _, kc := range pair.Positive.KeyCode {
					if k := convertSingleKey(kc); k != "" {
						keys[k] = true
					}
				}
				for _, kc := range pair.Negative.KeyCode {
					if k := convertSingleKey(kc); k != "" {
						keys[k] = true
					}
				}
			}
		}
	}

	result := make([]string, 0, len(keys))
	for k := range keys {
		result = append(result, k)
	}
	return result
}

func convertSingleKey(tarkovKey string) string {
	if ourKey, ok := tarkovKeyToOurs[tarkovKey]; ok {
		return ourKey
	}
	if len(tarkovKey) == 1 {
		return strings.ToLower(tarkovKey)
	}
	return ""
}
