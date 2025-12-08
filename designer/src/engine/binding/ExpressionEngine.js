/**
 * ExpressionEngine - 表达式解析和计算引擎
 *
 * 支持 {{ expression }} 语法，提供内置函数和上下文变量
 */
import { Parser } from 'expr-eval';

class ExpressionEngine {
    constructor() {
        this.parser = new Parser();
        this.context = {};
        this.registerBuiltInFunctions();
    }

    /**
     * 设置表达式上下文
     * @param {Object} context - 上下文对象 { vars, data, $user, $route, etc. }
     */
    setContext(context) {
        this.context = context;
    }

    /**
     * 计算表达式
     * @param {string} expression - 表达式字符串，如 "{{ vars.count * 2 }}"
     * @returns {any} 计算结果
     */
    evaluate(expression) {
        if (!expression || typeof expression !== 'string') {
            return expression;
        }

        // 检查是否是表达式（包含 {{ }} ）
        if (!expression.includes('{{')) {
            return expression;
        }

        try {
            // 提取表达式内容
            const matches = expression.match(/\{\{(.+?)\}\}/g);
            if (!matches) {
                return expression;
            }

            let result = expression;
            for (const match of matches) {
                const expr = match.replace(/^\{\{|\}\}$/g, '').trim();
                const value = this.parser.evaluate(expr, this.context);
                result = result.replace(match, value);
            }

            // 如果整个字符串都是表达式，返回计算值
            if (matches.length === 1 && result === String(this.parser.evaluate(matches[0].replace(/^\{\{|\}\}$/g, '').trim(), this.context))) {
                return this.parser.evaluate(matches[0].replace(/^\{\{|\}\}$/g, '').trim(), this.context);
            }

            return result;
        } catch (error) {
            console.error('Expression evaluation error:', error, 'Expression:', expression);
            return null;
        }
    }

    /**
     * 注册自定义函数
     * @param {string} name - 函数名
     * @param {Function} fn - 函数实现
     */
    registerFunction(name, fn) {
        this.parser.functions[name] = fn;
    }

    /**
     * 注册内置函数
     */
    registerBuiltInFunctions() {
        // 数字格式化
        this.registerFunction('format', (value, decimals = 2) => {
            return Number(value).toFixed(decimals);
        });

        // 条件判断
        this.registerFunction('if', (condition, trueVal, falseVal) => {
            return condition ? trueVal : falseVal;
        });

        // Switch 表达式
        this.registerFunction('switch', (value, ...cases) => {
            for (let i = 0; i < cases.length - 1; i += 2) {
                if (value === cases[i]) {
                    return cases[i + 1];
                }
            }
            return cases[cases.length - 1]; // 默认值
        });

        // 数组求和
        this.registerFunction('sum', (arr, field) => {
            if (!Array.isArray(arr)) return 0;
            if (field) {
                return arr.reduce((sum, item) => sum + (item[field] || 0), 0);
            }
            return arr.reduce((sum, val) => sum + (val || 0), 0);
        });

        // 数组平均值
        this.registerFunction('avg', (arr, field) => {
            if (!Array.isArray(arr) || arr.length === 0) return 0;
            const total = this.parser.functions.sum(arr, field);
            return total / arr.length;
        });

        // 数组最大值
        this.registerFunction('max', (arr, field) => {
            if (!Array.isArray(arr) || arr.length === 0) return null;
            if (field) {
                return Math.max(...arr.map((item) => item[field] || 0));
            }
            return Math.max(...arr);
        });

        // 数组最小值
        this.registerFunction('min', (arr, field) => {
            if (!Array.isArray(arr) || arr.length === 0) return null;
            if (field) {
                return Math.min(...arr.map((item) => item[field] || 0));
            }
            return Math.min(...arr);
        });

        // 字符串截断
        this.registerFunction('truncate', (str, length = 20, suffix = '...') => {
            if (!str || str.length <= length) return str;
            return str.substring(0, length) + suffix;
        });

        // 日期格式化（简单实现）
        this.registerFunction('formatDate', (date, format = 'YYYY-MM-DD') => {
            const d = new Date(date);
            if (isNaN(d.getTime())) return '';

            const year = d.getFullYear();
            const month = String(d.getMonth() + 1).padStart(2, '0');
            const day = String(d.getDate()).padStart(2, '0');
            const hours = String(d.getHours()).padStart(2, '0');
            const minutes = String(d.getMinutes()).padStart(2, '0');
            const seconds = String(d.getSeconds()).padStart(2, '0');

            return format.replace('YYYY', year).replace('MM', month).replace('DD', day).replace('HH', hours).replace('mm', minutes).replace('ss', seconds);
        });

        // 线性变换（工业场景常用）
        this.registerFunction('scale', (value, rawMin, rawMax, euMin, euMax) => {
            return ((value - rawMin) / (rawMax - rawMin)) * (euMax - euMin) + euMin;
        });

        // 范围限制
        this.registerFunction('clamp', (value, min, max) => {
            return Math.max(min, Math.min(max, value));
        });
    }
}

// 导出单例
const expressionEngine = new ExpressionEngine();
export default expressionEngine;
