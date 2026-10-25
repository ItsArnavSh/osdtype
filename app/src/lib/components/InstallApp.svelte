<script lang="ts">
	import { onMount } from 'svelte';

	interface BeforeInstallPromptEvent extends Event {
		prompt(): Promise<void>;
		userChoice: Promise<{ outcome: 'accepted' | 'dismissed'; platform: string }>;
	}

	let promptEvent = $state<BeforeInstallPromptEvent | null>(null);
	let installed = $state(false);

	onMount(() => {
		const onPrompt = (event: Event) => {
			event.preventDefault();
			promptEvent = event as BeforeInstallPromptEvent;
		};
		const onInstalled = () => {
			installed = true;
			promptEvent = null;
		};

		installed = window.matchMedia('(display-mode: standalone)').matches;
		window.addEventListener('beforeinstallprompt', onPrompt);
		window.addEventListener('appinstalled', onInstalled);

		return () => {
			window.removeEventListener('beforeinstallprompt', onPrompt);
			window.removeEventListener('appinstalled', onInstalled);
		};
	});

	async function install() {
		const event = promptEvent;
		if (!event) return;

		await event.prompt();
		const choice = await event.userChoice;
		if (choice.outcome === 'accepted') installed = true;
		promptEvent = null;
	}
</script>

{#if promptEvent && !installed}
	<span class="md:hidden">
		<button
			class="ui-btn ui-btn-sm ui-btn-primary"
			onclick={install}
			title="Install OSDTYP on this device">install app</button
		>
	</span>
{/if}
