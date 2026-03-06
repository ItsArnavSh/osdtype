<script lang="ts">
	export interface GameConfig {
		Time: number;
		Language: string;
		Editable: boolean;
	}

	interface Props {
		config?: GameConfig;
	}

	let { config = { Time: 60, Language: 'CPP', Editable: true } }: Props = $props();

	let time = $state(config.Time);
	let language = $state(config.Language);

	$effect(() => {
		config.Time = time;
		config.Language = language;
	});

	// Options
	const timeOptions = [30, 60, 90, 120, 300];
	const languageOptions = ['CPP', 'Rust', 'Python', 'JavaScript', 'TypeScript', 'Go'];

	// Modal state
	let showModal = $state(false);
	let modalPosition = $state({ x: 0, y: 0 });
	let modalType = $state<'time' | 'language'>('time');

	function handleTimeClick(event: MouseEvent) {
		if (config.Editable) {
			event.stopPropagation();
			modalPosition = { x: event.clientX, y: event.clientY };
			modalType = 'time';
			showModal = true;
		}
	}

	function handleLanguageClick(event: MouseEvent) {
		if (config.Editable) {
			event.stopPropagation();
			modalPosition = { x: event.clientX, y: event.clientY };
			modalType = 'language';
			showModal = true;
		}
	}

	function closeModal() {
		showModal = false;
	}

	function selectTime(newTime: number) {
		time = newTime;
		closeModal();
	}

	function selectLanguage(newLanguage: string) {
		language = newLanguage;
		closeModal();
	}
</script>

<div class="flex gap-8">
	<div
		class="flex flex-col {config.Editable ? 'cursor-pointer' : ''}"
		onclick={handleTimeClick}
		role={config.Editable ? 'button' : undefined}
		tabindex={config.Editable ? 0 : undefined}
	>
		<p class="fira-code text-4xl text-(--mer)">{time}</p>
		<p class="fira-code m-0 p-0 text-(--silver)">Timer</p>
	</div>
	<div
		class="flex flex-col {config.Editable ? 'cursor-pointer' : ''}"
		onclick={handleLanguageClick}
		role={config.Editable ? 'button' : undefined}
		tabindex={config.Editable ? 0 : undefined}
	>
		<p class="fira-code text-3xl font-bold text-(--mer)">{language}</p>
		<p class="fira-code text-(--silver)">Language</p>
	</div>
</div>

<!-- Minimal Modal -->
{#if showModal}
	<div class="fixed inset-0 z-40" onclick={closeModal} role="button" tabindex="-1" />
	<div
		class="fira-code fixed z-50 rounded border border-(--silver) bg-(--fbg) p-2 shadow-lg"
		style="left: {modalPosition.x}px; top: {modalPosition.y}px;"
	>
		{#if modalType === 'time'}
			{#each timeOptions as time}
				<button
					class="w-full px-4 py-2 text-left text-sm text-(--silver) transition-colors hover:bg-(--bg) hover:text-(--red)"
					onclick={() => selectTime(time)}
				>
					{time}s
				</button>
			{/each}
		{:else}
			{#each languageOptions as language}
				<button
					class="w-full px-4 py-2 text-left text-sm text-(--silver) transition-colors hover:bg-(--bg) hover:text-(--red)"
					onclick={() => selectLanguage(language)}
				>
					{language}
				</button>
			{/each}
		{/if}
	</div>
{/if}
