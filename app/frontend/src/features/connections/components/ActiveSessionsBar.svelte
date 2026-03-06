<script>
  import {Button} from '$lib/components/ui/button/index.js'
  import {locale, localeOptions, setLocale, t} from '$lib/i18n/index.js'
  import {cn} from '$lib/utils/cn.js'
  import Icon from '@iconify/svelte'
  import mdiCogOutline from '@iconify-icons/mdi/cog-outline'
  import mdiKeyOutline from '@iconify-icons/mdi/key-outline'
  import mdiLockOutline from '@iconify-icons/mdi/lock-outline'
  import {onMount} from 'svelte'

  export let sessions = []
  export let current = 'connections'
  export let onSelect = (_id) => {}
  export let locked = false
  export let canLock = false
  export let onLock = () => {}
  export let onChangePassword = () => {}

  let openMenu = false

  function statusLabel(status) {
    const key = `status.${String(status ?? '')}`
    const label = $t(key)
    return label === key ? String(status ?? '') : label
  }

  function badgeClass(status) {
    if (status === 'connected') return 'bg-emerald-500/15 text-emerald-700 border-emerald-500/30'
    if (status === 'connecting') return 'bg-amber-500/15 text-amber-700 border-amber-500/30'
    if (status === 'error') return 'bg-destructive/10 text-destructive border-destructive/30'
    return 'bg-muted text-muted-foreground border-border'
  }

  function toggleMenu() {
    openMenu = !openMenu
  }

  function closeMenu() {
    openMenu = false
  }

  function handleLock() {
    closeMenu()
    onLock()
  }

  function handleChangePassword() {
    closeMenu()
    onChangePassword()
  }

  function handleDocumentClick(event) {
    const path = event?.composedPath?.() ?? []
    for (const el of path) {
      if (el instanceof Element && el.hasAttribute('data-settings-menu')) {
        return
      }
    }
    closeMenu()
  }

  onMount(() => {
    document.addEventListener('click', handleDocumentClick)
    return () => {
      document.removeEventListener('click', handleDocumentClick)
    }
  })
</script>

<div class="flex items-center gap-2 rounded-lg border border-border bg-card px-2 py-2">
  <div class="flex min-w-0 flex-1 items-center gap-2 overflow-x-auto">
    <Button
      size="sm"
      variant={current === 'connections' ? 'secondary' : 'ghost'}
      on:click={() => onSelect('connections')}
    >
      {$t('nav.connections')}
    </Button>

    <Button
      size="sm"
      variant={current === 'transfers' ? 'secondary' : 'ghost'}
      on:click={() => onSelect('transfers')}
    >
      {$t('nav.transfers')}
    </Button>

    <div class="h-6 w-px bg-border"></div>

    {#each sessions as s (s.sessionID)}
      <button
        type="button"
        class={cn(
          'inline-flex items-center gap-2 rounded-md border px-3 py-1.5 text-sm transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring',
          current === s.sessionID ? 'bg-accent' : 'hover:bg-accent/60',
          badgeClass(s.status)
        )}
        on:click={() => onSelect(s.sessionID)}
      >
        <span class="font-medium">{s.profileName || s.sessionID.slice(0, 8)}</span>
        <span class="text-xs opacity-80">{statusLabel(s.status)}</span>
      </button>
    {/each}
  </div>

  {#if canLock && !locked}
    <div class="h-6 w-px bg-border"></div>
    <div class="relative" data-settings-menu>
      <Button
        size="icon"
        variant="ghost"
        aria-label={$t('common.actions')}
        aria-haspopup="menu"
        aria-expanded={openMenu}
        on:click={toggleMenu}
      >
        <Icon icon={mdiCogOutline} width={18} height={18} class="opacity-80" />
      </Button>

      {#if openMenu}
        <div class="absolute right-0 top-full z-20 mt-1 w-44 rounded-md border border-border bg-background shadow-md">
          <div class="px-3 py-2 text-xs text-muted-foreground">{$t('nav.language')}</div>
          {#each localeOptions as opt (opt.id)}
            <button
              type="button"
              class={cn(
                'flex w-full items-center justify-between gap-2 rounded-sm px-3 py-2 text-left text-xs hover:bg-accent',
                $locale === opt.id ? 'bg-accent/60' : ''
              )}
              on:click={() => setLocale(opt.id)}
            >
              <span>{opt.label}</span>
              {#if $locale === opt.id}
                <span class="text-muted-foreground">✓</span>
              {/if}
            </button>
          {/each}

          <div class="h-px bg-border my-1"></div>
          <button
            type="button"
            class="flex w-full items-center gap-2 rounded-sm px-3 py-2 text-left text-xs hover:bg-accent"
            on:click={handleChangePassword}
          >
            <Icon icon={mdiKeyOutline} width={16} height={16} class="shrink-0 opacity-80" />
            <span>{$t('security.changePassword')}</span>
          </button>

          <div class="h-px bg-border my-1"></div>
          <button
            type="button"
            class="flex w-full items-center gap-2 rounded-sm px-3 py-2 text-left text-xs hover:bg-accent"
            on:click={handleLock}
          >
            <Icon icon={mdiLockOutline} width={16} height={16} class="shrink-0 opacity-80" />
            <span>{$t('nav.lock')}</span>
          </button>
        </div>
      {/if}
    </div>
  {/if}
</div>
