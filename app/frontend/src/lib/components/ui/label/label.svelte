<script>
  import {cn} from '$lib/utils/cn.js'

  export let className = ''

  $: computed = cn('text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70', className)
</script>

<label class={computed} {...$$restProps}>
  <slot />
</label>

