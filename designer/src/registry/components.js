/**
 * Component Registration - 组件注册
 *
 * 统一注册所有可用组件
 */

import { registerComponent } from './index';

// 导入各分类组件
import layoutComponents from './layout';
import basicComponents from './basic';
import uiComponents from './ui';
import chartComponents from './charts';

/**
 * 注册布局组件
 */
export function registerLayoutComponents() {
    layoutComponents.forEach((component) => {
        registerComponent(component);
    });
    console.log(`✅ Registered ${layoutComponents.length} layout components`);
}

/**
 * 注册基础组件
 */
export function registerBasicComponents() {
    basicComponents.forEach((component) => {
        registerComponent(component);
    });
    console.log(`✅ Registered ${basicComponents.length} basic components`);
}

/**
 * 注册 Element UI 组件
 */
export function registerUIComponents() {
    uiComponents.forEach((component) => {
        registerComponent(component);
    });
    console.log(`✅ Registered ${uiComponents.length} UI components`);
}

/**
 * 注册图表组件
 */
export function registerChartComponents() {
    chartComponents.forEach((component) => {
        registerComponent(component);
    });
    console.log(`✅ Registered ${chartComponents.length} chart components`);
}

/**
 * 注册所有组件
 */
export function registerAllComponents() {
    registerLayoutComponents();
    registerBasicComponents();
    registerUIComponents();
    registerChartComponents();

    const total = layoutComponents.length + basicComponents.length + uiComponents.length + chartComponents.length;
    console.log(`✅ Total registered: ${total} components`);
}

export default {
    registerLayoutComponents,
    registerBasicComponents,
    registerUIComponents,
    registerChartComponents,
    registerAllComponents,
};
