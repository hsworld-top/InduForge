import type { ComponentDescriptor } from "./registry";
import UserAvatarMenuRenderer from "@/materials/UserAvatarMenu/UserAvatarMenuRenderer.vue";

export const descriptor: ComponentDescriptor = {
  renderTag: "div",
  isContainer: false,
  defaultStyle: {},
  acceptChildren: false,
  isMovable: true,
  childResizable: true,
  displayContent: () => null,
  customRenderer: UserAvatarMenuRenderer,
  defaultSize: { width: 180, height: 44 },
};

export default descriptor;
