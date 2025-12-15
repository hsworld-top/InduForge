import * as ElementPlusIconsVue from '@element-plus/icons-vue';

export function resolveEpIcon(iconName) {
    if (!iconName) return null;
    const match = Object.keys(ElementPlusIconsVue).find((key) => key.toLowerCase() === iconName.toLowerCase());
    return match ? ElementPlusIconsVue[match] : null;
}
