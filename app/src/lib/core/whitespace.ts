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
		'Spaces are ordinary characters, and Enter always moves to the next line. Tabs and line indentation advance automatically.',
	adaptive:
		'Missed spaces pass, wrong spaces count as errors, and Enter always moves to the next line. Tabs and line indentation advance automatically.',
	auto: 'Spaces, enters, tabs, and line indentation advance automatically.'
};
