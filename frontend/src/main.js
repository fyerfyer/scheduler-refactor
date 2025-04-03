import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import store from './store'

// Bootstrap JS
import 'bootstrap/dist/js/bootstrap.bundle.min.js'

// Create and mount the app
const app = createApp(App)

app.use(store)
app.use(router)

app.mount('#app')