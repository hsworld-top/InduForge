/**
 * 内置组件清单注册：将 manifests 层数据注册到 ComponentRegistry
 */

import type { EditorComponentManifest } from "./component-registry";
import type { ComponentManifest } from "@/manifests/manifest-registry";
import { getAllManifests } from "../../manifests";
import { getComponentEventDefinitions } from "./component-events";
import { componentRegistry } from "./component-registry";

function extractDefaultProps(props: ComponentManifest["props"]): Record<string, unknown> {
  const defaultProps: Record<string, unknown> = {};
  for (const prop of props) {
    if (prop.defaultValue !== undefined) {
      defaultProps[prop.name] = prop.defaultValue;
    }
  }
  return defaultProps;
}

function getDefaultStyle(type: string): Record<string, string | number> {
  const containerTypes = [
    "HorizontalLayout",
    "VerticalLayout",
    "FormLayout",
    "Tabs",
    "Collapse",
    "ElContainer",
    "FreeContainer",
  ];
  const baseStyle: Record<string, string | number> = {
    minHeight: "40px",
  };

  if (type === "ElHeader" || type === "ElFooter") {
    return {
      ...baseStyle,
      width: "100%",
      padding: "8px",
    };
  }

  if (type === "ElAside") {
    return {
      ...baseStyle,
      width: "200px",
      padding: "8px",
    };
  }

  if (type === "ElMain") {
    return {
      ...baseStyle,
      width: "100%",
      padding: "8px",
      flex: "1 1 auto",
    };
  }

  if (containerTypes.includes(type)) {
    return {
      ...baseStyle,
      width: "100%",
      padding: "8px",
    };
  }

  if (type === "Button") {
    return {
      padding: "8px 16px",
    };
  }

  return baseStyle;
}

function mapCategory(category: string): string {
  const categoryMap: Record<string, string> = {
    布局: "layout",
    基础: "basic",
    表单: "form",
    图形: "canvas",
    UI组件: "ui",
    PC端组件: "uiPc",
    图表: "chart",
    绘图: "canvas",
  };
  return categoryMap[category] ?? "custom";
}

export function registerBuiltinComponents(): void {
  const manifests = getAllManifests();

  for (const manifest of manifests) {
    const baseStyle = getDefaultStyle(manifest.type);
    const mergedStyle: Record<string, string | number> = {
      ...baseStyle,
      ...(manifest.defaultStyle ?? {}),
    };

    const rawEvents = getComponentEventDefinitions(manifest.type);
    const events: EditorComponentManifest["events"] =
      Array.isArray(manifest.events) && manifest.events.length > 0 ? manifest.events : rawEvents;

    const componentManifest: EditorComponentManifest = {
      type: manifest.type,
      name: manifest.name,
      category: mapCategory(manifest.category),
      description: `${manifest.name}组件`,
      defaultProps: extractDefaultProps(manifest.props),
      defaultStyle: mergedStyle,
      propsSchema: manifest.props,
      events,
      isContainer: manifest.isContainer === true,
    };
    if (manifest.defaultSize !== undefined) {
      componentManifest.defaultSize = manifest.defaultSize;
    }

    componentRegistry.register(componentManifest);
  }

  console.warn(`[Designer] 已注册 ${manifests.length} 个内置组件`);
}

export default {
  registerBuiltinComponents,
};
