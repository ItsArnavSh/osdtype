<script lang="ts">
	import { fakeLogin, getUserInfo, OSDTypeSession, ping } from '$lib/core/api';
	import { onMount } from 'svelte';
	onMount(async () => {
		console.log(ping());
		fakeLogin('Arnav');
		console.log(getUserInfo('Arnav'));
		let session = new OSDTypeSession();
		console.log(session.connect());
		session.disconnect();
	});
</script>

<div>Home Page of OSDType</div>
Todo
