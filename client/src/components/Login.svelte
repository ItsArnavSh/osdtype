<script lang="ts">
	import type { User } from '$lib/core/entity/user';
	import { login } from '$lib/core/login';
	import { whoami } from '$lib/core/userdata';
	import { GameComm } from '$lib/core/wshandler';
	import { BASE_URL } from '$lib/config';
	import { onMount, onDestroy } from 'svelte';

	let logged_in = $state(false);
	let { userdata = $bindable<User>() } = $props();
	let username = '';
	let ws: GameComm | null = null;
	let connected = $state(false); // Use $state for reactivity

	// Chat-specific state
	let input = $state('');
	let messages = $state<{ from: string; text: string }[]>([]);

	onMount(async () => {
		const current = await whoami();
		if (current) {
			userdata = current;
			logged_in = true;
			await initWebSocket();
		}
	});

	onDestroy(() => {
		if (ws) ws.disconnect();
	});

	async function handleLogin() {
		if (!username.trim()) {
			alert('Please enter a username');
			return;
		}
		await login(username);
		const current = await whoami();
		if (current) {
			userdata = current;
			logged_in = true;
			await initWebSocket();
		}
	}

	async function initWebSocket() {
		ws = new GameComm();

		try {
			await ws.connect(true);
			connected = true; // This will now trigger reactivity
			console.log('✅ Connected via WebSocket');
		} catch (err) {
			console.error('❌ WebSocket connect failed:', err);
			connected = false;
			return;
		}

		const handleMessage = (data: any) => {
			console.log('📨 Received message:', data);
			if (data?.text) {
				messages = [...messages, { from: data.from || 'Server', text: data.text }];
			}
		};

		ws.onMessage(handleMessage);
	}

	function sendMessage() {
		if (!input.trim() || !ws || !connected) return;

		const msg = { from: userdata.Username, text: input.trim() };
		ws.send(msg);
		messages = [...messages, msg];
		input = '';
	}

	async function joinLobby() {
		try {
			const res = await fetch(`${BASE_URL}/user/join-lobby?duration=30`, {
				method: 'GET',
				credentials: 'include'
			});
			if (!res.ok) {
				throw new Error('Failed to join lobby');
			}
			const data = await res.json();
			console.log('🎮 Joined lobby:', data);
			alert('Successfully joined lobby!');
		} catch (err) {
			console.error('❌ Failed to join lobby:', err);
			alert('Failed to join lobby');
		}
	}
</script>

<div class="flex w-full flex-col items-center justify-center space-y-4 p-4">
	{#if !logged_in}
		<div class="flex flex-col items-center space-y-2">
			<input
				bind:value={username}
				type="text"
				placeholder="Enter username"
				class="rounded border border-gray-300 px-3 py-2 focus:ring focus:ring-blue-400 focus:outline-none"
			/>
			<button
				onclick={handleLogin}
				class="rounded bg-blue-600 px-4 py-2 text-white transition hover:bg-blue-700"
			>
				Login
			</button>
		</div>
	{:else}
		<div class="flex w-full max-w-md flex-col items-center space-y-2">
			<div class="flex flex-col items-center space-y-2">
				<img
					src={userdata.AvatarURL}
					alt={userdata.Username}
					class="h-24 w-24 rounded-full object-cover"
				/>
				<p class="text-lg font-semibold">{userdata.Username}</p>
			</div>

			<!-- Chat UI -->
			<h2 class="mt-4 text-xl font-semibold">
				{connected ? '💬 WebSocket Chat' : '⏳ Connecting...'}
			</h2>

			<div class="h-64 w-full overflow-y-auto rounded-md border bg-gray-50 p-2 shadow-inner">
				{#each messages as m}
					<div class="my-1">
						<strong>{m.from}:</strong>
						{m.text}
					</div>
				{/each}
			</div>

			<div class="mt-2 flex w-full">
				<input
					bind:value={input}
					onkeydown={(e) => e.key === 'Enter' && sendMessage()}
					class="flex-grow rounded-l-md border p-2 outline-none"
					placeholder="Type a message..."
				/>
				<button
					onclick={sendMessage}
					class="rounded-r-md bg-blue-600 px-4 text-white hover:bg-blue-700"
					disabled={!connected}
				>
					Send
				</button>
			</div>

			<button
				onclick={joinLobby}
				class="mt-4 rounded bg-green-600 px-6 py-2 text-white transition hover:bg-green-700"
			>
				🎮 Join Lobby
			</button>
		</div>
	{/if}
</div>
