<script lang="ts">
	type ModeIconName = 'ranked' | 'rooms' | 'practice' | 'zen' | 'sprint' | 'standard' | 'marathon';

	let { name, class: className = '' }: { name: ModeIconName; class?: string } = $props();
</script>

<svg
	class={className}
	viewBox="0 0 64 64"
	fill="none"
	stroke="currentColor"
	stroke-width="2.5"
	stroke-linecap="round"
	stroke-linejoin="round"
	aria-hidden="true"
>
	{#if name === 'ranked'}
		<path d="M20 11h24v13c0 8.3-5.4 13.5-12 13.5S20 32.3 20 24V11Z" />
		<path d="M20 16h-8v5c0 5.2 3.6 8.4 8.6 8.4" />
		<path d="M44 16h8v5c0 5.2-3.6 8.4-8.6 8.4" />
		<path d="M32 37.5V47" />
		<path d="M22 53h20" />
		<path d="M26 47h12" />
		<path
			d="m32 17 1.7 3.6 3.9.5-2.9 2.7.8 3.9-3.5-2-3.5 2 .8-3.9-2.9-2.7 3.9-.5Z"
			fill="currentColor"
			stroke="none"
			opacity="0.65"
		/>
	{:else if name === 'rooms'}
		<circle cx="32" cy="15" r="6.5" />
		<circle cx="15" cy="47" r="6.5" />
		<circle cx="49" cy="47" r="6.5" />
		<path d="m27.3 20.6-8.6 20" />
		<path d="m36.7 20.6 8.6 20" />
		<path d="M21.5 47h21" />
		<circle cx="32" cy="34" r="2.2" fill="currentColor" stroke="none" opacity="0.55" />
	{:else if name === 'practice'}
		<rect x="7" y="18" width="50" height="29" rx="3" />
		<path d="M13 26h2M21 26h2M29 26h2M37 26h2M45 26h2M51 26h1" />
		<path d="M13 33h2M21 33h2M29 33h2M37 33h2M45 33h2M51 33h1" />
		<path d="M18 40h28" />
	{:else if name === 'zen'}
		<circle cx="32" cy="32" r="22" opacity="0.45" />
		<path d="M13 30c6.3-7.5 12.7-7.5 19 0s12.7 7.5 19 0" />
		<path d="M13 39c6.3-7.5 12.7-7.5 19 0s12.7 7.5 19 0" opacity="0.55" />
	{:else if name === 'sprint'}
		<path d="M37 7 19 36h12l-4 21 18-30H33l4-20Z" />
	{:else if name === 'standard'}
		<path d="M11 45a21 21 0 0 1 42 0" />
		<path d="M32 45 43 31" />
		<circle cx="32" cy="45" r="2.5" fill="currentColor" stroke="none" />
	{:else if name === 'marathon'}
		<path d="m6 50 16-24 9 13 7-9 20 20" />
		<path d="M6 50h52" opacity="0.5" />
		<path d="m22 26 4-7 4 7" opacity="0.6" />
	{/if}
</svg>
