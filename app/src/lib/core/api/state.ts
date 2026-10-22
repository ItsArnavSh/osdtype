import { api } from './client';
import { ENDPOINTS } from './config';
import type { StateView } from './types';

export function fetchState(): Promise<StateView> {
	return api<StateView>(ENDPOINTS.STATE);
}

export function acknowledgeResults(): Promise<StateView> {
	return api<StateView>(ENDPOINTS.ACK_RESULTS, { method: 'POST' });
}
