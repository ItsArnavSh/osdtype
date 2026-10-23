import { BASE_URL } from './config';

export class ApiError extends Error {
	status: number;
	code?: string;
	requestId?: string;

	constructor(message: string, status: number, code?: string, requestId?: string) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
		this.code = code;
		this.requestId = requestId;
	}
}

export interface RequestOptions {
	method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE';
	body?: unknown;
	headers?: Record<string, string>;
	idempotencyKey?: string;
}

export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
	const headers: Record<string, string> = { ...(options.headers ?? {}) };
	if (options.body !== undefined) headers['Content-Type'] = 'application/json';
	if (options.idempotencyKey) headers['Idempotency-Key'] = options.idempotencyKey;

	const res = await fetch(`${BASE_URL}${path}`, {
		method: options.method ?? 'GET',
		credentials: 'include',
		headers,
		body: options.body === undefined ? undefined : JSON.stringify(options.body)
	});
	const text = await res.text();
	let data: unknown = null;
	if (text) {
		try {
			data = JSON.parse(text);
		} catch {
			data = text;
		}
	}
	if (!res.ok) {
		const obj = (data ?? {}) as { error?: string; code?: string; request_id?: string };
		throw new ApiError(
			obj.error ?? `Request failed (${res.status})`,
			res.status,
			obj.code,
			obj.request_id ?? res.headers.get('X-Request-ID') ?? undefined
		);
	}
	return data as T;
}

export function newIdempotencyKey(): string {
	if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) return crypto.randomUUID();
	return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export function errorMessage(err: unknown): string {
	if (err instanceof ApiError) return err.message;
	if (err instanceof Error) return err.message;
	return 'Something went wrong';
}
