import { createApp } from 'vue'
import { configureRuntime, createHttpRuntime } from '@induforge/runtime-sdk'
import '@induforge/runtime-sdk/scene-elements'
import './style.css'
import App from './App.vue'

configureRuntime(createHttpRuntime())
createApp(App).mount('#app')
