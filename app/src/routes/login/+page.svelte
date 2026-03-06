<script lang="ts">
	import { resolve } from '$app/paths';
	import { goto } from '$app/navigation';
	import { connect, fakeLogin, ping } from '$lib/core/api';
	import { onMount } from 'svelte';
	let name = $state('');
	let connected = $state(false);
	onMount(async () => {
		let res = await ping();
		if (res == 'pong') {
			connected = true;
		}
	});
	async function login() {
		await fakeLogin(name);
		await connect();
		goto(resolve('/play'));
	}
</script>

<div class="flex min-h-screen flex-col items-center justify-center bg-(--bg) p-8 text-(--silver)">
	{#if connected}
		<div>Connection to server esstablished</div>
	{/if}
	<input bind:value={name} class="m-y-10 border-2 border-(--mer)" />
	<button class="text-(--silver)" onclick={() => login()}>Login</button>
</div>
