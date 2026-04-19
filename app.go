package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"bitbotgo/internal/debuglog"
	"bitbotgo/internal/actions"
	"bitbotgo/internal/config"
	"bitbotgo/internal/keylock"
	"bitbotgo/internal/tarkov"
	"bitbotgo/internal/twitch"
)

type App struct {
	ctx       context.Context
	cfg       *config.Config
	twClient  *twitch.Client
	executor  *actions.Executor
	keyLocker *keylock.KeyLocker
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}
	a.cfg = cfg

	a.keyLocker = keylock.New(cfg.TargetWindow)
	a.keyLocker.Start()
	a.keyLocker.SetOnKeyBlocked(func(key string) {
		runtime.EventsEmit(a.ctx, "key-blocked", key)
	})

	a.executor = actions.NewExecutor(cfg, a.keyLocker)
	a.executor.SetOnAction(func(actionID, userName string) {
		runtime.EventsEmit(a.ctx, "action-executed", map[string]string{
			"action": actionID,
			"user":   userName,
		})
	})

	// Auto-import Tarkov keybinds if control.ini exists
	go a.autoImportKeybinds()

	if cfg.Twitch.AccessToken != "" {
		go a.autoConnect()
	}
}

func (a *App) autoImportKeybinds() {
	path := a.cfg.TarkovPath
	if path == "" {
		path = tarkov.DefaultConfigPath()
	}
	if path == "" {
		return
	}
	if _, err := os.Stat(path); err != nil {
		debuglog.Log("autoImportKeybinds: control.ini nicht gefunden (%s)", path)
		return
	}
	n, err := a.ImportTarkovKeybinds()
	if err != nil {
		debuglog.Log("autoImportKeybinds: error: %s", err)
		return
	}
	if n > 0 {
		debuglog.Log("autoImportKeybinds: %d Keybinds automatisch importiert", n)
		runtime.EventsEmit(a.ctx, "twitch-log", fmt.Sprintf("%d Tarkov-Keybinds automatisch importiert", n))
	}
}

func (a *App) shutdown(ctx context.Context) {
	debuglog.Log("=== TarkovTroll beendet ===")
	if a.twClient != nil {
		a.twClient.Disconnect()
	}
	a.keyLocker.Stop()
	a.cfg.Save()
	debuglog.Close()
}

// --- Config Methods ---

func (a *App) GetConfig() *config.Config {
	return a.cfg
}

func (a *App) GetActions() []config.Action {
	return a.cfg.GetActions()
}

func (a *App) UpdateAction(action config.Action) error {
	a.cfg.SetAction(action)
	if err := a.cfg.Save(); err != nil {
		return err
	}

	// Sync changes to Twitch if connected and reward exists
	if a.twClient != nil && a.twClient.IsConnected() && action.RewardID != "" {
		fields := map[string]interface{}{
			"title": action.RewardTitle,
			"cost":  action.RewardCost,
		}
		twCooldown := action.Cooldown / 1000
		if action.TwitchCooldown > 0 {
			twCooldown = action.TwitchCooldown
		}
		if twCooldown >= 60 {
			fields["is_global_cooldown_enabled"] = true
			fields["global_cooldown_seconds"] = twCooldown
		} else if twCooldown > 0 {
			fields["is_global_cooldown_enabled"] = true
			fields["global_cooldown_seconds"] = 60
		}
		if err := a.twClient.UpdateReward(action.RewardID, fields, action.RewardColor); err != nil {
			debuglog.Log("UpdateAction: Twitch sync error: %s", err)
		} else {
			debuglog.Log("UpdateAction: Twitch reward updated for %s", action.ID)
		}
	}
	return nil
}

