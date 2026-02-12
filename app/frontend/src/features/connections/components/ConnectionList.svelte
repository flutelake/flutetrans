<script>
  import {cn} from '$lib/utils/cn.js'
  import {Button} from '$lib/components/ui/button/index.js'
  import {Card, CardContent, CardDescription, CardHeader, CardTitle} from '$lib/components/ui/card/index.js'
  import {onMount} from 'svelte'

  export let items = []
  export let selectedId = null
  export let onSelect = (_id) => {}
  export let onConnect = (_id) => {}
  export let onEdit = (_id) => {}
  export let onDelete = (_id) => {}

  let openMenuFor = null

  function toggleMenu(id) {
    openMenuFor = openMenuFor === id ? null : id
  }

  function closeMenu() {
    openMenuFor = null
  }

  function handleEdit(id) {
    closeMenu()
    onEdit(id)
  }

  function handleDelete(id) {
    closeMenu()
    onDelete(id)
  }

  function handleDocumentClick(event) {
    const path = event?.composedPath?.() ?? []
    for (const el of path) {
      if (el instanceof Element && el.hasAttribute('data-conn-actions')) {
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

<Card className="h-full flex flex-col">
  <CardHeader className="flex-row items-center justify-between space-y-0">
    <div class="space-y-1 text-left">
      <CardTitle className="text-base">Connections</CardTitle>
      <CardDescription>Saved connection profiles.</CardDescription>
    </div>
  </CardHeader>

  <CardContent className="flex-1 min-h-0">
    {#if items.length === 0}
      <div class="rounded-md border border-dashed border-border p-6 text-left">
        <div class="text-sm font-medium">No connections</div>
        <div class="mt-1 text-sm text-muted-foreground">Create your first connection to get started.</div>
      </div>
    {:else}
      <div class="h-full min-h-0 overflow-auto pr-1">
        <div class="space-y-2">
        {#each items as item (item.id)}
          <div
            class={cn(
              'flex items-center justify-between gap-3 rounded-md border border-border bg-background/20 px-3 py-2 transition-colors',
              item.id === selectedId ? 'bg-accent' : 'hover:bg-accent/60'
            )}
          >
            <button
              class={cn(
                'min-w-0 flex-1 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-sm',
                item.id === selectedId ? '' : ''
              )}
              type="button"
              on:click={() => onSelect(item.id)}
              on:dblclick={() => onConnect(item.id)}
            >
              <div class="text-sm font-medium truncate">{item.name}</div>
              <div class="text-xs text-muted-foreground truncate">{item.protocol} · {item.host}{item.port ? `:${item.port}` : ''}</div>
            </button>

            <div class="flex items-center gap-2">
              <Button variant="outline" size="sm" on:click={() => onConnect(item.id)}>Connect</Button>
              <div class="relative" data-conn-actions>
                <Button
                  variant="ghost"
                  size="icon"
                  aria-haspopup="menu"
                  aria-expanded={openMenuFor === item.id}
                  on:click={() => toggleMenu(item.id)}
                >
                  ⋯
                </Button>

                {#if openMenuFor === item.id}
                  <div class="absolute right-0 top-full z-10 mt-1 w-36 rounded-md border border-border bg-background shadow-md">
                    <button
                      type="button"
                      class="flex w-full items-center gap-2 rounded-sm px-3 py-2 text-left text-xs hover:bg-accent"
                      on:click={() => handleEdit(item.id)}
                    >
                      <svg
                        class="h-4 w-4 text-muted-foreground"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        aria-hidden="true"
                      >
                        <path d="M12 20h9" />
                        <path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z" />
                      </svg>
                      <span>编辑</span>
                    </button>
                    <button
                      type="button"
                      class="flex w-full items-center gap-2 rounded-sm px-3 py-2 text-left text-xs text-destructive hover:bg-accent"
                      on:click={() => handleDelete(item.id)}
                    >
                      <svg
                        class="h-4 w-4"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        aria-hidden="true"
                      >
                        <path d="M3 6h18" />
                        <path d="M8 6V4h8v2" />
                        <path d="M19 6l-1 14H6L5 6" />
                        <path d="M10 11v6" />
                        <path d="M14 11v6" />
                      </svg>
                      <span>删除</span>
                    </button>
                  </div>
                {/if}
              </div>
            </div>
          </div>
        {/each}
        </div>
      </div>
    {/if}
  </CardContent>
</Card>
