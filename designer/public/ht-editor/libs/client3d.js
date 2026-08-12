/*
 * client3d.js - v5.0.2-dev1
 * Compiled Wed Sep 06 2023 17:02:53 GMT+0800 (GMT+08:00)
 * Copyright (c) 2023 www.hightopo.com
 */

var hteditor3d = function () {
    "use strict";

    function q(q) {
        return /^scenes/.test(q)
    }

    function a(q) {
        return /^models/.test(q)
    }

    function k(q) {
        return /^symbols/.test(q)
    }

    function w(q) {
        return /^assets/.test(q)
    }

    function I(q) {
        return /^materials/.test(q)
    }

    function K(q) {
        return /\.obj$/i.test(q)
    }

    function b(q) {
        return /\.mtl$/i.test(q)
    }

    function s(q) {
        return /\.(png|jpg|gif|jpeg|bmp|svg|hdr)$/i.test(q)
    }

    function u(q) {
        return /\.json$/i.test(q)
    }

    function C(q) {
        return /\.fbx$/i.test(q)
    }

    function L(q) {
        return /\.gltf$/i.test(q)
    }

    function j(q) {
        return /\.glb$/i.test(q)
    }

    function G(q) {
        return /\.hdr$/i.test(q)
    }

    function X(q) {
        return q && "object" == typeof q
    }

    function Z(q) {
        return "string" == typeof q || q instanceof String
    }

    function r(q) {
        var a = ht.Default.getImageMap();
        if (a[q]) return delete a[q], !0;
        var k = ht.Default.getCompTypeMap();
        if (k[q]) return delete k[q], !0;
        var w = ht.Default.getShape3dModelMap();
        return !!w[q] && (delete w[q], !0)
    }

    function W(q, a, k, w) {
        var I = new ht.Tab;
        return I.setName(a), I.setView(k), q.getTabModel().add(I), w && q.getTabModel().sm().ss(I), I
    }

    function T(q) {
        var a = new RegExp("(^|&)" + q + "=([^&]*)(&|$)"),
            k = window.location.search.substr(1).match(a);
        return null != k ? decodeURIComponent(k[2]) : null
    }

    function p(q, a, k) {
        if (q instanceof Array)
            for (var w = 0, I = q.length; w < I; w++) p(q[w], a, k);
        else if (q instanceof Object)
            for (var K in q)
                if (q.hasOwnProperty(K)) {
                    var b = q[K];
                    if (!X(b)) continue;
                    if (Z(b.func) && M.test(b.func)) {
                        var s = b.func.slice(5);
                        a && void 0 === a[s] && (a[s] = K), k && void 0 === k[s] && (k[s] = b.value)
                    } else p(b, a, k)
                }
    }

    function o(q, a) {
        var k = arguments.length > 2 && void 0 !== arguments[2] ? arguments[2] : H.encodeJSON;
        return ht.Default.stringify(q, a, k)
    }

    function g(q) {
        ht.Default.setImage("editor." + q, {
            dataBindings: [{
                attr: "text",
                valueType: "Multiline",
                defaultValue: q.toUpperCase()
            }, {
                attr: "textScale",
                valueType: "Number",
                defaultValue: 3 === q.length ? 1 : .7
            }],
            width: 64,
            height: 64,
            comps: [{
                type: "image",
                name: "editor.fileIcon",
                rect: [0, 0, 64, 64]
            }, {
                type: "text",
                text: {
                    func: "attr@text",
                    value: q.toUpperCase()
                },
                align: "right",
                vAlign: "top",
                font: "bold 20px sans-serif, Arial",
                anchorX: 1,
                anchorY: 1,
                scaleX: {
                    func: "attr@textScale"
                },
                scaleY: {
                    func: "attr@textScale"
                },
                rect: [15.38966, 42.52406, 44.54901, 19.55378]
            }]
        })
    }
    var R = hteditor,
        H = (R.getString, R.config);
    R.createTablePane = function () {
        var q = new ht.widget.TablePane;
        return q.getView().style.border = hteditor.config.color_line + " solid 1px", q
    };
    var f = R.FileNode.prototype.getFileImage;
    R.FileNode.prototype.getFileImage = function (q) {
        return /\.(json|tga|hdr|fbx|gltf|glb|htb|cob|obj|mtl|glsl)$/i.test(q) ? "editor." + this.postfix.toLowerCase() : f.call(this, q)
    };
    var n = function () {},
        M = /^attr@/,
        m = R.formatString,
        x = ht.Default,
        z = R.config;
    ! function (q) {
        for (var a in q) void 0 === z[a] && (z[a] = q[a])
    }({
        texureImage3D: !0,
        modelDialogClasses: {},
        modelViewSize: {
            width: 800,
            height: 500
        },
        imageViewSize: {
            width: 600,
            height: 500
        },
        materialViewSize: {
            width: 800,
            height: 500
        },
        sceneGridEnabled: !0,
        sceneGridBlockCount: 50,
        sceneGridBlockSize: 40,
        sceneGridColor: "rgb(191, 191, 191)",
        sceneEye: x.graph3dViewEye,
        sceneCenter: x.graph3dViewCenter,
        sceneNear: x.graph3dViewNear,
        sceneFar: x.graph3dViewFar,
        sceneBloom: !1,
        sceneBloomStrength: 1.5,
        sceneBloomThreshold: .55,
        sceneBloomRadius: .4,
        sceneDof: !1,
        sceneDofAperture: .025,
        sceneDofImage: null,
        sceneHighlightMode: "style",
        sceneHighlightType: "hard",
        sceneHighlightColor: ht.Style["highlight.color"],
        sceneHighlightWidth: ht.Style["highlight.width"],
        sceneHighlightGlow: .8,
        sceneHighlightStrength: 2,
        sceneDashEnable: !1,
        sceneSkyboxType: "sphere",
        sceneHeadlightEnable: !x.graph3dViewHeadlightDisabled,
        sceneHeadlightRange: x.graph3dViewHeadlightRange,
        sceneHeadlightColor: "rgb(255,255,255)",
        sceneHeadlightIntensity: x.graph3dViewHeadlightIntensity,
        sceneHeadlightAmbientIntensity: x.graph3dViewHeadlightAmbientIntensity,
        sceneHueSaturation: !1,
        sceneHueSaturationColorIndex: 0,
        sceneHueSaturationHue: [0, 0, 0, 0, 0, 0, 0],
        sceneHueSaturationSaturation: [0, 0, 0, 0, 0, 0, 0],
        sceneHueSaturationLightness: [0, 0, 0, 0, 0, 0, 0],
        sceneFogEnable: !x.graph3dViewFogDisabled,
        sceneFogMode: x.graph3dViewFogMode,
        sceneFogColor: x.graph3dViewFogColor,
        sceneFogNear: x.graph3dViewFogNear,
        sceneFogFar: x.graph3dViewFogFar,
        sceneFogDensity: x.graph3dViewFogDensity,
        sceneBatchBrightnessDisabled: x.graph3dViewBatchBrightnessDisabled,
        sceneBatchBlendDisabled: x.graph3dViewBatchBlendDisabled,
        sceneBatchColorDisabled: x.graph3dViewBatchColorDisabled,
        sceneBatchInstancedDisabled: x.graph3dViewBatchInstancedDisabled,
        sceneShadowEnabled: !1,
        sceneShadowDegreeX: ht.Default.graph3dViewShadowDegreeX,
        sceneShadowDegreeZ: ht.Default.graph3dViewShadowDegreeZ,
        sceneShadowIntensity: ht.Default.graph3dViewShadowIntensity,
        sceneShadowQuality: ht.Default.graph3dViewShadowQuality,
        sceneShadowType: ht.Default.graph3dViewShadowType,
        sceneShadowRadius: ht.Default.graph3dViewShadowRadius,
        sceneShadowBias: ht.Default.graph3dViewShadowBias,
        sceneEditHelperDisabled: !1,
        sceneOrthographic: !1,
        batchEditable: !0
    }), ["TitleScenes"].forEach(function (q) {
        null == z.expandedTitles[q] && (z.expandedTitles[q] = !0)
    });
    ht.Default.setImage("editor.axis", {
        background: "rgb(179,179,179)",
        width: 16,
        height: 16,
        blendMode: "override",
        comps: [{
            type: "shape",
            borderWidth: 1,
            borderColor: "rgb(138,138,138)",
            borderCap: "round",
            shadowColor: "#1ABC9C",
            points: [8, 2.48536, 8, 9, 1.08734, 12.01256, 8, 9, 14, 13]
        }]
    }), ht.Default.setImage("editor.cube.image", {
        width: 100,
        height: 100,
        comps: [{
            type: "rect",
            background: "rgba(156,156,156,0.4)",
            rect: [0, 0, 100, 100]
        }]
    }), ht.Default.setImage("editor.sphere.image", {
        width: 100,
        height: 100,
        comps: [{
            type: "oval",
            background: "rgba(156,156,156,0.4)",
            rect: [0, 0, 100, 100]
        }]
    }), ht.Default.setImage("editor.floor", {
        background: "rgb(0,0,0)",
        width: 16,
        height: 16,
        comps: [{
            type: "parallelogram",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            rect: [1.5056, 9.71598, 12.98879, 3.75]
        }, {
            type: "shape",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            borderCap: "round",
            opacity: .2,
            points: [14.49439, 9.71598, 14.49439, 1.61266, 4.70858, 1.61266, 4.70858, 9.71598]
        }]
    }), ht.Default.setImage("editor.wall", {
        background: "rgb(0,0,0)",
        width: 16,
        height: 16,
        comps: [{
            type: "shape",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            opacity: .2,
            points: [2, 15, 10, 15, 14, 11]
        }, {
            type: "shape",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            points: [6, 2, 14, 2, 14, 11, 6, 11, 2, 15, 2, 6, 6, 2, 6, 11]
        }]
    }), ht.Default.setImage("editor.cancel", {
        background: "red",
        width: 16,
        height: 16,
        comps: [{
            type: "circle",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            shadowColor: "#1ABC9C",
            rect: [1, 1, 14, 14]
        }, {
            type: "shape",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            borderCap: "round",
            points: [10.94702, 4.96729, 4.99893, 11.04194]
        }, {
            type: "shape",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            borderCap: "round",
            points: [5, 4.96729, 11.29505, 11.29505]
        }]
    }), ht.Default.setImage("editor.pipeline", {
        background: "rgb(82,82,82)",
        width: 16,
        height: 16,
        blendMode: "override",
        comps: [{
            type: "shape",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            points: [14.11524, 6.61831, 14.11524, 6.61831, 9.30492, 12.81031, 8.07676, 10.14929, 6.8486, 7.48826, 9.56079, 4.00846, 6.33686, 4.00846, 3.11293, 4.00846, .86129, 8.92112, .86129, 8.92112],
            segments: [1, 4, 4, 4]
        }, {
            type: "shape",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            opacity: .4,
            points: [15.28357, 7.93382, 15.28357, 7.93382, 10.47326, 14.12582, 9.24509, 11.46479, 8.01693, 8.80377, 10.72913, 5.32397, 7.50519, 5.32397, 4.28126, 5.32397, 2.02963, 10.23663, 2.02963, 10.23663],
            segments: [1, 4, 4, 4]
        }]
    }), ht.Default.setImage("editor.doorWindow", {
        width: 16,
        height: 16,
        comps: [{
            type: "rect",
            borderWidth: 1,
            borderColor: "#979797",
            rect: [4.5, .5, 7, 15]
        }, {
            type: "shape",
            borderWidth: 1,
            borderColor: "#979797",
            points: [8.5469, 8, 10.42756, 8]
        }]
    }), ht.Default.setImage("editor.selectAll", {
        width: 16,
        height: 16,
        blendMode: "override",
        comps: [{
            type: "shape",
            background: "#212121",
            pixelPerfect: !0,
            points: [5.08505, 2.30851, 5.34194, 2.54013, 5.36281, 2.94207, 5.13131, 3.19905, 3.23923, 5.30136, 3.12348, 5.42963, 2.95996, 5.50486, 2.78724, 5.50931, 2.61452, 5.51376, 2.44734, 5.44704, 2.32514, 5.3249, 1.27399, 4.27375, 1.04384, 4.02676, 1.05073, 3.63654, 1.28944, 3.39783, 1.52816, 3.15911, 1.91838, 3.15222, 2.16537, 3.38237, 2.74645, 3.96261, 4.19452, 2.35476, 4.42613, 2.09788, 4.82807, 2.07701, 5.08505, 2.30851, 6.55499, 4.03829, 6.55499, 3.6923, 6.83969, 3.4076, 7.18568, 3.4076, 14.33353, 3.4076, 14.67952, 3.4076, 14.96423, 3.6923, 14.96423, 4.03829, 14.96423, 4.38428, 14.67952, 4.66898, 14.33353, 4.66898, 7.18568, 4.66898, 6.83969, 4.66898, 6.55499, 4.38428, 6.55499, 4.03829, 6.55499, 8.24291, 6.55499, 7.89692, 6.83969, 7.61222, 7.18568, 7.61222, 14.33353, 7.61222, 14.67952, 7.61222, 14.96423, 7.89692, 14.96423, 8.24291, 14.96423, 8.5889, 14.67952, 8.8736, 14.33353, 8.8736, 7.18568, 8.8736, 6.83969, 8.8736, 6.55499, 8.5889, 6.55499, 8.24291, 7.18568, 11.81684, 6.83969, 11.81684, 6.55499, 12.10154, 6.55499, 12.44753, 6.55499, 12.79352, 6.83969, 13.07822, 7.18568, 13.07822, 14.33353, 13.07822, 14.67952, 13.07822, 14.96423, 12.79352, 14.96423, 12.44753, 14.96423, 12.10154, 14.67952, 11.81684, 14.33353, 11.81684, 7.18568, 11.81684, 5.13131, 11.60829, 5.28758, 11.44203, 5.34191, 11.20344, 5.27302, 10.98591, 5.20413, 10.76838, 5.02236, 10.60456, 4.79887, 10.55857, 4.57537, 10.51259, 4.34369, 10.59134, 4.19452, 10.764, 2.7456, 12.37269, 2.16537, 11.79161, 1.91959, 11.554, 1.52308, 11.55726, 1.28124, 11.79887, 1.0394, 12.04048, 1.03577, 12.43699, 1.27315, 12.68299, 2.3243, 13.73414, 2.4465, 13.85628, 2.61368, 13.92299, 2.7864, 13.91855, 2.95911, 13.9141, 3.12263, 13.83886, 3.23839, 13.7106, 5.13046, 11.60829],
            segments: [1, 4, 2, 4, 4, 2, 4, 4, 2, 2, 4, 5, 1, 4, 2, 4, 4, 2, 4, 5, 1, 4, 2, 4, 4, 2, 4, 5, 1, 4, 4, 2, 4, 4, 2, 5, 1, 4, 4, 4, 2, 2, 4, 4, 2, 4, 4, 2, 5]
        }]
    }), ht.Default.setImage("editor.cube", {
        background: "rgb(82,82,82)",
        width: 16,
        height: 16,
        blendMode: "override",
        comps: [{
            type: "rect",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            shadowColor: "#1ABC9C",
            rect: [2.48996, 3.51341, 9.97783, 10.97318]
        }, {
            type: "shape",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            opacity: .4,
            points: [15, 1, 12.39369, 3.59417, 12.39369, 14.44, 15, 11.41017, 15, 1, 5.46231, 1, 2.50972, 3.59417]
        }]
    }), ht.Default.setImage("editor.sphere", {
        background: "rgb(179,179,179)",
        width: 16,
        height: 16,
        comps: [{
            type: "oval",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            rect: [1.49999, 1.5, 13, 13]
        }, {
            type: "arc",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            borderCap: "round",
            arcFrom: 3.22886,
            arcTo: 5.96903,
            arcClose: !1,
            arcOval: !0,
            opacity: .4,
            rotation: 2.4578,
            rect: [.10993, 1.43114, 12.94079, 10.68594]
        }]
    }), ht.Default.setImage("editor.cylinder", {
        background: "rgb(179,179,179)",
        width: 16,
        height: 16,
        comps: [{
            type: "oval",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            rect: [1.5, 1.5, 13, 9.5]
        }, {
            type: "arc",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            arcClose: !1,
            arcOval: !0,
            opacity: .4,
            rotation: 3.14159,
            rect: [1.49362, 6, 13, 9.5]
        }, {
            type: "shape",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            opacity: .4,
            points: [1.49362, 10.75, 1.48724, 6.35177]
        }, {
            type: "shape",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            opacity: .4,
            points: [14.5, 10.75, 14.5, 6.10177]
        }]
    }), ht.Default.setImage("editor.scene.cone", {
        background: "rgb(82,82,82)",
        width: 16,
        height: 16,
        comps: [{
            type: "oval",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            opacity: .4,
            rect: [3, 10, 10, 3.48659]
        }, {
            type: "shape",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            borderCap: "round",
            points: [3, 12, 8, 2, 13, 12]
        }]
    }), ht.Default.setImage("editor.scene.torus", {
        background: "rgb(82,82,82)",
        width: 16,
        height: 16,
        comps: [{
            type: "oval",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            opacity: .4,
            rect: [1, 4, 14, 10]
        }, {
            type: "oval",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            rect: [2, 2, 12, 10]
        }]
    }), ht.Default.setImage("editor.scene.triangle", {
        background: "rgb(82,82,82)",
        width: 16,
        height: 16,
        comps: [{
            type: "triangle",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            opacity: .4,
            rect: [1.53493, 5, 13.1269, 9.05393]
        }, {
            type: "triangle",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            rect: [2.09838, 2, 12, 10]
        }]
    }), ht.Default.setImage("editor.scene.rightTriangle", {
        background: "rgb(82,82,82)",
        width: 16,
        height: 16,
        comps: [{
            type: "rightTriangle",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            opacity: .4,
            rect: [1.53493, 5, 13.1269, 9.05393]
        }, {
            type: "rightTriangle",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            rect: [3.09838, 2, 10.90162, 10]
        }]
    }), ht.Default.setImage("editor.scene.parallelogram", {
        background: "rgb(82,82,82)",
        width: 16,
        height: 16,
        comps: [{
            type: "parallelogram",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            opacity: .4,
            rect: [1, 5, 14, 8]
        }, {
            type: "parallelogram",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            rect: [1, 2, 13, 8]
        }]
    }), ht.Default.setImage("editor.scene.trapezoid", {
        background: "rgb(82,82,82)",
        width: 16,
        height: 16,
        comps: [{
            type: "trapezoid",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            opacity: .4,
            rect: [.90479, 6, 14, 8]
        }, {
            type: "trapezoid",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            rect: [1.40479, 2.5, 13, 9]
        }]
    }), ht.Default.setImage("editor.scene.rect", {
        background: "rgb(82,82,82)",
        width: 16,
        height: 16,
        comps: [{
            type: "rect",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            opacity: .4,
            rect: [1.53493, 5, 13.1269, 9.05393]
        }, {
            type: "rect",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            rect: [3.2894, 2.39375, 9.61796, 9.2125]
        }]
    }), ht.Default.setImage("editor.scene.roundRect", {
        background: "rgb(82,82,82)",
        width: 16,
        height: 16,
        comps: [{
            type: "roundRect",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            opacity: .4,
            rect: [1.53493, 5, 13.1269, 9.05393]
        }, {
            type: "roundRect",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            rect: [3.2894, 2.39375, 9.61796, 9.2125]
        }]
    }), ht.Default.setImage("editor.scene.star", {
        background: "rgb(82,82,82)",
        width: 16,
        height: 16,
        comps: [{
            type: "star",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            opacity: .4,
            rect: [1.59838, 2, 13, 13]
        }, {
            type: "star",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            rect: [2, 1, 12, 12]
        }]
    }), ht.Default.setImage("editor.scene.box", {
        background: "rgb(82,82,82)",
        width: 16,
        height: 16,
        comps: [{
            type: "rect",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            shadowColor: "#1ABC9C",
            opacity: .4,
            rect: [2.48996, 3.51341, 9.97783, 10.97318]
        }, {
            type: "shape",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            opacity: .4,
            points: [15, 1, 12.39369, 3.59417, 12.39369, 14.44, 15, 11.41017, 15, 1, 5.46231, 1, 2.50972, 3.59417]
        }]
    }), ht.Default.setImage("editor.scene.billboard", {
        background: "rgb(82,82,82)",
        width: 16,
        height: 16,
        comps: [{
            type: "shape",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            borderCap: "round",
            points: [8, 13, 6, 10, 3, 10, 3, 3, 13, 3, 13, 10, 10, 10, 8, 13]
        }]
    }), ht.Default.setImage("editor.scene.plane", {
        background: "rgb(82,82,82)",
        width: 16,
        height: 16,
        comps: [{
            type: "shape",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            borderCap: "round",
            points: [1.5, 12.5, 5.5, 7.5, 14.5, 7.5, 10.5, 12.5, 1.5, 12.5]
        }]
    }), ht.Default.setImage("editor.light", {
        background: "rgb(179,179,179)",
        width: 16,
        height: 16,
        comps: [{
            type: "shape",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            points: [14, 4, 12, 6, 8, 1.51088, 8, 4.69022, 2, 4, 4, 6],
            segments: [1, 2, 1, 2, 1, 2]
        }, {
            type: "oval",
            borderWidth: 1,
            borderColor: "rgb(255,255,255)",
            rect: [4, 6, 8, 8]
        }]
    }), ht.Default.setImage("editor.envmap.t", {
        width: 20,
        height: 20,
        comps: [{
            type: "shape",
            borderWidth: 1,
            borderColor: "#979797",
            borderCap: "round",
            points: [10, 1, 10, 19]
        }, {
            type: "shape",
            borderWidth: 1,
            borderColor: "#979797",
            borderCap: "round",
            points: [10, 10, 18, 10]
        }]
    }), ht.Default.setImage("editor.envmap.l", {
        width: 20,
        height: 20,
        comps: [{
            type: "shape",
            borderWidth: 1,
            borderColor: "#979797",
            borderCap: "round",
            points: [10, 1, 10, 10, 18, 10]
        }]
    }), ht.Default.setImage("editor.refresh", {
        width: 16,
        height: 16,
        comps: [{
            type: "shape",
            background: "#ffffff",
            pixelPerfect: !0,
            scaleX: -1,
            rotation: 4.84125,
            points: [12.18494, 3.75768, 11.01446, 2.62418, 9.45824, 2, 7.80294, 2, 7.80284, 2, 7.80306, 2, 7.80292, 2, 6.14776, 2, 4.59136, 2.62428, 3.42098, 3.75768, 3.4156, 3.7629, 3.39892, 3.7791, 3.37498, 3.80242, 3.56406, 3.11914, 3.6223, 2.9087, 3.4933, 2.69236, 3.276, 2.63596, 3.0586, 2.57952, 2.8353, 2.70446, 2.77706, 2.91492, 2.19456, 5.02004, 2.15688, 5.1562, 2.19708, 5.30144, 2.29998, 5.4011, 2.3774, 5.47606, 2.48144, 5.51664, 2.58806, 5.51664, 2.62322, 5.51664, 2.65864, 5.51222, 2.6935, 5.5032, 4.86732, 4.93912, 5.08464, 4.88272, 5.21362, 4.6664, 5.15538, 4.45594, 5.09716, 4.2455, 4.87374, 4.1206, 4.65644, 4.17698, 3.95142, 4.35992, 3.97002, 4.34182, 3.98576, 4.32654, 3.99708, 4.31554, 5.01368, 3.3311, 6.36528, 2.78892, 7.80292, 2.78896, 9.24056, 2.78896, 10.5922, 3.3311, 11.60876, 4.31554, 13.70732, 6.34774, 13.70732, 9.65436, 11.60878, 11.68654, 9.51024, 13.71866, 6.09566, 13.71872, 3.99708, 11.68654, 3.46488, 11.17114, 3.05778, 10.56844, 2.7871, 9.89524, 2.70548, 9.69224, 2.46932, 9.59174, 2.25968, 9.67072, 2.05002, 9.74976, 1.94622, 9.97844, 2.02786, 10.18146, 2.33974, 10.9572, 2.80842, 11.65132, 3.42098, 12.24444, 4.59142, 13.37792, 6.14766, 14.0021, 7.80294, 14.0021, 9.45826, 14.0021, 11.01448, 13.37788, 12.18494, 12.24444, 13.35542, 11.111, 14.00002, 9.604, 14, 8.00106, 14, 6.39812, 13.35542, 4.89114, 12.18494, 3.75768],
            segments: [1, 4, 4, 4, 4, 2, 4, 4, 2, 4, 4, 4, 2, 4, 4, 2, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5]
        }]
    }), ht.Default.setImage("editor.material.logo", {
        modified: "Mon Jan 30 2023 16:03:40 GMT+0800 (GMT+08:00)",
        width: 16,
        height: 16,
        comps: [{
            type: "shape",
            borderWidth: 1,
            borderColor: "#ffffff",
            borderCap: "round",
            points: [4.98775, 4.45122, 5.90406, 3.77865, 7.0088, 3.41041, 8.14537, 3.39868, 11.17676, 3.39868, 13.67121, 5.89313, 13.67121, 8.92451, 13.67121, 11.95589, 11.17676, 14.45035, 8.14537, 14.45035, 2.61955, 14.45035, 2.61955, 8.97714, 2.60784, 8.20711, 2.75847, 7.44318, 3.06161, 6.73523, 4.65094, 4.39859, 4.65518, 3.59799, 4.17314, 2.87031, 3.43428, 2.56196, 2.69542, 2.25362, 1.8391, 2.42277, 1.27298, 2.98888, .70687, 3.55501, .53771, 4.41133, .84605, 5.15019, 1.1544, 5.88905, 1.88208, 6.37109, 2.68268, 6.36684, 4.65094, 6.36684],
            segments: [1, 4, 4, 4, 2, 2, 4, 1, 4, 4, 4, 4, 2, 5]
        }, {
            type: "shape",
            borderWidth: 1,
            borderColor: "#ffffff",
            opacity: .4,
            points: [1.35649, 2.94609, 2.94582, 1.34622, 2.94582, 1.34622, 3.3134, 1.00896, 3.79422, .82115, 4.29307, .81996, 4.8167, .81715, 5.32038, 1.02343, 5.69165, 1.39272, 6.0629, 1.76199, 6.27186, 2.26457, 6.27185, 2.78821, 6.27185, 2.78821, 7.24735, 2.13885, 8.39445, 1.79435, 9.56631, 1.79882, 12.6633, 1.7346, 15.26565, 4.22774, 15.33423, 7.32465, 15.29762, 9.26685, 13.90114, 11.62854, 12.20591, 12.57705, 8.96636, 8.72453, 8.96636, 11.99793, 8.96636, 12.45408, 8.591, 12.82943, 8.13483, 12.82943, 7.27175, 12.82943, 7.27175, 8.72453, 6.13502, 8.72453, 5.93689, 8.72453, 5.74668, 8.64574, 5.60659, 8.50565, 5.4665, 8.36556, 5.38771, 8.17535, 5.38771, 7.97722, 5.38771, 7.06152, 10.12416, 7.06152, 10.53987, 7.06152, 10.88197, 7.40361, 10.88197, 7.81935, 10.88197, 7.97722, 10.88199, 8.17718, 10.80175, 8.36901, 10.65938, 8.5094, 10.51702, 8.6498, 10.32409, 8.72735, 10.12416, 8.72453],
            segments: [1, 2, 2, 4, 4, 4, 2, 4, 4, 4, 1, 2, 4, 2, 2, 2, 4, 4, 2, 2, 4, 2, 4, 4, 5]
        }]
    }), ht.Default.setImage("editor.fileIcon", {
        dataBindings: [{
            attr: "text",
            valueType: "Multiline"
        }, {
            attr: "textScale",
            valueType: "Number"
        }],
        width: 64,
        height: 64,
        fitSize: !0,
        animation: {},
        comps: [{
            type: "shape",
            background: "#000000",
            points: [56.21111, 19.11398, 40.59837, 2.34525, 40.23096, 1.94622, 39.70418, 1.92216, 39.15478, 1.92216, 8.11505, 1.92216, 7.04265, 1.92216, 6.07649, 2.35795, 6.07649, 3.40366, 6.07649, 60.1542, 6.07649, 61.1999, 7.04299, 62.07784, 8.11505, 62.07784, 13.93496, 62.07784, 15.00736, 62.07784, 15.87481, 61.28546, 15.87481, 60.23975, 15.87481, 59.19405, 15.00702, 58.40166, 13.93496, 58.40166, 9.84618, 58.40166, 9.84618, 5.59867, 35.20824, 5.59867, 35.20824, 21.86677, 35.20824, 22.91248, 36.27174, 23.64538, 37.3438, 23.64538, 52.6875, 23.64538, 52.6875, 35.56255, 52.6875, 36.60826, 53.67148, 37.45412, 54.74388, 37.45412, 55.81629, 37.45412, 56.80027, 36.60793, 56.80027, 35.56255, 56.80027, 20.28301, 56.80027, 19.81747, 56.52917, 19.46155, 56.21111, 19.11398, 56.21111, 19.11398, 39.32101, 6.67546, 51.68947, 19.9692, 39.32101, 19.9692, 39.32101, 6.67546, 39.32101, 6.67546],
            segments: [1, 2, 4, 2, 4, 2, 4, 2, 4, 4, 2, 2, 2, 2, 4, 2, 2, 4, 4, 2, 4, 2, 1, 2, 2, 2, 2]
        }]
    }), "json|tga|hdr|fbx|gltf|glb|htb|cob|obj|mtl|glsl".split("|").forEach(function (q) {
        g(q)
    });
    var V = (function () {
            function q(q) {
                this.value = q
            }

            function a(a) {
                function k(q, a) {
                    return new Promise(function (k, I) {
                        var s = {
                            key: q,
                            arg: a,
                            resolve: k,
                            reject: I,
                            next: null
                        };
                        b ? b = b.next = s : (K = b = s, w(q, a))
                    })
                }

                function w(k, K) {
                    try {
                        var b = a[k](K),
                            s = b.value;
                        s instanceof q ? Promise.resolve(s.value).then(function (q) {
                            w("next", q)
                        }, function (q) {
                            w("throw", q)
                        }) : I(b.done ? "return" : "normal", b.value)
                    } catch (q) {
                        I("throw", q)
                    }
                }

                function I(q, a) {
                    switch (q) {
                        case "return":
                            K.resolve({
                                value: a,
                                done: !0
                            });
                            break;
                        case "throw":
                            K.reject(a);
                            break;
                        default:
                            K.resolve({
                                value: a,
                                done: !1
                            })
                    }
                    K = K.next, K ? w(K.key, K.arg) : b = null
                }
                var K, b;
                this._invoke = k, "function" != typeof a.return && (this.return = void 0)
            }
            "function" == typeof Symbol && Symbol.asyncIterator && (a.prototype[Symbol.asyncIterator] = function () {
                return this
            }), a.prototype.next = function (q) {
                return this._invoke("next", q)
            }, a.prototype.throw = function (q) {
                return this._invoke("throw", q)
            }, a.prototype.return = function (q) {
                return this._invoke("return", q)
            }
        }(), function (q, a) {
            if (!(q instanceof a)) throw new TypeError("Cannot call a class as a function")
        }),
        v = function () {
            function q(q, a) {
                for (var k = 0; k < a.length; k++) {
                    var w = a[k];
                    w.enumerable = w.enumerable || !1, w.configurable = !0, "value" in w && (w.writable = !0), Object.defineProperty(q, w.key, w)
                }
            }
            return function (a, k, w) {
                return k && q(a.prototype, k), w && q(a, w), a
            }
        }(),
        E = function (q, a) {
            if ("function" != typeof a && null !== a) throw new TypeError("Super expression must either be null or a function, not " + typeof a);
            q.prototype = Object.create(a && a.prototype, {
                constructor: {
                    value: q,
                    enumerable: !1,
                    writable: !0,
                    configurable: !0
                }
            }), a && (Object.setPrototypeOf ? Object.setPrototypeOf(q, a) : q.__proto__ = a)
        },
        Q = function (q, a) {
            if (!q) throw new ReferenceError("this hasn't been initialised - super() hasn't been called");
            return !a || "object" != typeof a && "function" != typeof a ? q : a
        },
        l = function () {
            function q(a, k) {
                var w = this;
                V(this, q), this.handler = a, this.editor = k, this.cookie = 0, this.callbacks = {}, this.cmds = {};
                var I = hteditor.config.host || window.location.hostname,
                    K = hteditor.config.port || window.location.port,
                    b = window.location.protocol + "//" + I + ":" + K;
                this.socket = io.connect(b), this.socket.on("connect", function () {
                    w.handler({
                        type: "connected",
                        message: b
                    })
                }), this.socket.on("disconnect", function () {
                    w.handler({
                        type: "disconnected",
                        message: b
                    })
                }), this.socket.on("fileChanged", function (q) {
                    w.handler({
                        type: "fileChanged",
                        path: q.path,
                        event: q.event
                    })
                }), this.socket.on("operationDone", function (q, a) {
                    w.handleRespone(q, a)
                }), this.socket.on("download", function (q) {
                    w.handler({
                        type: "download",
                        path: q
                    })
                }), this.socket.on("confirm", function (q, a) {
                    w.handler({
                        type: "confirm",
                        path: q,
                        datas: a
                    })
                })
            }
            return q.prototype.request = function (q, a, k) {
                if ("uploadLocalFile" === q) return void this.requestHttp(q, a, k);
                var w = ++this.cookie;
                this.callbacks[w] = k, this.cmds[w] = q;
                var I = this.editor.sid;
                this.socket.emit(q, w, a, I ? {
                    sid: I
                } : null);
                var K = q;
                a && (ht.Default.isString(a) ? K = q + ": " + a : a.path && (K = q + ": " + a.path)), this.handler({
                    type: "request",
                    message: K,
                    cmd: q,
                    data: a
                })
            }, q.prototype.handleRespone = function (q, a) {
                var k = this.callbacks[q],
                    w = this.cmds[q];
                delete this.callbacks[q], delete this.cmds[q], k && k(a), this.handler({
                    type: "response",
                    message: w,
                    cmd: w,
                    data: a
                })
            }, q.prototype.requestHttp = function (q, a, k) {
                var w = ++this.cookie;
                this.callbacks[w] = k, this.cmds[w] = q;
                var I = this.editor.sid;
                this[q](w, a, I || null, k);
                var K = q;
                a && (ht.Default.isString(a) ? K = q + ": " + a : a.path && (K = q + ": " + a.path)), this.handler({
                    type: "request",
                    message: K,
                    cmd: q,
                    data: a
                })
            }, q.prototype.uploadLocalFile = function (q, a, k) {
                var w = this,
                    I = new FormData,
                    K = this.url + "/uploadFile",
                    b = a.path;
                K += "?sid=" + k, I.append("path", a.path), I.append("file", a.file), w.send("POST", K, I, function (a) {
                    var k = a.target;
                    if (200 === k.status || 0 === k.status) {
                        var I = k.response;
                        if (w.handleRespone(q, !0), I) {
                            var K = JSON.parse(I);
                            w.handler({
                                type: "confirm",
                                path: K.tempName,
                                datas: K.paths
                            })
                        } else w.handler({
                            type: "fileChanged",
                            path: b
                        })
                    }
                })
            }, q.prototype.send = function (q, a, k, w) {
                var I = new ht.Request,
                    K = {};
                K.url = encodeURI(a), K.method = q, K.data = k, w && (I.onload = w), I.send(K)
            }, q
        }(),
        _ = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "scenes"))
            }
            return E(a, q), a.prototype.initTreeMenu = function (q, a) {
                this.addLocateTreeFileItem(a), this.addNewFolderItem(a, this.tree), this.addRenameItem(a, this.tree), this.addDeleteItem(a, this.tree)
            }, a.prototype.initListMenu = function (q, a) {
                this.addLocateListFileItem(a), this.addNewFolderItem(a, this.list), this.addCopyItem(a), this.addPasteItem(a), this.addRenameItem(a, this.list), this.addDeleteItem(a, this.list), this.addExportItem(a)
            }, a
        }(hteditor.Explorer),
        F = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "models"))
            }
            return E(a, q), a.prototype.initTreeMenu = function (q, a) {
                this.addLocateTreeFileItem(a), this.addNewFolderItem(a, this.tree), this.addRenameItem(a, this.tree), this.addDeleteItem(a, this.tree)
            }, a.prototype.initListMenu = function (q, a) {
                this.addLocateListFileItem(a), this.addNewFolderItem(a, this.list), this.addCopyItem(a), this.addPasteItem(a), this.addRenameItem(a, this.list), this.addDeleteItem(a, this.list), this.addExportItem(a)
            }, a
        }(hteditor.Explorer),
        S = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "symbols"))
            }
            return E(a, q), a.prototype.initTreeMenu = function (q, a) {
                this.addLocateTreeFileItem(a), this.addNewFolderItem(a, this.tree), this.addRenameItem(a, this.tree), this.addDeleteItem(a, this.tree)
            }, a.prototype.initListMenu = function (q, a) {
                this.addLocateListFileItem(a), this.addNewFolderItem(a, this.list), this.addCopyItem(a), this.addPasteItem(a), this.addRenameItem(a, this.list), this.addDeleteItem(a, this.list), this.addExportItem(a)
            }, a
        }(hteditor.Explorer),
        d = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "assets"))
            }
            return E(a, q), a.prototype.initTreeMenu = function (q, a) {
                this.addLocateTreeFileItem(a), this.addNewFolderItem(a, this.tree), this.addRenameItem(a, this.tree), this.addDeleteItem(a, this.tree)
            }, a.prototype.initListMenu = function (q, a) {
                this.addLocateListFileItem(a), this.addNewFolderItem(a, this.list), this.addCopyItem(a), this.addPasteItem(a), this.addRenameItem(a, this.list), this.addDeleteItem(a, this.list)
            }, a
        }(hteditor.Explorer),
        e = R.getString,
        B = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "materials"))
            }
            return E(a, q), a.prototype.addFileNode = function (a, k, w, I) {
                I && "json" === I.postfix ? I.fileType = "material" : I && "png" === I.postfix && (I.fileType = "unknow");
                var K = q.prototype.addFileNode.call(this, a, k, w, I);
                if (!k) return K;
                var b = void 0,
                    s = void 0,
                    C = void 0,
                    L = void 0;
                return L = I && I.uuid && K.postfix ? K.uuid + "." + K.postfix : K.url, u(w) && "materials" === a && (b = "material", s = L.replace(".json", ".png"), K.s("label", w.replace(".json", ""))), X(I) && (Z(I.fileType) ? (b = I.fileType, C = I.fileIcon, s = I.fileImage || s || C, s || "dir" !== I.fileType || (s = "editor.dir"), K.a(I.attrs), K.s(I.styles)) : (!0 === I.dir || null == I.dir && null == I.fileType) && (b = "dir", s = "editor.dir")), s || (s = "editor.unknown"), K.fileType = b || "unknow", K.setIcon(C), K.setImage(s), K
            }, a.prototype.initTreeMenu = function (q, a) {
                this.addLocateTreeFileItem(a), this.addNewFolderItem(a, this.tree), this.addRenameItem(a, this.tree), this.addDeleteItem(a, this.tree)
            }, a.prototype.initListMenu = function (q, a) {
                this.addLocateListFileItem(a), this.addNewFolderItem(a, this.list), this.addCopyItem(a), this.addPasteItem(a), this.addRenameItem(a, this.list), this.addDeleteItem(a, this.list), this.addExportItem(a)
            }, a.prototype.addExportItem = function (q) {
                var a = this,
                    k = {
                        id: "export",
                        label: e("editor.export"),
                        action: function () {
                            var q = [],
                                k = a.getFileListView().sm();
                            k.size() > 0 && (k.each(function (a) {
                                q.push(a.url)
                            }), a.editor.request("export", q))
                        },
                        visible: function () {
                            var q = a.getFileListView().sm().ld();
                            if (!q) return !1;
                            if (a.onDrawingDir(q)) return !1;
                            var k = q.fileType;
                            return "display" === k || "symbol" === k || "component" === k || "scene" === k || "model" === k || "ui" === k || "dir" === k || "material" === k
                        }
                    };
                return q.push(k), k
            }, a
        }(hteditor.Explorer),
        A = hteditor.getString,
        h = function (q) {
            function a(k) {
                V(this, a);
                var w = Q(this, q.call(this));
                w.editor = k, w.editor.menus.push(w);
                var I = [{
                    label: A("editor.newsceneview"),
                    action: function () {
                        w.editor.newScene()
                    }
                }, {
                    label: A("editor.newmodel"),
                    items: [{
                        label: A("editor.obj"),
                        action: function () {
                            w.editor.newOBJModel()
                        }
                    }, {
                        label: A("editor.fbx"),
                        action: function () {
                            w.editor.newFBXModel()
                        }
                    }, {
                        label: A("editor.gltf"),
                        action: function () {
                            w.editor.newGLTFModel()
                        }
                    }]
                }, {
                    label: A("editor.newmaterial"),
                    action: function () {
                        w.editor.newMaterial()
                    }
                }];
                return w.setItems(I), w
            }
            return E(a, q), a
        }(ht.widget.ContextMenu),
        J = function (q) {
            function a(k) {
                V(this, a);
                var w = Q(this, q.call(this, k.gv));
                return w.editor = k, w
            }
            return E(a, q), a.prototype.beginTransaction = function () {
                this._isBeginTransaction || (this._isBeginTransaction = !0, this.dm.beginTransaction())
            }, a.prototype.endTransaction = function () {
                this._isBeginTransaction && (this._isBeginTransaction = !1, this.dm.endTransaction())
            }, a.prototype.getDataAt = function (q) {
                return this.gv.getDataAt(q)
            }, a.prototype.getNodeAt = function (q) {
                var a = this.getDataAt(q);
                return a instanceof ht.Node ? a : null
            }, a.prototype.setUp = function () {
                q.prototype.setUp.call(this), this.sm.cs()
            }, a.prototype.$0$ = function (q) {}, a.prototype.drawRect = function (q, a, k, w) {
                a = this.toRect(a), w && (q.fillStyle = w), k && (q.strokeStyle = k), q.lineWidth = 1, q.beginPath(), q.rect(a.x, a.y, a.width, a.height), q.stroke(), w && q.fill(), k && q.stroke()
            }, a.prototype.drawPoint = function (q, a, k, w, I) {
                a = this.toPoint(a), I && (q.fillStyle = I), w && (q.strokeStyle = w), q.lineWidth = 1, q.beginPath(), q.arc(a.x, a.y, k, 0, 2 * Math.PI, !0), I && q.fill(), w && q.stroke()
            }, a.prototype.drawPoints = function (q, a, k, w, I) {
                for (var K = 0; K < a.length; K++) this.drawPoint(q, a[K], k, w, I)
            }, a.prototype.drawLine = function (q, a, k, w) {
                a = this.toPoint(a), k = this.toPoint(k), q.lineWidth = 1, q.strokeStyle = w, q.beginPath(), q.moveTo(a.x, a.y), q.lineTo(k.x, k.y), q.stroke()
            }, a.prototype.toRect = function (q) {
                var a = this.zoom;
                return {
                    x: q.x * a + this.tx,
                    y: q.y * a + this.ty,
                    width: q.width * a,
                    height: q.height * a
                }
            }, a.prototype.toPoint = function (q) {
                return {
                    x: q.x * this.zoom + this.tx,
                    y: q.y * this.zoom + this.ty
                }
            }, a.prototype.addData = function (q) {
                this.dm.beginTransaction(), this.dm.add(q), this.editor.sm.ss(q), this.editor.gv.setFocus(), this.editor.fireEvent("dataCreated", {
                    sceneView: this.editor.scene,
                    data: q
                }), this.dm.endTransaction()
            }, a.prototype.remove = function (q) {
                this.dm.remove(q)
            }, a.prototype.lp = function (q) {
                return this.gv.lp(q)
            }, v(a, [{
                key: "dm",
                get: function () {
                    return this.gv.dm()
                }
            }, {
                key: "sm",
                get: function () {
                    return this.gv.dm().sm()
                }
            }, {
                key: "zoom",
                get: function () {
                    return this.gv.getZoom()
                }
            }, {
                key: "tx",
                get: function () {
                    return this.gv.tx()
                }
            }, {
                key: "ty",
                get: function () {
                    return this.gv.ty()
                }
            }]), a
        }(ht.graph.Interactor),
        t = function (q) {
            function a(k, w, I) {
                V(this, a);
                var K = Q(this, q.call(this, k));
                return K.createFunc = w, K.points = [], K.segments = [], K.nextPoint = null, K.fill = I, K
            }
            return E(a, q), a.prototype.tearDown = function () {
                q.prototype.tearDown.call(this), this.points = [], this.segments = [], this.nextPoint = null
            }, a.prototype.$48$ = function (q) {
                this.nextPoint ? (this.nextPoint = null, this.editor.rulerView.validateCanvas()) : this.createShape()
            }, a.prototype.createShape = function () {
                if (this.points.length > 1) {
                    var q = this.createFunc(this.points, this.segments);
                    this.addData(q)
                }
                this.editor.resetInteractionState(), this.editor.pointsEditingMode = !0
            }, a.prototype.handle_mousedown = function (q) {
                this.handle_touchstart(q)
            }, a.prototype.handle_touchstart = function (q) {
                if (ht.Default.preventDefault(q), ht.Default.isLeftButton(q))
                    if (this.gv.setFocus(q), ht.Default.isDoubleClick(q)) this.createShape();
                    else {
                        var a = this.judgeNextPoint(q);
                        if (this.points.length) {
                            var k = this.points[this.points.length - 1];
                            if (Math.abs(k.x - a.x) < .01 && Math.abs(k.y - a.y) < .01) return
                        }
                        this.points.push(a), this.segments.push(this.nextPoint ? 2 : 1), this.nextPoint = this.judgeNextPoint(q), this.editor.rulerView.validateCanvas()
                    }
            }, a.prototype.handle_mousemove = function (q) {
                this.nextPoint && (this.nextPoint = this.judgeNextPoint(q), this.editor.rulerView.validateCanvas())
            }, a.prototype.judgeNextPoint = function (q) {
                var a = this.lp(q),
                    k = this.points;
                if (ht.Default.isShiftDown() && k.length > 0) {
                    var w = k[k.length - 1],
                        I = a.x - w.x,
                        K = a.y - w.y,
                        b = Math.atan2(K, I),
                        s = ht.Default.getDistance(w, a),
                        u = b % (Math.PI / 4);
                    b -= u, Math.abs(u) > Math.PI / 8 && (u > 0 ? b += Math.PI / 4 : b -= Math.PI / 4), a = {
                        x: w.x + Math.cos(b) * s,
                        y: w.y + Math.sin(b) * s
                    }
                }
                return a
            }, a.prototype.$0$ = function (q) {
                q.beginPath();
                for (var a = 0, k = void 0, w = void 0, I = void 0, K = 0; K < this.segments.length; K++) {
                    var b = this.segments[K];
                    1 === b ? (k = this.toPoint(this.points[a++]), q.moveTo(k.x, k.y)) : 2 === b ? (k = this.toPoint(this.points[a++]), q.lineTo(k.x, k.y)) : 3 === b ? (k = this.toPoint(this.points[a++]), w = this.toPoint(this.points[a++]), q.quadraticCurveTo(k.x, k.y, w.x, w.y)) : 4 === b ? (k = this.toPoint(this.points[a++]), w = this.toPoint(this.points[a++]), I = this.toPoint(this.points[a++]), q.bezierCurveTo(k.x, po, y, w.x, w.y, I.x, I.y)) : 5 === b && q.closePath()
                }
                if (this.nextPoint) {
                    var s = this.toPoint(this.nextPoint);
                    q.lineTo(s.x, s.y)
                }
                q.strokeStyle = hteditor.config.color_data_border, q.lineWidth = 1, q.stroke(), this.fill && (q.fillStyle = hteditor.config.color_transparent, q.fill()), this.drawPoints(q, this.points, 4, hteditor.config.color_data_border, hteditor.config.color_data_background)
            }, a
        }(J),
        O = function (q) {
            function a(k, w) {
                V(this, a);
                var I = Q(this, q.call(this, k));
                return I.createFunc = w, I
            }
            return E(a, q), a.prototype.tearDown = function () {
                q.prototype.tearDown.call(this), this.node = this.p1 = this.p2 = null
            }, a.prototype.handle_mousedown = function (q) {
                this.handle_touchstart(q)
            }, a.prototype.handle_touchstart = function (q) {
                ht.Default.preventDefault(q), ht.Default.isLeftButton(q) && (this.gv.setFocus(q), this.p1 = this.lp(q), this.startDragging(q))
            }, a.prototype.handleWindowMouseMove = function (q) {
                this.handleWindowTouchMove(q)
            }, a.prototype.handleWindowMouseUp = function (q) {
                this.handleWindowTouchEnd(q)
            }, a.prototype.handleWindowTouchMove = function (q) {
                if (ht.Default.preventDefault(q), this.p1) {
                    this.p2 = this.lp(q);
                    var a = ht.Default.unionPoint(this.p1, this.p2);
                    if (this.node) this.node.setRect(a);
                    else {
                        if (!a.width || !a.height) return;
                        this.beginTransaction(), this.createNode(a, !1)
                    }
                }
            }, a.prototype.handleWindowTouchEnd = function (q) {
                ht.Default.preventDefault(q), this.endTransaction(), !this.node && this.p1 && this.createNode({
                    x: this.p1.x - 25,
                    y: this.p1.y - 25,
                    width: 50,
                    height: 50
                }, !0), hteditor.config.continuousCreating ? this.node = this.p1 = this.p2 = null : this.editor.resetInteractionState()
            }, a.prototype.createNode = function (q, a) {
                this.node = this.createFunc(q, a), this.node.s3(50, 50, 50), a ? this.node.setPosition(q.x + q.width / 2, q.y + q.height / 2) : this.node.setRect(q), this.addData(this.node)
            }, a.prototype.$0$ = function (q) {
                this.node && this.drawRect(q, this.node.getRect(), hteditor.config.color_data_border)
            }, a
        }(J),
        $ = hteditor.getString,
        c = function (q) {
            function a(k) {
                V(this, a);
                var w = Q(this, q.call(this));
                return w.editor = k, w.initMenu(), w.$68$(), w.enableToolTip(), w
            }
            return E(a, q), a.prototype.initMenu = function () {
                var q = this.editor.mainMenu = new h(this.editor),
                    a = function (a) {
                        q.isShowing() && !q.getView().contains(a.target) && q.hide()
                    };
                window.addEventListener("touchstart", a, !1), window.addEventListener("mousedown", a, !1)
            }, a.prototype.$68$ = function () {
                var q = [];
                q.push(this.$67$()), q.push(this.createEditItem()), q.push(this.createCubeItem()), q.push(this.createCylinderItem()), q.push(this.createSphereItem()), q.push(this.createWallItem()), q.push(this.createDoorWindowItem()), q.push(this.createFloorItem()), q.push(this.createPipelineItem()), q.push(this.createLightItem()), this.setItems(q)
            }, a.prototype.$67$ = function () {
                var q = this,
                    a = hteditor.createItem("menu", $("editor.menu"), "editor.menu");
                return a.action = function (a, k, w) {
                    q.editor.mainMenu.isShowing() || (q.editor.mainMenu.showOnView(q, 4, 20), w.preventDefault(), w.stopPropagation())
                }, a
            }, a.prototype.createEditItem = function () {
                return this.editor.createSceneItem("edit", $("editor.edit"), "editor.edit")
            }, a.prototype.createWallItem = function () {
                var q = R.config.color_data_background,
                    a = R.config.color_transparent;
                return this.editor.createSceneItem("wall", $("editor.wall"), "editor.wall", new t(this.editor, function (k, w) {
                    var I = new ht.Shape;
                    return I.setTall(280), I.setThickness(14), I.setElevation(I.getTall() / 2), I.setPoints(k), I.setSegments(w), I.s({
                        "all.color": q,
                        "shape.background": null,
                        "shape.border.width": I.getThickness(),
                        "shape.border.color": a
                    }), I
                }))
            }, a.prototype.createDoorWindowItem = function () {
                return this.editor.createSceneItem("doorWindow", $("editor.doorWindow"), "editor.doorWindow", new O(this.editor, function () {
                    var q = new ht.DoorWindow;
                    return q.s({
                        "all.color": R.config.color_data_background
                    }), q
                }, !0))
            }, a.prototype.createFloorItem = function () {
                var q = R.config.color_data_background,
                    a = R.config.color_transparent;
                return this.editor.createSceneItem("floor", $("editor.floor"), "editor.floor", new t(this.editor, function (k, w) {
                    var I = new ht.Shape;
                    return I.setTall(10), I.setThickness(-1), I.setElevation(-I.getTall() / 2), I.setPoints(k), I.setSegments(w), I.s({
                        "shape3d.color": q,
                        "shape3d.top.color": q,
                        "shape3d.bottom.color": q,
                        "shape.background": a
                    }), I.setAnchor3d(.5, 0, .5), I
                }, !0))
            }, a.prototype.createPipelineItem = function () {
                var q = R.config.color_data_background,
                    a = R.config.color_transparent;
                return this.editor.createSceneItem("pipeline", $("editor.pipeline"), "editor.pipeline", new t(this.editor, function (k, w) {
                    var I = new ht.Polyline;
                    return I.setPoints(k), I.setSegments(w), I.setThickness(10), I.s({
                        shape3d: "cylinder",
                        "shape3d.color": q,
                        "shape3d.top.color": q,
                        "shape3d.bottom.color": q,
                        "shape.background": null,
                        "shape.border.width": I.getThickness(),
                        "shape.border.color": a
                    }), I
                }))
            }, a.prototype.createCubeItem = function () {
                return this.editor.createSceneItem("cube", $("editor.cube"), "editor.cube", new O(this.editor, function () {
                    var q = new ht.Node;
                    return q.s({
                        "all.color": R.config.color_data_background
                    }), q.setAnchor3d(.5, 0, .5), q
                }, !0))
            }, a.prototype.createCylinderItem = function () {
                return this.editor.createSceneItem("cylinder", $("editor.cylinder"), "editor.cylinder", new O(this.editor, function () {
                    var q = new ht.Node;
                    return q.s({
                        shape3d: "cylinder",
                        "shape3d.color": R.config.color_data_background
                    }), q.setAnchor3d(.5, 0, .5), q
                }, !0))
            }, a.prototype.createSphereItem = function () {
                return this.editor.createSceneItem("sphere", $("editor.sphere"), "editor.sphere", new O(this.editor, function () {
                    var q = new ht.Node;
                    return q.s({
                        shape3d: "sphere",
                        "shape3d.color": R.config.color_data_background
                    }), q.setAnchor3d(.5, 0, .5), q
                }, !0))
            }, a.prototype.createLightItem = function () {
                return this.editor.createSceneItem("light", $("editor.light"), "editor.light", new O(this.editor, function () {
                    var q = new ht.Light;
                    return q.s({
                        "light.color": "red",
                        "light.range": 200,
                        "light.rotation.enable": !0,
                        shape3d: "sphere",
                        "shape3d.color": "red",
                        "shape3d.reverse.cull": !0
                    }), q
                }, !0))
            }, a
        }(ht.widget.Toolbar),
        D = hteditor.getString,
        Y = hteditor.createItem,
        i = function (q) {
            function a(k) {
                V(this, a);
                var w = Q(this, q.call(this));
                return w.editor = k, w.$68$(), w.enableToolTip(), w.setStickToRight(!0), w.editor.gv.addPropertyChangeListener(w.handlePropertyChange, w), w.$69$(), w
            }
            return E(a, q), a.prototype.handlePropertyChange = function (q) {
                "zoom" === q.property && this.$69$()
            }, a.prototype.$69$ = function () {
                this.label.innerHTML = parseInt(100 * this.editor.gv.getZoom()) + "%"
            }, a.prototype.$68$ = function () {
                var q = this,
                    a = void 0,
                    k = [];
                a = Y("save", D("editor.save"), "editor.save"), a.action = function () {
                    q.editor.save()
                }, k.push(a), a = Y("reload", D("editor.reload"), "editor.reload"), a.action = function () {
                    q.editor.reload()
                }, k.push(a), k.push({
                    separator: !0
                }), a = Y("rulers", D("editor.showrulers"), "editor.rulers", function () {
                    return q.editor.isRulerEnabled()
                }), a.action = function () {
                    q.editor.toggleRulerEnabled()
                }, k.push(a), a = Y("grid", D("editor.showaxis"), "editor.axis", function () {
                    return !q.editor.dm.a("sceneEditHelperDisabled")
                }), a.action = function () {
                    q.editor.dm.a("sceneEditHelperDisabled", !q.editor.dm.a("sceneEditHelperDisabled"))
                }, k.push(a), a = Y("grid", D("editor.showgrid"), "editor.grid", function () {
                    return q.editor.dm.a("sceneGridEnabled")
                }), a.action = function () {
                    q.editor.dm.a("sceneGridEnabled", !q.editor.dm.a("sceneGridEnabled"))
                }, k.push(a), a = Y("preview", D("editor.preview"), "editor.preview"), a.action = function () {
                    q.editor.preview()
                }, k.push(a), k.push({
                    separator: !0
                }), k.push(this.createZoomItem()), a = Y("zoomToFit", D("editor.zoomtofit"), "editor.zoomtofit"), a.action = function () {
                    q.editor.zoomToFit()
                }, k.push(a), k.push({
                    separator: !0
                }), a = Y("toggleLeft", D("editor.toggleleft"), "editor.toggleleft"), a.action = function () {
                    q.editor.toggleLeft()
                }, k.push(a), a = Y("toggleRight", D("editor.toggleright"), "editor.toggleright"), a.action = function () {
                    q.editor.toggleRight()
                }, k.push(a), this.setItems(k)
            }, a.prototype.createZoomItem = function () {
                var q = this,
                    a = this.label = ht.Default.createDiv(!0);
                a.style.font = this.getLabelFont(), a.style.color = this.getLabelColor(), a.style.cursor = "ew-resize", a.style.width = "40px", a.style.height = ht.Default.widgetRowHeight + "px", a.style.lineHeight = ht.Default.widgetRowHeight + "px", a.style.whiteSpace = "nowrap", a.style.textAlign = "center";
                var k = function (a) {
                    a.preventDefault();
                    var k = q.editor.gv;
                    ht.Default.isDoubleClick(a) && k.zoomReset();
                    var w = k.getZoom(),
                        I = ht.Default.getClientPoint(a).x,
                        K = function (q) {
                            q.preventDefault();
                            var a = ht.Default.getClientPoint(q).x - I;
                            k.setZoom(w * (1 + .005 * a))
                        };
                    window.addEventListener("mousemove", K, !1), window.addEventListener("touchmove", K, !1);
                    var b = function (q) {
                        q.preventDefault(), window.removeEventListener("mousemove", K, !1), window.removeEventListener("touchmove", K, !1), window.removeEventListener("mouseup", b, !1), window.removeEventListener("touchend", b, !1)
                    };
                    window.addEventListener("mouseup", b, !1), window.addEventListener("touchend", b, !1)
                };
                return a.addEventListener("mousedown", k, !1), a.addEventListener("touchstart", k, !1), {
                    id: "zoom",
                    unfocusable: !0,
                    element: a
                }
            }, a
        }(ht.widget.Toolbar),
        P = function (q) {
            function a(k) {
                V(this, a);
                var w = Q(this, q.call(this, k.dm));
                return w.editor = k, w.setEditable(!0), w.topDiv = ht.Default.createDiv(), w.getPostProcessing().setSerializable(!1), w
            }
            return E(a, q), a.prototype.isInteractive = function () {
                return !1
            }, a.prototype.handleDelete = function () {
                this.removeSelection()
            }, a.prototype.getBrightness = function (q) {
                return q.s("brightness")
            }, a.prototype.isDroppable = function (q, a) {
                return a.view === this.editor.models.list || (a.view === this.editor.symbols.list || a.view === this.editor.assets.list)
            }, a.prototype.$58$ = function (q) {
                if (this.dragImage) {
                    var a = hteditor.config.dragImageSize,
                        k = ht.Default.getPagePoint(q);
                    this.dragImage.style.left = k.x - a / 2 + "px", this.dragImage.style.top = k.y - a / 2 + "px"
                }
            }, a.prototype.removeDragImage = function () {
                this.dragImage && (ht.Default.removeHTML(this.dragImage), this.dragImage = null)
            }, a.prototype.handleCrossDrag = function (q, a, k) {
                var w = this,
                    I = k.view,
                    K = I.draggingData;
                if ("enter" === a) {
                    if (this._view.insertBefore(this.topDiv, this._scrollBarDiv), ht.Default.layout(this.topDiv, 0, 0, this.getWidth(), this.getHeight()), this.topDiv.style.border = "solid " + hteditor.config.color_select_dark + " 2px", !this.dragImage && K && ("model" === K.fileType || "symbol" === K.fileType || "image" === K.fileType) && ht.Default.getImage(K.getImage()) !== ht.Default.getImage("editor.unknown")) {
                        var b = hteditor.config.dragImageSize;
                        this.dragImage = ht.Default.toCanvas(K.getImage(), b, b, "centerUniform", K, k.view, null, ht.Default.devicePixelRatio), this.$58$(q), this.dragImage.style.opacity = hteditor.config.dragImageOpacity, ht.Default.appendToScreen(this.dragImage)
                    }
                } else if ("exit" === a || "cancel" === a) ht.Default.removeHTML(this.topDiv), this.removeDragImage();
                else if ("over" === a) this.$58$(q);
                else if ("drop" === a && (ht.Default.removeHTML(this.topDiv), K)) {
                    this.removeDragImage();
                    var s = this.getHitPosition(q);
                    I.isSelected(K) ? I.sm().toSelection().each(function (a) {
                        I.handleDropToEditView ? I.handleDropToEditView(w, a, s, q) : w.$98$(a, s)
                    }) : I.handleDropToEditView ? I.handleDropToEditView(this, K, s, q) : this.$98$(K, s)
                }
            }, a.prototype.$98$ = function (q, a) {
                if (q) {
                    var k = q.getFileUUID(),
                        w = q.fileType;
                    if ("model" === w || "symbol" === w || s(k)) {
                        var I = new ht.Node;
                        "model" === w ? I.s({
                            shape3d: k
                        }) : ("symbol" === w || s(k)) && (I.s({
                            shape3d: "billboard",
                            "shape3d.image": k,
                            "texture.cache": !1,
                            autorotate: !0,
                            fixSizeOnScreen: [-1, -1]
                        }), I.setImage(k)), I.setAnchor3d(.5, 0, .5), I.p3(a), I.s("wf.visible", "selected"), this.dm().beginTransaction(), this.dm().add(I), this.sm().ss(I), this.editor.fireEvent("dataCreated", {
                            sceneView: this.editor.scene,
                            data: I
                        }), this.dm().endTransaction()
                    }
                }
            }, a.prototype.toImage = function (q) {
                var a = R.config.imageSize || 400,
                    k = this.getWidth(),
                    w = this.getHeight(),
                    I = Math.max(k, w),
                    K = a / I;
                return k = Math.floor(k * K), w = Math.floor(w * K), this.toCanvas(q, k, w).toDataURL("image/png", 1)
            }, a.prototype.invalidateAll = function (a, k) {
                if ("imageLoaded" === k || "modelLoaded" === k) {
                    var w = this.editor.inspector;
                    w && ("modelLoaded" === k ? w.filterPropertiesLater() : w.$34$(), w.$85$ && w.$85$())
                }
                q.prototype.invalidateAll.call(this, a, k)
            }, a
        }(ht.graph3d.Graph3dView),
        N = {
            width: !0,
            height: !0,
            position: !0,
            rotation: !0,
            anchor: !0,
            scale: !0,
            expanded: !0
        },
        U = function () {
            function q(a, k) {
                var w = this;
                V(this, q), this.editor = a, this.gv = this.graphView = k, this._rulerEnabled = !0, this._view = ht.Default.createView(!0, this), this.graphView.addViewListener(function (q) {
                    "validate" === q.kind && w.validateCanvas()
                }), this._hRuler = ht.Default.createCanvas(), this._vRuler = ht.Default.createCanvas(), this._view.appendChild(this._hRuler), this._view.appendChild(this._vRuler), this._view.appendChild(this.graphView.getView()), this.graphView.addPropertyChangeListener(function () {
                    w.iv()
                }), this.dm.sm().addSelectionChangeListener(function (q) {
                    w.iv()
                }), this.dm.addDataModelChangeListener(function (q) {
                    w.iv()
                }), this.dm.addDataPropertyChangeListener(function (q) {
                    w.handleDataPropertyChanged(q)
                }), this.dm.addPropertyChangeListener(function (q) {
                    w.handleDataModelPropertyChanged(q)
                }), this.dm.addHierarchyChangeListener(function (q) {
                    w.iv()
                }), this.dm.layout = function (q) {
                    w._autoLayout || (w._autoLayout = new ht.layout.AutoLayout(w.graphView)), w._autoLayout.layout(q, function () {
                        w.graphView.fitContent(!0)
                    })
                }, this.graphView.setEditable(!0), this.graphView.getInteractors().each(function (q) {
                    q.keep = !0
                }), k.getEditInteractor().alignmentGuideEnabled = !0, hteditor.EditView.prototype.initRuler.call(this)
            }
            return q.prototype.handleDataPropertyChanged = function (q) {
                N[q.property] && this.iv()
            }, q.prototype.handleDataModelPropertyChanged = function () {
                hteditor.EditView.prototype.redraw.apply(this, arguments)
            }, q.prototype.onPropertyChanged = function (q) {
                this.iv()
            }, q.prototype.addData = function (q) {
                this.dm.add(q), this.dm.sm().ss(q), this.graphView.setFocus()
            }, q.prototype.validateCanvas = function () {
                var q = this.graphView._topCanvas;
                if (q) {
                    var a = hteditor.initContext(q);
                    a.clearRect(0, 0, q.clientWidth, q.clientHeight), this.graphView.getInteractors().each(function (q) {
                        q.$0$ && q.$0$(a)
                    }), a.restore()
                }
            }, q.prototype.redraw = function () {
                hteditor.EditView.prototype.redraw.apply(this, arguments)
            }, q.prototype.validateImpl = function () {
                hteditor.EditView.prototype.validateImpl.apply(this, arguments)
            }, q.prototype.drawVerticalText = function () {
                hteditor.EditView.prototype.drawVerticalText.apply(this, arguments)
            }, q.prototype.drawRuler = function () {
                hteditor.EditView.prototype.drawRuler.apply(this, arguments)
            }, q.prototype.updateAlignGuide = function () {
                hteditor.EditView.prototype.updateAlignGuide.apply(this, arguments)
            }, q.prototype.getGridGuide = function () {
                return hteditor.EditView.prototype.getGridGuide.apply(this, arguments)
            }, q.prototype.setRulerEnabled = function (q) {
                this.dm.a("rulerEnabled", q)
            }, q.prototype.isRulerEnabled = function () {
                return this.dm.a("rulerEnabled")
            }, q.prototype._initMenu = function () {}, v(q, [{
                key: "dm",
                get: function () {
                    return this.graphView.dm()
                }
            }]), q
        }();
    hteditor.msClass(U, {
        ms_v: 1,
        ms_fire: 1,
        ms_ac: ["rulerBackground", "rulerFont", "rulerColor", "rulerSize", "guideColor"],
        _rulerFont: hteditor.config.smallFont,
        _rulerBackground: hteditor.config.color_pane,
        _rulerColor: hteditor.config.color_dark,
        _rulerSize: hteditor.config.rulerSize,
        _guideColor: hteditor.config.color_transparent,
        _interactionState: void 0
    });
    var qq = R.getString,
        aq = function (q) {
            function a(k) {
                V(this, a);
                var w = Q(this, q.call(this, k.dm));
                return w.editor = k, w._topCanvas = ht.Default.createCanvas(), w._topDiv = ht.Default.createDiv(), w._view.insertBefore(w._topCanvas, w._scrollBarDiv), w._view.insertBefore(w._topDiv, w._scrollBarDiv), w.initMenu(), w.mi(function (q) {
                    "prepareMove" !== q.kind && "prepareEdit" !== q.kind || !q.event.altKey || (w.editor.copy(), w.editor.paste(), w.validate())
                }), w.editable = !0, w
            }
            return E(a, q), a.prototype.isInteractive = function () {
                return !1
            }, a.prototype.initMenu = function () {
                var q = this,
                    a = [];
                this.menu = new ht.widget.ContextMenu(a), a.push({
                    id: "copy",
                    label: qq("editor.copy"),
                    action: function () {
                        q.editor.copy()
                    },
                    visible: function () {
                        return !!q.editor.ld
                    }
                }), a.push({
                    id: "paste",
                    label: qq("editor.paste"),
                    action: function () {
                        q.editor.paste()
                    },
                    visible: function () {
                        return q.editable && q.editor.hasCopyInfo()
                    }
                });
                var k = function () {
                        return q.sm().size() > 0
                    },
                    w = function () {
                        for (var a = 0; a < q.sm().size(); a++) {
                            var k = q.sm().getSelection().get(a);
                            if (k instanceof ht.Block && !(k instanceof ht.RefGraph)) return !0
                        }
                        return !1
                    };
                a.push({
                    separator: !0,
                    visible: function () {
                        return q.editable && (k() || w())
                    }
                }), a.push({
                    id: "block",
                    label: qq("editor.block"),
                    action: function () {
                        q.editor.block()
                    },
                    visible: function () {
                        return q.editable && k()
                    }
                }), a.push({
                    id: "unblock",
                    label: qq("editor.unblock"),
                    action: function () {
                        q.editor.unblock()
                    },
                    visible: function () {
                        return q.editable && w()
                    }
                }), this.menu.addTo(this.getView()), this.editor.menus.push(this.menu)
            }, a.prototype.getUnionNodeRect = function (q) {
                var a = this,
                    k = void 0;
                return q = q._as || q, q.forEach(function (q) {
                    q instanceof ht.Node && (k = ht.Default.unionRect(k, a.getNodeRect(q)))
                }), k
            }, a.prototype.getImage = function (q) {
                var a = q.s("shape3d");
                if (a) {
                    var k = ht.Default.getShape3dModel(a);
                    if (k && k.json && k.json.image) return k.json.image;
                    if ("sphere" === a || "cylinder" === a) return "editor.sphere.image"
                }
                var w = q.getImage();
                return w && "node_image" !== w ? w : q.s("shape3d.image") ? q.s("shape3d.image") : "editor.cube.image"
            }, a.prototype.getSelectWidth = function (q) {
                return !this.isEditable(q) || q instanceof ht.Edge ? q.s("select.width") : 0
            }, a.prototype.isDroppable = function (q, a) {
                return a.view === this.editor.models.list || (a.view === this.editor.symbols.list || a.view === this.editor.assets.list)
            }, a.prototype.$58$ = function (q) {
                if (this.dragImage) {
                    var a = hteditor.config.dragImageSize,
                        k = ht.Default.getPagePoint(q);
                    this.dragImage.style.left = k.x - a / 2 + "px", this.dragImage.style.top = k.y - a / 2 + "px"
                }
            }, a.prototype.removeDragImage = function () {
                this.dragImage && (ht.Default.removeHTML(this.dragImage), this.dragImage = null)
            }, a.prototype.handleCrossDrag = function (q, a, k) {
                var w = this,
                    I = k.view,
                    K = I.draggingData;
                if ("enter" === a) {
                    if (this._topDiv.style.border = "solid " + hteditor.config.color_select_dark + " 2px", !this.dragImage && K && ("model" === K.fileType || "symbol" === K.fileType || "image" === K.fileType) && ht.Default.getImage(K.getImage()) !== ht.Default.getImage("editor.unknown")) {
                        var b = hteditor.config.dragImageSize;
                        this.dragImage = ht.Default.toCanvas(K.getImage(), b, b, "centerUniform", K, k.view, null, ht.Default.devicePixelRatio), this.$58$(q), this.dragImage.style.opacity = hteditor.config.dragImageOpacity, ht.Default.appendToScreen(this.dragImage)
                    }
                } else "exit" === a || "cancel" === a ? (this._topDiv.style.border = "", this.removeDragImage()) : "over" === a ? this.$58$(q) : "drop" === a && (this._topDiv.style.border = "", K && (this.removeDragImage(), this.sm().cs(), I.handleDropToEditView ? I.handleDropToEditView(this, K, this.lp(q), q) : I.isSelected(K) ? I.sm().toSelection().each(function (a) {
                    w.$98$(a, q)
                }) : this.$98$(K, q)))
            }, a.prototype.$98$ = function (q, a) {
                var k = q.getFileUUID(),
                    w = this.lp(a);
                if ("model" === q.fileType) {
                    var I = new ht.Node;
                    I.s({
                        shape3d: k
                    }), I.setAnchor3d(.5, 0, .5), I.setPosition(w), this.dm().beginTransaction(), this.dm().add(I), this.sm().ss(I), this.editor.fireEvent("dataCreated", {
                        sceneView: this.editor.scene,
                        data: I
                    }), this.dm().endTransaction()
                } else if ("symbol" === q.fileType || "image" === q.fileType) {
                    var K = new ht.Node;
                    K.s({
                        shape3d: "billboard",
                        "shape3d.image": k,
                        "texture.cache": !1,
                        autorotate: !0,
                        fixSizeOnScreen: [-1, -1]
                    }), K.setAnchor3d(.5, 0, .5), K.p(w), K.setImage(k), this.dm().beginTransaction(), this.dm().add(K), this.sm().ss(K), this.editor.fireEvent("dataCreated", {
                        sceneView: this.editor.scene,
                        data: K
                    }), this.dm().endTransaction()
                }
            }, a
        }(ht.graph.GraphView),
        kq = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k))
            }
            return E(a, q), a.prototype.commit = function () {
                this.editor.reset(!0), this.editor.dm.disableHistoryManager(), this.editor.scene.deserialize(this.content, {
                    disableOnPostDeserialize: !0,
                    disableOnPreDeserialize: !0
                }), this.editing = !1, this.editor.dm.enableHistoryManager(), this.editor.dm.clearHistoryManager()
            }, a.prototype.updateContent = function () {
                this.visible && !this.editing && (this._updateContentLater = !1, this.content = o(this.editor.toJSON(), void 0, !1))
            }, a.prototype.updateUrl = function () {
                this.v("url", this.editor.url || "")
            }, a.prototype.$5$ = function () {}, a
        }(hteditor.DataView),
        wq = hteditor.getString,
        Iq = function (q) {
            function a(k) {
                V(this, a);
                var w = Q(this, q.call(this));
                w.editor = k, w.buttons = {};
                var I = [];
                I.push(w.createItem("cluster", wq("editor.align.cluster"), [{
                    type: "shape",
                    points: [5, 2, 11, 2, 5, 14, 11, 14],
                    segments: [1, 2, 1, 2],
                    borderWidth: 3,
                    borderColor: hteditor.config.color_select
                }, {
                    type: "shape",
                    points: [8, 5, 8, 11, 5, 8, 11, 8],
                    segments: [1, 2, 1, 2],
                    borderWidth: 1,
                    borderColor: hteditor.config.color_dark
                }])), I.push(w.createItem("distributeHorizontal", wq("editor.align.distributehorizontal"), [{
                    type: "shape",
                    points: [2, 2, 2, 14, 14, 2, 14, 14],
                    segments: [1, 2, 1, 2],
                    borderWidth: 1
                }, {
                    type: "shape",
                    points: [8, 5, 8, 11],
                    borderWidth: 3
                }])), I.push(w.createItem("distributeVertical", wq("editor.align.distributevertical"), [{
                    type: "shape",
                    points: [2, 2, 14, 2, 2, 14, 14, 14],
                    segments: [1, 2, 1, 2],
                    borderWidth: 1
                }, {
                    type: "shape",
                    points: [5, 8, 11, 8],
                    borderWidth: 3
                }])), I.push(w.createItem("alignLeft", wq("editor.align.alignleft"), [{
                    type: "shape",
                    points: [2, 2, 2, 14],
                    segments: [1, 2],
                    borderWidth: 1
                }, {
                    type: "shape",
                    points: [5, 5, 13, 5, 5, 11, 8, 11],
                    segments: [1, 2, 1, 2],
                    borderWidth: 3
                }])), I.push(w.createItem("alignHorizontal", wq("editor.align.alignhorizontal"), [{
                    type: "shape",
                    points: [8, 2, 8, 14],
                    segments: [1, 2],
                    borderWidth: 1
                }, {
                    type: "shape",
                    points: [3, 5, 13, 5, 6, 11, 10, 11],
                    segments: [1, 2, 1, 2],
                    borderWidth: 3
                }])), I.push(w.createItem("alignRight", wq("editor.align.alignright"), [{
                    type: "shape",
                    points: [14, 2, 14, 14],
                    segments: [1, 2],
                    borderWidth: 1
                }, {
                    type: "shape",
                    points: [3, 5, 11, 5, 11, 11, 8, 11],
                    segments: [1, 2, 1, 2],
                    borderWidth: 3
                }])), I.push(w.createItem("alignTop", wq("editor.align.aligntop"), [{
                    type: "shape",
                    points: [2, 2, 14, 2],
                    segments: [1, 2],
                    borderWidth: 1
                }, {
                    type: "shape",
                    points: [5, 5, 5, 13, 11, 5, 11, 8],
                    segments: [1, 2, 1, 2],
                    borderWidth: 3
                }])), I.push(w.createItem("alignVertical", wq("editor.align.alignvertical"), [{
                    type: "shape",
                    points: [2, 8, 14, 8],
                    segments: [1, 2],
                    borderWidth: 1
                }, {
                    type: "shape",
                    points: [5, 3, 5, 13, 11, 6, 11, 10],
                    segments: [1, 2, 1, 2],
                    borderWidth: 3
                }])), I.push(w.createItem("alignBottom", wq("editor.align.alignbottom"), [{
                    type: "shape",
                    points: [2, 14, 14, 14],
                    segments: [1, 2],
                    borderWidth: 1
                }, {
                    type: "shape",
                    points: [5, 3, 5, 11, 11, 11, 11, 8],
                    segments: [1, 2, 1, 2],
                    borderWidth: 3
                }]));
                var K = .1,
                    b = [K, K, K, K, K, K, K, K, K];
                return w.addRow(I, b), w.updateItems(), w.editor.sm.ms(function () {
                    return w.updateItems()
                }), w
            }
            return E(a, q), a.prototype.createItem = function (q, a, k) {
                var w = this;
                k.forEach(function (q) {
                    q.borderCap = "round"
                }), k[0].borderColor = k[0].borderColor || hteditor.config.color_dark, k[1].borderColor = k[1].borderColor || hteditor.config.color_select;
                var I = {
                    width: 16,
                    height: 16,
                    fitSize: !0,
                    comps: k
                };
                return this.buttons[q] = hteditor.createButton(null, a, I, function () {
                    w[q]()
                })
            }, a.prototype.updateItems = function () {
                var q = [];
                this.editor.sm.each(function (a) {
                    a instanceof ht.Node && q.push(a)
                }), q.length > 2 ? (this.buttons.cluster.setDisabled(!1), this.buttons.distributeHorizontal.setDisabled(!1), this.buttons.distributeVertical.setDisabled(!1), this.buttons.alignLeft.setDisabled(!1), this.buttons.alignHorizontal.setDisabled(!1), this.buttons.alignRight.setDisabled(!1), this.buttons.alignTop.setDisabled(!1), this.buttons.alignVertical.setDisabled(!1), this.buttons.alignBottom.setDisabled(!1)) : q.length > 1 ? (this.buttons.distributeHorizontal.setDisabled(!0), this.buttons.distributeVertical.setDisabled(!0), this.buttons.alignLeft.setDisabled(!1), this.buttons.alignHorizontal.setDisabled(!1), this.buttons.alignRight.setDisabled(!1), this.buttons.alignTop.setDisabled(!1), this.buttons.alignVertical.setDisabled(!1), this.buttons.alignBottom.setDisabled(!1)) : (this.buttons.cluster.setDisabled(!0), this.buttons.distributeHorizontal.setDisabled(!0), this.buttons.distributeVertical.setDisabled(!0), this.buttons.alignLeft.setDisabled(!0), this.buttons.alignHorizontal.setDisabled(!0), this.buttons.alignRight.setDisabled(!0), this.buttons.alignTop.setDisabled(!0), this.buttons.alignVertical.setDisabled(!0), this.buttons.alignBottom.setDisabled(!0))
            }, a.prototype.getUnionRect = function (q) {
                return this.editor.gv ? this.editor.gv.getUnionNodeRect(q) : null
            }, a.prototype.isSelected = function (q) {
                return this.editor.sm.contains(q)
            }, a.prototype.hasSelectedGroupParent = function (q) {
                return hteditor.InspectorTool.prototype.hasSelectedGroupParent.apply(this, arguments)
            }, a.prototype.hasSelectedHost = function (q) {
                return hteditor.InspectorTool.prototype.hasSelectedHost.apply(this, arguments)
            }, a.prototype.cluster = function () {
                var q = this.selection;
                if (q.length) {
                    this.editor.beginTransaction(), this.alignHorizontal(), this.alignVertical();
                    var a = void 0,
                        k = void 0;
                    q.forEach(function (q) {
                        void 0 === a ? a = k = q.getElevation() : (a > q.getElevation() && (a = q.getElevation()), k < q.getElevation() && (k = q.getElevation()))
                    }), q.sort(function (q, a) {
                        return q.getElevation() - a.getElevation()
                    });
                    for (var w = (k - a) / (q.length - 1), I = 1; I < q.length - 1; I++) q[I].setElevation(a + I * w);
                    this.editor.endTransaction()
                }
            }, a.prototype.distributeHorizontal = function () {
                hteditor.InspectorTool.prototype.distributeHorizontal.apply(this, arguments)
            }, a.prototype.distributeVertical = function () {
                hteditor.InspectorTool.prototype.distributeVertical.apply(this, arguments)
            }, a.prototype.alignLeft = function () {
                hteditor.InspectorTool.prototype.alignLeft.apply(this, arguments)
            }, a.prototype.alignHorizontal = function () {
                hteditor.InspectorTool.prototype.alignHorizontal.apply(this, arguments)
            }, a.prototype.alignRight = function () {
                hteditor.InspectorTool.prototype.alignRight.apply(this, arguments)
            }, a.prototype.alignTop = function () {
                hteditor.InspectorTool.prototype.alignTop.apply(this, arguments)
            }, a.prototype.alignVertical = function () {
                hteditor.InspectorTool.prototype.alignVertical.apply(this, arguments)
            }, a.prototype.alignBottom = function () {
                hteditor.InspectorTool.prototype.alignBottom.apply(this, arguments)
            }, v(a, [{
                key: "selection",
                get: function () {
                    var q = this,
                        a = [];
                    return this.editor.sm.each(function (k) {
                        k instanceof ht.Node && (q.hasSelectedGroupParent(k.getParent()) || q.hasSelectedHost(k) || a.push(k))
                    }), a
                }
            }]), a
        }(ht.widget.FormPane),
        Kq = function (q) {
            function a(k, w, I) {
                var K = arguments.length > 3 && void 0 !== arguments[3] && arguments[3];
                V(this, a);
                var b = Q(this, q.call(this, k, w, I, K));
                return b.setHPadding(8), b.setVPadding(4), b.addUpdateHandler(function () {
                    b.$85$()
                }), b._cacheImgObjects = [], b
            }
            return E(a, q), a.prototype.initForm = function () {
                q.prototype.initForm.call(this), this.addCustomProperties()
            }, a.prototype.addOneRow = function (q, a, k) {
                for (var w = [this.indent, .1], I = 2, K = q.length; I < K; I++) w.push(20);
                var b = this.addRow(q, w, a ? this.getRowHeight() * a : null, k);
                return b.index = this.getRows().length - 1, b
            }, a.prototype.$85$ = function () {
                var q = this;
                this._updateDataBindingsLater || (this._updateDataBindingsLater = !0, requestAnimationFrame(function () {
                    q.initDataBindings()
                }))
            }, a.prototype.initDataBindings = function () {
                var q = this;
                this._updateDataBindingsLater = !1;
                var a = this.data;
                if (a) {
                    for (var k = [a.s("all.image"), a.s("top.image"), a.s("bottom.image"), a.s("left.image"), a.s("right.image"), a.s("front.image"), a.s("back.image"), a.s("shape3d.image"), a.s("shape3d.top.image"), a.s("shape3d.bottom.image"), a.s("shape3d.from.image"), a.s("shape3d.to.image")], w = !1, I = [], K = 0, b = k.length; K < b; K++) {
                        var s = k[K];
                        if (s) {
                            if (!(s = ht.Default.getImage(s))) {
                                w = !0;
                                break
                            }
                            I.push(s)
                        }
                    }
                    if (w) {
                        var C = ht.Default.handleImageLoaded;
                        return void(ht.Default.handleImageLoaded = function () {
                            for (var a = arguments.length, k = Array(a), w = 0; w < a; w++) k[w] = arguments[w];
                            C.apply(ht.Default, k), ht.Default.handleImageLoaded = C, q.initDataBindings()
                        })
                    }
                    if (0 === I.length) return void(this.$44$ && this.$44$.length > 0 && (this.clearDataBinding(), this.filterProperties()));
                    var L = this._cacheImgObjects,
                        j = L.length !== I.length;
                    if (!j)
                        for (var G = 0, X = I.length; G < X; G++)
                            if (L.indexOf(I[G]) < 0) {
                                j = !0;
                                break
                            } if (j) {
                        this.clearDataBinding(), this._cacheImgObjects = I;
                        for (var Z = [], r = {}, W = {}, T = {}, o = 0, g = I.length; o < g; o++) {
                            var H = I[o];
                            if (H.comps) {
                                H.comps.forEach(function (q) {
                                    if (u(q.type)) {
                                        var a = ht.Default.getCompType(q.type);
                                        if (a)
                                            for (var k in a.properties) r[k] = a.properties[k].defaultValue
                                    }
                                    p(q, W, T)
                                });
                                for (var f in W) void 0 === T[f] && (T[f] = r[W[f]])
                            }
                            H.dataBindings && Z.push.apply(Z, H.dataBindings)
                        }
                        if (Z.length > 0) {
                            var n = this.updateHandlers.length;
                            this.$44$.push(this.addTitle("TitleDataBinding"));
                            var y = {};
                            Z.forEach(function (a) {
                                var k = a.attr;
                                y[k] || (y[k] = 1, R.HTNodeInspector.prototype.$87$.call(q, a, void 0, T))
                            }), this.dataBindingHandlers = this.updateHandlers.slice(n)
                        }
                        this.$46$ = this._rows, this.filterProperties()
                    }
                }
            }, a.prototype.clearDataBinding = function () {
                var q = this;
                this._rows = this.$46$;
                var a = this.$44$;
                a && a.length > 0 && this.removeRows(a), this.$44$ = [];
                var k = this.dataBindingHandlers;
                k && k.length > 0 && k.forEach(function (a) {
                    q.removeUpdateHandler(a)
                }), this.dataBindingHandlers = [], this._cacheImgObjects = [], this.$46$ = this._rows
            }, a.prototype.$87$ = function (q, a) {
                var k = [],
                    w = q.attr,
                    I = hteditor.config.valueTypes[q.valueType];
                if (!I) return this.editor.fireEvent("error", {
                    message: "Wrong value type:" + q.valueType,
                    dataBind: q
                }), !1;
                var K = hteditor.getString(q.name) || w,
                    b = function (q) {
                        var k = q.a(w);
                        return void 0 === k && (k = a[w]), k
                    },
                    s = function (q, a) {
                        return q.a(w, a)
                    };
                return "int" === I.type || "number" === I.type ? I.angle ? this.addLabelRotation(k, K, b, s, I.min, I.max, I.step) : this.addLabelRange(k, K, b, s, I.min, I.max, I.step, I.type) : "color" === I.type ? this.addLabelColor(k, K, b, s) : "boolean" === I.type ? this.addLabelCheckBox(k, K, b, s) : "enum" === I.type ? this.addLabelComboBox(k, K, b, s, I.values, I.labels, I.icons) : "multiline" === I.type ? this.addLabelMultiline(k, K, b, s) : "font" === I.type ? this.addLabelFont(k, K, b, s) : "image" === I.type ? this.addLabelImage(k, K, b, s) : "url" === I.type ? this.addLabelURL(k, K, b, s) : "colorArray" === I.type ? this.addLabelArray(k, K, b, s, "color") : "numberArray" === I.type ? this.addLabelArray(k, K, b, s, "number") : "stringArray" === I.type ? this.addLabelArray(k, K, b, s, "string") : "objectArray" === I.type ? this.addLabelArray(k, K, b, s, "object") : "function" === I.type ? this.addLabelFunction(k, K, b, s, w) : "object" === I.type ? this.addLabelObject(k, K, b, s, w) : this.addLabelInput(k, K, b, s), k
            }, a.prototype.addLabelData = function (q, a, k, w) {
                var I = this,
                    K = this.addLabelInput(q, a, function (q) {
                        var a = k(q);
                        return a ? a.toLabel() || a.getClassName() : ""
                    }),
                    b = K.getElement();
                K.isDroppable = function (q, a) {
                    return a.view.draggingData && I.dataModel === a.view.draggingData.getDataModel()
                }, K.handleCrossDrag = function (q, a, k) {
                    "enter" === a ? b.style.border = "solid " + R.config.color_select_dark + " 2px" : "exit" === a || "cancel" === a ? b.style.border = "" : "over" === a || "drop" === a && (b.style.border = "", I.setValue(w, k.view.draggingData), K.setFocus())
                }, b.onkeydown = function (q) {
                    ht.Default.isDelete(q) && I.setValue(w, null)
                };
                var s = new ht.widget.Image;
                s.drawImage = function (q, a, k, w, K, b) {
                    var u = s.data,
                        C = I.editor.list;
                    u && C && C.dm().contains(u) && (a = C.getIcon(u)) && drawStretchImage(q, ht.Default.getImage(a), "centerUniform", k, w, K, b, u, C)
                }, this.updateHandlers.push(function () {
                    if (!q.hidden) {
                        var a = s.data;
                        s.data = I.getValue(k), s.fp("data", a, s.data)
                    }
                }), q.push(s)
            }, a.prototype.addImage = function (q, a, k, w, K) {
                var b = this,
                    s = new ht.widget.Image;
                s.vectorDataBindingDisabled = !0;
                var u = w || function (q) {
                    if (G(q)) return "editor.hdr";
                    if (I(q)) return q.replace(".json", ".png");
                    var a = ht.Default.getImage(q);
                    return a && a.snapshotURL ? a.snapshotURL : q
                };
                this.updateHandlers.push(function () {
                    if (!q.hidden) {
                        var k = b.getValue(a);
                        s.rawIcon = k, s.setIcon(u(k))
                    }
                });
                var C = s.getView();
                C.style.cursor = "pointer";
                var L = function (q) {
                    q.preventDefault(), K && !K(s) || (ht.Default.isDoubleClick(q) ? b.editor.open(s.rawIcon) : b.editor.selectFileNode(s.rawIcon))
                };
                return C.addEventListener("mousedown", L, !1), C.addEventListener("touchstart", L, !1), q.push(s), s
            }, a.prototype.addLabelSlider = function (q, a, k, w, I, K, b, s) {
                var u = arguments.length > 8 && void 0 !== arguments[8] ? arguments[8] : {};
                return this.addLabel(q, a, k, w, void 0, u), this.addSlider(q, k, w, I, K, b, s, u.onBeginEdit, u.onEndEditing)
            }, a.prototype.addSlider = function (q, a, k, w, I, K, b, s, u) {
                var C = this,
                    L = new ht.widget.Slider;
                return ht.Default.isNumber(w) && L.setMin(w), ht.Default.isNumber(I) && L.setMax(I), ht.Default.isNumber(K) && L.setStep(K), k ? (L.onBeginValueChanged = function () {
                    C.beginTransaction(), L.__editing__ = !0, s && s(L)
                }, L.onValueChanged = function () {
                    C.setValue(k, L.getValue())
                }, L.onEndValueChanged = function () {
                    u && u(L), L.__editing__ = !1, C.endTransaction()
                }) : L.setDisabled(!0), L.getToolTip = function () {
                    return Number(L.getValue().toFixed(6))
                }, b && (L.getToolTip = function () {
                    return b(L.getValue())
                }), this.updateHandlers.push(function () {
                    if (!q.hidden && !L.__editing__) {
                        var k = C.getValue(a);
                        L.getValue() !== k && L.setValue(k)
                    }
                }), q.push(L), L
            }, a.prototype.addLabelImageOrMaterial = function (q, a, k, w) {
                var I = function (q, a) {
                    if (!a.view.draggingData) return !1;
                    var k = a.view.draggingData.fileType;
                    return s(a.view.draggingData.url) || "symbol" === k || "material" === k
                };
                return this.addLabelURL(q, a, k, w, I), this.addImage(q, k, w)
            }, a.prototype.addLabelTooltip = function (a, k, w, I) {
                var K = q.prototype.addLabel.apply(this, arguments),
                    b = void 0;
                return K.addEventListener("mousemove", function (q) {
                    b && (clearTimeout(b), b = void 0), ht.Default.isToolTipShowing() ? ht.Default.showToolTip(q, k) : b = setTimeout(function () {
                        ht.Default.showToolTip(q, k), b = void 0
                    }, ht.Default.toolTipDelay)
                }), K.addEventListener("mouseleave", function () {
                    b && clearTimeout(b), b = void 0, ht.Default.hideToolTip()
                }), K
            }, v(a, [{
                key: "type",
                set: function (q) {
                    this._type = q
                },
                get: function () {
                    return this._type
                }
            }, {
                key: "dataModel",
                get: function () {
                    return this.editor.dm
                }
            }]), a
        }(hteditor.Inspector),
        bq = hteditor.getString,
        sq = hteditor.createButton,
        uq = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "scene", "scene", !0))
            }
            return E(a, q), a.prototype.initForm = function () {
                    q.prototype.initForm.call(this), this.initSceneProperties(), this.addSkyboxProperties(), this.addBloomProperties(), this.addDofProperties(), this.addHighlightProperties(), this.addHeadlightProperties(), this.addFogProperties(), this.$77$(), this.addHueSaturation(), this.addEnvmapProperties(), this.editor.scene.shadowMap && this.initShadowProperties()
                },
                a.prototype.initSceneProperties = function () {
                    var q = this,
                        a = void 0;
                    this.addTitle("TitleScenes"), a = [], this.addLabelCheckBox(a, bq("editor.debugTip"), function () {
                        return q.editor.scene.isDebugTipShowing()
                    }, function (a, k) {
                        var w = q.editor.scene;
                        k ? w.showDebugTip() : w.hideDebugTip()
                    }), this.addRow(a, [this.indent, .1]), a = [], this.addLabelInput(a, bq("editor.previewurl"), hteditor.getter("a", "previewURL"), hteditor.setter("a", "previewURL")), this.addRow(a, [this.indent, .1]), a = [], a.push(bq("editor.camera")), this.addButton(a, null, bq("editor.reset"), "editor.resetsize.state", function () {
                        q.editor.scene.setEye(ht.Default.graph3dViewEye)
                    }), this.addLabelInput(a, "X", function (a) {
                        return parseFloat(q.editor.scene.getEye()[0])
                    }, function (a, k) {
                        var w = q.editor.scene.getEye();
                        q.editor.scene.setEye(k, w[1], w[2])
                    }, "int", 1), this.addLabelInput(a, "Y", function (a) {
                        return parseFloat(q.editor.scene.getEye()[1])
                    }, function (a, k) {
                        var w = q.editor.scene.getEye();
                        q.editor.scene.setEye(w[0], k, w[2])
                    }, "int", 1), this.addLabelInput(a, "Z", function (a) {
                        return parseFloat(q.editor.scene.getEye()[2])
                    }, function (a, k) {
                        var w = q.editor.scene.getEye();
                        q.editor.scene.setEye(w[0], w[1], k)
                    }, "int", 1), this.addRow(a, [this.indent - 40 - 8, 20, 20, .1, 20, .1, 20, .1]), a = [], a.push(bq("editor.center")), this.addButton(a, null, bq("editor.reset"), "editor.resetsize.state", function () {
                        q.editor.scene.setCenter(ht.Default.graph3dViewCenter)
                    }), this.addLabelInput(a, "X", function (a) {
                        return parseFloat(q.editor.scene.getCenter()[0])
                    }, function (a, k) {
                        var w = q.editor.scene.getCenter();
                        q.editor.scene.setCenter(k, w[1], w[2])
                    }, "int", 1), this.addLabelInput(a, "Y", function (a) {
                        return parseFloat(q.editor.scene.getCenter()[1])
                    }, function (a, k) {
                        var w = q.editor.scene.getCenter();
                        q.editor.scene.setCenter(w[0], k, w[2])
                    }, "int", 1), this.addLabelInput(a, "Z", function (a) {
                        return parseFloat(q.editor.scene.getCenter()[2])
                    }, function (a, k) {
                        var w = q.editor.scene.getCenter();
                        q.editor.scene.setCenter(w[0], w[1], k)
                    }, "int", 1), this.addRow(a, [this.indent - 40 - 8, 20, 20, .1, 20, .1, 20, .1]), a = [], this.addLabelRange(a, bq("editor.near"), function (q) {
                        return q.a("sceneNear")
                    }, function (q, a) {
                        return q.a("sceneNear", a)
                    }, 1, void 0, 1, "int"), this.addLabelRange(a, bq("editor.far"), function (q) {
                        return q.a("sceneFar")
                    }, function (q, a) {
                        return q.a("sceneFar", a)
                    }, 0, void 0, 100, "int"), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelColor(a, bq("editor.backgroundcolor"), hteditor.getter("p", "background"), hteditor.setter("p", "background")), this.addRow(a, [this.indent, .1]), a = [];
                    var k = [
                        ["Obejct", "json", "editor.comment.json"],
                        ["ht.DataModel", "dm", "editor.comment.datamodel"],
                        ["ht.graph.Graph3dView", "view", "editor.comment.graph3dview"]
                    ];
                    this.addLabelFunction(a, bq("editor.onpredeserialize"), function (q) {
                        return q.a("onPreDeserialize")
                    }, function (q, a) {
                        q.a("onPreDeserialize", a)
                    }, "onPreDeserialize", k), this.addOneRow(a), a = [], k = [
                        ["Obejct", "json", "editor.comment.json"],
                        ["ht.DataModel", "dm", "editor.comment.datamodel"],
                        ["ht.graph.Graph3dView", "view", "editor.comment.graph3dview"],
                        ["ht.List", "datas", "editor.comment.datas"]
                    ], this.addLabelFunction(a, bq("editor.onpostdeserialize"), function (q) {
                        return q.a("onPostDeserialize")
                    }, function (q, a) {
                        q.a("onPostDeserialize", a)
                    }, "onPostDeserialize", k), this.addOneRow(a), a = [], this.addLabelComboBox(a, bq("editor.orthographic"), function (q) {
                        return q.a("sceneOrthographic")
                    }, function (q, a) {
                        return q.a("sceneOrthographic", a)
                    }, [!1, "top", "right", "front"], [bq("editor.none"), bq("editor.viewtype.top"), bq("editor.viewtype.right"), bq("editor.viewtype.front")]), this.addOneRow(a)
                }, a.prototype.updateSkyboxDiameterRange = function (q) {
                    function a(q, a) {
                        return Math.log(a) / Math.log(q)
                    }
                    if (!this._updatingSkyboxDiameter) {
                        var k = this.editor.dm,
                            w = this.editor.scene;
                        this._updatingSkyboxDiameter = !0;
                        var I = k.a("sceneNear"),
                            K = k.a("sceneFar");
                        K = K * K / (K + I);
                        var b = Math.tan(w.getFovy() / 2) * I,
                            s = w.getAspect() * b,
                            u = Math.sqrt(3),
                            C = 1.1 * Math.sqrt(s * s + b * b + I * I),
                            L = K / u,
                            j = a(1.1, L),
                            G = a(1.1, C);
                        q._max = 2 * Math.pow(1.1, j), q._min = 2 * Math.pow(1.1, G), this._updatingSkyboxDiameter = !1
                    }
                }, a.prototype.getSkyboxDiameterMax = function () {
                    var q = this.editor.dm,
                        a = q.a("sceneNear"),
                        k = q.a("sceneFar");
                    k = k * k / (k + a);
                    var w = Math.sqrt(3),
                        I = k / w,
                        K = Math.log(I) / Math.log(1.1);
                    return 2 * Math.pow(1.1, K)
                }, a.prototype.addSkyboxProperties = function () {
                    var q = this;
                    this.addTitle("TitleSkybox");
                    var a = function (q) {
                            return "cube" === q.editor.dm.a("sceneSkyboxType")
                        },
                        k = function (q) {
                            return "sphere" === q.editor.dm.a("sceneSkyboxType")
                        },
                        w = function (q) {
                            return "color" === q.editor.dm.a("sceneSkyboxType")
                        },
                        I = [];
                    this.addLabelComboBox(I, bq("editor.type"), function (q) {
                        return q.a("sceneSkyboxType")
                    }, function (q, a) {
                        return q.a("sceneSkyboxType", a)
                    }, ["cube", "sphere", "color"], [bq("editor.cube"), bq("editor.sphere"), bq("editor.color")]), this.addRow(I, [this.indent, .1]), I = [];
                    var K = window.diameterSlider = this.addLabelSlider(I, bq("editor.skyboxdiameter"), function (a) {
                        var k = a.a("sceneSkyboxDiameter") || q.getSkyboxDiameterMax();
                        return q.updateSkyboxDiameterRange(K), Number(k.toFixed(0))
                    }, function (a, k) {
                        q.updateSkyboxDiameterRange(K), a.a("sceneSkyboxDiameter", Number(k.toFixed(0)))
                    }, 1, 100, 1, function (q) {
                        return Number(q.toFixed(0))
                    });
                    this.addInput(I, function (a) {
                        var k = a.a("sceneSkyboxDiameter") || q.getSkyboxDiameterMax();
                        return Number(k.toFixed(0))
                    }, function (a, k) {
                        q.updateSkyboxDiameterRange(K), a.a("sceneSkyboxDiameter", Number(k.toFixed(0)))
                    }, "number"), this.addRow(I, [this.indent, .1, 76]), I = [], this.addLabelColor(I, bq("editor.blend"), function (q) {
                        return q.a("sceneSkyboxBodyColor")
                    }, function (q, a) {
                        q.a("sceneSkyboxBodyColor", a || void 0)
                    }), this.addRow(I, [this.indent, .1]), I = [], I.push(bq("editor.rotation")), this.addButton(I, null, bq("editor.reset"), "editor.resetsize.state", function () {
                        q.editor.scene.dm().beginTransaction(), q.editor.scene.dm().a("sceneSkyboxRotationX", 0), q.editor.scene.dm().a("sceneSkyboxRotationY", 0), q.editor.scene.dm().a("sceneSkyboxRotationZ", 0), q.editor.scene.dm().endTransaction()
                    }), this.addLabelInput(I, "X", function (q) {
                        return Number((180 / Math.PI * q.a("sceneSkyboxRotationX")).toFixed(0))
                    }, function (q, a) {
                        q.a("sceneSkyboxRotationX", a * Math.PI / 180)
                    }, "int", Math.PI / 45), this.addLabelInput(I, "Y", function (q) {
                        return Number((180 / Math.PI * q.a("sceneSkyboxRotationY")).toFixed(0))
                    }, function (q, a) {
                        q.a("sceneSkyboxRotationY", a * Math.PI / 180)
                    }, "int", Math.PI / 45), this.addLabelInput(I, "Z", function (q) {
                        return Number((180 / Math.PI * q.a("sceneSkyboxRotationZ")).toFixed(0))
                    }, function (q, a) {
                        q.a("sceneSkyboxRotationZ", a * Math.PI / 180)
                    }, "int", Math.PI / 45), this.addRow(I, [this.indent - 40 - 8, 20, 20, .1, 20, .1, 20, .1]), I = [], this.addLabelImage(I, bq("editor.image"), function (a) {
                        var k = a.a("sceneSkyboxImage"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("sceneSkyboxImage", a)
                    }), this.addRow(I, [this.indent, .1, 20]).visible = k, I = [], this.addLabelImage(I, bq("editor.face.front"), function (a) {
                        var k = a.a("sceneSkyboxFrontImage"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("sceneSkyboxFrontImage", a)
                    }), this.addRow(I, [this.indent, .1, 20]).visible = a, I = [], this.addLabelImage(I, bq("editor.face.back"), function (a) {
                        var k = a.a("sceneSkyboxBackImage"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("sceneSkyboxBackImage", a)
                    }), this.addRow(I, [this.indent, .1, 20]).visible = a, I = [], this.addLabelImage(I, bq("editor.face.left"), function (a) {
                        var k = a.a("sceneSkyboxLeftImage"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("sceneSkyboxLeftImage", a)
                    }), this.addRow(I, [this.indent, .1, 20]).visible = a, I = [], this.addLabelImage(I, bq("editor.face.right"), function (a) {
                        var k = a.a("sceneSkyboxRightImage"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("sceneSkyboxRightImage", a)
                    }), this.addRow(I, [this.indent, .1, 20]).visible = a, I = [], this.addLabelImage(I, bq("editor.face.top"), function (a) {
                        var k = a.a("sceneSkyboxTopImage"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("sceneSkyboxTopImage", a)
                    }), this.addRow(I, [this.indent, .1, 20]).visible = a, I = [], this.addLabelImage(I, bq("editor.face.bottom"), function (a) {
                        var k = a.a("sceneSkyboxBottomImage"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("sceneSkyboxBottomImage", a)
                    }), this.addRow(I, [this.indent, .1, 20]).visible = a, I = [], this.addLabelColor(I, bq("editor.color"), function (q) {
                        return q.a("sceneSkyboxColor")
                    }, function (q, a) {
                        q.a("sceneSkyboxColor", a || void 0)
                    }), this.addRow(I, [this.indent, .1]).visible = w
                }, a.prototype.addBloomProperties = function () {
                    this.addTitle("TitleBloom");
                    var q = [];
                    this.addLabelCheckBox(q, bq("editor.bloom"), function (q) {
                        return q.a("sceneBloom")
                    }, function (q, a) {
                        return q.a("sceneBloom", a)
                    }), this.addLabelRange(q, bq("editor.strength"), function (q) {
                        return q.a("sceneBloomStrength")
                    }, function (q, a) {
                        return q.a("sceneBloomStrength", a)
                    }, 0, void 0, .01, "number"), this.addRow(q, [this.indent, .1, this.indent2, .1]), q = [], this.addLabelRange(q, bq("editor.radius"), function (q) {
                        return q.a("sceneBloomRadius")
                    }, function (q, a) {
                        return q.a("sceneBloomRadius", a)
                    }, 0, void 0, .01, "number"), this.addLabelRange(q, bq("editor.threshold"), function (q) {
                        return q.a("sceneBloomThreshold")
                    }, function (q, a) {
                        return q.a("sceneBloomThreshold", a)
                    }, 0, 1, .01, "number"), this.addRow(q, [this.indent, .1, this.indent2, .1]), q = [], this.addLabelCheckBox(q, bq("editor.bloomselective"), function (q) {
                        return q.a("sceneBloomSelective")
                    }, function (q, a) {
                        return q.a("sceneBloomSelective", a)
                    }), this.addRow(q, [this.indent, .1])
                }, a.prototype.addDofProperties = function () {
                    var q = this;
                    this.addTitle("TitleDOF");
                    var a = [];
                    this.addLabelCheckBox(a, bq("editor.dof"), function (q) {
                        return q.a("sceneDof")
                    }, function (q, a) {
                        return q.a("sceneDof", a)
                    }), this.addLabelRange(a, bq("editor.aperture"), function (q) {
                        return q.a("sceneDofAperture")
                    }, function (q, a) {
                        return q.a("sceneDofAperture", a)
                    }, 0, 1, .001, "number"), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelImage(a, bq("editor.image"), function (a) {
                        var k = a.a("sceneDofImage"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("sceneDofImage", a)
                    }), this.addRow(a, [this.indent, .1, 20])
                }, a.prototype.addHighlightProperties = function () {
                    var q = this;
                    this.addTitle("TitleHighlight");
                    var a = [];
                    this.addLabelComboBox(a, bq("editor.mode"), function (q) {
                        return q.a("sceneHighlightMode")
                    }, function (q, a) {
                        q.a("sceneHighlightMode", a)
                    }, ["disabled", "selected", "hover", "style"], [bq("editor.highlightmode.disabled"), bq("editor.highlightmode.selected"), bq("editor.highlightmode.hover"), bq("editor.highlightmode.style")]), this.addLabelComboBox(a, bq("editor.type"), function (q) {
                        return q.a("sceneHighlightType")
                    }, function (a, k) {
                        a.a("sceneHighlightType", k), q.editor.inspector.filterPropertiesLater()
                    }, ["hard", "soft"], [bq("editor.highlighttype.hard"), bq("editor.highlighttype.soft")]), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelRange(a, bq("editor.width"), function (q) {
                        return q.a("sceneHighlightWidth")
                    }, function (q, a) {
                        return q.a("sceneHighlightWidth", a)
                    }, 0, void 0, .1, "number"), this.addLabelColor(a, bq("editor.color"), function (q) {
                        return q.a("sceneHighlightColor")
                    }, function (q, a) {
                        q.a("sceneHighlightColor", a)
                    }), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelSlider(a, bq("editor.highlightglow"), function (q) {
                        return q.a("sceneHighlightGlow")
                    }, function (q, a) {
                        q.a("sceneHighlightGlow", a)
                    }, 0, 2, .02), this.addInput(a, function (q) {
                        return q.a("sceneHighlightGlow")
                    }, function (q, a) {
                        q.a("sceneHighlightGlow", a)
                    }, "number"), this.addRow(a, [this.indent, .1, 76]).visible = function (q) {
                        return "soft" === q.editor.dm.a("sceneHighlightType")
                    }, a = [], this.addLabelSlider(a, bq("editor.highlightstrength"), function (q) {
                        return q.a("sceneHighlightStrength")
                    }, function (q, a) {
                        q.a("sceneHighlightStrength", a)
                    }, 0, 5, .05), this.addInput(a, function (q) {
                        return q.a("sceneHighlightStrength")
                    }, function (q, a) {
                        q.a("sceneHighlightStrength", a)
                    }, "number"), this.addRow(a, [this.indent, .1, 76]).visible = function (q) {
                        return "soft" === q.editor.dm.a("sceneHighlightType")
                    }
                }, a.prototype.addHeadlightProperties = function () {
                    this.addTitle("TitleHeadlight");
                    var q = [];
                    this.addLabelCheckBox(q, bq("editor.enable"), function (q) {
                        return q.a("sceneHeadlightEnable")
                    }, function (q, a) {
                        return q.a("sceneHeadlightEnable", a)
                    }), this.addLabelColor(q, bq("editor.color"), function (q) {
                        return q.a("sceneHeadlightColor")
                    }, function (q, a) {
                        q.a("sceneHeadlightColor", a)
                    }), this.addRow(q, [this.indent, .1, this.indent2, .1]), q = [], this.addLabelRange(q, bq("editor.intensity"), function (q) {
                        return q.a("sceneHeadlightIntensity")
                    }, function (q, a) {
                        return q.a("sceneHeadlightIntensity", a)
                    }, 0, void 0, .01, "number"), this.addLabelRange(q, bq("editor.ambient"), function (q) {
                        return q.a("sceneHeadlightAmbientIntensity")
                    }, function (q, a) {
                        return q.a("sceneHeadlightAmbientIntensity", a)
                    }, void 0, void 0, .01, "number"), this.addRow(q, [this.indent, .1, this.indent2, .1])
                }, a.prototype.addFogProperties = function () {
                    var q = this,
                        a = function (q) {
                            return "exp2" === q.editor.dm.a("sceneFogMode")
                        },
                        k = function (q) {
                            return "linear" === q.editor.dm.a("sceneFogMode")
                        };
                    this.addTitle("TitleFog");
                    var w = [];
                    this.addLabelCheckBox(w, bq("editor.enable"), function (q) {
                        return q.a("sceneFogEnable")
                    }, function (q, a) {
                        return q.a("sceneFogEnable", a)
                    }), this.addLabelComboBox(w, bq("editor.mode"), function (q) {
                        return q.a("sceneFogMode")
                    }, function (a, k) {
                        a.a("sceneFogMode", k), q.editor.inspector.filterPropertiesLater()
                    }, ["exp2", "linear"], [bq("editor.fogmodeexp2"), bq("editor.fogmodelinear")]), this.addRow(w, [this.indent, .1, this.indent2, .1]), w = [], this.addLabelColor(w, bq("editor.color"), function (q) {
                        return q.a("sceneFogColor")
                    }, function (q, a) {
                        q.a("sceneFogColor", a)
                    }), this.addRow(w, [this.indent, .1]), w = [], this.addLabelSlider(w, bq("editor.strength"), function (q) {
                        return 1e4 * q.a("sceneFogDensity")
                    }, function (q, a) {
                        q.a("sceneFogDensity", a / 1e4)
                    }, 0, 100, 1), this.addInput(w, function (q) {
                        return 1e4 * q.a("sceneFogDensity")
                    }, function (q, a) {
                        q.a("sceneFogDensity", a / 1e4)
                    }, "number"), this.addRow(w, [this.indent, .1, 76]).visible = a, w = [], this.addLabelRange(w, bq("editor.near"), function (q) {
                        return q.a("sceneFogNear")
                    }, function (q, a) {
                        q.a("sceneFogNear", a)
                    }, 0, void 0, 10, "int"), this.addLabelRange(w, bq("editor.far"), function (q) {
                        return q.a("sceneFogFar")
                    }, function (q, a) {
                        q.a("sceneFogFar", a)
                    }, 0, void 0, 10, "int"), this.addRow(w, [this.indent, .1, this.indent, .1]).visible = k
                }, a.prototype.$77$ = function () {
                    this.addTitle("TitleGridsGuides");
                    var q = [];
                    this.addLabelColor(q, bq("editor.gridcolor"), function (q) {
                        return q.a("sceneGridColor")
                    }, function (q, a) {
                        q.a("sceneGridColor", a)
                    }), this.addRow(q, [this.indent, .1]), q = [], this.addLabelRange(q, bq("editor.gridsize"), function (q) {
                        return q.a("sceneGridBlockCount")
                    }, function (q, a) {
                        q.a("sceneGridBlockCount", a)
                    }, 0, void 0, 1, "int"), this.addLabelRange(q, bq("editor.gridgap"), function (q) {
                        return q.a("sceneGridBlockSize")
                    }, function (q, a) {
                        q.a("sceneGridBlockSize", a)
                    }, 1, void 0, 1, "int"), this.addRow(q, [this.indent, .1, this.indent2, .1])
                }, a.prototype.addHueSaturation = function () {
                    this.addTitle("TitleHueSaturation");
                    var q = [];
                    this.addLabelCheckBox(q, bq("editor.enable"), function (q) {
                        return q.a("sceneHueSaturation")
                    }, function (q, a) {
                        return q.a("sceneHueSaturation", a)
                    }), this.addLabelComboBox(q, bq("editor.color"), function (q) {
                        return q.a("sceneHueSaturationColorIndex")
                    }, function (q, a) {
                        q.a("sceneHueSaturationColorIndex", a)
                    }, [0, 1, 2, 3, 4, 5, 6], [bq("editor.huesaturationcolor.all"), bq("editor.huesaturationcolor.red"), bq("editor.huesaturationcolor.yellow"), bq("editor.huesaturationcolor.green"), bq("editor.huesaturationcolor.cyan"), bq("editor.huesaturationcolor.blue"), bq("editor.huesaturationcolor.magenta")]), this.addRow(q, [this.indent, .1, this.indent2, .1]), q = [], this.addLabelSlider(q, bq("editor.hue"), function (q) {
                        var a = q.a("sceneHueSaturationColorIndex") || 0;
                        return (q.a("sceneHueSaturationHue") || [])[a]
                    }, function (q, a) {
                        var k = q.a("sceneHueSaturationColorIndex") || 0,
                            w = ht.Default.clone(q.a("sceneHueSaturationHue"));
                        w[k] = a, q.a("sceneHueSaturationHue", w)
                    }, -180, 180, 1), this.addInput(q, function (q) {
                        var a = q.a("sceneHueSaturationColorIndex") || 0;
                        return (q.a("sceneHueSaturationHue") || [])[a]
                    }, function (q, a) {
                        var k = q.a("sceneHueSaturationColorIndex") || 0,
                            w = ht.Default.clone(q.a("sceneHueSaturationHue"));
                        w[k] = a, q.a("sceneHueSaturationHue", w)
                    }, "int"), this.addRow(q, [this.indent, .1, 76]), q = [], this.addLabelSlider(q, bq("editor.saturation"), function (q) {
                        var a = q.a("sceneHueSaturationColorIndex") || 0;
                        return (q.a("sceneHueSaturationSaturation") || [])[a]
                    }, function (q, a) {
                        var k = q.a("sceneHueSaturationColorIndex") || 0,
                            w = ht.Default.clone(q.a("sceneHueSaturationSaturation"));
                        w[k] = a, q.a("sceneHueSaturationSaturation", w)
                    }, -180, 180, 1), this.addInput(q, function (q) {
                        var a = q.a("sceneHueSaturationColorIndex") || 0;
                        return (q.a("sceneHueSaturationSaturation") || [])[a]
                    }, function (q, a) {
                        var k = q.a("sceneHueSaturationColorIndex") || 0,
                            w = ht.Default.clone(q.a("sceneHueSaturationSaturation"));
                        w[k] = a, q.a("sceneHueSaturationSaturation", w)
                    }, "int"), this.addRow(q, [this.indent, .1, 76]), q = [], this.addLabelSlider(q, bq("editor.lightness"), function (q) {
                        var a = q.a("sceneHueSaturationColorIndex") || 0;
                        return (q.a("sceneHueSaturationLightness") || [])[a]
                    }, function (q, a) {
                        var k = q.a("sceneHueSaturationColorIndex") || 0,
                            w = ht.Default.clone(q.a("sceneHueSaturationLightness"));
                        w[k] = a, q.a("sceneHueSaturationLightness", w)
                    }, -180, 180, 1), this.addInput(q, function (q) {
                        var a = q.a("sceneHueSaturationColorIndex") || 0;
                        return (q.a("sceneHueSaturationLightness") || [])[a]
                    }, function (q, a) {
                        var k = q.a("sceneHueSaturationColorIndex") || 0,
                            w = ht.Default.clone(q.a("sceneHueSaturationLightness"));
                        w[k] = a, q.a("sceneHueSaturationLightness", w)
                    }, "int"), this.addRow(q, [this.indent, .1, 76])
                }, a.prototype.initShadowProperties = function () {
                    this.editor.scene, this.editor.scene.shadowMap;
                    this.addTitle("TitleShadow");
                    var q = [];
                    this.addLabelCheckBox(q, bq("editor.shadow.enable"), function (q) {
                        return q.a("sceneShadowEnabled")
                    }, function (q, a) {
                        return q.a("sceneShadowEnabled", a)
                    }), this.addRow(q, [this.indent, .1]), q = [], this.addLabelRange(q, bq("editor.shadow.degreeX"), function (q) {
                        return q.a("sceneShadowDegreeX")
                    }, function (q, a) {
                        return q.a("sceneShadowDegreeX", a)
                    }, 0, 360, .01, "number"), this.addRow(q, [this.indent, .1]), q = [], this.addLabelRange(q, bq("editor.shadow.degreeZ"), function (q) {
                        return q.a("sceneShadowDegreeZ")
                    }, function (q, a) {
                        return q.a("sceneShadowDegreeZ", a)
                    }, -90, 90, .01, "number"), this.addRow(q, [this.indent, .1]), q = [], this.addLabelRange(q, bq("editor.shadow.intensity"), function (q) {
                        return q.a("sceneShadowIntensity")
                    }, function (q, a) {
                        return q.a("sceneShadowIntensity", a)
                    }, 0, 1, .01, "number"), this.addRow(q, [this.indent, .1]), q = [], this.addLabelComboBox(q, bq("editor.shadow.quality"), function (q) {
                        return q.a("sceneShadowQuality")
                    }, function (q, a) {
                        return q.a("sceneShadowQuality", a)
                    }, ["low", "medium", "high", "ultra"], [bq("editor.shadow.quality.low"), bq("editor.shadow.quality.medium"), bq("editor.shadow.quality.high"), bq("editor.shadow.quality.ultra")]), this.addRow(q, [this.indent, .1]), q = [], this.addLabelComboBox(q, bq("editor.shadow.type"), function (q) {
                        return q.a("sceneShadowType")
                    }, function (q, a) {
                        return q.a("sceneShadowType", a)
                    }, ["none", "hard", "soft"], [bq("editor.shadow.type.none"), bq("editor.shadow.type.hard"), bq("editor.shadow.type.soft")]), this.addRow(q, [this.indent, .1]), q = [], this.addLabelRange(q, bq("editor.shadow.radius"), function (q) {
                        return q.a("sceneShadowRadius")
                    }, function (q, a) {
                        return q.a("sceneShadowRadius", a)
                    }, 0, 100, .1, "number"), this.addRow(q, [this.indent, .1]), q = [], this.addLabelRange(q, bq("editor.shadow.bias"), function (q) {
                        return q.a("sceneShadowBias")
                    }, function (q, a) {
                        return q.a("sceneShadowBias", a)
                    }, -.01, .01, 1e-4, "number"), this.addRow(q, [this.indent, .1])
                }, a.prototype.addEnvmapProperties = function () {
                    var q = this,
                        a = function (q) {
                            return "new" !== q.editor.dm.getEnvmapType()
                        },
                        k = function (q) {
                            return "new" === q.editor.dm.getEnvmapType()
                        };
                    this.addTitle("TitleEnvmap");
                    var w = [];
                    this.addLabelComboBox(w, bq("editor.envmaptype"), function (q) {
                        return q.getEnvmapType() || "legacy"
                    }, function (a, k) {
                        a.setEnvmapType(k), "new" === k ? a.setEnvmap([{
                            type: "sphere"
                        }]) : a.setEnvmap(void 0), q.editor.inspector.filterPropertiesLater(), q.editor.batchView && q.editor.batchView.updateBatchInfoDialog()
                    }, ["legacy", "new"], [bq("editor.envmaptype.legacy"), bq("editor.envmaptype.new")]), this.addRow(w, [this.indent, .1]).visible = a, w = [], this.addLabelComboBox(w, bq("editor.envmaptype"), function (q) {
                        return q.getEnvmapType() || "legacy"
                    }, function (a, k) {
                        a.setEnvmapType(k), "new" === k ? a.setEnvmap([{
                            type: "sphere"
                        }]) : a.setEnvmap(void 0), q.editor.inspector.filterPropertiesLater(), q.editor.batchView && q.editor.batchView.updateBatchInfoDialog()
                    }, ["legacy", "new"], [bq("editor.envmaptype.legacy"), bq("editor.envmaptype.new")]);
                    var I = sq(null, bq("editor.add"), "editor.add", function () {
                        q.addEnvmap()
                    });
                    w.push(I), I.getIconColor = function () {
                        return this.isPressed() ? R.config.color_select : R.config.color_dark
                    }, this.addRow(w, [this.indent, .1, 20]).visible = k, w = [], this.addLabelImage(w, bq("editor.image"), function (a) {
                        var k = a.getEnvmap();
                        if (!Z(k)) return null;
                        var w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.setEnvmap(a)
                    }), this.addRow(w, [this.indent, .1, 20]).visible = a, w = [], this.envmapView || (this.envmapView = this.initEnvmapView()), w.push(this.envmapView), this.envmapRow = this.addRow(w, [.1], 2), this.envmapRow.visible = k, this.updateHandlers.push(function () {
                        q.editor.dm && "new" === q.editor.dm.getEnvmapType("envmapType") && q.isEnvmapChange() && (q.envmapView._editing ? q.envmapView._editing = !1 : q.parseEnvmap(q.editor.dm.getEnvmap()))
                    })
                }, a.prototype.initEnvmapView = function () {
                    var q = new Kq(this.editor, "Envmap", "data");
                    return q.setPadding(0), q.setLabelVPadding(0), q.getValue = function (q) {
                        return q(this.data)
                    }, q.setValue = function (q, a, k) {
                        q(this.data, k ? k() : a)
                    }, q.updateProperties = function () {
                        this._updating = !0, this.updateHandlers.forEach(function (q) {
                            q()
                        }), this._updating = !1, this._updatePropertiesLater = !1
                    }, q.data = new ht.Data, q
                }, a.prototype.envmapViewAddRow = function (q) {
                    var a = this,
                        k = function (a) {
                            return a.data.a(q) && "sphere" === a.data.a(q).type
                        },
                        w = function (a) {
                            return a.data.a(q) && "cube" === a.data.a(q).type
                        },
                        I = function (a) {
                            return a.data.a(q) && "probe" === a.data.a(q).type
                        },
                        K = this.envmapView,
                        b = [];
                    K.addLabelInput(b, bq("editor.name"), function (a) {
                        return a.a(q).name
                    }, function (a, k) {
                        var w = ht.Default.clone(a.a(q));
                        w.name = k, a.a(q, w)
                    }), K.addLabelComboBox(b, bq("editor.type"), function (a) {
                        return a.a(q).type
                    }, function (k, w) {
                        var I = ht.Default.clone(k.a(q));
                        I.type !== w && (I = "cube" === w ? {
                            type: "cube",
                            images: []
                        } : "probe" === w ? {
                            type: "probe",
                            pos: [0, 0, 0],
                            size: 100
                        } : {
                            type: w
                        }, k.a(q, I), a.resizeEnvmapRow())
                    }, ["sphere", "cube", "skybox", "probe"], [bq("editor.sphere"), bq("editor.cube"), bq("editor.skybox"), bq("editor.envmapitem.type.prob")]);
                    var s = sq(null, bq("editor.delete"), "editor.delete", function () {
                        a.deleteEnvmap(q)
                    });
                    b.push(s), s.getIconColor = function () {
                        return this.isPressed() ? R.config.color_select : R.config.color_dark
                    }, K.addRow(b, [this.indent, .1, this.indent2, .1, 20]), b = [];
                    var u = new ht.widget.Image;
                    u.setIcon("editor.envmap.l"), b.push(u), b.push(bq("editor.image")), K.addLabelImage(b, "", function (k) {
                        var w = k.a(q).image;
                        if (!Z(w)) return null;
                        var I = a.editor.getFileNode(w);
                        return I ? I.url : w
                    }, function (a, k) {
                        var w = ht.Default.clone(a.a(q));
                        w.image = k, a.a(q, w)
                    }), K.addRow(b, [20, this.indent - 20 - 8, 0, .1, 20]).visible = k;
                    for (var C = ["editor.face.right", "editor.face.left", "editor.face.top", "editor.face.bottom", "editor.face.front", "editor.face.back"], L = 0; L < 6; L++) ! function (k) {
                        b = [];
                        var I = new ht.widget.Image;
                        5 === k ? I.setIcon("editor.envmap.l") : I.setIcon("editor.envmap.t"), b.push(I), b.push(bq(C[k])), K.addLabelImage(b, "", function (w) {
                            var I = w.a(q).images;
                            if (I instanceof Array) {
                                var K = a.editor.getFileNode(I[k]);
                                return K ? K.url : I[k]
                            }
                        }, function (a, w) {
                            var I = ht.Default.clone(a.a(q));
                            I.images[k] = w, a.a(q, I)
                        }), K.addRow(b, [20, a.indent - 20 - 8, 0, .1, 20]).visible = w
                    }(L);
                    b = [];
                    var j = new ht.widget.Image;
                    j.setIcon("editor.envmap.t"), b.push(j), b.push(bq("editor.size")), K.addInput(b, function (a) {
                        return a.a(q).size
                    }, function (a, k) {
                        var w = ht.Default.clone(a.a(q));
                        w.size = k, a.a(q, w)
                    }, "int"), K.addLabelFunction(b, bq("editor.envmapitem.filter"), function (a) {
                        return a.a(q).filter
                    }, function (k, w) {
                        var I = ht.Default.clone(k.a(q));
                        I.filter = w, k.a(q, I), a.editor.scene.invalidateEnvmap()
                    }, "filter", "data");
                    var G = sq(null, bq("editor.refresh"), "editor.refresh", function () {
                        a.editor.scene.invalidateEnvmap()
                    });
                    b.push(G), G.getIconColor = function () {
                        return this.isPressed() ? R.config.color_select : R.config.color_dark
                    }, K.addRow(b, [20, this.indent - 20 - 4, .1, this.indent2, .1, 20]).visible = I, b = [];
                    var X = new ht.widget.Image;
                    X.setIcon("editor.envmap.l"), b.push(X), b.push(bq("editor.position")), K.addLabelInput(b, "X", function (a) {
                        if (a.a(q).pos) return a.a(q).pos[0]
                    }, function (a, k) {
                        var w = ht.Default.clone(a.a(q));
                        w.pos[0] = k, a.a(q, w)
                    }, "int"), K.addLabelInput(b, "Y", function (a) {
                        if (a.a(q).pos) return a.a(q).pos[1]
                    }, function (a, k) {
                        var w = ht.Default.clone(a.a(q));
                        w.pos[1] = k, a.a(q, w)
                    }, "int"), K.addLabelInput(b, "Z", function (a) {
                        if (a.a(q).pos) return a.a(q).pos[2]
                    }, function (a, k) {
                        var w = ht.Default.clone(a.a(q));
                        w.pos[2] = k, a.a(q, w)
                    }, "int"), K.addRow(b, [20, this.indent - 8 - 20 - 20, 20, .1, 20, .1, 20, .1]).visible = I
                }, a.prototype.parseEnvmap = function (q) {
                    var a = this,
                        k = this.envmapView;
                    if (this.clearEnvmap(), q) {
                        var w = {};
                        q.forEach(function (q, k) {
                            w[k] = q, a.envmapViewAddRow(k)
                        }), k.$46$ = k._rows, k._envmap = ht.Default.stringify(q), k.data.onPropertyChanged = function (q) {
                            k.updateProperties(), "attrObject" !== q.property && a.toEnvmap(k.data)
                        }, k.data.setAttrObject(w)
                    }
                    this.resizeEnvmapRow()
                }, a.prototype.resizeEnvmapRow = function () {
                    this.envmapView.filterProperties(), this.envmapView.setRowHeight(this.getRowHeight()), this.envmapView.setVGap(this.getVGap());
                    var q = this.envmapView.getRows().length;
                    this.envmapRow.height = q ? q * (this.getRowHeight() + this.getVGap()) - this.getVGap() : 2, this.filterPropertiesLater()
                }, a.prototype.toEnvmap = function (q) {
                    this.envmapView._editing = !0;
                    var a = q.getAttrObject(),
                        k = [];
                    for (var w in a) k.push(a[w]);
                    this.envmapView._envmap = ht.Default.stringify(k), this.editor.dm.setEnvmap(k)
                }, a.prototype.isEnvmapChange = function () {
                    return ht.Default.stringify(this.editor.dm.getEnvmap()) !== this.envmapView._envmap
                }, a.prototype.addEnvmap = function () {
                    var q = ht.Default.clone(this.editor.dm.getEnvmap()) || [];
                    q.push({
                        type: "sphere"
                    }), this.editor.dm.setEnvmap(q), this.parseEnvmap(this.editor.dm.getEnvmap())
                }, a.prototype.deleteEnvmap = function (q) {
                    var a = ht.Default.clone(this.editor.dm.getEnvmap());
                    a.splice(q, 1), this.editor.dm.setEnvmap(a), this.parseEnvmap(this.editor.dm.getEnvmap()), ht.Default.hideToolTip()
                }, a.prototype.clearEnvmap = function () {
                    var q = this.envmapView;
                    q.data.onPropertyChanged = function () {}, q._envmap = void 0, q.$46$ = [], q.data.setAttrObject({}), q.updateHandlers = [], q.clear()
                }, a
        }(Kq),
        Cq = hteditor.getString,
        Lq = {
            undefined: "editor.cube",
            null: "editor.cube",
            sphere: "editor.sphere",
            cylinder: "editor.cylinder",
            cone: "editor.scene.cone",
            torus: "editor.scene.torus",
            triangle: "editor.scene.triangle",
            rightTriangle: "editor.scene.rightTriangle",
            parallelogram: "editor.scene.parallelogram",
            trapezoid: "editor.scene.trapezoid",
            rect: "editor.scene.rect",
            roundRect: "editor.scene.roundRect",
            star: "editor.scene.star",
            box: "editor.scene.box",
            billboard: "editor.scene.billboard",
            plane: "editor.scene.plane"
        },
        jq = function (q) {
            function a(k, w) {
                return V(this, a), Q(this, q.call(this, k, w, "data3d", !1))
            }
            return E(a, q), a.prototype.initForm = function () {
                q.prototype.initForm.call(this), this.$11$(), this.addTransformProperties(), this.addWireframeProperties(), this.addBloomProperties(), this.addHighlightProperties(), this.addReflectorProperties(), this.addEnvmapProperties(), this.addClipboxProperties(), this.editor.scene.shadowMap && this.$19$()
            }, a.prototype.$11$ = function () {
                var q = this;
                this.addTitle("TitleBasic"), this.addEventProperties();
                var a = [];
                this.addLabelInput(a, Cq("editor.name"), hteditor.getter("p", "displayName"), hteditor.setter("p", "displayName")), this.addLabelInput(a, Cq("editor.tag"), hteditor.getter("p", "tag"), function (a, k) {
                    if (R.config.checkTagConflicts && k)
                        for (var w = q.dataModel.getDatas(), I = w.size() - 1; I >= 0; I--) {
                            var K = w.get(I);
                            if (K !== a && K.getTag() === k) return q.editor.messageView.show(m(Cq("editor.errormessage.tagalreadyexists"), k), "error"), void q.$34$()
                        }
                    a.setTag(k)
                }), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelInput(a, Cq("editor.tooltip"), hteditor.getter("p", "toolTip"), hteditor.setter("p", "toolTip")), this.addRow(a, [this.indent, .1]);
                var k = [void 0, "billboard", "plane", "box", "sphere", "cone", "torus", "cylinder", "star", "rect", "roundRect", "triangle", "rightTriangle", "parallelogram", "trapezoid"],
                    w = [],
                    I = [];
                k.forEach(function (q) {
                    w.push(Cq("editor." + (q ? q.toLowerCase() : "cube"))), I.push({
                        width: 16,
                        height: 16,
                        comps: [{
                            type: "image",
                            stretch: "centerUniform",
                            color: R.config.color_dark,
                            name: Lq[q],
                            rect: [0, 0, 16, 16]
                        }]
                    })
                }), a = [], this.addLabelComboBoxURL(a, Cq("editor.type"), function (a) {
                    var k = a.s("shape3d");
                    if (u(k)) {
                        var w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }
                    return k
                }, function (a, k) {
                    a.s("shape3d", k), q.editor.updateInspector()
                }, k, w, I, function (q, a) {
                    return !!a.view.draggingData && "model" === a.view.draggingData.fileType
                }), this.addImage(a, function (q) {
                    return q.s("shape3d")
                }, void 0, function (q) {
                    return u(q) ? q.substr(0, q.length - 5) + ".png" : null
                }, function (q) {
                    return hteditor.isJSON(q.rawIcon)
                }), this.addOneRow(a).visible = function (q) {
                    return !(q.data instanceof ht.Shape)
                }, a = [], this.addLabelCheckBox(a, Cq("editor.selectable"), function (q) {
                    return q.s("3d.selectable")
                }, function (q, a) {
                    q.s("2d.selectable", a), q.s("3d.selectable", a)
                }), this.addLabelCheckBox(a, Cq("editor.movable"), function (q) {
                    return q.s("3d.movable")
                }, function (q, a) {
                    q.s("2d.movable", a), q.s("3d.movable", a)
                }), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelCheckBox(a, Cq("editor.editable"), function (q) {
                    return q.s("3d.editable")
                }, function (q, a) {
                    q.s("2d.editable", a), q.s("3d.editable", a)
                }), this.addLabelCheckBox(a, Cq("editor.visible"), function (q) {
                    return q.s("3d.visible")
                }, function (q, a) {
                    q.s("2d.visible", a), q.s("3d.visible", a)
                }), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [];
                var K = function (q) {
                    var a = q.s("fixSizeOnScreen");
                    return null == a && (a = q.s("shape3d.fixSizeOnScreen")), !!a
                };
                this.addLabelCheckBox(a, Cq("editor.fitsize"), K, function (q, a) {
                    q.s("fixSizeOnScreen", !!a && [-1, -1]), null != q.s("shape3d.fixSizeOnScreen") && q.s("shape3d.fixSizeOnScreen", void 0), a && q.s("autorotate", !0)
                });
                var b = this.addLabelCheckBox(a, Cq("editor.dynamic"), function (q) {
                    var a = q.s("vector.dynamic");
                    return null == a && (a = q.s("shape3d.vector.dynamic")), !!a
                }, function (q, a) {
                    q.s("vector.dynamic", a), null != q.s("shape3d.vector.dynamic") && q.s("shape3d.vector.dynamic", void 0)
                });
                this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelCheckBox(a, Cq("editor.alwaysontop"), function (q) {
                    if (q.getRenderLayer) return "top" === q.getRenderLayer();
                    var a = q.s("alwaysOnTop");
                    return null == a && (a = q.s("shape3d.alwaysOnTop")), a
                }, function (q, a) {
                    q.setRenderLayer ? (q.setRenderLayer(a ? "top" : null), q.s("alwaysOnTop", void 0)) : q.s("alwaysOnTop", a), q.s("shape3d.alwaysOnTop", void 0)
                });
                var s = function (q) {
                        var a = q.s("autorotate");
                        return null == a && (a = q.s("shape3d.autorotate")), a
                    },
                    C = this.addLabelComboBox(a, Cq("editor.autorotate"), s, function (q, a) {
                        q.s("autorotate", a), null != q.s("shape3d.autorotate") && q.s("shape3d.autorotate", void 0)
                    }, [!1, !0, "x", "y", "z"]);
                this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelCheckBox(a, Cq("editor.texturecache"), function (q) {
                    return q.s("texture.cache")
                }, function (a, k) {
                    a.s("texture.cache", k), null != a.s("shape3d.image.cache") && a.s("shape3d.image.cache", void 0), q.editor.scene.invalidateShape3dCachedImage(a)
                }), this.addLabelRange(a, Cq("editor.texturescale"), function (q) {
                    var a = void 0;
                    return 1 !== q.s("texture.scale") ? a = q.s("texture.scale") : 1 !== q.s("shape3d.texture.scale") && (a = q.s("shape3d.texture.scale")), a || 1
                }, function (q, a) {
                    q.s("texture.scale", a), q.s("shape3d.texture.scale", void 0)
                }, 0, 10, .1, "number"), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelColor(a, Cq("editor.blend"), hteditor.getter("s", "body.color"), hteditor.setter("s", "body.color")), this.addRow(a, [this.indent, .1]).visible = function (q) {
                    var a = q.data,
                        k = ht.Default.getShape3dModel(a.s("shape3d"));
                    return !k || !k.model3d
                }, a = [];
                var L = this.addLabelSlider(a, Cq("editor.clip"), hteditor.getter("s", "3d.clip.percentage"), hteditor.setter("s", "3d.clip.percentage"), 0, 1, .001);
                this.addInput(a, hteditor.getter("s", "3d.clip.percentage"), hteditor.setter("s", "3d.clip.percentage"), "number"), this.addComboBox(a, function (q) {
                        return q.s("3d.clip.direction")
                    }, function (q, a) {
                        q.s("3d.clip.direction", a)
                    }, [null, "left", "right", "top", "bottom", "front", "back"], ["", Cq("editor.clipdirection.left"), Cq("editor.clipdirection.right"), Cq("editor.clipdirection.top"), Cq("editor.clipdirection.bottom"), Cq("editor.clipdirection.front"), Cq("editor.clipdirection.back")]), this.addRow(a, [this.indent, .1, 52, .1]), a = [],
                    this.addLabelSlider(a, Cq("editor.transparent"), function (q) {
                        return void 0 === q.s("shape3d.opacity") ? 1 : q.s("shape3d.opacity")
                    }, hteditor.setter("s", "shape3d.opacity"), 0, 1, .01), this.addInput(a, function (q) {
                        return void 0 === q.s("shape3d.opacity") ? 1 : q.s("shape3d.opacity")
                    }, hteditor.setter("s", "shape3d.opacity"), "number"), this.addCheckBox(a, hteditor.getter("s", "shape3d.transparent"), hteditor.setter("s", "shape3d.transparent")), this.addRow(a, [this.indent, .1, 52, 20]).visible = function (q) {
                        if (q.data instanceof ht.Shape) return !!q.data.s("shape3d") || -1 === q.data.getThickness();
                        if (q.data.s("shape3d")) {
                            var a = ht.Default.getShape3dModel(q.data.s("shape3d"));
                            return !a || !a.model3d
                        }
                        return !1
                    }, a = [], this.addLabelRange(a, Cq("editor.alphaTest"), hteditor.getter("s", "alphaTest"), hteditor.setter("s", "alphaTest"), 0, 1, .01, "number"), this.addLabelCheckBox(a, Cq("editor.transparentmask"), hteditor.getter("s", "transparent.mask"), hteditor.setter("s", "transparent.mask")), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelRange(a, Cq("editor.selectedBrightness"), function (q) {
                        return q.s("select.brightness")
                    }, hteditor.setter("s", "select.brightness"), 0, 1, .01, "number"), this.addLabelRange(a, Cq("editor.brightness"), function (q) {
                        return q.s("brightness")
                    }, hteditor.setter("s", "brightness"), 0, 1, .01, "number"), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelObject(a, Cq("editor.polygonoffset"), function (q) {
                        return q.s("polygonOffset")
                    }, hteditor.setter("s", "polygonOffset"), "number"), this.addRow(a, [this.indent, .1]), a = [];
                var j = this.addLabelComboBox(a, Cq("editor.renderlayer"), function (q) {
                    return q.getRenderLayer()
                }, function (q, a) {
                    q.setRenderLayer(a)
                });
                this.addOneRow(a), a = [];
                var G = function (q) {
                    var a = this;
                    G.superClass.constructor.call(a, q);
                    var k = a._listView = new ht.widget.ListView;
                    k.getView().style.background = "white", k.setCheckMode(!0), k.sm().ms(function (w) {
                        var I = k.dm().getDataByTag(4294967295);
                        if ("append" === w.kind)
                            if (w.datas.get(0) === I) k.sm().selectAll();
                            else {
                                var K = k.sm().getSelection().length;
                                K === k.dm().getDatas().length - 1 && k.sm().selectAll()
                            }
                        else if ("remove" === w.kind)
                            if (w.datas.get(0) === I) k.sm().clearSelection();
                            else {
                                var b = k.sm().getSelection().toList(function (q) {
                                    return q !== I
                                });
                                k.sm().setSelection(b)
                            } q.setValue(a.getValue())
                    })
                };
                ht.Default.def(G, ht.widget.BaseDropDownTemplate, {
                    getView: function () {
                        return this._listView.getView()
                    },
                    onOpened: function (q) {
                        var a = this._listView,
                            k = new ht.Data;
                        k.setName(Cq("editor.all")), k.setTag(4294967295), a.dm().add(k);
                        var w = new ht.Data;
                        w.setName(Cq("editor.default")), w.setTag(0), a.dm().add(w);
                        var I = this._master.getMaskMap();
                        q = this._master.getValue();
                        for (var K in I) {
                            K = parseInt(K);
                            var b = new ht.Data;
                            b.setName(I[K]), b.setTag(K), a.dm().add(b)
                        }
                        if (q)
                            if (4294967295 === q) a.sm().selectAll();
                            else {
                                var s = new ht.List;
                                a.dm().each(function (a) {
                                    q & 1 << a.getTag() && s.add(a)
                                }), a.sm().setSelection(s)
                            }
                    },
                    onClosed: function () {},
                    getValue: function () {
                        var q = this._listView.dm(),
                            a = q.sm();
                        if (0 === a.size()) return 0;
                        var k = q.getDataByTag(4294967295);
                        if (a.contains(k)) return 4294967295;
                        var w = 0;
                        return a.each(function (q) {
                            var a = q.getTag();
                            4294967295 !== a && q.getName() && (w += 1 << a)
                        }), w
                    },
                    getHeight: function () {
                        return 100
                    }
                });
                var X = new ht.widget.MultiComboBox;
                X.setEditable(!1), X.setDropDownComponent(G), X.drawValue = function (q, a, k, w, I) {
                    var K = X.getValue(),
                        b = X.getMaskMap(),
                        s = [];
                    if (4294967295 === K) s = Cq("editor.all");
                    else if (0 === K) s = "";
                    else {
                        1 & K && s.push(Cq("editor.default"));
                        for (var u in b) u = parseInt(u), K & 1 << u && s.push(b[u]);
                        s.join(", ")
                    }
                    ht.Default.drawText(q, s, X.getLabelFont(), X.getLabelColor(), a + 1, k, 0, I)
                }, X.getMaskMap = function () {
                    return q.editor.dm.a("sceneLightGruopAlias")
                }, X.getValue = function () {
                    return q.data.s("light.mask")
                }, X.onValueChanged = function (a, k) {
                    q.editor.sm.getSelection().each(function (q) {
                        q.s("light.mask", k)
                    })
                }, a.push(Cq("editor.lightmask")), a.push(X), this.addLabelCheckBox(a, Cq("editor.light"), hteditor.getter("s", "shape3d.light"), hteditor.setter("s", "shape3d.light")), this.addRow(a, [this.indent, .1, this.indent2, 20]), a = [];
                var Z = function (q) {
                    var a = this;
                    Z.superClass.constructor.call(a, q);
                    var k = a._listView = new ht.widget.ListView;
                    k.getView().style.background = "white", k.setCheckMode(!0), k.sm().ms(function (w) {
                        var I = k.dm().getDataByTag(4294967295);
                        if ("append" === w.kind)
                            if (w.datas.get(0) === I) k.sm().selectAll();
                            else {
                                var K = k.sm().getSelection().length;
                                K === k.dm().getDatas().length - 1 && k.sm().selectAll()
                            }
                        else if ("remove" === w.kind)
                            if (w.datas.get(0) === I) k.sm().clearSelection();
                            else {
                                var b = k.sm().getSelection().toList(function (q) {
                                    return q !== I
                                });
                                k.sm().setSelection(b)
                            } q.setValue(a.getValue())
                    })
                };
                ht.Default.def(Z, ht.widget.BaseDropDownTemplate, {
                    getView: function () {
                        return this._listView.getView()
                    },
                    onOpened: function (q) {
                        var a = this._listView,
                            k = new ht.Data;
                        k.setName(Cq("editor.all")), k.setTag(4294967295), a.dm().add(k);
                        var w = new ht.Data;
                        w.setName(Cq("editor.default")), w.setTag(0), a.dm().add(w);
                        var I = this._master.getMaskMap();
                        q = this._master.getValue();
                        for (var K in I) {
                            K = parseInt(K);
                            var b = new ht.Data;
                            b.setName(I[K]), b.setTag(K), a.dm().add(b)
                        }
                        if (q)
                            if (4294967295 === q) a.sm().selectAll();
                            else {
                                var s = new ht.List;
                                a.dm().each(function (a) {
                                    q & 1 << a.getTag() && s.add(a)
                                }), a.sm().setSelection(s)
                            }
                    },
                    onClosed: function () {},
                    getValue: function () {
                        var q = this._listView.dm(),
                            a = q.sm();
                        if (0 === a.size()) return 0;
                        var k = q.getDataByTag(4294967295);
                        if (a.contains(k)) return 4294967295;
                        var w = 0;
                        return a.each(function (q) {
                            var a = q.getTag();
                            4294967295 !== a && q.getName() && (w += 1 << a)
                        }), w
                    },
                    getHeight: function () {
                        return 100
                    }
                });
                var r = new ht.widget.MultiComboBox;
                if (r.setEditable(!1), r.setDropDownComponent(Z), r.drawValue = function (q, a, k, w, I) {
                        var K = r.getValue(),
                            b = r.getMaskMap(),
                            s = [];
                        if (4294967295 === K) s = Cq("editor.all");
                        else if (0 === K) s = "";
                        else {
                            1 & K && s.push(Cq("editor.default"));
                            for (var u in b) u = parseInt(u), K & 1 << u && s.push(b[u]);
                            s.join(", ")
                        }
                        ht.Default.drawText(q, s, r.getLabelFont(), r.getLabelColor(), a + 1, k, 0, I)
                    }, r.getMaskMap = function () {
                        return q.editor.dm.a("sceneFlowEffectGroupAlias")
                    }, r.getValue = function () {
                        return q.data.s("effect.flow.mask")
                    }, r.onValueChanged = function (a, k) {
                        q.editor.sm.getSelection().each(function (q) {
                            q.s("effect.flow.mask", k)
                        })
                    }, a.push(Cq("editor.effect.flowmask")), a.push(r), this.addRow(a, [this.indent, .1]), R.config.batchEditable) {
                    a = [];
                    this.addLabelComboBox(a, Cq("editor.batch"), function (q) {
                        return q.s("batch")
                    }, function (q, a) {
                        return q.s("batch", a)
                    }, [], []).getValues = function () {
                        return q.editor.batchView.keys
                    }, this.addOneRow(a).visible = function (q) {
                        var a = q.data,
                            k = ht.Default.getShape3dModel(a.s("shape3d"));
                        return !k || !k.model3d
                    }, a = [];
                    this.addLabelComboBox(a, Cq("editor.batch"), function (q) {
                        return q.s("static.group")
                    }, function (q, a) {
                        return q.s("static.group", a)
                    }, ["0", "1", "2", "3", "4", "5", "6", "7"], []).setEditable(!0), this.addCheckBox(a, hteditor.getter("s", "static"), hteditor.setter("s", "static")), this.addButton(a, null, null, "editor.selectAll", function (a) {
                        if (a && a.s("static")) {
                            var k = q.editor.dm.toDatas(function (q) {
                                return !!q.s("static") && String(q.s("static.group")) === String(a.s("static.group"))
                            });
                            q.editor.dm.sm().setSelection(k)
                        }
                    }), this.addOneRow(a).visible = function (q) {
                        var a = q.data,
                            k = ht.Default.getShape3dModel(a.s("shape3d"));
                        return !!k && !!k.model3d
                    }
                }
                this.updateHandlers.push(function () {
                    if (q.data) {
                        var a = [],
                            k = q.editor.scene.getRenderLayerInfoMap() || {},
                            w = Object.values(k),
                            I = Object.keys(k);
                        w.forEach(function (q, a) {
                            q.name = I[a]
                        }), w.sort(function (q, a) {
                            return q.priority - a.priority
                        }), w.forEach(function (q) {
                            a.push(q.name)
                        }), 0 === a.length && a.push("main", "top"), j.setValues(a), L.setDisabled(!q.data.s("3d.clip.direction"));
                        var s = K(q.data);
                        C.setDisabled(s), b.setDisabled(s)
                    }
                })
            }, a.prototype.addTransformProperties = function () {
                this.addTitle("TitleTransform");
                var q = R.config.numberPrecision,
                    a = [];
                this.addLabelComboBox(a, Cq("editor.rotationMode"), function (q) {
                    return q.getRotationMode()
                }, function (q, a) {
                    return q.setRotationMode(a)
                }, ["xyz", "xzy", "yxz", "yzx", "zxy", "zyx"]), this.addRow(a, [this.indent, .1]);
                var k = q.rotation || 0;
                I = 0 === k ? "int" : "number", K = 1 / Math.pow(10, k), a = [], a.push(Cq("editor.rotation")), this.addButton(a, null, Cq("editor.reset"), "editor.resetsize.state", function (q) {
                    q instanceof ht.Node && q.r3(0, 0, 0)
                }), this.addLabelInput(a, "X", function (q) {
                    return Number((180 / Math.PI * q.getRotationX()).toFixed(k))
                }, function (q, a) {
                    q instanceof ht.Node && q.setRotationX(a * Math.PI / 180)
                }, I, K), this.addLabelInput(a, "Y", function (q) {
                    return Number((180 / Math.PI * q.getRotationY()).toFixed(k))
                }, function (q, a) {
                    q instanceof ht.Node && q.setRotationY(a * Math.PI / 180)
                }, I, K), this.addLabelInput(a, "Z", function (q) {
                    return Number((180 / Math.PI * q.getRotationZ()).toFixed(k))
                }, function (q, a) {
                    q instanceof ht.Node && q.setRotationZ(a * Math.PI / 180)
                }, I, K), this.addRow(a, [this.indent - 40 - 8, 20, 20, .1, 20, .1, 20, .1]);
                var w = q.position || 0,
                    I = 0 === w ? "int" : "number",
                    K = 1 / Math.pow(10, w);
                a = [], a.push(Cq("editor.position")), this.addButton(a, null, Cq("editor.reset"), "editor.resetsize.state", function (q) {
                    q instanceof ht.Node && q.p3(0, 0, 0)
                }), this.addLabelInput(a, "X", function (q) {
                    return Number(q.getX().toFixed(w))
                }, function (q, a) {
                    return q.setX(parseFloat(a))
                }, I, K), this.addLabelInput(a, "Y", function (q) {
                    return Number(q.getElevation().toFixed(w))
                }, function (q, a) {
                    return q.setElevation(parseFloat(a))
                }, I, K), this.addLabelInput(a, "Z", function (q) {
                    return Number(q.getY().toFixed(w))
                }, function (q, a) {
                    return q.setY(parseFloat(a))
                }, I, K), this.addRow(a, [this.indent - 40 - 8, 20, 20, .1, 20, .1, 20, .1]);
                var b = q.size || 0;
                I = 0 === b ? "int" : "number", K = 1 / Math.pow(10, b), a = [], a.push(Cq("editor.size")), this.addButton(a, null, Cq("editor.reset"), "editor.resetsize.state", function (q) {
                    q instanceof ht.Node && q.s3(-1, -1, -1)
                }), this.addLabelInput(a, "X", function (q) {
                    return Number(q.getWidth().toFixed(b))
                }, function (q, a) {
                    return q.setWidth(parseFloat(a))
                }, I, K), this.addLabelInput(a, "Y", function (q) {
                    return Number(q.getTall().toFixed(b))
                }, function (q, a) {
                    return q.setTall(parseFloat(a))
                }, I, K), this.addLabelInput(a, "Z", function (q) {
                    return Number(q.getHeight().toFixed(b))
                }, function (q, a) {
                    return q.setHeight(parseFloat(a))
                }, I, K), this.addRow(a, [this.indent - 40 - 8, 20, 20, .1, 20, .1, 20, .1]);
                var s = q.anchor || 0;
                I = 0 === s ? "int" : "number", K = 1 / Math.pow(10, s), a = [], a.push(Cq("editor.anchor")), this.addButton(a, null, Cq("editor.reset"), "editor.resetsize.state", function (q) {
                    q instanceof ht.Node && q.setAnchor3d(.5, 0, .5, !0)
                }), this.addLabelInput(a, "X", function (q) {
                    return Number(q.getAnchorX().toFixed(s))
                }, function (q, a) {
                    q instanceof ht.Node && q.setAnchor3d(parseFloat(a), q.getAnchorElevation(), q.getAnchorY(), !0)
                }, I, K), this.addLabelInput(a, "Y", function (q) {
                    return Number(q.getAnchorElevation().toFixed(s))
                }, function (q, a) {
                    q instanceof ht.Node && q.setAnchor3d(q.getAnchorX(), parseFloat(a), q.getAnchorY(), !0)
                }, I, K), this.addLabelInput(a, "Z", function (q) {
                    return Number(q.getAnchorY().toFixed(s))
                }, function (q, a) {
                    q instanceof ht.Node && q.setAnchor3d(q.getAnchorX(), q.getAnchorElevation(), parseFloat(a), !0)
                }, I, K), this.addRow(a, [this.indent - 40 - 8, 20, 20, .1, 20, .1, 20, .1]);
                var u = q.scale || 0;
                I = 0 === u ? "int" : "number", K = 1 / Math.pow(10, u), a = [], a.push(Cq("editor.scale")), this.addButton(a, null, Cq("editor.reset"), "editor.resetsize.state", function (q) {
                    q instanceof ht.Node && q.setScale3d(1, 1, 1)
                }), this.addLabelInput(a, "X", function (q) {
                    return Number(q.getScaleX().toFixed(u))
                }, function (q, a) {
                    q instanceof ht.Node && q.setScaleX(parseFloat(a))
                }, I, K), this.addLabelInput(a, "Y", function (q) {
                    return Number(q.getScaleTall().toFixed(u))
                }, function (q, a) {
                    q instanceof ht.Node && q.setScaleTall(parseFloat(a))
                }, I, K), this.addLabelInput(a, "Z", function (q) {
                    return Number(q.getScaleY().toFixed(u))
                }, function (q, a) {
                    q instanceof ht.Node && q.setScaleY(parseFloat(a))
                }, I, K), this.addRow(a, [this.indent - 40 - 8, 20, 20, .1, 20, .1, 20, .1])
            }, a.prototype.addWireframeProperties = function () {
                this.addTitle("TitleWireFrame");
                var q = [];
                this.addLabelComboBox(q, Cq("editor.visible"), hteditor.getter("s", "wf.visible"), function (q, a) {
                    q.s("wf.visible", a)
                }, [!1, !0, "selected"]), this.addLabelCheckBox(q, Cq("editor.wfshort"), hteditor.getter("s", "wf.short"), hteditor.setter("s", "wf.short")), this.addRow(q, [this.indent, .1, this.indent2, .1]), q = [], this.addLabelColor(q, Cq("editor.color"), hteditor.getter("s", "wf.color"), hteditor.setter("s", "wf.color")), this.addRow(q, [this.indent, .1]), q = [], this.addLabelCheckBox(q, Cq("editor.wfgeometry"), hteditor.getter("s", "wf.geometry"), hteditor.setter("s", "wf.geometry")), this.addLabelCheckBox(q, Cq("editor.wfloadQuadWireframe"), hteditor.getter("s", "wf.loadQuadWireframe"), hteditor.setter("s", "wf.loadQuadWireframe")), this.addRow(q, [this.indent, .1, this.indent2, .1]), q = [], this.addLabelComboBox(q, Cq("editor.wfcombineTriangle"), hteditor.getter("s", "wf.combineTriangle"), hteditor.setter("s", "wf.combineTriangle"), [!1, !0, 2, 3], [Cq("editor.wfdontcombine"), Cq("editor.wfcombineadjacent"), Cq("editor.wfcombinecommon"), Cq("editor.wfcombinesmooth")]), this.addRow(q, [this.indent, .1])
            }, a.prototype.addHighlightProperties = function () {
                var q = function (q) {
                    return "style" === q.editor.dm.a("sceneHighlightMode")
                };
                this.addTitle("TitleHighlight");
                var a = [];
                this.addLabelComboBox(a, Cq("editor.type"), hteditor.getter("s", "highlight.mode"), hteditor.setter("s", "highlight.mode"), [!1, !0, "selected", "hover"], [Cq("editor.no"), Cq("editor.yes"), Cq("editor.selected"), Cq("editor.hover")]), this.addRow(a, [this.indent, .1]).visible = q
            }, a.prototype.addReflectorProperties = function () {
                this.addTitle("TitleReflector");
                var q = [];
                this.addLabelCheckBox(q, Cq("editor.reflectable"), hteditor.getter("s", "3d.reflectable"), hteditor.setter("s", "3d.reflectable")), this.addRow(q, [this.indent, .1])
            }, a.prototype.addEnvmapProperties = function () {
                var q = this,
                    a = function (q) {
                        return "new" === q.editor.dm.getEnvmapType()
                    };
                this.addTitle("TitleEnvmap");
                var k = [];
                this.addLabelRange(k, Cq("editor.envmap.reflectivity"), function (q) {
                    return q.s("envmap") || 0
                }, function (q, a) {
                    return q.s("envmap", a)
                }, 0, 1, .01, "number"), this.addLabelRange(k, Cq("editor.headlight.ambientintensity"), function (q) {
                    return q.s("headlight.ambientIntensity") || .7
                }, function (q, a) {
                    return q.s("headlight.ambientIntensity", a)
                }, void 0, void 0, .01, "number"), this.addRow(k, [this.indent, .1, this.indent2, .1]), k = [], this.addLabelRange(k, Cq("editor.envmap.roughness"), function (q) {
                    return q.s("roughness")
                }, function (q, a) {
                    return q.s("roughness", a)
                }, 0, 1, .01, "number");
                var w = this.addLabelComboBox(k, Cq("editor.envmap.probe"), hteditor.getter("s", "envmap.probe"), hteditor.setter("s", "envmap.probe"), [], []);
                this.addRow(k, [this.indent, .1, this.indent2, .1]).visible = a, w.getLabels = function () {
                    var a = q.editor.dm.getEnvmap();
                    if (a instanceof Array) {
                        var k = [];
                        return a.forEach(function (q, a) {
                            var w = q.type;
                            void 0 !== q.name && (w = q.name), k.push(a + " - " + w)
                        }), k
                    }
                }, w.getValues = function () {
                    var a = q.editor.dm.getEnvmap();
                    if (a instanceof Array) {
                        var k = [];
                        return a.forEach(function (q, a) {
                            k.push(a)
                        }), k
                    }
                }
            }, a.prototype.addClipboxProperties = function () {
                var q = this;
                this.addTitle("TitleClipBox");
                var a = [];
                this.addLabelComboBox(a, Cq("editor.clipbox"), hteditor.getter("s", "3d.clipbox"), function (a, k) {
                    a.s("3d.clipbox", k), q.editor.inspector.filterPropertiesLater()
                }, [void 0, "inner", "outer"], ["", Cq("editor.clipbox.inner"), Cq("editor.clipbox.outer")]), this.addLabelComboBox(a, Cq("editor.clipboxshape"), hteditor.getter("s", "3d.clipbox.shape"), hteditor.setter("s", "3d.clipbox.shape"), ["cube", "sphere"], [Cq("editor.cube"), Cq("editor.sphere")]), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [];
                var k = this.addLabelComboBox(a, Cq("editor.group"), hteditor.getter("s", "3d.clipbox.group"), hteditor.setter("s", "3d.clipbox.group"), [0], [Cq("editor.default")]);
                this.addRow(a, [this.indent, .1, this.indent2, .1]).visible = function (q) {
                    return void 0 !== q.data.s("3d.clipbox")
                }, k.getLabels = function () {
                    var a = q.editor.dm;
                    if (!a) return [Cq("editor.default")];
                    var k = a.a("sceneClipboxGroupAlias");
                    if (!k) return [Cq("editor.default")];
                    var w = ht.Default.clone(k);
                    return [Cq("editor.default")].concat(Object.values(w))
                }, k.getValues = function () {
                    var a = q.editor.dm;
                    if (!a) return [Cq("editor.default")];
                    var k = a.a("sceneClipboxGroupAlias");
                    if (!k) return [0];
                    var w = ht.Default.clone(k),
                        I = [0];
                    return Object.keys(w).forEach(function (q) {
                        I.push(Number(q))
                    }), I
                }, a = [];
                var w = function (q) {
                    var a = this;
                    w.superClass.constructor.call(a, q);
                    var k = a._listView = new ht.widget.ListView;
                    k.getView().style.background = "white", k.setCheckMode(!0), k.sm().ms(function (w) {
                        var I = k.dm().getDataByTag(4294967295);
                        if ("append" === w.kind)
                            if (w.datas.get(0) === I) k.sm().selectAll();
                            else {
                                var K = k.sm().getSelection().length;
                                K === k.dm().getDatas().length - 1 && k.sm().selectAll()
                            }
                        else if ("remove" === w.kind)
                            if (w.datas.get(0) === I) k.sm().clearSelection();
                            else {
                                var b = k.sm().getSelection().toList(function (q) {
                                    return q !== I
                                });
                                k.sm().setSelection(b)
                            } q.setValue(a.getValue())
                    })
                };
                ht.Default.def(w, ht.widget.BaseDropDownTemplate, {
                    getView: function () {
                        return this._listView.getView()
                    },
                    onOpened: function (q) {
                        var a = this._listView,
                            k = new ht.Data;
                        k.setName(Cq("editor.all")), k.setTag(4294967295), a.dm().add(k);
                        var w = new ht.Data;
                        w.setName(Cq("editor.default")), w.setTag(0), a.dm().add(w);
                        var I = this._master.getMaskMap();
                        q = this._master.getValue();
                        for (var K in I) {
                            K = parseInt(K);
                            var b = new ht.Data;
                            b.setName(I[K]), b.setTag(K), a.dm().add(b)
                        }
                        if (q)
                            if (4294967295 === q) a.sm().selectAll();
                            else {
                                var s = new ht.List;
                                a.dm().each(function (a) {
                                    q & 1 << a.getTag() && s.add(a)
                                }), a.sm().setSelection(s)
                            }
                    },
                    onClosed: function () {},
                    getValue: function () {
                        var q = this._listView.dm(),
                            a = q.sm();
                        if (0 === a.size()) return 0;
                        var k = q.getDataByTag(4294967295);
                        if (a.contains(k)) return 4294967295;
                        var w = 0;
                        return a.each(function (q) {
                            var a = q.getTag();
                            4294967295 !== a && q.getName() && (w += 1 << a)
                        }), w
                    },
                    getHeight: function () {
                        return 100
                    }
                });
                var I = new ht.widget.MultiComboBox;
                I.setEditable(!1), I.setDropDownComponent(w), I.drawValue = function (q, a, k, w, K) {
                    var b = I.getValue(),
                        s = I.getMaskMap(),
                        u = [];
                    if (4294967295 === b) u = Cq("editor.all");
                    else if (0 === b) u = "";
                    else {
                        1 & b && u.push(Cq("editor.default"));
                        for (var C in s) C = parseInt(C), b & 1 << C && u.push(s[C]);
                        u.join(", ")
                    }
                    ht.Default.drawText(q, u, I.getLabelFont(), I.getLabelColor(), a + 1, k, 0, K)
                }, I.getMaskMap = function () {
                    return q.editor.dm.a("sceneClipboxGroupAlias")
                }, I.getValue = function () {
                    return q.data.s("3d.clipbox.mask")
                }, I.onValueChanged = function (a, k) {
                    q.editor.sm.getSelection().each(function (q) {
                        q.s("3d.clipbox.mask", k)
                    })
                }, a.push(Cq("editor.clipboxmask")), a.push(I), this.addRow(a, [this.indent, .1]).visible = function (q) {
                    return void 0 === q.data.s("3d.clipbox")
                }, this.updateHandlers.push(function () {
                    q.data
                })
            }, a.prototype.addBloomProperties = function () {
                var q = function (q) {
                    return q.editor.dm.a("sceneBloomSelective")
                };
                this.addTitle("TitleBloom");
                var a = [];
                this.addLabelCheckBox(a, Cq("editor.enable"), hteditor.getter("s", "bloom"), hteditor.setter("s", "bloom")), this.addRow(a, [this.indent, .1]).visible = q
            }, a.prototype.$19$ = function () {
                this.addTitle("TitleShadow");
                var q = [];
                this.addLabelCheckBox(q, Cq("editor.shadow.cast"), hteditor.getter("s", "shadow.cast"), hteditor.setter("s", "shadow.cast")), this.addLabelCheckBox(q, Cq("editor.shadow.receive"), hteditor.getter("s", "shadow.receive"), hteditor.setter("s", "shadow.receive")), this.addRow(q, [this.indent, .1, this.indent2, .1])
            }, a
        }(Kq),
        Gq = hteditor.getString,
        Xq = ["all", "front", "back", "left", "right", "top", "bottom"],
        Zq = function (q, a, k) {
            return function (w) {
                var I = w.s(q + "." + a);
                return k && void 0 === I ? w.s("all." + a) : I
            }
        },
        rq = function (q) {
            return function (a) {
                var k = a.s(q);
                return k || (k = a.s("all.uv.scale")), k ? k[0] : 1
            }
        },
        Wq = function (q) {
            return function (a, k) {
                var w = a.s(q);
                w || (w = a.s("all.uv.scale")), a.s(q, [k, w ? w[1] : 1])
            }
        },
        Tq = function (q) {
            return function (a) {
                var k = a.s(q);
                return k || (k = a.s("all.uv.scale")), k ? k[1] : 1
            }
        },
        pq = function (q) {
            return function (a, k) {
                var w = a.s(q);
                w || (w = a.s("all.uv.scale")), a.s(q, [w ? w[0] : 1, k])
            }
        },
        oq = function (q) {
            return function (a) {
                a.s(q, null)
            }
        },
        gq = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "Cube"))
            }
            return E(a, q), a.prototype.initForm = function () {
                this.addCustomProperties(), this.$11$(), this.addTransformProperties(), this.addCubeImageVisibleLightProperties(), this.addCubeColoruvScaleProperties(), this.addCubeBlendUvOffsetProperties(), this.addCubeUvAnchorRotationProperties(), this.addCubeReverseColorFlipCullProperties(), this.addCubeOpacityTransparentSelectProperties(), this.addWireframeProperties(), this.addBloomProperties(), this.addHighlightProperties(), this.addReflectorProperties(), this.addEnvmapProperties(), this.addClipboxProperties(), this.editor.scene.shadowMap && this.$19$()
            }, a.prototype.addCubeImageVisibleLightProperties = function () {
                var q = this;
                this.addTitle("TitleCubeImageVisibleLight");
                for (var a = [], k = 0; k < Xq.length; k++) ! function (k) {
                    var w = Xq[k] + ".image",
                        I = Xq[k] + ".material";
                    a = [], q.addLabelImageOrMaterial(a, Gq("editor.face." + Xq[k]), function (a) {
                        var w = Zq(Xq[k], "material", !0)(a),
                            I = q.editor.getFileNode(w);
                        if (w) return I ? I.url : K;
                        var K = Zq(Xq[k], "image", !0)(a),
                            b = q.editor.getFileNode(K);
                        return b ? b.url : K
                    }, function (a, k) {
                        var K = q.editor.getFileNode(k);
                        K && "material" === K.fileType ? (a.s(I, k), a.s(w, void 0)) : (a.s(w, k), a.s(I, void 0), a.s("texture.cache") && q.editor.scene.invalidateShape3dCachedImage(a)), q.filterPropertiesLater()
                    });
                    var K = Xq[k] + ".visible";
                    q.addLabelCheckBox(a, Gq("editor.visible"), Zq(Xq[k], "visible", !0), hteditor.setter("s", K));
                    var b = Xq[k] + ".light";
                    q.addLabelCheckBox(a, Gq("editor.light"), Zq(Xq[k], "light", !0), hteditor.setter("s", b)), q.addRow(a, [40, .1, 20, 40, 20, 40, 20])
                }(k)
            }, a.prototype.addCubeUvProperties = function () {}, a.prototype.addCubeColoruvScaleProperties = function () {
                var q = this;
                this.addTitle("TitleCubeColorUvScale");
                for (var a = [], k = 0; k < Xq.length; k++) ! function (k) {
                    var w = Xq[k] + ".color";
                    a = [], q.addLabelColor(a, Gq("editor.face." + Xq[k]), Zq(Xq[k], "color"), hteditor.setter("s", w));
                    var I = Xq[k] + ".uv.scale";
                    q.addLabelInput(a, "U", rq(I), Wq(I), "number", 1), q.addLabelInput(a, "V", Tq(I), pq(I), "number", 1), q.addRow(a, [40, .1, 20, .05, 20, .05]).visible = function () {
                        return !Zq(Xq[k], "material", !0)(q.data)
                    }
                }(k)
            }, a.prototype.addCubeBlendUvOffsetProperties = function () {
                var q = this;
                this.addTitle("TitleCubeBlendUvOffset");
                for (var a = [], k = 0; k < Xq.length; k++) ! function (k) {
                    var w = Xq[k] + ".blend";
                    a = [], q.addLabelColor(a, Gq("editor.face." + Xq[k]), Zq(Xq[k], "blend"), hteditor.setter("s", w));
                    var I = Xq[k] + ".uv.offset";
                    q.addLabelInput(a, "U", function (q) {
                        var a = q.s(I);
                        return a || (a = q.s("all.uv.offset")), a ? a[0] : 0
                    }, function (q, a) {
                        var k = q.s(I);
                        k || (k = q.s("all.uv.offset")), q.s(I, [a, k ? k[1] : 0])
                    }, "number", .1), q.addLabelInput(a, "V", function (q) {
                        var a = q.s(I);
                        return a || (a = q.s("all.uv.offset")), a ? a[1] : 0
                    }, function (q, a) {
                        var k = q.s(I);
                        k || (k = q.s("all.uv.offset")), q.s(I, [k ? k[0] : 0, a])
                    }, "number", .1), q.addRow(a, [40, .1, 20, .05, 20, .05]).visible = function () {
                        return !Zq(Xq[k], "material", !0)(q.data)
                    }
                }(k)
            }, a.prototype.addCubeUvAnchorRotationProperties = function () {
                var q = this;
                this.addTitle("TitleCubeUvAnchorRotation");
                for (var a = [], k = 0; k < Xq.length; k++) ! function (k) {
                    a = [], q.addLabel(a, Gq("editor.face." + Xq[k]));
                    var w = Xq[k] + ".uv.anchor";
                    q.addLabelInput(a, "U", function (q) {
                        var a = q.s(w);
                        return a || (a = q.s("all.uv.anchor")), a ? a[0] : 0
                    }, function (q, a) {
                        var k = q.s(w);
                        k || (k = q.s("all.uv.anchor")), q.s(w, [a, k ? k[1] : 0])
                    }, "number", .01), q.addLabelInput(a, "V", function (q) {
                        var a = q.s(w);
                        return a || (a = q.s("all.uv.anchor")), a ? a[1] : 0
                    }, function (q, a) {
                        var k = q.s(w);
                        k || (k = q.s("all.uv.anchor")), q.s(w, [k ? k[0] : 0, a])
                    }, "number", .01);
                    var I = Xq[k] + ".uv.rotation";
                    q.addLabelInput(a, "R", hteditor.getter("s", I), hteditor.setter("s", I), "number", .1), q.addRow(a, [40, 20, .1, 20, .1, 20, .1]).visible = function () {
                        return !Zq(Xq[k], "material", !0)(q.data)
                    }
                }(k)
            }, a.prototype.addCubeReverseColorFlipCullProperties = function () {
                var q = this;
                this.addTitle("TitleCubeReverseColorFlipCull");
                for (var a = [], k = 0; k < Xq.length; k++) ! function (k) {
                    var w = Xq[k] + ".reverse.color";
                    a = [], q.addLabelColor(a, Gq("editor.face." + Xq[k]), Zq(Xq[k], "reverse.color"), hteditor.setter("s", w));
                    var I = Xq[k] + ".reverse.flip";
                    q.addLabelCheckBox(a, Gq("editor.flip"), Zq(Xq[k], "reverse.flip", !0), hteditor.setter("s", I));
                    var K = Xq[k] + ".reverse.cull";
                    q.addLabelCheckBox(a, Gq("editor.cull"), Zq(Xq[k], "reverse.cull", !0), hteditor.setter("s", K)), q.addRow(a, [40, .1, 40, 20, 40, 20]).visible = function () {
                        return !Zq(Xq[k], "material", !0)(q.data)
                    }
                }(k)
            }, a.prototype.addCubeOpacityTransparentSelectProperties = function () {
                var q = this;
                this.addTitle("titleCubeOpacityTransparentSelect");
                for (var a = [], k = 0; k < Xq.length; k++) ! function (k) {
                    var w = Xq[k] + ".opacity";
                    a = [], q.addLabelRange(a, Gq("editor.face." + Xq[k]), hteditor.getter("s", w), hteditor.setter("s", w), 0, 1, .01, "number");
                    var I = Xq[k] + ".transparent";
                    q.addLabelCheckBox(a, Gq("editor.transparent"), Zq(Xq[k], "transparent", !0), hteditor.setter("s", I));
                    var K = Xq[k] + ".discard.selectable";
                    q.addLabelCheckBox(a, Gq("editor.discardselectable"), Zq(Xq[k], "discard.selectable", !0), hteditor.setter("s", K)), q.addRow(a, [40, .1, 40, 20, 80, 20]).visible = function () {
                        return !Zq(Xq[k], "material", !0)(q.data)
                    }
                }(k)
            }, a.prototype.addCubeProperties = function () {
                var q = this;
                this.addTitle("TitleFace");
                var a = void 0,
                    k = void 0,
                    w = 0,
                    I = new ht.widget.TabView;
                k = new ht.widget.FormPane;
                for (var K = 0; K < Xq.length; K++) ! function () {
                    var w = Xq[K] + ".image";
                    a = [];
                    var I = Zq(Xq[K], "image");
                    q.addLabelImage(a, Gq("editor.face." + Xq[K]), function (a) {
                        var k = I(a),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, hteditor.setter("s", w)), k.addRow(a, [q.indent, .1, 20])
                }();
                W(I, Gq("editor.image"), k, !0), k.validateImpl(), w = Math.max(w, k.getScrollHeight()), k = new ht.widget.FormPane;
                for (var K = 0; K < Xq.length; K++) {
                    var b = Xq[K] + ".uv.scale";
                    a = [], this.addLabel(a, Gq("editor.face." + Xq[K])), this.addButton(a, null, Gq("editor.reset"), "editor.resetsize.state", oq(b)), this.addLabelInput(a, "U", rq(b), Wq(b), "number", 1), this.addLabelInput(a, "V", Tq(b), pq(b), "number", 1), k.addRow(a, [this.indent - 20 - 8, 20, 20, .1, 20, .1])
                }
                W(I, Gq("editor.repeat"), k, !0), k.validateImpl(), w = Math.max(w, k.getScrollHeight()), k = new ht.widget.FormPane;
                for (var K = 0; K < Xq.length; K++) {
                    var s = Xq[K] + ".color";
                    a = [], this.addLabelColor(a, Gq("editor.face." + Xq[K]), Zq(Xq[K], "color"), hteditor.setter("s", s)), k.addRow(a, [this.indent, .1])
                }
                W(I, Gq("editor.color"), k), k.validateImpl(), w = Math.max(w, k.getScrollHeight()), k = new ht.widget.FormPane;
                for (var K = 0; K < Xq.length; K++) {
                    var u = Xq[K] + ".transparent";
                    a = [], this.addLabelCheckBox(a, Gq("editor.face." + Xq[K]), Zq(Xq[K], "transparent"), hteditor.setter("s", u)), k.addRow(a, [this.indent, .1])
                }
                W(I, Gq("editor.transparent"), k), k.validateImpl(), w = Math.max(w, k.getScrollHeight()), k = new ht.widget.FormPane;
                for (var K = 0; K < Xq.length; K++) {
                    var C = Xq[K] + ".visible";
                    a = [], this.addLabelCheckBox(a, Gq("editor.face." + Xq[K]), Zq(Xq[K], "visible", !0), hteditor.setter("s", C)), k.addRow(a, [this.indent, .1])
                }
                W(I, Gq("editor.visible"), k), k.validateImpl(), w = Math.max(w, k.getScrollHeight()), k = new ht.widget.FormPane;
                for (var K = 0; K < Xq.length; K++) {
                    var L = Xq[K] + ".reverse.color";
                    a = [], this.addLabelColor(a, Gq("editor.face." + Xq[K]), Zq(Xq[K], "reverse.color"), hteditor.setter("s", L)), k.addRow(a, [this.indent, .1])
                }
                W(I, Gq("editor.reversecolor"), k), k.validateImpl(), w = Math.max(w, k.getScrollHeight()), k = new ht.widget.FormPane;
                for (var K = 0; K < Xq.length; K++) {
                    var j = Xq[K] + ".reverse.flip";
                    a = [], this.addLabelCheckBox(a, Gq("editor.face." + Xq[K]), Zq(Xq[K], "reverse.flip", !0), hteditor.setter("s", j)), k.addRow(a, [this.indent, .1])
                }
                W(I, Gq("editor.reverseflip"), k), k.validateImpl(), w = Math.max(w, k.getScrollHeight()), k = new ht.widget.FormPane;
                for (var K = 0; K < Xq.length; K++) {
                    var G = Xq[K] + ".reverse.cull";
                    a = [], this.addLabelCheckBox(a, Gq("editor.face." + Xq[K]), Zq(Xq[K], "reverse.cull", !0), hteditor.setter("s", G)), k.addRow(a, [this.indent, .1])
                }
                W(I, Gq("editor.reversecull"), k), k.validateImpl(), w = Math.max(w, k.getScrollHeight());
                var X = I.getTabHeight() + w;
                a = [], a.push(I), this.addRow(a, [.1], X)
            }, a
        }(jq),
        Rq = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "Cone"))
            }
            return E(a, q), a
        }(jq),
        Hq = hteditor.getString,
        fq = ["shape3d", "shape3d.top", "shape3d.bottom", "shape3d.from", "shape3d.to"],
        nq = ["shape3d", "shape3d.top", "shape3d.bottom"],
        yq = function (q) {
            return function (a) {
                var k = a.s(q);
                return k ? k[0] : 1
            }
        },
        Mq = function (q) {
            return function (a, k) {
                var w = a.s(q);
                a.s(q, [k, w ? w[1] : 1])
            }
        },
        mq = function (q) {
            return function (a) {
                var k = a.s(q);
                return k ? k[1] : 1
            }
        },
        xq = function (q) {
            return function (a, k) {
                var w = a.s(q);
                a.s(q, [w ? w[0] : 1, k])
            }
        },
        zq = function (q) {
            return function (a) {
                a.s(q, null)
            }
        },
        Vq = function (q) {
            function a(k, w) {
                return V(this, a), Q(this, q.call(this, k, w || "Shape3d"))
            }
            return E(a, q), a.prototype.initForm = function () {
                q.prototype.initForm.call(this)
            }, a.prototype.getCurrentPoint = function (q) {
                return this.editor.currentPoint ? this.editor.currentPoint[q] : 0
            }, a.prototype.setCurrentPoint = function (q, a) {
                this.editor.gv.getEditInteractor().getSubModule("Curve").setCurrentPoint(q, a)
            }, a.prototype.addTopBodyBottomProperties = function () {
                var q = this;
                this.addTitle("TitleFace");
                var a = void 0,
                    k = void 0,
                    w = 0,
                    I = new ht.widget.TabView,
                    K = this.faces;
                k = new ht.widget.FormPane;
                for (var b = 0; b < K.length; b++) ! function () {
                    var w = K[b] + ".image",
                        I = K[b] + ".material";
                    a = [], q.addLabelImageOrMaterial(a, Hq("editor.face." + K[b]), function (a) {
                        var k = faceGetter(K[b], "material")(a),
                            w = q.editor.getFileNode(k);
                        if (k) return w ? w.url : I;
                        var I = faceGetter(K[b], "image")(a),
                            s = q.editor.getFileNode(I);
                        return s ? s.url : I
                    }, function (a, k) {
                        var K = q.editor.getFileNode(k);
                        K && "material" === K.fileType ? (a.s(I, k), a.s(w, void 0)) : (a.s(w, k), a.s(I, void 0), a.s("texture.cache") && q.editor.scene.invalidateShape3dCachedImage(a))
                    }), k.addRow(a, [q.indent, .1, 20])
                }();
                W(I, Hq("editor.image"), k, !0), k.validateImpl(), w = Math.max(w, k.getScrollHeight()), k = new ht.widget.FormPane;
                for (var b = 0; b < K.length; b++) {
                    var s = K[b] + ".uv.scale";
                    a = [], this.addLabel(a, Hq("editor.face." + K[b])), this.addButton(a, null, Hq("editor.reset"), "editor.resetsize.state", zq(s)), this.addLabelInput(a, "U", yq(s), Mq(s), "number", 1), this.addLabelInput(a, "V", mq(s), xq(s), "number", 1), k.addRow(a, [this.indent - 20 - 8, 20, 20, .1, 20, .1])
                }
                W(I, Hq("editor.repeat"), k, !0), k.validateImpl(), w = Math.max(w, k.getScrollHeight()), k = new ht.widget.FormPane;
                for (var b = 0; b < K.length; b++) {
                    var u = K[b] + ".color";
                    a = [], this.addLabelColor(a, Hq("editor.face." + K[b]), hteditor.getter("s", u), hteditor.setter("s", u)), k.addRow(a, [this.indent, .1, 20])
                }
                W(I, Hq("editor.color"), k), k.validateImpl(), w = Math.max(w, k.getScrollHeight()), k = new ht.widget.FormPane;
                for (var b = 0; b < K.length; b++) {
                    var C = K[b] + ".visible";
                    a = [], this.addLabelCheckBox(a, Hq("editor.face." + K[b]), hteditor.getter("s", C), hteditor.setter("s", C)), k.addRow(a, [this.indent, .1, 20])
                }
                W(I, Hq("editor.visible"), k), k.validateImpl(), w = Math.max(w, k.getScrollHeight()), k = new ht.widget.FormPane, a = [], this.addLabelColor(a, Hq("editor.color"), hteditor.getter("s", "shape3d.reverse.color"), hteditor.setter("s", "shape3d.reverse.color")), k.addRow(a, [this.indent, .1]), a = [], this.addLabelCheckBox(a, Hq("editor.flip"), hteditor.getter("s", "shape3d.reverse.flip"), hteditor.setter("s", "shape3d.reverse.flip")), k.addRow(a, [this.indent, .1]), a = [], this.addLabelCheckBox(a, Hq("editor.cull"), hteditor.getter("s", "shape3d.reverse.cull"), hteditor.setter("s", "shape3d.reverse.cull")), k.addRow(a, [this.indent, .1]), W(I, Hq("editor.reverse"), k), k.validateImpl(), w = Math.max(w, k.getScrollHeight());
                var L = I.getTabHeight() + w;
                a = [], a.push(I), this.addRow(a, [.1], L)
            }, a.prototype.addEffectProperties = function () {
                this.addTitle("TitleEffect");
                var q = [];
                this.addLabelCheckBox(q, Hq("editor.transparent"), hteditor.getter("s", "shape3d.transparent"), hteditor.setter("s", "shape3d.transparent")), this.addRow(q, [this.indent, .1]), q = [], this.addLabelColor(q, Hq("editor.blend"), hteditor.getter("s", "shape3d.blend"), hteditor.setter("s", "shape3d.blend")), this.addRow(q, [this.indent, .1]), q = [], this.addLabelRange(q, Hq("editor.opacity"), hteditor.getter("s", "shape3d.opacity"), hteditor.setter("s", "shape3d.opacity"), 0, 1, .01, "number"), this.addRow(q, [this.indent, .1]), q = [], this.addLabelRange(q, Hq("editor.side"), hteditor.getter("s", "shape3d.side"), hteditor.setter("s", "shape3d.side"), 0, void 0, 1, "int"), this.addRow(q, [this.indent, .1]), this.simple || (q = [], this.addLabelRange(q, Hq("editor.side.from"), hteditor.getter("s", "shape3d.side.from"), hteditor.setter("s", "shape3d.side.from"), 0, void 0, 1, "int"), this.addRow(q, [this.indent, .1]), q = [], this.addLabelRange(q, Hq("editor.side.to"), hteditor.getter("s", "shape3d.side.to"), hteditor.setter("s", "shape3d.side.to"), 0, void 0, 1, "int"), this.addRow(q, [this.indent, .1]))
            }, a.prototype.addExtraProperties = function (q) {}, v(a, [{
                key: "simple",
                get: function () {
                    return !1
                }
            }, {
                key: "faces",
                get: function () {
                    return this.simple ? nq : fq
                }
            }]), a
        }(jq),
        vq = hteditor.getString,
        Eq = ["shape3d", "shape3d.top", "shape3d.bottom", "shape3d.from", "shape3d.to"],
        Qq = function (q, a, k) {
            return function (w) {
                var I = w.s(q + "." + a);
                return k && void 0 === I ? w.s("shape3d." + a) : I
            }
        },
        lq = function (q) {
            function a(k, w) {
                return V(this, a), Q(this, q.call(this, k, w || "Cylinder"))
            }
            return E(a, q), a.prototype.$11$ = function () {
                q.prototype.$11$.call(this), this.addEffectProperties()
            }, a.prototype.addEffectProperties = function () {
                var q = [];
                this.addLabelRange(q, vq("editor.side"), hteditor.getter("s", "shape3d.side"), hteditor.setter("s", "shape3d.side"), 0, void 0, 1, "int"), this.addLabelCheckBox(q, vq("editor.smooth"), hteditor.getter("s", "shape3d.smooth"), hteditor.setter("s", "shape3d.smooth")), this.addRow(q, [this.indent, .1, this.indent2, 20]), q = [], this.addLabelRange(q, vq("editor.side.from"), hteditor.getter("s", "shape3d.side.from"), hteditor.setter("s", "shape3d.side.from"), 0, void 0, 1, "int"), this.addLabelRange(q, vq("editor.side.to"), hteditor.getter("s", "shape3d.side.to"), hteditor.setter("s", "shape3d.side.to"), 0, void 0, 1, "int"), this.addRow(q, [this.indent, .1, this.indent2, .1])
            }, a.prototype.addTransformProperties = function () {
                q.prototype.addTransformProperties.call(this), this.addShape3dImageVisibleProperties(), this.addShape3dColoruvScaleProperties(), this.addShape3dUvOffsetSelectProperties(), this.addShape3dUvAnchorRotationProperties(), this.addShape3dReverseColorFlipCullProperties()
            }, a.prototype.addShape3dImageVisibleProperties = function () {
                var q = this;
                this.addTitle("TitleShape3dImageVisible");
                for (var a = this.faces, k = [], w = 0; w < a.length; w++) ! function (w) {
                    var I = a[w] + ".image",
                        K = a[w] + ".material";
                    k = [], q.addLabelImageOrMaterial(k, vq("editor.face." + a[w]), function (k) {
                        var I = Qq(a[w], "material")(k),
                            K = q.editor.getFileNode(I);
                        if (I) return K ? K.url : b;
                        var b = Qq(a[w], "image")(k),
                            s = q.editor.getFileNode(b);
                        return s ? s.url : b
                    }, function (a, k) {
                        var w = q.editor.getFileNode(k);
                        w && "material" === w.fileType ? (a.s(K, k), a.s(I, void 0)) : (a.s(I, k), a.s(K, void 0), a.s("texture.cache") && q.editor.scene.invalidateShape3dCachedImage(a)), q.filterPropertiesLater()
                    });
                    var b = a[w] + ".visible";
                    q.addLabelCheckBox(k, vq("editor.visible"), Qq(a[w], "visible", !0), hteditor.setter("s", b)), q.addRow(k, [40, .1, 20, 40, 20])
                }(w)
            }, a.prototype.addShape3dColoruvScaleProperties = function () {
                var q = this;
                this.addTitle("TitleShape3dColoruvScale");
                for (var a = this.faces, k = [], w = 0; w < a.length; w++) ! function (w) {
                    var I = a[w] + ".color";
                    k = [], q.addLabelColor(k, vq("editor.face." + a[w]), Qq(a[w], "color"), hteditor.setter("s", I));
                    var K = a[w] + ".uv.scale";
                    q.addLabelInput(k, "U", function (q) {
                        var a = q.s(K);
                        return a || (a = q.s("shape3d.scale")), a ? a[0] : 1
                    }, function (q, a) {
                        var k = q.s(K);
                        k || (k = q.s("shape3d.scale")), q.s(K, [a, k ? k[1] : 1])
                    }, "number", 1), q.addLabelInput(k, "V", function (q) {
                        var a = q.s(K);
                        return a || (a = q.s("shape3d.scale")), a ? a[1] : 1
                    }, function (q, a) {
                        var k = q.s(K);
                        k || (k = q.s("shape3d.scale")), q.s(K, [k ? k[0] : 1, a])
                    }, "number", 1), q.addRow(k, [40, .1, 20, .05, 20, .05]).visible = function () {
                        return !Qq(a[w], "material")(q.data)
                    }
                }(w)
            }, a.prototype.addShape3dUvOffsetSelectProperties = function () {
                var q = this;
                this.addTitle("TitleShape3dUvOffsetSelect");
                for (var a = this.faces, k = [], w = 0; w < a.length; w++) ! function (w) {
                    var I = a[w] + ".uv.offset";
                    k = [], q.addLabel(k, vq("editor.face." + a[w])), q.addLabelInput(k, "U", function (q) {
                        var a = q.s(I);
                        return a || (a = q.s("shape3d.offset")), a ? a[0] : 0
                    }, function (q, a) {
                        var k = q.s(I);
                        k || (k = q.s("shape3d.offset")), q.s(I, [a, k ? k[1] : 0])
                    }, "number", .1), q.addLabelInput(k, "V", function (q) {
                        var a = q.s(I);
                        return a || (a = q.s("shape3d.offset")), a ? a[1] : 0
                    }, function (q, a) {
                        var k = q.s(I);
                        k || (k = q.s("shape3d.offset")), q.s(I, [k ? k[0] : 0, a])
                    }, "number", .1);
                    var K = a[w] + ".discard.selectable";
                    q.addLabelCheckBox(k, vq("editor.discardselectable"), Qq(a[w], "discard.selectable", !0), hteditor.setter("s", K)), q.addRow(k, [40, 20, .1, 20, .1, q.indent, 20]).visible = function () {
                        return !Qq(a[w], "material")(q.data)
                    }
                }(w)
            }, a.prototype.addShape3dUvAnchorRotationProperties = function () {
                var q = this;
                this.addTitle("TitleShape3dUvAnchorRotation");
                for (var a = this.faces, k = [], w = 0; w < a.length; w++) ! function (w) {
                    k = [], q.addLabel(k, vq("editor.face." + a[w]));
                    var I = a[w] + ".uv.anchor";
                    q.addLabelInput(k, "U", function (q) {
                        var a = q.s(I);
                        return a || (a = q.s("shape3d.anchor")), a ? a[0] : 0
                    }, function (q, a) {
                        var k = q.s(I);
                        k || (k = q.s("shape3d.anchor")), q.s(I, [a, k ? k[1] : 0])
                    }, "number", .01), q.addLabelInput(k, "V", function (q) {
                        var a = q.s(I);
                        return a || (a = q.s("shape3d.anchor")), a ? a[1] : 0
                    }, function (q, a) {
                        var k = q.s(I);
                        k || (k = q.s("shape3d.anchor")), q.s(I, [k ? k[0] : 0, a])
                    }, "number", .01);
                    var K = a[w] + ".uv.rotation";
                    q.addLabelInput(k, "R", hteditor.getter("s", K), hteditor.setter("s", K), "number", .1), q.addRow(k, [40, 20, .1, 20, .1, 20, .1]).visible = function () {
                        return !Qq(a[w], "material")(q.data)
                    }
                }(w)
            }, a.prototype.addShape3dReverseColorFlipCullProperties = function () {
                var q = this;
                this.addTitle("TitleShape3dReverseColorFlipCull");
                var a = (this.faces, []);
                a = [], this.addLabelColor(a, vq("editor.face.shape3d"), hteditor.getter("s", "shape3d.reverse.color"), hteditor.setter("s", "shape3d.reverse.color"));
                this.addLabelCheckBox(a, vq("editor.flip"), hteditor.getter("s", "shape3d.reverse.flip"), hteditor.setter("s", "shape3d.reverse.flip"));
                this.addLabelCheckBox(a, vq("editor.cull"), hteditor.getter("s", "shape3d.reverse.cull"), hteditor.setter("s", "shape3d.reverse.cull")), this.addRow(a, [40, .1, 40, 20, 40, 20]).visible = function () {
                    return !Qq("shape3d", "material")(q.data)
                }
            }, v(a, [{
                key: "faces",
                get: function () {
                    return Eq
                }
            }]), a
        }(Vq),
        _q = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "RoundRect"))
            }
            return E(a, q), a
        }(jq),
        Fq = hteditor.getString,
        Sq = ["shape3d", "shape3d.from", "shape3d.to"],
        dq = function (q) {
            function a(k, w) {
                return V(this, a), Q(this, q.call(this, k, w || "Sphere"))
            }
            return E(a, q), a.prototype.addEffectProperties = function () {
                var q = [];
                this.addLabelRange(q, Fq("editor.side"), hteditor.getter("s", "shape3d.side"), hteditor.setter("s", "shape3d.side"), 0, void 0, 1, "int"), this.addLabelRange(q, Fq("editor.resolution"), hteditor.getter("s", "shape3d.resolution"), hteditor.setter("s", "shape3d.resolution"), 0, void 0, 1, "int"), this.addLabelCheckBox(q, Fq("editor.smooth"), hteditor.getter("s", "shape3d.smooth"), hteditor.setter("s", "shape3d.smooth")), this.addRow(q, [this.indent, .1, this.indent2, .1, this.indent2, 20]), q = [], this.addLabelRange(q, Fq("editor.side.from"), hteditor.getter("s", "shape3d.side.from"), hteditor.setter("s", "shape3d.side.from"), 0, void 0, 1, "int"), this.addLabelRange(q, Fq("editor.side.to"), hteditor.getter("s", "shape3d.side.to"), hteditor.setter("s", "shape3d.side.to"), 0, void 0, 1, "int"), this.addRow(q, [this.indent, .1, this.indent2, .1])
            }, v(a, [{
                key: "faces",
                get: function () {
                    return Sq
                }
            }]), a
        }(lq),
        eq = hteditor.getString,
        Bq = function (q) {
            function a(k, w) {
                return V(this, a), Q(this, q.call(this, k, w || "Sphere"))
            }
            return E(a, q), a.prototype.initForm = function () {
                this.addCustomProperties(), this.$11$(), this.addLightProperties(), this.addTransformProperties()
            }, a.prototype.$11$ = function () {
                var q = this;
                this.addTitle("TitleBasic"), this.addEventProperties();
                var a = [];
                this.addLabelInput(a, eq("editor.name"), hteditor.getter("p", "displayName"), hteditor.setter("p", "displayName")), this.addLabelInput(a, eq("editor.tag"), hteditor.getter("p", "tag"), function (a, k) {
                    if (R.config.checkTagConflicts && k)
                        for (var w = q.dataModel.getDatas(), I = w.size() - 1; I >= 0; I--) {
                            var K = w.get(I);
                            if (K !== a && K.getTag() === k) return q.editor.messageView.show(m(eq("editor.errormessage.tagalreadyexists"), k), "error"), void q.$34$()
                        }
                    a.setTag(k)
                }), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelInput(a, eq("editor.tooltip"), hteditor.getter("p", "toolTip"), hteditor.setter("p", "toolTip")), this.addRow(a, [this.indent, .1]), a = [], this.addLabelCheckBox(a, eq("editor.selectable"), function (q) {
                    return q.s("3d.selectable")
                }, function (q, a) {
                    q.s("2d.selectable", a), q.s("3d.selectable", a)
                }), this.addLabelCheckBox(a, eq("editor.movable"), function (q) {
                    return q.s("3d.movable")
                }, function (q, a) {
                    q.s("2d.movable", a), q.s("3d.movable", a)
                }), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelCheckBox(a, eq("editor.editable"), function (q) {
                    return q.s("3d.editable")
                }, function (q, a) {
                    q.s("2d.editable", a), q.s("3d.editable", a)
                }), this.addLabelCheckBox(a, eq("editor.visible"), function (q) {
                    return q.s("3d.visible")
                }, function (q, a) {
                    q.s("2d.visible", a), q.s("3d.visible", a)
                }), this.addRow(a, [this.indent, .1, this.indent2, .1])
            }, a.prototype.addLightProperties = function () {
                var q = this,
                    a = R.config.numberPrecision,
                    k = [];
                this.addLabelComboBox(k, eq("editor.type"), hteditor.getter("s", "light.type"), function (a, k) {
                    a.s("light.type", k), q.editor.updateInspector()
                }, ["point", "spot", "directional"], [eq("editor.light.point"), eq("editor.light.spot"), eq("editor.light.directional")]), this.addLabelCheckBox(k, eq("editor.disabled"), hteditor.getter("s", "light.disabled"), hteditor.setter("s", "light.disabled")), this.addRow(k, [this.indent, .1, this.indent2, .1]), k = [], this.addLabelColor(k, eq("editor.color"), hteditor.getter("s", "light.color"), hteditor.setter("s", "light.color")), this.addLabelRange(k, eq("editor.strength"), hteditor.getter("s", "light.intensity"), hteditor.setter("s", "light.intensity"), 0, void 0, .01, "number"), this.addRow(k, [this.indent, .1, this.indent2, .1]), k = [], this.addLabelRange(k, eq("editor.range"), hteditor.getter("s", "light.range"), hteditor.setter("s", "light.range"), 0, void 0, 1, "number"), this.addRow(k, [this.indent, .1]).visible = function (q) {
                    return "directional" !== q.data.s("light.type")
                }, k = [], this.addLabelRange(k, eq("editor.light.angle"), function (q) {
                    return Number((180 / Math.PI * q.s("light.angle")).toFixed(0))
                }, function (q, a) {
                    q.s("light.angle", a * Math.PI / 180)
                }, 0, 180, 1, "number"), this.addLabelRange(k, eq("editor.dampingexponent"), hteditor.getter("s", "light.exponent"), hteditor.setter("s", "light.exponent"), 0, void 0, 1, "number"), this.addRow(k, [this.indent, .1, this.indent2, .1]).visible = function (q) {
                    return "spot" === q.data.s("light.type")
                }, k = [], k.push(eq("editor.center")), this.addButton(k, null, eq("editor.reset"), "editor.resetsize.state", function (q) {
                    q.s("light.center", [0, 0, 0])
                });
                var w = a.position || 0,
                    I = 0 === w ? "int" : "number",
                    K = 1 / Math.pow(10, w);
                this.addLabelInput(k, "X", function (q) {
                    return Number(q.s("light.center")[0].toFixed(w))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("light.center"));
                    k[0] = parseFloat(a), q.s("light.center", k)
                }, I, K), this.addLabelInput(k, "Y", function (q) {
                    return Number(q.s("light.center")[1].toFixed(w))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("light.center"));
                    k[1] = parseFloat(a), q.s("light.center", k)
                }, I, K), this.addLabelInput(k, "Z", function (q) {
                    return Number(q.s("light.center")[2].toFixed(w))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("light.center"));
                    k[2] = parseFloat(a), q.s("light.center", k)
                }, I, K), this.addRow(k, [this.indent - 40 - 8, 20, 20, .1, 20, .1, 20, .1]).visible = function (q) {
                    return "spot" === q.data.s("light.type")
                }, k = [];
                var b = this.addLabelComboBox(k, eq("editor.group"), hteditor.getter("s", "light.group"), hteditor.setter("s", "light.group"), [0], [eq("editor.default")]);
                this.addRow(k, [this.indent, .1]), b.getLabels = function () {
                    var a = q.editor.dm;
                    if (!a) return [eq("editor.default")];
                    var k = a.a("sceneLightGruopAlias");
                    if (!k) return [eq("editor.default")];
                    var w = ht.Default.clone(k);
                    return [eq("editor.default")].concat(Object.values(w))
                }, b.getValues = function () {
                    var a = q.editor.dm;
                    if (!a) return [eq("editor.default")];
                    var k = a.a("sceneLightGruopAlias");
                    if (!k) return [0];
                    var w = ht.Default.clone(k),
                        I = [0];
                    return Object.keys(w).forEach(function (q) {
                        I.push(Number(q))
                    }), I
                }
            }, a
        }(jq),
        Aq = hteditor.getString,
        hq = function (q) {
            function a(k, w) {
                return V(this, a), Q(this, q.call(this, k, w || "Torus"))
            }
            return E(a, q), a.prototype.addEffectProperties = function () {
                var q = [];
                this.addLabelRange(q, Aq("editor.side"), hteditor.getter("s", "shape3d.side"), hteditor.setter("s", "shape3d.side"), 0, void 0, 1, "int"), this.addLabelCheckBox(q, Aq("editor.smooth"), hteditor.getter("s", "shape3d.smooth"), hteditor.setter("s", "shape3d.smooth")), this.addRow(q, [this.indent, .1, this.indent2, 20]), q = [], this.addLabelRange(q, Aq("editor.torus.radius"), hteditor.getter("s", "shape3d.torus.radius"), hteditor.setter("s", "shape3d.torus.radius"), 0, .25, .001, "number"), this.addLabelRange(q, Aq("editor.resolution"), hteditor.getter("s", "shape3d.resolution"), hteditor.setter("s", "shape3d.resolution"), 0, void 0, 1, "int"), this.addRow(q, [this.indent, .1, this.indent2, .1]), q = [], this.addLabelRange(q, Aq("editor.side.from"), hteditor.getter("s", "shape3d.side.from"), hteditor.setter("s", "shape3d.side.from"), 0, void 0, 1, "int"), this.addLabelRange(q, Aq("editor.side.to"), hteditor.getter("s", "shape3d.side.to"), hteditor.setter("s", "shape3d.side.to"), 0, void 0, 1, "int"), this.addRow(q, [this.indent, .1, this.indent2, .1])
            }, a
        }(dq),
        Jq = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "Triangle"))
            }
            return E(a, q), a
        }(jq),
        tq = hteditor.getString,
        Oq = ht.Default.getShape3dModel,
        $q = ht.Default.clone,
        cq = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "Model"))
            }
            return E(a, q), a.prototype.addTransformProperties = function () {
                var a = this;
                q.prototype.addTransformProperties.call(this);
                var k = [];
                this.addButton(k, tq("editor.resetposition"), null, null, function (q) {
                    var k = Oq(q.s("shape3d"));
                    k && k.center && (a.editor.beginTransaction(), q.setAnchor3d(.5, .5, .5), q.p3(k.center), a.editor.endTransaction())
                }), this.addRow(k, [.1]).visible = function () {
                    var q = a.editor.ld;
                    if (!q) return !1;
                    var k = Oq(q.s("shape3d"));
                    return k && k.center
                };
                var w = function () {
                    var q = a.data,
                        k = Oq(q.s("shape3d"));
                    if (k) return !k.model3d
                };
                this.addTitle("TitleShape3dStyle").visible = w, k = [], this.addLabelImageOrMaterial(k, tq("editor.imageormaterial"), function (q) {
                    var k = q.s("shape3d.material"),
                        w = a.editor.getFileNode(k);
                    if (k) return w ? w.url : I;
                    var I = q.s("shape3d.image"),
                        K = a.editor.getFileNode(I);
                    return K ? K.url : I
                }, function (q, k) {
                    var w = a.editor.getFileNode(k);
                    w && "material" === w.fileType ? (q.s("shape3d.material", k), q.s("shape3d.image", void 0), q.setImage(void 0)) : (q.s("shape3d.image", k), q.s("shape3d.material", void 0), q.setImage(k), q.s("texture.cache") && a.editor.scene.invalidateShape3dCachedImage(q)), a.filterPropertiesLater()
                }), this.addOneRow(k);
                var I = function () {
                    return !a.data.s("shape3d.material")
                };
                k = [], this.addLabelColor(k, tq("editor.color"), hteditor.getter("s", "shape3d.color"), hteditor.setter("s", "shape3d.color")), this.addLabelCheckBox(k, tq("editor.discardselectable"), hteditor.getter("s", "shape3d.discard.selectable"), hteditor.setter("s", "shape3d.discard.selectable")), this.addRow(k, [this.indent, .1, this.indent, 20]).visible = I, k = [], this.addLabelColor(k, tq("editor.reversecolor"), hteditor.getter("s", "shape3d.reverse.color"), hteditor.setter("s", "shape3d.reverse.color")), this.addRow(k, [this.indent, .1]).visible = I, k = [], this.addLabelCheckBox(k, tq("editor.reversecull"), hteditor.getter("s", "shape3d.reverse.cull"), hteditor.setter("s", "shape3d.reverse.cull")), this.addLabelCheckBox(k, tq("editor.reverseflip"), hteditor.getter("s", "shape3d.reverse.flip"), hteditor.setter("s", "shape3d.reverse.flip")), this.addRow(k, [this.indent, .1, this.indent2, .1]).visible = I, k = [], this.addLabelInput(k, tq("editor.uvrotation"), function (q) {
                    var a = q.s("shape3d.uv.rotation") || 0;
                    return Number((180 / Math.PI * a).toFixed(1))
                }, function (q, a) {
                    q.s("shape3d.uv.rotation", a * Math.PI / 180)
                }, "number", 1), this.addRow(k, [this.indent, .1]).visible = I, k = [], k.push(tq("editor.uvscale")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.uv.scale") || [1, 1];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.scale")) || [1, 1];
                    k[0] = Number(a), q.s("shape3d.uv.scale", k)
                }, "number", 1), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.uv.scale") || [1, 1];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.scale")) || [1, 1];
                    k[1] = Number(a), q.s("shape3d.uv.scale", k)
                }, "number", 1), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = I, k = [], k.push(tq("editor.uvoffset")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.uv.offset") || [0, 0];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.offset")) || [0, 0];
                    k[0] = Number(a), q.s("shape3d.uv.offset", k)
                }, "number", .1), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.uv.offset") || [0, 0];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.offset")) || [0, 0];
                    k[1] = Number(a), q.s("shape3d.uv.offset", k)
                }, "number", .1), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = I, k = [], k.push(tq("editor.uvanchor")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.uv.anchor") || [0, 0];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.anchor")) || [0, 0];
                    k[0] = Number(a), q.s("shape3d.uv.anchor", k)
                }, "number", .01), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.uv.anchor") || [0, 0];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.anchor")) || [0, 0];
                    k[1] = Number(a), q.s("shape3d.uv.anchor", k)
                }, "number", .01), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = I, this.addMatDefProperty()
            }, a.prototype.addMatDefProperty = function () {
                var q = this,
                    a = function () {
                        var a = q.data,
                            k = Oq(a.s("shape3d"));
                        if (k) return !!k.model3d
                    };
                this.addTitle("TitleMaterialList").visible = a;
                var k = this._initMaterialsInspector(),
                    w = this._materialsRow = this.addRow([k], [.1], 2);
                this.updateHandlers.push(function () {
                    if (!w.items.hidden) {
                        var a = q.data,
                            I = a.s("matDef"),
                            K = Oq(a.s("shape3d"));
                        K ? k.data._rawMatDef = K.matDef : K = {};
                        var b = Object.assign($q(K.matDef || {}), I);
                        q._isMatDefChange(b) && q._updateMaterials(b)
                    }
                })
            }, a.prototype._isMatDefChange = function (q) {
                var a = this._materialsInspector,
                    k = a._matDef ? JSON.parse(a._matDef) : {},
                    w = !1;
                if (JSON.stringify(q) === a._matDef) return w;
                for (var I in q)
                    if (X(q[I])) {
                        if (!X(k[I])) return w = !0
                    } else if (q[I] !== k[I]) return w = !0;
                return w
            }, a.prototype._initMaterialsInspector = function () {
                var q = this,
                    a = this._materialsInspector = new Kq(this.editor, "materiallist");
                a.setPadding(0);
                var k = new ht.Data;
                return k.onPropertyChanged = function (a) {
                    if (!k._updatingObjProperties) {
                        var w = a.property,
                            I = a.newValue;
                        if (w.startsWith("a:")) {
                            if (void 0 === I) {
                                var K = w.split(":")[1];
                                k.getAttrObject()["" + K] = q._materialsInspector.data._rawMatDef[K]
                            }
                            q._updateMatDef()
                        }
                    }
                }, a.data = k, a
            }, a.prototype._updateMaterials = function (q) {
                var a = this._materialsInspector;
                a._matDef = JSON.stringify(q), a.data.setAttrObject(q);
                var k = a.indent;
                a.clear(), a.updateHandlers = [];
                for (var w in q) ! function (q) {
                    var w = [],
                        I = function (q, a) {
                            return !!a.view.draggingData && "material" === a.view.draggingData.fileType
                        };
                    a.addLabelTooltip(w, q), a.addURL(w, function (k) {
                        return X(a.data.a(q)) ? "" : a.data.a(q)
                    }, function (k, w) {
                        void 0 === w && X(a.data.a(q)) || a.data.a(q, w)
                    }, I), a.addImage(w, function (a) {
                        return a.a(q)
                    }, void 0, function (q) {
                        return u(q) ? q.substr(0, q.length - 5) + ".png" : null
                    }), a.addRow(w, [k, .1, 20])
                }(w);
                a.$46$ = a._rows, a.filterProperties(), a.setRowHeight(this.getRowHeight()), a.setVGap(this.getVGap());
                var I = a.getRows().length;
                this._materialsRow.height = I ? I * (a.getRowHeight() + a.getHGap()) - a.getHGap() : 2, this.filterPropertiesLater()
            }, a.prototype._updateMatDef = function () {
                var q = this._materialsInspector.data.getAttrObject(),
                    a = this.data,
                    k = {};
                for (var w in q) X(q[w]) || "" === q[w] || q[w] === this._materialsInspector.data._rawMatDef[w] || (k[w] = q[w]);
                this._materialsInspector._matDef = JSON.stringify(Object.assign($q(this._materialsInspector.data._rawMatDef), k)), a.s("matDef", k), this._materialsInspector.filterPropertiesLater()
            }, a
        }(jq),
        Dq = hteditor.getString,
        Yq = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "Box"))
            }
            return E(a, q), a.prototype.addTransformProperties = function () {
                var a = this;
                q.prototype.addTransformProperties.call(this), this.addTitle("TitleShape3dStyle");
                var k = [];
                this.addLabelImageOrMaterial(k, Dq("editor.imageormaterial"), function (q) {
                    var k = q.s("shape3d.material"),
                        w = a.editor.getFileNode(k);
                    if (k) return w ? w.url : I;
                    var I = q.s("shape3d.image"),
                        K = a.editor.getFileNode(I);
                    return K ? K.url : I
                }, function (q, k) {
                    var w = a.editor.getFileNode(k);
                    w && "material" === w.fileType ? (q.s("shape3d.material", k), q.s("shape3d.image", void 0), q.setImage(void 0)) : (q.s("shape3d.image", k), q.s("shape3d.material", void 0), q.setImage(k), q.s("texture.cache") && a.editor.scene.invalidateShape3dCachedImage(q)), a.filterPropertiesLater()
                }), this.addOneRow(k);
                var w = function () {
                    return !a.data.s("shape3d.material")
                };
                k = [], this.addLabelColor(k, Dq("editor.color"), hteditor.getter("s", "shape3d.color"), hteditor.setter("s", "shape3d.color")), this.addLabelCheckBox(k, Dq("editor.discardselectable"), hteditor.getter("s", "shape3d.discard.selectable"), hteditor.setter("s", "shape3d.discard.selectable")), this.addRow(k, [this.indent, .1, this.indent, 20]).visible = w, k = [], this.addLabelColor(k, Dq("editor.reversecolor"), hteditor.getter("s", "shape3d.reverse.color"), hteditor.setter("s", "shape3d.reverse.color")), this.addRow(k, [this.indent, .1]).visible = w, k = [], this.addLabelCheckBox(k, Dq("editor.reversecull"), hteditor.getter("s", "shape3d.reverse.cull"), hteditor.setter("s", "shape3d.reverse.cull")), this.addLabelCheckBox(k, Dq("editor.reverseflip"), hteditor.getter("s", "shape3d.reverse.flip"), hteditor.setter("s", "shape3d.reverse.flip")), this.addRow(k, [this.indent, .1, this.indent2, .1]).visible = w, k = [], this.addLabelInput(k, Dq("editor.uvrotation"), function (q) {
                    var a = q.s("shape3d.uv.rotation") || 0;
                    return Number((180 / Math.PI * a).toFixed(1))
                }, function (q, a) {
                    q.s("shape3d.uv.rotation", a * Math.PI / 180)
                }, "number", 1), this.addRow(k, [this.indent, .1]).visible = w, k = [], k.push(Dq("editor.uvscale")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.uv.scale") || [1, 1];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.scale")) || [1, 1];
                    k[0] = Number(a), q.s("shape3d.uv.scale", k)
                }, "number", 1), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.uv.scale") || [1, 1];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.scale")) || [1, 1];
                    k[1] = Number(a), q.s("shape3d.uv.scale", k)
                }, "number", 1), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = w, k = [], k.push(Dq("editor.uvoffset")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.uv.offset") || [0, 0];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.offset")) || [0, 0];
                    k[0] = Number(a), q.s("shape3d.uv.offset", k)
                }, "number", .1), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.uv.offset") || [0, 0];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.offset")) || [0, 0];
                    k[1] = Number(a), q.s("shape3d.uv.offset", k)
                }, "number", .1), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = w, k = [], k.push(Dq("editor.uvanchor")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.uv.anchor") || [0, 0];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.anchor")) || [0, 0];
                    k[0] = Number(a), q.s("shape3d.uv.anchor", k)
                }, "number", .01), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.uv.anchor") || [0, 0];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.anchor")) || [0, 0];
                    k[1] = Number(a), q.s("shape3d.uv.anchor", k)
                }, "number", .01), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = w
            }, a
        }(jq),
        iq = hteditor.getString,
        Pq = hteditor.getter,
        Nq = hteditor.setter,
        Uq = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "Billboard"))
            }
            return E(a, q), a.prototype.addTransformProperties = function () {
                var a = this;
                q.prototype.addTransformProperties.call(this), this.addTitle("TitleShape3dStyle");
                var k = [];
                this.addLabelImageOrMaterial(k, iq("editor.imageormaterial"), function (q) {
                    var k = q.s("shape3d.material"),
                        w = a.editor.getFileNode(k);
                    if (k) return w ? w.url : I;
                    var I = q.s("shape3d.image"),
                        K = a.editor.getFileNode(I);
                    return K ? K.url : I
                }, function (q, k) {
                    var w = a.editor.getFileNode(k);
                    w && "material" === w.fileType ? (q.s("shape3d.material", k), q.s("shape3d.image", void 0), q.setImage(void 0)) : (q.s("shape3d.image", k), q.s("shape3d.material", void 0), q.setImage(k), q.s("texture.cache") && a.editor.scene.invalidateShape3dCachedImage(q)), a.filterPropertiesLater()
                }), this.addOneRow(k);
                var w = function () {
                    return !a.data.s("shape3d.material")
                };
                k = [], this.addLabelColor(k, iq("editor.color"), hteditor.getter("s", "shape3d.color"), hteditor.setter("s", "shape3d.color")), this.addLabelCheckBox(k, iq("editor.discardselectable"), hteditor.getter("s", "shape3d.discard.selectable"), hteditor.setter("s", "shape3d.discard.selectable")), this.addRow(k, [this.indent, .1, this.indent, 20]).visible = w, k = [], this.addLabelColor(k, iq("editor.reversecolor"), hteditor.getter("s", "shape3d.reverse.color"), hteditor.setter("s", "shape3d.reverse.color")), this.addRow(k, [this.indent, .1]).visible = w, k = [], this.addLabelCheckBox(k, iq("editor.reversecull"), hteditor.getter("s", "shape3d.reverse.cull"), hteditor.setter("s", "shape3d.reverse.cull")), this.addLabelCheckBox(k, iq("editor.reverseflip"), hteditor.getter("s", "shape3d.reverse.flip"), hteditor.setter("s", "shape3d.reverse.flip")), this.addRow(k, [this.indent, .1, this.indent2, .1]).visible = w, k = [], this.addLabelInput(k, iq("editor.uvrotation"), function (q) {
                    var a = q.s("shape3d.uv.rotation") || 0;
                    return Number((180 / Math.PI * a).toFixed(1))
                }, function (q, a) {
                    q.s("shape3d.uv.rotation", a * Math.PI / 180)
                }, "number", 1), this.addRow(k, [this.indent, .1]).visible = w, k = [], k.push(iq("editor.uvscale")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.uv.scale") || [1, 1];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.scale")) || [1, 1];
                    k[0] = Number(a), q.s("shape3d.uv.scale", k)
                }, "number", 1), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.uv.scale") || [1, 1];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.scale")) || [1, 1];
                    k[1] = Number(a), q.s("shape3d.uv.scale", k)
                }, "number", 1), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = w, k = [], k.push(iq("editor.uvoffset")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.uv.offset") || [0, 0];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.offset")) || [0, 0];
                    k[0] = Number(a), q.s("shape3d.uv.offset", k)
                }, "number", .1), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.uv.offset") || [0, 0];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.offset")) || [0, 0];
                    k[1] = Number(a), q.s("shape3d.uv.offset", k)
                }, "number", .1), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = w, k = [], k.push(iq("editor.uvanchor")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.uv.anchor") || [0, 0];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.anchor")) || [0, 0];
                    k[0] = Number(a), q.s("shape3d.uv.anchor", k)
                }, "number", .01), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.uv.anchor") || [0, 0];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.anchor")) || [0, 0];
                    k[1] = Number(a), q.s("shape3d.uv.anchor", k)
                }, "number", .01), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = w, this.addTitle("TitleEffectFlow"), k = [], this.addLabelCheckBox(k, iq("editor.effect.flowenable"), hteditor.getter("s", "effect.flow"), hteditor.setter("s", "effect.flow"));
                var I = this.addLabelComboBox(k, iq("editor.group"), hteditor.getter("s", "effect.flow.group"), hteditor.setter("s", "effect.flow.group"), [0], [iq("editor.default")]);
                this.addRow(k, [this.indent, .1, this.indent2, .1]), I.getLabels = function () {
                    var q = a.editor.dm;
                    if (!q) return [iq("editor.default")];
                    var k = q.a("sceneFlowEffectGroupAlias");
                    if (!k) return [iq("editor.default")];
                    var w = ht.Default.clone(k);
                    return [iq("editor.default")].concat(Object.values(w))
                }, I.getValues = function () {
                    var q = a.editor.dm;
                    if (!q) return [iq("editor.default")];
                    var k = q.a("sceneFlowEffectGroupAlias");
                    if (!k) return [0];
                    var w = ht.Default.clone(k),
                        I = [0];
                    return Object.keys(w).forEach(function (q) {
                        I.push(Number(q))
                    }), I
                }, k = [];
                var K = ht.Default.clone(hteditor.config.gradients);
                K[0] = iq("editor.effect.flowgradient.defaultvalue"), this.addLabelComboBox(k, iq("editor.gradient"), hteditor.getter("s", "effect.flow.gradient"), hteditor.setter("s", "effect.flow.gradient"), hteditor.config.gradients, K, hteditor.config.gradientIcons), this.addLabelComboBox(k, iq("editor.effect.flowuvshape"), hteditor.getter("s", "effect.flow.uvshape"), hteditor.setter("s", "effect.flow.uvshape"), ["rect", "circle", "polygon3", "polygon4", "polygon8"], [iq("editor.effect.flowuvshape.rect"), iq("editor.effect.flowuvshape.circle"), iq("editor.effect.flowuvshape.polygon3"), iq("editor.effect.flowuvshape.polygon4"), iq("editor.effect.flowuvshape.polygon8")]), this.addRow(k, [this.indent, .1, this.indent2, .1]), k = [], this.addLabelColor(k, iq("editor.blend"), Pq("s", "effect.flow.blend"), Nq("s", "effect.flow.blend")), this.addLabelComboBox(k, iq("editor.effect.flowsize"), hteditor.getter("s", "effect.flow.size"), hteditor.setter("s", "effect.flow.size"), [128, 256, 512, 1024, 2048, 4096, 8192], [iq("editor.effect.flowsize.128"), iq("editor.effect.flowsize.256"), iq("editor.effect.flowsize.512"), iq("editor.effect.flowsize.1024"), iq("editor.effect.flowsize.2048"), iq("editor.effect.flowsize.4096"), iq("editor.effect.flowsize.8192")]), this.addRow(k, [this.indent, .1, this.indent2, .1]), k = [], this.addLabelSlider(k, iq("editor.effect.flowintensity"), hteditor.getter("s", "effect.flow.intensity"), hteditor.setter("s", "effect.flow.intensity"), 0, 1, .001), this.addInput(k, hteditor.getter("s", "effect.flow.intensity"), hteditor.setter("s", "effect.flow.intensity"), "number"), this.addRow(k, [this.indent, .1, 76])
            }, a.prototype.addReflectorProperties = function () {
                this.addTitle("TitleReflector");
                var q = [];
                this.addLabelCheckBox(q, iq("editor.reflector"), Pq("s", "shape3d.reflector"), Nq("s", "shape3d.reflector")), this.addLabelRange(q, iq("editor.reflectorblur"), Pq("s", "shape3d.reflector.blur"), Nq("s", "shape3d.reflector.blur"), 0, void 0, .1, "number"), this.addRow(q, [this.indent, .1, this.indent2, .1]), q = [], this.addLabelColor(q, iq("editor.background"), Pq("s", "shape3d.reflector.background"), Nq("s", "shape3d.reflector.background")), this.addLabelColor(q, iq("editor.reflectorcolor"), Pq("s", "shape3d.reflector.color"), Nq("s", "shape3d.reflector.color")), this.addRow(q, [this.indent, .1, this.indent2, .1])
            }, a
        }(jq),
        qa = hteditor.getString,
        aa = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "Wall"))
            }
            return E(a, q), a.prototype.$11$ = function () {
                q.prototype.$11$.call(this);
                var a = [];
                this.addLabelCheckBox(a, qa("editor.closepath"), hteditor.getter("p", "closePath"), hteditor.setter("p", "closePath")), this.addRow(a, [this.indent, .1])
            }, a.prototype.addTransformProperties = function () {
                q.prototype.addTransformProperties.call(this);
                var a = [];
                this.addLabelRange(a, qa("editor.uvlength"), hteditor.getter("s", "repeat.uv.length"), hteditor.setter("s", "repeat.uv.length"), 0, void 0, 1, "int"), this.addLabelRange(a, qa("editor.thickness"), hteditor.getter("p", "thickness"), hteditor.setter("p", "thickness"), 0, Number.MAX_VALUE, 1, "number"), this.addLabelRange(a, qa("editor.resolution"), hteditor.getter("s", "shape3d.resolution"), hteditor.setter("s", "shape3d.resolution"), 0, 200, 1, "int"), this.addRow(a, [this.indent, .1, 30, .1, 30, .1])
            }, a
        }(gq),
        ka = ["shape3d", "shape3d.top", "shape3d.bottom"],
        wa = hteditor.getString,
        Ia = hteditor.getter,
        Ka = hteditor.setter,
        ba = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "Floor"))
            }
            return E(a, q), a.prototype.addEffectProperties = function () {}, a.prototype.addReflectorProperties = function () {
                this.addTitle("TitleReflector");
                var q = [];
                this.addLabelCheckBox(q, wa("editor.reflector"), Ia("s", "shape3d.reflector"), Ka("s", "shape3d.reflector")), this.addLabelRange(q, wa("editor.reflectorblur"), Ia("s", "shape3d.reflector.blur"), Ka("s", "shape3d.reflector.blur"), 0, void 0, .1, "number"), this.addRow(q, [this.indent, .1, this.indent2, .1]), q = [], this.addLabelColor(q, wa("editor.background"), Ia("s", "shape3d.reflector.background"), Ka("s", "shape3d.reflector.background")), this.addLabelColor(q, wa("editor.reflectorcolor"), Ia("s", "shape3d.reflector.color"), Ka("s", "shape3d.reflector.color")), this.addRow(q, [this.indent, .1, this.indent2, .1])
            }, v(a, [{
                key: "faces",
                get: function () {
                    return ka
                }
            }]), a
        }(lq),
        sa = hteditor.getString,
        ua = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "Pipeline"))
            }
            return E(a, q), a.prototype.addTransformProperties = function () {
                q.prototype.addTransformProperties.call(this);
                var a = [];
                this.addLabelRange(a, sa("editor.uvlength"), hteditor.getter("s", "repeat.uv.length"), hteditor.setter("s", "repeat.uv.length"), 0, void 0, 1, "int"), this.addLabelRange(a, sa("editor.thickness"), hteditor.getter("p", "thickness"), hteditor.setter("p", "thickness"), 0, Number.MAX_VALUE, 1), this.addLabelRange(a, sa("editor.resolution"), hteditor.getter("s", "shape3d.resolution"), hteditor.setter("s", "shape3d.resolution"), 0, 200, 1, "int"), this.addRow(a, [this.indent, .1, 30, .1, 30, .1])
            }, a.prototype.addExtraProperties = function (a) {
                q.prototype.addExtraProperties.call(this);
                var k = [];
                this.addLabelInput(k, sa("editor.startangle"), function (q) {
                    return Math.round(180 / Math.PI * q.s("shape3d.start.angle"))
                }, function (q, a) {
                    q.s("shape3d.start.angle", a * Math.PI / 180)
                }, "int", 1), a.addRow(k, [this.indent, .1]), k = [], this.addLabelInput(k, sa("editor.sweepangle"), function (q) {
                    return Math.round(180 / Math.PI * q.s("shape3d.sweep.angle"))
                }, function (q, a) {
                    q.s("shape3d.sweep.angle", a * Math.PI / 180)
                }, "int", 1), a.addRow(k, [this.indent, .1])
            }, v(a, [{
                key: "simple",
                get: function () {
                    return !0
                }
            }]), a
        }(Vq),
        Ca = hteditor.getString,
        La = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "Polyline"))
            }
            return E(a, q), a.prototype.$11$ = function () {
                var a = this;
                q.prototype.$11$.call(this);
                var k = [];
                this.addLabelRange(k, Ca("editor.thickness"), hteditor.getter("p", "thickness"), hteditor.setter("p", "thickness"), 0, Number.MAX_VALUE, 1, "number"), this.addLabelRange(k, Ca("editor.uvlength"), hteditor.getter("s", "repeat.uv.length"), hteditor.setter("s", "repeat.uv.length"), 0, void 0, 1, "int"), this.addRow(k, [this.indent, .1, this.indent2, .1]), k = [], this.addLabelRange(k, Ca("editor.side"), hteditor.getter("s", "shape3d.side"), hteditor.setter("s", "shape3d.side"), 0, void 0, 1, "int"), this.addLabelRange(k, Ca("editor.resolution"), hteditor.getter("s", "shape3d.resolution"), hteditor.setter("s", "shape3d.resolution"), 0, void 0, 1, "int"), this.addLabelCheckBox(k, Ca("editor.smooth"), hteditor.getter("s", "shape3d.smooth"), hteditor.setter("s", "shape3d.smooth")), this.addRow(k, [this.indent, .1, this.indent2, .1, this.indent2, 20]), k = [], this.addLabelRange(k, Ca("editor.startangle"), function (q) {
                    return Number((180 / Math.PI * q.s("shape3d.start.angle")).toFixed(0))
                }, function (q, a) {
                    q.s("shape3d.start.angle", a * Math.PI / 180)
                }, void 0, void 0, 1, "number"), this.addLabelRange(k, Ca("editor.sweepangle"), function (q) {
                    return Number((180 / Math.PI * q.s("shape3d.sweep.angle")).toFixed(0))
                }, function (q, a) {
                    q.s("shape3d.sweep.angle", a * Math.PI / 180)
                }, 0, 360, 1, "int"), this.addRow(k, [this.indent, .1, this.indent2, .1]), k = [], this.addLabelComboBox(k, Ca("editor.topcap"), hteditor.getter("s", "shape3d.top.cap"), function (q, k) {
                    q.s("shape3d.top.cap", k), a.editor.updateInspector()
                }, [void 0, "round", "flat"], ["", Ca("editor.cap.round"), Ca("editor.cap.flat")]), this.addLabelComboBox(k, Ca("editor.bottomcap"), hteditor.getter("s", "shape3d.bottom.cap"), function (q, k) {
                    q.s("shape3d.bottom.cap", k), a.editor.updateInspector()
                }, [void 0, "round", "flat"], ["", Ca("editor.cap.round"), Ca("editor.cap.flat")]), this.addRow(k, [this.indent, .1, this.indent, .1])
            }, a.prototype.addTransformProperties = function () {
                var a = this;
                q.prototype.addTransformProperties.call(this);
                var k = [];
                k.push(Ca("editor.point")), this.addButton(k, null, Ca("editor.reset"), "editor.resetsize.state", function (q) {
                    a.setCurrentPoint("e", 0)
                }), this.addLabelInput(k, "X", function (q) {
                    return a.getCurrentPoint("x")
                }, function (q, k) {
                    a.setCurrentPoint("x", k)
                }, "int", 1), this.addLabelInput(k, "Y", function (q) {
                    return a.getCurrentPoint("e")
                }, function (q, k) {
                    a.setCurrentPoint("e", k)
                }, "int", 1), this.addLabelInput(k, "Z", function (q) {
                    return a.getCurrentPoint("y")
                }, function (q, k) {
                    a.setCurrentPoint("y", k)
                }, "int", 1), this.addRow(k, [this.indent - 40 - 8, 20, 20, .1, 20, .1, 20, .1]), this.addTitle("TitleShape3dStyle"), k = [], this.addLabelImageOrMaterial(k, Ca("editor.imageormaterial"), function (q) {
                    var k = q.s("shape3d.material"),
                        w = a.editor.getFileNode(k);
                    if (k) return w ? w.url : I;
                    var I = q.s("shape3d.image"),
                        K = a.editor.getFileNode(I);
                    return K ? K.url : I
                }, function (q, k) {
                    var w = a.editor.getFileNode(k);
                    w && "material" === w.fileType ? (q.s("shape3d.material", k), q.s("shape3d.image", void 0)) : (q.s("shape3d.image", k), q.s("shape3d.material", void 0), q.s("texture.cache") && a.editor.scene.invalidateShape3dCachedImage(q)), a.filterPropertiesLater()
                }), this.addOneRow(k);
                var w = function () {
                    return !a.data.s("shape3d.material")
                };
                k = [], this.addLabelColor(k, Ca("editor.color"), hteditor.getter("s", "shape3d.color"), hteditor.setter("s", "shape3d.color")), this.addLabelCheckBox(k, Ca("editor.discardselectable"), hteditor.getter("s", "shape3d.discard.selectable"), hteditor.setter("s", "shape3d.discard.selectable")), this.addRow(k, [this.indent, .1, this.indent, 20]).visible = w, k = [], this.addLabelColor(k, Ca("editor.reversecolor"), hteditor.getter("s", "shape3d.reverse.color"), hteditor.setter("s", "shape3d.reverse.color")), this.addRow(k, [this.indent, .1]).visible = w, k = [], this.addLabelCheckBox(k, Ca("editor.reversecull"), hteditor.getter("s", "shape3d.reverse.cull"), hteditor.setter("s", "shape3d.reverse.cull")), this.addLabelCheckBox(k, Ca("editor.reverseflip"), hteditor.getter("s", "shape3d.reverse.flip"), hteditor.setter("s", "shape3d.reverse.flip")), this.addRow(k, [this.indent, .1, this.indent2, .1]).visible = w, k = [], this.addLabelInput(k, Ca("editor.uvrotation"), function (q) {
                    var a = q.s("shape3d.uv.rotation") || 0;
                    return Number((180 / Math.PI * a).toFixed(1))
                }, function (q, a) {
                    q.s("shape3d.uv.rotation", a * Math.PI / 180)
                }, "number", 1), this.addRow(k, [this.indent, .1]).visible = w, k = [], k.push(Ca("editor.uvscale")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.uv.scale") || [1, 1];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.scale")) || [1, 1];
                    k[0] = Number(a), q.s("shape3d.uv.scale", k)
                }, "number", 1), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.uv.scale") || [1, 1];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.scale")) || [1, 1];
                    k[1] = Number(a), q.s("shape3d.uv.scale", k)
                }, "number", 1), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = w, k = [], k.push(Ca("editor.uvoffset")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.uv.offset") || [0, 0];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.offset")) || [0, 0];
                    k[0] = Number(a), q.s("shape3d.uv.offset", k)
                }, "number", .1), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.uv.offset") || [0, 0];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.offset")) || [0, 0];
                    k[1] = Number(a), q.s("shape3d.uv.offset", k)
                }, "number", .1), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = w, k = [], k.push(Ca("editor.uvanchor")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.uv.anchor") || [0, 0];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.anchor")) || [0, 0];
                    k[0] = Number(a), q.s("shape3d.uv.anchor", k)
                }, "number", .01), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.uv.anchor") || [0, 0];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.uv.anchor")) || [0, 0];
                    k[1] = Number(a), q.s("shape3d.uv.anchor", k)
                }, "number", .01), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = w, this.addTitle("TitleShape3dTopStyle").visible = function (q) {
                    return "flat" === q.data.s("shape3d.top.cap")
                }, k = [], this.addLabelImageOrMaterial(k, Ca("editor.imageormaterial"), function (q) {
                    var k = q.s("shape3d.top.material"),
                        w = a.editor.getFileNode(k);
                    if (k) return w ? w.url : I;
                    var I = q.s("shape3d.top.image"),
                        K = a.editor.getFileNode(I);
                    return K ? K.url : I
                }, function (q, k) {
                    var w = a.editor.getFileNode(k);
                    w && "material" === w.fileType ? (q.s("shape3d.top.material", k), q.s("shape3d.top.image", void 0)) : (q.s("shape3d.top.image", k), q.s("shape3d.top.material", void 0), q.s("texture.cache") && a.editor.scene.invalidateShape3dCachedImage(q)), a.filterPropertiesLater()
                }), this.addOneRow(k);
                var I = function () {
                    return !a.data.s("shape3d.top.material")
                };
                k = [], this.addLabelColor(k, Ca("editor.color"), hteditor.getter("s", "shape3d.top.color"), hteditor.setter("s", "shape3d.top.color")), this.addLabelCheckBox(k, Ca("editor.discardselectable"), hteditor.getter("s", "shape3d.top.discard.selectable"), hteditor.setter("s", "shape3d.top.discard.selectable")), this.addRow(k, [this.indent, .1, this.indent, 20]).visible = I, k = [], this.addLabelInput(k, Ca("editor.uvrotation"), function (q) {
                    var a = q.s("shape3d.top.uv.rotation") || 0;
                    return Number((180 / Math.PI * a).toFixed(1))
                }, function (q, a) {
                    q.s("shape3d.top.uv.rotation", a * Math.PI / 180)
                }, "number", 1), this.addRow(k, [this.indent, .1]).visible = I, k = [], k.push(Ca("editor.uvscale")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.top.uv.scale") || [1, 1];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.top.uv.scale")) || [1, 1];
                    k[0] = Number(a), q.s("shape3d.top.uv.scale", k)
                }, "number", 1), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.top.uv.scale") || [1, 1];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.top.uv.scale")) || [1, 1];
                    k[1] = Number(a), q.s("shape3d.top.uv.scale", k)
                }, "number", 1), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = I, k = [], k.push(Ca("editor.uvoffset")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.top.uv.offset") || [0, 0];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.top.uv.offset")) || [0, 0];
                    k[0] = Number(a), q.s("shape3d.top.uv.offset", k)
                }, "number", .1), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.top.uv.offset") || [0, 0];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.top.uv.offset")) || [0, 0];
                    k[1] = Number(a), q.s("shape3d.top.uv.offset", k)
                }, "number", .1), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = I, k = [], k.push(Ca("editor.uvanchor")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.top.uv.anchor") || [0, 0];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.top.uv.anchor")) || [0, 0];
                    k[0] = Number(a), q.s("shape3d.top.uv.anchor", k)
                }, "number", .01), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.top.uv.anchor") || [0, 0];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.top.uv.anchor")) || [0, 0];
                    k[1] = Number(a), q.s("shape3d.top.uv.anchor", k)
                }, "number", .01), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = I, this.addTitle("TitleShape3dBottomStyle").visible = function (q) {
                    return "flat" === q.data.s("shape3d.bottom.cap")
                }, k = [], this.addLabelImageOrMaterial(k, Ca("editor.imageormaterial"), function (q) {
                    var k = q.s("shape3d.bottom.material"),
                        w = a.editor.getFileNode(k);
                    if (k) return w ? w.url : I;
                    var I = q.s("shape3d.bottom.image"),
                        K = a.editor.getFileNode(I);
                    return K ? K.url : I
                }, function (q, k) {
                    var w = a.editor.getFileNode(k);
                    w && "material" === w.fileType ? (q.s("shape3d.bottom.material", k), q.s("shape3d.bottom.image", void 0)) : (q.s("shape3d.bottom.image", k), q.s("shape3d.bottom.material", void 0), q.s("texture.cache") && a.editor.scene.invalidateShape3dCachedImage(q)), a.filterPropertiesLater()
                }), this.addOneRow(k);
                var K = function () {
                    return !a.data.s("shape3d.bottom.material")
                };
                k = [], this.addLabelColor(k, Ca("editor.color"), hteditor.getter("s", "shape3d.bottom.color"), hteditor.setter("s", "shape3d.bottom.color")), this.addLabelCheckBox(k, Ca("editor.discardselectable"), hteditor.getter("s", "shape3d.bottom.discard.selectable"), hteditor.setter("s", "shape3d.bottom.discard.selectable")), this.addRow(k, [this.indent, .1, this.indent, 20]).visible = K, k = [], this.addLabelInput(k, Ca("editor.uvrotation"), function (q) {
                    var a = q.s("shape3d.bottom.uv.rotation") || 0;
                    return Number((180 / Math.PI * a).toFixed(1))
                }, function (q, a) {
                    q.s("shape3d.bottom.uv.rotation", a * Math.PI / 180)
                }, "number", 1), this.addRow(k, [this.indent, .1]).visible = K, k = [], k.push(Ca("editor.uvscale")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.bottom.uv.scale") || [1, 1];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.bottom.uv.scale")) || [1, 1];
                    k[0] = Number(a), q.s("shape3d.bottom.uv.scale", k)
                }, "number", 1), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.bottom.uv.scale") || [1, 1];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.bottom.uv.scale")) || [1, 1];
                    k[1] = Number(a), q.s("shape3d.bottom.uv.scale", k)
                }, "number", 1), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = K, k = [], k.push(Ca("editor.uvoffset")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.bottom.uv.offset") || [0, 0];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.bottom.uv.offset")) || [0, 0];
                    k[0] = Number(a), q.s("shape3d.bottom.uv.offset", k)
                }, "number", .1), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.bottom.uv.offset") || [0, 0];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.bottom.uv.offset")) || [0, 0];
                    k[1] = Number(a), q.s("shape3d.bottom.uv.offset", k)
                }, "number", .1), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = K, k = [], k.push(Ca("editor.uvanchor")), this.addLabelInput(k, "U", function (q) {
                    var a = q.s("shape3d.bottom.uv.anchor") || [0, 0];
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.bottom.uv.anchor")) || [0, 0];
                    k[0] = Number(a), q.s("shape3d.bottom.uv.anchor", k)
                }, "number", .01), this.addLabelInput(k, "V", function (q) {
                    var a = q.s("shape3d.bottom.uv.anchor") || [0, 0];
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.s("shape3d.bottom.uv.anchor")) || [0, 0];
                    k[1] = Number(a), q.s("shape3d.bottom.uv.anchor", k)
                }, "number", .01), this.addRow(k, [this.indent - 20 - 4, 20, .1, 20, .1]).visible = K
            }, v(a, [{
                key: "simple",
                get: function () {
                    return !0
                }
            }]), a
        }(Vq),
        ja = hteditor.getString,
        Ga = function (q) {
            return function (a) {
                var k = a.s(q);
                return k ? k[0] : 1
            }
        },
        Xa = function (q) {
            return function (a, k) {
                var w = a.s(q);
                a.s(q, [k, w ? w[1] : 1])
            }
        },
        Za = function (q) {
            return function (a) {
                var k = a.s(q);
                return k ? k[1] : 1
            }
        },
        ra = function (q) {
            return function (a, k) {
                var w = a.s(q);
                a.s(q, [w ? w[0] : 1, k])
            }
        },
        Wa = function (q) {
            return function (a) {
                a.s(q, null)
            }
        },
        Ta = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "Edge"))
            }
            return E(a, q), a.prototype.initForm = function () {
                q.prototype.initForm.call(this), this.$88$(), this.$90$(), this.addEdgeSourceNodeProperties(), this.addEdgeTargetNodeProperties()
            }, a.prototype.$11$ = function () {
                var q = this;
                this.addTitle("TitleBasic");
                var a = [];
                this.addLabelInput(a, ja("editor.name"), hteditor.getter("p", "displayName"), hteditor.setter("p", "displayName")), this.addLabelInput(a, ja("editor.tag"), hteditor.getter("p", "tag"), function (a, k) {
                    if (R.config.checkTagConflicts && k)
                        for (var w = q.dataModel.getDatas(), I = w.size() - 1; I >= 0; I--) {
                            var K = w.get(I);
                            if (K !== a && K.getTag() === k) return q.editor.messageView.show(m(ja("editor.errormessage.tagalreadyexists"), k), "error"), void q.$34$()
                        }
                    a.setTag(k)
                }), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelURL(a, ja("editor.navigate"), function (q) {
                    return q.a("navigate") || ""
                }, function (q, a) {
                    q.a("navigate", a)
                }, function (q, a) {
                    var k = a.view.draggingData;
                    return k && ("scene" === k.fileType || "display" === k.fileType)
                }), this.addLabelInput(a, ja("editor.tooltip"), hteditor.getter("p", "toolTip"), hteditor.setter("p", "toolTip")), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelComboBoxURL(a, ja("editor.type"), function (a) {
                    var k = a.s("shape3d");
                    if (u(k)) {
                        var w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }
                    return k
                }, function (a, k) {
                    a.s("shape3d", k), q.editor.updateInspector()
                }, ["", "cylinder"], void 0, void 0, function (q, a) {
                    return !!a.view.draggingData && "model" === a.view.draggingData.fileType
                }), this.addImage(a, function (q) {
                    return q.s("shape3d")
                }, void 0, function (q) {
                    return hteditor.isJSON(q) ? q.substr(0, q.length - 5) + ".png" : null
                }, function (q) {
                    return hteditor.isJSON(q.rawIcon)
                }), this.addOneRow(a), a = [], this.addLabelCheckBox(a, ja("editor.selectable"), function (q) {
                    return q.s("3d.selectable")
                }, function (q, a) {
                    q.s("2d.selectable", a), q.s("3d.selectable", a)
                }), this.addLabelCheckBox(a, ja("editor.movable"), function (q) {
                    return q.s("3d.movable")
                }, function (q, a) {
                    q.s("2d.movable", a), q.s("3d.movable", a)
                }), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelCheckBox(a, ja("editor.editable"), function (q) {
                    return q.s("3d.editable")
                }, function (q, a) {
                    q.s("2d.editable", a), q.s("3d.editable", a)
                }), this.addLabelCheckBox(a, ja("editor.visible"), function (q) {
                    return q.s("3d.visible")
                }, function (q, a) {
                    q.s("2d.visible", a), q.s("3d.visible", a)
                }), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelCheckBox(a, ja("editor.interactive"), function (q) {
                    return q.s("interactive")
                }, function (q, a) {
                    return q.s("interactive", a)
                }), this.addRow(a, [this.indent, .1]), a = [], this.addLabelRange(a, ja("editor.envmap"), function (q) {
                    return q.s("envmap") || 0
                }, hteditor.setter("s", "envmap"), 0, 1, .01, "number"), this.addRow(a, [this.indent, .1]), a = [], this.addLabelRange(a, ja("editor.brightness"), function (q) {
                    return q.s("select.brightness")
                }, hteditor.setter("s", "select.brightness"), 0, 1, .01, "number"), this.addRow(a, [this.indent, .1]), a = [], this.addLabelObject(a, ja("editor.polygonoffset"), function (q) {
                    return q.s("polygonOffset")
                }, hteditor.setter("s", "polygonOffset"), "number"), this.addRow(a, [this.indent, .1])
            }, a.prototype.addTransformProperties = function () {}, a.prototype.$88$ = function () {
                var q = this;
                this.addTitle("TitleEdgeBasic");
                var a = [];
                this.addLabelComboBox(a, ja("editor.type"), hteditor.getter("s", "edge.type"), hteditor.setter("s", "edge.type"), [void 0, "points"]), this.addRow(a, [this.indent, .1]), a = [], this.addLabelInput(a, ja("editor.width"), hteditor.getter("s", "edge.width"), hteditor.setter("s", "edge.width"), "number", 1), this.addLabelInput(a, ja("editor.offset"), hteditor.getter("s", "edge.offset"), hteditor.setter("s", "edge.offset"), "number", 1), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelInput(a, ja("editor.side"), hteditor.getter("s", "shape3d.side"), hteditor.setter("s", "shape3d.side"), "number", 1), this.addLabelInput(a, ja("editor.resolution"), hteditor.getter("s", "shape3d.resolution"), hteditor.setter("s", "shape3d.resolution"), "number", 1), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelColor(a, ja("editor.color"), hteditor.getter("s", "edge.color"), hteditor.setter("s", "edge.color")), this.addLabelCheckBox(a, ja("editor.center"), hteditor.getter("s", "edge.center"), hteditor.setter("s", "edge.center")), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [];
                var k = hteditor.getter("s", "shape3d.image");
                this.addLabelImage(a, ja("editor.image"), function (a) {
                    var w = k(a),
                        I = q.editor.getFileNode(w);
                    return I ? I.url : w
                }, hteditor.setter("s", "shape3d.image")), this.addRow(a, [this.indent, .1, 20]), a = [], this.addLabel(a, "UV"), this.addButton(a, null, ja("editor.reset"), "editor.resetsize.state", Wa("shape3d.uv.scale")), this.addLabelInput(a, "U", Ga("shape3d.uv.scale"), Xa("shape3d.uv.scale"), "number", 1), this.addLabelInput(a, "V", Za("shape3d.uv.scale"), ra("shape3d.uv.scale"), "number", 1), this.addRow(a, [this.indent - 20 - 8, 20, 20, .1, 20, .1]), a = [], this.addLabel(a, "UV " + ja("editor.offset")), this.addButton(a, null, ja("editor.reset"), "editor.resetsize.state", Wa("shape3d.uv.offset")), this.addLabelInput(a, "U", function (q) {
                    return (q.s("shape3d.uv.offset") || [0, 0])[0]
                }, function (q, a) {
                    var k = q.s("shape3d.uv.offset") || [];
                    q.s("shape3d.uv.offset", [a, k[1]])
                }, "number", .1), this.addLabelInput(a, "V", function (q) {
                    return (q.s("shape3d.uv.offset") || [0, 0])[1]
                }, function (q, a) {
                    var k = q.s("shape3d.uv.offset") || [];
                    q.s("shape3d.uv.offset", [k[0], a])
                }, "number", .1), this.addRow(a, [this.indent - 20 - 8, 20, 20, .1, 20, .1])
            }, a.prototype.$90$ = function () {}, a.prototype.addEdgeSourceNodeProperties = function () {
                var q = R.config.numberPrecision;
                this.addTitle("TitleEdgeSourceNode");
                var a = [];
                this.addLabelData(a, ja("editor.node"), hteditor.getter("p", "source"), function (q, a) {
                    q instanceof ht.Edge && (null == a || a instanceof ht.Node || a instanceof ht.Edge) && (q.setSource(a), q.s("edge.source.percent", .5))
                }), this.addRow(a, [this.indent, .1, 20]);
                var k = q.anchor || 0,
                    w = 0 === k ? "int" : "number",
                    I = 1 / Math.pow(10, k);
                a = [], a.push(ja("editor.anchor")), this.addButton(a, null, ja("editor.reset"), "editor.resetsize.state", function (q) {
                    q instanceof ht.Edge && q.s({
                        "edge.source.anchor.x": .5,
                        "edge.source.anchor.elevation": .5,
                        "edge.source.anchor.y": .5
                    })
                }), this.addLabelInput(a, "X", function (q) {
                    var a = q.s("edge.source.anchor.x");
                    return null == a && (a = .5), Number(a.toFixed(k))
                }, function (q, a) {
                    q instanceof ht.Edge && q.s("edge.source.anchor.x", parseFloat(a))
                }, w, I), this.addLabelInput(a, "Y", function (q) {
                    var a = q.s("edge.source.anchor.elevation");
                    return null == a && (a = .5), Number(a.toFixed(k))
                }, function (q, a) {
                    q instanceof ht.Edge && q.s("edge.source.anchor.elevation", parseFloat(a))
                }, w, I), this.addLabelInput(a, "Z", function (q) {
                    var a = q.s("edge.source.anchor.y");
                    return null == a && (a = .5), Number(a.toFixed(k))
                }, function (q, a) {
                    q instanceof ht.Edge && q.s("edge.source.anchor.y", parseFloat(a))
                }, w, I), this.addRow(a, [this.indent - 40 - 8, 20, 20, .1, 20, .1, 20, .1])
            }, a.prototype.addEdgeTargetNodeProperties = function () {
                var q = R.config.numberPrecision;
                this.addTitle("TitleEdgeTargetNode");
                var a = [];
                this.addLabelData(a, ja("editor.node"), hteditor.getter("p", "target"), function (q, a) {
                    q instanceof ht.Edge && (null == a || a instanceof ht.Node || a instanceof ht.Edge) && (q.setTarget(a), q.s("edge.target.percent", .5))
                }), this.addRow(a, [this.indent, .1, 20]);
                var k = q.anchor || 0,
                    w = 0 === k ? "int" : "number",
                    I = 1 / Math.pow(10, k);
                a = [], a.push(ja("editor.anchor")), this.addButton(a, null, ja("editor.reset"), "editor.resetsize.state", function (q) {
                    q instanceof ht.Edge && q.s({
                        "edge.target.anchor.x": .5,
                        "edge.target.anchor.elevation": .5,
                        "edge.target.anchor.y": .5
                    })
                }), this.addLabelInput(a, "X", function (q) {
                    var a = q.s("edge.target.anchor.x");
                    return null == a && (a = .5), Number(a.toFixed(k))
                }, function (q, a) {
                    q instanceof ht.Edge && q.s("edge.target.anchor.x", parseFloat(a))
                }, w, I), this.addLabelInput(a, "Y", function (q) {
                    var a = q.s("edge.target.anchor.elevation");
                    return null == a && (a = .5), Number(a.toFixed(k))
                }, function (q, a) {
                    q instanceof ht.Edge && q.s("edge.target.anchor.elevation", parseFloat(a))
                }, w, I), this.addLabelInput(a, "Z", function (q) {
                    var a = q.s("edge.target.anchor.y");
                    return null == a && (a = .5), Number(a.toFixed(k))
                }, function (q, a) {
                    q instanceof ht.Edge && q.s("edge.target.anchor.y", parseFloat(a))
                }, w, I), this.addRow(a, [this.indent - 40 - 8, 20, 20, .1, 20, .1, 20, .1])
            }, a
        }(jq),
        pa = ["edit_tx", "edit_ty", "edit_tz", "edit_rx", "edit_ry", "edit_rz", "edit_sx", "edit_sy", "edit_sz"],
        oa = function (q) {
            function a(k) {
                V(this, a);
                var w = Q(this, q.call(this, k));
                return w.keep = !0, w
            }
            return E(a, q), a.prototype.handle_touchstart = function (a) {
                q.prototype.handle_touchstart.call(this, a);
                var k = this.getState();
                if (!("move" === k || pa.indexOf(k) >= 0 || "select" === k)) {
                    var w = ht.Default.isLeftButton(a),
                        I = ht.Default.isMiddleButton(a),
                        K = 1 === ht.Default.getTouchCount(a);
                    (w || I) && K && this.setState("pan")
                }
            }, a
        }(ht.graph3d.DefaultInteractor),
        ga = R.getString,
        Ra = R.getter,
        Ha = R.setter,
        fa = ht.Default.clone,
        na = ht.Default.setShape3dModel,
        ya = "__modelView.model__",
        Ma = ht.Default.createBoxModel();
    ht.Default.handleUnfoundModel = function () {
        return Ma
    };
    var ma = function (q) {
        function a(k) {
            V(this, a);
            var w = Q(this, q.call(this));
            return w.editor = k, w.init(), w.isDrop = !1, k.addEventListener(function (q) {
                if ("fileChanged" === q.type) {
                    var a = q.params.path,
                        k = w.g3d,
                        I = k.getTextureMap();
                    I && I[a] && k.deleteTexture(a), document.body.contains(k.getView()) && k.invalidateAll(), document.body.contains(w.inspector.getView()) && w.inspector.filterPropertiesLater()
                }
            }), w
        }
        return E(a, q), a.prototype.init = function () {
            var q = this.initG3d(),
                a = this.initInspector(),
                k = this._mainView = new ht.widget.SplitView(a, q, "h", 300);
            this.setConfig({
                title: '<span style="margin-left: 20px">' + ga("editor.modelview") + "</span>",
                width: R.config.modelViewSize.width,
                height: R.config.modelViewSize.height,
                maximizable: !0,
                content: k,
                resizeMode: "wh",
                buttonsAlign: "right",
                buttons: this.buttons,
                closable: !0,
                draggable: !0,
                titleHeight: 38,
                titleBackground: ht.Default.dialogHeaderBackground,
                titleColor: "#303033",
                titleIconGap: 16,
                borderWidth: 0,
                contentPadding: 0
            }), this.setModal(!1)
        }, a.prototype.initG3d = function () {
            return this.dm = new ht.DataModel, this.node = new ht.Node, this.node.s({
                "all.visible": !1,
                "wf.visible": "selected",
                "select.brightness": 1,
                "envmap.probe": -1
            }), this.node.setAnchor3d(.5, .5, .5), this.dm.add(this.node), this.g3d = new ht.graph3d.Graph3dView(this.dm)
        }, a.prototype.initInspector = function () {
            var q = this,
                a = this.inspector = new Kq(this.editor, "model");
            a.getView().style.background = "none", a.setPadding(6), a.v = function () {
                if (this.data) {
                    var q = arguments[0],
                        a = arguments[1];
                    if (1 === arguments.length) return this.data.a(q);
                    this.data.a(q, a)
                }
            }, a.getValue = function (q) {
                if (this.data) return q(this.data)
            }, a.setValue = function (q, a, k) {
                this.data && q(this.data, k ? k() : a)
            };
            var k = a.addTitle;
            a.addTitle = function (q) {
                return R.config.expandedTitles[q] = !0, k.apply(this, arguments)
            };
            var w = new ht.Data;
            return w.onPropertyChanged = function (k) {
                if (!q._updating && !w._updating) {
                    var I = k.property;
                    if (I.startsWith("a:")) {
                        var K = k.newValue;
                        if ("a:scene.background" === I) return void q.g3d.dm().setBackground(K);
                        ["a:modelType", "a:url", "a:flipY", "a:obj", "a:mtl", "a:prefix", "a:t3", "a:s3", "a:r3", "a:ignoreImage", "a:ignoreNormal", "a:ignoreColor", "a:center", "a:reverseFlipMtls", "a:rotationInterpolation", "a:playAutomatically", "a:matDef"].indexOf(I) >= 0 && !a._labelDragging && q.updateModel(), a.updateProperties(), a.filterPropertiesLater()
                    }
                }
            }, a.data = w, this._initBasicProperties(), this._initModelProperties(), this._init2dProperties(), a
        }, a.prototype._initBasicProperties = function () {
            var q = this.inspector,
                a = this.inspector.indent,
                k = [];
            q.addLabelComboBox(k, ga("editor.type"), Ra("a", "modelType"), Ha("a", "modelType"), ["obj", "fbx", "gltf"], ["obj", "fbx", "gltf"]), q.addLabelCheckBox(k, ga("editor.modelview.focusmodel"), Ra("a", "focusModel"), Ha("a", "focusModel")), q.addRow(k, [a, .1, a, 20]), k = [], q.addLabelColor(k, ga("editor.background"), Ra("a", "scene.background"), Ha("a", "scene.background")), q.addRow(k, [a, .1]), k = [], q.addLabelInput(k, ga("editor.path"), Ra("a", "path"), Ha("a", "path")).setEditable(!1), q.addRow(k, [a, .1]), k = [], this.nameInput = q.addLabelInput(k, ga("editor.name"), Ra("a", "name"), Ha("a", "name")), q.addRow(k, [a, .1])
        }, a.prototype._initModelProperties = function () {
            this._initObjProjects(), this._initFbxProjects(), this._initGltfProjects()
        }, a.prototype._initObjProjects = function () {
            var q = this,
                a = this.inspector,
                k = a.indent,
                w = a.getHGap();
            a.addTitle("editor.modelview.title.obj").visible = function () {
                return "obj" === a.v("modelType")
            };
            var I = [];
            a.addRow([{
                element: "OBJ"
            }, {
                id: "obj",
                textField: {
                    handleChange: function (k) {
                        if (a.data._updating = !0, a.data.a("obj", k), !q.autoFillInput("obj", k)) return void(a.data._updating = !1);
                        a.data._updating = !1, q.updateModel(), a.updateProperties()
                    }
                }
            }, {
                image: {
                    icon: "editor.obj",
                    onClicked: function (k) {
                        q.editor.selectFileNode(a.v("obj"))
                    }
                }
            }], [k, .1, 20]);
            var s = a.getViewById("obj");
            this.handleDragAndDrop(s, function (q, a) {
                return !!a.view.draggingData && K(a.view.draggingData.getFileUUID())
            }), a.addRow([{
                element: "MTL"
            }, {
                id: "mtl",
                textField: {
                    handleChange: function (k) {
                        if (a.data._updating = !0, a.data.a("mtl", k), !q.autoFillInput("mtl", k)) return void(a.data._updating = !1);
                        a.data._updating = !1, q.updateModel(), a.updateProperties()
                    }
                }
            }, {
                image: {
                    icon: "editor.mtl",
                    onClicked: function (k) {
                        q.editor.selectFileNode(a.v("mtl"))
                    }
                }
            }], [k, .1, 20]);
            var u = a.getViewById("mtl");
            this.handleDragAndDrop(u, function (q, a) {
                return !!a.view.draggingData && b(a.view.draggingData.getFileUUID())
            }), a.updateHandlers.push(function () {
                a.data && (s.setValue(a.data.a("obj") || ""), u.setValue(a.data.a("mtl") || ""))
            }), I = [], a.addLabelURL(I, ga("editor.prefix"), Ra("a", "prefix"), Ha("a", "prefix")), a.addRow(I, [k, .1]), I = [], I.push(ga("editor.translation"));
            var C = function () {
                a.setValue(Ha("a", "t3"), fa(a.getValue(Ra("a", "t3"))))
            };
            a.addLabelInput(I, "X", function (q) {
                var a = q.a("t3") || [0, 0, 0];
                return Number(a[0].toFixed(5))
            }, function (q, a) {
                var k = fa(q.a("t3") || [0, 0, 0]);
                k[0] = Number(a), q.a("t3", k)
            }, "number", .01, void 0, {
                onEndEditing: C
            }), a.addLabelInput(I, "Y", function (q) {
                var a = q.a("t3") || [0, 0, 0];
                return Number(a[1].toFixed(5))
            }, function (q, a) {
                var k = fa(q.a("t3") || [0, 0, 0]);
                k[1] = Number(a), q.a("t3", k)
            }, "number", .01, void 0, {
                onEndEditing: C
            }), a.addLabelInput(I, "Z", function (q) {
                var a = q.a("t3") || [0, 0, 0];
                return Number(a[2].toFixed(5))
            }, function (q, a) {
                var k = fa(q.a("t3") || [0, 0, 0]);
                k[2] = Number(a), q.a("t3", k)
            }, "number", .01, void 0, {
                onEndEditing: C
            }), a.addRow(I, [k - 20 - w, 20, .1, 20, .1, 20, .1]), I = [], I.push(ga("editor.scale"));
            var L = function () {
                a.setValue(Ha("a", "s3"), fa(a.getValue(Ra("a", "s3"))))
            };
            a.addLabelInput(I, "X", function (q) {
                var a = q.a("s3") || [1, 1, 1];
                return Number(a[0].toFixed(5))
            }, function (q, a) {
                var k = fa(q.a("s3") || [1, 1, 1]);
                k[0] = Number(a), q.a("s3", k)
            }, "number", .01, void 0, {
                onEndEditing: L
            }), a.addLabelInput(I, "Y", function (q) {
                var a = q.a("s3") || [1, 1, 1];
                return Number(a[1].toFixed(5))
            }, function (q, a) {
                var k = fa(q.a("s3") || [1, 1, 1]);
                k[1] = Number(a), q.a("s3", k)
            }, "number", .01, void 0, {
                onEndEditing: L
            }), a.addLabelInput(I, "Z", function (q) {
                var a = q.a("s3") || [1, 1, 1];
                return Number(a[2].toFixed(5))
            }, function (q, a) {
                var k = fa(q.a("s3") || [1, 1, 1]);
                k[2] = Number(a), q.a("s3", k)
            }, "number", .01, void 0, {
                onEndEditing: L
            }), a.addRow(I, [k - 20 - w, 20, .1, 20, .1, 20, .1]), I = [], I.push(ga("editor.rotation"));
            var j = function () {
                a.setValue(Ha("a", "r3"), fa(a.getValue(Ra("a", "r3"))))
            };
            a.addLabelInput(I, "X", function (q) {
                var a = q.a("r3") || [0, 0, 0];
                return 180 / Math.PI * Number(a[0])
            }, function (q, a) {
                var k = fa(q.a("r3") || [0, 0, 0]);
                k[0] = Number(a * Math.PI / 180), q.a("r3", k)
            }, "number", .1, void 0, {
                onEndEditing: j
            }), a.addLabelInput(I, "Y", function (q) {
                var a = q.a("r3") || [0, 0, 0];
                return 180 / Math.PI * Number(a[1])
            }, function (q, a) {
                var k = fa(q.a("r3") || [0, 0, 0]);
                k[1] = Number(a * Math.PI / 180), q.a("r3", k)
            }, "number", .1, void 0, {
                onEndEditing: j
            }), a.addLabelInput(I, "Z", function (q) {
                var a = q.a("r3") || [0, 0, 0];
                return 180 / Math.PI * Number(a[2])
            }, function (q, a) {
                var k = fa(q.a("r3") || [0, 0, 0]);
                k[2] = Number(a * Math.PI / 180), q.a("r3", k)
            }, "number", .1, void 0, {
                onEndEditing: j
            }), a.addRow(I, [k - 20 - w, 20, .1, 20, .1, 20, .1]), I = [], a.addLabelCheckBox(I, ga("editor.ignoreimage"), Ra("a", "ignoreImage"), Ha("a", "ignoreImage")), a.addLabelCheckBox(I, ga("editor.centerpivot"), Ra("a", "center"), Ha("a", "center")), a.addRow(I, [k, 20, .1, 20]), I = [], a.addLabelCheckBox(I, ga("editor.ignorenormal"), Ra("a", "ignoreNormal"), Ha("a", "ignoreNormal")), a.addLabelCheckBox(I, ga("editor.reverseflip"), Ra("a", "reverseFlipMtls"), Ha("a", "reverseFlipMtls")), a.addRow(I, [k, 20, .1, 20]), I = [], a.addLabelCheckBox(I, ga("editor.ignorecolor"), Ra("a", "ignoreColor"), Ha("a", "ignoreColor")), a.addRow(I, [k, 20])
        }, a.prototype._initFbxProjects = function () {
            var q = this,
                a = this.inspector,
                k = a.indent;
            a.addTitle("editor.modelview.title.fbx").visible = function () {
                return "fbx" === a.v("modelType")
            }, a.addRow([{
                element: ga("editor.path")
            }, {
                id: "url",
                textField: {
                    handleChange: function (q) {
                        a.data.a("url", q)
                    }
                }
            }, {
                image: {
                    icon: "editor.fbx",
                    onClicked: function (k) {
                        q.editor.selectFileNode(a.data.a("url"))
                    }
                }
            }], [k, .1, 20]);
            var w = a.getViewById("url");
            this.handleDragAndDrop(w, function (q, a) {
                return !!a.view.draggingData && C(a.view.draggingData.getFileUUID())
            }), a.updateHandlers.push(function () {
                a.data && w.setValue(a.data.a("url") || "")
            });
            var I = [];
            a.addLabelComboBox(I, ga("editor.modelview.rotationInterpolation"), Ra("a", "rotationInterpolation"), Ha("a", "rotationInterpolation"), [void 0, !0, !1], [ga("editor.auto"), ga("editor.open"), ga("editor.close")]), a.addLabelCheckBox(I, ga("editor.centerpivot"), Ra("a", "center"), Ha("a", "center")), a.addRow(I, [k, .1, k, 20]), I = [], a.addLabelCheckBox(I, ga("editor.flipy"), Ra("a", "flipY"), Ha("a", "flipY")), a.addLabelCheckBox(I, ga("editor.modelview.playautomatically"), Ra("a", "playAutomatically"), Ha("a", "playAutomatically")), a.addRow(I, [k, 20, .1, 20]), a.addTitle(ga("editor.materials")).visible = function () {
                return !("fbx" !== a.v("modelType") || !a.v("matDef"))
            }, a.fbxMaterialsInspector = this._initMaterialsInspector(), I = [a.fbxMaterialsInspector], a.fbxMaterialsRow = a.addRow(I, [.1], 2), a.addTitle(ga("editor.animations")).visible = function () {
                return "fbx" === a.v("modelType") && a.v("animations")
            }
        }, a.prototype._initGltfProjects = function () {
            var q = this,
                a = this.inspector,
                k = a.indent;
            a.addTitle("editor.modelview.title.gltf").visible = function () {
                return "gltf" === a.v("modelType")
            }, a.addRow([{
                element: ga("editor.path")
            }, {
                id: "url",
                textField: {
                    handleChange: function (q) {
                        a.data.a("url", q);
                        var k = q.lastIndexOf("/") + 1;
                        a.data.a("prefix", q.substr(0, k)), a.updateProperties()
                    }
                }
            }, {
                image: {
                    icon: "editor.gltf",
                    onClicked: function (k) {
                        q.editor.selectFileNode(a.data.a("url"))
                    }
                }
            }], [k, .1, 20]), I = [], a.addLabelURL(I, ga("editor.prefix"), Ra("a", "prefix"), Ha("a", "prefix")), a.addRow(I, [k, .1]);
            var w = a.getViewById("url");
            this.handleDragAndDrop(w, function (q, a) {
                return !!a.view.draggingData && L(a.view.draggingData.getFileUUID()) || j(a.view.draggingData.getFileUUID())
            }), a.updateHandlers.push(function () {
                a.data && w.setValue(a.data.a("url") || "")
            });
            var I = [];
            a.addLabelComboBox(I, ga("editor.modelview.rotationInterpolation"), Ra("a", "rotationInterpolation"), Ha("a", "rotationInterpolation"), [void 0, !0, !1], [ga("editor.auto"), ga("editor.open"), ga("editor.close")]), a.addLabelCheckBox(I, ga("editor.centerpivot"), Ra("a", "center"), Ha("a", "center")), a.addRow(I, [k, .1, k, 20]), I = [], a.addLabelCheckBox(I, ga("editor.flipy"), Ra("a", "flipY"), Ha("a", "flipY")), a.addLabelCheckBox(I, ga("editor.modelview.playautomatically"), Ra("a", "playAutomatically"), Ha("a", "playAutomatically")), a.addRow(I, [k, 20, .1, 20]), a.addTitle(ga("editor.materials")).visible = function () {
                return !("gltf" !== a.v("modelType") || !a.v("matDef"))
            }, a.gltfMaterialsInspector = this._initMaterialsInspector(), I = [a.gltfMaterialsInspector], a.gltfMaterialsRow = a.addRow(I, [.1], 2), a.addTitle(ga("editor.animations")).visible = function () {
                return "gltf" === a.v("modelType") && a.v("animations")
            }
        }, a.prototype._init2dProperties = function () {}, a.prototype._initMaterialsInspector = function () {
            var q = this,
                a = new Kq(this.editor, "material");
            a.setHPadding(0), a.setVPadding(0);
            var k = new ht.Data;
            return k.onPropertyChanged = function (w) {
                if (!k._updating && !q._updating) {
                    w.property.startsWith("a:") && q._updateMatDef(a)
                }
            }, a.data = k, a
        }, a.prototype._updateMaterials = function (q, a, k) {
            var w = this;
            if (!this.inspector.v("url")) return a._url = this.inspector.v("url"), void a.clear();
            var I = {};
            for (var K in q) Z(q[K]) && "" !== q[K] ? I[K] = q[K] : I[K] = "";
            a.data.setAttrObject(I);
            var b = a.indent;
            a.clear(), a.updateHandlers = [];
            for (var s in q) ! function (q) {
                var k = [];
                a.addLabelTooltip(k, q), k.push({
                    id: q,
                    textField: {
                        handleChange: function (k) {
                            a.data.a(q, k)
                        }
                    }
                }), a.addImage(k, function (a) {
                    return a.a(q)
                }, void 0, function (q) {
                    return u(q) ? q.substr(0, q.length - 5) + ".png" : null
                }, function (q) {
                    return u(q.rawIcon)
                }), a.addRow(k, [b, .1, 20]);
                var I = a.getViewById(q);
                w.handleDragAndDrop(I, function (q, a) {
                    return !!a.view.draggingData && "material" === a.view.draggingData.fileType
                }), a.updateHandlers.push(function () {
                    a.data && !k.hidden && I.setValue(X(a.data.a(q)) ? "" : a.data.a(q))
                })
            }(s);
            a.$46$ = a._rows, a.filterProperties();
            var C = a.getRows().length;
            k.height = C ? C * (a.getRowHeight() + a.getHGap()) + 2 * a.getHPadding() : 2, this.inspector.filterPropertiesLater(), a._url = this.inspector.v("url")
        }, a.prototype._updateMatDef = function (q) {
            var a = q.data.getAttrObject(),
                k = this.inspector.data,
                w = {};
            for (var I in a) Z(a[I]) && "" !== a[I] && (w[I] = a[I]);
            k.a("matDef", w), q.filterPropertiesLater()
        }, a.prototype.toJSON = function () {
            var q = this.inspector,
                a = {
                    modelType: q.v("modelType")
                };
            if ("obj" === a.modelType) {
                a.obj = q.v("obj"), a.mtl = q.v("mtl"), a.prefix = q.v("prefix"), a.center = q.v("center");
                q.v("reverseFlipMtls") && (a.reverseFlipMtls = "*"), a.ignoreImage = !!q.v("ignoreImage"), a.ignoreColor = !!q.v("ignoreColor"), a.ignoreNormal = !!q.v("ignoreNormal"), a.s3 = q.v("s3") || [1, 1, 1], a.r3 = q.v("r3") || [0, 0, 0], a.t3 = q.v("t3") || [0, 0, 0], a.image || delete a.image, a.prefix || delete a.prefix, a.center && delete a.center, a.ignoreImage || delete a.ignoreImage, a.ignoreColor || delete a.ignoreColor, a.ignoreNormal || delete a.ignoreNormal, "1,1,1" === a.s3.toString() && delete a.s3, "0,0,0" === a.r3.toString() && delete a.r3, "0,0,0" === a.t3.toString() && delete a.t3
            } else if ("fbx" === a.modelType) {
                a.cube = !0, a.url = q.v("url"), a.center = q.v("center"), a.rotationInterpolation = !!q.v("rotationInterpolation"), a.flipY = !!q.v("flipY"), a.playAutomatically = !!q.v("playAutomatically");
                var k = q.v("matDef"),
                    w = {};
                for (var I in k) Z(k[I]) && "" !== k[I] && (w[I] = k[I]);
                a.matDef = w
            } else if ("gltf" === a.modelType) {
                a.cube = !0, a.url = q.v("url"), a.prefix = q.v("prefix"), a.center = q.v("center"), a.rotationInterpolation = !!q.v("rotationInterpolation"), a.flipY = !!q.v("flipY"), a.playAutomatically = !!q.v("playAutomatically");
                var K = q.v("matDef"),
                    b = {};
                for (var s in K) Z(K[s]) && "" !== K[s] && (b[s] = K[s]);
                a.matDef = b
            }
            return a
        }, a.prototype.ok = function (q) {
            var a = this,
                k = this.toJSON();
            if (k) {
                var w = this.inspector.v("name"),
                    I = this.inspector.v("path");
                if (!w) return void this.showError("name");
                if ("obj" === k.modelType) {
                    var s = this.inspector.v("obj");
                    if (!s || !K(s)) return void this.showError("obj");
                    var u = this.inspector.v("mtl");
                    if (u && !b(u)) return void this.showError("mtl")
                }
                var C = I + w + ".json";
                if (this.newModel && this.editor.models.dataModel.getDataById(C)) return void this.showError("conflict", C);
                this.json = k;
                var L = {
                    path: C,
                    content: o(this.json)
                };
                this.editor.request("upload", L, function (k) {
                    if (a.g3d.dm().setBackground(null), a.editor.saveImage(a.g3d, C.substr(0, C.length - 5) + ".png", function () {
                            a.hide(), a.inspector.v("focusModel") && a.editor.selectFileNode(C)
                        }), q) {
                        var w = new ht.Node;
                        w.s("shape3d", C), w.setAnchor3d([.5, 0, .5]), a.editor.dm.add(w), a.editor.sm.ss(w)
                    }
                })
            }
        }, a.prototype.open = function (q, a) {
            var k = this;
            this.isShowing() || (this.show(), this.__oldHandleModelLoadedFunc__ = ht.Default.handleModelLoaded, ht.Default.handleModelLoaded = function (q, a) {
                if (q === ya) {
                    k.node.s("shape3d", ya), k.node.iv(), k.updateScene(a), k._updating = !0;
                    var w = k.inspector.data;
                    if (a && a.json && ("gltf" === a.json.modelType || "fbx" === a.json.modelType)) {
                        w._rawMatDef = a.matDef;
                        var I = ht.Default.clone(w.a("matDef")) || {},
                            K = {};
                        for (var b in a.matDef) K[b] = I[b] ? I[b] : a.matDef[b];
                        w.a("matDef", K)
                    } else w._rawMatDef = void 0, w.a("matDef", void 0);
                    a && a.json && "gltf" === a.json.modelType && k._updateMaterials(k.inspector.v("matDef"), k.inspector.gltfMaterialsInspector, k.inspector.gltfMaterialsRow), a && a.json && "fbx" === a.json.modelType && k._updateMaterials(k.inspector.v("matDef"), k.inspector.fbxMaterialsInspector, k.inspector.fbxMaterialsRow), void 0 === a && (k._updateMaterials(k.inspector.v("matDef"), k.inspector.gltfMaterialsInspector, k.inspector.gltfMaterialsRow), k._updateMaterials(k.inspector.v("matDef"), k.inspector.fbxMaterialsInspector, k.inspector.fbxMaterialsRow)), k.inspector.filterPropertiesLater(), k._updating = !1
                }
            }, this.__oldHandleMaterialLoadedFunc__ = ht.Default.handleMaterialLoaded, ht.Default.handleMaterialLoaded = function (q, a) {
                k.inspector.filterPropertiesLater()
            }), this.newModel = !q, this.nameInput.setEditable(!q), this.url = q, this.json = a, this.node.p3(0, 0, 0), this.update()
        }, a.prototype.hide = function () {
            q.prototype.hide.call(this), na(ya, void 0), ht.Default.handleModelLoaded = this.__oldHandleModelLoadedFunc__, this.__oldHandleModelLoadedFunc__ = null, ht.Default.handleMaterialLoaded = this.__oldHandleMaterialLoadedFunc__, this.__oldHandleMaterialLoadedFunc__ = null
        }, a.prototype.update = function () {
            var q = this.json,
                a = this.inspector,
                k = a.data;
            if (this._updating = !0, this.url) {
                var w = this.editor.getFileNode(this.url),
                    I = w.s("label");
                q.path = w.path + "/", q.name = I
            }
            q || (q = {}), q.modelType || (q.modelType = "obj"), q.path || (this.editor.models.tree.sm().ld() ? q.path = this.editor.models.tree.sm().ld().url + "/" : q.path = "models/"), q.name || (q.name = ga("editor.untitled")), void 0 === q.center && (q.center = !0), void 0 === q.flipY && (q.flipY = !0), k.setAttrObject(q), this.updateModel(), a.filterPropertiesLater(), delete this._updating
        }, a.prototype.updateModel = function () {
            var q = this.toJSON();
            if ("obj" === q.modelType) q && q.obj ? na(ya, q) : (na(ya, null), ht.Default.handleModelLoaded(ya));
            else if ("fbx" === q.modelType) {
                var a = q.url;
                C(a) ? (q.playAutomatically = !1, na(ya, q)) : (na(ya, null), ht.Default.handleModelLoaded(ya))
            } else if ("gltf" === q.modelType) {
                var k = q.url;
                L(k) || j(k) ? (q.playAutomatically = !1, na(ya, q)) : (na(ya, null), ht.Default.handleModelLoaded(ya))
            }
        }, a.prototype.handleDragAndDrop = function (q, a) {
            function k() {
                I.style.border = K, K = null
            }
            var w = this,
                I = q.getElement();
            q.isDroppable = a;
            var K = void 0;
            q.handleCrossDrag = function (a, b, s) {
                if ("enter" === b) K = I.style.border, I.style.border = "solid " + R.config.color_select_dark + " 2px";
                else if ("exit" === b || "cancel" === b) k();
                else if ("over" === b);
                else if ("drop" === b) {
                    w.isDrop = !0;
                    var u = I.value;
                    I.value = s.view.draggingData.getFileUUID(), k(), q.setFocus(), q.handleChange(I.value, u)
                }
            }, this.handleBlurAndKeydown(I, function (a) {
                q.isEditable() && q.handleChange(a)
            })
        }, a.prototype.handleBlurAndKeydown = function (q, a) {
            q.onblur = function (k) {
                a(q.value)
            }, q.onkeydown = function (k) {
                ht.Default.isEnter(k) && a(q.value)
            }
        }, a.prototype.autoFillInput = function (q, a) {
            if (this.isDrop) {
                if (a.length <= 4) return !0;
                var k = this.inspector,
                    w = a.length,
                    I = a.lastIndexOf("/") + 1,
                    K = a.substr(0, w - 4).substr(I),
                    b = k.v("name");
                k.v("obj"), k.v("mtl"), k.v("prefix");
                return b && b !== ga("editor.untitled") || k.v("name", K), "obj" === q ? k.v("mtl", a.substr(0, w - 3) + "mtl") : "mtl" === q && k.v("obj", a.substr(0, w - 3) + "obj"), k.v("prefix", a.substr(0, I)), this.isDrop = !1, !0
            }
        }, a.prototype.showError = function (q, a) {
            var k = this;
            if (!this.__error__) {
                var w = void 0,
                    I = void 0;
                "name" === q ? (I = ga("editor.error.title"), w = ga("editor.error.name")) : "obj" === q || "mtl" === q ? (I = ga("editor.error.title"), w = ga("editor.error." + q + ".file")) : "conflict" === q && (I = ga("editor.filenameconflict"), w = a), this.__error__ = !0, hteditor.alert(I, w, function () {
                    ht.Default.callLater(function () {
                        return k.__error__ = !1
                    })
                })
            }
        }, a.prototype.clearInput = function () {
            var q = this.inspector;
            q.v("obj", ""), q.v("mtl", ""), q.v("prefix", ""), q.getViewById("url").isEditable() && q.v("url", "")
        }, a.prototype.updateScene = function (q) {
            if (q && q.rawS3) {
                var a = this.g3d,
                    k = q.rawS3[0],
                    w = q.rawS3[1],
                    I = q.rawS3[2],
                    K = Math.max(w, k / (a.getAspect() || 1)),
                    b = K / 2 / Math.tan(a.getFovy() / 2);
                b = 2 * Math.max(I / 2, b), a.setNear(Math.min(10, b / 4)), a.setFar(Math.max(1e4, 4 * b)), a.flyTo(this.node, {
                    animation: !0,
                    direction: [1, .5, 1]
                })
            }
        }, v(a, [{
            key: "buttons",
            get: function () {
                var q = this,
                    a = this._buttons;
                return a || (a = this._buttons = [], a.push({
                    label: ga("editor.ok"),
                    action: function () {
                        q.ok()
                    }
                }), a.push({
                    label: ga("editor.cancel"),
                    action: function () {
                        q.hide()
                    }
                })), a
            }
        }]), a
    }(ht.widget.Dialog);
    ht.Default.setImage("editor.animation.loop.repeat", {
        width: 16,
        height: 16,
        comps: [{
            type: "shape",
            borderWidth: 1,
            borderColor: "#2C2C2C",
            borderCap: "round",
            pixelPerfect: !0,
            points: [12.37868, 1.8808, 14.5, 4.00212, 12.37868, 6.12344, 8, 4.00212, 14.5, 4.00212, 1.5, 8.00212, 1.5, 8.00212, 1.5, 6.89755, 1.89052, 5.95474, 2.67157, 5.17369, 3.45262, 4.39264, 4.39543, 4.00212, 5.5, 4.00212, 8, 4.00212, 3.62132, 14.1192, 1.5, 11.99788, 3.62132, 9.87656, 8, 11.99788, 1.5, 11.99788, 14.5, 7.99788, 14.5, 7.99788, 14.5, 9.10245, 14.10948, 10.04526, 13.32843, 10.82631, 12.54738, 11.60736, 11.60457, 11.99788, 10.5, 11.99788, 8, 11.99788],
            segments: [1, 2, 2, 1, 2, 1, 2, 4, 4, 2, 1, 2, 2, 1, 2, 1, 2, 4, 4, 2]
        }]
    }), ht.Default.setImage("editor.animation.loop.pingpong", {
        width: 16,
        height: 16,
        comps: [{
            type: "shape",
            borderWidth: 1,
            borderColor: "#2C2C2C",
            borderCap: "round",
            pixelPerfect: !0,
            points: [10.44889, 11.65111, 12.5, 9.6, 3.5, 9.6, 5.55111, 4.34889, 3.5, 6.4, 12.5, 6.4],
            segments: [1, 2, 2, 1, 2, 2]
        }]
    }), ht.Default.setImage("editor.animation.loop.once", {
        width: 16,
        height: 16,
        comps: [{
            type: "text",
            text: "1",
            align: "center",
            color: "#2C2C2C",
            scaleX: .56,
            scaleY: .56,
            rect: [7.67332, 8.59384, .001, .001]
        }, {
            type: "shape",
            borderWidth: 1,
            borderColor: "#2C2C2C",
            borderCap: "round",
            pixelPerfect: !0,
            points: [12.37868, 1.8808, 14.5, 4.00212, 12.37868, 6.12344, 8, 4.00212, 14.5, 4.00212, 14.5, 8.00212, 14.5, 8.00212, 14.5, 9.10669, 14.10947, 10.0495, 13.32843, 10.83055, 12.54738, 11.61159, 11.60457, 12.00212, 10.5, 12.00212, 5.5, 12.00212, 4.39543, 12.00212, 3.45262, 11.61159, 2.67157, 10.83055, 1.89052, 10.0495, 1.5, 9.10669, 1.5, 8.00212, 1.5, 8.00212, 1.5, 6.89755, 1.89052, 5.95474, 2.67157, 5.17369, 3.45262, 4.39264, 4.39543, 4.00212, 5.5, 4.00212, 8, 4.00212],
            segments: [1, 2, 2, 1, 2, 1, 2, 4, 4, 2, 4, 4, 2, 4, 4, 2]
        }]
    }), ht.Default.setImage("editor.animation.play", {
        width: 16,
        height: 16,
        comps: [{
            type: "shape",
            borderWidth: 1,
            borderColor: "#2C2C2C",
            borderCap: "round",
            pixelPerfect: !0,
            points: [11.25, 8, 4.75, 12.5, 4.75, 10.62106, 4.75, 8.70087, 4.75, 7.29073, 4.75, 5.39054, 4.75, 3.5, 11.25, 8],
            segments: [1, 2, 2, 1, 2, 1, 2, 2]
        }]
    }), ht.Default.setImage("editor.animation.pause", {
        width: 16,
        height: 16,
        comps: [{
            type: "shape",
            borderWidth: 1,
            borderColor: "#2C2C2C",
            borderCap: "round",
            pixelPerfect: !0,
            points: [10.8, 12, 10.8, 4, 5.2, 12, 5.2, 4],
            segments: [1, 2, 1, 2]
        }]
    }), ht.Default.setImage("editor.animation.stop", {
        modified: "Mon Jun 19 2023 11:47:26 GMT+0800 (GMT+08:00)",
        width: 16,
        height: 16,
        comps: [{
            type: "rect",
            borderWidth: 1,
            borderColor: "#2C2C2C",
            borderJoin: "miter",
            rect: [4, 4, 8, 8]
        }]
    });
    var xa = R.getString,
        za = function (q) {
            function a(k) {
                V(this, a);
                var w = Q(this, q.call(this));
                return w.editor = k, w.animations = void 0, w.duration = 1, w.loop = "repeat", w.formPane = new ht.widget.FormPane, w.currentDurationTextField = new ht.widget.TextField, w.durationTextField = new ht.widget.TextField, w.progressSlider = new ht.widget.Slider, w.progressSlider.setMin(0), w.progressSlider.setMax(1), w.speedSlider = new ht.widget.Slider, w.speedSlider.setMin(0), w.speedSlider.setMax(5), w.animationNamesComboBox = window.an = new ht.widget.ComboBox, w.changeModeButton = hteditor.createButton("", "", "editor.animation.loop.repeat", function () {
                    "repeat" === w.loop ? (w.changeModeButton.setIcon("editor.animation.loop.pingpong"), w.loop = "pingpong") : "pingpong" === w.loop ? (w.changeModeButton.setIcon("editor.animation.loop.once"), w.loop = "once") : "once" === w.loop && (w.changeModeButton.setIcon("editor.animation.loop.repeat"), w.loop = "repeat"), w.ld.isAnimationPlaying() && !w.ld.isAnimationPaused() && (w.ld.pauseAnimation(), w.ld.playAnimation(w.animationNamesComboBox.getValue(), w.speedSlider.getValue(), w.progressSlider.getValue(), w.loop))
                }), w.playButton = hteditor.createButton("", "", "editor.animation.play", function () {
                    w.playButton.play ? (w.playButton.play = !1, w.ld.pauseAnimation(), w.playButton.setIcon("editor.animation.play")) : (w.playButton.play = !0, w.ld.playAnimation(w.animationNamesComboBox.getValue(), w.speedSlider.getValue(), w.progressSlider.getValue(), w.loop), w.playButton.setIcon("editor.animation.pause"))
                }), w.playButton.play = !1, w.stopButton = hteditor.createButton("", "", "editor.animation.stop", function () {
                    w.ld.stopAnimation(), w.playButton.play = !1, w.playButton.setIcon("editor.animation.play")
                }), w.currentSpeedTextField = new ht.widget.TextField, w.maxSpeedTextField = new ht.widget.TextField, w.init(), w.editor.scene.addViewListener(function (q) {
                    "validate" === q.kind && (w.update(), w.updateAnimationProgressBar())
                }), w.editor.dm.md(function (q) {
                    "animationIteration" === q.property && "once" === q.data.getCurrentAnimationState("loop") && (q.data.playAnimation(q.data.getCurrentAnimationState("name"), q.data.getCurrentAnimationState("speed"), 0, q.data.getCurrentAnimationState("loop")), q.data.pauseAnimation(), q.data === w.ld && (w.playButton.play = !1, w.playButton.setIcon("editor.animation.play")))
                }), w
            }
            return E(a, q), a.prototype.init = function () {
                var q = this;
                this.setConfig({
                    title: '<span style="margin-left: 8px">' + xa("editor.animation.title") + "</span>",
                    draggable: !0,
                    closable: !1,
                    width: 600,
                    height: 96,
                    borderWidth: 0,
                    content: this.formPane,
                    resizeMode: "none",
                    titleHeight: 30,
                    titleBackground: ht.Default.dialogHeaderBackground,
                    titleColor: "#303033",
                    contentPadding: 0
                }), this.formPane.setVGap(8), this.formPane.setPadding(8), this.formPane.setRowHeight(20), this.formPane.addRow([this.currentDurationTextField, this.progressSlider, this.durationTextField], [36, .1, 36]), this.formPane.addRow([this.animationNamesComboBox, "", this.changeModeButton, this.playButton, this.stopButton, "", this.currentSpeedTextField, this.speedSlider, this.maxSpeedTextField], [192 + 2 * this.formPane.getVGap(), .1, 20, 20, 20, .1, 36, 120, 36]), this.setTextFieldStyle(this.currentDurationTextField), this.setTextFieldStyle(this.durationTextField), this.setTextFieldStyle(this.currentSpeedTextField), this.setTextFieldStyle(this.maxSpeedTextField), this.progressSlider.onValueChanged = function (a, k) {
                    q.currentDurationTextField.setValue(k.toFixed(3)), q.progressSlider._progressEditing && (q.ld.playAnimation(q.animationNamesComboBox.getValue(), q.speedSlider.getValue(), k, q.loop), q.ld.pauseAnimation())
                }, this.progressSlider.onBeginValueChanged = function () {
                    q.progressSlider._progressEditing = !0
                }, this.progressSlider.onEndValueChanged = function () {
                    q.progressSlider._progressEditing = !1, q.playButton.play && q.ld.resumeAnimation()
                }, this.progressSlider.getToolTip = function () {
                    return q.progressSlider.getValue().toFixed(3)
                }, this.speedSlider.onValueChanged = function (a, k) {
                    q.currentSpeedTextField.setValue(k.toFixed(2) + "x"), q.ld.changeCurrentAnimationSpeed(Number(k.toFixed(2)))
                }, this.speedSlider.getToolTip = function () {
                    return q.speedSlider.getValue().toFixed(2)
                }, this.animationNamesComboBox.onValueChanged = function (a, k) {
                    if (!q.animationNamesComboBox._update) {
                        var w = q.ld.getAnimation(k);
                        q.durationTextField.setValue(w.duration.toFixed(3)), q.currentDurationTextField.setValue("0.000"), q.progressSlider.setMax(w.duration), q.progressSlider.setValue(0), q.ld.isAnimationPlaying() && (q.ld.isAnimationPaused() ? (q.ld.playAnimation(k, q.speedSlider.getValue(), q.progressSlider.getValue(), q.loop), q.ld.pauseAnimation()) : q.ld.playAnimation(k, q.speedSlider.getValue(), q.progressSlider.getValue(), q.loop))
                    }
                }, this.maxSpeedTextField.setValue("5.00x")
            }, a.prototype.setTextFieldStyle = function (q) {
                q.setDisabled(!0), q.setBorder("0px"), q.getDisabledDiv().style.backgroundColor = "rgba(255, 255, 255, 0)"
            }, a.prototype.showModelAnimDialog = function (q) {
                if (q) {
                    var a = this.getConfig(),
                        k = this.editor.mainSplitView.getWidth(),
                        w = this.editor.mainSplitView.getHeight();
                    a.position = {
                        x: (k + this.editor.leftSplitView.getPosition() + this.editor.mainSplitView.getPosition()) / 2 - 300,
                        y: w - 130
                    }, this.setConfig(a), this.show(), this.addToDOM(this.editor.mainSplitView.getView()), this.getView().childNodes[1].style.position = "absolute", this.getView().style.width = "0px", this.getView().style.height = "0px"
                } else this.hide()
            }, a.prototype.update = function () {
                var q = this,
                    a = void 0;
                if (this.ld && this.ld.getAnimations && (a = this.ld.getAnimations()), this.lastData !== this.ld || a !== this.animations) {
                    this.animations = a, this.lastData = this.ld;
                    var k = !!(a && a.length > 0);
                    if (this.showModelAnimDialog(k), k) {
                        if (this.animationNamesComboBox._update = !0, this.animationNamesComboBox.setValues(this.ld.getAnimationNames() || []), ht.Default.callLater(function () {
                                q.updateDropDownWidth()
                            }), this.ld.isAnimationPlaying()) {
                            this.ld.isAnimationPaused() ? (this.playButton.play = !1, this.playButton.setIcon("editor.animation.play")) : (this.playButton.play = !0, this.playButton.setIcon("editor.animation.pause"));
                            var w = this.ld.getCurrentAnimationState();
                            "repeat" === w.loop ? (this.changeModeButton.setIcon("editor.animation.loop.repeat"), this.loop = "repeat") : "pingpong" === w.loop ? (this.changeModeButton.setIcon("editor.animation.loop.pingpong"), this.loop = "pingpong") : "once" === w.loop && (this.changeModeButton.setIcon("editor.animation.loop.once"), this.loop = "once"), this.animationNamesComboBox.setValue(w.name), this.durationTextField.setValue(w.duration.toFixed(3)), this.currentDurationTextField.setValue(w.time), this.progressSlider.setMax(w.duration), this.progressSlider.setValue(w.time), this.currentSpeedTextField.setValue(w.speed.toFixed(2) + "x"), this.speedSlider.setValue(w.speed)
                        } else {
                            this.playButton.play = !1, this.changeModeButton.setIcon("editor.animation.loop.repeat"), this.loop = "repeat", this.playButton.setIcon("editor.animation.play");
                            var I = this.ld.getDefaultAnimationName();
                            this.animationNamesComboBox.setValue(I);
                            var K = this.ld.getAnimation(I);
                            this.durationTextField.setValue(K.duration.toFixed(3)), this.currentDurationTextField.setValue("0.000"), this.progressSlider.setMax(K.duration), this.progressSlider.setValue(0), this.currentSpeedTextField.setValue("1.00x"), this.speedSlider.setValue(1)
                        }
                        this.animationNamesComboBox._update = !1
                    }
                }
            }, a.prototype.updateAnimationProgressBar = function () {
                if (this.ld && !this.progressSlider._progressEditing && this.isShowing()) {
                    var q = this.ld.getCurrentAnimationState("time");
                    this.progressSlider.setValue(q)
                }
            }, a.prototype.updateDropDownWidth = function () {
                var q = this,
                    a = this.animationNamesComboBox.getWidth();
                this.animationNamesComboBox.getValues().forEach(function (k) {
                    var w = ht.Default.getTextSize(q.animationNamesComboBox.getLabelFont(), k).width;
                    w > a && (a = w)
                }), this.animationNamesComboBox.setDropDownWidth(a)
            }, v(a, [{
                key: "ld",
                get: function () {
                    var q = this.editor.ld;
                    if (q && q.s("static")) {
                        var a = this.editor.scene,
                            k = a.getRenderLists().get(q.getRenderLayer()),
                            w = k.getModel3dInstancedBatchMap(),
                            I = q.getId(),
                            K = void 0;
                        for (var b in w) {
                            if (K = w[b].ds, K.indexOf(I) >= 0) break;
                            K = null
                        }
                        K && (q = a.dm().getDataById(K[0]))
                    }
                    return q
                }
            }]), a
        }(ht.widget.Dialog),
        Va = hteditor.getString,
        va = hteditor.getter,
        Ea = hteditor.setter,
        Qa = ht.Default.clone,
        la = function (q) {
            function a(k) {
                V(this, a);
                var w = Q(this, q.call(this));
                return w.editor = k, w.currentModel = "cube", R.config.materialViewModel && (w.currentModel = "logo"), w.newMaterial = !1, w.uuid = ht.Math.generateUUID(), w.materialDefaultStyle = Object.assign(Qa(ht.Default.getMaterialDefaultStyle().phong), Qa(ht.Default.getMaterialDefaultStyle().pbr)), w.init(), w
            }
            return E(a, q), a.prototype.init = function () {
                var q = this.initG3d(),
                    a = this.initInspector();
                a.filterPropertiesLater();
                var k = this._mainView = new ht.widget.SplitView(a, q, "h", 300);
                this.setConfig({
                    title: '<span style="margin-left: 20px">' + Va("editor.material.title.edit") + "</span>",
                    width: R.config.materialViewSize.width,
                    height: R.config.materialViewSize.height,
                    maximizable: !0,
                    content: k,
                    resizeMode: "wh",
                    buttonsAlign: "right",
                    buttons: this.buttons,
                    closable: !0,
                    draggable: !0,
                    titleHeight: 38,
                    titleBackground: ht.Default.dialogHeaderBackground,
                    titleColor: "#303033",
                    titleIconGap: 16,
                    borderWidth: 0,
                    contentPadding: 0
                }), this.setModal(!1)
            }, a.prototype.initG3d = function () {
                return this.dm = new ht.DataModel, this.node = new ht.Node, this.node.s({
                    "select.brightness": 1,
                    "envmap.probe": -1
                }), this.node.setAnchor3d(.5, 0, .5), this.node.p3(0, 0, 0), this.dm.add(this.node), this.g3d = new ht.graph3d.Graph3dView(this.dm), this.g3d.setGridVisible(!0), this.g3d.setGridColor("rgb(191, 191, 191)"), this.g3d.setEye(0, 600, 1400), this.g3d.setCenter(0, 250, 0), this.g3d.setNear(10), this.g3d.setFar(1e4), this.g3d.setSelectableFunc(function () {
                    return !1
                }), this.skyBox || (this.skyBox = new ht.Node), this.skyBox.s({
                    shape3d: "sphere",
                    "shape3d.color": "rgba(0, 0, 0,0)"
                }), this.g3d.setSkyBox(this.skyBox), this.g3d
            }, a.prototype.initInspector = function () {
                var q = this,
                    a = this.inspector = new Kq(this.editor, "material", !1);
                a.getView().style.background = "none", a.setPadding(6), a.v = function () {
                    if (this.data) {
                        var q = arguments[0],
                            a = arguments[1];
                        if (1 === arguments.length) return this.data.a(q);
                        this.data.a(q, a)
                    }
                }, a.getValue = function (q) {
                    if (this.data) return q(this.data)
                }, a.setValue = function (q, a, k) {
                    this.data && q(this.data, k ? k() : a)
                };
                var k = a.addTitle;
                a.addTitle = function (q) {
                    return R.config.expandedTitles[q] = !0, k.apply(this, arguments)
                };
                var w = new ht.Data;
                return w.onPropertyChanged = function (k) {
                    if (!w._updatingObjProperties && !q._updating) {
                        var I = k.property;
                        if (I.startsWith("a:")) {
                            var K = k.newValue;
                            "a:type" === I && a.filterPropertiesLater(), "a:envMap" === I && q.g3d.getSkyBox().s("shape3d.image", K), q.refresh(), a.updateProperties()
                        }
                    }
                }, a.data = w, this._initBasicProperties(), this._initProperties(), a
            }, a.prototype._initBasicProperties = function () {
                var q = this,
                    a = this.inspector,
                    k = this.inspector.indent,
                    w = [];
                if (R.config.materialViewModel) {
                    var I = hteditor.createButton(null, "", "editor.material.logo", function () {
                        q.changeModel("logo")
                    });
                    I.getIconColor = function () {
                        return "logo" === q.currentModel ? R.config.color_select : R.config.color_dark
                    }, w.push(I)
                }
                var K = hteditor.createButton(null, Va("editor.cube"), "editor.cube", function () {
                    q.changeModel("cube")
                });
                K.getIconColor = function () {
                    return "cube" === q.currentModel ? R.config.color_select : R.config.color_dark
                }, w.push(K);
                var b = hteditor.createButton(null, Va("editor.sphere"), "editor.sphere", function () {
                    q.changeModel("sphere")
                });
                b.getIconColor = function () {
                    return "sphere" === q.currentModel ? R.config.color_select : R.config.color_dark
                }, w.push(b), a.addRow(w, [20, 20, 20]), a.addTitle(Va("editor.material.inspector.title.basic")), w = [], a.addLabelInput(w, Va("editor.path"), va("a", "path"), Ea("a", "path")).setEditable(!1), a.addRow(w, [k, .1]), w = [], this.inspector.nameInput = a.addLabelInput(w, Va("editor.name"), va("a", "name"), Ea("a", "name")), a.addRow(w, [k, .1]), w = [], a.addLabelComboBox(w, Va("editor.type"), va("a", "type"), Ea("a", "type"), ["pbr", "phong"], ["pbr", "phong"]), a.addLabelComboBox(w, Va("editor.material.clipMode"), function (q) {
                    return q.a("cullFace") ? q.a("flipSide") ? "front" : "back" : "none"
                }, function (q, a) {
                    "front" === a ? (q.a("cullFace", !0), q.a("flipSide", !0)) : "back" === a ? (q.a("cullFace", !0), q.a("flipSide", !1)) : (q.a("cullFace", !1), q.a("flipSide", !1))
                }, ["none", "front", "back"], [Va("editor.material.clipMode.none"), Va("editor.material.clipMode.front"), Va("editor.material.clipMode.back")]), a.addRow(w, [k, .1, k, .1]), w = [], a.addLabelInput(w, Va("editor.material.fresnelIntensity"), va("a", "fresnelIntensity"), Ea("a", "fresnelIntensity"), "number", .01), a.addRow(w, [k, .1]), a.addTitle(Va("editor.material.inspector.title.uv")), w = [], a.addLabelSlider(w, Va("editor.uvrotation"), function (q) {
                    return Number((180 / Math.PI * q.a("uvRotation")).toFixed(.2))
                }, function (q, a) {
                    q.a("uvRotation", a * Math.PI / 180)
                }, 0, 360, .01), a.addInput(w, function (q) {
                    return Number((180 / Math.PI * q.a("uvRotation")).toFixed(.2))
                }, function (q, a) {
                    q.a("uvRotation", a * Math.PI / 180)
                }, "number", .01), a.addRow(w, [k, .1, 76]), w = [], w.push(Va("editor.uvscale")), a.addLabelInput(w, "U", function (q) {
                    var a = q.a("uvScale");
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.a("uvScale"));
                    k[0] = Number(a), q.a("uvScale", k)
                }, "number", .01), a.addLabelInput(w, "V", function (q) {
                    var a = q.a("uvScale");
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.a("uvScale"));
                    k[1] = Number(a), q.a("uvScale", k)
                }, "number", .01), a.addRow(w, [k - 20 - 4, 20, .1, 20, .1]), w = [], w.push(Va("editor.uvoffset")), a.addLabelInput(w, "U", function (q) {
                    var a = q.a("uvOffset");
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.a("uvOffset"));
                    k[0] = Number(a), q.a("uvOffset", k)
                }, "number", .01), a.addLabelInput(w, "V", function (q) {
                    var a = q.a("uvOffset");
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.a("uvOffset"));
                    k[1] = Number(a), q.a("uvOffset", k)
                }, "number", .01), a.addRow(w, [k - 20 - 4, 20, .1, 20, .1]), w = [], w.push(Va("editor.uvanchor")), a.addLabelInput(w, "U", function (q) {
                    var a = q.a("uvAnchor");
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.a("uvAnchor"));
                    k[0] = Number(a), q.a("uvAnchor", k)
                }, "number", .01), a.addLabelInput(w, "V", function (q) {
                    var a = q.a("uvAnchor");
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = ht.Default.clone(q.a("uvAnchor"));
                    k[1] = Number(a), q.a("uvAnchor", k)
                }, "number", .01), a.addRow(w, [k - 20 - 4, 20, .1, 20, .1])
            }, a.prototype._initProperties = function () {
                var q = this,
                    a = this.inspector,
                    k = this.inspector.indent;
                a.addTitle(Va("editor.material.inspector.title.diffuse"));
                var w = [];
                a.addLabelColor(w, Va("editor.color"), function (q) {
                        var a = q.a("diffuse");
                        return "rgb(" + (255 * a[0]).toFixed(0) + "," + (255 * a[1]).toFixed(0) + "," + (255 * a[2]).toFixed(0) + ")"
                    }, function (a, k) {
                        if (k) {
                            var w = ht.Default.toColorData(k);
                            a.a("diffuse", [w[0] / 255, w[1] / 255, w[2] / 255])
                        } else a.a("diffuse", q.materialDefaultStyle.diffuse)
                    }), a.addRow(w, [k, .1]), w = [], a.addLabelImage(w, Va("editor.material.inspector.image"), function (a) {
                        var k = a.a("map"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("map", a)
                    }), a.addRow(w, [k, .1, 20]), a.addTitle(Va("editor.material.inspector.title.roughness")).visible = function () {
                        return "pbr" === a.v("type")
                    }, w = [], a.addLabelImage(w, Va("editor.material.inspector.image"), function (a) {
                        var k = a.a("roughnessMap"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("roughnessMap", a)
                    }), a.addRow(w, [k, .1, 20]), w = [], a.addLabelInput(w, Va("editor.material.roughness"), va("a", "roughness"), Ea("a", "roughness"), "number", .01), a.addLabelComboBox(w, Va("editor.material.inspector.channel"), va("a", "roughnessChannel"), Ea("a", "roughnessChannel"), ["R", "G", "B"], ["R", "G", "B"]), a.addRow(w, [k, .1, k, .1]), a.addTitle(Va("editor.material.inspector.title.metalness")).visible = function () {
                        return "pbr" === a.v("type")
                    }, w = [], a.addLabelImage(w, Va("editor.material.inspector.image"), function (a) {
                        var k = a.a("metalnessMap"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("metalnessMap", a)
                    }), a.addRow(w, [k, .1, 20]), w = [], a.addLabelInput(w, Va("editor.material.metalness"), va("a", "metalness"), Ea("a", "metalness"), "number", .01), a.addLabelComboBox(w, Va("editor.material.inspector.channel"), va("a", "metalnessChannel"), Ea("a", "metalnessChannel"), ["R", "G", "B"], ["R", "G", "B"]), a.addRow(w, [k, .1, k, .1]), a.addTitle(Va("editor.material.inspector.shininess")).visible = function () {
                        return "phong" === a.v("type")
                    }, w = [], a.addLabelInput(w, Va("editor.material.inspector.shininess"), va("a", "shininess"), Ea("a", "shininess"), "number", .01), a.addRow(w, [k, .1]), a.addTitle(Va("editor.material.inspector.title.specular")).visible = function () {
                        return "phong" === a.v("type")
                    }, w = [], a.addLabelColor(w, Va("editor.color"), function (q) {
                        var a = q.a("specular");
                        return "rgb(" + (255 * a[0]).toFixed(0) + "," + (255 * a[1]).toFixed(0) + "," + (255 * a[2]).toFixed(0) + ")"
                    }, function (a, k) {
                        if (k) {
                            var w = ht.Default.toColorData(k);
                            a.a("specular", [w[0] / 255, w[1] / 255, w[2] / 255])
                        } else a.a("specular", q.materialDefaultStyle.specular)
                    }), a.addRow(w, [k, .1]), w = [], a.addLabelImage(w, Va("editor.material.inspector.image"), function (a) {
                        var k = a.a("specularMap"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("specularMap", a)
                    }), a.addRow(w, [k, .1, 20]), a.addTitle("editor.material.inspector.title.ao"), w = [], a.addLabelInput(w, Va("editor.material.inspector.intensity"), va("a", "aoMapIntensity"), Ea("a", "aoMapIntensity"), "number", .01), a.addRow(w, [k, .1]), w = [], a.addLabelImage(w, Va("editor.material.inspector.image"), function (a) {
                        var k = a.a("aoMap"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("aoMap", a)
                    }), a.addRow(w, [k, .1, 20]), w = [], a.addLabelComboBox(w, Va("editor.material.inspector.channel"), va("a", "aoChannel"), Ea("a", "aoChannel"), ["R", "G", "B"], ["R", "G", "B"]), a.addLabelComboBox(w, Va("editor.material.inspector.uvchannel"), va("a", "aoUvChannel"), Ea("a", "aoUvChannel"), [1, 2], [1, 2]), a.addRow(w, [k, .1, k, .1]), a.addTitle(Va("editor.material.inspector.title.normal")), w = [], a.addLabelImage(w, Va("editor.material.inspector.image"), function (a) {
                        var k = a.a("normalMap"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("normalMap", a)
                    }), a.addRow(w, [k, .1, 20]), w = [], a.addLabelRange(w, Va("editor.material.inspector.intensity"), function (q) {
                        return q.a("normalScale")[0]
                    }, function (q, a) {
                        var k = Qa(q.a("normalScale")),
                            w = 1;
                        k[1] < 0 && (w = -1), q.a("normalScale", [a, a * w])
                    }, 0, void 0, .01, "number"), a.addLabelCheckBox(w, Va("editor.material.inspector.flipy"), function (q) {
                        return q.a("normalScale")[1] < 0
                    }, function (q, a) {
                        var k = Qa(q.a("normalScale")),
                            w = 1;
                        a && (w = -1), q.a("normalScale", [k[0], k[0] * w])
                    }), a.addRow(w, [k, .1, k, 20]), a.addTitle(Va("editor.material.inspector.title.emissive")), w = [], a.addLabelColor(w, Va("editor.color"), function (q) {
                        var a = q.a("emissive");
                        return "rgb(" + (255 * a[0]).toFixed(0) + "," + (255 * a[1]).toFixed(0) + "," + (255 * a[2]).toFixed(0) + ")"
                    }, function (a, k) {
                        if (k) {
                            var w = ht.Default.toColorData(k);
                            a.a("emissive", [w[0] / 255, w[1] / 255, w[2] / 255])
                        } else a.a("emissive", q.materialDefaultStyle.emissive)
                    }), a.addRow(w, [k, .1]), w = [], a.addLabelImage(w, Va("editor.material.inspector.image"), function (a) {
                        var k = a.a("emissiveMap"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("emissiveMap", a)
                    }), a.addRow(w, [k, .1, 20]), a.addTitle(Va("editor.material.inspector.title.alpha")), w = [], a.addLabelSlider(w, Va("editor.alphaTest"), va("a", "alphaTest"), Ea("a", "alphaTest"), 0, 1, .01), a.addInput(w, va("a", "alphaTest"), Ea("a", "alphaTest"), "number", .01), a.addRow(w, [k, .1, 76]), w = [], a.addLabelCheckBox(w, Va("editor.transparent"), va("a", "transparent"), Ea("a", "transparent")), a.addRow(w, [k, .1]), w = [], a.addLabelSlider(w, Va("editor.material.opacity"), va("a", "opacity"), Ea("a", "opacity"), 0, 1, .001), a.addInput(w, va("a", "opacity"), Ea("a", "opacity"), "number", .01), a.addRow(w, [k, .1, 76]), w = [], a.addLabelImage(w, Va("editor.material.inspector.image"), function (a) {
                        var k = a.a("alphaMap"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("alphaMap", a)
                    }), a.addRow(w, [k, .1, 20]), w = [], a.addLabelComboBox(w, Va("editor.material.inspector.channel"), va("a", "alphaChannel"), Ea("a", "alphaChannel"), ["R", "G", "B", "A"], ["R", "G", "B", "A"]), a.addRow(w, [k, .1]), a.addTitle(Va("editor.material.inspector.title.light")), w = [], a.addLabelInput(w, Va("editor.material.inspector.intensity"), va("a", "lightMapIntensity"), Ea("a", "lightMapIntensity"), "number", .01), a.addRow(w, [k, .1]), w = [], a.addLabelImage(w, Va("editor.material.inspector.image"), function (a) {
                        var k = a.a("lightMap"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("lightMap", a)
                    }), a.addRow(w, [k, .1, 20]), a.addTitle(Va("editor.material.inspector.title.rim")).visible = function () {
                        return "pbr" === a.v("type")
                    }, w = [], a.addLabelColor(w, Va("editor.color"), function (q) {
                        var a = q.a("rimColor");
                        return "rgb(" + (255 * a[0]).toFixed(0) + "," + (255 * a[1]).toFixed(0) + "," + (255 * a[2]).toFixed(0) + ")"
                    }, function (a, k) {
                        if (k) {
                            var w = ht.Default.toColorData(k);
                            a.a("rimColor", [w[0] / 255, w[1] / 255, w[2] / 255])
                        } else a.a("rimColor", q.materialDefaultStyle.rimColor)
                    }), a.addRow(w, [k, .1]), w = [], a.addLabelInput(w, Va("editor.material.rimPower"), va("a", "rimPower"), Ea("a", "rimPower"), "number", .01), a.addRow(w, [k, .1]), a.addTitle(Va("editor.material.inspector.title.transmission")).visible = function () {
                        return "pbr" === a.v("type")
                    }, w = [], a.addLabelCheckBox(w, Va("editor.enable"), va("a", "enableTransmission"), Ea("a", "enableTransmission")), a.addRow(w, [k, .1]), w = [], a.addLabelSlider(w, Va("editor.material.transmission"), va("a", "transmission"), Ea("a", "transmission"), 0, 3, .001), a.addInput(w, va("a", "transmission"), Ea("a", "transmission"), "number", .001), a.addRow(w, [k, .1, 76]), w = [], a.addLabelSlider(w, Va("editor.thickness"), va("a", "thickness"), Ea("a", "thickness"), 0, 3, .001), a.addInput(w, va("a", "thickness"), Ea("a", "thickness"), "number", .001), a.addRow(w, [k, .1, 76]), w = [], a.addLabelSlider(w, Va("editor.material.ior"), va("a", "ior"), Ea("a", "ior"), 0, 5, .001), a.addInput(w, va("a", "ior"), Ea("a", "ior"), "number", .001), a.addRow(w, [k, .1, 76]), w = [], a.addLabelSlider(w, Va("editor.material.specular.intensity"), va("a", "specularIntensity"), Ea("a", "specularIntensity"), 0, 3, .001),
                    a.addInput(w, va("a", "specularIntensity"), Ea("a", "specularIntensity"), "number", .001), a.addRow(w, [k, .1, 76]), w = [], a.addLabelColor(w, Va("editor.material.specular.color"), function (q) {
                        var a = q.a("specularColor");
                        return "rgb(" + (255 * a[0]).toFixed(0) + "," + (255 * a[1]).toFixed(0) + "," + (255 * a[2]).toFixed(0) + ")"
                    }, function (a, k) {
                        if (k) {
                            var w = ht.Default.toColorData(k);
                            a.a("specularColor", [w[0] / 255, w[1] / 255, w[2] / 255])
                        } else a.a("specularColor", q.materialDefaultStyle.specularColor)
                    }), a.addRow(w, [k, .1]), a.addTitle("editor.material.inspector.title.env"), w = [], a.addLabelComboBox(w, Va("editor.mode"), va("a", "envMapCombine"), Ea("a", "envMapCombine"), ["multiply", "mix", "add"], [Va("editor.material.envmapcombine.multiply"), Va("editor.material.envmapcombine.mix"), Va("editor.material.envmapcombine.add")]), a.addRow(w, [k, .1]).visible = function () {
                        return "phong" === a.v("type")
                    }, w = [], a.addLabelInput(w, Va("editor.material.inspector.intensity"), va("a", "envMapIntensity"), Ea("a", "envMapIntensity"), "number", .01), a.addRow(w, [k, .1]), w = [], a.addLabelImage(w, Va("editor.material.inspector.image"), function (a) {
                        var k = a.a("envMap"),
                            w = q.editor.getFileNode(k);
                        return w ? w.url : k
                    }, function (q, a) {
                        return q.a("envMap", a)
                    }), a.addRow(w, [k, .1, 20])
            }, a.prototype.refresh = function () {
                ht.Default.setMaterial(this.url, this.toJSON()), this.g3d.invalidateAll(), this.editor.scene.invalidateAll()
            }, a.prototype.reload = function () {
                this.json = Qa(this.oldJson), this.update()
            }, a.prototype.toJSON = function () {
                var q = Qa(this.inspector.data.getAttrObject());
                delete q.path, delete q.name, q.cullFace || delete q.cullFace, q.flipSide || delete q.flipSide, q.transparent || delete q.transparent, 0 === q.uvRotation && delete q.uvRotation, "1,1" === q.uvScale.toString() && delete q.uvScale, "0,0" === q.uvOffset.toString() && delete q.uvOffset, "0,0" === q.uvAnchor.toString() && delete q.uvAnchor;
                var a = Qa(this.materialDefaultStyle);
                for (var k in q) a[k] === q[k] && delete q[k], X(a[k]) && a[k].toString() === q[k].toString() && delete q[k];
                return delete q.uUvMatrix, q
            }, a.prototype.ok = function () {
                var q = this,
                    a = this.toJSON(),
                    k = this.inspector.data.a("path"),
                    w = this.inspector.data.a("name"),
                    I = k + "/" + w + ".json";
                if (a) {
                    if (!w) return void this.showError("empty");
                    if (this.newMaterial && this.editor.materials.dataModel.getDataById(I)) return void this.showError("conflict", I);
                    this.json = Qa(a), this.oldJson = Qa(a);
                    var K = {
                        path: I,
                        content: o(a)
                    };
                    this.editor.request("upload", K, function (a) {
                        q.hide();
                        var k = q.skyBox.s("shape3d.image"),
                            w = Qa(q.g3d.getEye()),
                            K = Qa(q.g3d.getCenter());
                        q.g3d.setSkyBox(void 0), q.g3d.setEye(0, 300, 1e3), q.g3d.setCenter(0, 250, 0), q.node.s("shape3d", "sphere"), q.node.setSize3d(500, 500, 500), q.editor.saveImage(q.g3d, I.substr(0, I.length - 5) + ".png", function () {
                            q.editor.selectFileNode(I), q.changeModel(q.currentModel), q.g3d.setSkyBox(q.skyBox), q.g3d.getSkyBox().s("shape3d.image", k), q.g3d.setEye(w), q.g3d.setCenter(K)
                        })
                    })
                }
            }, a.prototype.open = function (q, a, k, w) {
                if (this.isShowing() && this.reload(), this.show(), q) {
                    this.path = k, this.name = w.replace(".json", ""), this.url = q, this.json = a, this.oldJson = Qa(a), this.newMaterial = !1;
                    ht.Default.getMaterial(this.url) && ht.Default.setMaterial(q, a)
                } else {
                    this.path = this.editor.materials.currentDir, this.name = Va("editor.untitled"), this.url = this.uuid, this.newMaterial = !0;
                    var I = this.editor.getFileNode(R.config.materialViewEnvMap) ? R.config.materialViewEnvMap : void 0;
                    this.json = {
                        type: "pbr",
                        envMap: I
                    }, this.oldJson = Qa(this.json), ht.Default.setMaterial(this.url, this.json)
                }
                this.inspector.nameInput.setEditable(this.newMaterial), this.changeModel(this.currentModel), this.update(), this.g3d.moveCamera([0, 600, 1400], [0, 250, 0], !0)
            }, a.prototype.hide = function () {
                this.reload(), q.prototype.hide.call(this)
            }, a.prototype.update = function () {
                this.node.s("shape3d.material", this.url), this.g3d.getSkyBox().s("shape3d.image", this.json.envMap);
                var q = this.json,
                    a = this.inspector,
                    k = a.data;
                q || (q = {}), q.type || (q.type = "pbr");
                var w = Qa(this.materialDefaultStyle);
                w.cullFace = !1, w.flipSide = !1, w.transparent = !1, w.uvRotation = 0, w.uvScale = [1, 1], w.uvOffset = [0, 0], w.uvAnchor = [0, 0], w.path = this.path, w.name = this.name;
                for (var I in q) w[I] = q[I];
                k.setAttrObject(w), a.filterPropertiesLater(), this.refresh()
            }, a.prototype.handleDragAndDrop = function (q, a) {
                function k() {
                    w.style.border = I, I = null
                }
                var w = q.getElement();
                q.isDroppable = a;
                var I = void 0;
                q.handleCrossDrag = function (a, K, b) {
                    if ("enter" === K) I = w.style.border, w.style.border = "solid " + R.config.color_select_dark + " 2px";
                    else if ("exit" === K || "cancel" === K) k();
                    else if ("over" === K);
                    else if ("drop" === K) {
                        var s = w.value;
                        w.value = b.view.draggingData.getFileUUID(), k(), q.setFocus(), q.handleChange(w.value, s)
                    }
                }, this.handleBlurAndKeydown(w, function (a) {
                    q.handleChange(a)
                })
            }, a.prototype.handleBlurAndKeydown = function (q, a) {
                q.onblur = function (k) {
                    a(q.value)
                }, q.onkeydown = function (k) {
                    ht.Default.isEnter(k) && a(q.value)
                }
            }, a.prototype.showError = function (q, a) {
                var k = this;
                if (!this.__error__) {
                    var w = void 0,
                        I = void 0;
                    "empty" === q ? (I = Va("editor.error.title"), w = Va("editor.material.error.name")) : "conflict" === q && (I = Va("editor.filenameconflict"), w = a), this.__error__ = !0, hteditor.alert(I, w, function () {
                        ht.Default.callLater(function () {
                            return k.__error__ = !1
                        })
                    })
                }
            }, a.prototype.changeModel = function (q) {
                this.inspector.iv(), this.currentModel = q, "cube" === q ? (this.node.s("shape3d", "box"), this.node.setSize3d(450, 450, 450)) : "sphere" === q ? (this.node.s("shape3d", "sphere"), this.node.setSize3d(500, 500, 500)) : (this.node.s("shape3d", {
                    modelType: "obj",
                    obj: R.config.materialViewModel
                }), this.node.setSize3d(500, 500, 80))
            }, v(a, [{
                key: "buttons",
                get: function () {
                    var q = this,
                        a = this._buttons;
                    return a || (a = this._buttons = [], a.push({
                        label: Va("editor.reload"),
                        action: function () {
                            q.g3d.moveCamera([0, 600, 1400], [0, 250, 0], !0), q.reload()
                        }
                    }), a.push({
                        label: Va("editor.ok"),
                        action: function () {
                            q.ok()
                        }
                    }), a.push({
                        label: Va("editor.cancel"),
                        action: function () {
                            q.hide()
                        }
                    })), a
                }
            }]), a
        }(ht.widget.Dialog),
        _a = hteditor.getString,
        Fa = function (q) {
            function a(k) {
                return V(this, a), Q(this, q.call(this, k, "Block"))
            }
            return E(a, q), a.prototype.$19$ = function () {}, a.prototype.$11$ = function () {
                var q = this;
                this.addTitle("TitleBasic"), this.addEventProperties();
                var a = [];
                this.addLabelInput(a, _a("editor.name"), hteditor.getter("p", "displayName"), hteditor.setter("p", "displayName")), this.addLabelInput(a, _a("editor.tag"), hteditor.getter("p", "tag"), function (a, k) {
                    if (R.config.checkTagConflicts && k)
                        for (var w = q.dataModel.getDatas(), I = w.size() - 1; I >= 0; I--) {
                            var K = w.get(I);
                            if (K !== a && K.getTag() === k) return q.editor.messageView.show(m(_a("editor.errormessage.tagalreadyexists"), k), "error"), void q.$34$()
                        }
                    a.setTag(k)
                }), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelInput(a, _a("editor.tooltip"), hteditor.getter("p", "toolTip"), hteditor.setter("p", "toolTip")), this.addRow(a, [this.indent, .1]), a = [], this.addLabelCheckBox(a, _a("editor.selectable"), function (q) {
                    return q.s("3d.selectable")
                }, function (q, a) {
                    q.s("2d.selectable", a), q.s("3d.selectable", a)
                }), this.addLabelCheckBox(a, _a("editor.movable"), function (q) {
                    return q.s("3d.movable")
                }, function (q, a) {
                    q.s("2d.movable", a), q.s("3d.movable", a)
                }), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelCheckBox(a, _a("editor.editable"), function (q) {
                    return q.s("3d.editable")
                }, function (q, a) {
                    q.s("2d.editable", a), q.s("3d.editable", a)
                }), this.addLabelCheckBox(a, _a("editor.visible"), function (q) {
                    return q.s("3d.visible")
                }, function (q, a) {
                    q.s("2d.visible", a), q.s("3d.visible", a)
                }), this.addRow(a, [this.indent, .1, this.indent2, .1]), a = [], this.addLabelCheckBox(a, _a("editor.clickthroughenabled"), function (q) {
                    return q.isClickThroughEnabled()
                }, function (q, a) {
                    return q.setClickThroughEnabled(a)
                }), this.addLabelCheckBox(a, _a("editor.syncsize"), function (q) {
                    return q.isSyncSize()
                }, function (q, a) {
                    return q.setSyncSize(a)
                }), this.addRow(a, [this.indent, .1, this.indent2, .1])
            }, a
        }(jq),
        Sa = {
            width: 20,
            height: 16,
            comps: [{
                type: "image",
                stretch: "uniform",
                color: {
                    func: function (q, a) {
                        if (q instanceof ht.Block) return null;
                        if (q instanceof ht.Node) {
                            var k = q.s("shape3d");
                            if (ht.Default.isString(k) && hteditor.isJSON(k)) return null
                        }
                        return q instanceof ht.Light ? q.s("light.color") : hteditor.config.color_dark
                    }
                },
                name: {
                    func: function (q, a) {
                        if (q instanceof ht.Block) return "editor.block";
                        if (q instanceof ht.Shape) return q.getThickness() > 0 ? "editor.wall" : "editor.floor";
                        if (q instanceof ht.Node) {
                            var k = q.s("shape3d");
                            if (k) {
                                if (ht.Default.isString(k) && hteditor.isJSON(k)) return k.substr(0, k.length - 4) + "png";
                                if ("sphere" === k) return "editor.sphere";
                                if ("cylinder" === k) return "editor.cylinder"
                            }
                            return "editor.cube"
                        }
                        return q.getIcon()
                    }
                },
                rect: [0, 0, 20, 16]
            }]
        },
        da = hteditor.getString,
        ea = function (q) {
            function a(k) {
                V(this, a);
                var w = Q(this, q.call(this, k, k.dm));
                return w.initMenu(), w
            }
            return E(a, q), a.prototype.handleDelete = function () {
                this.removeSelection()
            }, a.prototype.onDataDoubleClicked = function (q, a) {
                a.altKey ? this.editor.gv.fitData(q, !0, hteditor.config.fitDataPadding) : (this.editor.gv.makeCenter(q, !0), this.editor.scene.flyTo(q, {
                    animation: !0
                }))
            }, a.prototype.getLabel = function (q) {
                var a = q.getDisplayName() || q.getName() || q.getTag();
                if (!a) {
                    var k = q.s("shape3d");
                    if (q instanceof ht.Block) a = da(q.getClassName());
                    else if (q instanceof ht.Light) a = da("ht.Light");
                    else if (q instanceof ht.Shape) a = da("cylinder" === k ? "editor.pipeline" : q.getThickness() > 0 ? "editor.wall" : "editor.floor");
                    else if (q instanceof ht.Node) {
                        var w = "editor." + (k || "cube");
                        if (!(a = da(w)) || w === a)
                            if (hteditor.isJSON(k)) {
                                var I = this.editor.getFileNode(k),
                                    K = I ? I.getName() : k;
                                a = hteditor.fileNameToDisplayName(K)
                            } else a = k
                    }
                    a || (a = da("editor.cube"))
                }
                return a || ""
            }, a.prototype.getIcon = function (q) {
                return Sa
            }, a.prototype._endDrag = function (q, a) {
                var k = this,
                    w = a.type,
                    I = a.parent,
                    K = a.refData;
                if (this.isSelected(this.draggingData)) {
                    var b = this.getTopRowOrderSelection();
                    "down" === w && b.reverse(), b.each(function (q) {
                        k._dropData(q, w, I, K)
                    })
                } else this._dropData(this.draggingData, w, I, K), this.sm().ss(this.draggingData);
                I && this.expand(I)
            }, a.prototype._dropData = function (q, a, k, w) {
                if (q.setParent(k), "down" === a || "up" === a) {
                    var I = k ? k.getChildren() : this.dm().getRoots(),
                        K = I.indexOf(w);
                    "down" === a && K++, I.indexOf(q) < K && K--, this.dm().moveTo(q, K)
                } else k || this.dm().moveToBottom(q)
            }, a.prototype.isEditable = function (q) {
                return !0
            }, a.prototype.rename = function (q, a) {
                q.setDisplayName(a)
            }, a.prototype.initMenu = function () {
                this.menu = new ht.widget.ContextMenu;
                var q = [],
                    a = this.editor;
                this.addSelectDescendantItems(q), this.addBlockItems(q), this.menu.setItems(q), this.menu.addTo(this.getView()), a.menus.push(this.menu)
            }, a.prototype.onPropertyChanged = function (a) {
                "filter" === a.property && (this.visibleMap = null, this.ivm()), q.prototype.onPropertyChanged.call(this, a)
            }, a.prototype.handleDataPropertyChange = function (a) {
                !this._filter || "name" !== a.property && "displayName" !== a.property ? q.prototype.handleDataPropertyChange.call(this, a) : (this.visibleMap = null, this.ivm())
            }, a.prototype.checkVisible = function (q) {
                if (!this._filter) return !0;
                if (R.config.sceneTreeVisibleFunc) return R.config.sceneTreeVisibleFunc(q, this._filter);
                var a = this.getLabel(q),
                    k = (this._filter + "").toLowerCase();
                return null != a && (a = (a + "").toLowerCase(), a.indexOf(k) >= 0)
            }, a.prototype.validateModel = function () {
                var a = this;
                this._filter && !this.visibleMap && (this.visibleMap = {}, this.getDataModel().each(function (q) {
                    if (a.checkVisible(q)) {
                        a.visibleMap[q._id] = !0;
                        for (var k = q.getParent(); k && !a.visibleMap[k._id];) a.visibleMap[k._id] = !0, k = k.getParent()
                    }
                })), q.prototype.validateModel.call(this)
            }, a.prototype.isVisible = function (q, a) {
                if (!a && q._refGraph) return !1;
                var k = this.getVisibleFunc();
                return !(k && !k(q)) && (!this._filter || this.visibleMap && this.visibleMap[q._id])
            }, a
        }(hteditor.DNDTree);
    hteditor.msClass(ea, {
        ms_ac: ["filter"]
    });
    var Ba = function (q) {
            function a(k) {
                V(this, a);
                var w = Q(this, q.call(this));
                w.editor = k, w.tree = new ea(k), w.setCenterView(w.tree), w.controlPane = new ht.widget.FormPane, w.controlPane.addRow([{
                    id: "filter",
                    textField: {}
                }], [.1, ht.Default.isTouchable ? 32 : 18]), w.setBottomView(w.controlPane), w.setBottomHeight(ht.Default.widgetRowHeight + 8);
                var I = !1,
                    K = w.input = w.controlPane.getViewById("filter").getElement();
                return K.addEventListener("compositionstart", function () {
                    return I = !0
                }), K.addEventListener("compositionend", function () {
                    return I = !1
                }), K.addEventListener("keyup", function (q) {
                    I || (ht.Default.isEsc(q) && (K.value = ""), w.tree.setFilter(K.value))
                }), w
            }
            return E(a, q), a.prototype.makeVisible = function (q) {
                var a = this;
                q instanceof ht.Data && (q = [q]), q.forEach(function (q) {
                    return a.tree.makeVisible(q)
                })
            }, a
        }(ht.widget.BorderPane),
        Aa = hteditor.getString,
        ha = hteditor.getter,
        Ja = hteditor.setter,
        ta = hteditor.createCodeEditor,
        Oa = ht.Default.clone,
        $a = ht.Default.isNumber,
        ca = ht.Default.isArray,
        Da = function (q) {
            function a(k) {
                V(this, a);
                var w = Q(this, q.call(this));
                w.editor = k, w._updateHandles = [], (w.batchInfoDialog = new ia(k)).handleComplete = function (q) {
                    var a = w.sm.ld();
                    a && a.a("batchInfo", q)
                };
                var I = w._tablePane = new ht.widget.TablePane;
                return w.initTable(), w.addRow([Aa("editor.batchbrightness"), w.addChickBox(function () {
                    return !w.editor.dm.a("sceneBatchBrightnessDisabled")
                }, function (q) {
                    return w.editor.dm.a("sceneBatchBrightnessDisabled", !q)
                }), Aa("editor.batchblend"), w.addChickBox(function () {
                    return !w.editor.dm.a("sceneBatchBlendDisabled")
                }, function (q) {
                    return w.editor.dm.a("sceneBatchBlendDisabled", !q)
                }), Aa("editor.batchinstanced"), w.addChickBox(function () {
                    return !w.editor.dm.a("sceneBatchInstancedDisabled")
                }, function (q) {
                    return w.editor.dm.a("sceneBatchInstancedDisabled", !q)
                })], [80, .1, 80, .1, 80, .1]), w.addRow([I], [.1], .1), w.addRow([{
                    button: {
                        icon: "editor.add",
                        toolTip: Aa("editor.add"),
                        onClicked: function () {
                            w.addData()
                        }
                    }
                }, {
                    button: {
                        icon: "editor.delete",
                        toolTip: Aa("editor.delete"),
                        onClicked: function () {
                            if (w.sm.size()) {
                                w.editor.beginTransaction();
                                var q = [],
                                    a = w.editor.scene.getBatchInfoMap();
                                w.sm.each(function (k) {
                                    var I = k.a("name");
                                    q.push(I);
                                    var K = a[I];
                                    delete a[I], w.addBatchInfoHistory("remove", void 0, {
                                        name: I,
                                        info: K
                                    })
                                }), w._tablePane.getTableView().removeSelection(), w.editor.scene.dm().each(function (a) {
                                    var k = a.s("batch");
                                    k && q.indexOf(k) > -1 && a.s("batch", void 0)
                                }), q.forEach(function (q) {
                                    return w.editor.scene.invalidateBatch(q)
                                }), w.editor.endTransaction()
                            }
                        }
                    }
                }], [20, 20]), w
            }
            return E(a, q), a.prototype.isNameExist = function (q, a) {
                for (var k = !1, w = this.dm.getDatas(), I = 0, K = w.size(); I < K; I++) {
                    var b = w.get(I);
                    if (b !== a && (k = b.a("name") === q)) break
                }
                return k
            }, a.prototype.onDataPropertyChanged = function (q) {
                if (!this._parsing) {
                    var a = q.property,
                        k = q.data,
                        w = q.newValue,
                        I = q.oldValue,
                        K = this.editor.scene;
                    if ("a:name" === a) {
                        if (this.isNameExist(w, k)) return void k.a("name", I);
                        this.editor.beginTransaction();
                        var b = "add",
                            s = void 0;
                        if (I) {
                            b = "rename";
                            var u = K.getBatchInfoMap();
                            s = u[I], delete u[I]
                        }
                        var C = k.a("batchInfo");
                        K.setBatchInfo(w, C), this.addBatchInfoHistory(b, {
                            name: w,
                            info: k.a("batchInfo")
                        }, {
                            name: I,
                            info: s
                        }), this.editor.scene.dm().each(function (q) {
                            q.s("batch") && q.s("batch") === I && q.s("batch", w)
                        }), this.editor.endTransaction(), this.editor.scene.invalidateBatch(I), this.editor.scene.invalidateBatch(w)
                    } else if ("a:batchInfo" === a) {
                        var L = k.a("name");
                        if (!L) return;
                        var j = K.getBatchInfo(L);
                        K.setBatchInfo(L, w), this.addBatchInfoHistory("updateInfo", {
                            name: L,
                            info: k.a("batchInfo")
                        }, {
                            name: L,
                            info: j
                        }), this.editor.scene.invalidateBatch(L)
                    }
                }
            }, a.prototype.addBatchInfoHistory = function (q, a, k) {
                var w = this,
                    I = this.editor.scene;
                I.dm().addHistory({
                    kind: q,
                    property: "batchInfo",
                    undo: function () {
                        "add" === q ? delete I.getBatchInfoMap()[a.name] : "remove" === q ? I.setBatchInfo(k.name, k.info) : "rename" === q ? (delete I.getBatchInfoMap()[a.name], k.name && I.setBatchInfo(k.name, k.info)) : "updateInfo" === q && I.setBatchInfo(k.name, k.info), w.clearBatchInfo(), w.parseBatchInfoMap(I.getBatchInfoMap()), a && a.name && I.invalidateBatch(a.name), k && k.name && I.invalidateBatch(k.name)
                    },
                    redo: function () {
                        "add" === q ? I.setBatchInfo(a.name, a.info) : "remove" === q ? delete I.getBatchInfoMap()[k.name] : "rename" === q ? (k.name && delete I.getBatchInfoMap()[k.name], I.setBatchInfo(a.name, a.info)) : "updateInfo" === q && I.setBatchInfo(a.name, a.info), w.clearBatchInfo(), w.parseBatchInfoMap(I.getBatchInfoMap()), a && a.name && I.invalidateBatch(a.name), k && k.name && I.invalidateBatch(k.name)
                    }
                })
            }, a.prototype.onDataModelChanged = function (q) {
                this._parsing
            }, a.prototype.initTable = function () {
                var q = this,
                    a = this._tablePane;
                return a.getTableView().setEditable(!0), a.addColumns([this.$73$("name", Aa("editor.name"), 80), this.$73$("batchInfo", Aa("editor.batchInfo"), 200), {
                    name: "operation",
                    tag: "operation",
                    width: 80,
                    displayName: Aa("editor.operation"),
                    align: "center",
                    editable: !1,
                    drawCell: function (a, k, w, I, K, b, s, u) {
                        var C = k.__btn__;
                        return C || (C = k.__btn__ = new ht.widget.Button, C.setLabel(Aa("editor.selectall")), C.onClicked = function () {
                            var a = k.a("name"),
                                w = q.editor.dm.toDatas(function (q) {
                                    return q.s("batch") === a
                                });
                            0 !== w.length && (q.editor.sm.ss(w), q.editor.treePane.makeVisible(w), q.editor.rightBottomTabView.getTabModel().sm().ss(q.editor.treePaneTab))
                        }), C.setWidth(s), C.setHeight(u), C.getView()
                    }
                }]), a.getColumnModel().getDataByTag("batchInfo").isCellEditable = function (a, k, w, I) {
                    return q.batchInfoDialog.parseBatchInfo(a.a("batchInfo")), q.batchInfoDialog.show(), !1
                }, this.dm.md(this.onDataPropertyChanged, this), this.dm.mm(this.onDataModelChanged, this), a
            }, a.prototype.$73$ = function (q, a, k) {
                return {
                    name: q,
                    tag: q,
                    width: k,
                    displayName: a,
                    align: "center",
                    editable: !0,
                    accessType: "attr"
                }
            }, a.prototype.toBatchInfoMap = function () {
                var q = {};
                return this.dm.each(function (a) {
                    var k = a.a("name");
                    k && (q[k] = a.a("batchInfo"))
                }), q
            }, a.prototype.addData = function (q) {
                var a = arguments.length > 1 && void 0 !== arguments[1] ? arguments[1] : null,
                    k = new ht.Data;
                k.a({
                    name: q,
                    batchInfo: a
                }), this.dm.add(k)
            }, a.prototype.parseBatchInfoMap = function (q) {
                this._parsing = !0;
                for (var a in q) this.addData(a, q[a]);
                this._parsing = !1
            }, a.prototype.clearBatchInfo = function () {
                this.dm.clear(), this.updateProperties()
            }, a.prototype.addChickBox = function (q, a) {
                var k = this,
                    w = new ht.widget.CheckBox;
                return w.getValue = q, w.setValue(q()), w.onValueChanged = function (q, w) {
                    k._updatting || a(w)
                }, this._updateHandles.push(function () {
                    w.setValue(q())
                }), w
            }, a.prototype.updateProperties = function () {
                this._updatting = !0, this._updateHandles.forEach(function (q) {
                    return q()
                }), this._updatting = !1
            }, a.prototype.updateBatchInfoDialog = function () {
                this.batchInfoDialog.isShowing() && this.batchInfoDialog._commonView.filterPropertiesLater()
            }, v(a, [{
                key: "dm",
                get: function () {
                    return this._tablePane.getDataModel()
                }
            }, {
                key: "sm",
                get: function () {
                    return this.dm.sm()
                }
            }, {
                key: "keys",
                get: function () {
                    var q = [void 0];
                    return this.dm.each(function (a) {
                        return q.push(a.a("name"))
                    }), q
                }
            }]), a
        }(ht.widget.FormPane),
        Ya = {
            alphaTest: ht.Style.alphaTest,
            brightness: !1,
            blend: !1,
            color: void 0,
            image: void 0,
            material: void 0,
            light: !0,
            lightMask: ht.Style["light.mask"],
            reverseCull: !1,
            reverseFlip: !1,
            reverseColor: ht.Color.reverse,
            transparentMask: ht.Style["transparent.mask"],
            uvOffset: [0, 0],
            uvScale: [1, 1],
            discardSelectable: !0,
            opacity: 1,
            transparent: !1,
            autoSort: !1,
            envmap: ht.Style.envmap,
            envmapProbe: ht.Style["envmap.probe"],
            roughness: ht.Style.roughness,
            shadowCast: ht.Style["shadow.cast"],
            shadowReceive: ht.Style["shadow.receive"],
            reflectable: ht.Style["3d.reflectable"],
            bloom: ht.Style.bloom,
            clipboxMask: ht.Style["3d.clipbox.mask"],
            headlightAmbientIntensity: ht.Style["headlight.ambientIntensity"] || ht.Default.graph3dViewHeadlightAmbientIntensity,
            effectFlowMask: ht.Style["effect.flow.mask"],
            highlight: !0
        },
        ia = function (q) {
            function a(k) {
                V(this, a);
                var w = Q(this, q.call(this));
                w.editor = k;
                var I = w._tabView = w.initTabView(),
                    K = w.initButtons();
                return w.setConfig({
                    title: '<span style="margin-left: 20px">' + Aa("editor.batchInfo") + "</span>",
                    closable: !0,
                    draggable: !0,
                    width: 448,
                    height: 485,
                    resizeMode: "wh",
                    maximizable: !0,
                    content: I,
                    buttons: K,
                    buttonsAlign: "right",
                    titleHeight: 38,
                    titleBackground: ht.Default.dialogHeaderBackground,
                    titleColor: "#303033",
                    titleIconGap: 16,
                    borderWidth: 0,
                    contentPadding: 6
                }), w.setModal(!1), w
            }
            return E(a, q), a.prototype.initTabView = function () {
                var q = this,
                    a = new ht.widget.TabView,
                    k = this._commonView = this.initCommonView(),
                    w = this._commonTab = a.add(Aa("editor.common"), k, !0),
                    I = this._advanceView = this.initAdvanceView(),
                    K = this._advanceTab = a.add(Aa("editor.advance"), I),
                    b = a.getTabModel().sm();
                return b.ms(function () {
                    var a = b.ld();
                    a === w ? q.parseBatchInfo(JSON.parse(I.getValue())) : a === K && I.setValue(JSON.stringify(k.data.getAttrObject(), function (q, a) {
                        return void 0 === a ? null : a
                    }, 4))
                }), a
            }, a.prototype.initCommonView = function () {
                var q = new Kq(this.editor, "batchInfo");
                q.setLabelVPadding(0), q.getValue = function (q) {
                    return q(this.data)
                }, q.setValue = function (q, a, k) {
                    q(this.data, k ? k() : a)
                };
                var a = q.data = new ht.Data;
                a.onPropertyChanged = function (a) {
                    return q.updateProperties()
                };
                var k = q.indent,
                    w = q.getHGap(),
                    I = function () {
                        return !q.getValue(function (q) {
                            return q.a("material")
                        })
                    },
                    K = this.editor,
                    b = [],
                    C = function (q, a) {
                        if (!a.view.draggingData) return !1;
                        var k = a.view.draggingData.fileType;
                        return "symbol" === k || "material" === k || s(a.view.draggingData.url)
                    };
                q.addLabelURL(b, Aa("editor.batchview.imageormaterial"), function (q) {
                    return q.a("image") || q.a("material")
                }, function (a, k) {
                    k ? "material" === K.getFileNode(k).fileType ? (a.a("image", void 0), a.a("material", k)) : (a.a("image", k), a.a("material", void 0)) : (a.a("image", void 0), a.a("material", void 0)), q.filterProperties()
                }, C), q.addImage(b, function (q) {
                    return q.a("image") || q.a("material")
                }, void 0, function (q) {
                    return u(q) ? q.substr(0, q.length - 5) + ".png" : q
                }), q.addLabelCheckBox(b, Aa("editor.instanced"), ha("a", "instanced"), Ja("a", "instanced")), q.addRow(b, [k, .1, 20, k, 20]), b = [];
                var L = function (q) {
                    var a = this;
                    L.superClass.constructor.call(a, q);
                    var k = a._listView = new ht.widget.ListView;
                    k.getView().style.background = "white", k.setCheckMode(!0), k.sm().ms(function (w) {
                        var I = k.dm().getDataByTag(4294967295);
                        if ("append" === w.kind)
                            if (w.datas.get(0) === I) k.sm().selectAll();
                            else {
                                var K = k.sm().getSelection().length;
                                K === k.dm().getDatas().length - 1 && k.sm().selectAll()
                            }
                        else if ("remove" === w.kind)
                            if (w.datas.get(0) === I) k.sm().clearSelection();
                            else {
                                var b = k.sm().getSelection().toList(function (q) {
                                    return q !== I
                                });
                                k.sm().setSelection(b)
                            } q.setValue(a.getValue())
                    })
                };
                ht.Default.def(L, ht.widget.BaseDropDownTemplate, {
                    getView: function () {
                        return this._listView.getView()
                    },
                    onOpened: function (q) {
                        q = j.getValue();
                        var a = this._listView,
                            k = new ht.Data;
                        k.setName(Aa("editor.all")), k.setTag(4294967295), a.dm().add(k);
                        var w = new ht.Data;
                        w.setName(Aa("editor.default")), w.setTag(0), a.dm().add(w);
                        var I = this._master.getMaskMap();
                        for (var K in I) {
                            K = parseInt(K);
                            var b = new ht.Data;
                            b.setName(I[K]), b.setTag(K), a.dm().add(b)
                        }
                        if (q)
                            if (4294967295 === q) a.sm().selectAll();
                            else {
                                var s = new ht.List;
                                a.dm().each(function (a) {
                                    q & 1 << a.getTag() && s.add(a)
                                }), a.sm().setSelection(s)
                            }
                    },
                    onClosed: function () {},
                    getValue: function () {
                        var q = this._listView.dm(),
                            a = q.sm();
                        if (0 === a.size()) return 0;
                        var k = q.getDataByTag(4294967295);
                        if (a.contains(k)) return 4294967295;
                        var w = 0;
                        return a.each(function (q) {
                            var a = q.getTag();
                            4294967295 !== a && q.getName() && (w += 1 << a)
                        }), w
                    },
                    getHeight: function () {
                        return 100
                    }
                });
                var j = new ht.widget.MultiComboBox;
                j.setEditable(!1), j.setDropDownComponent(L), j.drawValue = function (q, a, k, w, I) {
                    var K = j.getValue(),
                        b = j.getMaskMap(),
                        s = [];
                    if (4294967295 === K) s = Aa("editor.all");
                    else if (0 === K) s = "";
                    else {
                        1 & K && s.push(Aa("editor.default"));
                        for (var u in b) u = parseInt(u), K & 1 << u && s.push(b[u]);
                        s.join(", ")
                    }
                    ht.Default.drawText(q, s, j.getLabelFont(), j.getLabelColor(), a + 1, k, 0, I)
                }, j.getMaskMap = function () {
                    return K.dm.a("sceneLightGruopAlias")
                }, j.getValue = function () {
                    return void 0 === a.a("lightMask") || null === a.a("lightMask") ? Ya.lightMask : a.a("lightMask")
                }, j.onValueChanged = function (q, k) {
                    a.a("lightMask", k)
                }, b.push(Aa("editor.lightmask")), b.push(j), q.addLabelCheckBox(b, Aa("editor.light"), ha("a", "light"), Ja("a", "light")), q.addRow(b, [k, .1, k, 20]), b = [], q.addLabelRange(b, Aa("editor.headlight.ambientintensity"), ha("a", "headlightAmbientIntensity"), Ja("a", "headlightAmbientIntensity"), void 0, void 0, .01, "number"), q.addLabelCheckBox(b, Aa("editor.autoSort"), ha("a", "autoSort"), Ja("a", "autoSort")), q.addLabelCheckBox(b, Aa("editor.reflectable"), ha("a", "reflectable"), Ja("a", "reflectable")), q.addRow(b, [k, .1, k, 20, k, 20]), b = [], q.addLabelRange(b, Aa("editor.alphatest"), ha("a", "alphaTest"), Ja("a", "alphaTest"), 0, 1, .01, "number"), q.addLabelCheckBox(b, Aa("editor.shadow.cast"), ha("a", "shadowCast"), Ja("a", "shadowCast")), q.addLabelCheckBox(b, Aa("editor.shadow.receive"), ha("a", "shadowReceive"), Ja("a", "shadowReceive")), q.addRow(b, [k, .1, k, 20, k, 20]), b = [];
                var G = function (q) {
                    var a = this;
                    G.superClass.constructor.call(a, q);
                    var k = a._listView = new ht.widget.ListView;
                    k.getView().style.background = "white", k.setCheckMode(!0), k.sm().ms(function (w) {
                        var I = k.dm().getDataByTag(4294967295);
                        if ("append" === w.kind)
                            if (w.datas.get(0) === I) k.sm().selectAll();
                            else {
                                var K = k.sm().getSelection().length;
                                K === k.dm().getDatas().length - 1 && k.sm().selectAll()
                            }
                        else if ("remove" === w.kind)
                            if (w.datas.get(0) === I) k.sm().clearSelection();
                            else {
                                var b = k.sm().getSelection().toList(function (q) {
                                    return q !== I
                                });
                                k.sm().setSelection(b)
                            } q.setValue(a.getValue())
                    })
                };
                ht.Default.def(G, ht.widget.BaseDropDownTemplate, {
                    getView: function () {
                        return this._listView.getView()
                    },
                    onOpened: function (q) {
                        q = X.getValue();
                        var a = this._listView,
                            k = new ht.Data;
                        k.setName(Aa("editor.all")), k.setTag(4294967295), a.dm().add(k);
                        var w = new ht.Data;
                        w.setName(Aa("editor.default")), w.setTag(0), a.dm().add(w);
                        var I = this._master.getMaskMap();
                        for (var K in I) {
                            K = parseInt(K);
                            var b = new ht.Data;
                            b.setName(I[K]), b.setTag(K), a.dm().add(b)
                        }
                        if (q)
                            if (4294967295 === q) a.sm().selectAll();
                            else {
                                var s = new ht.List;
                                a.dm().each(function (a) {
                                    q & 1 << a.getTag() && s.add(a)
                                }), a.sm().setSelection(s)
                            }
                    },
                    onClosed: function () {},
                    getValue: function () {
                        var q = this._listView.dm(),
                            a = q.sm();
                        if (0 === a.size()) return 0;
                        var k = q.getDataByTag(4294967295);
                        if (a.contains(k)) return 4294967295;
                        var w = 0;
                        return a.each(function (q) {
                            var a = q.getTag();
                            4294967295 !== a && q.getName() && (w += 1 << a)
                        }), w
                    },
                    getHeight: function () {
                        return 100
                    }
                });
                var X = new ht.widget.MultiComboBox;
                X.setEditable(!1), X.setDropDownComponent(G), X.drawValue = function (q, a, k, w, I) {
                    var K = X.getValue(),
                        b = X.getMaskMap(),
                        s = [];
                    if (4294967295 === K) s = Aa("editor.all");
                    else if (0 === K) s = "";
                    else {
                        1 & K && s.push(Aa("editor.default"));
                        for (var u in b) u = parseInt(u), K & 1 << u && s.push(b[u]);
                        s.join(", ")
                    }
                    ht.Default.drawText(q, s, X.getLabelFont(), X.getLabelColor(), a + 1, k, 0, I)
                }, X.getMaskMap = function () {
                    return K.dm.a("sceneClipboxGroupAlias")
                }, X.getValue = function () {
                    return void 0 === a.a("clipboxMask") || null === a.a("clipboxMask") ? Ya.clipboxMask : a.a("clipboxMask")
                }, X.onValueChanged = function (q, k) {
                    a.a("clipboxMask", k)
                }, b.push(Aa("editor.batchview.clipboxmask")), b.push(X);
                var Z = function (q) {
                    var a = this;
                    Z.superClass.constructor.call(a, q);
                    var k = a._listView = new ht.widget.ListView;
                    k.getView().style.background = "white", k.setCheckMode(!0), k.sm().ms(function (w) {
                        var I = k.dm().getDataByTag(4294967295);
                        if ("append" === w.kind)
                            if (w.datas.get(0) === I) k.sm().selectAll();
                            else {
                                var K = k.sm().getSelection().length;
                                K === k.dm().getDatas().length - 1 && k.sm().selectAll()
                            }
                        else if ("remove" === w.kind)
                            if (w.datas.get(0) === I) k.sm().clearSelection();
                            else {
                                var b = k.sm().getSelection().toList(function (q) {
                                    return q !== I
                                });
                                k.sm().setSelection(b)
                            } q.setValue(a.getValue())
                    })
                };
                ht.Default.def(Z, ht.widget.BaseDropDownTemplate, {
                    getView: function () {
                        return this._listView.getView()
                    },
                    onOpened: function (q) {
                        q = this._master.getValue();
                        var a = this._listView,
                            k = new ht.Data;
                        k.setName(Aa("editor.all")), k.setTag(4294967295), a.dm().add(k);
                        var w = new ht.Data;
                        w.setName(Aa("editor.default")), w.setTag(0), a.dm().add(w);
                        var I = this._master.getMaskMap();
                        for (var K in I) {
                            K = parseInt(K);
                            var b = new ht.Data;
                            b.setName(I[K]), b.setTag(K), a.dm().add(b)
                        }
                        if (q)
                            if (4294967295 === q) a.sm().selectAll();
                            else {
                                var s = new ht.List;
                                a.dm().each(function (a) {
                                    q & 1 << a.getTag() && s.add(a)
                                }), a.sm().setSelection(s)
                            }
                    },
                    onClosed: function () {},
                    getValue: function () {
                        var q = this._listView.dm(),
                            a = q.sm();
                        if (0 === a.size()) return 0;
                        var k = q.getDataByTag(4294967295);
                        if (a.contains(k)) return 4294967295;
                        var w = 0;
                        return a.each(function (q) {
                            var a = q.getTag();
                            4294967295 !== a && q.getName() && (w += 1 << a)
                        }), w
                    },
                    getHeight: function () {
                        return 100
                    }
                });
                var r = new ht.widget.MultiComboBox;
                r.setEditable(!1), r.setDropDownComponent(Z), r.drawValue = function (q, a, k, w, I) {
                    var K = r.getValue(),
                        b = r.getMaskMap(),
                        s = [];
                    if (4294967295 === K) s = Aa("editor.all");
                    else if (0 === K) s = "";
                    else {
                        1 & K && s.push(Aa("editor.default"));
                        for (var u in b) u = parseInt(u), K & 1 << u && s.push(b[u]);
                        s.join(", ")
                    }
                    ht.Default.drawText(q, s, r.getLabelFont(), r.getLabelColor(), a + 1, k, 0, I)
                }, r.getMaskMap = function () {
                    return K.dm.a("sceneFlowEffectGroupAlias")
                }, r.getValue = function () {
                    return void 0 === a.a("effectFlowMask") || null === a.a("effectFlowMask") ? Ya.effectFlowMask : a.a("effectFlowMask")
                }, r.onValueChanged = function (q, k) {
                    a.a("effectFlowMask", k)
                }, b.push(Aa("editor.effect.flowmask")), b.push(r), q.addRow(b, [k, .1, k, .1]), b = [];
                var W = q.addLabelComboBox(b, Aa("editor.envmap.probe"), ha("a", "envmapProbe"), Ja("a", "envmapProbe"), [], []);
                return q.updateHandlers.push(function () {
                    if (q.data) {
                        var a = q.dataModel.getEnvmap();
                        if (ca(a)) {
                            var k = a.map(function (q, a) {
                                return a
                            });
                            W.setValues(k);
                            var w = a.map(function (q, a) {
                                return a + " - " + (q.name || q.type)
                            });
                            W.setLabels(w);
                            var I = q.getValue(ha("a", "envmapProbe"));
                            W.getValue() !== I && W.setValue(I)
                        }
                    }
                }), q.addLabelRange(b, Aa("editor.envmap"), ha("a", "envmap"), Ja("a", "envmap"), 0, 1, .01, "number"), q.addRow(b, [k, .1, k, .1]).visible = function () {
                    return !!q.dataModel && "legacy" !== (q.dataModel.getEnvmapType() || "legacy")
                }, b = [], q.addLabelRange(b, Aa("editor.envmap.roughness"), ha("a", "roughness"), Ja("a", "roughness"), 0, 1, .01, "number"), q.addLabelCheckBox(b, Aa("editor.bloom"), ha("a", "bloom"), Ja("a", "bloom")), q.addLabelCheckBox(b, Aa("editor.reverseflip"), ha("a", "reverseFlip"), Ja("a", "reverseFlip")), q.addRow(b, [k, .1, k, 20, k, 20]).visible = function () {
                    return !(!q.dataModel || !I()) && "legacy" !== (q.dataModel.getEnvmapType() || "legacy")
                }, b = [], q.addLabelRange(b, Aa("editor.envmap"), ha("a", "envmap"), Ja("a", "envmap"), 0, 1, .01, "number"), q.addLabelCheckBox(b, Aa("editor.bloom"), ha("a", "bloom"), Ja("a", "bloom")), q.addLabelCheckBox(b, Aa("editor.reverseflip"), ha("a", "reverseFlip"), Ja("a", "reverseFlip")), q.addRow(b, [k, .1, k, 20, k, 20]).visible = function () {
                    return !(!q.dataModel || !I()) && "legacy" === (q.dataModel.getEnvmapType() || "legacy")
                }, b = [], q.addLabelRange(b, Aa("editor.envmap"), ha("a", "envmap"), Ja("a", "envmap"), 0, 1, .01, "number"), b.push(null, null, null, null), q.addRow(b, [k, .1, k, 20, k, 20]).visible = function () {
                    return !(!q.dataModel || I()) && "legacy" === (q.dataModel.getEnvmapType() || "legacy")
                }, b = [], q.addLabelRange(b, Aa("editor.opacity"), ha("a", "opacity"), Ja("a", "opacity"), 0, 1, .01, "number"), q.addLabelCheckBox(b, Aa("editor.transparent"), ha("a", "transparent"), Ja("a", "transparent")), q.addLabelCheckBox(b, Aa("editor.discardselectable"), ha("a", "discardSelectable"), Ja("a", "discardSelectable")), q.addRow(b, [k, .1, k, 20, k, 20]).visible = I, b = [], q.addLabelColor(b, Aa("editor.color"), ha("a", "color"), Ja("a", "color")), q.addLabelCheckBox(b, Aa("editor.blend"), ha("a", "blend"), Ja("a", "blend")), q.addRow(b, [k, .1, k, 20]).visible = I, b = [], q.addLabelColor(b, Aa("editor.reversecolor"), ha("a", "reverseColor"), Ja("a", "reverseColor")), q.addLabelCheckBox(b, Aa("editor.reversecull"), ha("a", "reverseCull"), Ja("a", "reverseCull")), q.addRow(b, [k, .1, k, 20]).visible = I, b = [Aa("editor.uvscale")], q.addLabelInput(b, "U", function (q) {
                    var a = q.a("uvScale") || Ya.uvScale;
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = Oa(q.a("uvScale") || Ya.uvScale);
                    k[0] = Number(a), q.a("uvScale", k)
                }, "number", .01), q.addLabelInput(b, "V", function (q) {
                    var a = q.a("uvScale") || Ya.uvScale;
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = Oa(q.a("uvScale") || Ya.uvScale);
                    k[1] = Number(a), q.a("uvScale", k)
                }, "number", .01), q.addLabelCheckBox(b, Aa("editor.transparentmask"), ha("a", "transparentMask"), Ja("a", "transparentMask")), q.addRow(b, [k - 20 - w, 20, .1, 20, .1, k, 20]).visible = I, b = [Aa("editor.uvoffset")], q.addLabelInput(b, "U", function (q) {
                    var a = q.a("uvOffset") || Ya.uvOffset;
                    return Number(a[0].toFixed(5))
                }, function (q, a) {
                    var k = Oa(q.a("uvOffset") || Ya.uvOffset);
                    k[0] = Number(a), q.a("uvOffset", k)
                }, "number", .01), q.addLabelInput(b, "V", function (q) {
                    var a = q.a("uvOffset") || Ya.uvOffset;
                    return Number(a[1].toFixed(5))
                }, function (q, a) {
                    var k = Oa(q.a("uvOffset") || Ya.uvOffset);
                    k[1] = Number(a), q.a("uvOffset", k)
                }, "number", .01), q.addLabelCheckBox(b, Aa("editor.brightness"), ha("a", "brightness"), Ja("a", "brightness")), q.addRow(b, [k - 20 - w, 20, .1, 20, .1, k, 20]).visible = I, b = [], q.addLabelCheckBox(b, Aa("editor.highlight"), ha("a", "highlight"), Ja("a", "highlight")), q.addRow(b, [k, .1]), q
            }, a.prototype.initAdvanceView = function () {
                return ta({
                    value: "",
                    language: "json",
                    theme: R.config.codeEditorTheme,
                    readOnly: !1,
                    minimap: {
                        enabled: !1
                    }
                })
            }, a.prototype.initButtons = function () {
                var q = this,
                    a = [];
                return a.push({
                    label: Aa("editor.ok"),
                    action: function () {
                        q._tabView.getTabModel().sm().ld() === q._advanceTab && q.parseBatchInfo(JSON.parse(q._advanceView.getValue())), q.handleComplete(q.toBatchInfo()), q.hide()
                    }
                }), a.push({
                    label: Aa("editor.cancel"),
                    action: function () {
                        q.hide()
                    }
                }), a
            }, a.prototype.toBatchInfo = function () {
                var q = Oa(this._commonView.data.getAttrObject());
                for (var a in Ya) {
                    var k = Ya[a];
                    (ca(k) && ca(q[a]) && q[a].join(",") === k.join(",") || q[a] === k || void 0 === k && null === q[a]) && delete q[a]
                }
                return q
            }, a.prototype.parseBatchInfo = function (q) {
                q = Oa(q) || {};
                for (var a in Ya)
                    if (q.hasOwnProperty(a)) {
                        if ("uvOffset" === a || "uvScale" === a) {
                            var k = q[a];
                            $a(k) && (q[a] = [k, k])
                        }
                    } else q[a] = Ya[a];
                this._commonView.data.setAttrObject(q)
            }, a.prototype.handleComplete = function (q) {
                console.info(q)
            }, a.prototype.show = function () {
                for (var a, k = arguments.length, w = Array(k), I = 0; I < k; I++) w[I] = arguments[I];
                (a = q.prototype.show).call.apply(a, [this].concat(w)), this._commonView.filterProperties()
            }, a.prototype.hide = function () {
                this._commonView.data.setAttrObject({}), this._tabView.getTabModel().sm().ss(this._commonTab), q.prototype.hide.call(this)
            }, a
        }(ht.widget.Dialog),
        Pa = hteditor.getString,
        Na = hteditor.createButton,
        Ua = function (q) {
            function a(k) {
                V(this, a);
                var w = Q(this, q.call(this));
                w.editor = k;
                var I = w._tablePane = new ht.widget.TablePane;
                w.initTable(), w.addRow([I], [.1], .1);
                var K = [],
                    b = Na(null, Pa("editor.add"), "editor.add", function () {
                        w.editor.editable && (w._editing = !0, w.addData(w.createRenderLayerName()), w.updateRenderLayers())
                    });
                K.push(b);
                var s = Na(null, Pa("editor.delete"), "editor.delete", function () {
                    w.editor.editable && w.sm.size() && (w._editing = !0, w.removeRenderLayer(), w.updateRenderLayers())
                });
                K.push(s), K.push(null);
                var u = Na(null, Pa("editor.bringtofront"), "editor.top", function () {
                    w.editor.editable && w.sm.size() && (w._editing = !0, w.moveRenderLayers("top"), w.updateRenderLayers())
                });
                K.push(u);
                var C = Na(null, Pa("editor.bringforward"), "editor.up", function () {
                    w.editor.editable && w.sm.size() && (w._editing = !0, w.moveRenderLayers("up"), w.updateRenderLayers())
                });
                K.push(C);
                var L = Na(null, Pa("editor.sendbackward"), "editor.down", function () {
                    w.editor.editable && w.sm.size() && (w._editing = !0, w.moveRenderLayers("down"), w.updateRenderLayers())
                });
                K.push(L);
                var j = Na(null, Pa("editor.sendtoback"), "editor.bottom", function () {
                    w.editor.editable && w.sm.size() && (w._editing = !0, w.moveRenderLayers("bottom"), w.updateRenderLayers())
                });
                return K.push(j), K.forEach(function (q) {
                    q instanceof ht.widget.Button && (q.getIconColor = function () {
                        return this.isPressed() ? R.config.color_select : R.config.color_dark
                    })
                }), w.addRow(K, [20, 20, .1, 20, 20, 20, 20]), w
            }
            return E(a, q), a.prototype.isNameExist = function (q, a) {
                for (var k = !1, w = this.dm.getDatas(), I = 0, K = w.size(); I < K; I++) {
                    var b = w.get(I);
                    if (b !== a && (k = b.a("name") === q)) break
                }
                return k
            }, a.prototype.onDataPropertyChanged = function (q) {
                if (!this._parsing) {
                    var a = q.property,
                        k = q.data,
                        w = q.newValue,
                        I = q.oldValue;
                    return "a:name" === a && this.isNameExist(w, k) ? void k.a("name", I) : void 0
                }
            }, a.prototype.onDataModelChanged = function (q) {}, a.prototype.initTable = function () {
                var q = this,
                    a = this._tablePane;
                return a.getTableView().setEditable(!0), a.addColumns([this.$73$("name", Pa("editor.name"), 220), this.$73$("visible", Pa("editor.visible"), 50), this.$73$("clearDepth", Pa("editor.override"), 50)]), a.getTableView().isCellEditable = function (a, k, w, I) {
                    return !!q.editor.editable && ("name" !== k.getTag() || "main" !== a.a("name") && "top" !== a.a("name"))
                }, this.dm.md(this.onDataPropertyChanged, this), a
            }, a.prototype.$73$ = function (q, a, k) {
                var w = this;
                return {
                    name: q,
                    tag: q,
                    width: k,
                    displayName: a,
                    align: "center",
                    editable: !0,
                    accessType: "attr",
                    sortable: !1,
                    setValue: function (a, k, I) {
                        I !== a.a(q) && (a.a(q, I), w.updateRenderLayers())
                    }
                }
            }, a.prototype.updateRenderLayers = function () {
                this._parsing || (this.updatePriority(), this.editor.dm.a("sceneRenderLayerInfoMap", this.toRenderLayerInfoMap()))
            }, a.prototype.toRenderLayerInfoMap = function () {
                var q = {};
                return this.dm.eachByHierarchical(function (a) {
                    var k = a.a("name");
                    k && (q[k] = {
                        ignore: !a.a("visible"),
                        clearDepth: a.a("clearDepth"),
                        priority: a.a("priority")
                    })
                }), q
            }, a.prototype.parseRenderLayerInfoMap = function (q) {
                var a = this;
                if (this._editing) return void(this._editing = !1);
                this._parsing = !0, this.dm.clear();
                var k = Object.values(q),
                    w = Object.keys(q);
                k.forEach(function (q, a) {
                    q.name = w[a]
                }), k.sort(function (q, a) {
                    return q.priority - a.priority
                }), k.forEach(function (q) {
                    a.addData(q.name, !q.ignore, q.clearDepth, q.priority)
                }), q || (this.addData("main", !0, !1, 1e4), this.addData("top", !0, !0, 1e7)), this._parsing = !1
            }, a.prototype.clearRenderLayerInfo = function () {
                var q = ht.Default.getRenderLayerInfo("main"),
                    a = ht.Default.getRenderLayerInfo("top");
                this.editor.scene.setRenderLayerInfoMap({
                    main: q,
                    top: a
                }), this.dm.clear(), this.addData("main", !q.ignore, q.clearDepth, q.priority), this.addData("top", !a.ignore, a.clearDepth, a.priority)
            }, a.prototype.addData = function (q) {
                var a = !(arguments.length > 1 && void 0 !== arguments[1]) || arguments[1],
                    k = !(arguments.length > 2 && void 0 !== arguments[2]) || arguments[2],
                    w = arguments.length > 3 && void 0 !== arguments[3] ? arguments[3] : 10000001,
                    I = (arguments.length > 4 && void 0 !== arguments[4] && arguments[4], new ht.Data);
                I.a({
                    name: q,
                    visible: a,
                    clearDepth: k,
                    priority: w
                }), this.dm.add(I)
            }, a.prototype.removeRenderLayer = function () {
                var q = this;
                this.sm.getSelection().toList().each(function (a) {
                    "main" !== a.a("name") && "top" !== a.a("name") && q.dm.remove(a)
                })
            }, a.prototype.moveRenderLayers = function (q) {
                var a = this;
                this.sm.getSelection().toList().each(function (q) {
                    "main" !== q.a("name") && "top" !== q.a("name") || a.sm.rs(q)
                }), "top" === q ? this.dm.moveSelectionToTop() : "up" === q ? this.dm.moveSelectionUp() : "down" === q ? this.dm.moveSelectionDown() : "bottom" === q && this.dm.moveSelectionToBottom()
            }, a.prototype.updatePriority = function () {
                var q = 0;
                this.dm.eachByHierarchical(function (a) {
                    1e4 === a.a("priority") ? q = 1e4 : 1e7 === a.a("priority") ? q = 1e7 : (q += 1, a.a("priority", q))
                })
            }, a.prototype.createRenderLayerName = function () {
                var q = 0;
                return this.dm.each(function (a) {
                    if (a.a("name").indexOf(Pa("editor.untitled") + "_") > -1) {
                        var k = Number(a.a("name").replace(Pa("editor.untitled") + "_", ""));
                        !isNaN(k) && Math.ceil(k) > q && (q = k)
                    }
                }), Pa("editor.untitled") + "_" + (q + 1)
            }, v(a, [{
                key: "dm",
                get: function () {
                    return this._tablePane.getDataModel()
                }
            }, {
                key: "sm",
                get: function () {
                    return this.dm.sm()
                }
            }]), a
        }(ht.widget.FormPane),
        qk = hteditor.getString,
        ak = function (q) {
            function a(k) {
                V(this, a);
                var w = Q(this, q.call(this));
                w.editor = k, w._map = {
                    light: "sceneLightGruopAlias",
                    flow: "sceneFlowEffectGroupAlias",
                    clipbox: "sceneClipboxGroupAlias"
                };
                var I = w._comboBox = new ht.widget.ComboBox;
                I.setLabels([qk("editor.grouptype.light"), qk("editor.grouptype.flow"), qk("editor.grouptype.clipbox")]), I.setValues(["light", "flow", "clipbox"]), I.setValue("light"), I.onValueChanged = function () {
                    w.parseGroup(w.editor.dm.a(w._map[I.getValue()]) || {})
                }, w.addRow([qk("editor.grouptype"), I], [hteditor.config.indent, .1]);
                var K = w._tablePane = new ht.widget.TablePane;
                return w.initTable(), w.addRow([K], [.1], .1), w
            }
            return E(a, q), a.prototype.isNameExist = function (q, a) {
                for (var k = !1, w = this.dm.getDatas(), I = 0, K = w.size(); I < K; I++) {
                    var b = w.get(I);
                    if (b !== a && (k = b.a("name") === q)) break
                }
                return k
            }, a.prototype.initTable = function () {
                var q = this._tablePane;
                q.getTableView().setEditable(!0), q.addColumns([this.$73$("id", qk("editor.id"), 60, !1), this.$73$("name", qk("editor.name"), 260, !0)]);
                for (var a = 1; a < 32; a++) this.addData(a, void 0);
                return q
            }, a.prototype.$73$ = function (q, a, k, w) {
                var I = this;
                return {
                    name: q,
                    tag: q,
                    width: k,
                    displayName: a,
                    align: "center",
                    editable: w,
                    accessType: "attr",
                    sortable: !1,
                    setValue: function (a, k, w) {
                        a.a(q, w), I.updateGroup()
                    }
                }
            }, a.prototype.updateGroup = function () {
                this.editor.dm.a(this._map[this._comboBox.getValue()], this.toGroupMap())
            }, a.prototype.toGroupMap = function () {
                var q = {};
                return this.dm.each(function (a) {
                    var k = a.a("name");
                    k && (q[a.a("id")] = k)
                }), q
            }, a.prototype.parseGroup = function (q) {
                this.dm.each(function (a) {
                    a.a("name", q[a.a("id")])
                })
            }, a.prototype.clearGroup = function () {
                this.dm.each(function (q) {
                    q.a("name", void 0)
                })
            }, a.prototype.addData = function (q, a) {
                var k = new ht.Data;
                k.a({
                    id: q,
                    name: a
                }), this.dm.add(k)
            }, a.prototype.isCurrent = function (q) {
                return this._map[this._comboBox.getValue()] === q
            }, v(a, [{
                key: "dm",
                get: function () {
                    return this._tablePane.getDataModel()
                }
            }, {
                key: "sm",
                get: function () {
                    return this.dm.sm()
                }
            }]), a
        }(ht.widget.FormPane),
        kk = R.getString,
        wk = function () {
            function K(q) {
                var a = this;
                V(this, K), this.params = q, this._eventNotifier = new ht.Notifier, this._clones = [], this.modelDialogs = {}, this.$38$ = q.$38$;
                var k = ht.Default.getClass(hteditor.config.serviceClass || l);
                this.$39$ = new k(this.handleServiceEvent.bind(this), this), ht.Default.handleModelLoaded = function (q, k) {
                    a.inspector && a.inspector.updateProperties()
                };
                var w = this.serializer = new ht.JSONSerializer;
                w.isSerializable = function (q) {
                    if (a.scene.sm().contains(q)) return !0;
                    var k = !1;
                    return w._blocks && w._blocks.each(function (a) {
                        q.isDescendantOf(a) && (k = !0)
                    }), k
                };
                var I = this.serializer.getProperties;
                w.getProperties = function (q) {
                    var a = I.call(w, q);
                    return hteditor.config.cloneTag || delete a.tag, a
                }, this.editable = !0, this.modelView
            }
            return K.prototype.addEventListener = function () {
                hteditor.Editor.prototype.addEventListener.apply(this, arguments)
            }, K.prototype.removeEventListener = function (q, a) {
                hteditor.Editor.prototype.removeEventListener.apply(this, arguments)
            }, K.prototype.fireEvent = function (q, a) {
                hteditor.Editor.prototype.fireEvent.apply(this, arguments)
            }, K.prototype.handleServiceEvent = function (K) {
                if ("connected" === K.type) this.init && (this.init(), this.init = null);
                else if ("fileChanged" === K.type) {
                    var b = K.path;
                    if (r(b)) {
                        var s = this.scene,
                            u = s.getTextureMap();
                        u[b] && s.deleteTexture(b), s.invalidateAll(), this.gv.invalidateAll()
                    }
                    q(b) ? this.requestScenes() : a(b) ? (this.requestModels(), this.scene.invalidateAll(), this.gv.invalidateAll()) : k(b) ? this.requestSymbols() : w(b) ? this.requestAssets() : I(b) && this.requestMaterials()
                } else "download" === K.type ? this.downloadFile(K.path) : "confirm" === K.type && this.requestImport(K.path);
                this.fireEvent(K.type, K)
            }, K.prototype.downloadFile = function (q) {
                hteditor.Editor.prototype.downloadFile.apply(this, arguments)
            }, K.prototype.requestImport = function (q) {
                hteditor.Editor.prototype.requestImport.apply(this, arguments)
            }, K.prototype.beginTransaction = function () {
                this.dm.beginTransaction()
            }, K.prototype.endTransaction = function () {
                this.dm.endTransaction()
            }, K.prototype.init = function () {
                var q = this;
                this.menus = [], this.dm = new ht.DataModel, this.gv = new aq(this), this.rulerView = new U(this, this.gv), this.eventView = new hteditor.EventView(this), this.scene = new P(this), this.functionView = new hteditor.FunctionView(this), this.fontView = new hteditor.FontView(this), this.objectView = new hteditor.ObjectView(this), this.messageView = new hteditor.MessageView(this), this.mainToolbar = new c(this), this.fireEvent("mainToolbarCreated"), this.rightToolbar = new i(this), this.fireEvent("rightToolbarCreated"), this.topBorderPane = new ht.widget.BorderPane, this.topBorderPane.setCenterView(this.mainToolbar), this.topBorderPane.setRightView(this.rightToolbar), this.topBorderPane.setHeight(28), this.rightToolbar.onSumWidthChanged = function () {
                    q.topBorderPane.setRightWidth(q.rightToolbar.getSumWidth())
                }, this.resetInteractionState(), this.mainTabView = new ht.widget.TabView, this.scenes = new _(this), this.models = new F(this), this.symbols = new S(this), this.assets = new d(this), this.materials = new B(this), this.leftTopTabView = new ht.widget.TabView, W(this.leftTopTabView, kk("editor.scenes"), this.scenes, !0), W(this.leftTopTabView, kk("editor.models"), this.models), W(this.leftTopTabView, kk("editor.symbols"), this.symbols), W(this.leftTopTabView, kk("editor.materials"), this.materials), W(this.leftTopTabView, kk("editor.assets"), this.assets), this.leftTopTabView.onTabChanged = function (q, a) {
                    if (a) {
                        var k = a.getView();
                        0 === k.tree.sm().size() && k.tree.sm().ss(k.rootNode), a.getView().list.doLayout()
                    }
                }, this.treePane = new Ba(this), this.dataView = new kq(this), this.rightBottomTabView = new ht.widget.TabView, this.treePaneTab = W(this.rightBottomTabView, kk("editor.list"), this.treePane, !0), W(this.rightBottomTabView, kk("editor.data"), this.dataView), this.renderLayerView = new Ua(this), W(this.rightBottomTabView, kk("editor.renderlayer"), this.renderLayerView), this.groupView = new ak(this), W(this.rightBottomTabView, kk("editor.group"), this.groupView), R.config.batchEditable && (this.batchView = new Da(this), W(this.rightBottomTabView, kk("editor.batch"), this.batchView));
                var a = this.rightBottomTabView.getTabModel().sm();
                a.ms(function () {
                    var k = a.ld();
                    q.dataView.visible = (k ? k.getView() : null) === q.dataView
                }), this.alignPane = new Iq(this), this.rightTopBorderPane = new ht.widget.BorderPane, this.rightTopBorderPane.setTopView(this.alignPane), this.rightTopBorderPane.setCenterView(this.inspectorPane), this.rightTopBorderPane.setTopHeight(ht.Default.widgetHeaderHeight + 6), this.centerSplitView = new ht.widget.SplitView(this.rulerView, this.scene, "v"), this.leftSplitView = new ht.widget.SplitView(this.leftTopTabView, this.centerSplitView, "h", R.config.leftSplitViewPosition || 260), this.rightSplitView = new ht.widget.SplitView(this.rightTopBorderPane, this.rightBottomTabView, "v", R.config.rightSplitViewPosition || -300), this.mainSplitView = new ht.widget.SplitView(this.leftSplitView, this.rightSplitView, "h", R.config.mainSplitViewPosition || -360), this.mainPane = new ht.widget.BorderPane, this.mainPane.setTopView(this.topBorderPane), this.mainPane.setCenterView(this.mainSplitView), hteditor.layoutMainView(this.mainPane, this.$38$), this.modelAnimDialog = new za(this), this.$38$ && this.$38$.addEventListener("keydown", this.handleKeydown.bind(this), !1), this.dnd = new hteditor.DND(this), this.dndFromOutside = new hteditor.DNDFromOutside(this, this.$38$), this.requestScenes(), this.requestModels(), this.requestSymbols(), this.requestMaterials(), this.requestAssets(), this.dm.addDataModelChangeListener(this.handleDataModelChange, this), this.dm.addPropertyChangeListener(this.handleDataModelPropertyChange, this), this.dm.addDataPropertyChangeListener(this.handleDataPropertyChange, this), this.sm.ms(this.handleSelectionChange, this), this.scene.addPropertyChangeListener(this.handleScenelPropertyChange, this), this.scene.mi(this.handleSceneInteractive, this), this.gv.addInteractorListener(this.handleInteractor, this), this.updateInspector(), this.fireEvent("editor3dCreated"), this.dm.enableHistoryManager(hteditor.config.maxUndoRedoSteps), this.dm.getHistoryManager().ignoredPropertyMap = Object.assign({}, this.dm.getHistoryManager().ignoredPropertyMap, {
                    animationIteration: !0,
                    animationPause: !0,
                    animationResume: !0,
                    animationStart: !0,
                    animationStop: !0
                }), this.initSID();
                var k = this.params.open || T("hteditor") || hteditor.config.open;
                k && (hteditor.isJSON(k) ? this._pendingOpenJSON = k : "newscene" === k && this.newScene()), this.reset()
            }, K.prototype.initSID = function () {
                var q = window.location.href.match("sid=([0-9a-z-]*)");
                q && (this.sid = q[1])
            }, K.prototype.handleDataModelPropertyChange = function (q) {
                this.inspector && this.inspector.global && this.inspector.$34$(q), this.dataView.$66$(q);
                var a = this.scene,
                    k = q.property,
                    w = q.newValue;
                if ("a:sceneSkyboxType" === k || "a:sceneSkyboxBlurSize" === k || "a:sceneSkyboxBodyColor" === k || "a:sceneSkyboxRotationX" === k || "a:sceneSkyboxRotationY" === k || "a:sceneSkyboxRotationZ" === k || "a:sceneSkyboxImage" === k || "a:sceneSkyboxFrontImage" === k || "a:sceneSkyboxBackImage" === k || "a:sceneSkyboxLeftImage" === k || "a:sceneSkyboxRightImage" === k || "a:sceneSkyboxTopImage" === k || "a:sceneSkyboxBottomImage" === k || "a:sceneSkyboxDiameter" === k || "a:sceneSkyboxColor" === k) {
                    var I = {};
                    I["body.color"] = a.dm().a("sceneSkyboxBodyColor"), I["all.color"] = "rgba(0,0,0,0)", I["shape3d.color"] = "rgba(0,0,0,0)", "cube" === a.dm().a("sceneSkyboxType") ? (I["front.image"] = a.dm().a("sceneSkyboxFrontImage"), I["back.image"] = a.dm().a("sceneSkyboxBackImage"), I["left.image"] = a.dm().a("sceneSkyboxLeftImage"), I["right.image"] = a.dm().a("sceneSkyboxRightImage"), I["top.image"] = a.dm().a("sceneSkyboxTopImage"), I["bottom.image"] = a.dm().a("sceneSkyboxBottomImage")) : "color" === a.dm().a("sceneSkyboxType") ? (I.shape3d = "sphere", I["shape3d.color"] = a.dm().a("sceneSkyboxColor") || "rgba(0,0,0,0)") : (I.shape3d = "sphere", I["shape3d.image"] = a.dm().a("sceneSkyboxImage"));
                    var K = new ht.Node;
                    K.setRotationX(a.dm().a("sceneSkyboxRotationX")), K.setRotationY(a.dm().a("sceneSkyboxRotationY")), K.setRotationZ(a.dm().a("sceneSkyboxRotationZ")), K.s(I);
                    var b = void 0;
                    if (a.dm().a("sceneSkyboxDiameter")) b = a.dm().a("sceneSkyboxDiameter"), K.s3(b, b, b);
                    else {
                        var s = a.dm().a("sceneNear"),
                            u = a.dm().a("sceneFar");
                        u = u * u / (u + s);
                        var C = Math.sqrt(3),
                            L = u / C,
                            j = Math.log(L) / Math.log(1.1);
                        b = 2 * Math.pow(1.1, j), a.dm().a("sceneSkyboxDiameter", b)
                    }
                    a.setSkybox(K), this.inspector.filterPropertiesLater()
                } else if ("a:sceneRenderLayerInfoMap" === k) {
                    var G = w;
                    void 0 === w && (G = {
                        main: ht.Default.getRenderLayerInfo("main"),
                        top: ht.Default.getRenderLayerInfo("top")
                    }), a.setRenderLayerInfoMap(G);
                    var X = this.renderLayerView;
                    X && (X.parseRenderLayerInfoMap(G), this.inspector && this.inspector.filterPropertiesLater())
                } else if ("a:sceneLightGruopAlias" === k || "a:sceneFlowEffectGroupAlias" === k || "a:sceneClipboxGroupAlias" === k) {
                    var Z = this.groupView;
                    Z && Z.isCurrent(k.replace("a:", "")) && Z.parseGroup(w)
                } else "a:sceneDofImage" === k && a.setPostProcessingValue("Dof", "image", w);
                if (null != w)
                    if ("a:sceneEditHelperDisabled" === k) a.setEditHelperDisabled(w), this.rightToolbar.iv();
                    else if ("a:sceneGridEnabled" === k) a.setGridVisible(w), a.setOriginAxisVisible(w), this.gv.getEditInteractor().gridEnabled = w, this.rightToolbar.iv();
                else if ("a:sceneGridBlockCount" === k) a.setGridSize(w);
                else if ("a:sceneGridBlockSize" === k) a.setGridGap(w), this.gv.setEditStyle("gridBlockSize", w);
                else if ("a:sceneGridColor" === k) {
                    var r = ht.Default.toColorData(w);
                    a.setGridColor([r[0] / 255, r[1] / 255, r[2] / 255, 1]), this.gv.setEditStyle("gridThickColor", w), this.gv.setEditStyle("gridLightColor", "rgba(" + r[0] + ", " + r[1] + ", " + r[2] + ", 0.4)")
                } else if ("a:sceneNear" === k) a.setNear(w);
                else if ("a:sceneFar" === k) a.setFar(w);
                else if ("a:sceneBloom" === k) a.setPostProcessingValue("Bloom", "enable", w);
                else if ("a:sceneBloomStrength" === k) a.setPostProcessingValue("Bloom", "strength", w);
                else if ("a:sceneBloomThreshold" === k) a.setPostProcessingValue("Bloom", "threshold", w);
                else if ("a:sceneBloomRadius" === k) a.setPostProcessingValue("Bloom", "radius", w);
                else if ("a:sceneBloomSelective" === k) a.setPostProcessingValue("Bloom", "selective", w);
                else if ("a:sceneDof" === k) a.setPostProcessingValue("Dof", "enable", w);
                else if ("a:sceneDofAperture" === k) a.setPostProcessingValue("Dof", "aperture", w);
                else if ("a:sceneHighlightMode" === k) a.setHighlightMode(w);
                else if ("a:sceneHighlightType" === k) a.setHighlightType(w);
                else if ("a:sceneHighlightWidth" === k) a.setHighlightWidth(w);
                else if ("a:sceneHighlightColor" === k) a.setHighlightColor(w);
                else if ("a:sceneHighlightGlow" === k) a.setHighlightGlow(w);
                else if ("a:sceneHighlightStrength" === k) a.setHighlightStrength(w);
                else if ("a:sceneDashEnable" === k) a.setDashDisabled(!w);
                else if ("a:sceneHeadlightEnable" === k) a.setHeadlightDisabled(!w);
                else if ("a:sceneHeadlightRange" === k) a.setHeadlightRange(w);
                else if ("a:sceneHeadlightColor" === k) a.setHeadlightColor(w);
                else if ("a:sceneHeadlightIntensity" === k) a.setHeadlightIntensity(w);
                else if ("a:sceneHeadlightAmbientIntensity" === k) a.setHeadlightAmbientIntensity(w);
                else if ("a:sceneHueSaturation" === k) a.setPostProcessingValue("HueSaturation", "enable", w);
                else if ("a:sceneHueSaturationHue" === k) a.setPostProcessingValue("HueSaturation", "hue", w);
                else if ("a:sceneHueSaturationSaturation" === k) a.setPostProcessingValue("HueSaturation", "saturation", w);
                else if ("a:sceneHueSaturationLightness" === k) a.setPostProcessingValue("HueSaturation", "lightness", w);
                else if ("a:sceneFogEnable" === k) a.setFogDisabled(!w);
                else if ("a:sceneFogMode" === k) a.setFogMode(w);
                else if ("a:sceneFogColor" === k) a.setFogColor(w);
                else if ("a:sceneFogDensity" === k) a.setFogDensity(w);
                else if ("a:sceneFogNear" === k) a.setFogNear(w);
                else if ("a:sceneFogFar" === k) a.setFogFar(w);
                else if ("a:sceneBatchBrightnessDisabled" === k) a.setBatchBrightnessDisabled(w);
                else if ("a:sceneBatchBlendDisabled" === k) a.setBatchBlendDisabled(w);
                else if ("a:sceneBatchColorDisabled" === k) a.setBatchColorDisabled(w);
                else if ("a:sceneBatchInstancedDisabled" === k) a.setBatchInstancedDisabled(w);
                else if ("a:sceneShadowEnabled" === k) w ? a.enableShadow() : a.disableShadow();
                else if ("a:sceneShadowDegreeX" === k) a.setShadowDegreeX(w);
                else if ("a:sceneShadowDegreeZ" === k) a.setShadowDegreeZ(w);
                else if ("a:sceneShadowIntensity" === k) a.setShadowIntensity(w);
                else if ("a:sceneShadowQuality" === k) a.setShadowQuality(w);
                else if ("a:sceneShadowType" === k) a.setShadowType(w);
                else if ("a:sceneShadowRadius" === k) a.setShadowRadius(w);
                else if ("a:sceneShadowBias" === k) a.setShadowBias(w);
                else if ("a:sceneBatchInfoMap" === k) {
                    a.setBatchInfoMap(w);
                    var W = this.batchView;
                    W && (W.parseBatchInfoMap(w), W.updateProperties())
                } else "a:sceneOrthographic" === k && this._setSceneOrtho(w)
            }, K.prototype._setSceneOrtho = function (q) {
                var a = this.scene,
                    k = this.__updateEye__;
                k || (k = this.__updateEye__ = function () {
                    var q = new ht.Math.Vector3,
                        k = new ht.Math.Vector3,
                        w = new ht.Math.Vector3;
                    return function (I) {
                        if (I) {
                            q.fromArray(a.getCenter()), k.fromArray(a.getEye()), k.sub(q);
                            var K = [0, 1, 0];
                            "right" === I ? K = [1, 0, 0] : "front" === I && (K = [0, 0, 1]), w.fromArray(K).setLength(k.length()).add(q), a.setEye(w.toArray())
                        }
                    }
                }());
                var w = this.__defaultInteractor__;
                if (!w)
                    for (var I = a.getInteractors(), K = 0, b = I.length; K < b; K++) {
                        var s = I.get(0);
                        if (s instanceof ht.graph3d.DefaultInteractor) {
                            w = this.__defaultInteractor__ = s;
                            break
                        }
                    }
                var u = this.__orthoPanInteractor__;
                u || (u = this.__orthoPanInteractor__ = new oa(a)), a.setOrtho(!!q), q ? (a.setInteractors([u]), k(q)) : a.setInteractors([w])
            }, K.prototype.handleScenelPropertyChange = function (q) {
                this.inspector && this.inspector.global && this.inspector.$34$(q), this._sceneInteracting || this.dataView.$66$(q)
            }, K.prototype.handleSceneInteractive = function (q) {
                this._sceneInteracting = "betweenRotate" === q.kind || "betweenPan" === q.kind, this._sceneInteracting || this.dataView.$66$(q)
            }, K.prototype.handleDataModelChange = function (q) {
                this.inspector && this.inspector.global && this.inspector.updateProperties(), this.dataView.$66$(q)
            }, K.prototype.handleDataPropertyChange = function (q) {
                var a = q.data;
                if ("parent" === q.property && a instanceof ht.Node && !ht.Default.isIsolating()) {
                    var k = a.getParent();
                    k instanceof ht.Block && (k = null), (!k || k instanceof ht.Node) && a.setHost(k)
                }
                this.inspector && !this.inspector.global && this.inspector.data === a && this.inspector.$34$(q), this.dataView.$66$(q)
            }, K.prototype.handleSelectionChange = function () {
                this.updateInspector()
            }, K.prototype.handleInteractor = function (q) {
                "selectPoint" === q.kind && (this.currentPoint = void 0 === q.x ? null : {
                    x: q.x,
                    y: q.y,
                    e: q.e
                }, this.updateInspector())
            }, K.prototype.updateInspector = function () {
                this.inspector = null;
                var q = this.ld;
                if (q) {
                    var a = q.s("shape3d");
                    q instanceof ht.Block ? this.inspector = this.blockInspector : q instanceof ht.Polyline ? this.inspector = this.polylineInspector : q instanceof ht.Shape ? "cylinder" === a ? this.inspector = this.pipelineInspector : q.getThickness() > 0 ? this.inspector = this.wallInspector : this.inspector = this.floorInspector : q instanceof ht.Light ? this.inspector = this.lightInspector : q instanceof ht.Node ? this.inspector = a ? "billboard" === a || "plane" === a ? this.billboardInspector : "cube" === a ? this.cubeInspector : "box" === a ? this.boxInspector : "cone" === a ? this.coneInspector : "cylinder" === a ? this.cylinderInspector : "roundRect" === a ? this.roundRectInspector : "sphere" === a ? this.sphereInspector : "torus" === a ? this.torusInspector : "triangle" === a ? this.triangleInspector : "star" === a || "rect" === a || "rightTriangle" === a || "parallelogram" === a || "trapezoid" === a ? this.dataInspector : this.modelInspector : this.cubeInspector : q instanceof ht.Edge && (this.inspector = this.edgeInspector), this.inspector || (this.inspector = this.dataInspector)
                } else this.inspector = this.sceneInspector;
                this.inspector && (this.inspector.data = q, this.inspector.filterPropertiesLater()), this.rightTopBorderPane.setCenterView(this.inspector)
            }, K.prototype.reset = function (q) {
                q || (this.url = null);
                var a = this.dm;
                a.disableHistoryManager(), a.setPostProcessingData(void 0), a.setName(void 0), a.setBackground(void 0), a.setHierarchicalRendering(!0), a.setLayers(void 0), a.setAttrObject(void 0), a.setEnvmap(void 0), a.setEnvmapType(void 0), a.clear();
                var k = this.gv;
                k.setZoom(1), k.tx(0), k.ty(0);
                var w = this.scene;
                w.setEye(R.config.sceneEye), w.setCenter(R.config.sceneCenter), a.a("sceneNear", R.config.sceneNear), a.a("sceneFar", R.config.sceneFar), a.a("connectActionType", R.config.sceneConnectActionType), a.a("sceneEditHelperDisabled", R.config.sceneEditHelperDisabled), a.a("sceneGridEnabled", R.config.sceneGridEnabled), a.a("sceneGridBlockCount", R.config.sceneGridBlockCount), a.a("sceneGridBlockSize", R.config.sceneGridBlockSize), a.a("sceneGridColor", R.config.sceneGridColor), a.a("sceneBatchBrightnessDisabled", R.config.sceneBatchBrightnessDisabled), a.a("sceneBatchBlendDisabled", R.config.sceneBatchBlendDisabled), a.a("sceneBatchColorDisabled", R.config.sceneBatchColorDisabled), a.a("sceneBatchInstancedDisabled", R.config.sceneBatchInstancedDisabled), a.a("sceneSkyboxType", R.config.sceneSkyboxType), a.a("sceneHighlightMode", R.config.sceneHighlightMode), a.a("sceneHighlightType", R.config.sceneHighlightType), a.a("sceneHighlightColor", R.config.sceneHighlightColor), a.a("sceneHighlightWidth", R.config.sceneHighlightWidth), a.a("sceneDashEnable", R.config.sceneDashEnable), a.a("sceneHeadlightEnable", R.config.sceneHeadlightEnable), a.a("sceneHeadlightRange", R.config.sceneHeadlightRange), a.a("sceneHeadlightColor", R.config.sceneHeadlightColor), a.a("sceneHeadlightIntensity", R.config.sceneHeadlightIntensity), a.a("sceneHeadlightAmbientIntensity", R.config.sceneHeadlightAmbientIntensity), a.a("sceneHighlightGlow", R.config.sceneHighlightGlow), a.a("sceneHighlightStrength", R.config.sceneHighlightStrength), a.a("sceneFogEnable", R.config.sceneFogEnable), a.a("sceneFogMode", R.config.sceneFogMode), a.a("sceneFogColor", R.config.sceneFogColor), a.a("sceneFogNear", R.config.sceneFogNear), a.a("sceneFogFar", R.config.sceneFogFar), a.a("sceneFogDensity", R.config.sceneFogDensity), a.a("sceneShadowEnabled", R.config.sceneShadowEnabled), a.a("sceneShadowDegreeX", R.config.sceneShadowDegreeX), a.a("sceneShadowDegreeZ", R.config.sceneShadowDegreeZ), a.a("sceneShadowIntensity", R.config.sceneShadowIntensity), a.a("sceneShadowQuality", R.config.sceneShadowQuality), a.a("sceneShadowType", R.config.sceneShadowType), a.a("sceneShadowRadius", R.config.sceneShadowRadius), a.a("sceneShadowBias", R.config.sceneShadowBias), a.a("sceneDof", R.config.sceneDof), a.a("sceneDofAperture", R.config.sceneDofAperture), a.a("sceneDofImage", R.config.sceneDofImage), a.a("sceneBloom", R.config.sceneBloom), a.a("sceneBloomStrength", R.config.sceneBloomStrength), a.a("sceneBloomThreshold", R.config.sceneBloomThreshold), a.a("sceneBloomRadius", R.config.sceneBloomRadius), a.a("sceneHueSaturation", R.config.sceneHueSaturation), a.a("sceneHueSaturationHue", R.config.sceneHueSaturationHue), a.a("sceneHueSaturationColorIndex", R.config.sceneHueSaturationColorIndex), a.a("sceneHueSaturationSaturation", R.config.sceneHueSaturationSaturation), a.a("sceneHueSaturationLightness", R.config.sceneHueSaturationLightness), a.a("sceneOrthographic", R.config.sceneOrthographic), w.setSkybox(null), w.setBatchInfoMap({}), this.batchView && this.batchView.clearBatchInfo(), this.renderLayerView.clearRenderLayerInfo(), this.groupView.clearGroup(), a.enableHistoryManager(), a.clearHistoryManager()
            }, K.prototype.getModelDialog = function (q) {
                var a = this.modelDialogs[q];
                if (!a) {
                    var k = hteditor.config.modelDialogClasses ? hteditor.config.modelDialogClasses[q] : null;
                    k || (k = ma), a = this.modelDialogs[q] = new k(this)
                }
                return a
            }, K.prototype.open = function (q, a) {
                var k = this;
                if (ht.Default.isString(q) && (q = this.getFileNode(q)), q && !q.$33$) {
                    var w = q.url,
                        I = q.path,
                        K = q.getName ? q.getName() : void 0;
                    "scene" === q.fileType ? this.newScene(w) : "model" === q.fileType ? ht.Default.xhrLoad(w, function (q) {
                        q = ht.Default.parse(q), k.getModelDialog(q.modelType).open(w, q)
                    }) : "material" === q.fileType && ht.Default.xhrLoad(w, function (q) {
                        q = ht.Default.parse(q), k.materialView.open(w, q, I, K)
                    })
                }
            }, K.prototype.newOBJModel = function () {
                this.modelView.open(void 0, {
                    modelType: "obj"
                })
            }, K.prototype.newFBXModel = function () {
                this.modelView.open(void 0, {
                    modelType: "fbx"
                })
            }, K.prototype.newGLTFModel = function () {
                this.modelView.open(void 0, {
                    modelType: "gltf"
                })
            }, K.prototype.newMaterial = function () {
                this.materialView.open()
            }, K.prototype.newScene = function (q) {
                q ? (this.opened = !1, this.url = q, this.reload()) : (this.reset(), this.fireEvent("sceneViewCreated", {
                    sceneView: this
                }))
            }, K.prototype.readForOld3dEditorFormat = function (q) {
                var a = this.dm,
                    k = q.scene;
                if (k && (k.near && a.a("sceneNear", k.near), k.far && a.a("sceneFar", k.far), null != k.shadow && k.shadowParams)) {
                    a.a("sceneShadowEnabled", k.shadow);
                    var w = k.shadowParams;
                    for (var I in w) {
                        var K = w[I];
                        null != K && a.a("sceneShadow" + I.charAt(0).toUpperCase() + I.substr(1), K)
                    }
                }
                var b = a.getPostProcessingData();
                if (b) {
                    a.setPostProcessingData(void 0);
                    var s = b.bloom;
                    s && (null != s.enable && a.a("sceneBloom", s.enable), null != s.strength && a.a("sceneBloomStrength", s.strength), null != s.threshold && a.a("sceneBloomThreshold", s.threshold), null != s.radius && a.a("sceneBloomRadius", s.radius));
                    var u = b.dof;
                    u && (null != u.enable && a.a("sceneDof", u.enable), null != u.aperture && a.a("sceneDofAperture", u.aperture), null != u.image && a.a("sceneDofImage", u.image))
                }
            }, K.prototype.reload = function () {
                var q = this,
                    a = this.url;
                if (a) {
                    this.fireEvent(this.opened ? "SceneViewReloading" : "SceneViewOpening", {
                        sceneView: this,
                        url: a
                    });
                    var k = this.scene;
                    this.reset(!0), this.dm.disableHistoryManager(), k.deserialize(a, {
                        setId: R.config.setId,
                        finishFunc: function (k, w) {
                            q.gv.fitContent(!0), q.readForOld3dEditorFormat(k);
                            var I = k.scene;
                            I && (I.eye && q.scene.setEye(I.eye), I.center && q.scene.setCenter(I.center)), w.setPostProcessingData(void 0), q.dm.enableHistoryManager(), q.dm.clearHistoryManager(), q.fireEvent(q.opened ? "SceneViewReloaded" : "SceneViewOpened", {
                                sceneView: q,
                                url: a
                            }), q.opened = !0
                        },
                        disableOnPreDeserialize: !0,
                        disableOnPostDeserialize: !0
                    })
                }
            }, K.prototype.save = function () {
                var q = this;
                if (this.url) this.saveImpl(this.url);
                else {
                    var a = this.scenes.currentDir + "/",
                        k = kk("editor.inputnewscenename"),
                        w = function (a, k, w) {
                            if ("ok" === k) {
                                var I = hteditor.isJSON(a) ? hteditor.trimExtension(a) : a,
                                    K = w + I + ".json",
                                    b = !1;
                                if (q.scenes.dataModel.each(function (q) {
                                        q.url === K && (b = !0)
                                    }), b) {
                                    var s = [{
                                            label: kk("editor.ok"),
                                            action: function () {
                                                u.hide()
                                            }
                                        }],
                                        u = new ht.widget.Dialog({
                                            title: kk("editor.filenameconflict"),
                                            contentPadding: 20,
                                            width: 300,
                                            draggable: !0,
                                            content: "<p>" + K + "</p>",
                                            buttons: s
                                        });
                                    u.show()
                                } else q.saveImpl(K)
                            }
                        },
                        I = {
                            name: void 0,
                            scene: this.scene
                        };
                    this.fireEvent("sceneNewNameInputing", I), I.name ? w(I.name, "ok", a) : hteditor.getInput(k, "", {
                        nullable: !1,
                        trim: !0,
                        maxLength: hteditor.config.maxFileNameLength,
                        path: a,
                        editor: this,
                        checkFunc: R.config.checkFileName,
                        $38$: this.$38$
                    }, w)
                }
            }, K.prototype.saveImpl = function (q) {
                var a = this;
                if (q) {
                    this.url = q;
                    var k = {
                        url: q,
                        scene: this.scene
                    };
                    if (this.fireEvent("sceneSaving", k), k.preventDefault) return !1;
                    k = {
                        path: k.url,
                        content: this.serialize()
                    }, this.request("upload", k, function (w) {
                        !0 === w && (a.fireEvent("sceneSaved", k), a.messageView.show(kk("editor.savedsuccessfully")), k = {
                            path: q.substr(0, q.length - 5) + ".png",
                            content: a.scene.toImage(a.dm.getBackground())
                        }, a.request("upload", k, function (k) {
                            !0 === k && a.selectFileNode(q)
                        }))
                    })
                }
            }, K.prototype.preview = function () {
                var q = this.dm.a("previewURL") || "scene.html";
                this.url && -1 === q.indexOf("?") && (q += "?tag=" + encodeURI(this.url));
                var a = {
                    url: q
                };
                this.fireEvent("sceneViewPreviewing", a), window.open(a.url, a.url), a = {
                    path: "previews/scene.json",
                    content: this.serialize()
                }, this.request("upload", a, function (q) {})
            }, K.prototype.deserialize = function (q) {
                this.scene.deserialize(q, function (q, a, k) {})
            }, K.prototype.serialize = function () {
                return o(this.toJSON())
            }, K.prototype.toJSON = function () {
                var q = this.dm,
                    a = q.toJSON(),
                    k = {},
                    w = ht.Default.clone(this.scene.getEye());
                w.join(",") !== R.config.sceneEye.join(",") && (k.eye = w);
                var I = ht.Default.clone(this.scene.getCenter());
                I.join(",") !== R.config.sceneCenter.join(",") && (k.center = I), ht.Default.isEmptyObject(k) || (a.scene = k);
                var K = this.scene.getBatchInfoMap();
                ht.Default.isEmptyObject(K) || (a.a.sceneBatchInfoMap = K);
                var b = this.scene.getRenderLayerInfoMap();
                return ht.Default.isEmptyObject(b) || ht.Default.stringify(b) !== ht.Default.stringify({
                        main: ht.Default.getRenderLayerInfo("main"),
                        top: ht.Default.getRenderLayerInfo("top")
                    }) && (a.a.sceneRenderLayerInfoMap = b), q.a("sceneNear") === R.config.sceneNear && delete a.a.sceneNear, q.a("sceneFar") === R.config.sceneFar && delete a.a.sceneFar, q.a("sceneEditHelperDisabled") === R.config.sceneEditHelperDisabled && delete a.a.sceneEditHelperDisabled, q.a("sceneGridEnabled") === R.config.sceneGridEnabled && delete a.a.sceneGridEnabled,
                    q.a("sceneGridBlockCount") === R.config.sceneGridBlockCount && delete a.a.sceneGridBlockCount, q.a("sceneGridBlockSize") === R.config.sceneGridBlockSize && delete a.a.sceneGridBlockSize, q.a("sceneGridColor") === R.config.sceneGridColor && delete a.a.sceneGridColor, q.a("sceneBatchBrightnessDisabled") === R.config.sceneBatchBrightnessDisabled && delete a.a.sceneBatchBrightnessDisabled, q.a("sceneBatchBlendDisabled") === R.config.sceneBatchBlendDisabled && delete a.a.sceneBatchBlendDisabled, q.a("sceneBatchColorDisabled") === R.config.sceneBatchColorDisabled && delete a.a.sceneBatchColorDisabled, q.a("sceneBatchInstancedDisabled") === R.config.sceneBatchInstancedDisabled && delete a.a.sceneBatchInstancedDisabled, q.a("sceneSkyboxType") === R.config.sceneSkyboxType && delete a.a.sceneSkyboxType, q.a("sceneHighlightMode") === R.config.sceneHighlightMode && delete a.a.sceneHighlightMode, q.a("sceneHighlightType") === R.config.sceneHighlightType && delete a.a.sceneHighlightType, q.a("sceneHighlightColor") === R.config.sceneHighlightColor && delete a.a.sceneHighlightColor, q.a("sceneHighlightWidth") === R.config.sceneHighlightWidth && delete a.a.sceneHighlightWidth, q.a("sceneHighlightGlow") === R.config.sceneHighlightGlow && delete a.a.sceneHighlightGlow, q.a("sceneHighlightStrength") === R.config.sceneHighlightStrength && delete a.a.sceneHighlightStrength, q.a("sceneDashEnable") === R.config.sceneDashEnable && delete a.a.sceneDashEnable, q.a("sceneHeadlightEnable") === R.config.sceneHeadlightEnable && delete a.a.sceneHeadlightEnable, q.a("sceneHeadlightRange") === R.config.sceneHeadlightRange && delete a.a.sceneHeadlightRange, q.a("sceneHeadlightColor") === R.config.sceneHeadlightColor && delete a.a.sceneHeadlightColor, q.a("sceneHeadlightIntensity") === R.config.sceneHeadlightIntensity && delete a.a.sceneHeadlightIntensity, q.a("sceneHeadlightAmbientIntensity") === R.config.sceneHeadlightAmbientIntensity && delete a.a.sceneHeadlightAmbientIntensity, q.a("sceneFogEnable") === R.config.sceneFogEnable && delete a.a.sceneFogEnable, q.a("sceneFogColor") === R.config.sceneFogColor && delete a.a.sceneFogColor, q.a("sceneFogNear") === R.config.sceneFogNear && delete a.a.sceneFogNear, q.a("sceneFogFar") === R.config.sceneFogFar && delete a.a.sceneFogFar, q.a("sceneFogMode") === R.config.sceneFogMode && delete a.a.sceneFogMode, q.a("sceneFogDensity") === R.config.sceneFogDensity && delete a.a.sceneFogDensity, q.a("sceneShadowEnabled") === R.config.sceneShadowEnabled && delete a.a.sceneShadowEnabled, q.a("sceneShadowDegreeX") === R.config.sceneShadowDegreeX && delete a.a.sceneShadowDegreeX, q.a("sceneShadowDegreeZ") === R.config.sceneShadowDegreeZ && delete a.a.sceneShadowDegreeZ, q.a("sceneShadowIntensity") === R.config.sceneShadowIntensity && delete a.a.sceneShadowIntensity, q.a("sceneShadowQuality") === R.config.sceneShadowQuality && delete a.a.sceneShadowQuality, q.a("sceneShadowType") === R.config.sceneShadowType && delete a.a.sceneShadowType, q.a("sceneShadowRadius") === R.config.sceneShadowRadius && delete a.a.sceneShadowRadius, q.a("sceneShadowBias") === R.config.sceneShadowBias && delete a.a.sceneShadowBias, q.a("sceneDof") === R.config.sceneDof && delete a.a.sceneDof, q.a("sceneDofAperture") === R.config.sceneDofAperture && delete a.a.sceneDofAperture, q.a("sceneDofImage") === R.config.sceneDofImage && delete a.a.sceneDofImage, q.a("sceneBloom") === R.config.sceneBloom && delete a.a.sceneBloom, q.a("sceneBloomStrength") === R.config.sceneBloomStrength && delete a.a.sceneBloomStrength, q.a("sceneBloomThreshold") === R.config.sceneBloomThreshold && delete a.a.sceneBloomThreshold, q.a("sceneBloomRadius") === R.config.sceneBloomRadius && delete a.a.sceneBloomRadius, q.a("sceneHueSaturation") === R.config.sceneHueSaturation && delete a.a.sceneHueSaturation, q.a("sceneHueSaturationHue") === R.config.sceneHueSaturationHue && delete a.a.sceneHueSaturationHue, q.a("sceneHueSaturationColorIndex") && delete a.a.sceneHueSaturationColorIndex, q.a("sceneHueSaturationSaturation") === R.config.sceneHueSaturationSaturation && delete a.a.sceneHueSaturationSaturation, q.a("sceneHueSaturationLightness") === R.config.sceneHueSaturationLightness && delete a.a.sceneHueSaturationLightness, a
            }, K.prototype.$3$ = function () {
                return this._interactionState
            }, K.prototype.$4$ = function (q, a) {
                var k = this;
                if (this._interactionState !== q) {
                    this._interactionState = q, this.pointsEditingMode = !1;
                    var w = [];
                    this.gv.getInteractors().each(function (a) {
                        a.keep ? a.interactiveDisabled = "edit" !== q : w.push(a)
                    }), w.forEach(function (q) {
                        q.tearDown(), k.gv.getInteractors().remove(q)
                    }), "edit" !== q && (a.setUp(), this.gv.getInteractors().add(a)), this.gv.invalidateSelection(), this.rulerView.validateCanvas(), this.mainToolbar.iv()
                }
            }, K.prototype.request = function (q, a, k, w) {
                var I = this;
                k = k || n, w ? setTimeout(function () {
                    I.$39$.request(q, a, k)
                }, w) : this.$39$.request(q, a, k)
            }, K.prototype.requestScenes = function () {
                var a = this;
                this._requestingScenes || (this.request("explore", "/scenes", function (k) {
                    a._requestingScenes = !1, a.scenes.parse(k), a._pendingOpenJSON && q(a._pendingOpenJSON) && (a.scenes.dataModel.getDataById(a._pendingOpenJSON) ? (a.open(a._pendingOpenJSON), a.selectFileNode(a._pendingOpenJSON)) : R.config.newIfFailToOpen && (a.newScene(), a.url = a._pendingOpenJSON, a.save()), delete a._pendingOpenJSON), a._pendingSelectURL && q(a._pendingSelectURL) && a.selectFileNode(a._pendingSelectURL)
                }, hteditor.config.requestDelay), this._requestingScenes = !0)
            }, K.prototype.requestSymbols = function () {
                hteditor.Editor.prototype.requestSymbols.apply(this, arguments)
            }, K.prototype.requestMaterials = function () {
                var q = this;
                this._requestingMaterials || (this.request("explore", "/materials", function (a) {
                    q._requestingMaterials = !1, q.materials.parse(a), q._pendingOpenJSON && I(q._pendingOpenJSON) && (q.materials.dataModel.getDataById(q._pendingOpenJSON) ? (q.open(q._pendingOpenJSON), q.selectFileNode(q._pendingOpenJSON)) : R.config.newIfFailToOpen && (q.newScene(), q.url = q._pendingOpenJSON, q.save()), delete q._pendingOpenJSON), q._pendingSelectURL && I(q._pendingSelectURL) && q.selectFileNode(q._pendingSelectURL)
                }, hteditor.config.requestDelay), this._requestingMaterials = !0)
            }, K.prototype.requestModels = function () {
                hteditor.Editor.prototype.requestModels.apply(this, arguments)
            }, K.prototype.requestAssets = function () {
                hteditor.Editor.prototype.requestAssets.apply(this, arguments)
            }, K.prototype.requestBase64 = function (q, a) {
                this.request("source", {
                    url: q,
                    encoding: "base64",
                    prefix: "data:;base64,"
                }, a)
            }, K.prototype.copy = function () {
                if (this.ld) {
                    this.serializer.dm = this.scene.dm(), this.serializer._blocks = this.scene.sm().toSelection(function (q) {
                        return q instanceof ht.Block
                    });
                    var q = this.serializer.toJSON();
                    delete q.p, delete q.a, this._cloneJSON = ht.Default.stringify(q)
                }
            }, K.prototype.paste = function () {
                var q = this;
                if (this._cloneJSON) {
                    var a = new ht.DataModel;
                    new ht.JSONSerializer(a).deserialize(this._cloneJSON), a.each(function (a) {
                        q.dm.getDataByTag(a.getTag()) && a.setTag(void 0)
                    });
                    var k = ht.Default.parse(a.serialize());
                    delete k.p;
                    var w = this.serializer.deserialize(k);
                    this.scene.sm().ss(w)
                }
            }, K.prototype.hasCopyInfo = function () {
                return !!this._cloneJSON
            }, K.prototype.undo = function () {
                hteditor.Editor.prototype.undo.apply(this, arguments)
            }, K.prototype.redo = function () {
                hteditor.Editor.prototype.redo.apply(this, arguments)
            }, K.prototype.renameFile = function () {
                hteditor.Editor.prototype.renameFile.apply(this, arguments)
            }, K.prototype.getFileNode = function () {
                return hteditor.Editor.prototype.getFileNode.apply(this, arguments)
            }, K.prototype.moveFile = function () {
                hteditor.Editor.prototype.moveFile.apply(this, arguments)
            }, K.prototype.removeFiles = function () {
                hteditor.Editor.prototype.removeFiles.apply(this, arguments)
            }, K.prototype.locate = function () {
                hteditor.Editor.prototype.locate.apply(this, arguments)
            }, K.prototype.newFolder = function () {
                hteditor.Editor.prototype.newFolder.apply(this, arguments)
            }, K.prototype.toggleLeft = function () {
                "normal" === this.leftSplitView.getStatus() ? this.leftSplitView.setStatus("cl") : this.leftSplitView.setStatus("normal")
            }, K.prototype.toggleRight = function () {
                "normal" === this.mainSplitView.getStatus() ? this.mainSplitView.setStatus("cr") : this.mainSplitView.setStatus("normal")
            }, K.prototype.fitSelection = function () {
                var q = this.dm.sm().getSelection();
                0 === q.length && (q = null), this.zoomToFit(q)
            }, K.prototype.$48$ = function (q) {
                hteditor.Editor.prototype.$48$.apply(this, arguments)
            }, K.prototype.$47$ = function (q) {
                hteditor.Editor.prototype.$47$.apply(this, arguments)
            }, K.prototype.handleKeydown = function (q) {
                hteditor.Editor.prototype.handleKeydown.apply(this, arguments)
            }, K.prototype.selectFileNode = function (q) {
                hteditor.Editor.prototype.selectFileNode.apply(this, arguments)
            }, K.prototype.dropLocalFileOnDir = function (q, a) {
                hteditor.Editor.prototype.dropLocalFileOnDir.apply(this, arguments)
            }, K.prototype.uploadLocalFile = function (q, a) {
                hteditor.Editor.prototype.uploadLocalFile.apply(this, arguments)
            }, K.prototype.zoomIn = function () {
                this.gv && this.gv.zoomIn(R.config.animate), this.scene && this.scene.zoomIn(R.config.animate)
            }, K.prototype.zoomOut = function () {
                this.gv && this.gv.zoomOut(R.config.animate), this.scene && this.scene.zoomOut(R.config.animate)
            }, K.prototype.zoomToFit = function (q) {
                this.scene.flyTo(q, {
                    animation: R.config.animate
                });
                var a = this.gv;
                if (!q) return void a.fitContent(R.config.animate, R.config.fitPadding);
                var k;
                q.each(function (q) {
                    k = ht.Default.unionRect(k, a.getDataUIBounds(q))
                }), ht.Default.grow(k, 20, 20), a.fitRect(k, R.config.animate)
            }, K.prototype.zoomReset = function () {
                this.gv && this.gv.zoomReset(R.config.animate), this.scene && this.scene.isResettable() && this.scene.reset()
            }, K.prototype.saveImage = function () {
                hteditor.Editor.prototype.saveImage.apply(this, arguments)
            }, K.prototype.createSceneItem = function (q, a, k, w) {
                var I = this,
                    K = function () {
                        return I.$3$() === q
                    },
                    b = hteditor.createItem(q, a, k, K);
                return b.action = function () {
                    I.$4$(q, w)
                }, b
            }, K.prototype.resetInteractionState = function () {
                this.$4$("edit")
            }, K.prototype.isRulerEnabled = function () {
                return this.rulerView.isRulerEnabled()
            }, K.prototype.setRulerEnabled = function (q) {
                this.rulerView.setRulerEnabled(q)
            }, K.prototype.toggleRulerEnabled = function () {
                this.setRulerEnabled(!this.isRulerEnabled())
            }, K.prototype.block = function () {
                if (this.editable) {
                    var q = this.dm.sm().getSelection();
                    if (q.length) {
                        this.beginTransaction();
                        var a = new ht.Block;
                        q.forEach(function (q) {
                            q.setParent(a)
                        }), this.dm.add(a), this.endTransaction()
                    }
                }
            }, K.prototype.unblock = function () {
                var q = this;
                if (this.editable) {
                    this.beginTransaction();
                    var a = [];
                    this.sm.toSelection().each(function (k) {
                        if (k instanceof ht.Block && !(k instanceof ht.RefGraph)) {
                            var w = k.getParent();
                            k.getChildren().toArray().forEach(function (q) {
                                q.setParent(w), a.push(q)
                            }), q.dm.remove(k)
                        }
                    }), a.length && this.sm.ss(a), this.endTransaction()
                }
            }, K.prototype.bringToFront = function () {}, K.prototype.sendToBack = function () {}, K.prototype.bringForward = function () {}, K.prototype.sendBackward = function () {}, K.prototype.filterProperties = function () {
                hteditor.Editor.prototype.filterProperties.apply(this, arguments)
            }, v(K, [{
                key: "url",
                get: function () {
                    return this._url
                },
                set: function (q) {
                    this._url = q, this.dataView.updateUrl()
                }
            }, {
                key: "explorer",
                get: function () {
                    var q = this.leftTopTabView.getTabModel().sm().ld();
                    return q && q.getView()
                }
            }, {
                key: "dir",
                get: function () {
                    return this.explorer ? this.explorer.tree.getSelectionModel().getLastData() : null
                }
            }, {
                key: "file",
                get: function () {
                    return this.explorer ? this.explorer.getFileListView().getSelectionModel().getLastData() : null
                }
            }, {
                key: "ld",
                get: function () {
                    return this.sm.ld()
                }
            }, {
                key: "sm",
                get: function () {
                    return this.dm.sm()
                }
            }, {
                key: "editInteractor",
                get: function () {
                    return this.gv.getEditInteractor()
                }
            }, {
                key: "pointsEditingMode",
                get: function () {
                    return this.editInteractor.pointsEditingMode
                },
                set: function (q) {
                    this.editInteractor.pointsEditingMode = q
                }
            }, {
                key: "anchorVisible",
                get: function () {
                    return this.editInteractor.getStyle("anchorVisible")
                },
                set: function (q) {
                    this.editInteractor.setStyle("anchorVisible", q)
                }
            }, {
                key: "sceneInspector",
                get: function () {
                    return this._sceneInspector || (this._sceneInspector = new uq(this)), this._sceneInspector
                }
            }, {
                key: "dataInspector",
                get: function () {
                    return this._dataInspector || (this._dataInspector = new jq(this)), this._dataInspector
                }
            }, {
                key: "cubeInspector",
                get: function () {
                    return this._cubeInspector || (this._cubeInspector = new gq(this)), this._cubeInspector
                }
            }, {
                key: "coneInspector",
                get: function () {
                    return this._coneInspector || (this._coneInspector = new Rq(this)), this._coneInspector
                }
            }, {
                key: "cylinderInspector",
                get: function () {
                    return this._cylinderInspector || (this._cylinderInspector = new lq(this)), this._cylinderInspector
                }
            }, {
                key: "roundRectInspector",
                get: function () {
                    return this._roundRectInspector || (this._roundRectInspector = new _q(this)), this._roundRectInspector
                }
            }, {
                key: "sphereInspector",
                get: function () {
                    return this._sphereInspector || (this._sphereInspector = new dq(this)), this._sphereInspector
                }
            }, {
                key: "lightInspector",
                get: function () {
                    return this._lightInspector || (this._lightInspector = new Bq(this)), this._lightInspector
                }
            }, {
                key: "torusInspector",
                get: function () {
                    return this._torusInspector || (this._torusInspector = new hq(this)), this._torusInspector
                }
            }, {
                key: "triangleInspector",
                get: function () {
                    return this._triangleInspector || (this._triangleInspector = new Jq(this)), this._triangleInspector
                }
            }, {
                key: "modelInspector",
                get: function () {
                    return this._modelInspector || (this._modelInspector = new cq(this)), this._modelInspector
                }
            }, {
                key: "boxInspector",
                get: function () {
                    return this._boxInspector || (this._boxInspector = new Yq(this)), this._boxInspector
                }
            }, {
                key: "billboardInspector",
                get: function () {
                    return this._billboardInspector || (this._billboardInspector = new Uq(this)), this._billboardInspector
                }
            }, {
                key: "wallInspector",
                get: function () {
                    return this._wallInspector || (this._wallInspector = new aa(this)), this._wallInspector
                }
            }, {
                key: "floorInspector",
                get: function () {
                    return this._floorInspector || (this._floorInspector = new ba(this)), this._floorInspector
                }
            }, {
                key: "pipelineInspector",
                get: function () {
                    return this._pipelineInspector || (this._pipelineInspector = new ua(this)), this._pipelineInspector
                }
            }, {
                key: "polylineInspector",
                get: function () {
                    return this._polylineInspector || (this._polylineInspector = new La(this)), this._polylineInspector
                }
            }, {
                key: "edgeInspector",
                get: function () {
                    return this._edgeInspector || (this._edgeInspector = new Ta(this)), this._edgeInspector
                }
            }, {
                key: "blockInspector",
                get: function () {
                    return this._blockInspector || (this._blockInspector = new Fa(this)), this._blockInspector
                }
            }, {
                key: "selection",
                get: function () {
                    var q = this,
                        a = [];
                    return this.gv && this.gv.each(function (k) {
                        q.gv.isSelected(k) && a.push(k)
                    }), a
                }
            }, {
                key: "modelView",
                get: function () {
                    return this._modelView ? this._modelView : (this._modelView = new ma(this), this.modelDialogs.obj = this._modelView, this.modelDialogs.fbx = this._modelView, this.modelDialogs.gltf = this._modelView, this._modelView)
                }
            }, {
                key: "materialView",
                get: function () {
                    return this._materialView ? this._materialView : (this._materialView = new la(this), this._materialView)
                }
            }, {
                key: "chooseDirectoryDialog",
                get: function () {
                    return !this._chooseDirectoryDialog && R.ChooseDirectoryDialog && (this._chooseDirectoryDialog = new R.ChooseDirectoryDialog(this)), this._chooseDirectoryDialog
                }
            }]), K
        }();
    return R.Editor3d = wk, R.version = "5.0.2-dev1", R.logEditorInfo = function () {
        R.config.logEditorInfo && console.log("HT 3D Editor v5.0.2-dev1 powered by HT for Web v" + ht.Default.getVersion())
    }, R.createEditor3d = function (q) {
        if (R.logEditorInfo(), window.location.host.indexOf("hightopo") >= 0) {
            var a = "https:" == document.location.protocol ? " https://" : " http://";
            document.write(unescape("%3Cdiv style='display:none'%3E%3Cspan id='cnzz_stat_icon_1000279011'%3E%3C/span%3E%3Cscript src='" + a + "s23.cnzz.com/z_stat.php%3Fid%3D1000279011%26show%3Dpic' type='text/javascript'%3E%3C/script%3E%3C/div%3E")), document.body.style.margin = "0px", document.body.style.padding = "0px"
        }
        q && (ht.Default.isString(q) ? q = {
            container: q
        } : q.tagName && q.appendChild && (q = {
            container: q
        })), q = q || {};
        var k = void 0,
            w = void 0 === q.container ? R.config.container : q.container;
        return null === w ? k = null : w ? ht.Default.isString(w) ? k = document.getElementById(w) : w.length ? (k = ht.Default.createDiv(), k.style.left = w[0] + "px", k.style.top = w[1] + "px", k.style.width = w[2] + "px", k.style.height = w[3] + "px", document.body.appendChild(k)) : k = w : k = document.body, q.$38$ = k, new wk(q)
    }, R.config.valueTypes || (R.config.valueTypes = {}), R.config.valueTypes.HighlightEnum = {
        type: "enum",
        values: [0, 1, 2],
        i18nLabels: ["editor.none", "editor.selected", "editor.hover"]
    }, R
}();