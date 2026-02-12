import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'
import path from 'path'
import {fileURLToPath} from 'url'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [svelte()],
  resolve: {
    alias: {
      $lib: path.resolve(path.dirname(fileURLToPath(import.meta.url)), './src/lib')
    }
  }
})
