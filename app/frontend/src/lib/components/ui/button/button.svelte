<script>
  import {cn} from '$lib/utils/cn.js'
  import {createEventDispatcher} from 'svelte'

  export let variant = 'default'
  export let size = 'default'
  export let className = ''

  const variantClasses = {
    default: 'bg-primary text-primary-foreground hover:bg-primary/90',
    secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/80',
    outline: 'border border-border bg-transparent hover:bg-accent hover:text-accent-foreground',
    ghost: 'hover:bg-accent hover:text-accent-foreground',
    destructive: 'bg-destructive text-destructive-foreground hover:bg-destructive/90'
  }

  const sizeClasses = {
    default: 'h-10 px-4 py-2',
    sm: 'h-9 rounded-md px-3',
    lg: 'h-11 rounded-md px-8',
    icon: 'h-10 w-10'
  }

  $: computed = cn(
    'inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50',
    variantClasses[variant] ?? variantClasses.default,
    sizeClasses[size] ?? sizeClasses.default,
    className
  )

  const dispatch = createEventDispatcher()

  function handleClick(event) {
    dispatch('click', event)
  }
</script>

<button class={computed} on:click={handleClick} {...$$restProps}>
  <slot />
</button>
