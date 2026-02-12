<script>
  import {cn} from '$lib/utils/cn.js'

  export let className = ''

  $: computed = cn('text-lg font-semibold leading-none tracking-tight', className)
</script>

<div class={computed} {...$$restProps}>
  <slot />
</div>

