import type { MatchHistory } from './api/types';

/** The three practice rivals. Their pace is not fixed: it is derived from the
 *  player's recent form, then scaled into three difficulty bands. */
export type PracticeBotID = 'ace' | 'nova' | 'rune';

export interface PracticeBotProfile {
	id: PracticeBotID;
	label: string;
	/** Racing-inspired identity colours: warm red, deep blue, racing green. */
	color: string;
	/** Speed multiplier applied to the player's form. */
	factor: number;
	/** Small flat offset keeps the bots readable for very low paces. */
	offset: number;
	accuracy: number;
}

export const PRACTICE_BOTS: PracticeBotProfile[] = [
	{
		id: 'ace',
		label: 'Ace',
		color: '#e8322e',
		factor: 0.86,
		offset: 4,
		accuracy: 98
	},
	{
		id: 'nova',
		label: 'Nova',
		color: '#3f7cff',
		factor: 1,
		offset: 4,
		accuracy: 97
	},
	{
		id: 'rune',
		label: 'Rune',
		color: '#24b47e',
		factor: 1.14,
		offset: 4,
		accuracy: 95
	}
];

export const DEFAULT_PACE_WPM = 70;

export interface PaceEstimate {
	wpm: number;
	runs: number;
}

/**
 * Exponential moving average over the player's latest runs, oldest to newest.
 * A single very fast (or very slow) run therefore only nudges the bots instead
 * of permanently defining them.
 */
export function estimatePace(
	history: MatchHistory[],
	window = 8,
	alpha = 0.45
): PaceEstimate {
	const recent = (history ?? [])
		.filter((run) => run.wpm > 0)
		.slice(0, window)
		.reverse();
	if (recent.length === 0) return { wpm: DEFAULT_PACE_WPM, runs: 0 };

	let ema = recent[0].wpm;
	for (let i = 1; i < recent.length; i++) {
		ema = alpha * recent[i].wpm + (1 - alpha) * ema;
	}
	return {
		wpm: Math.min(180, Math.max(30, Math.round(ema))),
		runs: recent.length
	};
}

export function targetForBot(profile: PracticeBotProfile, paceWpm: number): number {
	return Math.min(220, Math.max(24, Math.round(paceWpm * profile.factor + profile.offset)));
}
