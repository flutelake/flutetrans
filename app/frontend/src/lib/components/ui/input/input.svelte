<script>
  import {cn} from '$lib/utils/cn.js'
  import {createEventDispatcher} from 'svelte'

  export let value = ''
  export let className = ''

  $: computed = cn(
    'flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50',
    className
  )

  const dispatch = createEventDispatcher()

  function handleInput(event) {
    dispatch('input', event)
  }

  function handleChange(event) {
    dispatch('change', event)
  }
</script>

<input class={computed} bind:value on:input={handleInput} on:change={handleChange} {...$$restProps} />
