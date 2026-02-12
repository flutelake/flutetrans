<script>
  import {cn} from '$lib/utils/cn.js'

  export let orientation = 'horizontal'
  export let className = ''

  $: computed = cn(
    'shrink-0 bg-border',
    orientation === 'vertical' ? 'h-full w-px' : 'h-px w-full',
    className
  )
</script>

<div class={computed} {...$$restProps}></div>
