package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type KeyLockConfig struct {
	Enabled  bool     `json:"enabled"`
	Keys     []string `json:"keys"`
	Duration int      `json:"duration_ms"`
}

type ActionStep struct {
	Key      string `json:"key"`
	HoldMs   int    `json:"hold_ms"`
	DelayMs  int    `json:"delay_ms,omitempty"`
	HoldDown bool   `json:"hold_down,omitempty"` // true = press down, don't release until "release" step
	Release  string `json:"release,omitempty"`    // key to release (from a previous hold_down step)
}

type Action struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Enabled       bool          `json:"enabled"`
	RewardTitle   string        `json:"reward_title"`
	RewardCost    int           `json:"reward_cost"`
	RewardID      string        `json:"reward_id,omitempty"`
	Key           string        `json:"key"`
	HoldMs        int           `json:"hold_ms"`
	Repeat        int           `json:"repeat,omitempty"`
	RepeatDelayMs int           `json:"repeat_delay_ms,omitempty"`
	Steps         []ActionStep  `json:"steps,omitempty"`
	KeyLock       KeyLockConfig `json:"key_lock"`
	Cooldown        int           `json:"cooldown_ms"`
	TwitchCooldown  int           `json:"twitch_cooldown_sec,omitempty"`
	RewardColor     string        `json:"reward_color,omitempty"`
	Category      string        `json:"category"`
	TarkovBind    string        `json:"tarkov_bind,omitempty"`
	Custom        bool          `json:"custom,omitempty"`
}

type TwitchConfig struct {
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret,omitempty"`
	AccessToken   string `json:"access_token,omitempty"`
	RefreshToken  string `json:"refresh_token,omitempty"`
	ChannelName   string `json:"channel_name"`
	BroadcasterID string `json:"broadcaster_id,omitempty"`
}

type Config struct {
	mu           sync.RWMutex
	Twitch       TwitchConfig `json:"twitch"`
	Actions      []Action     `json:"actions"`
	TargetWindow string       `json:"target_window"`
	GlobalEnable bool         `json:"global_enable"`
	Language     string       `json:"language"`
	TarkovPath   string       `json:"tarkov_path,omitempty"`
}

