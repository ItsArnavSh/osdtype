<script lang="ts">
	import type { Confi } from '../../api/snippet';
	export let config: Confi;
	let language = {
		id: 'typescript',
		name: 'TypeScript'
	};

	const languages = [
		{ id: 'typescript', name: 'TypeScript' },
		{ id: 'python', name: 'Python' },
		{ id: 'java', name: 'Java' },
		{ id: 'cpp', name: 'C++' },
		{ id: 'go', name: 'Go' }
	];

	let isHovering = false;

	function selectLanguage(lang) {
		config.lang = lang.name;
		// Don't close immediately, let the hover state handle it
	}

	function getLogoUrl(langId) {
		return `https://raw.githubusercontent.com/abranhe/programming-languages-logos/master/src/${langId}/${langId}_24x24.png`;
	}
</script>

<div class="mt-10 flex flex-col items-end justify-end space-y-2 font-mono text-white">
	<!-- Language Selector -->
	<div
		class="relative"
		on:mouseenter={() => (isHovering = true)}
		on:mouseleave={() => (isHovering = false)}
	>
		{#if isHovering}
			<div class="logo-container right-0 bottom-full mb-2">
				<div class="flex flex-col gap-1">
					<div class="flex justify-end gap-2">
						{#each languages.slice(0, 6) as lang, index}
							<button
								class="fade-in-logo rounded p-1 transition-all duration-200 hover:scale-110 hover:bg-gray-700"
								style="animation-delay: {index * 30}ms;"
								class:ring-2={lang.id === language.id}
								class:ring-white={lang.id === language.id}
								class:ring-opacity-50={lang.id === language.id}
								on:click={() => selectLanguage(lang)}
								title={lang.name}
							>
								<img src={getLogoUrl(lang.id)} alt={lang.name} class="h-6 w-6" />
							</button>
						{/each}
					</div>
					<div class="flex justify-end gap-2">
						{#each languages.slice(6) as lang, index}
							<button
								class="fade-in-logo rounded p-1 transition-all duration-200 hover:scale-110 hover:bg-gray-700"
								style="animation-delay: {(index + 6) * 30}ms;"
								class:ring-2={lang.id === language.id}
								class:ring-white={lang.id === language.id}
								class:ring-opacity-50={lang.id === language.id}
								on:click={() => selectLanguage(lang)}
								title={lang.name}
							>
								<img src={getLogoUrl(lang.id)} alt={lang.name} class="h-6 w-6" />
							</button>
						{/each}
					</div>
				</div>
			</div>
		{/if}

		<div
			class="cursor-pointer text-right text-4xl font-bold tracking-wide opacity-40 transition-opacity hover:opacity-60"
		>
			{config.lang}
		</div>
	</div>
	<div class="text-sm tracking-widest uppercase">Language</div>
</div>

<style>
	@keyframes logoContainerAppear {
		from {
			opacity: 0;
			transform: translateY(10px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	@keyframes fadeInLogo {
		from {
			opacity: 0;
			transform: scale(0.8);
		}
		to {
			opacity: 1;
			transform: scale(1);
		}
	}

	.logo-container {
		animation: logoContainerAppear 0.2s ease-out forwards;
	}

	.fade-in-logo {
		opacity: 0;
		animation: fadeInLogo 0.3s ease-out forwards;
	}
</style>
