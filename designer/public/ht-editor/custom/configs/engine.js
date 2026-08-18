window.htconfig = {
  Default: {
    toolTipDelay: 100,
    toolTipContinual: true,
    mockTouch: true,
    autoHideScrollBar: window.navigator.platform.indexOf('Win') > -1 || !('ontouchend' in document),
    convertURL: function (url) {
      return url
    },
  },
}