func (a *App) AddCustomAction(action config.Action) (config.Action, error) {
	action.ID = fmt.Sprintf("custom_%d", time.Now().UnixNano())
	action.Custom = true
	action.Enabled = false

	if action.Name == "" || action.Key == "" || action.RewardTitle == "" {
		return action, fmt.Errorf("Name, Key und Reward-Titel sind erforderlich")
	}

	a.cfg.SetAction(action)
	if err := a.cfg.Save(); err != nil {
		return action, err
	}

	debuglog.Log("AddCustomAction: created %s (%s)", action.ID, action.Name)
	runtime.EventsEmit(a.ctx, "actions-updated", nil)
	return action, nil
}

func (a *App) DeleteAction(id string) error {
	actions := a.cfg.GetActions()
	var target *config.Action
	for _, act := range actions {
		if act.ID == id {
			act := act
			target = &act
			break
		}
	}
	if target == nil {
		return fmt.Errorf("action not found: %s", id)
	}
	if !target.Custom {
		return fmt.Errorf("cannot delete default action: %s", id)
	}

	// Delete Twitch reward if it exists
	if target.RewardID != "" && a.twClient != nil && a.twClient.IsConnected() {
		if err := a.twClient.DeleteReward(target.RewardID); err != nil {
			debuglog.Log("DeleteAction: Twitch reward delete error: %s", err)
		}
	}

	if !a.cfg.DeleteAction(id) {
		return fmt.Errorf("failed to delete action: %s", id)
	}

	debuglog.Log("DeleteAction: deleted %s (%s)", id, target.Name)
	runtime.EventsEmit(a.ctx, "actions-updated", nil)
	return a.cfg.Save()
}

func (a *App) ToggleAction(id string, enabled bool) error {
	a.cfg.ToggleAction(id, enabled)

	actns := a.cfg.GetActions()
	for _, act := range actns {
		if act.ID == id && a.twClient != nil && a.twClient.IsConnected() {
			if act.RewardID != "" {
				// Reward auf Twitch aktivieren/deaktivieren (pausieren)
				debuglog.Log("ToggleAction: %s enabled=%v rewardID=%s", id, enabled, act.RewardID)
				if err := a.twClient.UpdateRewardEnabled(act.RewardID, enabled, act.RewardColor); err != nil {
					debuglog.Log("ToggleAction: UpdateRewardEnabled error: %s", err)
				}
			} else if enabled {
				// Kein Reward vorhanden, neu erstellen
				twCooldown := act.Cooldown
				if act.TwitchCooldown > 0 {
					twCooldown = act.TwitchCooldown * 1000
				}
				rewardID, err := a.twClient.CreateReward(act.RewardTitle, act.RewardCost, twCooldown, act.RewardColor)
				if err == nil {
					act.RewardID = rewardID
					a.cfg.SetAction(act)
				}
			}
		}
	}

	return a.cfg.Save()
}

func (a *App) SetGlobalEnable(enabled bool) error {
	a.cfg.GlobalEnable = enabled
	runtime.EventsEmit(a.ctx, "global-toggle", enabled)

	// AUS = Rewards löschen, AN = Rewards neu erstellen
	if a.twClient != nil && a.twClient.IsConnected() {
		go func() {
			if enabled {
				// Rewards neu erstellen
				runtime.EventsEmit(a.ctx, "twitch-log", "Erstelle Rewards auf Twitch...")
				a.SyncRewards()
				runtime.EventsEmit(a.ctx, "twitch-log", "Rewards erstellt")
			} else {
				// Rewards löschen
				runtime.EventsEmit(a.ctx, "twitch-log", "Lösche Rewards von Twitch...")
				a.DeleteAllRewards()
				runtime.EventsEmit(a.ctx, "twitch-log", "Rewards gelöscht")
			}
		}()
	}

	return a.cfg.Save()
}

func (a *App) GetGlobalEnable() bool {
	return a.cfg.GlobalEnable
}

func (a *App) SetTargetWindow(name string) error {
	a.cfg.TargetWindow = name
	return a.cfg.Save()
}

// --- Twitch Device Code Flow ---

type DeviceAuthInfo struct {
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
}

