<script>
  import {cn} from '$lib/utils/cn.js'

  export let className = ''

  $: computed = cn('flex flex-col space-y-1.5 p-6', className)
</script>

<div class={computed} {...$$restProps}>
  <slot />
</div>

