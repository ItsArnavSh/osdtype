<script>
	import LanguageSel from '../typing/LanguageSel.svelte';
	import Timer from '../typing/Timer.svelte';
</script>

<div class="flex flex-1 flex-row justify-between">
	<Timer time={30} />
	<LanguageSel />
</div>
