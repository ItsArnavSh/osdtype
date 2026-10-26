/**
 * Shared state for one-time site prompts.
 *
 * The onboarding card is rendered by the root layout, while the typing
 * surfaces own the global keydown listener. This flag lets them stay inert
 * while the card is open without coupling either component to the other.
 */
export const promptState = $state({
	blocking: false
});
