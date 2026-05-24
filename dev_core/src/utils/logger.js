const winston = require('winston')
const { createLogger, format, transports } = winston
const DailyRotateFile = require('winston-daily-rotate-file')
const path = require('path')
const fs = require('fs')

const isProduction = process.env.NODE_ENV === 'production'

// 自定义颜色
winston.addColors({
  error: 'bold red',
  warn: 'yellow',
  info: 'green',
  http: 'magenta',
  verbose: 'cyan',
  debug: 'blue',
  silly: 'magenta',
})

const consoleFormat = isProduction
  ? format.combine(format.timestamp(), format.errors({ stack: true }), format.json())
  : format.combine(
      format.colorize({ all: true }),
      format.timestamp(),
      format.printf(({ level, message, timestamp, ...meta }) => {
        const metaString = Object.keys(meta).length ? ` ${JSON.stringify(meta)}` : ''
        return `[${timestamp}] ${level}: ${message}${metaString}`
      }),
    )

const loggerTransports = [
  new transports.Console({
    level: isProduction ? 'info' : 'debug',
    handleExceptions: true,
    format: consoleFormat,
  }),
]

if (isProduction) {
  const logsDir = path.resolve(__dirname, '../../logs')
  if (!fs.existsSync(logsDir)) {
    fs.mkdirSync(logsDir, { recursive: true })
  }

  loggerTransports.push(
    new DailyRotateFile({
      level: 'info',
      dirname: logsDir,
      filename: '%DATE%.log',
      datePattern: 'YYYY-MM-DD',
      zippedArchive: true,
      maxFiles: '30d',
      auditFile: false, // 禁用审计文件生成
      format: format.combine(format.timestamp(), format.errors({ stack: true }), format.json()),
    }),
  )
}

const logger = createLogger({
  level: isProduction ? 'info' : 'debug',
  transports: loggerTransports,
  exitOnError: false,
})

module.exports = { logger }
