/**
 * AnimationManager - 动画管理器
 *
 * 使用 GSAP 管理组件动画效果
 */
import gsap from 'gsap';

export class AnimationManager {
    constructor() {
        this.animations = new Map();
        this.timelines = new Map();
    }

    /**
     * 播放动画
     * @param {string} componentId - 组件ID
     * @param {Object} animationConfig - 动画配置
     * @param {Function} conditionEvaluator - 条件评估函数
     */
    play(componentId, animationConfig, conditionEvaluator = null) {
        const { id, type, config, condition, trigger } = animationConfig;

        // 检查条件
        if (condition && conditionEvaluator) {
            const shouldPlay = conditionEvaluator(condition);
            if (!shouldPlay) {
                this.stop(componentId, id);
                return;
            }
        }

        // 获取目标元素
        const target = document.getElementById(componentId);
        if (!target) {
            console.warn(`Animation target not found: ${componentId}`);
            return;
        }

        // 停止已存在的同名动画
        this.stop(componentId, id);

        // 创建动画
        let animation;
        const animKey = `${componentId}:${id}`;

        switch (type) {
            case 'rotate':
                animation = this.createRotateAnimation(target, config);
                break;
            case 'flash':
                animation = this.createFlashAnimation(target, config);
                break;
            case 'pulse':
                animation = this.createPulseAnimation(target, config);
                break;
            case 'shake':
                animation = this.createShakeAnimation(target, config);
                break;
            case 'scale':
                animation = this.createScaleAnimation(target, config);
                break;
            case 'fadeIn':
                animation = this.createFadeInAnimation(target, config);
                break;
            case 'fadeOut':
                animation = this.createFadeOutAnimation(target, config);
                break;
            case 'slideIn':
                animation = this.createSlideInAnimation(target, config);
                break;
            case 'slideOut':
                animation = this.createSlideOutAnimation(target, config);
                break;
            default:
                console.warn(`Unknown animation type: ${type}`);
                return;
        }

        if (animation) {
            this.animations.set(animKey, animation);
        }
    }

    /**
     * 停止动画
     * @param {string} componentId - 组件ID
     * @param {string} animationId - 动画ID（可选，不传则停止所有动画）
     */
    stop(componentId, animationId = null) {
        if (animationId) {
            const animKey = `${componentId}:${animationId}`;
            const animation = this.animations.get(animKey);
            if (animation) {
                animation.kill();
                this.animations.delete(animKey);
            }
        } else {
            // 停止组件的所有动画
            for (const [key, animation] of this.animations.entries()) {
                if (key.startsWith(componentId + ':')) {
                    animation.kill();
                    this.animations.delete(key);
                }
            }
        }
    }

    /**
     * 创建旋转动画
     */
    createRotateAnimation(target, config) {
        const { duration = 2000, iterations = 'infinite', direction = 'normal', easing = 'linear' } = config;

        return gsap.to(target, {
            rotation: direction === 'reverse' ? -360 : 360,
            duration: duration / 1000,
            repeat: iterations === 'infinite' ? -1 : iterations - 1,
            ease: easing,
        });
    }

    /**
     * 创建闪烁动画
     */
    createFlashAnimation(target, config) {
        const { duration = 500, iterations = 'infinite', colors = [] } = config;

        if (colors.length >= 2) {
            return gsap.to(target, {
                backgroundColor: colors[1],
                duration: duration / 1000 / 2,
                repeat: iterations === 'infinite' ? -1 : iterations * 2 - 1,
                yoyo: true,
                ease: 'none',
            });
        } else {
            return gsap.to(target, {
                opacity: 0,
                duration: duration / 1000 / 2,
                repeat: iterations === 'infinite' ? -1 : iterations * 2 - 1,
                yoyo: true,
                ease: 'none',
            });
        }
    }

    /**
     * 创建脉冲动画
     */
    createPulseAnimation(target, config) {
        const { duration = 1000, iterations = 'infinite', scale = 1.1 } = config;

        return gsap.to(target, {
            scale: scale,
            duration: duration / 1000 / 2,
            repeat: iterations === 'infinite' ? -1 : iterations * 2 - 1,
            yoyo: true,
            ease: 'power1.inOut',
        });
    }

    /**
     * 创建抖动动画
     */
    createShakeAnimation(target, config) {
        const { duration = 500, intensity = 10 } = config;

        return gsap.to(target, {
            x: `+=${intensity}`,
            duration: duration / 1000 / 8,
            repeat: 7,
            yoyo: true,
            ease: 'power1.inOut',
        });
    }

    /**
     * 创建缩放动画
     */
    createScaleAnimation(target, config) {
        const { from = 1, to = 1.5, duration = 1000, iterations = 1 } = config;

        return gsap.fromTo(
            target,
            { scale: from },
            {
                scale: to,
                duration: duration / 1000,
                repeat: iterations === 'infinite' ? -1 : iterations - 1,
                yoyo: iterations > 1,
                ease: 'power2.inOut',
            },
        );
    }

    /**
     * 创建淡入动画
     */
    createFadeInAnimation(target, config) {
        const { duration = 300, delay = 0 } = config;

        return gsap.fromTo(
            target,
            { opacity: 0 },
            {
                opacity: 1,
                duration: duration / 1000,
                delay: delay / 1000,
                ease: 'power2.out',
            },
        );
    }

    /**
     * 创建淡出动画
     */
    createFadeOutAnimation(target, config) {
        const { duration = 300, delay = 0 } = config;

        return gsap.to(target, {
            opacity: 0,
            duration: duration / 1000,
            delay: delay / 1000,
            ease: 'power2.in',
        });
    }

    /**
     * 创建滑入动画
     */
    createSlideInAnimation(target, config) {
        const { direction = 'left', duration = 300, delay = 0 } = config;

        const fromProps = {};
        switch (direction) {
            case 'left':
                fromProps.x = -100;
                break;
            case 'right':
                fromProps.x = 100;
                break;
            case 'top':
                fromProps.y = -100;
                break;
            case 'bottom':
                fromProps.y = 100;
                break;
        }

        return gsap.fromTo(
            target,
            { ...fromProps, opacity: 0 },
            {
                x: 0,
                y: 0,
                opacity: 1,
                duration: duration / 1000,
                delay: delay / 1000,
                ease: 'power2.out',
            },
        );
    }

    /**
     * 创建滑出动画
     */
    createSlideOutAnimation(target, config) {
        const { direction = 'left', duration = 300, delay = 0 } = config;

        const toProps = { opacity: 0 };
        switch (direction) {
            case 'left':
                toProps.x = -100;
                break;
            case 'right':
                toProps.x = 100;
                break;
            case 'top':
                toProps.y = -100;
                break;
            case 'bottom':
                toProps.y = 100;
                break;
        }

        return gsap.to(target, {
            ...toProps,
            duration: duration / 1000,
            delay: delay / 1000,
            ease: 'power2.in',
        });
    }

    /**
     * 清理所有动画
     */
    clear() {
        for (const animation of this.animations.values()) {
            animation.kill();
        }
        this.animations.clear();

        for (const timeline of this.timelines.values()) {
            timeline.kill();
        }
        this.timelines.clear();
    }
}

export default AnimationManager;
