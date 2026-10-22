<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { initTheme } from '$lib/core/theme.svelte';
	import { page } from '$app/state';
	import { onMount } from 'svelte';

	let { children } = $props();

	onMount(() => initTheme());
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<meta property="og:image" content="{page.url.origin}/og.png" />
	<meta property="og:url" content={page.url.href} />
	<meta name="twitter:image" content="{page.url.origin}/og.png" />
</svelte:head>
{@render children()}
