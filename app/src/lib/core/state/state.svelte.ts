import { fetchState } from '$lib/core/api/state';
import { me } from '$lib/core/api/auth';
import { onSocketEvent, sendSocket } from '$lib/core/api/socket';
import type { InviteView, StateView, UserView } from '$lib/core/api/types';

export const appState = $state<{
	booted: boolean;
	user: UserView | null;
	state: StateView | null;
	invites: InviteView[];
	error: string;
	notice: string;
}>({
	booted: false,
	user: null,
	state: null,
	invites: [],
	error: '',
	notice: ''
});

export function applyState(next: StateView): void {
	appState.state = next;
	appState.user = next.user;
	appState.invites = next.invites ?? [];
	appState.error = '';
}

export async function refreshState(): Promise<StateView | null> {
	try {
		const next = await fetchState();
		applyState(next);
		return next;
	} catch (err) {
		appState.error = err instanceof Error ? err.message : 'Could not load state';
		return null;
	}
}

let bootPromise: Promise<void> | null = null;

export async function bootSession(): Promise<void> {
	if (bootPromise) return bootPromise;
	bootPromise = (async () => {
		const user = await me();
		if (!user) {
			appState.user = null;
			appState.state = null;
			appState.booted = true;
			return;
		}
		appState.user = user;
		await refreshState();
		appState.booted = true;
	})();
	return bootPromise;
}

export function resetSession(): void {
	bootPromise = null;
	appState.user = null;
	appState.state = null;
	appState.invites = [];
	appState.booted = true;
}

// A single global listener keeps invites and state fresh even when a page
// forgets to subscribe. Page-local listeners handle game/room detail events.
let wired = false;
export function wireGlobalSocket(): () => void {
	if (wired) return () => {};
	wired = true;
	const off = onSocketEvent((event) => {
		switch (event.type) {
			case 'state':
				applyState(event.data as StateView);
				break;
			case 'socket_open':
				// Ask for authoritative state after a reconnect.
				void refreshState();
				break;
			case 'invite':
				if (event.data) {
					const invite = event.data as InviteView;
					if (!appState.invites.some((i) => i.id === invite.id)) {
						appState.invites = [invite, ...appState.invites];
					}
					appState.notice = `${invite.from} invited you to ${invite.room_name}`;
				}
				break;
			case 'room_closed':
				appState.notice = 'The room was closed';
				void refreshState();
				break;
			case 'game_start':
				// Non-hosts learn about the start from this event; the state
				// endpoint is authoritative and will route them to the board.
				void refreshState();
				break;
			case 'room_left':
				appState.notice = event.data
					? `Removed from room (${(event.data as { reason?: string }).reason ?? 'left'})`
					: 'Left room';
				void refreshState();
				break;
			case 'left':
				void refreshState();
				break;
		}
	});
	return () => {
		wired = false;
		off();
	};
}

export function inviteCount(): number {
	return appState.invites.length;
}

export function currentActive(): string {
	const state = appState.state?.state ?? 'idle';
	if (state === 'game') return `/game/${appState.state?.game?.id}`;
	if (state === 'results') return `/results/${appState.state?.results?.id}`;
	if (state === 'room' && appState.state?.room) return `/room/${appState.state.room.code}`;
	if (state === 'queue') return '/play/ranked';
	return '/play';
}

export function sendStateRequest(): void {
	sendSocket('state');
}
