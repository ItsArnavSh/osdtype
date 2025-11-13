<script lang="ts">
	import '../app.css';

	let { children } = $props();
</script>

<svelte:head></svelte:head>

{@render children()}
