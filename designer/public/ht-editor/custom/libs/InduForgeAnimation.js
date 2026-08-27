;(function () {
  function numberValue(value, fallback) {
    value = Number(value)
    return isFinite(value) ? value : fallback
  }

  function rememberBase(data) {
    if (data.a('induforge.animation.baseOpacity') == null) {
      data.a('induforge.animation.baseOpacity', numberValue(data.s('opacity'), 1))
    }
    if (data.a('induforge.animation.baseScaleX') == null) {
      data.a('induforge.animation.baseScaleX', typeof data.getScaleX === 'function' ? numberValue(data.getScaleX(), 1) : 1)
      data.a('induforge.animation.baseScaleY', typeof data.getScaleY === 'function' ? numberValue(data.getScaleY(), 1) : 1)
    }
    if (data.a('induforge.animation.baseRotation') == null) {
      data.a('induforge.animation.baseRotation', typeof data.getRotation === 'function' ? numberValue(data.getRotation(), 0) : 0)
    }
  }

  function restoreBase(data) {
    data.s('opacity', numberValue(data.a('induforge.animation.baseOpacity'), 1))
    if (typeof data.setScaleX === 'function') data.setScaleX(numberValue(data.a('induforge.animation.baseScaleX'), 1))
    if (typeof data.setScaleY === 'function') data.setScaleY(numberValue(data.a('induforge.animation.baseScaleY'), 1))
    if (typeof data.setRotation === 'function') data.setRotation(numberValue(data.a('induforge.animation.baseRotation'), 0))
  }

  function apply(data) {
    if (!data || typeof data.a !== 'function' || typeof data.setAnimation !== 'function') return
    rememberBase(data)
    data.setAnimation(null)
    restoreBase(data)
    var preset = data.a('induforge.animation.preset') || 'off'
    var duration = Math.max(100, 1000 / Math.max(0.25, numberValue(data.a('induforge.animation.speed'), 1)))
    if (preset === 'blink') {
      var opacity = numberValue(data.a('induforge.animation.baseOpacity'), 1)
      data.setAnimation({
        start: ['fadeOut'],
        fadeOut: { property: 'opacity', accessType: 'style', from: opacity, to: opacity * 0.25, duration: duration / 2, repeat: -1, easing: 'Sine.easeInOut', next: 'fadeIn' },
        fadeIn: { property: 'opacity', accessType: 'style', from: opacity * 0.25, to: opacity, duration: duration / 2, repeat: -1, easing: 'Sine.easeInOut', next: 'fadeOut' },
      })
    } else if (preset === 'pulse') {
      var scaleX = numberValue(data.a('induforge.animation.baseScaleX'), 1)
      var scaleY = numberValue(data.a('induforge.animation.baseScaleY'), 1)
      data.setAnimation({
        start: ['scaleXUp', 'scaleYUp'],
        scaleXUp: { property: 'scaleX', from: scaleX, to: scaleX * 1.08, duration: duration / 2, repeat: -1, easing: 'Sine.easeInOut', next: 'scaleXDown' },
        scaleXDown: { property: 'scaleX', from: scaleX * 1.08, to: scaleX, duration: duration / 2, repeat: -1, easing: 'Sine.easeInOut', next: 'scaleXUp' },
        scaleYUp: { property: 'scaleY', from: scaleY, to: scaleY * 1.08, duration: duration / 2, repeat: -1, easing: 'Sine.easeInOut', next: 'scaleYDown' },
        scaleYDown: { property: 'scaleY', from: scaleY * 1.08, to: scaleY, duration: duration / 2, repeat: -1, easing: 'Sine.easeInOut', next: 'scaleYUp' },
      })
    } else if (preset === 'rotate') {
      var rotation = numberValue(data.a('induforge.animation.baseRotation'), 0)
      data.setAnimation({
        start: ['rotate'],
        rotate: { property: 'rotation', from: rotation, to: rotation + Math.PI * 2, duration: duration * 3, repeat: 0, easing: 'Linear' },
      })
    }
  }

  function applyModel(dataModel) {
    if (!dataModel) return
    if (typeof dataModel.each === 'function') dataModel.each(apply)
    if (typeof dataModel.enableAnimation === 'function') dataModel.enableAnimation()
  }

  window.InduForgeAnimation = Object.freeze({ apply: apply, applyModel: applyModel })
})()
