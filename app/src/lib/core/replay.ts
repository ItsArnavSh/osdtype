import type { WpmSample } from './stats';
import { KeypressAction, type ClientKeypress } from './api/types';

export interface RunStats {
	typed: string;
	raw: number;
	net: number;
	accuracy: number;
	correct: number;
	wrong: number;
}

export interface DisplayState {
	/** Typed characters keyed by display-character index. */
	typed: (string | null)[];
	/** Index of the next character the player would touch. */
	cursor: number;
}

export function isWhitespace(char: string): boolean {
	return char === ' ' || char === '\n' || char === '\t' || char === '\r';
}

export function byteToChar(value: string, bytes: number): number {
	if (bytes <= 0) return 0;
	const encoder = new TextEncoder();
	let consumed = 0;
	let index = 0;
	for (const ch of value) {
		if (consumed >= bytes) break;
		consumed += encoder.encode(ch).length;
		index++;
	}
	return index;
}

export function charsOf(value: string): string[] {
	return Array.from(value);
}

export function nonWhitespace(value: string | string[]): string[] {
	return (Array.isArray(value) ? value : charsOf(value)).filter((char) => !isWhitespace(char));
}

export function applyEvent(typed: string[], event: ClientKeypress): void {
	if (event.action === KeypressAction.BACKSPACE) {
		const value = charsOf(event.value ?? '');
		if (value.length > 0 && typed.length >= value.length) {
			const suffix = typed.slice(typed.length - value.length);
			if (suffix.join('') === value.join('')) {
				typed.splice(typed.length - value.length);
				return;
			}
		}
		typed.pop();
		return;
	}
	typed.push(...charsOf(event.value ?? ''));
}

function sortedEvents(events: ClientKeypress[]): ClientKeypress[] {
	return [...(events ?? [])].sort((a, b) => (a.time_ms ?? 0) - (b.time_ms ?? 0));
}

export function typedAt(events: ClientKeypress[], elapsedMs: number): string[] {
	const typed: string[] = [];
	for (const event of sortedEvents(events)) {
		if ((event.time_ms ?? 0) > elapsedMs) break;
		applyEvent(typed, event);
	}
	return typed;
}

function statsFromTyped(typed: string[], original: string[], elapsedMs: number): RunStats {
	const user = nonWhitespace(typed);
	let correct = 0;
	let wrong = 0;
	const compare = Math.min(user.length, original.length);
	for (let i = 0; i < compare; i++) {
		if (user[i] === original[i]) correct++;
		else wrong++;
	}
	if (user.length > original.length) wrong += user.length - original.length;
	const elapsedSeconds = Math.max(0.001, elapsedMs / 1000);
	const raw = user.length > 0 ? user.length / 5 / (elapsedSeconds / 60) : 0;
	const net = user.length > 0 ? correct / 5 / (elapsedSeconds / 60) : 0;
	const accuracy = user.length > 0 ? correct / user.length : 1;
	return {
		typed: typed.join(''),
		raw: Math.max(0, raw),
		net: Math.max(0, net),
		accuracy,
		correct,
		wrong
	};
}

export function statsAt(
	events: ClientKeypress[],
	snippet: string,
	startOffset: number,
	elapsedMs: number
): RunStats {
	const chars = charsOf(snippet);
	const start = byteToChar(snippet, startOffset);
	const original = nonWhitespace(chars.slice(start));
	return statsFromTyped(typedAt(events, elapsedMs), original, elapsedMs);
}

/**
 * Builds a cumulative WPM series at one-second boundaries, matching the
 * server's scoring model (whitespace is skipped and net WPM is error-adjusted).
 */
export function deriveSamples(
	events: ClientKeypress[],
	snippet: string,
	startOffset: number,
	duration: number
): WpmSample[] {
	const maxSeconds = Math.max(1, Math.ceil(duration));
	const sorted = sortedEvents(events);
	const chars = charsOf(snippet);
	const start = byteToChar(snippet, startOffset);
	const original = nonWhitespace(chars.slice(start));
	if (sorted.length === 0) return [];

	const typed: string[] = [];
	let eventIndex = 0;
	const samples: WpmSample[] = [{ t: 0, raw: 0, net: 0, errors: 0 }];
	for (let second = 1; second <= maxSeconds; second++) {
		const cutoff = second * 1000;
		while (eventIndex < sorted.length && (sorted[eventIndex].time_ms ?? 0) <= cutoff) {
			applyEvent(typed, sorted[eventIndex]);
			eventIndex++;
		}
		const stats = statsFromTyped(typed, original, cutoff);
		samples.push({
			t: second,
			raw: stats.raw,
			net: stats.net,
			errors: stats.wrong
		});
	}
	return samples;
}

/**
 * Old games and bot rows may not have a recording. Their final result is still
 * known, so draw a plausible cumulative line rather than omitting the player.
 */
export function syntheticSamples(wpm: number, accuracy: number, duration: number): WpmSample[] {
	const maxSeconds = Math.max(1, Math.ceil(duration));
	const safeAccuracy = Math.min(1, Math.max(0.01, accuracy));
	const targetNet = Math.max(0, wpm);
	const targetRaw = targetNet / safeAccuracy;
	const samples: WpmSample[] = [{ t: 0, raw: 0, net: 0, errors: 0 }];
	for (let second = 1; second <= maxSeconds; second++) {
		const ramp = Math.min(1, second / Math.max(2, maxSeconds * 0.24));
		const wobble = Math.sin(second * 1.71 + targetNet * 0.03) * targetNet * 0.035;
		const net = Math.max(0, targetNet * ramp + wobble * ramp);
		const raw = Math.max(net, targetRaw * ramp + wobble * ramp);
		samples.push({
			t: second,
			raw,
			net,
			errors: Math.round(((1 - safeAccuracy) * targetRaw * 60) / 5)
		});
	}
	return samples;
}

export function mapTypedToDisplay(
	snippet: string,
	startOffset: number,
	typedChars: string[]
): DisplayState {
	const chars = charsOf(snippet);
	const start = byteToChar(snippet, startOffset);
	const mapped: (string | null)[] = new Array(chars.length).fill(null);
	let typedIndex = 0;
	let cursor = start;
	for (let index = start; index < chars.length; index++) {
		if (isWhitespace(chars[index])) continue;
		if (typedIndex < typedChars.length) {
			mapped[index] = typedChars[typedIndex++];
			cursor = index + 1;
		} else {
			cursor = index;
			break;
		}
	}
	// Move the caret past automatically skipped whitespace.
	while (cursor < chars.length && isWhitespace(chars[cursor])) cursor++;
	return { typed: mapped, cursor };
}

export function lastEventTime(events: ClientKeypress[]): number {
	let last = 0;
	for (const event of events) last = Math.max(last, event.time_ms ?? 0);
	return last;
}
