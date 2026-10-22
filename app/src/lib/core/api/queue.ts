import { api } from './client';
import { ENDPOINTS } from './config';
import type { LobbyDuration, QueueStatus } from './types';

export async function joinQueue(duration: LobbyDuration): Promise<QueueStatus> {
	const res = await api<{ queue: QueueStatus }>(ENDPOINTS.QUEUE, {
		method: 'POST',
		body: { duration }
	});
	return res.queue;
}

export async function leaveQueue(duration: LobbyDuration): Promise<void> {
	await api(`${ENDPOINTS.QUEUE}?duration=${duration}`, { method: 'DELETE' });
}

export async function queueStatus(duration: LobbyDuration): Promise<QueueStatus | null> {
	const res = await api<{ queue: QueueStatus | null }>(
		`${ENDPOINTS.QUEUE}?duration=${duration}`
	);
	return res.queue;
}
