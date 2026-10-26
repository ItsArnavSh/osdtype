export const WHITESPACE_MODES = ['manual', 'adaptive', 'auto'] as const;

export type WhitespaceMode = (typeof WHITESPACE_MODES)[number];

export const DEFAULT_WHITESPACE_MODE: WhitespaceMode = 'adaptive';

export const WHITESPACE_MODE_LABELS: Record<WhitespaceMode, string> = {
	manual: 'manual',
	adaptive: 'adaptive',
	auto: 'auto'
};

export const WHITESPACE_MODE_HINTS: Record<WhitespaceMode, string> = {
	manual:
		'Type every space and enter. Tabs advance automatically; missed whitespace counts as mistakes.',
	adaptive:
		'Missed spaces pass, extra spaces count, and enters are manual. Tabs advance automatically.',
	auto: 'Spaces, enters, and tabs advance automatically.'
};
