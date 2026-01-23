-- ============================================
-- InduForge 低代码平台 - 数据库初始化脚本
-- 版本: 2.0.0
-- 更新日期: 2025-12
-- ============================================

-- ============================================
-- 第一部分：基础平台表
-- ============================================

-- 1.1 租户表
CREATE TABLE IF NOT EXISTS `tenants` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `name` varchar(100) NOT NULL COMMENT '租户名称',
  `code` varchar(50) NOT NULL COMMENT '租户代码(唯一标识)',
  `description` text COMMENT '租户描述',
  `status` enum('active','inactive','suspended') NOT NULL DEFAULT 'active' COMMENT '租户状态',
  `contactEmail` varchar(255) DEFAULT NULL COMMENT '联系邮箱',
  `contactPhone` varchar(20) DEFAULT NULL COMMENT '联系电话',
  `maxUsers` int NOT NULL DEFAULT 100 COMMENT '最大用户数',
  `maxProjects` int NOT NULL DEFAULT 50 COMMENT '最大工程数',
  `maxStorage` bigint NOT NULL DEFAULT 10737418240 COMMENT '最大存储空间(字节,默认10GB)',
  `usedStorage` bigint NOT NULL DEFAULT 0 COMMENT '已用存储空间(字节)',
  `logoUrl` varchar(500) DEFAULT NULL COMMENT 'Logo资源路径',
  `loginBackgroundUrl` varchar(500) DEFAULT NULL COMMENT '登录页背景图路径',
  `companyName` varchar(200) DEFAULT NULL COMMENT '公司名称',
  `companyAddress` text COMMENT '公司地址',
  `companyPhone` varchar(20) DEFAULT NULL COMMENT '公司电话',
  `companyWebsite` varchar(500) DEFAULT NULL COMMENT '公司网站',
  `settings` json DEFAULT NULL COMMENT '租户级配置(主题色/语言等)',
  `expiresAt` datetime(6) DEFAULT NULL COMMENT '租户到期时间',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='租户表';

CREATE UNIQUE INDEX `tenants_code_uq` ON `tenants` (`code`);
CREATE INDEX `tenants_status_idx` ON `tenants` (`status`);
CREATE INDEX `tenants_expires_idx` ON `tenants` (`expiresAt`);

-- 1.2 用户表
CREATE TABLE IF NOT EXISTS `users` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `tenantId` char(36) NOT NULL COMMENT '所属租户ID',
  `username` varchar(50) NOT NULL COMMENT '用户名',
  `password` varchar(255) NOT NULL COMMENT '密码哈希',
  `email` varchar(255) DEFAULT NULL COMMENT '邮箱',
  `phone` varchar(20) DEFAULT NULL COMMENT '手机号',
  `fullName` varchar(100) DEFAULT NULL COMMENT '真实姓名',
  `avatar` varchar(500) DEFAULT NULL COMMENT '头像URL',
  `role` enum('SUPER_ADMIN','SYSTEM_ADMIN','PROJECT_ADMIN','DEVELOPER','OPERATOR','VIEWER') NOT NULL COMMENT '系统角色',
  `status` enum('active','inactive','suspended') NOT NULL DEFAULT 'active' COMMENT '用户状态',
  `preferences` json DEFAULT NULL COMMENT '用户偏好设置(主题/语言/布局)',
  `lastLoginAt` datetime(6) DEFAULT NULL COMMENT '最后登录时间',
  `lastLoginIp` varchar(45) DEFAULT NULL COMMENT '最后登录IP',
  `passwordChangedAt` datetime(6) DEFAULT NULL COMMENT '密码最后修改时间',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `users_fk_tenant` FOREIGN KEY (`tenantId`) REFERENCES `tenants` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

CREATE UNIQUE INDEX `users_tenant_username_uq` ON `users` (`tenantId`, `username`);
CREATE INDEX `users_tenant_role_idx` ON `users` (`tenantId`, `role`);
CREATE INDEX `users_status_idx` ON `users` (`status`);
CREATE INDEX `users_email_idx` ON `users` (`email`);