func (a *App) StartTwitchAuth() (*DeviceAuthInfo, error) {
	a.twClient = twitch.NewClient(&a.cfg.Twitch)
	a.setupTwitchCallbacks()

	dcr, err := a.twClient.RequestDeviceCode()
	if err != nil {
		return nil, err
	}

	// Open the verification URL in the browser
	runtime.BrowserOpenURL(a.ctx, dcr.VerificationURI)

	// Poll for token in background, then auto-connect EventSub
	go func() {
		err := a.twClient.PollForToken(dcr.DeviceCode, dcr.Interval, dcr.ExpiresIn)
		if err != nil {
			runtime.EventsEmit(a.ctx, "twitch-error", err.Error())
			return
		}
		a.cfg.Save()
		runtime.EventsEmit(a.ctx, "twitch-authenticated", a.cfg.Twitch.ChannelName)

		// Auto-connect to EventSub
		if err := a.twClient.Connect(); err != nil {
			runtime.EventsEmit(a.ctx, "twitch-log",
				fmt.Sprintf("EventSub Auto-Connect fehlgeschlagen: %s", err.Error()))
			return
		}
		runtime.EventsEmit(a.ctx, "twitch-log", "EventSub verbunden")

		// Sync rewards
		if a.cfg.GlobalEnable {
			a.SyncRewards()
			runtime.EventsEmit(a.ctx, "twitch-log", "Rewards synchronisiert")
		}
	}()

	return &DeviceAuthInfo{
		UserCode:        dcr.UserCode,
		VerificationURI: dcr.VerificationURI,
	}, nil
}

func (a *App) ConnectTwitch() error {
	return a.connectTwitch()
}

func (a *App) DisconnectTwitch() {
	if a.twClient != nil {
		a.twClient.Disconnect()
		runtime.EventsEmit(a.ctx, "twitch-disconnected", nil)
	}
}

func (a *App) IsTwitchConnected() bool {
	return a.twClient != nil && a.twClient.IsConnected()
}

func (a *App) GetTwitchChannel() string {
	return a.cfg.Twitch.ChannelName
}

func (a *App) autoConnect() {
	a.twClient = twitch.NewClient(&a.cfg.Twitch)
	a.setupTwitchCallbacks()

	// Try to refresh token first
	runtime.EventsEmit(a.ctx, "twitch-log", "Token wird aktualisiert...")
	if err := a.twClient.RefreshAccessToken(); err != nil {
		runtime.EventsEmit(a.ctx, "twitch-log",
			fmt.Sprintf("Token-Refresh fehlgeschlagen: %s — bitte neu verbinden", err.Error()))
		runtime.EventsEmit(a.ctx, "twitch-disconnected", "Token abgelaufen")
		return
	}
	a.cfg.Save()
	runtime.EventsEmit(a.ctx, "twitch-log", "Token aktualisiert")

	// Connect to EventSub
	if err := a.twClient.Connect(); err != nil {
		runtime.EventsEmit(a.ctx, "twitch-log",
			fmt.Sprintf("Auto-Connect fehlgeschlagen: %s", err.Error()))
		return
	}

	runtime.EventsEmit(a.ctx, "twitch-authenticated", a.cfg.Twitch.ChannelName)
}

func (a *App) connectTwitch() error {
	if a.twClient == nil {
		a.twClient = twitch.NewClient(&a.cfg.Twitch)
		a.setupTwitchCallbacks()
	}

	// Refresh token before connecting
	if a.cfg.Twitch.RefreshToken != "" {
		if err := a.twClient.RefreshAccessToken(); err == nil {
			a.cfg.Save()
		}
	}

	return a.twClient.Connect()
}

