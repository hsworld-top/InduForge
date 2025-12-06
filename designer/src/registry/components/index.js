/**
 * Basic Components Index
 * 
 * 参考 OpenTiny 的组件分类
 * - container: 基本容器
 * - layout: 行列容器
 * - basic: 基础元素
 * - form: 表单组件
 */

import Container from './Container.js'
import Div from './Div.js'
import Row from './Row.js'
import Col from './Col.js'
import Text from './Text.js'
import Button from './Button.js'
import Image from './Image.js'
import Link from './Link.js'
import Divider from './Divider.js'
import Input from './Input.js'
import { registerComponent } from '../index.js'

// Export individual component definitions
export { 
  Container, 
  Div,
  Row,
  Col,
  Text, 
  Button, 
  Image, 
  Link,
  Divider,
  Input 
}

// All basic components
export const basicComponents = [
  // 基本容器
  Container,
  Div,
  
  // 行列容器
  Row,
  Col,
  
  // 基础元素
  Text,
  Button,
  Image,
  Link,
  Divider,
  
  // 表单组件
  Input,
]

/**
 * Register all basic components with the registry
 */
export function registerBasicComponents() {
  basicComponents.forEach(component => {
    registerComponent(component)
  })
}
