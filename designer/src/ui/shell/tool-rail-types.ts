import type { Component } from "vue";

export interface ToolRailItem {
  key: string;
  label: string;
  icon?: Component;
  placement?: "bottom";
}
