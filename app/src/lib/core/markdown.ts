// A deliberately small Markdown renderer for contest descriptions. It only
// emits the handful of tags event pages need and never trusts raw HTML.
function escapeHTML(value: string): string {
	return value
		.replaceAll('&', '&amp;')
		.replaceAll('<', '&lt;')
		.replaceAll('>', '&gt;')
		.replaceAll('"', '&quot;')
		.replaceAll("'", '&#39;');
}

function safeHref(raw: string): string {
	const value = raw.trim();
	if (/^https?:\/\//i.test(value) || value.startsWith('/')) return escapeHTML(value);
	return '#';
}

function inline(raw: string): string {
	const tokens: string[] = [];
	const stash = (html: string): string => {
		tokens.push(html);
		return `\u0000${tokens.length - 1}\u0000`;
	};

	let value = raw.replace(/`([^`]+)`/g, (_match, code: string) =>
		stash(`<code>${escapeHTML(code)}</code>`)
	);
	value = value.replace(
		/!?\[([^\]]*)\]\(([^)\s]+)\)/g,
		(_match, label: string, href: string) =>
			stash(
				`<a href="${safeHref(href)}" target="_blank" rel="noreferrer nofollow">${escapeHTML(label)}</a>`
			)
	);
	value = escapeHTML(value);
	value = value.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
	value = value.replace(/__([^_]+)__/g, '<strong>$1</strong>');
	value = value.replace(/(^|[^*])\*([^*\n]+)\*/g, '$1<em>$2</em>');
	value = value.replace(/(^|[^_])_([^_\n]+)_/g, '$1<em>$2</em>');
	return value.replace(/\u0000(\d+)\u0000/g, (_match, index: string) => tokens[Number(index)] ?? '');
}

function isBlockStart(line: string): boolean {
	return (
		/^#{1,6}\s+/.test(line) ||
		/^\s*([-*_])(?:\s*\1){2,}\s*$/.test(line) ||
		/^\s*[-*+]\s+/.test(line) ||
		/^\s*\d+\.\s+/.test(line) ||
		/^>\s?/.test(line) ||
		/^```/.test(line)
	);
}

export function renderMarkdown(source: string): string {
	const lines = (source ?? '').replace(/\r\n?/g, '\n').split('\n');
	const out: string[] = [];
	let paragraph: string[] = [];
	let listType: 'ul' | 'ol' | null = null;
	let quote: string[] = [];
	let code: string[] = [];
	let inCode = false;
	let codeLanguage = '';

	function flushParagraph(): void {
		if (paragraph.length === 0) return;
		out.push(`<p>${paragraph.map(inline).join('<br>')}</p>`);
		paragraph = [];
	}

	function flushList(): void {
		if (!listType) return;
		out.push(`</${listType}>`);
		listType = null;
	}

	function flushQuote(): void {
		if (quote.length === 0) return;
		out.push(`<blockquote>${quote.map(inline).join('<br>')}</blockquote>`);
		quote = [];
	}

	for (const line of lines) {
		if (inCode) {
			if (line.startsWith('```')) {
				out.push(
					`<pre><code${codeLanguage ? ` class="language-${escapeHTML(codeLanguage)}"` : ''}>${escapeHTML(code.join('\n'))}</code></pre>`
				);
				code = [];
				inCode = false;
				codeLanguage = '';
			} else {
				code.push(line);
			}
			continue;
		}

		if (line.startsWith('```')) {
			flushParagraph();
			flushList();
			flushQuote();
			inCode = true;
			codeLanguage = line.slice(3).trim().split(/\s+/)[0] ?? '';
			continue;
		}

		const heading = line.match(/^(#{1,6})\s+(.*)$/);
		if (heading) {
			flushParagraph();
			flushList();
			flushQuote();
			const level = heading[1].length;
			out.push(`<h${level}>${inline(heading[2].trim())}</h${level}>`);
			continue;
		}

		if (/^\s*([-*_])(?:\s*\1){2,}\s*$/.test(line)) {
			flushParagraph();
			flushList();
			flushQuote();
			out.push('<hr>');
			continue;
		}

		const unordered = line.match(/^\s*[-*+]\s+(.*)$/);
		const ordered = line.match(/^\s*\d+\.\s+(.*)$/);
		if (unordered || ordered) {
			flushParagraph();
			flushQuote();
			const nextType = unordered ? 'ul' : 'ol';
			if (listType !== nextType) {
				flushList();
				out.push(`<${nextType}>`);
				listType = nextType;
			}
			out.push(`<li>${inline((unordered ?? ordered)?.[1] ?? '')}</li>`);
			continue;
		}

		if (/^>\s?/.test(line)) {
			flushParagraph();
			flushList();
			quote.push(line.replace(/^>\s?/, ''));
			continue;
		}

		if (line.trim() === '') {
			flushParagraph();
			flushList();
			flushQuote();
			continue;
		}

		if (isBlockStart(line)) {
			flushParagraph();
			flushList();
			flushQuote();
		}
		flushList();
		flushQuote();
		paragraph.push(line);
	}

	if (inCode) {
		out.push(`<pre><code>${escapeHTML(code.join('\n'))}</code></pre>`);
	}
	flushParagraph();
	flushList();
	flushQuote();
	return out.join('\n');
}
