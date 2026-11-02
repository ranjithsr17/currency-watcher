import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: 'localhost',
    port: 5173,
    // The backend's CORS allowlist names this exact origin, so fail loudly
    // rather than silently moving to another port.
    strictPort: true,
  },
})
