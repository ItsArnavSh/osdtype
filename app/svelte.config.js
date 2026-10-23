import { mdsvex } from 'mdsvex';
import adapterNode from '@sveltejs/adapter-node';
import adapterStatic from '@sveltejs/adapter-static';

const staticBuild = process.env.ADAPTER === 'static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		// `make prod` and the Docker image use adapter-node. Render serves the
		// same SPA from its static CDN when ADAPTER=static is set at build time.
		adapter: staticBuild ? adapterStatic({ fallback: 'index.html' }) : adapterNode()
	},
	preprocess: [mdsvex()],
	extensions: ['.svelte', '.svx']
};

export default config;
