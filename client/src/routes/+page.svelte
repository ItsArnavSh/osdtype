<script lang="ts">
	import type { User } from '$lib/core/entity/user';
	import Login from '../components/Login.svelte';
	let temp: User = { ID: '', Username: '', AvatarURL: '', CurrentRank: 0 };
	let user = $state(temp);
</script>

<Login bind:userdata={user} />
{user.Username}