var defaultActions = []Action{
	{ID: "grenade", Name: "Granate werfen", Description: "Stoppt, zieht Granate und wirft sie", Enabled: true, RewardTitle: "Granate werfen!", RewardCost: 500, Key: "g", HoldMs: 200, Steps: []ActionStep{{Key: "g", HoldMs: 200}, {DelayMs: 2500}, {Key: "mouse0", HoldMs: 500}}, KeyLock: KeyLockConfig{Enabled: true, Keys: []string{"w", "a", "s", "d", "space"}, Duration: 5000}, Cooldown: 30000, Category: "combat", TarkovBind: "PressThrowGrenade"},
	{ID: "reload", Name: "Nachladen", Description: "Lädt die Waffe nach", Enabled: true, RewardTitle: "Nachladen!", RewardCost: 200, Key: "r", HoldMs: 100, KeyLock: KeyLockConfig{Enabled: false}, Cooldown: 15000, Category: "combat", TarkovBind: "ReloadWeapon"},
	{ID: "inventory", Name: "Inventar öffnen", Description: "Öffnet das Inventar", Enabled: true, RewardTitle: "Inventar auf!", RewardCost: 300, Key: "tab", HoldMs: 100, KeyLock: KeyLockConfig{Enabled: true, Keys: []string{"w", "a", "s", "d"}, Duration: 3000}, Cooldown: 20000, Category: "movement", TarkovBind: "Inventory"},
	{ID: "prone", Name: "Hinlegen", Description: "Geht in die Bauchlage", Enabled: true, RewardTitle: "Hinlegen!", RewardCost: 150, Key: "x", HoldMs: 100, KeyLock: KeyLockConfig{Enabled: false}, Cooldown: 10000, Category: "movement", TarkovBind: "Prone"},
	{ID: "drop_mag", Name: "Magazin droppen", Description: "Droppt das aktuelle Magazin", Enabled: true, RewardTitle: "Magazin weg!", RewardCost: 800, Key: "alt+r", HoldMs: 100, KeyLock: KeyLockConfig{Enabled: true, Keys: []string{"w", "a", "s", "d"}, Duration: 1500}, Cooldown: 60000, Category: "combat", TarkovBind: "UnloadMagazine"},
	{ID: "heal", Name: "Heilen", Description: "Benutzt Heilitem (Slot 4)", Enabled: true, RewardTitle: "Heilen!", RewardCost: 250, Key: "4", HoldMs: 100, KeyLock: KeyLockConfig{Enabled: false}, Cooldown: 20000, Category: "survival", TarkovBind: "Slot4"},
	{ID: "check_mag", Name: "Magazin checken", Description: "Checkt das Magazin", Enabled: true, RewardTitle: "Mag Check!", RewardCost: 100, Key: "alt+t", HoldMs: 300, KeyLock: KeyLockConfig{Enabled: false}, Cooldown: 10000, Category: "combat", TarkovBind: "CheckAmmo"},
	{ID: "mumble", Name: "Voiceline (Schnell)", Description: "Spielt eine schnelle Voiceline ab", Enabled: true, RewardTitle: "Voiceline!", RewardCost: 50, Key: "f1", HoldMs: 100, KeyLock: KeyLockConfig{Enabled: false}, Cooldown: 5000, Category: "fun"},
	{ID: "mumble_double", Name: "Voiceline (Doppelt)", Description: "Spielt eine andere Voiceline ab (Doppelklick)", Enabled: false, RewardTitle: "Voiceline 2!", RewardCost: 50, Key: "f1", HoldMs: 50, Steps: []ActionStep{{Key: "f1", HoldMs: 50}, {Key: "f1", HoldMs: 50, DelayMs: 100}}, KeyLock: KeyLockConfig{Enabled: false}, Cooldown: 5000, Category: "fun"},
	{ID: "jump", Name: "Springen", Description: "Springt", Enabled: true, RewardTitle: "Spring!", RewardCost: 100, Key: "space", HoldMs: 50, KeyLock: KeyLockConfig{Enabled: false}, Cooldown: 5000, Category: "movement", TarkovBind: "Jump"},
	{ID: "lean_left", Name: "Links lehnen", Description: "Lehnt sich nach links", Enabled: true, RewardTitle: "Links lehnen!", RewardCost: 100, Key: "q", HoldMs: 2000, KeyLock: KeyLockConfig{Enabled: true, Keys: []string{"e"}, Duration: 2000}, Cooldown: 10000, Category: "movement", TarkovBind: "LeanLockLeft"},
	{ID: "lean_right", Name: "Rechts lehnen", Description: "Lehnt sich nach rechts", Enabled: true, RewardTitle: "Rechts lehnen!", RewardCost: 100, Key: "e", HoldMs: 2000, KeyLock: KeyLockConfig{Enabled: true, Keys: []string{"q"}, Duration: 2000}, Cooldown: 10000, Category: "movement", TarkovBind: "LeanLockRight"},
	{ID: "flashlight", Name: "Taschenlampe", Description: "Schaltet die Taschenlampe um", Enabled: true, RewardTitle: "Licht an/aus!", RewardCost: 75, Key: "j", HoldMs: 100, KeyLock: KeyLockConfig{Enabled: false}, Cooldown: 10000, Category: "fun", TarkovBind: "Tactical"},
	// New actions
	{ID: "shoot", Name: "Schuss abfeuern", Description: "Feuert Schüsse ab (Anzahl einstellbar)", Enabled: false, RewardTitle: "Pew Pew!", RewardCost: 1000, Key: "mouse0", HoldMs: 50, Repeat: 3, RepeatDelayMs: 200, KeyLock: KeyLockConfig{Enabled: true, Keys: []string{"w", "a", "s", "d"}, Duration: 2000}, Cooldown: 30000, Category: "combat", TarkovBind: "Shoot"},
	{ID: "fire_mode", Name: "Feuermodus wechseln", Description: "Wechselt zwischen Einzelschuss und Automatik", Enabled: false, RewardTitle: "Feuermodus wechseln!", RewardCost: 300, Key: "b", HoldMs: 100, KeyLock: KeyLockConfig{Enabled: false}, Cooldown: 15000, Category: "combat", TarkovBind: "ShootingMode"},
	{ID: "nightvision", Name: "Nachtsicht", Description: "Schaltet Nachtsichtgerät um", Enabled: false, RewardTitle: "Nachtsicht!", RewardCost: 200, Key: "n", HoldMs: 100, KeyLock: KeyLockConfig{Enabled: false}, Cooldown: 10000, Category: "fun", TarkovBind: "ToggleGoggles"},
	{ID: "drop_backpack", Name: "Rucksack droppen", Description: "Droppt den Rucksack", Enabled: false, RewardTitle: "Rucksack weg!", RewardCost: 2000, Key: "z", HoldMs: 50, Steps: []ActionStep{{Key: "z", HoldMs: 50}, {Key: "z", HoldMs: 50, DelayMs: 150}}, KeyLock: KeyLockConfig{Enabled: true, Keys: []string{"w", "a", "s", "d"}, Duration: 1500}, Cooldown: 120000, Category: "combat", TarkovBind: "DropBackpack"},
	{ID: "duck", Name: "Ducken", Description: "Geht in die Hocke", Enabled: false, RewardTitle: "Duck dich!", RewardCost: 100, Key: "c", HoldMs: 100, KeyLock: KeyLockConfig{Enabled: false}, Cooldown: 8000, Category: "movement", TarkovBind: "Duck"},
	{ID: "headlight", Name: "Helmlampe", Description: "Schaltet die Helmlampe um", Enabled: false, RewardTitle: "Helmlampe!", RewardCost: 75, Key: "h", HoldMs: 100, KeyLock: KeyLockConfig{Enabled: false}, Cooldown: 10000, Category: "fun", TarkovBind: "ToggleHeadLight"},
}

func configPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(configDir, "TarkovTroll")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			_ = cfg.Save()
			return cfg, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), nil
	}

	// Merge new default actions that don't exist yet in saved config
	cfg.mergeNewActions()

	// Save merged config so changes persist
	_ = cfg.Save()

	return &cfg, nil
}

func DefaultConfig() *Config {
	actions := make([]Action, len(defaultActions))
	copy(actions, defaultActions)
	return &Config{
		Actions:      actions,
		TargetWindow: "EscapeFromTarkov",
		GlobalEnable: true,
		Language:     "de",
	}
}

func (c *Config) mergeNewActions() {
	existing := make(map[string]int)
	for i, a := range c.Actions {
		if a.Custom {
			continue // never touch custom actions
		}
		existing[a.ID] = i
	}
	defaultByID := make(map[string]Action)
	for _, da := range defaultActions {
		defaultByID[da.ID] = da
	}

	for _, da := range defaultActions {
		idx, found := existing[da.ID]
		if !found {
			// New action — add it
			c.Actions = append(c.Actions, da)
			continue
		}
		// Existing action — update fields that were added in newer versions
		a := &c.Actions[idx]
		// Always update steps from defaults (complex sequences should stay in sync)
		if len(da.Steps) > 0 {
			a.Steps = da.Steps
		} else if len(da.Steps) == 0 && len(a.Steps) > 0 {
			// Default removed steps — clear them
			a.Steps = nil
		}
		if a.TarkovBind == "" && da.TarkovBind != "" {
			a.TarkovBind = da.TarkovBind
		}
		// Always sync key and hold_ms from defaults
		a.Key = da.Key
		a.HoldMs = da.HoldMs
		// Always sync key lock from defaults
		a.KeyLock = da.KeyLock
		// Sync repeat settings
		a.Repeat = da.Repeat
		a.RepeatDelayMs = da.RepeatDelayMs
	}
}

func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (c *Config) SetAction(action Action) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, a := range c.Actions {
		if a.ID == action.ID {
			c.Actions[i] = action
			return
		}
	}
	c.Actions = append(c.Actions, action)
}

func (c *Config) GetActions() []Action {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]Action, len(c.Actions))
	copy(result, c.Actions)
	return result
}

func (c *Config) ToggleAction(id string, enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, a := range c.Actions {
		if a.ID == id {
			c.Actions[i].Enabled = enabled
			return
		}
	}
}

func (c *Config) DeleteAction(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, a := range c.Actions {
		if a.ID == id {
			c.Actions = append(c.Actions[:i], c.Actions[i+1:]...)
			return true
		}
	}
	return false
}
