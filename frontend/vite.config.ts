import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0', // Allows access from other devices on the network
    port: 5173,      // Optional: specify a port, defaults to 5173
    strictPort: true, // Optional: don't try the next port if this one is taken
    allowedHosts: ['frontend-service'],
  }
})
