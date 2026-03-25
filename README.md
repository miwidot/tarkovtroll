# TarkovTroll

**Let your Twitch viewers troll your Escape from Tarkov gameplay!**

TarkovTroll connects Twitch Channel Points with Escape from Tarkov. Your viewers redeem Channel Point rewards and the app automatically triggers in-game actions — throw grenades, go prone, reload, play voicelines and much more.

---

## Features

- **18+ built-in actions** — Throw grenades, reload, go prone, jump, crouch, drop magazine, flashlight, night vision, fire mode switch and more
- **Custom actions** — Create your own Channel Point actions with the "+" button
- **Key Lock system** — Blocks keyboard keys while an action is running (e.g. lock WASD during grenade throw so the streamer can't move)
- **Twitch Channel Points integration** — Rewards are automatically created, enabled/disabled and deleted on Twitch
- **Reward customization** — Color, cost, title and cooldown configurable per action
- **Separate Twitch cooldown** — Twitch cooldown configurable independently from local cooldown
- **Global master toggle** — Pause/resume all Twitch rewards with a single click
- **Target window detection** — Actions only execute when Tarkov is in the foreground
- **Event log** — All redemptions and actions are logged live
- **Multi-monitor support** — Keyboard hook only active during key locks, won't block typing on other monitors
- **German & English** — Fully bilingual UI

## Screenshots

*Coming soon*

## Installation

### Requirements
- Windows 10/11
- [Twitch Account](https://twitch.tv) (Affiliate or Partner for Channel Points)

### Download
1. Download the latest `TarkovTroll.exe` from [Releases](https://github.com/miwidot/tarkovtroll/releases)
2. Run it — done!

### Twitch Setup
1. Open TarkovTroll and go to the **Twitch** tab
2. Enter your **Client ID** and **Client Secret** (from [dev.twitch.tv](https://dev.twitch.tv/console/apps))
3. Click **Authenticate** — a browser window will open
4. After login, rewards are automatically created on your channel

> **Redirect URI** for the Twitch app: `http://localhost:19384/callback`
> **Scopes**: `channel:manage:redemptions`, `channel:read:redemptions`

## Building from Source

### Requirements
- [Go 1.21+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/)
- [Wails v2](https://wails.io/docs/gettingstarted/installation)

```bash
git clone https://github.com/miwidot/tarkovtroll.git
cd tarkovtroll

# Development mode (hot reload)
wails dev

# Production build
wails build
# -> build/bin/TarkovTroll.exe
```

## Built-in Actions

| Action | Key | Description | Cost |
|--------|-----|-------------|------|
| Throw grenade | G | Stops, pulls grenade and throws | 500 |
| Reload | R | Reloads the weapon | 200 |
| Open inventory | Tab | Opens inventory | 300 |
| Go prone | X | Drops to prone | 150 |
| Drop magazine | Alt+R | Drops the current magazine | 800 |
| Heal | 4 | Uses healing item (slot 4) | 250 |
| Check magazine | Alt+T | Checks the magazine | 100 |
| Voiceline | F1 | Plays a voiceline | 50 |
| Jump | Space | Jumps | 100 |
| Lean left/right | Q/E | Leans to the side | 100 |
| Flashlight | J | Toggles flashlight | 75 |
| Fire weapon | Mouse0 | Fires shots | 1000 |
| Drop backpack | Z | Drops the backpack | 2000 |
| ... and more | | | |

All actions are fully customizable — keys, costs, cooldowns and key lock settings can be changed in the edit modal.

## Tech Stack

- **Backend**: Go with [Wails v2](https://wails.io/)
- **Frontend**: [Svelte](https://svelte.dev/)
- **Twitch**: EventSub WebSocket + Helix API
- **Input simulation**: Windows SendInput via user32.dll
- **Key Lock**: WH_KEYBOARD_LL hook (on-demand, only active during key locks)

## FAQ

**Can I get banned?**
TarkovTroll simulates key presses like any other macro software. Use at your own risk. See disclaimer below.

**Does it work with other games?**
In theory yes — you can change the target window in settings and create custom actions. However, TarkovTroll is primarily designed for Tarkov.

**My viewers can't redeem the rewards?**
Make sure you are a Twitch Affiliate or Partner and have Channel Points enabled on your channel.

**The app blocks my keyboard on other monitors?**
This should no longer happen since Alpha 1.1. The keyboard hook is only active while an action with key lock is running.

## Contributing

Pull requests are welcome! For major changes, please open an issue first.

## License

MIT License — see [LICENSE](LICENSE)

---

## Disclaimer

> **TarkovTroll is NOT an official Battlestate Games product.**
>
> This project is not affiliated with, authorized by, sponsored by, or in any way officially connected with Battlestate Games Limited. "Escape from Tarkov" is a registered trademark of Battlestate Games Limited.
>
> Use of this software is at your own risk. The developers assume no liability for any consequences resulting from the use of this software, including but not limited to account bans or other actions taken by Battlestate Games or Twitch.
>
> This software simulates keyboard input and may be classified as third-party software depending on the interpretation of the Escape from Tarkov terms of service. Please review the current Battlestate Games terms of service before using this software.

---

*Inspired by [InstructBot](https://www.instructbot.co.uk/)*
