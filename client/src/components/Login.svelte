<script lang="ts">
	import type { User } from '$lib/core/entity/user';
	import { login } from '$lib/core/login';
	import { imonline, whoami } from '$lib/core/userdata';
	import { GameComm } from '$lib/core/wshandler';
	import { onMount } from 'svelte';
	import GameComponent from './GameComponent.svelte';

	let logged_in = $state(false);
	let { userdata = $bindable<User>() } = $props();

	onMount(async () => {
		const current = await whoami();
		if (current) {
			userdata = current;
			logged_in = true;
			let ws: GameComm = new GameComm();
			ws.connect(true);
		}
	});

	async function handleLogin() {
		await login('itsarnavsh');
		const current = await whoami();
		if (current) {
			userdata = current;
			logged_in = true;
		}
	}
</script>

<div class="flex flex-col items-center justify-center p-4">
	{#if logged_in}
		<img
			src={userdata.AvatarURL}
			alt={userdata.Username}
			class="h-24 w-24 rounded-full object-cover"
		/>
	{/if}

	{#if !logged_in}
		<button onclick={handleLogin} class=""> Login </button>
	{/if}
</div>
