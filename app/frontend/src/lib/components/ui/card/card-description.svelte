<script>
  import {cn} from '$lib/utils/cn.js'

  export let className = ''

  $: computed = cn('text-sm text-muted-foreground', className)
</script>

<div class={computed} {...$$restProps}>
  <slot />
</div>

