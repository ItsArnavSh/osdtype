<script>
	let loggedIn = false; // for now assume logged out
</script>

<nav
	class="fixed top-20 right-[2%] z-40 flex flex-col items-end space-y-3 rounded-xl border border-[#3d4459]/30 bg-[#2a2f42]/90 p-5 shadow-xl backdrop-blur-sm"
>
	{#if !loggedIn}
		<a
			href="/login"
			class="group relative rounded-lg px-3 py-2 transition-all duration-300 hover:bg-[#343854]/50"
		>
			<span
				class="relative z-10 text-sm tracking-[0.2em] text-gray-300 uppercase transition-colors duration-300 group-hover:text-white"
			>
				Login
			</span>
			<span
				class="absolute bottom-0 left-0 h-[2px] w-0 bg-blue-400 transition-all duration-300 group-hover:w-full"
			></span>
		</a>
	{/if}

	<a
		href="/rooms"
		class="group relative rounded-lg px-3 py-2 transition-all duration-300 hover:bg-[#343854]/50"
	>
		<span
			class="relative z-10 text-sm tracking-[0.2em] text-gray-300 uppercase transition-colors duration-300 group-hover:text-white"
		>
			Rooms
		</span>
		<span
			class="absolute bottom-0 left-0 h-[2px] w-0 bg-green-400 transition-all duration-300 group-hover:w-full"
		></span>
	</a>

	<a
		href="/snippet"
		class="group relative rounded-lg px-3 py-2 transition-all duration-300 hover:bg-[#343854]/50"
	>
		<span
			class="relative z-10 text-sm tracking-[0.2em] text-gray-300 uppercase transition-colors duration-300 group-hover:text-white"
		>
			Snippet of the Day
		</span>
		<span
			class="absolute bottom-0 left-0 h-[2px] w-0 bg-orange-400 transition-all duration-300 group-hover:w-full"
		></span>
	</a>
</nav>
