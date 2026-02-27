<script>
  import {Button} from '$lib/components/ui/button/index.js'
  import {Card, CardContent, CardHeader, CardTitle} from '$lib/components/ui/card/index.js'
  import {t} from '$lib/i18n/index.js'

  export let open = false
  export let title = ''
  export let error = null
  export let onClose = () => {}
  export let onRetry = null

  let copied = false

  $: resolvedTitle = title || $t('common.details')
  $: text = formatErrorText(error)

  function formatErrorText(err) {
    if (err == null) return ''
    if (typeof err === 'string') return err
    try {
      return JSON.stringify(err, null, 2)
    } catch {
      return String(err)
    }
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(text)
      copied = true
      setTimeout(() => {
        copied = false
      }, 1200)
    } catch {
      copied = false
    }
  }
</script>

{#if open}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-6">
    <button
      type="button"
      class="absolute inset-0 bg-background/70 backdrop-blur-sm"
      aria-label={$t('common.close')}
      on:click={onClose}
    ></button>

    <Card className="relative w-full max-w-2xl">
      <CardHeader className="flex-row items-center justify-between space-y-0">
        <CardTitle className="text-base">{resolvedTitle}</CardTitle>
        <div class="flex items-center gap-2">
          {#if onRetry}
            <Button size="sm" variant="secondary" on:click={onRetry}>{$t('common.retry')}</Button>
          {/if}
          <Button size="sm" variant="outline" on:click={copy} disabled={!text}>{copied ? $t('common.copied') : $t('common.copy')}</Button>
          <Button size="sm" variant="ghost" on:click={onClose}>{$t('common.close')}</Button>
        </div>
      </CardHeader>
      <CardContent>
        <pre class="max-h-[60vh] overflow-auto rounded-md border border-border bg-muted/40 p-3 text-left text-xs">{text}</pre>
      </CardContent>
    </Card>
  </div>
{/if}
