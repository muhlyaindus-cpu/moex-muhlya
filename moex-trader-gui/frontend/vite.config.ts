import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'

// Dev-сервер Vite (порт по умолчанию 5173, при занятости — следующий свободный) нужен Wails для HMR.
// Его не отключают: без него не будет быстрой пересборки фронта при `wails dev`.
// В браузере для работы с Go открывайте URL из консоли Wails (обычно http://localhost:34115), а не сырой порт Vite.
// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
})