func (a *App) reconnectLoop() {
	retries := 0
	maxRetries := 10
	for retries < maxRetries {
		retries++
		delay := time.Duration(min(retries*5, 30)) * time.Second
		runtime.EventsEmit(a.ctx, "twitch-log",
			fmt.Sprintf("Reconnect in %ds... (Versuch %d/%d)", int(delay.Seconds()), retries, maxRetries))
		time.Sleep(delay)

		if a.twClient == nil {
			return
		}

		// Refresh token
		if a.cfg.Twitch.RefreshToken != "" {
			if err := a.twClient.RefreshAccessToken(); err == nil {
				a.cfg.Save()
			}
		}

		if err := a.twClient.Connect(); err != nil {
			runtime.EventsEmit(a.ctx, "twitch-log",
				fmt.Sprintf("Reconnect fehlgeschlagen: %s", err.Error()))
			continue
		}

		runtime.EventsEmit(a.ctx, "twitch-log", "Reconnect erfolgreich!")
		return
	}
	runtime.EventsEmit(a.ctx, "twitch-log", "Reconnect aufgegeben — bitte manuell verbinden")
}

func (a *App) setupTwitchCallbacks() {
	a.twClient.SetOnRedemption(func(rewardID, userName, rewardTitle string) {
		debuglog.Log("Redemption: user=%s reward=%q id=%s", userName, rewardTitle, rewardID)
		runtime.EventsEmit(a.ctx, "twitch-log",
			fmt.Sprintf("Redemption von %s: %s", userName, rewardTitle))
		for _, action := range a.cfg.GetActions() {
			if action.RewardID == rewardID {
				debuglog.Log("Redemption matched: action=%s", action.ID)
				if err := a.executor.Execute(action.ID, userName); err != nil {
					debuglog.Log("Redemption execute error: %s", err)
					runtime.EventsEmit(a.ctx, "action-error", map[string]string{
						"action": action.ID,
						"error":  err.Error(),
					})
				}
				return
			}
		}
		debuglog.Log("Redemption: NO MATCH for rewardID=%s", rewardID)
		runtime.EventsEmit(a.ctx, "twitch-log", fmt.Sprintf("Unknown reward: %s (%s)", rewardTitle, rewardID))
	})

	a.twClient.SetOnConnect(func() {
		runtime.EventsEmit(a.ctx, "twitch-connected", nil)
	})

	a.twClient.SetOnDisconnect(func(err error) {
		msg := "disconnected"
		if err != nil {
			msg = err.Error()
		}
		runtime.EventsEmit(a.ctx, "twitch-disconnected", msg)

		// Auto-reconnect if we have tokens
		if a.cfg.Twitch.RefreshToken != "" && err != nil {
			go a.reconnectLoop()
		}
	})

	a.twClient.SetOnLog(func(msg string) {
		runtime.EventsEmit(a.ctx, "twitch-log", msg)
	})
}

// --- Reward Management ---

