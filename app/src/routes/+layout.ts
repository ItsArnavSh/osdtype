// OSDTYP is a client-rendered game shell. Keeping SSR off makes the adapter-node
// and adapter-static builds behave the same, and lets Render serve the SPA from
// its CDN with a single fallback document.
export const ssr = false;
