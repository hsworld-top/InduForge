/**
 * Layout Components - 布局组件
 *
 * 用于页面布局和组件排列的容器组件
 *
 * Task 2.3: 重构布局组件注册（Container, Row/Col, Flex, Grid, CenterLayout）
 *
 * 布局组件列表：
 * 1. Container - 盒子容器（支持 Flex/Grid/Block 三种布局模式）
 * 2. Row - 行容器（24 栅格系统）
 * 3. Col - 列容器（24 栅格系统）
 * 4. FlexLayout - 弹性容器（Flexbox 布局）
 * 5. Grid - 栅格布局（CSS Grid）
 * 6. CenterLayout - 全宽居中布局
 */

import Container from './Container.js';
import FlexLayout from './FlexLayout.js';
import CenterLayout from './CenterLayout.js';
import Row from './Row.js';
import Col from './Col.js';
import Grid from './Grid.js';

// 导出所有布局组件定义
export default [Container, Row, Col, FlexLayout, Grid, CenterLayout];

// 命名导出
export { Container, Row, Col, FlexLayout, Grid, CenterLayout };
