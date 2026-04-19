<script>
  import { onMount } from 'svelte'
  import { GetConfig, SetTargetWindow, GetLanguage, SetLanguage, ImportTarkovKeybinds, GetTarkovConfigPath } from '../../wailsjs/go/main/App'
  import { i18n, locale } from '../i18n'

  let targetWindow = ''
  let currentLang = 'de'
  let tarkovPath = ''
  let importStatus = ''
  let importing = false

  onMount(async () => {
    const cfg = await GetConfig()
    targetWindow = cfg.target_window || ''
    currentLang = cfg.language || 'de'
    locale.set(currentLang)
    try {
      tarkovPath = await GetTarkovConfigPath()
    } catch(e) {}
  })

  async function saveTargetWindow() {
    await SetTargetWindow(targetWindow)
  }

  async function changeLang(lang) {
    currentLang = lang
    locale.set(lang)
    await SetLanguage(lang)
  }

  async function doImport() {
    importing = true
    importStatus = ''
    try {
      const count = await ImportTarkovKeybinds()
      if (count > 0) {
        importStatus = currentLang === 'de'
          ? `${count} Keybind(s) aktualisiert!`
          : `${count} keybind(s) updated!`
      } else {
        importStatus = currentLang === 'de'
          ? 'Alle Keybinds sind bereits aktuell.'
          : 'All keybinds are already up to date.'
      }
    } catch(e) {
      importStatus = (currentLang === 'de' ? 'Fehler: ' : 'Error: ') + e
    }
    importing = false
  }
</script>

<div class="panel">
  <h2>{$i18n('settings.title')}</h2>

  <div class="section">
    <h3>{$i18n('settings.language')}</h3>
    <div class="lang-selector">
      <button class="lang-btn" class:active={currentLang === 'de'} on:click={() => changeLang('de')}>
        Deutsch
      </button>
      <button class="lang-btn" class:active={currentLang === 'en'} on:click={() => changeLang('en')}>
        English
      </button>
    </div>
  </div>

  <div class="section tarkov-import">
    <h3>{currentLang === 'de' ? 'Tarkov Keybinds importieren' : 'Import Tarkov Keybinds'}</h3>
    <p class="hint">
      {currentLang === 'de'
        ? 'Liest deine Tarkov control.ini und passt alle Aktionen automatisch an deine Keybinds an.'
        : 'Reads your Tarkov control.ini and automatically matches all actions to your keybinds.'}
    </p>
    {#if tarkovPath}
      <p class="path">{tarkovPath}</p>
    {/if}
    <button class="btn import-btn" on:click={doImport} disabled={importing}>
      {importing
        ? (currentLang === 'de' ? 'Importiere...' : 'Importing...')
        : (currentLang === 'de' ? 'Keybinds importieren' : 'Import Keybinds')}
    </button>
    {#if importStatus}
      <p class="import-status" class:error={importStatus.startsWith('Fehler') || importStatus.startsWith('Error')}>
        {importStatus}
      </p>
    {/if}
  </div>

  <div class="section">
    <h3>{$i18n('settings.target_window')}</h3>
    <p class="hint">{$i18n('settings.target_window.hint')}</p>
    <div class="form-row">
      <input type="text" bind:value={targetWindow} placeholder="EscapeFromTarkov" />
      <button class="btn" on:click={saveTargetWindow}>{$i18n('actions.save')}</button>
    </div>
  </div>

  <div class="section">
    <h3>{$i18n('settings.keylock.title')}</h3>
    <p class="hint">{$i18n('settings.keylock.desc')}</p>
    <p class="hint muted">{$i18n('settings.keylock.hint')}</p>
  </div>

  <div class="section about">
    <h3>{$i18n('settings.about')}</h3>
    <p class="hint">{$i18n('settings.about.desc')}</p>
    <div class="about-footer">
      <span class="about-brand">TARKOVTROLL</span>
      <span class="about-version">v1.1.1 Alpha</span>
    </div>
  </div>
</div>

<style>
  .panel { max-width: 640px; }
  h2 {
    font-size: 20px;
    font-weight: 700;
    color: #f3f4f6;
    letter-spacing: 1px;
    margin-bottom: 22px;
  }
  h3 {
    font-size: 14px;
    color: var(--text-primary);
    font-weight: 600;
    margin-bottom: 10px;
  }

  .section {
    margin-bottom: 16px;
    padding: 18px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }

  .hint {
    font-size: 12px;
    color: var(--text-secondary);
    margin-bottom: 12px;
    line-height: 1.6;
  }
  .hint.muted {
    color: var(--text-muted);
    font-style: italic;
    margin-bottom: 0;
  }

  .lang-selector {
    display: flex;
    gap: 8px;
  }
  .lang-btn {
    padding: 8px 24px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    cursor: pointer;
    border-radius: var(--radius-sm);
    font-size: 13px;
    font-weight: 600;
    transition: all 0.15s;
  }
  .lang-btn.active {
    background: rgba(83, 252, 24, 0.10);
    border-color: rgba(83, 252, 24, 0.40);
    color: #53FC18;
  }
  .lang-btn:hover:not(.active) {
    border-color: var(--border-hover);
    color: var(--text-primary);
  }

  .form-row {
    display: flex;
    gap: 10px;
  }
  .form-row input {
    flex: 1;
    padding: 8px 12px;
    background: var(--bg-input);
    border: 1px solid var(--border);
    color: var(--text-primary);
    border-radius: var(--radius-sm);
    font-size: 13px;
  }
  .form-row input:focus {
    outline: none;
    border-color: rgba(83, 252, 24, 0.50);
    box-shadow: 0 0 0 2px rgba(83, 252, 24, 0.12);
  }

  .btn {
    padding: 8px 18px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    color: var(--text-primary);
    cursor: pointer;
    border-radius: var(--radius-sm);
    font-size: 12px;
    font-weight: 600;
    transition: all 0.15s;
  }
  .btn:hover { border-color: rgba(83, 252, 24, 0.40); color: #53FC18; }

  .tarkov-import {
    border-color: rgba(105, 122, 58, 0.4);
    background: linear-gradient(135deg, var(--bg-card), rgba(105, 122, 58, 0.08));
  }

  .import-btn {
    background: #697a3a !important;
    color: #f3f4f6 !important;
    border-color: #697a3a !important;
    font-weight: 700;
    padding: 10px 24px;
    font-size: 13px;
  }
  .import-btn:hover { filter: brightness(1.1); }
  .import-btn:disabled { opacity: 0.45; cursor: not-allowed; }

  .path {
    font-size: 11px;
    color: var(--text-muted);
    font-family: 'Consolas', monospace;
    margin-bottom: 12px;
    word-break: break-all;
    opacity: 0.7;
  }

  .import-status {
    margin-top: 12px;
    font-size: 13px;
    color: var(--success);
    font-weight: 600;
  }
  .import-status.error { color: var(--danger); }

  .about {
    border-color: var(--border);
  }
  .about-footer {
    display: flex;
    align-items: baseline;
    gap: 10px;
    margin-top: 8px;
  }
  .about-brand {
    font-size: 14px;
    font-weight: 800;
    color: #53FC18;
    letter-spacing: 3px;
  }
  .about-version {
    font-size: 11px;
    color: var(--text-muted);
  }
</style>
