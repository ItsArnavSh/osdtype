import type { Language } from '$lib/core/entity/languages';

/**
 * Character-level syntax classification for the generated snippets.
 *
 * The generator emits complete (if quirky) programs, so we scan the flat
 * string once and tag each character. This keeps merged chunks such as
 * `for(int` working, and lets a token span a keyword, a string or a comment.
 *
 * The result is aligned index-for-index with the input string. Styling is
 * intentionally restrained — see the `--syntax-*` theme variables.
 */
export type SyntaxKind =
	| 'keyword'
	| 'type'
	| 'literal'
	| 'comment'
	| 'meta'
	| 'punct'
	| 'plain'
	| 'ws';

interface LangRules {
	keywords: Set<string>;
	types: Set<string>;
	/** Line-comment prefixes, e.g. `//`. */
	line: string[];
	/** Block-comment delimiter pairs, commonly slash-star style. */
	block: [string, string][];
	/** Quote characters that open a string literal. */
	strings: string[];
	/** Meaning of a leading `#`: a comment, a preprocessor directive or nothing. */
	hash: 'comment' | 'meta' | 'none';
	/** Use the HTML tag/attribute scanner. */
	html?: boolean;
}

const set = (words: string) => new Set(words.toLowerCase().split(/\s+/));

const KEYWORDS: Record<Language, string> = {
	c: 'if else while for return break continue switch case default sizeof do goto struct enum typedef union static const volatile extern inline unsigned signed',
	cpp: 'if else while for return break continue switch case default sizeof do goto class struct enum typedef union static const volatile extern inline unsigned signed public private protected new delete namespace using template typename auto virtual override final noexcept try catch throw constexpr explicit operator friend this nullptr true false',
	go: 'package import func var const type struct interface map chan go defer if else for range return break continue switch case default select goto fallthrough',
	haskell: 'module where import data type class instance deriving do let in if then else case of return',
	html: '',
	java: 'public private protected static final class interface extends implements new if else while for return break continue void package import try catch finally throw throws this super abstract synchronized instanceof switch case default do enum',
	js: 'function const let var if else while for return break continue class extends new import export from async await try catch finally throw switch case default do instanceof typeof in of this super null undefined true false void delete yield static get set',
	python: 'def class if elif else for while return break continue import from as with try except finally raise pass lambda yield global nonlocal in is not and or none true false assert del async await',
	rust: 'fn let mut if else while for in return break continue pub use struct impl enum match loop move as where trait mod crate super self ref dyn async await unsafe const static type',
	sql: 'select from where insert into values create table index order by limit and or not null as update set delete join left right inner on group having distinct'
};

const TYPES: Record<Language, string> = {
	c: 'char int float double long short void bool size_t uint8_t uint16_t uint32_t uint64_t int8_t int16_t int32_t int64_t file',
	cpp: 'char int float double long short void bool auto string size_t uint8_t uint32_t uint64_t int32_t int64_t std vector map unordered_map pair ostream istream',
	go: 'int int8 int16 int32 int64 uint uint8 uint16 uint32 uint64 uintptr float32 float64 complex64 complex128 byte rune string bool error any',
	haskell: 'int integer double float char string bool maybe either io',
	html: '',
	java: 'int long short byte float double char boolean void string integer long double float boolean character list map arraylist hashmap',
	js: 'number string boolean object any unknown never void array promise map set symbol bigint',
	python: 'int float str bool list dict set tuple bytes object none',
	rust: 'i8 i16 i32 i64 i128 isize u8 u16 u32 u64 u128 usize f32 f64 bool char str string vec option result hashmap box rc arc',
	sql: 'int integer text varchar timestamp boolean float double decimal numeric'
};

const RULES: Record<Language, LangRules> = {
	c: { keywords: set(KEYWORDS.c), types: set(TYPES.c), line: ['//'], block: [['/*', '*/']], strings: ['"', "'"], hash: 'meta' },
	cpp: { keywords: set(KEYWORDS.cpp), types: set(TYPES.cpp), line: ['//'], block: [['/*', '*/']], strings: ['"', "'"], hash: 'meta' },
	go: { keywords: set(KEYWORDS.go), types: set(TYPES.go), line: ['//'], block: [['/*', '*/']], strings: ['"', '`'], hash: 'none' },
	haskell: { keywords: set(KEYWORDS.haskell), types: set(TYPES.haskell), line: ['--'], block: [['{-', '-}']], strings: ['"'], hash: 'none' },
	html: { keywords: set(''), types: set(''), line: [], block: [], strings: [], hash: 'none', html: true },
	java: { keywords: set(KEYWORDS.java), types: set(TYPES.java), line: ['//'], block: [['/*', '*/']], strings: ['"'], hash: 'none' },
	js: { keywords: set(KEYWORDS.js), types: set(TYPES.js), line: ['//'], block: [['/*', '*/']], strings: ['"', "'", '`'], hash: 'none' },
	python: { keywords: set(KEYWORDS.python), types: set(TYPES.python), line: [], block: [], strings: ['"', "'"], hash: 'comment' },
	rust: { keywords: set(KEYWORDS.rust), types: set(TYPES.rust), line: ['//'], block: [['/*', '*/']], strings: ['"'], hash: 'meta' },
	sql: { keywords: set(KEYWORDS.sql), types: set(TYPES.sql), line: ['--'], block: [['/*', '*/']], strings: ["'"], hash: 'none' }
};

