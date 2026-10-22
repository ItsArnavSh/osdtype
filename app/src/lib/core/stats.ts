/** One per-second snapshot used to draw the post-run graph. */
export interface WpmSample {
	/** Seconds elapsed since the run started. */
	t: number;
	/** Cumulative raw words per minute. */
	raw: number;
	/** Cumulative error-adjusted words per minute. */
	net: number;
	/** Cumulative mistakes at this point in the run. */
	errors: number;
}
