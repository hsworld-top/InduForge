/**
 * Basic Components Index
 * 
 * Exports all basic component definitions and provides
 * a function to register them with the component registry.
 */

import Container from './Container.js'
import Text from './Text.js'
import Button from './Button.js'
import Image from './Image.js'
import Input from './Input.js'
import { registerComponent } from '../index.js'

// Export individual component definitions
export { Container, Text, Button, Image, Input }

// All basic components
export const basicComponents = [
  Container,
  Text,
  Button,
  Image,
  Input
]

/**
 * Register all basic components with the registry
 */
export function registerBasicComponents() {
  basicComponents.forEach(component => {
    registerComponent(component)
  })
}