const WORD_START = /[A-Za-z_$]/;
const WORD_CHAR = /[A-Za-z0-9_$]/;

function isWs(ch: string): boolean {
	return ch === ' ' || ch === '\t' || ch === '\n' || ch === '\r';
}

function isDigit(ch: string): boolean {
	return ch >= '0' && ch <= '9';
}

export function highlightCode(code: string, lang: Language): SyntaxKind[] {
	const rules = RULES[lang] ?? RULES.c;
	const kinds = new Array<SyntaxKind>(code.length).fill('plain');
	const len = code.length;
	let i = 0;

	const mark = (from: number, to: number, kind: SyntaxKind) => {
		for (let k = from; k < to; k++) kinds[k] = kind;
	};

	while (i < len) {
		const ch = code[i];

		if (isWs(ch)) {
			kinds[i] = 'ws';
			i++;
			continue;
		}

		// Line comments.
		const line = rules.line.find((p) => code.startsWith(p, i));
		if (line) {
			let j = i;
			while (j < len && code[j] !== '\n') j++;
			mark(i, j, 'comment');
			i = j;
			continue;
		}

		// Block comments.
		let blocked = false;
		for (const [open, close] of rules.block) {
			if (!code.startsWith(open, i)) continue;
			const end = code.indexOf(close, i + open.length);
			const stop = end === -1 ? len : end + close.length;
			mark(i, stop, 'comment');
			i = stop;
			blocked = true;
			break;
		}
		if (blocked) continue;

		// Strings.
		if (rules.strings.includes(ch)) {
			let j = i + 1;
			while (j < len && code[j] !== ch && code[j] !== '\n') {
				if (code[j] === '\\') j++;
				j++;
			}
			if (j < len && code[j] === ch) j++;
			mark(i, j, 'literal');
			i = j;
			continue;
		}

		// `#` handling: preprocessor directive or Python comment.
		if (ch === '#' && rules.hash !== 'none') {
			if (rules.hash === 'comment') {
				let j = i;
				while (j < len && code[j] !== '\n') j++;
				mark(i, j, 'comment');
				i = j;
			} else {
				// Preprocessor: colour the whole directive line as meta.
				let j = i;
				while (j < len && code[j] !== '\n') j++;
				mark(i, j, 'meta');
				i = j;
			}
			continue;
		}

		// HTML tags and comments.
		if (rules.html && ch === '<') {
			i = scanHtml(code, i, kinds);
			continue;
		}

		// Numbers.
		if (isDigit(ch)) {
			let j = i + 1;
			while (j < len && (isDigit(code[j]) || code[j] === '_')) j++;
			if (code[j] === '.' && isDigit(code[j + 1] ?? '')) {
				j++;
				while (j < len && (isDigit(code[j]) || code[j] === '_')) j++;
			}
			if (j < len && 'fFuUlL'.includes(code[j])) j++;
			mark(i, j, 'literal');
			i = j;
			continue;
		}

		// Words.
		if (WORD_START.test(ch)) {
			let j = i + 1;
			while (j < len && WORD_CHAR.test(code[j])) j++;
			const word = code.slice(i, j).toLowerCase();
			const kind: SyntaxKind = rules.keywords.has(word)
				? 'keyword'
				: rules.types.has(word)
					? 'type'
					: 'plain';
			mark(i, j, kind);
			i = j;
			continue;
		}

		// Operators, brackets, arrows, `:=`, `..` — punctuation.
		kinds[i] = 'punct';
		i++;
	}

	return kinds;
}

/** Scans one HTML tag (or comment) starting at `<`. Returns the next index. */
function scanHtml(code: string, i: number, kinds: SyntaxKind[]): number {
	const len = code.length;
	const mark = (from: number, to: number, kind: SyntaxKind) => {
		for (let k = from; k < to; k++) kinds[k] = kind;
	};

	if (code.startsWith('<!--', i)) {
		const end = code.indexOf('-->', i + 4);
		const stop = end === -1 ? len : end + 3;
		mark(i, stop, 'comment');
		return stop;
	}

	let j = i;
	kinds[j] = 'punct';
	j++; // `<`
	if (code[j] === '/') {
		kinds[j] = 'punct';
		j++;
	}
	while (j < len && /[A-Za-z0-9-]/.test(code[j])) {
		kinds[j] = 'keyword';
		j++;
	}
	while (j < len && code[j] !== '>') {
		if (isWs(code[j])) {
			kinds[j] = 'ws';
			j++;
			continue;
		}
		if (code[j] === '"' || code[j] === "'") {
			const quote = code[j];
			let k = j + 1;
			while (k < len && code[k] !== quote) k++;
			if (k < len) k++;
			mark(j, k, 'literal');
			j = k;
			continue;
		}
		if (WORD_START.test(code[j])) {
			let k = j + 1;
			while (k < len && /[A-Za-z0-9_:@.-]/.test(code[k])) k++;
			mark(j, k, 'type');
			j = k;
			continue;
		}
		kinds[j] = 'punct';
		j++;
	}
	if (j < len && code[j] === '>') {
		kinds[j] = 'punct';
		j++;
	}
	return j;
}
