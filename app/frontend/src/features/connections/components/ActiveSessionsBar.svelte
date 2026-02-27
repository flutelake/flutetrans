<script>
  import {Button} from '$lib/components/ui/button/index.js'
  import {Select} from '$lib/components/ui/select/index.js'
  import {locale, localeOptions, t} from '$lib/i18n/index.js'
  import {cn} from '$lib/utils/cn.js'

  export let sessions = []
  export let current = 'connections'
  export let onSelect = (_id) => {}
  export let locked = false
  export let canLock = false
  export let onLock = () => {}

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

  <div class="h-6 w-px bg-border"></div>
  <Select className="h-8 w-[120px]" bind:value={$locale} aria-label={$t('nav.language')}>
    {#each localeOptions as opt (opt.id)}
      <option value={opt.id}>{opt.label}</option>
    {/each}
  </Select>

  {#if canLock && !locked}
    <div class="h-6 w-px bg-border"></div>
    <Button size="sm" variant="outline" on:click={onLock}>{$t('nav.lock')}</Button>
  {/if}
</div>
