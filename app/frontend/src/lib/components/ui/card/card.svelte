<script>
  import {cn} from '$lib/utils/cn.js'

  export let className = ''

  $: computed = cn('rounded-lg border border-border bg-card text-card-foreground shadow-sm', className)
</script>

<div class={computed} {...$$restProps}>
  <slot />
</div>

