# TarkovTroll

**Lass deine Twitch-Zuschauer dein Escape from Tarkov Spiel trolllen!**

TarkovTroll verbindet Twitch Channel Points mit Escape from Tarkov. Deine Zuschauer loesen Channel-Point-Rewards ein und die App fuehrt automatisch Aktionen im Spiel aus — Granate werfen, hinlegen, nachladen, Voiceline abspielen und vieles mehr.

---

## Features

- **18+ vordefinierte Aktionen** — Granate werfen, nachladen, hinlegen, springen, ducken, Magazin droppen, Taschenlampe, Nachtsicht, Feuermodus wechseln und mehr
- **Eigene Aktionen erstellen** — Erstelle beliebige eigene Channel-Point-Aktionen mit dem "+" Button
- **Key Lock System** — Blockiert Tasten waehrend eine Aktion laeuft (z.B. WASD sperren beim Granate werfen, damit der Streamer stehen bleibt)
- **Twitch Channel Points Integration** — Rewards werden automatisch auf Twitch erstellt, aktiviert/deaktiviert und geloescht
- **Reward-Anpassung** — Farbe, Kosten, Titel und Cooldown pro Aktion einstellbar
- **Separater Twitch-Cooldown** — Twitch-Cooldown unabhaengig vom lokalen Cooldown konfigurierbar
- **Globaler Master-Toggle** — Alle Rewards auf Twitch mit einem Klick pausieren/aktivieren
- **Target Window Detection** — Aktionen werden nur ausgefuehrt wenn Tarkov im Vordergrund ist
- **Event-Log** — Alle Einloesungen und Aktionen werden live im Log angezeigt
- **Multi-Monitor Support** — Keyboard Hook nur aktiv wenn Keys gesperrt sind, blockiert nicht das Tippen auf anderen Monitoren
- **Deutsch & Englisch** — Vollstaendig zweisprachig

## Screenshots

*Coming soon*

## Installation

### Voraussetzungen
- Windows 10/11
- [Twitch Account](https://twitch.tv) (Affiliate oder Partner fuer Channel Points)

### Download
1. Lade die neueste `TarkovTroll.exe` aus den [Releases](https://github.com/miwidot/tarkovtroll/releases) herunter
2. Starte die Exe — fertig!

### Twitch einrichten
1. Oeffne TarkovTroll und gehe zum **Twitch** Tab
2. Gib deine **Client-ID** und **Client-Secret** ein (von [dev.twitch.tv](https://dev.twitch.tv/console/apps))
3. Klicke auf **Authentifizieren** — es oeffnet sich ein Browser-Fenster
4. Nach der Anmeldung werden die Rewards automatisch auf deinem Kanal erstellt

> **Redirect URI** bei der Twitch App: `http://localhost:19384/callback`
> **Scopes**: `channel:manage:redemptions`, `channel:read:redemptions`

## Selbst bauen (Development)

### Voraussetzungen
- [Go 1.21+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/)
- [Wails v2](https://wails.io/docs/gettingstarted/installation)

```bash
# Repository klonen
git clone https://github.com/miwidot/tarkovtroll.git
cd tarkovtroll

# Development-Modus (Hot Reload)
wails dev

# Production Build
wails build
# -> build/bin/TarkovTroll.exe
```

## Vordefinierte Aktionen

| Aktion | Key | Beschreibung | Kosten |
|--------|-----|-------------|--------|
| Granate werfen | G | Stoppt, zieht Granate und wirft | 500 |
| Nachladen | R | Laedt die Waffe nach | 200 |
| Inventar oeffnen | Tab | Oeffnet das Inventar | 300 |
| Hinlegen | X | Bauchlage | 150 |
| Magazin droppen | Alt+R | Droppt das aktuelle Magazin | 800 |
| Heilen | 4 | Benutzt Heilitem (Slot 4) | 250 |
| Magazin checken | Alt+T | Checkt das Magazin | 100 |
| Voiceline | F1 | Spielt eine Voiceline ab | 50 |
| Springen | Space | Springt | 100 |
| Links/Rechts lehnen | Q/E | Lehnt sich zur Seite | 100 |
| Taschenlampe | J | Licht an/aus | 75 |
| Schuss abfeuern | Mouse0 | Feuert Schuesse ab | 1000 |
| Rucksack droppen | Z | Droppt den Rucksack | 2000 |
| ... und mehr | | | |

Alle Aktionen sind anpassbar — Keys, Kosten, Cooldowns und KeyLock-Einstellungen koennen im Edit-Modal geaendert werden.

## Technik

- **Backend**: Go mit [Wails v2](https://wails.io/)
- **Frontend**: [Svelte](https://svelte.dev/)
- **Twitch**: EventSub WebSocket + Helix API
- **Input**: Windows SendInput via user32.dll
- **Key Lock**: WH_KEYBOARD_LL Hook (on-demand, nur aktiv wenn Keys gesperrt sind)

## FAQ

**Kann ich gebannt werden?**
TarkovTroll simuliert Tastendruecke wie jede andere Makro-Software. Die Nutzung erfolgt auf eigene Verantwortung. Siehe Disclaimer unten.

**Funktioniert es mit anderen Spielen?**
Theoretisch ja — du kannst das Target Window in den Einstellungen aendern und eigene Aktionen erstellen. TarkovTroll ist aber primaer fuer Tarkov gedacht.

**Meine Zuschauer koennen die Rewards nicht einloesen?**
Stelle sicher, dass du Twitch Affiliate oder Partner bist und Channel Points auf deinem Kanal aktiviert sind.

**Die App blockiert meine Tastatur auf anderen Monitoren?**
Das sollte seit Alpha 1.1 nicht mehr passieren. Der Keyboard Hook ist nur noch aktiv waehrend eine Aktion mit Key Lock laeuft.

## Contributing

Pull Requests sind willkommen! Bei groesseren Aenderungen bitte vorher ein Issue erstellen.

## Lizenz

MIT License — siehe [LICENSE](LICENSE)

---

## Disclaimer

> **TarkovTroll ist KEIN offizielles Produkt von Battlestate Games.**
>
> Dieses Projekt ist nicht mit Battlestate Games Limited affiliiert, autorisiert, gesponsert oder in irgendeiner Weise offiziell verbunden. "Escape from Tarkov" ist eine eingetragene Marke von Battlestate Games Limited.
>
> Die Nutzung dieser Software erfolgt auf eigene Verantwortung. Die Entwickler uebernehmen keine Haftung fuer moegliche Konsequenzen durch die Nutzung dieser Software, einschliesslich aber nicht beschraenkt auf Account-Sperrungen oder andere Massnahmen durch Battlestate Games oder Twitch.
>
> Diese Software simuliert Tastatureingaben und kann je nach Auslegung der Nutzungsbedingungen von Escape from Tarkov als Drittanbieter-Software eingestuft werden. Bitte informiere dich vor der Nutzung ueber die aktuellen Nutzungsbedingungen von Battlestate Games.

---

*Inspired by [InstructBot](https://www.instructbot.co.uk/)*