func (a *App) SyncRewards() error {
	if a.twClient == nil || !a.twClient.IsConnected() {
		debuglog.Log("SyncRewards: nicht verbunden")
		return fmt.Errorf("not connected to Twitch")
	}

	debuglog.Log("=== SyncRewards START ===")

	// Fetch existing rewards to avoid duplicates
	existing, err := a.twClient.GetExistingRewards()
	if err != nil {
		debuglog.Log("SyncRewards: GetExistingRewards error: %s", err)
		runtime.EventsEmit(a.ctx, "twitch-log",
			fmt.Sprintf("Konnte bestehende Rewards nicht laden: %s", err.Error()))
		existing = nil
	}

	// Build a title -> ID map of existing rewards
	existingByTitle := make(map[string]string)
	existingByID := make(map[string]bool)
	for _, r := range existing {
		existingByTitle[r.Title] = r.ID
		existingByID[r.ID] = true
	}
	debuglog.Log("SyncRewards: %d Rewards auf Twitch gefunden", len(existing))

	actns := a.cfg.GetActions()
	debuglog.Log("SyncRewards: %d Aktionen in Config", len(actns))

	for _, action := range actns {
		debuglog.Log("SyncRewards: action=%s enabled=%v rewardID=%q title=%q",
			action.ID, action.Enabled, action.RewardID, action.RewardTitle)

		if !action.Enabled {
			// Wenn disabled aber noch eine RewardID hat, aufräumen
			if action.RewardID != "" {
				debuglog.Log("SyncRewards: %s ist disabled aber hat RewardID=%s — lösche", action.ID, action.RewardID)
				a.twClient.DeleteReward(action.RewardID)
				action.RewardID = ""
				a.cfg.SetAction(action)
			}
			continue
		}

		// Check if already linked AND reward still exists on Twitch
		if action.RewardID != "" {
			if existingByID[action.RewardID] {
				debuglog.Log("SyncRewards: %s bereits verknüpft und existiert (ID: %s)", action.ID, action.RewardID)
				runtime.EventsEmit(a.ctx, "twitch-log",
					fmt.Sprintf("Reward '%s' bereits verknüpft (ID: %s)", action.RewardTitle, action.RewardID))
				continue
			}
			// RewardID saved but doesn't exist on Twitch anymore — clear it
			debuglog.Log("SyncRewards: %s hat RewardID=%s aber existiert NICHT auf Twitch — wird neu erstellt", action.ID, action.RewardID)
			action.RewardID = ""
		}

		// Check if reward already exists on Twitch by title
		if existingID, ok := existingByTitle[action.RewardTitle]; ok {
			debuglog.Log("SyncRewards: %s gefunden über Titel: %q → ID=%s", action.ID, action.RewardTitle, existingID)
			action.RewardID = existingID
			a.cfg.SetAction(action)
			runtime.EventsEmit(a.ctx, "twitch-log",
				fmt.Sprintf("Bestehender Reward gefunden: %s (ID: %s)", action.RewardTitle, existingID))
			continue
		}

		// Create new reward
		twCooldown := action.Cooldown
		if action.TwitchCooldown > 0 {
			twCooldown = action.TwitchCooldown * 1000
		}
		debuglog.Log("SyncRewards: erstelle neuen Reward für %s: title=%q cost=%d twCooldown=%dms", action.ID, action.RewardTitle, action.RewardCost, twCooldown)
		rewardID, err := a.twClient.CreateReward(action.RewardTitle, action.RewardCost, twCooldown, action.RewardColor)
		if err != nil {
			debuglog.Log("SyncRewards: FEHLER beim Erstellen von %s: %s", action.ID, err)
			runtime.EventsEmit(a.ctx, "twitch-log",
				fmt.Sprintf("Fehler beim Erstellen von '%s': %s", action.RewardTitle, err.Error()))
			continue
		}

		action.RewardID = rewardID
		a.cfg.SetAction(action)
		debuglog.Log("SyncRewards: %s erstellt → ID=%s", action.ID, rewardID)
		runtime.EventsEmit(a.ctx, "twitch-log",
			fmt.Sprintf("Reward erstellt: %s (ID: %s)", action.RewardTitle, rewardID))
	}

	debuglog.Log("=== SyncRewards ENDE ===")
	return a.cfg.Save()
}

func (a *App) DeleteAllRewards() error {
	if a.twClient == nil {
		return fmt.Errorf("not connected to Twitch")
	}

	debuglog.Log("=== DeleteAllRewards START ===")
	actns := a.cfg.GetActions()
	for _, action := range actns {
		if action.RewardID == "" {
			continue
		}
		debuglog.Log("DeleteAllRewards: lösche %s (RewardID=%s)", action.ID, action.RewardID)
		a.twClient.DeleteReward(action.RewardID)
		action.RewardID = ""
		a.cfg.SetAction(action)
	}
	debuglog.Log("=== DeleteAllRewards ENDE ===")

	return a.cfg.Save()
}

func (a *App) GetDebugLogPath() string {
	return debuglog.GetLogPath()
}

// --- Language ---

func (a *App) GetLanguage() string {
	if a.cfg.Language == "" {
		return "de"
	}
	return a.cfg.Language
}

func (a *App) SetLanguage(lang string) error {
	a.cfg.Language = lang
	return a.cfg.Save()
}

// --- Tarkov Keybind Import ---

