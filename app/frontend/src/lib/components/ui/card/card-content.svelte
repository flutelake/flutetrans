<script>
  import {cn} from '$lib/utils/cn.js'

  export let className = ''

  $: computed = cn('p-6 pt-0', className)
</script>

<div class={computed} {...$$restProps}>
  <slot />
</div>

