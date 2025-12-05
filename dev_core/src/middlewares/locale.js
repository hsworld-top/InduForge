/**
 * 语言检测中间件
 * 从请求头中获取语言设置（Accept-Language 或 X-Language）
 * 支持的语言：zh-CN, en
 * 默认语言：zh-CN
 */
function localeMiddleware(req, res, next) {
  // 优先使用 X-Language 头（前端可以显式指定）
  let language = req.headers['x-language'] || req.headers['accept-language'];
  
  // 如果没有指定，使用默认语言
  if (!language) {
    req.language = 'zh-CN';
    res.locals.language = 'zh-CN';
    return next();
  }

  // 处理 Accept-Language 格式（如：zh-CN,zh;q=0.9,en;q=0.8）
  if (language.includes(',')) {
    language = language.split(',')[0].trim();
  }

  // 标准化语言代码
  if (language.toLowerCase().startsWith('zh')) {
    req.language = 'zh-CN';
    res.locals.language = 'zh-CN';
  } else if (language.toLowerCase().startsWith('en')) {
    req.language = 'en';
    res.locals.language = 'en';
  } else {
    // 默认使用中文
    req.language = 'zh-CN';
    res.locals.language = 'zh-CN';
  }

  next();
}

module.exports = { localeMiddleware };

