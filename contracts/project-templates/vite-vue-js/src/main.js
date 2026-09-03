import { createApp } from 'vue'
import { configureRuntime, createBrowserRuntime } from '@induforge/runtime-sdk'
import '@induforge/runtime-sdk/scene-elements'
import './style.css'
import App from './App.vue'

configureRuntime(createBrowserRuntime())
createApp(App).mount('#app')
