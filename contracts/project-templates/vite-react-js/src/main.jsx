import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { configureRuntime, createHttpRuntime } from '@induforge/runtime-sdk'
import '@induforge/runtime-sdk/scene-elements'
import './index.css'
import App from './App.jsx'

configureRuntime(createHttpRuntime())
createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