func (a *App) GetTarkovConfigPath() string {
	if a.cfg.TarkovPath != "" {
		return a.cfg.TarkovPath
	}
	return tarkov.DefaultConfigPath()
}

func (a *App) SetTarkovConfigPath(path string) error {
	a.cfg.TarkovPath = path
	return a.cfg.Save()
}

func (a *App) ImportTarkovKeybinds() (int, error) {
	debuglog.Log("=== ImportTarkovKeybinds START ===")
	ctrl, err := tarkov.ReadKeybinds(a.cfg.TarkovPath)
	if err != nil {
		debuglog.Log("ImportTarkovKeybinds: error reading control.ini: %s", err)
		return 0, err
	}

	movementKeys := ctrl.GetMovementKeys()
	updated := 0

	// Also get the stop key (S by default)
	stopKey := ctrl.GetKeyForAction("Duck") // fallback
	for _, ab := range ctrl.AxisBindings {
		if ab.AxisName == "MoveY" && len(ab.Pairs) > 0 {
			neg := ab.Pairs[0].Negative.KeyCode
			if len(neg) > 0 {
				stopKey = tarkov.ConvertKeyCodes(neg)
			}
		}
	}

	actns := a.cfg.GetActions()
	for _, action := range actns {
		tarkovKeyName := action.TarkovBind
		if tarkovKeyName == "" {
			if mapped, ok := tarkov.ActionToTarkovKey[action.ID]; ok {
				tarkovKeyName = mapped
			}
		}
		if tarkovKeyName == "" {
			continue
		}

		newKey := ctrl.GetKeyForAction(tarkovKeyName)
		if newKey == "" {
			continue
		}

		changed := false
		if action.Key != newKey {
			oldKey := action.Key
			action.Key = newKey
			changed = true

			// Update Steps that reference the old key
			for i, step := range action.Steps {
				if step.Key == oldKey {
					action.Steps[i].Key = newKey
				}
				if step.Release == oldKey {
					action.Steps[i].Release = newKey
				}
			}
		}

		// Update grenade stop-key in Steps
		if action.ID == "grenade" && len(action.Steps) > 0 && stopKey != "" {
			if action.Steps[0].Key != stopKey && !action.Steps[0].HoldDown {
				action.Steps[0].Key = stopKey
				changed = true
			}
		}

		// Update movement key locks with actual movement keys
		if action.KeyLock.Enabled && len(movementKeys) > 0 {
			isMovementLock := false
			for _, k := range action.KeyLock.Keys {
				if k == "w" || k == "a" || k == "s" || k == "d" {
					isMovementLock = true
					break
				}
			}
			if isMovementLock {
				// Keep non-movement keys (like "space") and replace movement keys
				var newLockKeys []string
				for _, k := range action.KeyLock.Keys {
					if k != "w" && k != "a" && k != "s" && k != "d" {
						newLockKeys = append(newLockKeys, k)
					}
				}
				action.KeyLock.Keys = append(movementKeys, newLockKeys...)
				changed = true
			}
		}

		debuglog.Log("ImportTarkovKeybinds: action=%s tarkovBind=%s currentKey=%q newKey=%q changed=%v",
			action.ID, tarkovKeyName, action.Key, newKey, changed)

		if changed {
			a.cfg.SetAction(action)
			updated++
			runtime.EventsEmit(a.ctx, "twitch-log",
				fmt.Sprintf("Keybind aktualisiert: %s → %s", action.Name, newKey))
		}
	}

	debuglog.Log("ImportTarkovKeybinds: %d aktualisiert", updated)
	a.cfg.Save()
	runtime.EventsEmit(a.ctx, "actions-updated", nil)
	debuglog.Log("=== ImportTarkovKeybinds ENDE ===")

	return updated, nil
}

// --- Status ---

func (a *App) GetCooldownRemaining(actionID string) int {
	return a.executor.GetCooldownRemaining(actionID)
}

func (a *App) GetLockedKeys() []string {
	return a.keyLocker.GetLockedKeys()
}
