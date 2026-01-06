import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

const BACKEND_PORT = process.env.PGVOYAGER_BACKEND_PORT || '5138';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		port: 5137,
		strictPort: true,
		proxy: {
			'/api': {
				target: `http://localhost:${BACKEND_PORT}`,
				changeOrigin: true,
				ws: true
			},
			'/ws': {
				target: `http://localhost:${BACKEND_PORT}`,
				ws: true,
				changeOrigin: true
			}
		}
	}
});
