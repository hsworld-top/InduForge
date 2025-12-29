import { ComponentOptionsMixin } from 'vue';
import { ComponentProvideOptions } from 'vue';
import { CurrentValues } from 'simple-panzoom';
import { DefineComponent } from 'vue';
import { PanOptions } from 'simple-panzoom';
import { PanzoomObject } from 'simple-panzoom';
import { PanzoomOptions } from 'simple-panzoom';
import { PublicProps } from 'vue';
import { Ref } from 'vue';
import { ZoomOptions } from 'simple-panzoom';

declare const __VLS_component: DefineComponent<SketchRulerProps, {
initPanzoom: () => void;
panzoomInstance: Ref<    {
bind: () => void;
destroy: () => void;
getPan: () => {
x: number;
y: number;
};
getScale: () => number;
getOptions: () => PanzoomOptions;
handleDown: (event: PointerEvent) => void;
handleMove: (event: PointerEvent) => void;
handleUp: (event: PointerEvent) => void;
pan: (x: number | string, y: number | string, panOptions?: PanOptions, originalEvent?: PointerEvent) => CurrentValues;
reset: (resetOptions?: PanzoomOptions) => CurrentValues;
resetStyle: () => void;
setOptions: (options?: PanzoomOptions) => void;
setStyle: (name: string, value: string) => void;
zoom: (scale: number, zoomOptions?: ZoomOptions, originalEvent?: PointerEvent) => CurrentValues;
zoomIn: (zoomOptions?: ZoomOptions) => CurrentValues;
zoomOut: (zoomOptions?: ZoomOptions) => CurrentValues;
zoomToPoint: (scale: number, point: {
clientX: number;
clientY: number;
}, zoomOptions?: ZoomOptions, originalEvent?: PointerEvent) => CurrentValues;
zoomWithWheel: (event: WheelEvent, zoomOptions?: ZoomOptions) => CurrentValues;
} | null, PanzoomObject | {
bind: () => void;
destroy: () => void;
getPan: () => {
x: number;
y: number;
};
getScale: () => number;
getOptions: () => PanzoomOptions;
handleDown: (event: PointerEvent) => void;
handleMove: (event: PointerEvent) => void;
handleUp: (event: PointerEvent) => void;
pan: (x: number | string, y: number | string, panOptions?: PanOptions, originalEvent?: PointerEvent) => CurrentValues;
reset: (resetOptions?: PanzoomOptions) => CurrentValues;
resetStyle: () => void;
setOptions: (options?: PanzoomOptions) => void;
setStyle: (name: string, value: string) => void;
zoom: (scale: number, zoomOptions?: ZoomOptions, originalEvent?: PointerEvent) => CurrentValues;
zoomIn: (zoomOptions?: ZoomOptions) => CurrentValues;
zoomOut: (zoomOptions?: ZoomOptions) => CurrentValues;
zoomToPoint: (scale: number, point: {
clientX: number;
clientY: number;
}, zoomOptions?: ZoomOptions, originalEvent?: PointerEvent) => CurrentValues;
zoomWithWheel: (event: WheelEvent, zoomOptions?: ZoomOptions) => CurrentValues;
} | null>;
reset: () => CurrentValues | undefined;
zoomIn: () => CurrentValues | undefined;
zoomOut: () => CurrentValues | undefined;
cursorClass: Ref<string, string>;
}, {}, {}, {}, ComponentOptionsMixin, ComponentOptionsMixin, {
onCornerClick: (...args: any[]) => void;
"update:scale": (...args: any[]) => void;
zoomchange: (...args: any[]) => void;
"update:lockLine": (...args: any[]) => void;
}, string, PublicProps, Readonly<SketchRulerProps> & Readonly<{
onOnCornerClick?: ((...args: any[]) => any) | undefined;
"onUpdate:scale"?: ((...args: any[]) => any) | undefined;
onZoomchange?: ((...args: any[]) => any) | undefined;
"onUpdate:lockLine"?: ((...args: any[]) => any) | undefined;
}>, {
scale: number;
width: number;
height: number;
gridRatio: number;
showShadowText: boolean;
canvasWidth: number;
canvasHeight: number;
lines: LineType;
isShowReferLine: boolean;
rate: number;
snapThreshold: number;
snapsObj: LineType;
lockLine: boolean;
thick: number;
showRuler: boolean;
paddingRatio: number;
autoCenter: boolean;
shadow: ShadowType;
selfHandle: boolean;
}, {}, {}, {}, string, ComponentProvideOptions, false, {}, HTMLDivElement>;

declare function __VLS_template(): {
    attrs: Partial<{}>;
    slots: {
        btn?(_: {
            reset: () => CurrentValues | undefined;
            zoomIn: () => CurrentValues | undefined;
            zoomOut: () => CurrentValues | undefined;
        }): any;
        default?(_: {}): any;
    };
    refs: {};
    rootEl: HTMLDivElement;
};

declare type __VLS_TemplateResult = ReturnType<typeof __VLS_template>;

declare type __VLS_WithTemplateSlots<T, S> = T & {
    new (): {
        $slots: S;
    };
};

declare const _default: __VLS_WithTemplateSlots<typeof __VLS_component, __VLS_TemplateResult["slots"]>;
export default _default;

declare interface LineType {
    h: number[];
    v: number[];
}

export declare interface PaletteType {
    bgColor?: string;
    longfgColor?: string;
    fontColor?: string;
    fontShadowColor?: string;
    shadowColor?: string;
    lineColor?: string;
    lineType?: string;
    lockLineColor?: string;
    borderColor?: string;
    hoverBg?: string;
    hoverColor?: string;
    cornerActiveColor?: string;
}

declare interface ShadowType {
    x: number;
    y: number;
    width: number;
    height: number;
}

export declare interface SketchRulerProps {
    showRuler?: boolean;
    eyeIcon?: string;
    closeEyeIcon?: string;
    scale?: number;
    rate?: number;
    thick?: number;
    palette: PaletteType;
    width?: number;
    height?: number;
    paddingRatio?: number;
    autoCenter?: boolean;
    shadow?: ShadowType;
    lines?: LineType;
    isShowReferLine?: boolean;
    canvasWidth?: number;
    canvasHeight?: number;
    snapsObj?: LineType;
    snapThreshold?: number;
    gridRatio?: number;
    lockLine?: boolean;
    selfHandle?: boolean;
    showShadowText?: boolean;
    panzoomOption?: PanzoomOptions;
}

export { }
