import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import '@induforge/runtime-sdk/scene-elements'
import './index.css'
import App from './App.jsx'

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
