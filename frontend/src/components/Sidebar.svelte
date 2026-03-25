<script>
  import { i18n } from '../i18n'

  export let currentTab = 'actions'

  $: tabs = [
    { id: 'actions', label: $i18n('nav.actions'), icon: 'A' },
    { id: 'twitch', label: $i18n('nav.twitch'), icon: 'T' },
    { id: 'log', label: $i18n('nav.log'), icon: 'L' },
    { id: 'settings', label: $i18n('nav.settings'), icon: 'S' },
  ]
</script>

<nav class="sidebar">
  <div class="nav-items">
    {#each tabs as tab}
      <button
        class="nav-item"
        class:active={currentTab === tab.id}
        on:click={() => currentTab = tab.id}
      >
        <span class="nav-icon">{tab.icon}</span>
        <span class="nav-label">{tab.label}</span>
      </button>
    {/each}
  </div>
  <div class="sidebar-footer">
    <span class="version">v1.0</span>
  </div>
</nav>

<style>
  .sidebar {
    width: 180px;
    background: var(--bg-secondary);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    justify-content: space-between;
  }

  .nav-items {
    display: flex;
    flex-direction: column;
    padding: 12px 8px;
    gap: 2px;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 14px;
    background: none;
    border: none;
    color: var(--text-secondary);
    font-size: 13px;
    cursor: pointer;
    transition: all 0.15s;
    border-radius: var(--radius-sm);
    text-align: left;
  }

  .nav-item:hover {
    color: var(--text-primary);
    background: rgba(255,255,255,0.04);
  }

  .nav-item.active {
    color: #53FC18;
    background: rgba(83, 252, 24, 0.08);
    border-left: 3px solid #53FC18;
    padding-left: 11px;
    box-shadow: inset 3px 0 8px rgba(83, 252, 24, 0.15);
  }

  .nav-icon {
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 11px;
    font-weight: 700;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: inherit;
  }

  .nav-item.active .nav-icon {
    background: rgba(83, 252, 24, 0.12);
    border-color: rgba(83, 252, 24, 0.4);
    color: #53FC18;
  }

  .nav-label {
    font-weight: 500;
  }

  .sidebar-footer {
    padding: 12px 16px;
    border-top: 1px solid var(--border);
  }
  .version {
    font-size: 11px;
    color: var(--text-muted);
  }
</style>