-- 1.3 工程表
CREATE TABLE IF NOT EXISTS `projects` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `tenantId` char(36) NOT NULL COMMENT '所属租户ID',
  `name` varchar(200) NOT NULL COMMENT '工程名称',
    `code` varchar(50) DEFAULT NULL COMMENT '工程代码',
    `description` text COMMENT '工程描述',
    `projectVariables` json DEFAULT NULL COMMENT '工程级别全局变量',
    `entryConfig` json DEFAULT NULL COMMENT '入口配置：homePageId, loginPageId, logoutPageId 等',
    `colorTag` varchar(20) NOT NULL DEFAULT '#3b82f6' COMMENT '颜色标签',
  `icon` varchar(100) DEFAULT NULL COMMENT '工程图标',
  `status` enum('active','archived','deleted') NOT NULL DEFAULT 'active' COMMENT '工程状态',
  `visibility` enum('private','internal','public') NOT NULL DEFAULT 'private' COMMENT '可见性',
  `createdBy` char(36) NOT NULL COMMENT '创建者ID',
  `updatedBy` char(36) DEFAULT NULL COMMENT '更新者ID',
  `archivedAt` datetime(6) DEFAULT NULL COMMENT '归档时间',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `projects_fk_tenant` FOREIGN KEY (`tenantId`) REFERENCES `tenants` (`id`) ON DELETE CASCADE,
  CONSTRAINT `projects_fk_creator` FOREIGN KEY (`createdBy`) REFERENCES `users` (`id`),
  CONSTRAINT `projects_fk_updater` FOREIGN KEY (`updatedBy`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='工程表';

CREATE INDEX `projects_tenant_idx` ON `projects` (`tenantId`);
CREATE INDEX `projects_status_idx` ON `projects` (`status`);
CREATE UNIQUE INDEX `projects_tenant_code_uq` ON `projects` (`tenantId`, `code`);

-- 1.4 工程成员表
CREATE TABLE IF NOT EXISTS `project_members` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '工程ID',
  `userId` char(36) NOT NULL COMMENT '用户ID',
  `role` enum('OWNER','ADMIN','DEVELOPER','VIEWER') NOT NULL DEFAULT 'VIEWER' COMMENT '工程内角色',
  `permissions` json DEFAULT NULL COMMENT '细粒度权限配置',
  `joinedAt` datetime(6) NOT NULL COMMENT '加入时间',
  `invitedBy` char(36) DEFAULT NULL COMMENT '邀请人ID',
  PRIMARY KEY (`id`),
  UNIQUE KEY `project_user_uq` (`projectId`, `userId`),
  CONSTRAINT `pm_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
  CONSTRAINT `pm_fk_user` FOREIGN KEY (`userId`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='工程成员表';

-- 1.5 操作日志表
CREATE TABLE IF NOT EXISTS `logs` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `tenantId` char(36) DEFAULT NULL COMMENT '租户ID',
  `projectId` char(36) DEFAULT NULL COMMENT '工程ID',
  `userId` char(36) DEFAULT NULL COMMENT '操作用户ID',
  `level` enum('debug','info','warning','error') NOT NULL DEFAULT 'info' COMMENT '日志级别',
  `category` varchar(50) DEFAULT NULL COMMENT '日志分类(auth/data/design/system)',
  `action` varchar(100) NOT NULL COMMENT '操作类型',
  `resource` varchar(100) DEFAULT NULL COMMENT '资源类型',
  `resourceId` char(36) DEFAULT NULL COMMENT '资源ID',
  `message` text NOT NULL COMMENT '日志消息',
  `details` json DEFAULT NULL COMMENT '详细信息',
  `ip` varchar(45) DEFAULT NULL COMMENT 'IP地址',
  `userAgent` varchar(500) DEFAULT NULL COMMENT '用户代理',
  `duration` int DEFAULT NULL COMMENT '操作耗时(ms)',
  `createdAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `logs_fk_tenant` FOREIGN KEY (`tenantId`) REFERENCES `tenants` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='操作日志表';

CREATE INDEX `logs_tenant_idx` ON `logs` (`tenantId`);
CREATE INDEX `logs_project_idx` ON `logs` (`projectId`);
CREATE INDEX `logs_user_idx` ON `logs` (`userId`);
CREATE INDEX `logs_category_action_idx` ON `logs` (`category`, `action`);
CREATE INDEX `logs_created_at_idx` ON `logs` (`createdAt`);

-- 1.6 数据字典表
CREATE TABLE IF NOT EXISTS `dictionaries` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `tenantId` char(36) DEFAULT NULL COMMENT '租户ID(NULL表示系统级)',
  `code` varchar(50) NOT NULL COMMENT '字典代码',
  `name` varchar(100) NOT NULL COMMENT '字典名称',
  `description` varchar(255) DEFAULT NULL COMMENT '描述',
  `isSystem` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否系统内置',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='数据字典表';

CREATE UNIQUE INDEX `dict_tenant_code_uq` ON `dictionaries` (`tenantId`, `code`);

-- 1.7 数据字典项表
CREATE TABLE IF NOT EXISTS `dictionary_items` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `dictionaryId` char(36) NOT NULL COMMENT '字典ID',
  `value` varchar(100) NOT NULL COMMENT '字典值',
  `label` varchar(200) NOT NULL COMMENT '显示标签',
  `labelEn` varchar(200) DEFAULT NULL COMMENT '英文标签',
  `color` varchar(20) DEFAULT NULL COMMENT '颜色标识',
  `icon` varchar(100) DEFAULT NULL COMMENT '图标',
  `sortOrder` int NOT NULL DEFAULT 0 COMMENT '排序',
  `isDefault` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否默认值',
  `isEnabled` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  `extra` json DEFAULT NULL COMMENT '扩展数据',
  PRIMARY KEY (`id`),
  CONSTRAINT `dict_items_fk_dict` FOREIGN KEY (`dictionaryId`) REFERENCES `dictionaries` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='数据字典项表';

CREATE UNIQUE INDEX `dict_items_value_uq` ON `dictionary_items` (`dictionaryId`, `value`);
CREATE INDEX `dict_items_sort_idx` ON `dictionary_items` (`dictionaryId`, `sortOrder`);


-- ============================================
-- 第二部分：数据中心相关表
-- ============================================

-- 2.1 数据连接表
CREATE TABLE IF NOT EXISTS `data_connections` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '所属工程ID',
  `name` varchar(100) NOT NULL COMMENT '连接名称',
  `type` enum('relational','mqtt','websocket','opcua','modbus','http','s7') NOT NULL COMMENT '连接类型',
  `category` enum('database','message','protocol','api') NOT NULL DEFAULT 'api' COMMENT '连接类别',
  `status` enum('connected','disconnected','error','unknown') NOT NULL DEFAULT 'unknown' COMMENT '连接状态',
  `isEnabled` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  `retryCount` int NOT NULL DEFAULT 3 COMMENT '重连次数',
  `retryInterval` int NOT NULL DEFAULT 5000 COMMENT '重连间隔(ms)',
  `healthCheckInterval` int DEFAULT 30000 COMMENT '健康检查间隔(ms)',
  `lastConnectedAt` datetime(6) DEFAULT NULL COMMENT '最后连接时间',
  `lastErrorMessage` text DEFAULT NULL COMMENT '最后错误信息',
  `createdBy` char(36) NOT NULL COMMENT '创建者ID',
  `updatedBy` char(36) DEFAULT NULL COMMENT '更新者ID',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `data_conn_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
  CONSTRAINT `data_conn_fk_creator` FOREIGN KEY (`createdBy`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='数据连接表';

CREATE UNIQUE INDEX `data_conn_project_name_uq` ON `data_connections` (`projectId`, `name`);
CREATE INDEX `data_conn_type_idx` ON `data_connections` (`type`);
CREATE INDEX `data_conn_status_idx` ON `data_connections` (`status`);

-- 2.2 关系型数据库配置表
CREATE TABLE IF NOT EXISTS `data_relational_configs` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `connectionId` char(36) NOT NULL COMMENT '连接ID',
  `dbType` enum('mysql','postgresql','sqlserver','oracle','sqlite','clickhouse') NOT NULL COMMENT '数据库类型',
  `host` varchar(255) NOT NULL COMMENT '主机地址',
  `port` int NOT NULL COMMENT '端口号',
  `database` varchar(100) NOT NULL COMMENT '数据库名',
  `username` varchar(100) NOT NULL COMMENT '用户名',
  `password` text NOT NULL COMMENT '密码(加密存储)',
  `schema` varchar(100) DEFAULT NULL COMMENT 'Schema名(PostgreSQL/Oracle)',
  `charset` varchar(50) DEFAULT 'utf8mb4' COMMENT '字符集',
  `timezone` varchar(50) DEFAULT NULL COMMENT '时区',
  `ssl` tinyint(1) DEFAULT 0 COMMENT '是否启用SSL',
  `sslConfig` json DEFAULT NULL COMMENT 'SSL配置',
  `poolMin` int DEFAULT 2 COMMENT '连接池最小连接数',
  `poolMax` int DEFAULT 10 COMMENT '连接池最大连接数',
  `acquireTimeout` int DEFAULT 60000 COMMENT '获取连接超时(ms)',
  `idleTimeout` int DEFAULT 30000 COMMENT '空闲超时(ms)',
  `queryTimeout` int DEFAULT 60000 COMMENT '查询超时(ms)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `connectionId` (`connectionId`),
  CONSTRAINT `data_rel_fk_conn` FOREIGN KEY (`connectionId`) REFERENCES `data_connections` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='关系型数据库配置表';

-- 2.3 MQTT配置表
CREATE TABLE IF NOT EXISTS `data_mqtt_configs` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `connectionId` char(36) NOT NULL COMMENT '连接ID',
  `protocol` enum('mqtt','mqtts','ws','wss') NOT NULL DEFAULT 'mqtt' COMMENT '协议',
  `brokerUrl` varchar(500) NOT NULL COMMENT 'Broker地址(支持多个,逗号分隔)',
  `port` int DEFAULT 1883 COMMENT '端口号',
  `clientId` varchar(100) DEFAULT NULL COMMENT '客户端ID',
  `username` varchar(100) DEFAULT NULL COMMENT '用户名',
  `password` varchar(255) DEFAULT NULL COMMENT '密码',
  `keepalive` int DEFAULT 60 COMMENT '心跳间隔(秒)',
  `cleanSession` tinyint(1) DEFAULT 1 COMMENT '清除会话',
  `qos` tinyint DEFAULT 0 COMMENT '默认QoS(0/1/2)',
  `reconnectPeriod` int DEFAULT 5000 COMMENT '重连间隔(ms)',
  `connectTimeout` int DEFAULT 30000 COMMENT '连接超时(ms)',
  `will` json DEFAULT NULL COMMENT '遗嘱消息配置',
  `sslConfig` json DEFAULT NULL COMMENT 'SSL/TLS配置',
  PRIMARY KEY (`id`),
  UNIQUE KEY `connectionId` (`connectionId`),
  CONSTRAINT `data_mqtt_fk_conn` FOREIGN KEY (`connectionId`) REFERENCES `data_connections` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='MQTT配置表';

-- 2.3.1 MQTT主题订阅表
CREATE TABLE IF NOT EXISTS `data_mqtt_subscriptions` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '所属工程ID',
  `connectionId` char(36) NOT NULL COMMENT 'MQTT连接ID',
  `name` varchar(100) NOT NULL COMMENT '订阅名称',
  `topic` varchar(500) NOT NULL COMMENT 'MQTT主题(支持通配符)',
  `qos` tinyint DEFAULT 0 COMMENT 'QoS等级(0/1/2)',
  `description` text COMMENT '订阅描述',
  `isEnabled` tinyint(1) DEFAULT 1 COMMENT '是否启用',
  `messageRetention` int DEFAULT 100 COMMENT '消息保留数量',
  `createdBy` char(36) NOT NULL COMMENT '创建者ID',
  `updatedBy` char(36) DEFAULT NULL COMMENT '更新者ID',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `mqtt_subs_project_name_uq` (`projectId`, `name`),
  KEY `mqtt_subs_conn_idx` (`connectionId`),
  KEY `mqtt_subs_enabled_idx` (`connectionId`, `isEnabled`),
  CONSTRAINT `mqtt_subs_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
  CONSTRAINT `mqtt_subs_fk_conn` FOREIGN KEY (`connectionId`) REFERENCES `data_connections` (`id`) ON DELETE CASCADE,
  CONSTRAINT `mqtt_subs_fk_creator` FOREIGN KEY (`createdBy`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='MQTT主题订阅表';

-- 2.3.0 MQTT变量组表
CREATE TABLE IF NOT EXISTS `data_mqtt_tag_groups` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '所属工程ID',
  `subscriptionId` char(36) NOT NULL COMMENT '订阅ID',
  `name` varchar(100) NOT NULL COMMENT '分组名称',
  `code` varchar(100) NOT NULL COMMENT '分组标识符',
  `description` text COMMENT '分组描述',
  `color` varchar(20) DEFAULT NULL COMMENT '分组颜色(用于UI显示)',
  `icon` varchar(50) DEFAULT NULL COMMENT '分组图标',
  `order` int NOT NULL DEFAULT 0 COMMENT '显示顺序',
  `createdBy` char(36) NOT NULL COMMENT '创建者ID',
  `updatedBy` char(36) DEFAULT NULL COMMENT '更新者ID',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `mqtt_tag_groups_subscription_code_uq` (`subscriptionId`, `code`),
  KEY `mqtt_tag_groups_project_idx` (`projectId`),
  KEY `mqtt_tag_groups_subscription_idx` (`subscriptionId`),
  CONSTRAINT `mqtt_tag_groups_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
  CONSTRAINT `mqtt_tag_groups_fk_subscription` FOREIGN KEY (`subscriptionId`) REFERENCES `data_mqtt_subscriptions` (`id`) ON DELETE CASCADE,
  CONSTRAINT `mqtt_tag_groups_fk_creator` FOREIGN KEY (`createdBy`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='MQTT变量分组表';

-- 2.3.1 MQTT变量定义表
CREATE TABLE IF NOT EXISTS `data_mqtt_tags` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '所属工程ID',
  `subscriptionId` char(36) NOT NULL COMMENT '订阅ID',
  `groupId` char(36) DEFAULT NULL COMMENT '所属分组ID',
  `name` varchar(100) NOT NULL COMMENT '变量名称',
  `code` varchar(100) NOT NULL COMMENT '变量标识符(用于引用)',
  `description` text COMMENT '变量描述',
  `dataType` enum('string','number','boolean','object','array') NOT NULL DEFAULT 'string' COMMENT '数据类型',
  `parseType` enum('jsonpath','regex','script','fixed') NOT NULL DEFAULT 'jsonpath' COMMENT '解析类型',
  `parseRule` text NOT NULL COMMENT '解析规则(JSONPath表达式/正则表达式/脚本代码)',
  `defaultValue` text COMMENT '默认值',
  `unit` varchar(50) DEFAULT NULL COMMENT '单位',
  `transform` text COMMENT '值转换函数(JavaScript代码)',
  `validation` json DEFAULT NULL COMMENT '验证规则(min/max/pattern等)',
  `isEnabled` tinyint(1) DEFAULT 1 COMMENT '是否启用',
  `order` int DEFAULT 0 COMMENT '显示顺序',
  `createdBy` char(36) NOT NULL COMMENT '创建者ID',
  `updatedBy` char(36) DEFAULT NULL COMMENT '更新者ID',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `mqtt_tags_project_code_uq` (`projectId`, `code`),
  KEY `mqtt_tags_sub_idx` (`subscriptionId`),
  KEY `mqtt_tags_group_idx` (`groupId`),
  KEY `mqtt_tags_enabled_idx` (`subscriptionId`, `isEnabled`),
  CONSTRAINT `mqtt_tags_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
  CONSTRAINT `mqtt_tags_fk_subscription` FOREIGN KEY (`subscriptionId`) REFERENCES `data_mqtt_subscriptions` (`id`) ON DELETE CASCADE,
  CONSTRAINT `mqtt_tags_fk_group` FOREIGN KEY (`groupId`) REFERENCES `data_mqtt_tag_groups` (`id`) ON DELETE SET NULL,
  CONSTRAINT `mqtt_tags_fk_creator` FOREIGN KEY (`createdBy`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='MQTT变量定义表';

-- 注意：MQTT变量值将存储在 Redis 中，不再使用 MySQL 表

-- 2.4 HTTP配置表
CREATE TABLE IF NOT EXISTS `data_http_configs` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `connectionId` char(36) NOT NULL COMMENT '连接ID',
  `baseUrl` varchar(500) NOT NULL COMMENT '基础URL',
  `timeout` int DEFAULT 30000 COMMENT '请求超时(ms)',
  `headers` json DEFAULT NULL COMMENT '默认请求头',
  `authType` enum('none','basic','bearer','apikey','oauth2') DEFAULT 'none' COMMENT '认证类型',
  `authConfig` json DEFAULT NULL COMMENT '认证配置',
  `proxy` json DEFAULT NULL COMMENT '代理配置',
  `retryConfig` json DEFAULT NULL COMMENT '重试配置',
  PRIMARY KEY (`id`),
  UNIQUE KEY `connectionId` (`connectionId`),
  CONSTRAINT `data_http_fk_conn` FOREIGN KEY (`connectionId`) REFERENCES `data_connections` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='HTTP配置表';

-- 2.5 通用驱动配置表(OPC UA/Modbus/S7等)
CREATE TABLE IF NOT EXISTS `data_driver_configs` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `connectionId` char(36) NOT NULL COMMENT '连接ID',
  `driverType` varchar(50) NOT NULL COMMENT '驱动类型(opcua/modbus-tcp/modbus-rtu/s7)',
  `config` json NOT NULL COMMENT '驱动特定配置',
  `securityConfig` json DEFAULT NULL COMMENT '安全配置',
  PRIMARY KEY (`id`),
  UNIQUE KEY `connectionId` (`connectionId`),
  CONSTRAINT `data_driver_fk_conn` FOREIGN KEY (`connectionId`) REFERENCES `data_connections` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='通用驱动配置表';

-- 2.6 点位分组表
CREATE TABLE IF NOT EXISTS `data_tag_groups` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `connectionId` char(36) NOT NULL COMMENT '所属连接ID',
  `parentId` char(36) DEFAULT NULL COMMENT '父分组ID',
  `name` varchar(100) NOT NULL COMMENT '分组名称',
  `description` varchar(255) DEFAULT NULL COMMENT '分组描述',
  `scanRate` int DEFAULT 1000 COMMENT '采集周期(ms)',
  `isEnabled` tinyint(1) DEFAULT 1 COMMENT '是否启用',
  `sortOrder` int DEFAULT 0 COMMENT '排序',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `tag_groups_fk_conn` FOREIGN KEY (`connectionId`) REFERENCES `data_connections` (`id`) ON DELETE CASCADE,
  CONSTRAINT `tag_groups_fk_parent` FOREIGN KEY (`parentId`) REFERENCES `data_tag_groups` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='点位分组表';

CREATE UNIQUE INDEX `tag_groups_conn_name_uq` ON `data_tag_groups` (`connectionId`, `parentId`, `name`);

-- 2.7 数据点位表
CREATE TABLE IF NOT EXISTS `data_tags` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `connectionId` char(36) NOT NULL COMMENT '所属连接ID',
  `groupId` char(36) DEFAULT NULL COMMENT '点位分组ID',
  `name` varchar(100) NOT NULL COMMENT '点位标识(英文)',
  `displayName` varchar(200) DEFAULT NULL COMMENT '显示名称(中文)',
  `address` varchar(500) NOT NULL COMMENT '物理地址/Topic/寄存器地址',
  `dataType` enum('bool','int8','uint8','int16','uint16','int32','uint32','int64','uint64','float','double','string','json','bytes') NOT NULL DEFAULT 'float' COMMENT '数据类型',
  `accessMode` enum('read','write','read_write') NOT NULL DEFAULT 'read' COMMENT '访问模式',
  `defaultValue` varchar(255) DEFAULT NULL COMMENT '默认值',
  `unit` varchar(50) DEFAULT NULL COMMENT '单位',
  `precision` int DEFAULT NULL COMMENT '小数精度',
  `scaling` json DEFAULT NULL COMMENT '线性变换{rawMin,rawMax,euMin,euMax,formula}',
  `valueMapping` json DEFAULT NULL COMMENT '值映射{0:"停止",1:"运行"}',
  `minValue` double DEFAULT NULL COMMENT '最小值(用于校验)',
  `maxValue` double DEFAULT NULL COMMENT '最大值(用于校验)',
  `deadband` double DEFAULT NULL COMMENT '死区(变化阈值)',
  `description` text COMMENT '备注',
  `isEnabled` tinyint(1) DEFAULT 1 COMMENT '是否启用',
  `sortOrder` int DEFAULT 0 COMMENT '排序',
  `metadata` json DEFAULT NULL COMMENT '扩展元数据',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `tags_fk_conn` FOREIGN KEY (`connectionId`) REFERENCES `data_connections` (`id`) ON DELETE CASCADE,
  CONSTRAINT `tags_fk_group` FOREIGN KEY (`groupId`) REFERENCES `data_tag_groups` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='数据点位表';

CREATE UNIQUE INDEX `tags_conn_name_uq` ON `data_tags` (`connectionId`, `name`);
CREATE INDEX `tags_group_idx` ON `data_tags` (`groupId`);
CREATE INDEX `tags_enabled_idx` ON `data_tags` (`connectionId`, `isEnabled`);

-- 2.8 数据查询表
CREATE TABLE IF NOT EXISTS `data_queries` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '所属工程ID',
  `connectionId` char(36) NOT NULL COMMENT '数据连接ID',
  `name` varchar(200) NOT NULL COMMENT '查询名称',
  `description` text COMMENT '查询描述',
  `category` varchar(100) DEFAULT NULL COMMENT '查询分类',
  `queryType` enum('sql','tags','http','mqtt_pub','mqtt_sub') NOT NULL COMMENT '查询类型',
  `config` json NOT NULL COMMENT '查询配置(SQL/参数/URL等)',
  `transformer` text DEFAULT NULL COMMENT '数据转换脚本',
  `isEnabled` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  `cacheEnabled` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否启用缓存',
  `cacheTtl` int DEFAULT 300 COMMENT '缓存过期时间(秒)',
  `timeout` int DEFAULT 30000 COMMENT '执行超时(ms)',
  `createdBy` char(36) NOT NULL COMMENT '创建者ID',
  `updatedBy` char(36) DEFAULT NULL COMMENT '更新者ID',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `queries_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
  CONSTRAINT `queries_fk_conn` FOREIGN KEY (`connectionId`) REFERENCES `data_connections` (`id`) ON DELETE CASCADE,
  CONSTRAINT `queries_fk_creator` FOREIGN KEY (`createdBy`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='数据查询表';

CREATE UNIQUE INDEX `queries_project_name_uq` ON `data_queries` (`projectId`, `name`);
CREATE INDEX `queries_type_idx` ON `data_queries` (`queryType`);

-- 2.8.1 数据点表
CREATE TABLE IF NOT EXISTS `data_points` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `project_id` char(36) NOT NULL COMMENT '所属工程ID',
  `path` varchar(255) NOT NULL COMMENT '数据点路径',
  `name` varchar(100) NOT NULL COMMENT '数据点名称',
  `description` text COMMENT '描述',
  `source_type` varchar(50) NOT NULL COMMENT '来源类型',
  `source_id` char(36) DEFAULT NULL COMMENT '来源ID',
  `source_config` json DEFAULT NULL COMMENT '来源配置',
  `data_type` varchar(20) NOT NULL COMMENT '数据类型',
  `unit` varchar(20) DEFAULT NULL COMMENT '单位',
  `precision_num` int DEFAULT NULL COMMENT '精度',
  `default_value` text DEFAULT NULL COMMENT '默认值',
  `min_value` decimal(20,6) DEFAULT NULL COMMENT '最小值',
  `max_value` decimal(20,6) DEFAULT NULL COMMENT '最大值',
  `alarm_low` decimal(20,6) DEFAULT NULL COMMENT '低报警阈值',
  `alarm_high` decimal(20,6) DEFAULT NULL COMMENT '高报警阈值',
  `tags` json DEFAULT NULL COMMENT '标签',
  `refresh_mode` varchar(20) DEFAULT 'auto' COMMENT '刷新模式',
  `refresh_interval` int DEFAULT NULL COMMENT '刷新间隔',
  `status` varchar(20) DEFAULT 'active' COMMENT '状态',
  `created_by` char(36) DEFAULT NULL COMMENT '创建者ID',
  `updated_by` char(36) DEFAULT NULL COMMENT '更新者ID',
  `created_at` datetime(6) NOT NULL,
  `updated_at` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `data_points_project_path_uq` (`project_id`, `path`),
  KEY `data_points_project_idx` (`project_id`),
  KEY `data_points_source_type_idx` (`source_type`),
  KEY `data_points_status_idx` (`status`),
  CONSTRAINT `data_points_fk_project` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
  CONSTRAINT `data_points_fk_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='数据点表';

-- 2.9 SQL配置表（用于存储预定义SQL语句）
CREATE TABLE IF NOT EXISTS `data_sql_configs` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `connectionId` char(36) NOT NULL COMMENT '连接ID',
  `name` varchar(100) NOT NULL COMMENT 'SQL配置名称',
  `description` text COMMENT '描述',
  `sqlStatement` text NOT NULL COMMENT 'SQL语句',
  `parameters` json DEFAULT NULL COMMENT '参数定义',
  `isEnabled` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  `createdBy` char(36) NOT NULL COMMENT '创建者ID',
  `updatedBy` char(36) DEFAULT NULL COMMENT '更新者ID',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `sql_configs_fk_conn` FOREIGN KEY (`connectionId`) REFERENCES `data_connections` (`id`) ON DELETE CASCADE,
  CONSTRAINT `sql_configs_fk_creator` FOREIGN KEY (`createdBy`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='SQL配置表';

CREATE UNIQUE INDEX `sql_configs_conn_name_uq` ON `data_sql_configs` (`connectionId`, `name`);

-- 2.10 查询执行日志表
CREATE TABLE IF NOT EXISTS `data_query_logs` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `queryId` char(36) NOT NULL COMMENT '查询ID',
  `connectionId` char(36) NOT NULL COMMENT '连接ID',
  `executedBy` char(36) DEFAULT NULL COMMENT '执行者ID',
  `source` enum('manual','scheduled','trigger','api') NOT NULL DEFAULT 'manual' COMMENT '执行来源',
  `parameters` json DEFAULT NULL COMMENT '执行参数',
  `executionTime` int NOT NULL COMMENT '执行时间(ms)',
  `resultCount` int DEFAULT NULL COMMENT '结果行数',
  `resultSize` int DEFAULT NULL COMMENT '结果大小(字节)',
  `status` enum('success','error','timeout','cancelled') NOT NULL COMMENT '执行状态',
  `errorCode` varchar(50) DEFAULT NULL COMMENT '错误代码',
  `errorMessage` text COMMENT '错误信息',
  `executedAt` datetime(6) NOT NULL COMMENT '执行时间',
  PRIMARY KEY (`id`),
  KEY `query_logs_query_idx` (`queryId`),
  KEY `query_logs_time_idx` (`executedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='查询执行日志表';


-- ============================================
-- 第三部分：告警系统相关表
-- ============================================

-- 3.1 告警规则表
CREATE TABLE IF NOT EXISTS `alarm_rules` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '所属工程ID',
  `name` varchar(100) NOT NULL COMMENT '规则名称',
  `description` text COMMENT '规则描述',
  `type` enum('threshold','deviation','rate','expression','deadband') NOT NULL COMMENT '告警类型',
  `severity` enum('info','warning','error','critical') NOT NULL DEFAULT 'warning' COMMENT '告警级别',
  `source` json NOT NULL COMMENT '数据源配置{type,tagId/queryId,field}',
  `condition` json NOT NULL COMMENT '触发条件{operator,value,duration,expression}',
  `actions` json DEFAULT NULL COMMENT '触发动作[{type,config}]',
  `cooldown` int DEFAULT 60 COMMENT '冷却时间(秒)',
  `isEnabled` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  `priority` int DEFAULT 0 COMMENT '优先级',
  `tags` json DEFAULT NULL COMMENT '标签数组',
  `createdBy` char(36) NOT NULL COMMENT '创建者ID',
  `updatedBy` char(36) DEFAULT NULL COMMENT '更新者ID',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `alarm_rules_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
  CONSTRAINT `alarm_rules_fk_creator` FOREIGN KEY (`createdBy`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='告警规则表';

CREATE INDEX `alarm_rules_project_idx` ON `alarm_rules` (`projectId`);
CREATE INDEX `alarm_rules_severity_idx` ON `alarm_rules` (`severity`);
CREATE INDEX `alarm_rules_enabled_idx` ON `alarm_rules` (`isEnabled`);

-- 3.2 告警记录表
CREATE TABLE IF NOT EXISTS `alarm_records` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '工程ID',
  `ruleId` char(36) NOT NULL COMMENT '规则ID',
  `ruleName` varchar(100) NOT NULL COMMENT '规则名称(冗余)',
  `severity` enum('info','warning','error','critical') NOT NULL COMMENT '告警级别',
  `status` enum('active','acknowledged','resolved','suppressed') NOT NULL DEFAULT 'active' COMMENT '告警状态',
  `triggerValue` varchar(255) DEFAULT NULL COMMENT '触发值',
  `triggerCondition` varchar(500) DEFAULT NULL COMMENT '触发条件描述',
  `message` text NOT NULL COMMENT '告警消息',
  `details` json DEFAULT NULL COMMENT '详细信息',
  `occurredAt` datetime(6) NOT NULL COMMENT '发生时间',
  `acknowledgedAt` datetime(6) DEFAULT NULL COMMENT '确认时间',
  `acknowledgedBy` char(36) DEFAULT NULL COMMENT '确认人',
  `acknowledgeNote` text DEFAULT NULL COMMENT '确认备注',
  `resolvedAt` datetime(6) DEFAULT NULL COMMENT '解除时间',
  `resolvedBy` char(36) DEFAULT NULL COMMENT '解除人',
  `resolveNote` text DEFAULT NULL COMMENT '解除备注',
  `duration` int DEFAULT NULL COMMENT '持续时间(秒)',
  PRIMARY KEY (`id`),
  KEY `alarm_records_project_idx` (`projectId`),
  KEY `alarm_records_rule_idx` (`ruleId`),
  KEY `alarm_records_status_idx` (`status`),
  KEY `alarm_records_severity_idx` (`severity`),
  KEY `alarm_records_occurred_idx` (`occurredAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='告警记录表';

-- 3.3 告警通知配置表
CREATE TABLE IF NOT EXISTS `alarm_notifications` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '工程ID',
  `name` varchar(100) NOT NULL COMMENT '配置名称',
  `type` enum('email','sms','webhook','dingtalk','wechat','feishu') NOT NULL COMMENT '通知类型',
  `config` json NOT NULL COMMENT '通知配置',
  `template` text DEFAULT NULL COMMENT '消息模板',
  `severityFilter` json DEFAULT NULL COMMENT '级别过滤',
  `schedule` json DEFAULT NULL COMMENT '通知时间段配置',
  `isEnabled` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `alarm_notif_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='告警通知配置表';

-- ============================================
-- 第四部分：设计中心相关表
-- ============================================

-- 4.1 设计页面表
CREATE TABLE IF NOT EXISTS `design_pages` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '所属工程ID',
  `parentId` char(36) DEFAULT NULL COMMENT '父页面ID(文件夹结构)',
  `name` varchar(100) NOT NULL COMMENT '页面名称',
  `path` varchar(200) DEFAULT NULL COMMENT '路由路径',
  `type` enum('page','folder','dialog','template') NOT NULL DEFAULT 'page' COMMENT '类型',
  `icon` varchar(100) DEFAULT NULL COMMENT '菜单图标',
  `schemaVersion` varchar(20) DEFAULT '1.0.0' COMMENT 'DSL版本号',
  `schemaContent` json DEFAULT NULL COMMENT '页面DSL(Components Tree)',
  `pageConfig` json DEFAULT NULL COMMENT '页面配置(尺寸/背景/网格)',
  `variables` json DEFAULT NULL COMMENT '页面级变量定义',
  `dataSources` json DEFAULT NULL COMMENT '页面级数据源配置',
  `lifecycle` json DEFAULT NULL COMMENT '生命周期钩子',
  `isHome` tinyint(1) DEFAULT 0 COMMENT '是否为首页',
  `isPublished` tinyint(1) DEFAULT 0 COMMENT '是否已发布',
  `status` enum('draft','review','published','archived') DEFAULT 'draft' COMMENT '状态',
  `thumbnailUrl` varchar(500) DEFAULT NULL COMMENT '页面缩略图',
  `sortOrder` int DEFAULT 0 COMMENT '排序',
  `lockedBy` char(36) DEFAULT NULL COMMENT '当前锁定用户ID',
  `lockedAt` datetime(6) DEFAULT NULL COMMENT '锁定时间',
  `publishedAt` datetime(6) DEFAULT NULL COMMENT '最后发布时间',
  `publishedBy` char(36) DEFAULT NULL COMMENT '发布人',
  `createdBy` char(36) NOT NULL COMMENT '创建者ID',
  `updatedBy` char(36) DEFAULT NULL COMMENT '更新者ID',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `design_pages_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
  CONSTRAINT `design_pages_fk_parent` FOREIGN KEY (`parentId`) REFERENCES `design_pages` (`id`) ON DELETE CASCADE,
  CONSTRAINT `design_pages_fk_creator` FOREIGN KEY (`createdBy`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='设计页面表';

CREATE INDEX `design_pages_project_idx` ON `design_pages` (`projectId`);
CREATE INDEX `design_pages_parent_idx` ON `design_pages` (`parentId`);
CREATE INDEX `design_pages_type_idx` ON `design_pages` (`type`);
CREATE INDEX `design_pages_status_idx` ON `design_pages` (`status`);
CREATE INDEX `design_pages_locked_idx` ON `design_pages` (`lockedBy`);
CREATE INDEX `design_pages_sort_idx` ON `design_pages` (`projectId`, `parentId`, `sortOrder`);
CREATE UNIQUE INDEX `design_pages_project_path_uq` ON `design_pages` (`projectId`, `path`);

-- 4.2 资源文件夹表
CREATE TABLE IF NOT EXISTS `design_asset_folders` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '所属工程ID',
  `parentId` char(36) DEFAULT NULL COMMENT '父文件夹ID',
  `name` varchar(100) NOT NULL COMMENT '文件夹名称',
  `sortOrder` int DEFAULT 0 COMMENT '排序',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `asset_folders_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
  CONSTRAINT `asset_folders_fk_parent` FOREIGN KEY (`parentId`) REFERENCES `design_asset_folders` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='资源文件夹表';

CREATE UNIQUE INDEX `asset_folders_name_uq` ON `design_asset_folders` (`projectId`, `parentId`, `name`);

-- 4.3 资源文件表
CREATE TABLE IF NOT EXISTS `design_assets` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '所属工程ID',
  `folderId` char(36) DEFAULT NULL COMMENT '文件夹ID',
  `name` varchar(200) NOT NULL COMMENT '资源名称',
  `originalName` varchar(200) DEFAULT NULL COMMENT '原始文件名',
  `type` enum('image','svg','video','audio','model_3d','font','json','other') NOT NULL COMMENT '资源类型',
  `mimeType` varchar(100) DEFAULT NULL COMMENT 'MIME类型',
  `url` varchar(1000) NOT NULL COMMENT '资源访问路径',
  `thumbnailUrl` varchar(1000) DEFAULT NULL COMMENT '缩略图路径',
  `size` bigint DEFAULT 0 COMMENT '文件大小(字节)',
  `width` int DEFAULT NULL COMMENT '宽度(图片/视频)',
  `height` int DEFAULT NULL COMMENT '高度(图片/视频)',
  `duration` int DEFAULT NULL COMMENT '时长(视频/音频,秒)',
  `metadata` json DEFAULT NULL COMMENT '扩展元数据',
  `tags` json DEFAULT NULL COMMENT '标签数组',
  `usageCount` int DEFAULT 0 COMMENT '使用次数',
  `uploadedBy` char(36) NOT NULL COMMENT '上传者ID',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `assets_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
  CONSTRAINT `assets_fk_folder` FOREIGN KEY (`folderId`) REFERENCES `design_asset_folders` (`id`) ON DELETE SET NULL,
  CONSTRAINT `assets_fk_uploader` FOREIGN KEY (`uploadedBy`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='资源文件表';

CREATE INDEX `assets_project_type_idx` ON `design_assets` (`projectId`, `type`);
CREATE INDEX `assets_folder_idx` ON `design_assets` (`folderId`);

-- 4.4 页面历史版本表
CREATE TABLE IF NOT EXISTS `design_page_histories` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `pageId` char(36) NOT NULL COMMENT '关联页面ID',
  `version` varchar(50) NOT NULL COMMENT '版本号',
  `schemaVersion` varchar(20) DEFAULT '1.0.0' COMMENT 'DSL版本号',
  `schemaContent` json NOT NULL COMMENT '页面DSL快照',
  `pageConfig` json DEFAULT NULL COMMENT '页面配置快照',
  `variables` json DEFAULT NULL COMMENT '变量定义快照',
  `dataSources` json DEFAULT NULL COMMENT '数据源配置快照',
  `description` varchar(500) DEFAULT NULL COMMENT '版本描述',
  `changeType` enum('manual','auto','publish','rollback') NOT NULL DEFAULT 'manual' COMMENT '变更类型',
  `changeSummary` json DEFAULT NULL COMMENT '变更摘要',
  `createdBy` char(36) NOT NULL COMMENT '操作人',
  `createdAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `page_histories_page_idx` (`pageId`),
  KEY `page_histories_time_idx` (`createdAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='页面历史版本表';

-- 4.5 工程全局配置表
CREATE TABLE IF NOT EXISTS `design_project_settings` (
  `projectId` char(36) NOT NULL COMMENT '工程ID',
  `schemaVersion` varchar(20) DEFAULT '1.0.0' COMMENT 'DSL版本号',
  `globalVariables` json DEFAULT NULL COMMENT '全局变量定义',
  `globalStyles` json DEFAULT NULL COMMENT '全局样式/主题配置',
  `globalScripts` mediumtext COMMENT '全局JavaScript函数库',
  `globalDataSources` json DEFAULT NULL COMMENT '全局数据源配置',
  `componentMappings` json DEFAULT NULL COMMENT '自定义组件映射',
  `i18n` json DEFAULT NULL COMMENT '国际化配置',
  `permissions` json DEFAULT NULL COMMENT '全局权限配置',
  `buildConfig` json DEFAULT NULL COMMENT '构建配置',
  `runtimeConfig` json DEFAULT NULL COMMENT '运行时配置',
  `updatedBy` char(36) DEFAULT NULL COMMENT '更新者ID',
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`projectId`),
  CONSTRAINT `project_settings_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='工程全局配置表';


-- 4.6 复合组件表
CREATE TABLE IF NOT EXISTS `design_custom_components` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '所属工程ID',
  `name` varchar(100) NOT NULL COMMENT '组件名称',
  `displayName` varchar(200) DEFAULT NULL COMMENT '显示名称',
  `category` varchar(50) DEFAULT NULL COMMENT '分类',
  `description` text COMMENT '组件描述',
  `icon` varchar(100) DEFAULT NULL COMMENT '组件图标',
  `version` varchar(20) DEFAULT '1.0.0' COMMENT '版本号',
  `schemaContent` json NOT NULL COMMENT '组件DSL',
  `propsDefinition` json DEFAULT NULL COMMENT '属性定义(JSON Schema)',
  `eventsDefinition` json DEFAULT NULL COMMENT '事件定义',
  `slotsDefinition` json DEFAULT NULL COMMENT '插槽定义',
  `defaultProps` json DEFAULT NULL COMMENT '默认属性值',
  `previewConfig` json DEFAULT NULL COMMENT '预览配置',
  `thumbnailUrl` varchar(500) DEFAULT NULL COMMENT '组件缩略图',
  `isPublic` tinyint(1) DEFAULT 0 COMMENT '是否公开(租户内共享)',
  `usageCount` int DEFAULT 0 COMMENT '使用次数',
  `createdBy` char(36) NOT NULL COMMENT '创建者ID',
  `updatedBy` char(36) DEFAULT NULL COMMENT '更新者ID',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `custom_comp_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
  CONSTRAINT `custom_comp_fk_creator` FOREIGN KEY (`createdBy`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='复合组件表';

CREATE UNIQUE INDEX `custom_comp_name_uq` ON `design_custom_components` (`projectId`, `name`);
CREATE INDEX `custom_comp_category_idx` ON `design_custom_components` (`projectId`, `category`);

-- 4.7 复合组件版本历史表
CREATE TABLE IF NOT EXISTS `design_custom_component_histories` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `componentId` char(36) NOT NULL COMMENT '组件ID',
  `version` varchar(20) NOT NULL COMMENT '版本号',
  `schemaContent` json NOT NULL COMMENT '组件DSL快照',
  `propsDefinition` json DEFAULT NULL COMMENT '属性定义快照',
  `changelog` text COMMENT '更新日志',
  `createdBy` char(36) NOT NULL COMMENT '操作人',
  `createdAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `comp_histories_comp_idx` (`componentId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='复合组件版本历史表';

-- 4.8 工程级数据源模板表
CREATE TABLE IF NOT EXISTS `design_datasources` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '所属工程ID',
  `name` varchar(100) NOT NULL COMMENT '数据源名称',
  `displayName` varchar(200) DEFAULT NULL COMMENT '显示名称',
  `type` enum('dataCenter','http','websocket','static','computed') NOT NULL COMMENT '数据源类型',
  `config` json NOT NULL COMMENT '数据源配置',
  `transformer` text DEFAULT NULL COMMENT '数据转换脚本',
  `errorHandler` text DEFAULT NULL COMMENT '错误处理脚本',
  `description` text COMMENT '描述',
  `isGlobal` tinyint(1) DEFAULT 0 COMMENT '是否全局可用',
  `createdBy` char(36) NOT NULL COMMENT '创建者ID',
  `updatedBy` char(36) DEFAULT NULL COMMENT '更新者ID',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `datasources_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
  CONSTRAINT `datasources_fk_creator` FOREIGN KEY (`createdBy`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='工程级数据源模板表';

CREATE UNIQUE INDEX `datasources_name_uq` ON `design_datasources` (`projectId`, `name`);

-- 4.9 运行时角色表
CREATE TABLE IF NOT EXISTS `design_roles` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '所属工程ID',
  `name` varchar(50) NOT NULL COMMENT '角色标识',
  `displayName` varchar(100) DEFAULT NULL COMMENT '显示名称',
  `description` varchar(255) DEFAULT NULL COMMENT '角色描述',
  `permissions` json DEFAULT NULL COMMENT '权限配置',
  `isDefault` tinyint(1) DEFAULT 0 COMMENT '是否默认角色',
  `sortOrder` int DEFAULT 0 COMMENT '排序',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `roles_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='运行时角色表';

CREATE UNIQUE INDEX `roles_name_uq` ON `design_roles` (`projectId`, `name`);

-- 4.10 组件权限关系表
CREATE TABLE IF NOT EXISTS `design_permissions` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `pageId` char(36) NOT NULL COMMENT '页面ID',
  `componentId` varchar(100) NOT NULL COMMENT 'DSL内组件ID',
  `visibleFor` json DEFAULT NULL COMMENT '可见角色数组',
  `editableFor` json DEFAULT NULL COMMENT '可编辑角色数组',
  `disabledFor` json DEFAULT NULL COMMENT '禁用角色数组',
  `customRules` json DEFAULT NULL COMMENT '自定义权限规则',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `permissions_fk_page` FOREIGN KEY (`pageId`) REFERENCES `design_pages` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='组件权限关系表';

CREATE UNIQUE INDEX `permissions_component_uq` ON `design_permissions` (`pageId`, `componentId`);

-- ============================================
-- 第五部分：发布部署相关表
-- ============================================

-- 5.1 发布版本表
CREATE TABLE IF NOT EXISTS `deployments` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '工程ID',
  `tenantId` char(36) NOT NULL COMMENT '租户ID（冗余，便于查询）',
  `version` varchar(50) NOT NULL COMMENT '版本号（语义化版本）',
  `name` varchar(200) DEFAULT NULL COMMENT '版本名称（可选描述）',
  `description` text COMMENT '版本描述/发布说明',
  `type` enum('development','staging','production') NOT NULL DEFAULT 'development' COMMENT '部署类型',
  `mode` enum('DEV','RELEASE') NOT NULL DEFAULT 'RELEASE' COMMENT '运行模式：DEV=连接开发库实时同步，RELEASE=使用本地库',
  `status` enum('pending','building','success','failed') NOT NULL DEFAULT 'pending' COMMENT '构建状态',
  `buildConfig` json DEFAULT NULL COMMENT '构建配置',
  `buildLog` json DEFAULT NULL COMMENT '构建日志（数组）',
  `artifactUrl` varchar(1000) DEFAULT NULL COMMENT '构建产物URL（IFP包）',
  `artifactHash` varchar(64) DEFAULT NULL COMMENT '产物SHA256哈希',
  `artifactSize` bigint DEFAULT NULL COMMENT '产物大小（字节）',
  `snapshotUrl` varchar(1000) DEFAULT NULL COMMENT '快照文件URL',
  `snapshotHash` varchar(64) DEFAULT NULL COMMENT '快照SHA256哈希',
  `manifest` json DEFAULT NULL COMMENT '清单信息：dataRequirements, capabilities, security等',
  `pageCount` int DEFAULT 0 COMMENT '页面数量',
  `componentCount` int DEFAULT 0 COMMENT '组件数量',
  `datapointCount` int DEFAULT 0 COMMENT '数据点数量',
  `startedAt` datetime(6) DEFAULT NULL COMMENT '构建开始时间',
  `completedAt` datetime(6) DEFAULT NULL COMMENT '构建完成时间',
  `errorMessage` text DEFAULT NULL COMMENT '错误信息',
  `deployedBy` char(36) DEFAULT NULL COMMENT '发布者ID',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  `deletedAt` datetime(6) DEFAULT NULL COMMENT '软删除时间',
  PRIMARY KEY (`id`),
  CONSTRAINT `deployments_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
  CONSTRAINT `deployments_fk_tenant` FOREIGN KEY (`tenantId`) REFERENCES `tenants` (`id`) ON DELETE CASCADE,
  CONSTRAINT `deployments_fk_deployer` FOREIGN KEY (`deployedBy`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='发布版本表';

CREATE INDEX `deployments_project_idx` ON `deployments` (`projectId`);
CREATE INDEX `deployments_tenant_idx` ON `deployments` (`tenantId`);
CREATE INDEX `deployments_status_idx` ON `deployments` (`status`);
CREATE INDEX `deployments_type_idx` ON `deployments` (`type`);
CREATE INDEX `deployments_mode_idx` ON `deployments` (`mode`);
CREATE INDEX `deployments_created_at_idx` ON `deployments` (`createdAt`);
CREATE UNIQUE INDEX `deployments_version_uq` ON `deployments` (`projectId`, `version`);

-- 5.2 发布页面快照表
CREATE TABLE IF NOT EXISTS `deployment_pages` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `deploymentId` char(36) NOT NULL COMMENT '发布ID',
  `pageId` char(36) NOT NULL COMMENT '原页面ID',
  `path` varchar(200) NOT NULL COMMENT '路由路径',
  `schemaContent` json NOT NULL COMMENT '页面DSL快照',
  `pageConfig` json DEFAULT NULL COMMENT '页面配置快照',
  PRIMARY KEY (`id`),
  CONSTRAINT `deploy_pages_fk_deploy` FOREIGN KEY (`deploymentId`) REFERENCES `deployments` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='发布页面快照表';

CREATE INDEX `deploy_pages_deploy_idx` ON `deployment_pages` (`deploymentId`);

-- 5.3 运行时节点表
CREATE TABLE IF NOT EXISTS `nodes` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `tenantId` char(36) NOT NULL COMMENT '所属租户ID',
  `name` varchar(100) NOT NULL COMMENT '节点名称',
  `description` varchar(500) DEFAULT NULL COMMENT '节点描述',
  `agentVersion` varchar(20) DEFAULT NULL COMMENT 'NodeAgent版本',
  `status` enum('online','offline','error') NOT NULL DEFAULT 'offline' COMMENT '节点状态',
  `currentProjectId` char(36) DEFAULT NULL COMMENT '当前运行的工程ID',
  `currentVersion` varchar(50) DEFAULT NULL COMMENT '当前运行的版本号',
  `currentDeploymentId` char(36) DEFAULT NULL COMMENT '当前部署记录ID',
  `ipAddress` varchar(45) DEFAULT NULL COMMENT '节点IP地址',
  `port` int DEFAULT 8080 COMMENT 'RuntimeEngine运行端口',
  `lastHeartbeatAt` datetime(6) DEFAULT NULL COMMENT '最后心跳时间',
  `lastErrorMessage` text DEFAULT NULL COMMENT '最后错误信息',
  `lastErrorAt` datetime(6) DEFAULT NULL COMMENT '最后错误时间',
  `metrics` json DEFAULT NULL COMMENT '运行指标(cpu, memory, uptime等)',
  `config` json DEFAULT NULL COMMENT '节点配置(标签, 分组等)',
  `registrationToken` varchar(64) DEFAULT NULL COMMENT '注册令牌',
  `approvalStatus` enum('pending','approved','rejected') NOT NULL DEFAULT 'pending' COMMENT '审批状态',
  `approvedAt` datetime(6) DEFAULT NULL COMMENT '审批时间',
  `approvedBy` char(36) DEFAULT NULL COMMENT '审批人ID',
  `registeredBy` char(36) DEFAULT NULL COMMENT '注册申请人ID',
  `mode` enum('online','offline') NOT NULL DEFAULT 'online' COMMENT '节点模式(在线/离线)',
  `createdBy` char(36) DEFAULT NULL COMMENT '创建者ID',
  `updatedBy` char(36) DEFAULT NULL COMMENT '更新者ID',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  `deletedAt` datetime(6) DEFAULT NULL COMMENT '软删除时间',
  PRIMARY KEY (`id`),
  CONSTRAINT `nodes_fk_tenant` FOREIGN KEY (`tenantId`) REFERENCES `tenants` (`id`) ON DELETE CASCADE,
  CONSTRAINT `nodes_fk_project` FOREIGN KEY (`currentProjectId`) REFERENCES `projects` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='运行时节点表';

CREATE INDEX `nodes_tenant_idx` ON `nodes` (`tenantId`);
CREATE INDEX `nodes_status_idx` ON `nodes` (`status`);
CREATE INDEX `nodes_heartbeat_idx` ON `nodes` (`lastHeartbeatAt`);
CREATE UNIQUE INDEX `nodes_tenant_name_uq` ON `nodes` (`tenantId`, `name`);

-- 5.4 节点部署关系表
CREATE TABLE IF NOT EXISTS `node_deployments` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `nodeId` char(36) NOT NULL COMMENT '节点ID',
  `deploymentId` char(36) NOT NULL COMMENT '发布版本ID',
  `projectId` char(36) NOT NULL COMMENT '工程ID(冗余)',
  `version` varchar(50) NOT NULL COMMENT '版本号(冗余)',
  `mode` enum('DEV', 'RELEASE') NOT NULL DEFAULT 'RELEASE' COMMENT '运行模式',
  `status` enum('pending','deploying','running','stopped','error','rollback') NOT NULL DEFAULT 'pending' COMMENT '部署状态',
  `runtimeConfig` json DEFAULT NULL COMMENT '运行时配置(port, uiTarget等)',
  `deployedAt` datetime(6) DEFAULT NULL COMMENT '部署完成时间',
  `startedAt` datetime(6) DEFAULT NULL COMMENT '启动时间',
  `stoppedAt` datetime(6) DEFAULT NULL COMMENT '停止时间',
  `deployedBy` char(36) DEFAULT NULL COMMENT '部署操作者ID',
  `errorMessage` text DEFAULT NULL COMMENT '错误信息',
  `errorStack` text DEFAULT NULL COMMENT '错误堆栈',
  `deployLog` json DEFAULT NULL COMMENT '部署日志(数组)',
  `runtimeMetrics` json DEFAULT NULL COMMENT '运行时指标(onlineUsers, concurrentUsers等)',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  `deletedAt` datetime(6) DEFAULT NULL COMMENT '软删除时间',
  PRIMARY KEY (`id`),
  CONSTRAINT `node_deploy_fk_node` FOREIGN KEY (`nodeId`) REFERENCES `nodes` (`id`) ON DELETE CASCADE,
  CONSTRAINT `node_deploy_fk_deployment` FOREIGN KEY (`deploymentId`) REFERENCES `deployments` (`id`) ON DELETE CASCADE,
  CONSTRAINT `node_deploy_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
  CONSTRAINT `node_deploy_fk_deployer` FOREIGN KEY (`deployedBy`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='节点部署关系表';

CREATE INDEX `node_deploy_node_idx` ON `node_deployments` (`nodeId`);
CREATE INDEX `node_deploy_deployment_idx` ON `node_deployments` (`deploymentId`);
CREATE INDEX `node_deploy_project_idx` ON `node_deployments` (`projectId`);
CREATE INDEX `node_deploy_status_idx` ON `node_deployments` (`status`);
CREATE INDEX `node_deploy_mode_idx` ON `node_deployments` (`mode`);
CREATE UNIQUE INDEX `node_deploy_active_uq` ON `node_deployments` (`nodeId`, `projectId`) COMMENT '同一节点同一工程只能有一个活跃部署';

-- ============================================
-- 第六部分：运行时相关表
-- ============================================

-- 6.1 运行时用户表(发布后的终端用户)
CREATE TABLE IF NOT EXISTS `runtime_users` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '工程ID',
  `username` varchar(50) NOT NULL COMMENT '用户名',
  `password` varchar(255) NOT NULL COMMENT '密码哈希',
  `email` varchar(255) DEFAULT NULL COMMENT '邮箱',
  `phone` varchar(20) DEFAULT NULL COMMENT '手机号',
  `fullName` varchar(100) DEFAULT NULL COMMENT '姓名',
  `avatar` varchar(500) DEFAULT NULL COMMENT '头像',
  `roleId` char(36) DEFAULT NULL COMMENT '角色ID',
  `status` enum('active','inactive','suspended') NOT NULL DEFAULT 'active' COMMENT '状态',
  `metadata` json DEFAULT NULL COMMENT '扩展数据',
  `lastLoginAt` datetime(6) DEFAULT NULL COMMENT '最后登录时间',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `runtime_users_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
  CONSTRAINT `runtime_users_fk_role` FOREIGN KEY (`roleId`) REFERENCES `design_roles` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='运行时用户表';

CREATE UNIQUE INDEX `runtime_users_username_uq` ON `runtime_users` (`projectId`, `username`);
CREATE INDEX `runtime_users_role_idx` ON `runtime_users` (`roleId`);

-- 6.2 运行时会话表
CREATE TABLE IF NOT EXISTS `runtime_sessions` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '工程ID',
  `userId` char(36) NOT NULL COMMENT '用户ID',
  `token` varchar(500) NOT NULL COMMENT '会话Token',
  `deviceInfo` json DEFAULT NULL COMMENT '设备信息',
  `ip` varchar(45) DEFAULT NULL COMMENT 'IP地址',
  `expiresAt` datetime(6) NOT NULL COMMENT '过期时间',
  `createdAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `runtime_sessions_user_idx` (`userId`),
  KEY `runtime_sessions_token_idx` (`token`(255)),
  KEY `runtime_sessions_expires_idx` (`expiresAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='运行时会话表';

-- 6.3 运行时操作日志表
CREATE TABLE IF NOT EXISTS `runtime_logs` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '工程ID',
  `userId` char(36) DEFAULT NULL COMMENT '用户ID',
  `sessionId` char(36) DEFAULT NULL COMMENT '会话ID',
  `action` varchar(100) NOT NULL COMMENT '操作类型',
  `pageId` char(36) DEFAULT NULL COMMENT '页面ID',
  `componentId` varchar(100) DEFAULT NULL COMMENT '组件ID',
  `details` json DEFAULT NULL COMMENT '详细信息',
  `ip` varchar(45) DEFAULT NULL COMMENT 'IP地址',
  `createdAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `runtime_logs_project_idx` (`projectId`),
  KEY `runtime_logs_user_idx` (`userId`),
  KEY `runtime_logs_time_idx` (`createdAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='运行时操作日志表';

-- ============================================
-- 第七部分：定时任务相关表
-- ============================================

-- 7.1 定时任务表
CREATE TABLE IF NOT EXISTS `scheduled_tasks` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `projectId` char(36) NOT NULL COMMENT '工程ID',
  `name` varchar(100) NOT NULL COMMENT '任务名称',
  `description` text COMMENT '任务描述',
  `type` enum('query','script','notification','backup') NOT NULL COMMENT '任务类型',
  `config` json NOT NULL COMMENT '任务配置',
  `schedule` varchar(100) NOT NULL COMMENT 'Cron表达式',
  `timezone` varchar(50) DEFAULT 'Asia/Shanghai' COMMENT '时区',
  `isEnabled` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  `lastRunAt` datetime(6) DEFAULT NULL COMMENT '最后执行时间',
  `lastRunStatus` enum('success','error','timeout') DEFAULT NULL COMMENT '最后执行状态',
  `nextRunAt` datetime(6) DEFAULT NULL COMMENT '下次执行时间',
  `createdBy` char(36) NOT NULL COMMENT '创建者ID',
  `createdAt` datetime(6) NOT NULL,
  `updatedAt` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `tasks_fk_project` FOREIGN KEY (`projectId`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
  CONSTRAINT `tasks_fk_creator` FOREIGN KEY (`createdBy`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='定时任务表';

CREATE INDEX `tasks_project_idx` ON `scheduled_tasks` (`projectId`);
CREATE INDEX `tasks_enabled_idx` ON `scheduled_tasks` (`isEnabled`);
CREATE INDEX `tasks_next_run_idx` ON `scheduled_tasks` (`nextRunAt`);

-- 7.2 任务执行记录表
CREATE TABLE IF NOT EXISTS `scheduled_task_logs` (
  `id` char(36) NOT NULL DEFAULT (uuid()),
  `taskId` char(36) NOT NULL COMMENT '任务ID',
  `status` enum('running','success','error','timeout','cancelled') NOT NULL COMMENT '执行状态',
  `startedAt` datetime(6) NOT NULL COMMENT '开始时间',
  `completedAt` datetime(6) DEFAULT NULL COMMENT '完成时间',
  `duration` int DEFAULT NULL COMMENT '执行时长(ms)',
  `result` json DEFAULT NULL COMMENT '执行结果',
  `errorMessage` text COMMENT '错误信息',
  PRIMARY KEY (`id`),
  KEY `task_logs_task_idx` (`taskId`),
  KEY `task_logs_time_idx` (`startedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='任务执行记录表';
