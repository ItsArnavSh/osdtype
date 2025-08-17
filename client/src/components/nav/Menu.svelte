<script lang="ts">
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';

	// --- State Variables ---
	let isLoading = true;
	let user: { login: string } | null = null;
	let avatarUrl = '';
	let isMenuOpen = false;
	let closeTimer: number; // Timer for the menu closing delay

	// --- Constants ---
	const GITHUB_LOGIN_URL = 'http://localhost:8080/login/github';
	const API_ME_URL = 'http://localhost:8080/api/me';
	const LOGOUT_URL = 'http://localhost:8080/logout';

	function getGithubAvatarUrl(username: string, size: number = 40): string {
		if (!username) return '';
		return `https://github.com/${username}.png?size=${size}`;
	}

	// --- Menu Logic with Delay ---
	function handleMenuEnter() {
		// When mouse enters the avatar or menu, cancel any pending close actions
		clearTimeout(closeTimer);
		isMenuOpen = true;
	}

	function handleMenuLeave() {
		// When mouse leaves, start a timer to close the menu
		// This gives the user time to move the cursor to the menu
		closeTimer = setTimeout(() => {
			isMenuOpen = false;
		}, 150); // 150ms delay
	}

	onMount(async () => {
		isLoading = true;
		try {
			const response = await fetch(API_ME_URL, { credentials: 'include' });
			if (response.ok) {
				user = await response.json();
				if (user) {
					avatarUrl = getGithubAvatarUrl(user.login);
				}
			} else {
				user = null;
			}
		} catch (error) {
			console.error('Authentication check failed:', error);
			user = null;
		} finally {
			isLoading = false;
		}
	});

	async function handleLogout() {
		try {
			await fetch(LOGOUT_URL, { method: 'POST', credentials: 'include' });
		} catch (error) {
			console.error('Logout request failed:', error);
		} finally {
			user = null;
			avatarUrl = '';
			isMenuOpen = false;
		}
	}
</script>

<nav
	class="fixed top-5 right-[2%] z-40 flex flex-1 flex-row items-center justify-center space-x-6 p-5 font-sans tracking-widest text-gray-300"
>
	{#if isLoading}
		<div class="h-10 w-10 animate-pulse rounded-full bg-gray-700" />
	{:else if user && avatarUrl}
		<!-- The container now handles the enter/leave logic for both avatar and menu -->
		<div class="relative" on:mouseenter={handleMenuEnter} on:mouseleave={handleMenuLeave}>
			<img
				src={avatarUrl}
				alt="User Avatar"
				class="h-10 w-10 cursor-pointer rounded-full ring-2 ring-transparent transition-all hover:ring-white"
			/>
			{#if isMenuOpen}
				<!-- The menu itself is now part of the hover zone -->
				<div
					transition:fly={{ y: 10, duration: 200 }}
					class="ring-opacity-20 absolute top-full right-0 mt-3 w-48 origin-top-right rounded-md bg-[#181a1b] text-gray-300 shadow-lg ring-1 ring-black"
				>
					<div
						class="border-b border-gray-700 px-4 py-3 text-sm font-bold tracking-wider text-white uppercase"
					>
						{user.login}
					</div>
					<div class="py-1">
						<a
							href="/"
							class="block px-4 py-2 text-sm transition-colors hover:bg-gray-700/50 hover:text-white"
						>
							Rooms
						</a>
						<button
							on:click={handleLogout}
							class="block w-full px-4 py-2 text-left text-sm transition-colors hover:bg-gray-700/50 hover:text-white"
						>
							Logout
						</button>
					</div>
				</div>
			{/if}
		</div>
	{:else}
		<a href={GITHUB_LOGIN_URL} class="transition-colors hover:text-white">Login</a>
	{/if}
</nav>
