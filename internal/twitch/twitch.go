package twitch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"bitbotgo/internal/config"
	"bitbotgo/internal/debuglog"
)

const (
	ClientID         = "recddyemjfl0xbklhcnukacerbz11n"
	ClientSecret     = "yfdw6fxi4vbzdj73b0anc35yt9gjej"
	twitchDeviceURL  = "https://id.twitch.tv/oauth2/device"
	twitchTokenURL   = "https://id.twitch.tv/oauth2/token"
	twitchAPIURL     = "https://api.twitch.tv/helix"
	eventSubWSURL    = "wss://eventsub.wss.twitch.tv/ws"
	scopes           = "channel:manage:redemptions channel:read:redemptions"
)

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type Client struct {
	mu           sync.RWMutex
	cfg          *config.TwitchConfig
	httpClient   *http.Client
	ws           *websocket.Conn
	sessionID    string
	connected    bool
	onRedemption func(rewardID string, userName string, rewardTitle string)
	onConnect    func()
	onDisconnect func(err error)
	onLog        func(msg string)
	stopCh       chan struct{}
}

type eventSubMessage struct {
	Metadata struct {
		MessageID   string `json:"message_id"`
		MessageType string `json:"message_type"`
	} `json:"metadata"`
	Payload json.RawMessage `json:"payload"`
}

type welcomePayload struct {
	Session struct {
		ID string `json:"id"`
	} `json:"session"`
}

