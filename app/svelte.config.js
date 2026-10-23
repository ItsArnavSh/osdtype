import { mdsvex } from 'mdsvex';
import adapter from '@sveltejs/adapter-node';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		// adapter-node produces the self-contained `build/` server used by
		// `make prod`. Dev mode is unaffected.
		adapter: adapter()
	},
	preprocess: [mdsvex()],
	extensions: ['.svelte', '.svx']
};

export default config;
