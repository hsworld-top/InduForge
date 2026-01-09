-- 添加 entryConfig 字段到 projects 表
-- 用于存储项目入口配置：homePageId, loginPageId, logoutPageId 等
ALTER TABLE projects ADD COLUMN IF NOT EXISTS entryConfig JSON DEFAULT NULL COMMENT '入口配置：homePageId, loginPageId, logoutPageId 等' AFTER projectVariables;
