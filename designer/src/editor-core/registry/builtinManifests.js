/**
 * 内置组件清单注册
 * 将所有内置组件的 Manifest 注册到 ComponentRegistry
 */

import { componentRegistry } from "./componentRegistry.js";
import { getAllManifests } from "../../manifests/index.js";

/**
 * 从 Manifest props 提取默认属性
 * @param {Array} props - 属性定义数组
 * @returns {Object} 默认属性对象
 */
function extractDefaultProps(props) {
  const defaultProps = {};
  for (const prop of props) {
    if (prop.defaultValue !== undefined) {
      defaultProps[prop.name] = prop.defaultValue;
    }
  }
  return defaultProps;
}

/**
 * 获取组件的默认样式
 * @param {string} type - 组件类型
 * @returns {Object} 默认样式
 */
function getDefaultStyle(type) {
  const containerTypes = ["FlexContainer", "FreeContainer", "GridContainer"];
  const baseStyle = {
    minHeight: "40px",
  };

  if (containerTypes.includes(type)) {
    return {
      ...baseStyle,
      width: "100%",
      padding: "8px",
    };
  }

  // Text 组件
  if (type === "Text") {
    return {
      fontSize: "14px",
      color: "#333333",
    };
  }

  // Button 组件
  if (type === "Button") {
    return {
      padding: "8px 16px",
    };
  }

  return baseStyle;
}

/**
 * 判断是否为容器组件
 * @param {string} type - 组件类型
 * @returns {boolean}
 */
function isContainerType(type) {
  const containerTypes = ["FlexContainer", "FreeContainer", "GridContainer"];
  return containerTypes.includes(type);
}

/**
 * 获取组件分类
 * @param {string} category - Manifest 中的分类
 * @returns {string} ComponentRegistry 分类
 */
function mapCategory(category) {
  const categoryMap = {
    布局: "layout",
    基础: "basic",
    表单: "form",
    图形: "canvas",
  };
  return categoryMap[category] || "custom";
}

/**
 * 注册所有内置组件
 */
export function registerBuiltinComponents() {
  const manifests = getAllManifests();

  for (const manifest of manifests) {
    const componentManifest = {
      type: manifest.type,
      name: manifest.name,
      category: mapCategory(manifest.category),
      icon: undefined,
      description: `${manifest.name}组件`,
      defaultProps: extractDefaultProps(manifest.props),
      defaultStyle: getDefaultStyle(manifest.type),
      propsSchema: manifest.props,
      styleSchema: undefined,
      events: ["click", "dblclick", "mouseenter", "mouseleave"],
      isContainer: isContainerType(manifest.type),
      allowedChildren: undefined,
      slots: undefined,
    };

    componentRegistry.register(componentManifest);
  }

  console.log(`[Designer] 已注册 ${manifests.length} 个内置组件`);
}

export default {
  registerBuiltinComponents,
};
