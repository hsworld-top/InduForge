/**
 * 日期工具函数
 */
import dayjs from 'dayjs'
import { TIME_FORMAT } from '@/constants'

/**
 * 格式化日期
 * @param {Date|string|number} date - 日期
 * @param {string} format - 格式化字符串
 * @returns {string} 格式化后的日期字符串
 */
export function formatDate(date, format = TIME_FORMAT) {
  if (!date) return ''
  const parsed = dayjs(date)
  return parsed.isValid() ? parsed.format(format) : ''
}

/**
 * 格式化日期时间
 * @param {Date|string|number} date - 日期
 * @returns {string} 格式化后的日期时间字符串
 */
export function formatDateTime(date) {
  return formatDate(date, TIME_FORMAT)
}

/**
 * 获取相对时间
 * @param {Date|string|number} date - 日期
 * @returns {string} 相对时间字符串
 */
export function getRelativeTime(date) {
  if (!date) return ''

  const d = dayjs(date)
  if (!d.isValid()) return ''
  const diff = dayjs().diff(d)

  const minute = 60 * 1000
  const hour = 60 * minute
  const day = 24 * hour
  const week = 7 * day
  const month = 30 * day

  if (diff < minute) {
    return '刚刚'
  } else if (diff < hour) {
    return `${Math.floor(diff / minute)}分钟前`
  } else if (diff < day) {
    return `${Math.floor(diff / hour)}小时前`
  } else if (diff < week) {
    return `${Math.floor(diff / day)}天前`
  } else if (diff < month) {
    return `${Math.floor(diff / week)}周前`
  } else {
    return formatDate(d, 'YYYY-MM-DD')
  }
}

/**
 * 获取日期范围
 * @param {number} days - 天数
 * @returns {Array} [startDate, endDate]
 */
export function getDateRange(days) {
  const endDate = dayjs()
  const startDate = dayjs().subtract(days, 'day')

  return [startDate.toDate(), endDate.toDate()]
}

/**
 * 检查日期是否在范围内
 * @param {Date} date - 要检查的日期
 * @param {Date} startDate - 开始日期
 * @param {Date} endDate - 结束日期
 * @returns {boolean} 是否在范围内
 */
export function isDateInRange(date, startDate, endDate) {
  const d = dayjs(date)
  const start = dayjs(startDate)
  const end = dayjs(endDate)
  if (!d.isValid() || !start.isValid() || !end.isValid()) return false
  return d.valueOf() >= start.valueOf() && d.valueOf() <= end.valueOf()
}

/**
 * 格式化货币
 * @param {number} amount - 金额
 * @param {string} currency - 货币符号
 * @returns {string} 格式化后的货币字符串
 */
export function formatCurrency(amount, currency = '¥') {
  if (amount === null || amount === undefined || isNaN(amount)) {
    return '-'
  }

  return `${currency}${Number(amount).toLocaleString('zh-CN', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })}`
}
