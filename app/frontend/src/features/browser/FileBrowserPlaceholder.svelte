<script>
  import {Button} from '$lib/components/ui/button/index.js'
  import {Card, CardContent, CardDescription, CardHeader, CardTitle} from '$lib/components/ui/card/index.js'

  export let session
  export let onDisconnect = (_sessionID) => {}
</script>

<div class="h-full min-h-0 flex flex-col">
  <Card className="flex-1 min-h-0 flex flex-col">
    <CardHeader className="flex-row items-start justify-between space-y-0">
      <div class="space-y-1 text-left">
        <CardTitle className="text-base">File Browser</CardTitle>
        <CardDescription>Placeholder view. Session is ready for integration.</CardDescription>
      </div>
      {#if session}
        <Button variant="secondary" size="sm" on:click={() => onDisconnect(session.sessionID)}>Disconnect</Button>
      {/if}
    </CardHeader>
    <CardContent className="flex-1 min-h-0">
      {#if !session}
        <div class="rounded-md border border-dashed border-border p-6 text-left">
          <div class="text-sm font-medium">No active session</div>
          <div class="mt-1 text-sm text-muted-foreground">Connect to a server to open the browser.</div>
        </div>
      {:else}
        <div class="space-y-2 text-left text-sm">
          <div class="flex items-center justify-between rounded-md border border-border px-3 py-2">
            <div class="text-muted-foreground">Session</div>
            <div class="font-mono">{session.sessionID}</div>
          </div>
          <div class="flex items-center justify-between rounded-md border border-border px-3 py-2">
            <div class="text-muted-foreground">Status</div>
            <div class="font-medium">{session.status}</div>
          </div>
          {#if session.message}
            <div class="rounded-md border border-border px-3 py-2 text-muted-foreground">{session.message}</div>
          {/if}
        </div>
      {/if}
    </CardContent>
  </Card>
</div>

