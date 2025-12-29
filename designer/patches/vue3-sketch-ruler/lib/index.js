import { ref as O, computed as U, defineComponent as se, onMounted as we, watch as $, withDirectives as j, createElementBlock as V, openBlock as D, withModifiers as ie, normalizeStyle as I, unref as P, createElementVNode as J, createCommentVNode as ve, toDisplayString as ke, vShow as _, reactive as Pe, onUnmounted as Le, normalizeClass as ge, createVNode as pe, mergeProps as ye, Fragment as Xe, renderList as Ne, createBlock as ze, useCssVars as Ye, renderSlot as be } from "vue";
const Ue = (e) => e <= 0.25 ? 40 : e <= 0.5 ? 20 : e <= 1 ? 10 : e <= 2 ? 5 : e <= 4 ? 2 : 1;
function We(e, n, t, a, l, i) {
  i ? l.moveTo(e, 0) : l.moveTo(0, e), l.save(), i ? l.translate(e + 5, a * 0.2) : l.translate(t * 0.1, e + 32), i || l.rotate(-Math.PI / 2), l.fillText(Math.round(n).toString(), 4, 7), l.restore(), i ? l.lineTo(e, a) : l.lineTo(t, e), l.stroke(), l.closePath(), l.setTransform(1, 0, 0, 1, 0, 0);
}
function ne(e, n, t, a, l, i) {
  a.fillStyle = l.fontShadowColor, a.strokeStyle = l.longfgColor, a.save(), a.translate(e, n), i || a.rotate(-Math.PI / 2), a.font = "bold 12px  Aria", a.fillText(String(t), 0, 0), a.restore();
}
const He = (e, n, t, a, l, i) => {
  const { scale: d, width: c, height: w, ratio: R, palette: E, gridRatio: k, showShadowText: T } = l, { bgColor: A, fontColor: f, shadowColor: g, longfgColor: C } = E, y = i ? l.canvasWidth : l.canvasHeight;
  e.setTransform(1, 0, 0, 1, 0, 0), e.scale(R, R), e.clearRect(0, 0, c, w), e.fillStyle = A, e.fillRect(0, 0, c, w);
  const p = Ue(d) * k * 10, S = p * d, M = Math.floor(n / p) * p, X = (M - n) / p * S, N = n + Math.ceil((i ? c : w) / d);
  if (a) {
    const m = (t - n) * d, x = a * d;
    if (e.fillStyle = g, i ? e.fillRect(m, 0, x, w) : e.fillRect(0, m, c, x), T)
      if (i) {
        ne(m, w * 0.4, Math.round(t), e, E, i);
        const b = (t + a - n) * d;
        ne(
          b,
          w * 0.4,
          Math.round(t + a),
          e,
          E,
          i
        );
      } else {
        ne(c * 0.4, m, Math.round(t), e, E, i);
        const b = (t + a - n) * d;
        ne(
          c * 0.4,
          b,
          Math.round(t + a),
          e,
          E,
          i
        );
      }
  }
  e.beginPath(), e.fillStyle = f, e.strokeStyle = C;
  for (let m = M, x = 0; m < N; m += p, x++) {
    const b = X + x * S + 0.5;
    if (m - p < y && m > y || m == y) {
      const z = X + x * S + 0.5 + (y - m) * d;
      We(z, y, c, w, e, i);
      return;
    }
    m >= 0 && m <= y && (m == 0 ? i ? (e.moveTo(b, 0), e.lineTo(b, w)) : (e.moveTo(0, b), e.lineTo(c, b)) : i ? (e.moveTo(b, 20), e.lineTo(b, w / 1.3)) : (e.moveTo(20, b), e.lineTo(c / 1.3, b)), e.save(), m == 0 ? i ? e.translate(b - 15, w * 0.01) : e.translate(c * 0.3, b - 5) : i ? e.translate(b - 12, w * 0.05) : e.translate(c * 0.05, b + 12), i || e.rotate(-Math.PI / 2), y - m > p / 2 && (!T || a == 0 || Math.abs(m - t) > p / 2 && Math.abs(m - (t + a)) > p / 2) && e.fillText(m.toString(), 4, 9), e.restore());
  }
  e.stroke(), e.closePath();
};
function me(e, n = 100) {
  let t = null;
  const a = function(...l) {
    t !== null && clearTimeout(t), t = setTimeout(() => {
      e(...l);
    }, n);
  };
  return a.cancel = function() {
    t !== null && (clearTimeout(t), t = null);
  }, a;
}
function Re(e, n) {
  const t = O(0), a = O(0), l = O(!1), i = U(() => ({
    backgroundColor: e.palette.hoverBg,
    color: e.palette.hoverColor,
    [n ? "top" : "left"]: "-8px",
    [n ? "left" : "top"]: `${t.value + 10}px`
  })), d = ({ offsetX: g, offsetY: C }) => {
    t.value = n ? g : C;
  }, c = (g, C) => new Promise((y) => {
    if (e.lockLine) return;
    const L = n ? g.clientY : g.clientX;
    d(g);
    const p = C || a.value;
    let S = p;
    const M = (N) => {
      let b = ((n ? N.clientY : N.clientX) - L) / e.scale + p, z = b;
      const H = (n ? e.snapsObj.h : e.snapsObj.v).slice().sort((K, Z) => Math.abs(z - K) - Math.abs(z - Z));
      H.length && Math.abs(H[0] - b) < e.snapThreshold / e.scale && (z = H[0], b = z), S = Math.round(b), a.value = S;
    }, X = () => {
      document.removeEventListener("mousemove", M), w(S, e.index), y();
    };
    document.addEventListener("mousemove", M), document.addEventListener("mouseup", X, { once: !0 });
  }), w = (g, C) => {
    const y = n ? e.lines?.h : e.lines?.v, L = R(g);
    if (y)
      if (L)
        if (typeof C == "number")
          y.splice(C, 1);
        else
          return;
      else
        typeof C != "number" ? y.push(g) : y[C] = g;
  }, R = (g) => {
    const C = n ? e.canvasHeight : e.canvasWidth;
    return g < 0 || g > C;
  }, E = U(() => R(a.value) ? "放开删除" : `${n ? "Y" : "X"}：${a.value * e.rate}`), k = me(() => {
    l.value = !1;
  }, 200), T = me(() => {
    l.value = !0;
  }, 200);
  return {
    showLabel: l,
    startValue: a,
    actionStyle: i,
    labelContent: E,
    handleMouseDown: c,
    handleMouseenter: (g) => {
      e.lockLine || (d(g), T(), k.cancel());
    },
    handleMouseLeave: () => {
      k();
    }
  };
}
const De = {
  key: 0,
  class: "value"
}, Ie = /* @__PURE__ */ se({
  __name: "ruler-line",
  props: {
    scale: {},
    palette: {},
    index: {},
    start: {},
    vertical: { type: Boolean },
    value: {},
    canvasWidth: {},
    canvasHeight: {},
    lines: {},
    isShowReferLine: { type: Boolean },
    rate: {},
    snapThreshold: {},
    snapsObj: {},
    lockLine: { type: Boolean }
  },
  setup(e) {
    const n = e, t = O(!1), {
      actionStyle: a,
      handleMouseDown: l,
      labelContent: i,
      startValue: d,
      showLabel: c,
      handleMouseenter: w,
      handleMouseLeave: R
    } = Re(n, n.vertical), E = U(() => d.value >= n.start), k = U(() => {
      const { lineType: A, lockLineColor: f, lineColor: g } = n.palette, C = n.lockLine ? f : g ?? "black", y = n.lockLine || t.value ? "none" : "auto", L = n.isShowReferLine && !n.lockLine ? n.vertical ? "ns-resize" : "ew-resize" : "default", p = n.vertical ? "borderTop" : "borderLeft", S = (d.value - n.start) * n.scale;
      return {
        [p]: `1px ${A} ${C}`,
        pointerEvents: y,
        cursor: L,
        [n.vertical ? "top" : "left"]: `${S}px`
      };
    });
    we(() => {
      d.value = n.value ?? 0;
    });
    const T = me(() => {
      t.value = !1;
    }, 1e3);
    return $([() => n.scale], () => {
      t.value = !0, T();
    }), (A, f) => j((D(), V("div", {
      class: "line",
      style: I(k.value),
      onMouseenter: f[0] || (f[0] = ie(
        //@ts-ignore
        (...g) => P(w) && P(w)(...g),
        ["stop"]
      )),
      onMouseleave: f[1] || (f[1] = ie(
        //@ts-ignore
        (...g) => P(R) && P(R)(...g),
        ["stop"]
      )),
      onMousedown: f[2] || (f[2] = ie(
        //@ts-ignore
        (...g) => P(l) && P(l)(...g),
        ["stop"]
      ))
    }, [
      J("div", {
        class: "action",
        style: I(P(a))
      }, [
        P(c) ? (D(), V("span", De, ke(P(i)), 1)) : ve("", !0)
      ], 4)
    ], 36)), [
      [_, E.value]
    ]);
  }
}), Ve = /* @__PURE__ */ se({
  __name: "index",
  props: {
    scale: {},
    palette: {},
    vertical: { type: Boolean },
    showShadowText: { type: Boolean },
    start: {},
    width: {},
    height: {},
    selectStart: {},
    selectLength: {},
    canvasWidth: {},
    canvasHeight: {},
    rate: {},
    gridRatio: {}
  },
  emits: ["handleDragStart"],
  setup(e, { emit: n }) {
    const t = e, a = n, l = Pe({
      isDragging: !1,
      canvasContext: null
    });
    let i = window.devicePixelRatio;
    const d = O(null);
    we(() => {
      window.addEventListener("resize", c), w(), E(i), k(i);
    });
    const c = () => {
      i = window.devicePixelRatio, E(i), k(i);
    }, w = () => {
      l.canvasContext = d.value?.getContext("2d") || null;
    }, R = U(() => ({
      width: t.width + "px",
      height: t.height + "px",
      cursor: t.vertical ? "ew-resize" : "ns-resize",
      [t.vertical ? "borderRight" : "borderBottom"]: `1px solid ${t.palette.borderColor || "#eeeeef"} `
    }));
    Le(() => {
      window.removeEventListener("resize", c);
    });
    const E = (A) => {
      if (d.value) {
        d.value.width = Math.round(t.width * A), d.value.height = Math.round(t.height * A);
        const f = l.canvasContext;
        f && (f.font = `11px -apple-system,
                "Helvetica Neue", ".SFNSText-Regular",
                "SF UI Text", Arial, "PingFang SC", "Hiragino Sans GB",
                "Microsoft YaHei", "WenQuanYi Zen Hei", sans-serif`, f.lineWidth = 1, f.textBaseline = "middle");
      }
    }, k = (A) => {
      const f = {
        scale: t.scale,
        width: t.width,
        height: t.height,
        palette: t.palette,
        canvasWidth: t.canvasWidth,
        canvasHeight: t.canvasHeight,
        ratio: A,
        rate: t.rate,
        gridRatio: t.gridRatio,
        showShadowText: t.showShadowText
      };
      f.scale = t.scale / t.rate, f.canvasWidth = t.canvasWidth * t.rate, f.canvasHeight = t.canvasHeight * t.rate, l.canvasContext && He(
        l.canvasContext,
        t.start * t.rate,
        t.selectStart,
        t.selectLength,
        f,
        !t.vertical
      );
    };
    $(
      [
        () => t.width,
        () => t.height,
        () => t.start,
        () => t.palette,
        () => t.selectStart,
        () => t.selectLength,
        () => t.scale,
        () => t.canvasWidth,
        () => t.canvasHeight,
        () => t.rate,
        () => t.gridRatio,
        () => t.showShadowText
      ],
      () => {
        k(i);
      }
    ), $([() => t.width, () => t.height], () => {
      E(i);
    });
    const T = (A) => {
      a("handleDragStart", A);
    };
    return (A, f) => (D(), V("canvas", {
      ref_key: "canvas",
      ref: d,
      class: "ruler",
      style: I(R.value),
      onMousedown: ie(T, ["stop"])
    }, null, 36));
  }
}), Ke = { class: "lines" }, Fe = {
  key: 0,
  class: "value"
}, Ae = /* @__PURE__ */ se({
  __name: "ruler-wrapper",
  props: {
    scale: {},
    thick: {},
    canvasWidth: {},
    canvasHeight: {},
    palette: {},
    vertical: { type: Boolean },
    width: {},
    height: {},
    start: {},
    startOther: {},
    lines: {},
    selectStart: {},
    selectLength: {},
    isShowReferLine: { type: Boolean },
    rate: {},
    snapThreshold: {},
    snapsObj: {},
    gridRatio: {},
    lockLine: { type: Boolean },
    showShadowText: { type: Boolean }
  },
  emits: ["changeLineState"],
  setup(e, { emit: n }) {
    const t = e, {
      showLabel: a,
      actionStyle: l,
      handleMouseenter: i,
      handleMouseLeave: d,
      handleMouseDown: c,
      labelContent: w,
      startValue: R
    } = Re(t, !t.vertical), E = O(!1), k = O(!1), T = U(() => t.vertical ? "v-container" : "h-container"), A = n, f = U(() => t.vertical ? t.lines?.h : t.lines?.v), g = U(() => {
      const y = t.palette.lineType;
      let L = t.vertical ? "left" : "top", p = t.vertical ? "top" : "left", S = t.vertical ? "borderLeft" : "borderBottom";
      const M = (R.value - t.startOther) * t.scale + t.thick;
      return {
        [L]: M + "px",
        [p]: -t.thick + "px",
        cursor: t.vertical ? "ew-resize" : "ns-resize",
        [S]: `1px ${y} ${t.palette.lineColor}`
      };
    }), C = async (y) => {
      const { offsetX: L, offsetY: p } = y, { scale: S, vertical: M, thick: X, startOther: N } = t;
      if (!E.value) {
        k.value = !0, A("changeLineState", !1);
        let m = Math.round(
          N - X / S + (M ? L : p) / S
        );
        R.value = m, await c(y, m), k.value = !1;
      }
    };
    return $([() => t.lockLine], () => {
      E.value = t.lockLine;
    }), (y, L) => (D(), V("div", {
      class: ge(T.value)
    }, [
      pe(Ve, ye(y.$props, { onHandleDragStart: C }), null, 16),
      j(J("div", Ke, [
        (D(!0), V(Xe, null, Ne(f.value, (p, S) => (D(), ze(Ie, ye({
          key: p + S,
          index: S,
          value: p >> 0
        }, { ref_for: !0 }, y.$props), null, 16, ["index", "value"]))), 128))
      ], 512), [
        [_, e.isShowReferLine]
      ]),
      e.isShowReferLine ? j((D(), V("div", {
        key: 0,
        class: "indicator",
        style: I(g.value),
        onMouseenter: L[0] || (L[0] = //@ts-ignore
        (...p) => P(i) && P(i)(...p)),
        onMouseleave: L[1] || (L[1] = //@ts-ignore
        (...p) => P(d) && P(d)(...p))
      }, [
        J("div", {
          class: "action",
          style: I(P(l))
        }, [
          P(a) ? (D(), V("span", Fe, ke(P(w)), 1)) : ve("", !0)
        ], 4)
      ], 36)), [
        [_, k.value]
      ]) : ve("", !0)
    ], 2));
  }
}), Qe = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABUAAAAVCAYAAACpF6WWAAAAAXNSR0IArs4c6QAAAopJREFUOE/FlE9IVEEcx7+/N9ouds1Mu0QUSFZYdIgoUqQoKPBQHsKozpXE7jbTO/U8xLJvn6usBHWQ6hBFXupSkQeVbh0KJEPp0sH+eLGTsKs77xcj78m0ax0E8cHjzZv5zef3/c33xxA24KENYGJzoEEQbNNaN4Zh2OQ4znwYhr9c1/39vwrXVDo0NNS0tLR0GYB5D64BmAMwzMyvlFKz1es10Hw+f4mZ7wHYBeA9gNdENFepVOaEEM3M3OI4Thczn41gt6WUgQ3+C+r7/h0AWQD3mXnYqPA8L9nQ0HCemduIaFpKOWoAhUJhT6VSuQXgOjP3K6W8GLwKzeVyp4jonR0QBEErM48w8zFLyayUsjX+z+VyHhHdZebTSqkxM78CHRgYOKS1/ghgVErZY214RkQ7ADyRUj72ff8qgCtmXUrZGcf5vv8CwEUhxOF0Ov1pBRpla5dSdseBhUJhpznH6tIsZb1KqacW+BGArUaUXX63UuplHJTNZjuEEONSyhozfd/n6mQ1RkXZL2itz7mu+80EDA4ONi8vL/8AcM2UbikyR2BU9cSmmTU70YqKIAj2hWFo2uenlHK/BRg3Y2aeNO5GyU8S0ZbFxcUuz/NKEXAGQKPjOCcymcyX1dIi8DSAiWQyeaavr68cbSgCuBknYubnQoj+TCYzUywWE6VS6S2ADsdx2gxw1X3L7SNENMbMnwE8qK+vf5NKpRaMaeVyeW8ikfiaSqW+R7BuZr5BRMe11p2u607U9Gk8kc/ntzPzQwCmExYATDLzVBiGE0KIowAOADDf3QA+aK2VDaxRajto3K+rq+tl5nYAzQBamHmeiOYBTGmtR6ph/1Rqg9c73pz7dD1qN0TpHyNQRCUDJXrAAAAAAElFTkSuQmCC", $e = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABUAAAAVCAYAAACpF6WWAAAAAXNSR0IArs4c6QAAAjtJREFUOE/NlD9oFEEUxn9v9ghC0hpRUogIAUmniGAT/5Q2FrETPSNRJILg7RwimI0ox85eQFCEBGIUO1PYpFM0jSConQoBEZGgGPsgl+w+2eM2bC57SopAFqYZ3v7m+977ZoQt+GQLmGxPaBiGgYiMWWvXBHZUGoZhH3BERPYC+4F+4Keq/urt7b1RLpf/ZEBVHa9Wq0HWyg3QKIoGVPU8cA7wgK/pUtXPQJ8xZk+pVBpuNBqXUoUiEvi+P56fzTpo6+SbwHNg1lo7WzTITGEKXFlZeeB53tVCpa3CK8AFa+1cBgvD8LKIXAQOJkkyICJDeYVBEJS6u7s/qeoLa+1o+l9TqXNuBLhmjDlbqVQ+5ICjInIfOBPH8W9jzGCR5YmJiRNxHM+papgqzqDvgSlr7VTernPuO3An3c9bBt74vv+yrdaKyLDv+/1Sq9UGPc97nY9EVuycU2AQOA7cAm4Dr4D5TvVxHB9rKo2iaEFVp621Ln96FEUngaOqGmSxabVqsh3a2h+x1h5qQjNrae/yE4+iaCwDJkky73neTuBZe129Xk+H+BS4l7ZqLVIZ2BhzuFKpvMsDVXXWGPMxFdAedOfcKeCRqj7MYrUhpz09PfXl5eXrectFWXXODQHpOq2qd/95o/JXr6ura3J1dXU6SZIfwKKIHAD2tVYMPBGRx77vN10UXtO85fTkmZmZHUtLSzUR2QXsBhaAL6r6DXhbrVYXi1yss59GqOgub/bN3Z7v6X/tb9Zmp/q/kN8s+lJb8oEAAAAASUVORK5CYII=";
function Te(e, n) {
  let t = e.length;
  for (; t--; )
    if (e[t].pointerId === n.pointerId) return t;
  return -1;
}
function Se(e, n) {
  const t = Te(e, n);
  t > -1 && e.splice(t, 1), e.push(n);
}
function qe(e, n) {
  const t = Te(e, n);
  t > -1 && e.splice(t, 1);
}
function Ce(e) {
  const n = e.slice(0);
  let t = n.pop(), a;
  for (; a = n.pop(); )
    t = {
      clientX: (a.clientX - t.clientX) / 2 + t.clientX,
      clientY: (a.clientY - t.clientY) / 2 + t.clientY
    };
  return t;
}
var Be = {
  down: "pointerdown",
  move: "pointermove",
  up: "pointerup pointerleave pointercancel"
};
function he(e, n, t, a) {
  Be[e].split(" ").forEach((l) => {
    n.addEventListener(l, t, a);
  });
}
function fe(e, n, t) {
  Be[e].split(" ").forEach((a) => {
    n.removeEventListener(a, t);
  });
}
var Ee = ["webkit", "moz", "ms"], ae = {};
function Ze() {
  return document.createElement("div").style;
}
function Me(e) {
  if (ae[e]) return ae[e];
  const n = Ze();
  if (e in n) return ae[e] = e;
  const t = e[0].toUpperCase() + e.slice(1);
  for (let a = Ee.length - 1; a >= 0; a--) {
    const l = `${Ee[a]}${t}`;
    if (l in n) return ae[e] = l;
  }
  return e;
}
function G(e, n, t) {
  e.style[Me(n)] = t;
}
function Ge(e, n) {
  const t = Me("transform");
  G(e, "transition", `${t} ${n.duration}ms ${n.easing}`);
}
function je(e, { x: n, y: t, scale: a }, l) {
  G(e, "transform", `scale(${a}) translate(${n}px, ${t}px)`);
}
function oe(e, n, t = window.getComputedStyle(e)) {
  const a = n === "border" ? "Width" : "";
  return {
    left: parseFloat(t[`${n}Left${a}`]) || 0,
    right: parseFloat(t[`${n}Right${a}`]) || 0,
    top: parseFloat(t[`${n}Top${a}`]) || 0,
    bottom: parseFloat(t[`${n}Bottom${a}`]) || 0
  };
}
function Je(e) {
  const n = e.parentNode, t = e.getBoundingClientRect(), a = n.getBoundingClientRect();
  return {
    elem: {
      width: t.width,
      height: t.height,
      left: t.left,
      top: t.top
    },
    parent: {
      width: a.width,
      height: a.height,
      left: a.left,
      top: a.top
    }
  };
}
function re(e) {
  const n = e.parentNode, t = window.getComputedStyle(e), a = window.getComputedStyle(n), l = e.getBoundingClientRect(), i = n.getBoundingClientRect();
  return {
    elem: {
      width: l.width,
      height: l.height,
      left: l.left,
      top: l.top,
      margin: oe(e, "margin", t),
      border: oe(e, "border", t)
    },
    parent: {
      width: i.width,
      height: i.height,
      left: i.left,
      top: i.top,
      padding: oe(n, "padding", a),
      border: oe(n, "border", a)
    }
  };
}
function _e(e) {
  let n = e;
  for (; n && n.parentNode; ) {
    if (n.parentNode === document) return !0;
    n = n.parentNode instanceof ShadowRoot ? n.parentNode.host : n.parentNode;
  }
  return !1;
}
function et(e) {
  const n = {};
  for (const t in e)
    Object.prototype.hasOwnProperty.call(e, t) && (n[t] = e[t]);
  return n;
}
var xe = {
  animate: !1,
  canvas: !1,
  cursor: "move",
  disablePan: !1,
  disableZoom: !1,
  duration: 200,
  easing: "ease-in-out",
  handleStartEvent: (e) => {
    e.preventDefault(), e.stopPropagation();
  },
  maxScale: 4,
  minScale: 0.125,
  overflow: "hidden",
  setTransform: je,
  startX: 0,
  startY: 0,
  startScale: 1,
  step: 0.3,
  touchAction: "none",
  origin: "50% 50%"
};
function Oe(e, n) {
  if (!e || e.nodeType !== 1) throw new Error("Panzoom requires an element");
  if (!_e(e)) throw new Error("Panzoom should be called on attached elements");
  n = { ...xe, ...n };
  const t = e.parentNode;
  t.style.overflow = n.overflow, t.style.userSelect = "none", t.style.touchAction = n.touchAction, (n.canvas ? t : e).style.cursor = n.cursor, e.style.userSelect = "none", e.style.touchAction = n.touchAction, G(e, "transformOrigin", typeof n.origin == "string" ? n.origin : "50% 50%");
  function a() {
    t.style.overflow = "", t.style.userSelect = "", t.style.touchAction = "", t.style.cursor = "", e.style.cursor = "", e.style.userSelect = "", e.style.touchAction = "", G(e, "transformOrigin", "");
  }
  function l(r = {}) {
    for (const v in r) Object.prototype.hasOwnProperty.call(r, v) && (n[v] = r[v]);
    (r.hasOwnProperty("cursor") || r.hasOwnProperty("canvas")) && (t.style.cursor = e.style.cursor = "", (n.canvas ? t : e).style.cursor = n.cursor), r.hasOwnProperty("overflow") && (t.style.overflow = r.overflow), r.hasOwnProperty("touchAction") && (t.style.touchAction = r.touchAction, e.style.touchAction = r.touchAction);
  }
  let i = 0, d = 0, c = 1, w = !1;
  f(n.startScale, { animate: !1, force: !0 }), setTimeout(() => {
    A(n.startX, n.startY, { animate: !1, force: !0 });
  });
  function R(r, v) {
    if (v.silent) return;
    const h = new CustomEvent("panzoomchange", { detail: r });
    e.dispatchEvent(h);
  }
  function E(r, v) {
    const h = { x: i, y: d, scale: c };
    typeof r.animate == "boolean" && (r.animate ? Ge(e, r) : G(e, "transition", "none")), r.setTransform(e, h, r);
    const B = () => {
      const o = r.contain && r.contain !== "none" ? re(e) : Je(e);
      R({ x: i, y: d, scale: c, dimsOut: o, originalEvent: v }, r);
    };
    return r.animate ? setTimeout(B, r.duration + 50) : requestAnimationFrame(B), h;
  }
  function k(r, v, h, B) {
    const o = { ...n, ...B }, u = { x: i, y: d, opts: o };
    if (!o.force && o.disablePan === !0) return u;
    if (r = parseFloat(r), v = parseFloat(v), u.x = r, u.y = v, o.contain && o.contain !== "none") {
      const s = re(e), W = s.elem.width / c, Y = s.elem.height / c, ee = W * h, te = Y * h, F = (ee - W) / 2, Q = (te - Y) / 2;
      if (o.contain === "inside") {
        const le = (-s.elem.margin.left - s.parent.padding.left + F) / h, ce = (s.parent.width - ee - s.parent.padding.left - s.elem.margin.left - s.parent.border.left - s.parent.border.right + F) / h;
        u.x = Math.max(Math.min(u.x, ce), le);
        const ue = (-s.elem.margin.top - s.parent.padding.top + Q) / h, de = (s.parent.height - te - s.parent.padding.top - s.elem.margin.top - s.parent.border.top - s.parent.border.bottom + Q) / h;
        u.y = Math.max(Math.min(u.y, de), ue);
      } else if (o.contain === "outside") {
        const le = (-(ee - s.parent.width) - s.parent.padding.left - s.parent.border.left - s.parent.border.right + F) / h, ce = (F - s.parent.padding.left) / h;
        u.x = Math.max(Math.min(u.x, ce), le);
        const ue = (-(te - s.parent.height) - s.parent.padding.top - s.parent.border.top - s.parent.border.bottom + Q) / h, de = (Q - s.parent.padding.top) / h;
        u.y = Math.max(Math.min(u.y, de), ue);
      }
    }
    return u;
  }
  function T(r, v) {
    const h = { ...n, ...v }, B = { scale: c, opts: h };
    if (!h.force && h.disableZoom) return B;
    let o = n.minScale, u = n.maxScale;
    if (h.contain && h.contain !== "none") {
      const s = re(e), W = s.elem.width / c, Y = s.elem.height / c;
      if (W > 1 && Y > 1) {
        const ee = s.parent.width - s.parent.border.left - s.parent.border.right, te = s.parent.height - s.parent.border.top - s.parent.border.bottom, F = ee / W, Q = te / Y;
        n.contain === "inside" ? u = Math.min(u, F, Q) : n.contain === "outside" && (o = Math.max(o, F, Q));
      }
    }
    return B.scale = Math.min(Math.max(r, o), u), B;
  }
  function A(r, v, h, B) {
    const o = k(r, v, c, h);
    return i !== o.x || d !== o.y ? (i = o.x, d = o.y, E(o.opts, B)) : { x: i, y: d, scale: c };
  }
  function f(r, v, h) {
    const B = T(r, v), o = B.opts;
    if (!o.force && o.disableZoom) return { x: i, y: d, scale: c };
    r = B.scale;
    let u = i, s = d;
    if (o.focal) {
      const Y = o.focal;
      u = (Y.x / r - Y.x / c + i * r) / r, s = (Y.y / r - Y.y / c + d * r) / r;
    }
    const W = k(u, s, r, { relative: !1, force: !0 });
    return i = W.x, d = W.y, c = r, E(o, h);
  }
  function g(r, v) {
    const h = { ...n, animate: !0, ...v };
    return f(c * Math.exp((r ? 1 : -1) * h.step), h);
  }
  function C(r) {
    return g(!0, r);
  }
  function y(r) {
    return g(!1, r);
  }
  function L(r, v, h, B) {
    const o = re(e), u = {
      width: o.parent.width - o.parent.padding.left - o.parent.padding.right - o.parent.border.left - o.parent.border.right,
      height: o.parent.height - o.parent.padding.top - o.parent.padding.bottom - o.parent.border.top - o.parent.border.bottom
    };
    let s = v.clientX - o.parent.left - o.parent.padding.left - o.parent.border.left - o.elem.margin.left, W = v.clientY - o.parent.top - o.parent.padding.top - o.parent.border.top - o.elem.margin.top;
    n.origin !== "0 0" && (s -= o.elem.width / c / 2, W -= o.elem.height / c / 2);
    const Y = {
      x: s / u.width * (u.width * r),
      y: W / u.height * (u.height * r)
    };
    return f(r, { ...h, animate: !1, focal: Y }, B);
  }
  function p(r, v) {
    r.preventDefault();
    const h = { ...n, ...v, animate: !1 }, o = (r.deltaY === 0 && r.deltaX ? r.deltaX : r.deltaY) < 0 ? 1 : -1, u = T(c * Math.exp(o * h.step / 3), h).scale;
    return L(u, r, h, r);
  }
  function S(r) {
    const v = { ...n, animate: !0, force: !0, ...r };
    c = T(v.startScale, v).scale;
    const h = k(v.startX, v.startY, c, v);
    return i = h.x, d = h.y, E(v);
  }
  let M, X, N, m;
  const x = [];
  function b(r) {
    Se(x, r), w = !0, n.handleStartEvent(r), M = i, X = d;
    const v = Ce(x);
    N = v.clientX, m = v.clientY;
  }
  function z(r) {
    if (!w || M === void 0 || X === void 0 || N === void 0 || m === void 0) return;
    Se(x, r);
    const v = Ce(x), h = c;
    A(M + (v.clientX - N) / h, X + (v.clientY - m) / h, { animate: !1 }, r);
  }
  function q(r) {
    qe(x, r), w && (w = !1, M = X = N = m = void 0);
  }
  let H = !1;
  function K() {
    H || (H = !0, he("down", n.canvas ? t : e, b), he("move", document, z, { passive: !0 }), he("up", document, q, { passive: !0 }));
  }
  function Z() {
    H = !1, fe("down", n.canvas ? t : e, b), fe("move", document, z), fe("up", document, q);
  }
  return n.noBind || K(), {
    bind: K,
    destroy: Z,
    getPan: () => ({ x: i, y: d }),
    getScale: () => c,
    getOptions: () => et(n),
    handleDown: b,
    handleMove: z,
    handleUp: q,
    pan: A,
    reset: S,
    resetStyle: a,
    setOptions: l,
    setStyle: (r, v) => G(e, r, v),
    zoom: f,
    zoomIn: C,
    zoomOut: y,
    zoomToPoint: L,
    zoomWithWheel: p
  };
}
Oe.defaultOptions = xe;
var tt = Oe;
const nt = { class: "sketch-ruler" }, ot = /* @__PURE__ */ se({
  __name: "index",
  props: {
    showRuler: { type: Boolean, default: !0 },
    eyeIcon: {},
    closeEyeIcon: {},
    scale: { default: 1 },
    rate: { default: 1 },
    thick: { default: 16 },
    palette: {},
    width: { default: 1400 },
    height: { default: 800 },
    paddingRatio: { default: 0.2 },
    autoCenter: { type: Boolean, default: !0 },
    shadow: { default: () => ({
      x: 0,
      y: 0,
      width: 0,
      height: 0
    }) },
    lines: { default: () => ({
      h: [],
      v: []
    }) },
    isShowReferLine: { type: Boolean, default: !0 },
    canvasWidth: { default: 700 },
    canvasHeight: { default: 700 },
    snapsObj: { default: () => ({
      h: [],
      v: []
    }) },
    snapThreshold: { default: 5 },
    gridRatio: { default: 1 },
    lockLine: { type: Boolean, default: !1 },
    selfHandle: { type: Boolean, default: !1 },
    showShadowText: { type: Boolean, default: !0 },
    panzoomOption: {}
  },
  emits: ["onCornerClick", "update:scale", "zoomchange", "update:lockLine"],
  setup(e, { expose: n, emit: t }) {
    Ye((o) => ({
      v58ad60a2: B.value
    }));
    const a = e, l = t, i = O(null), d = O(null), c = O(null), w = O(0), R = O(0);
    let E = 0, k = 0;
    const T = O(1), A = O(a.isShowReferLine), f = O(null), g = O("defaultCursor"), C = U(() => ({
      bgColor: "#f6f7f9",
      // ruler bg color
      longfgColor: "#BABBBC",
      // ruler longer mark color
      fontColor: "#7D8694",
      // ruler font color
      fontShadowColor: "#106ebe",
      shadowColor: "#e9f7fe",
      // ruler shadow color
      lineColor: "#51d6a9",
      lineType: "solid",
      lockLineColor: "#d4d7dc",
      hoverBg: "#000",
      hoverColor: "#fff",
      borderColor: "#eeeeef",
      ...a.palette
    })), y = U(() => ({
      backgroundImage: A.value ? `url(${a.eyeIcon ?? Qe})` : `url(${a.closeEyeIcon ?? $e})`,
      width: a.thick + "px",
      height: a.thick + "px",
      borderRight: `1px solid ${C.value.borderColor}`,
      borderBottom: `1px solid ${C.value.borderColor}`
    })), L = U(() => ({
      background: C.value.bgColor,
      width: p.value + "px",
      height: S.value + "px"
    })), p = U(() => a.width - a.thick), S = U(() => a.height - a.thick), M = (o) => {
      (o.ctrlKey || o.metaKey) && (o.preventDefault(), f.value?.zoomWithWheel(o));
    }, X = (o) => {
      const u = document.activeElement;
      u?.closest(".monaco-editor") || u?.tagName === "INPUT" || u?.tagName === "TEXTAREA" || u?.getAttribute("contenteditable") === "true" || o.key === " " && (g.value = "grabCursor", f.value?.bind(), o.preventDefault());
    }, N = (o) => {
      const u = document.activeElement;
      u?.closest(".monaco-editor") || u?.tagName === "INPUT" || u?.tagName === "TEXTAREA" || u?.getAttribute("contenteditable") === "true" || o.key === " " && (f.value?.destroy(), g.value = "defaultCursor");
    };
    function m(o) {
      o.preventDefault(), o.stopPropagation(), f.value?.bind();
    }
    we(() => {
      if (z(), !a.selfHandle && i.value) {
        const o = i.value.parentElement;
        if (!o) return;
        d.value = o, o.addEventListener("wheel", M, { passive: !1 }), document.addEventListener("keydown", X), document.addEventListener("keyup", N), o.addEventListener("touchstart", m, { passive: !1 });
      }
    }), Le(() => {
      d.value && (d.value.removeEventListener("wheel", M), d.value.removeEventListener("touchstart", m)), document.removeEventListener("keydown", X), document.removeEventListener("keyup", N), c.value && c.value.removeEventListener("panzoomchange", b), f.value?.destroy();
    });
    const x = (o) => ({
      noBind: !0,
      startScale: o,
      // cursor: 'default',
      startX: E,
      startY: k,
      smoothScroll: !0,
      canvas: !0,
      ...a.panzoomOption
    }), b = (o) => {
      const { scale: u, dimsOut: s } = o.detail;
      if (s) {
        l("update:scale", u), T.value = u;
        const W = (s.parent.left - s.elem.left) / u, Y = (s.parent.top - s.elem.top) / u;
        w.value = W, l("zoomchange", o.detail), R.value = Y;
      }
    }, z = () => {
      if (c.value && c.value.removeEventListener("panzoomchange", b), f.value?.destroy(), i.value = document.querySelector(".canvasedit"), i.value) {
        let o = a.scale;
        a.autoCenter && (o = q()), f.value = tt(i.value, x(o)), i.value.addEventListener("panzoomchange", b), c.value = i.value;
      }
    }, q = () => {
      const o = p.value * (1 - a.paddingRatio) / a.canvasWidth, u = S.value * (1 - a.paddingRatio) / a.canvasHeight, s = Math.min(o, u);
      return E = p.value / 2 - a.canvasWidth / 2, s < 1 ? k = (a.canvasHeight * s / 2 - a.canvasHeight / 2) / s - (a.canvasHeight * s - S.value) / s / 2 : s > 1 ? k = (a.canvasHeight * s - a.canvasHeight) / 2 / s + (S.value - a.canvasHeight * s) / s / 2 : k = 0, s;
    }, H = () => f.value?.reset(), K = () => f.value?.zoomIn(), Z = () => f.value?.zoomOut(), r = () => {
      f.value?.setOptions(x(a.scale));
    }, v = () => {
      A.value = !A.value, l("onCornerClick", A.value);
    }, h = (o) => {
      l("update:lockLine", o);
    }, B = U(() => a.thick + "px");
    return $([() => a.isShowReferLine], () => {
      A.value = a.isShowReferLine;
    }), $(
      [() => a.canvasWidth, () => a.canvasHeight, () => a.width, () => a.height],
      () => {
        z();
      }
    ), $(
      () => a.panzoomOption,
      () => {
        r();
      },
      { deep: !0 }
    ), n({
      initPanzoom: z,
      panzoomInstance: f,
      reset: H,
      zoomIn: K,
      zoomOut: Z,
      cursorClass: g
    }), (o, u) => (D(), V("div", nt, [
      be(o.$slots, "btn", {
        reset: H,
        zoomIn: K,
        zoomOut: Z
      }),
      J("div", {
        class: ge(["canvasedit-parent", g.value]),
        style: I(L.value)
      }, [
        J("div", {
          class: ge(["canvasedit", g.value])
        }, [
          be(o.$slots, "default")
        ], 2)
      ], 6),
      j(pe(Ae, {
        style: I({ marginLeft: e.thick + "px", width: p.value + "px" }),
        vertical: !1,
        width: e.width,
        height: e.thick,
        "is-show-refer-line": A.value,
        thick: e.thick,
        start: w.value,
        "start-other": R.value,
        lines: e.lines,
        "select-start": e.shadow.x,
        "snap-threshold": e.snapThreshold,
        "snaps-obj": e.snapsObj,
        "select-length": e.shadow.width,
        scale: T.value,
        palette: C.value,
        "canvas-width": e.canvasWidth,
        "show-shadow-text": e.showShadowText,
        "canvas-height": e.canvasHeight,
        rate: e.rate,
        "grid-ratio": e.gridRatio,
        "lock-line": e.lockLine,
        onChangeLineState: h
      }, null, 8, ["style", "width", "height", "is-show-refer-line", "thick", "start", "start-other", "lines", "select-start", "snap-threshold", "snaps-obj", "select-length", "scale", "palette", "canvas-width", "show-shadow-text", "canvas-height", "rate", "grid-ratio", "lock-line"]), [
        [_, e.showRuler]
      ]),
      j(pe(Ae, {
        style: I({ marginTop: e.thick + "px", top: 0, height: S.value + "px" }),
        vertical: !0,
        width: e.thick,
        height: e.height,
        "is-show-refer-line": A.value,
        thick: e.thick,
        start: R.value,
        "start-other": w.value,
        lines: e.lines,
        "select-start": e.shadow.y,
        "select-length": e.shadow.height,
        "snap-threshold": e.snapThreshold,
        "snaps-obj": e.snapsObj,
        scale: T.value,
        palette: C.value,
        "canvas-width": e.canvasWidth,
        "canvas-height": e.canvasHeight,
        "show-shadow-text": e.showShadowText,
        rate: e.rate,
        "grid-ratio": e.gridRatio,
        "lock-line": e.lockLine,
        onChangeLineState: h
      }, null, 8, ["style", "width", "height", "is-show-refer-line", "thick", "start", "start-other", "lines", "select-start", "select-length", "snap-threshold", "snaps-obj", "scale", "palette", "canvas-width", "canvas-height", "show-shadow-text", "rate", "grid-ratio", "lock-line"]), [
        [_, e.showRuler]
      ]),
      j(J("a", {
        class: "corner",
        style: I(y.value),
        onClick: v
      }, null, 4), [
        [_, e.showRuler]
      ])
    ]));
  }
});
export {
  ot as default
};
