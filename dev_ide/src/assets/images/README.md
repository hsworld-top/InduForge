# 图片资源目录

此目录用于存放应用所需的图片资源文件。

## 必需图片文件

### logo.png

- **用途**: 系统Logo，显示在登录页面左侧和页面顶部
- **建议尺寸**: 200x80px 或更高分辨率
- **格式**: PNG（支持透明背景）
- **位置**: `src/assets/images/logo.png`

### login-bg.jpg

- **用途**: 登录页面背景图片
- **建议尺寸**: 1920x1080px 或更高分辨率
- **格式**: JPG/PNG/WebP
- **位置**: `src/assets/images/login-bg.jpg`

## 使用说明

1. 将您的Logo文件重命名为 `logo.png` 并放置在此目录
2. 将您的背景图片重命名为 `login-bg.jpg` 并放置在此目录
3. 图片会被自动打包到构建产物中
4. 图片路径通过应用配置动态配置，支持按租户自定义

## 示例配置

```javascript
// 在应用配置中设置图片路径
const config = {
  logoUrl: '/images/logo.png',
  loginBackgroundUrl: '/images/login-bg.jpg',
}
```

## 注意事项

- 图片文件大小建议控制在 2MB 以内
- Logo图片建议使用透明背景PNG格式
- 背景图片建议使用高分辨率以获得更好的视觉效果
- 移动端适配时图片会自动缩放
