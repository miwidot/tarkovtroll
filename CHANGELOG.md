# TarkovTroll Changelog

## Alpha 1.1.4

### Behoben
- **Rewards flackern nicht mehr beim schnellen Master-Toggle** — Wenn man den Haupt-Schalter schnell AN→AUS umlegte, liefen `SyncRewards` (erstellen) und `DeleteAllRewards` (löschen) gleichzeitig. Resultat: Rewards wurden auf Twitch erstellt und 0,5s später wieder gelöscht. Beide Operationen sind jetzt durch einen Mutex serialisiert.

## Alpha 1.1.3

### Behoben
- **App lockt nicht mehr ab** — Hook-Lifecycle wurde von Channel-basiert (limitierter Buffer) auf atomic Ref-Counter umgestellt. Bei vielen gleichzeitigen Redemptions hat sich vorher die App komplett aufgehängt, weil der `installHookCh` Buffer überlief und `LockKeys` für immer blockierte.
- **Granate funktioniert jetzt wirklich** — Zwei Bugs gefixt:
  1. One-Time-Migration für alle bestehenden User: alte broken Steps mit `mouse0` (was zum Schießen führte) werden automatisch durch `Key, wait, Key` ersetzt.
  2. `ImportTarkovKeybinds` hat bisher bei jedem Aufruf Step 0 der Granaten-Action mit dem "Stop-Key" (s = rückwärts) überschrieben. Diese Logik wurde komplett entfernt.
- **Twitch Token Auto-Refresh** — Wenn ein API-Call mit 401 Unauthorized fehlschlägt (Token abgelaufen), wird das Token jetzt automatisch refresht und der Call neu versucht. Vorher mussten User sich neu authentifizieren.
- **Keybinds bleiben nach Neustart** — `KeybindsImported`-Flag wird jetzt zuverlässig persistiert (omitempty entfernt). Dadurch läuft der Auto-Import nicht mehr bei jedem Start.
- **OS-Thread-Pinning für Windows-Hook** — `messagePump` ist jetzt mit `runtime.LockOSThread()` an einen OS-Thread gepinnt, wie es Windows für Hooks erfordert.

## Alpha 1.1.2

### Behoben
- **Granate funktioniert jetzt** — Steps neu sortiert: G zum Switch, dann G nochmal zum Werfen (statt Mouse0 was im Granate-Modus zum Schießen führte). Default-TarkovBind auf `ThrowGrenade` korrigiert.
- **Eigene Einstellungen werden nicht mehr überschrieben** — `mergeNewActions` hat bisher bei jedem Start Key/HoldMs/KeyLock/Repeat aus den Defaults überschrieben. User-Customizations bleiben jetzt erhalten.
- **Auto-Import läuft nur einmalig** — Beim ersten Start werden Tarkov-Keybinds importiert, danach werden User-Edits respektiert. Manueller Re-Import via Settings weiterhin möglich.

### Neu
- **Aktions-Queue** — Mehrere Redemptions die gleichzeitig kommen werden seriell abgearbeitet (max. 8 in Queue). Verhindert KeyLock-Konflikte und chaotisches Verhalten bei vielen gleichzeitigen Einlösungen.
- **Verbesserter Auto-Import-Fallback** — Wenn der Tarkov-Bind im control.ini nicht gefunden wird, gibt es jetzt einen Fallback auf das interne ID-Mapping.

## Alpha 1.1.1

### Neu
- **360° Spin** — Neue Aktion die den Charakter automatisch um 360 Grad dreht (links/rechts wählbar). Pixel-Anzahl pro Drehung einstellbar je nach Maus-Sensitivity.
- **Auto-Import Tarkov-Keybinds** — Beim Start werden deine Tarkov-Keybinds automatisch aus `control.ini` importiert. Keine manuelle Einrichtung mehr nötig.
- **Code-signiert** — Die Exe ist jetzt mit einem Certum Open-Source-Developer-Zertifikat signiert. SmartScreen-Warnungen sollten seltener werden.

## Alpha 1.1

### Neu
- **Eigene Aktionen erstellen** — Ihr könnt jetzt über den "+" Button eigene Channel-Point-Aktionen anlegen! Key, Cooldown, KeyLock, Reward-Farbe — alles einstellbar. Eigene Aktionen sind mit einem lila "CUSTOM" Badge markiert und lassen sich jederzeit wieder löschen.
- **Doppelstart-Schutz** — TarkovTroll kann nicht mehr versehentlich zweimal gestartet werden. Falls es schon läuft, kommt ein Hinweis.
- **Bessere Tarkov-Erkennung** — Die App erkennt Tarkov jetzt am Prozessnamen statt am Fenstertitel. Sollte deutlich zuverlässiger funktionieren.

### Behoben
- **Sprint wird jetzt zuverlässig gestoppt** — Wenn eine Aktion den Sprint unterbrechen soll (z.B. Granate werfen), wurde vorher manchmal weitergerannt. Das ist jetzt gefixt.
- **Event-Log zeigt alle Events** — Bisher gingen Twitch-Einlösungen im Log verloren wenn man gerade auf einem anderen Tab war. Jetzt werden alle Events mitgeschrieben, egal welcher Tab offen ist.

### Sonstiges
- Eigene Aktionen werden bei Updates nicht mehr von den Standard-Aktionen überschrieben
- Twitch-Reward wird beim Löschen einer eigenen Aktion automatisch mit entfernt
- Löschen erfordert doppelte Bestätigung damit nichts aus Versehen weg ist
