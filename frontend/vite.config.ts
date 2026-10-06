import path from 'path'
import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'

const root = path.resolve(__dirname)

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '#minpath': path.join(root, 'node_modules/vfile/lib/minpath.browser.js'),
      '#minproc': path.join(root, 'node_modules/vfile/lib/minproc.browser.js'),
      '#minurl': path.join(root, 'node_modules/vfile/lib/minurl.browser.js'),
    },
  },
})