type redemptionPayload struct {
	Subscription struct {
		Type string `json:"type"`
	} `json:"subscription"`
	Event struct {
		UserName string `json:"user_name"`
		Reward   struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"reward"`
	} `json:"event"`
}

func NewClient(cfg *config.TwitchConfig) *Client {
	cfg.ClientID = ClientID
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		stopCh:     make(chan struct{}),
	}
}

func (c *Client) SetOnRedemption(fn func(rewardID string, userName string, rewardTitle string)) {
	c.onRedemption = fn
}

func (c *Client) SetOnConnect(fn func()) {
	c.onConnect = fn
}

func (c *Client) SetOnDisconnect(fn func(err error)) {
	c.onDisconnect = fn
}

func (c *Client) SetOnLog(fn func(msg string)) {
	c.onLog = fn
}

func (c *Client) log(msg string) {
	if c.onLog != nil {
		c.onLog(msg)
	}
}

func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// --- Device Code Flow ---

// RequestDeviceCode initiates the Device Code Flow.
// Returns the user code and verification URI for the user to visit.
func (c *Client) RequestDeviceCode() (*DeviceCodeResponse, error) {
	data := url.Values{
		"client_id": {ClientID},
		"scopes":    {scopes},
	}

	resp, err := c.httpClient.PostForm(twitchDeviceURL, data)
	if err != nil {
		return nil, fmt.Errorf("device code request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("device code request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var dcr DeviceCodeResponse
	if err := json.Unmarshal(body, &dcr); err != nil {
		return nil, fmt.Errorf("failed to decode device code response: %w", err)
	}

	return &dcr, nil
}

// PollForToken polls Twitch until the user authorizes the device code.
func (c *Client) PollForToken(deviceCode string, interval int, expiresIn int) error {
	if interval < 1 {
		interval = 5
	}

	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)

	for time.Now().Before(deadline) {
		time.Sleep(time.Duration(interval) * time.Second)

		data := url.Values{
			"client_id":   {ClientID},
			"scopes":      {scopes},
			"device_code": {deviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		}

		resp, err := c.httpClient.PostForm(twitchTokenURL, data)
		if err != nil {
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 200 {
			var result struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				return fmt.Errorf("failed to decode token: %w", err)
			}
			c.cfg.AccessToken = result.AccessToken
			c.cfg.RefreshToken = result.RefreshToken
			return c.fetchBroadcasterID()
		}

		// Check for pending/slow_down
		var errResp struct {
			Message string `json:"message"`
		}
		json.Unmarshal(body, &errResp)

		if strings.Contains(errResp.Message, "authorization_pending") {
			c.log("Warte auf Autorisierung...")
			continue
		}
		if strings.Contains(errResp.Message, "slow_down") {
			interval += 5
			continue
		}

		// Some other error
		return fmt.Errorf("token poll error: %s", string(body))
	}

	return fmt.Errorf("device code expired")
}

func (c *Client) fetchBroadcasterID() error {
	req, _ := http.NewRequest("GET", twitchAPIURL+"/users", nil)
	req.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
	req.Header.Set("Client-Id", ClientID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID    string `json:"id"`
			Login string `json:"login"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if len(result.Data) == 0 {
		return fmt.Errorf("no user data returned")
	}

	c.cfg.BroadcasterID = result.Data[0].ID
	c.cfg.ChannelName = result.Data[0].Login
	return nil
}

// --- Channel Point Rewards ---

type ExistingReward struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// GetExistingRewards fetches all custom rewards created by this app (only manageable ones).
func (c *Client) GetExistingRewards() ([]ExistingReward, error) {
	debuglog.Log("GetExistingRewards: broadcaster_id=%s", c.cfg.BroadcasterID)
	req, _ := http.NewRequest("GET",
		fmt.Sprintf("%s/channel_points/custom_rewards?broadcaster_id=%s&only_manageable_rewards=true",
			twitchAPIURL, c.cfg.BroadcasterID), nil)
	req.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
	req.Header.Set("Client-Id", ClientID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		debuglog.Log("GetExistingRewards: HTTP error: %s", err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	debuglog.Log("GetExistingRewards: status=%d body=%s", resp.StatusCode, string(respBody))
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get rewards (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data []ExistingReward `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	debuglog.Log("GetExistingRewards: found %d rewards", len(result.Data))
	for _, r := range result.Data {
		debuglog.Log("  existing reward: title=%q id=%s", r.Title, r.ID)
	}
	return result.Data, nil
}

// SetAllRewardsEnabled enables or disables all manageable rewards.
func (c *Client) SetAllRewardsEnabled(enabled bool) error {
	rewards, err := c.GetExistingRewards()
	if err != nil {
		return err
	}
	for _, r := range rewards {
		c.UpdateRewardEnabled(r.ID, enabled, "")
	}
	return nil
}

func (c *Client) CreateReward(title string, cost int, cooldownMs int, color string) (string, error) {
	debuglog.Log("CreateReward: title=%q cost=%d cooldown=%dms color=%s broadcaster_id=%s", title, cost, cooldownMs, color, c.cfg.BroadcasterID)
	body := map[string]interface{}{
		"title":                         title,
		"cost":                          cost,
		"is_enabled":                    true,
		"is_user_input_required":        false,
		"should_redemptions_skip_queue": true,
	}
	// Twitch requires minimum 60 seconds for global cooldown
	cooldownSec := cooldownMs / 1000
	if cooldownSec >= 60 {
		body["is_global_cooldown_enabled"] = true
		body["global_cooldown_seconds"] = cooldownSec
	} else if cooldownSec > 0 {
		body["is_global_cooldown_enabled"] = true
		body["global_cooldown_seconds"] = 60
	}
	// Set reward color if specified (hex format like #9146FF)
	if color != "" {
		body["background_color"] = color
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST",
		fmt.Sprintf("%s/channel_points/custom_rewards?broadcaster_id=%s", twitchAPIURL, c.cfg.BroadcasterID),
		bytes.NewReader(jsonBody))
	req.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
	req.Header.Set("Client-Id", ClientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		debuglog.Log("CreateReward: HTTP error: %s", err)
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	debuglog.Log("CreateReward: status=%d body=%s", resp.StatusCode, string(respBody))
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to create reward (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	if len(result.Data) == 0 {
		return "", fmt.Errorf("no reward data returned")
	}

	debuglog.Log("CreateReward: success id=%s", result.Data[0].ID)
	return result.Data[0].ID, nil
}

func (c *Client) DeleteReward(rewardID string) error {
	debuglog.Log("DeleteReward: id=%s broadcaster_id=%s", rewardID, c.cfg.BroadcasterID)
	req, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/channel_points/custom_rewards?broadcaster_id=%s&id=%s",
			twitchAPIURL, c.cfg.BroadcasterID, rewardID), nil)
	req.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
	req.Header.Set("Client-Id", ClientID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		debuglog.Log("DeleteReward: HTTP error: %s", err)
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	debuglog.Log("DeleteReward: status=%d body=%s", resp.StatusCode, string(body))
	return nil
}

func (c *Client) UpdateRewardEnabled(rewardID string, enabled bool, color string) error {
	return c.UpdateReward(rewardID, map[string]interface{}{
		"is_enabled": enabled,
	}, color)
}

func (c *Client) UpdateReward(rewardID string, fields map[string]interface{}, color string) error {
	if color != "" {
		fields["background_color"] = color
	}
	jsonBody, _ := json.Marshal(fields)

	req, _ := http.NewRequest("PATCH",
		fmt.Sprintf("%s/channel_points/custom_rewards?broadcaster_id=%s&id=%s",
			twitchAPIURL, c.cfg.BroadcasterID, rewardID),
		bytes.NewReader(jsonBody))
	req.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
	req.Header.Set("Client-Id", ClientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		debuglog.Log("UpdateReward: status=%d body=%s", resp.StatusCode, string(respBody))
		return fmt.Errorf("update reward failed (status %d)", resp.StatusCode)
	}
	return nil
}

// --- EventSub WebSocket ---

func (c *Client) Connect() error {
	c.log("Connecting to Twitch EventSub...")

	ws, _, err := websocket.DefaultDialer.Dial(eventSubWSURL, nil)
	if err != nil {
		return fmt.Errorf("websocket connection failed: %w", err)
	}
	c.ws = ws

	_, msgBytes, err := ws.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read welcome: %w", err)
	}

	var msg eventSubMessage
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return fmt.Errorf("failed to parse welcome: %w", err)
	}

	if msg.Metadata.MessageType != "session_welcome" {
		return fmt.Errorf("expected welcome, got: %s", msg.Metadata.MessageType)
	}

	var welcome welcomePayload
	json.Unmarshal(msg.Payload, &welcome)
	c.sessionID = welcome.Session.ID
	c.log(fmt.Sprintf("Connected! Session: %s", c.sessionID))

	if err := c.subscribeToRedemptions(); err != nil {
		return err
	}

	c.mu.Lock()
	c.connected = true
	c.mu.Unlock()

	if c.onConnect != nil {
		c.onConnect()
	}

	go c.readLoop()
	return nil
}

func (c *Client) subscribeToRedemptions() error {
	body := map[string]interface{}{
		"type":    "channel.channel_points_custom_reward_redemption.add",
		"version": "1",
		"condition": map[string]string{
			"broadcaster_user_id": c.cfg.BroadcasterID,
		},
		"transport": map[string]string{
			"method":     "websocket",
			"session_id": c.sessionID,
		},
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", twitchAPIURL+"/eventsub/subscriptions", bytes.NewReader(jsonBody))
	req.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
	req.Header.Set("Client-Id", ClientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 202 {
		return fmt.Errorf("failed to subscribe (status %d): %s", resp.StatusCode, string(respBody))
	}

	c.log("Subscribed to Channel Point redemptions")
	return nil
}

func (c *Client) readLoop() {
	defer func() {
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		if c.onDisconnect != nil {
			c.onDisconnect(nil)
		}
	}()

	for {
		_, msgBytes, err := c.ws.ReadMessage()
		if err != nil {
			if c.onDisconnect != nil {
				c.onDisconnect(err)
			}
			return
		}

		var msg eventSubMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}

		switch msg.Metadata.MessageType {
		case "session_keepalive":
			// ignore
		case "notification":
			c.handleNotification(msg.Payload)
		case "session_reconnect":
			c.log("Reconnect requested")
		}
	}
}

func (c *Client) handleNotification(payload json.RawMessage) {
	var redemption redemptionPayload
	if err := json.Unmarshal(payload, &redemption); err != nil {
		return
	}

	if !strings.Contains(redemption.Subscription.Type, "redemption") {
		return
	}

	c.log(fmt.Sprintf("Redemption: %s by %s", redemption.Event.Reward.Title, redemption.Event.UserName))

	if c.onRedemption != nil {
		c.onRedemption(redemption.Event.Reward.ID, redemption.Event.UserName, redemption.Event.Reward.Title)
	}
}

func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = false
	if c.ws != nil {
		c.ws.Close()
		c.ws = nil
	}
}

func (c *Client) RefreshAccessToken() error {
	data := url.Values{
		"client_id":     {ClientID},
		"client_secret": {ClientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {c.cfg.RefreshToken},
	}

	resp, err := c.httpClient.PostForm(twitchTokenURL, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	c.cfg.AccessToken = result.AccessToken
	c.cfg.RefreshToken = result.RefreshToken
	return nil
}
