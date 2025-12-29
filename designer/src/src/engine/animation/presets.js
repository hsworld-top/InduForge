/**
 * 动画预设配置
 *
 * 提供常用的动画效果预设
 */

export const animationPresets = {
    // 旋转动画
    rotate: {
        name: '旋转',
        type: 'rotate',
        icon: 'refresh',
        config: {
            duration: 2000,
            iterations: 'infinite',
            direction: 'normal',
            easing: 'linear',
        },
        description: '持续旋转动画，常用于电机、风扇等设备',
    },

    // 闪烁动画
    flash: {
        name: '闪烁',
        type: 'flash',
        icon: 'lightning',
        config: {
            duration: 500,
            iterations: 'infinite',
            colors: [],
        },
        description: '闪烁效果，常用于报警提示',
    },

    // 脉冲动画
    pulse: {
        name: '脉冲',
        type: 'pulse',
        icon: 'radio-button-on',
        config: {
            duration: 1000,
            iterations: 'infinite',
            scale: 1.1,
        },
        description: '缩放脉冲效果，常用于强调重要信息',
    },

    // 抖动动画
    shake: {
        name: '抖动',
        type: 'shake',
        icon: 'vibration',
        config: {
            duration: 500,
            intensity: 10,
        },
        description: '抖动效果，常用于错误提示',
    },

    // 缩放动画
    scale: {
        name: '缩放',
        type: 'scale',
        icon: 'zoom-in',
        config: {
            from: 1,
            to: 1.5,
            duration: 1000,
            iterations: 1,
        },
        description: '缩放动画',
    },

    // 淡入动画
    fadeIn: {
        name: '淡入',
        type: 'fadeIn',
        icon: 'eye',
        config: {
            duration: 300,
            delay: 0,
        },
        description: '淡入效果，常用于元素显示',
    },

    // 淡出动画
    fadeOut: {
        name: '淡出',
        type: 'fadeOut',
        icon: 'eye-off',
        config: {
            duration: 300,
            delay: 0,
        },
        description: '淡出效果，常用于元素隐藏',
    },

    // 从左滑入
    slideInLeft: {
        name: '从左滑入',
        type: 'slideIn',
        icon: 'arrow-right',
        config: {
            direction: 'left',
            duration: 300,
            delay: 0,
        },
        description: '从左侧滑入',
    },

    // 从右滑入
    slideInRight: {
        name: '从右滑入',
        type: 'slideIn',
        icon: 'arrow-left',
        config: {
            direction: 'right',
            duration: 300,
            delay: 0,
        },
        description: '从右侧滑入',
    },

    // 从上滑入
    slideInTop: {
        name: '从上滑入',
        type: 'slideIn',
        icon: 'arrow-down',
        config: {
            direction: 'top',
            duration: 300,
            delay: 0,
        },
        description: '从顶部滑入',
    },

    // 从下滑入
    slideInBottom: {
        name: '从下滑入',
        type: 'slideIn',
        icon: 'arrow-up',
        config: {
            direction: 'bottom',
            duration: 300,
            delay: 0,
        },
        description: '从底部滑入',
    },
};

/**
 * 获取动画预设
 * @param {string} presetName - 预设名称
 * @returns {Object|null} 预设配置
 */
export function getAnimationPreset(presetName) {
    return animationPresets[presetName] || null;
}

/**
 * 获取所有动画预设
 * @returns {Array} 预设列表
 */
export function getAllAnimationPresets() {
    return Object.entries(animationPresets).map(([key, preset]) => ({
        key,
        ...preset,
    }));
}

/**
 * 获取动画预设分类
 * @returns {Object} 分类后的预设
 */
export function getAnimationPresetsByCategory() {
    return {
        '基础动画': [animationPresets.rotate, animationPresets.flash, animationPresets.pulse, animationPresets.shake, animationPresets.scale],
        '进入动画': [animationPresets.fadeIn, animationPresets.slideInLeft, animationPresets.slideInRight, animationPresets.slideInTop, animationPresets.slideInBottom],
        '退出动画': [animationPresets.fadeOut],
    };
}

export default animationPresets;
