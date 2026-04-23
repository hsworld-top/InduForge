/**
 * 发布服务 - 工程发布流水线
 * @description 处理工程发布：验证 → 编译清单 → 打包 IFP → 上传
 */
const path = require("path");
const fs = require("fs-extra");
const archiver = require("archiver");
const crypto = require("crypto");
const { dataDomainClient } = require("./dataDomainClient");
const {
  Project,
  DesignPage,
  Deployment,
  NodeDeployment,
  User,
  ProjectRuntimeUser,
  ProjectRole,
  ProjectUserRoleBinding,
  ProjectRoleGrant,
} = require("../models");
const AppError = require("../utils/AppError");
const ErrorCodes = require("../constants/errorCodes");

// 制品存储目录
const ARTIFACTS_DIR = process.env.ARTIFACTS_DIR || path.join(__dirname, "../../artifacts");
const RUNTIME_SECURITY_FILE_NAME = "runtime-security.json";

const asArray = (value) => (Array.isArray(value) ? value : []);

const normalizeArtifactProtocols = (protocols = {}) => ({
  kafka: asArray(protocols?.kafka),
  http: asArray(protocols?.http),
  websocket: asArray(protocols?.websocket),
  redis: asArray(protocols?.redis),
});

const resolveStoredArtifactFileName = (deployment) => {
  const explicitFileName = deployment?.buildConfig?.artifactFileName;
  if (explicitFileName) {
    return path.basename(String(explicitFileName));
  }

  const artifactUrl = String(deployment?.artifactUrl || "");
  const urlFileName = path.basename(artifactUrl);
  if (urlFileName && urlFileName !== "download" && /\.ifp$/i.test(urlFileName)) {
    return urlFileName;
  }

  return "";
};

const buildLegacyArtifactFilePrefix = (deployment) => {
  const projectId = String(deployment?.projectId || "").trim();
  const version = String(deployment?.version || "").trim();
  if (!projectId || !version) {
    return "";
  }
  return `${projectId}_v${version}_`;
};

const resolveArtifactFileCandidates = async (deployment) => {
  const directFileName = resolveStoredArtifactFileName(deployment);
  if (directFileName) {
    return [path.join(ARTIFACTS_DIR, directFileName)];
  }

  const legacyPrefix = buildLegacyArtifactFilePrefix(deployment);
  if (!legacyPrefix) {
    return [];
  }

  const artifactNames = await fs.readdir(ARTIFACTS_DIR).catch(() => []);
  return artifactNames
    .filter((name) => name.startsWith(legacyPrefix) && /\.ifp$/i.test(name))
    .sort((left, right) => right.localeCompare(left))
    .map((name) => path.join(ARTIFACTS_DIR, name));
};

const buildRelationalConfigsFromConnections = (connections = []) =>
  asArray(connections)
    .filter((connection) => connection?.type === "relational")
    .map((connection) => {
      const config = connection?.config || {};
      return {
        connectionId: connection.id,
        dbType: config.dbType || "postgresql",
        host: config.host || "",
        port: config.port ?? 5432,
        database: config.database || "",
        username: config.username || "",
        password: config.password || "",
        schema: config.schema || null,
        charset: config.charset || null,
        timezone: config.timezone || null,
        ssl: Boolean(config.ssl),
        sslConfig: config.sslConfig || {},
      };
    });

const buildMqttConfigsFromArtifact = (connections = []) =>
  asArray(connections).map((connection) => ({
    connectionId: connection.id,
    brokerUrl: connection.brokerUrl || "",
    protocol: connection.protocol || "mqtt",
    port: connection.port ?? 1883,
    clientId: connection.clientId ?? null,
    username: connection.username ?? null,
    password: connection.password ?? null,
    keepalive: connection.keepalive ?? 60,
    cleanSession: Boolean(connection.cleanSession),
    qos: connection.qos ?? 0,
    reconnectPeriod: connection.reconnectPeriod ?? 1000,
    connectTimeout: connection.connectTimeout ?? 30000,
    will: connection.will || {},
    sslConfig: connection.sslConfig || {},
  }));

const collectPlainRows = async (model, where) =>
  model.findAll({
    where,
    order: [["createdAt", "ASC"]],
    raw: true,
  });

const buildManifestWithRuntimeSecurity = (manifest, runtimeSecuritySnapshot) => {
  const snapshot = runtimeSecuritySnapshot || {
    version: new Date().toISOString(),
    users: [],
    roles: [],
    bindings: [],
    grants: [],
  };

  return {
    ...manifest,
    security: {
      ...(manifest.security || {}),
      runtimeSecuritySnapshot: {
        included: true,
        fileName: RUNTIME_SECURITY_FILE_NAME,
        version: snapshot.version,
      },
    },
  };
};

/**
 * 发布服务类
 */
class PublishService {
  constructor(options = {}) {
    this.dataDomainClient = options.dataDomainClient || dataDomainClient;
    // 确保制品目录存在
    fs.ensureDirSync(ARTIFACTS_DIR);
  }

  /**
   * 发布工程
   * @param {string} projectId - 工程ID
   * @param {Object} options - 发布选项
   * @param {string} options.version - 版本号
   * @param {string} options.name - 版本名称
   * @param {string} options.description - 版本描述
   * @param {string} options.type - 部署类型
   * @param {string} options.deployedBy - 发布者ID
   * @returns {Promise<Object>} 发布结果
   */
  async publish(projectId, options) {
    const {
      version,
      name,
      description,
      type = "development",
      deployedBy,
      authorization,
    } = options;

    // 获取工程信息
    const project = await Project.findByPk(projectId);
    if (!project) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        message: `工程 ${projectId} 不存在`,
      });
    }

    // 检查版本号是否已存在（包含软删除记录，避免唯一索引冲突）。
    const existingVersion = await Deployment.findOne({
      where: { projectId, version },
      paranoid: false,
    });
    if (existingVersion) {
      if (existingVersion.deletedAt) {
        // 软删除记录在唯一索引中仍占位。若无部署引用则物理删除以释放版本号。
        const referencedCount = await NodeDeployment.count({
          where: { deploymentId: existingVersion.id },
        });
        if (referencedCount > 0) {
          throw new AppError(ErrorCodes.RESOURCE_ALREADY_EXISTS, 409, {
            message: `版本 ${version} 已存在且被部署引用，无法复用`,
          });
        }
        await existingVersion.destroy({ force: true });
      } else {
        throw new AppError(ErrorCodes.RESOURCE_ALREADY_EXISTS, 409, {
          message: `版本 ${version} 已存在`,
        });
      }
    }

    // 创建发布记录
    const deploymentId = crypto.randomUUID();
    const deployment = await Deployment.create({
      id: deploymentId,
      projectId,
      tenantId: project.tenantId,
      version,
      name: name || `v${version}`,
      description,
      type,
      status: "building",
      deployedBy,
      startedAt: new Date(),
      buildLog: [{ time: new Date().toISOString(), message: "开始构建..." }],
    });

    try {
      // 1. 验证工程
      await this.addBuildLog(deploymentId, "验证工程配置...");
      const validation = await this.validateProject(projectId);
      if (!validation.valid) {
        throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
          message: `工程验证失败: ${validation.errors.join(", ")}`,
        });
      }

      // 2. 收集工程数据
      await this.addBuildLog(deploymentId, "收集工程数据...");
      const projectData = await this.collectProjectData(projectId, authorization);

      // 3. 编译清单
      await this.addBuildLog(deploymentId, "生成清单文件...");
      const manifest = await this.compileManifest(project, projectData, version);
      const manifestWithRuntimeSecurity = buildManifestWithRuntimeSecurity(
        manifest,
        projectData.runtimeSecuritySnapshot,
      );

      // 4. 打包 IFP
      await this.addBuildLog(deploymentId, "打包 IFP 文件...");
      const { ifpPath, hash, size } = await this.bundleIFP(
        deploymentId,
        projectId,
        manifestWithRuntimeSecurity,
        projectData
      );

      // 5. 生成受控下载 URL（统一通过发布下载接口）。
      const artifactUrl = `/api/v1/publish/deployment/${deploymentId}/download`;

      // 6. 更新发布记录
      await deployment.update({
        status: "success",
        artifactUrl,
        artifactHash: hash,
        artifactSize: size,
        buildConfig: {
          ...(deployment.buildConfig || {}),
          artifactFileName: path.basename(ifpPath),
        },
        manifest: manifestWithRuntimeSecurity,
        pageCount: projectData.pages.length,
        componentCount: this.countComponents(projectData.pages),
        datapointCount: projectData.dataPoints.length,
        completedAt: new Date(),
      });

      await this.addBuildLog(deploymentId, "发布成功！");

      return {
        id: deploymentId,
        version,
        artifactUrl,
        artifactHash: hash,
        artifactSize: size,
        pageCount: projectData.pages.length,
        componentCount: this.countComponents(projectData.pages),
      };
    } catch (error) {
      // 记录错误
      await deployment.update({
        status: "failed",
        errorMessage: error.message,
        completedAt: new Date(),
      });
      await this.addBuildLog(deploymentId, `发布失败: ${error.message}`);
      throw error;
    }
  }

  /**
   * 添加构建日志
   */
  async addBuildLog(deploymentId, message) {
    const deployment = await Deployment.findByPk(deploymentId);
    if (deployment) {
      const logs = deployment.buildLog || [];
      logs.push({ time: new Date().toISOString(), message });
      await deployment.update({ buildLog: logs });
    }
  }

  /**
   * 验证工程
   */
  async validateProject(projectId) {
    const errors = [];

    // 检查页面
    const pages = await DesignPage.findAll({
      where: { projectId, type: "page" },
    });
    if (pages.length === 0) {
      errors.push("工程没有任何页面");
    }

    // 检查是否有首页
    const homePage = pages.find((p) => p.isHome);
    if (!homePage && pages.length > 0) {
      // 警告但不阻止发布
      console.warn("工程没有设置首页，将使用第一个页面作为首页");
    }

    // 检查数据绑定（可选，仅警告）
    // ...

    return {
      valid: errors.length === 0,
      errors,
      warnings: [],
    };
  }

  /**
   * 收集工程数据
   */
  async collectProjectData(projectId, authorization) {
    // 获取所有页面
    const pages = await DesignPage.findAll({
      where: { projectId },
      order: [
        ["type", "ASC"],
        ["sortOrder", "ASC"],
      ],
    });
    // 发布包需要同时携带数据域制品与运行态安全快照，节点侧才能完成离线认证与鉴权。
    const [artifact, runtimeSecuritySnapshot] = await Promise.all([
      this.dataDomainClient.getProjectArtifact(projectId, authorization),
      this.collectRuntimeSecuritySnapshot(projectId),
    ]);
    const protocols = normalizeArtifactProtocols(artifact?.protocols);

    return {
      pages,
      artifactVersion: artifact?.version || "1.0",
      artifactGeneratedAt: artifact?.generatedAt || new Date().toISOString(),
      connections: asArray(artifact?.connections),
      relationalConfigs: buildRelationalConfigsFromConnections(artifact?.connections),
      queries: asArray(artifact?.queries),
      mqttConfigs: buildMqttConfigsFromArtifact(artifact?.mqtt?.connections),
      mqttSubscriptions: asArray(artifact?.mqtt?.subscriptions),
      mqttTagGroups: asArray(artifact?.mqtt?.tagGroups),
      mqttTags: asArray(artifact?.mqtt?.tags),
      dataPoints: asArray(artifact?.datapoints),
      protocols,
      runtimeSecuritySnapshot,
    };
  }

  /**
   * 收集工程运行态安全快照
   * @description 只导出当前工程自己的运行态用户、角色、绑定和授权记录，保持为普通对象数组，避免把 Sequelize 实例直接塞进发布包。
   */
  async collectRuntimeSecuritySnapshot(projectId) {
    const [users, roles, bindings, grants] = await Promise.all([
      collectPlainRows(ProjectRuntimeUser, { projectId }),
      collectPlainRows(ProjectRole, { projectId }),
      collectPlainRows(ProjectUserRoleBinding, { projectId }),
      collectPlainRows(ProjectRoleGrant, { projectId }),
    ]);

    return {
      version: new Date().toISOString(),
      users,
      roles,
      bindings,
      grants,
    };
  }

  /**
   * 编译清单
   */
  async compileManifest(project, projectData, version) {
    // 分析数据需求
    const dataRequirements = {
      connections: projectData.connections.map((c) => ({
        id: c.id,
        name: c.name,
        type: c.type,
      })),
      dataPoints: projectData.dataPoints.map((dp) => ({
        path: dp.path,
        sourceType: dp.sourceType,
        dataType: dp.dataType,
      })),
    };

    // 分析能力需求
    const capabilities = [];
    if (projectData.connections.some((c) => c.type === "mqtt")) {
      capabilities.push("mqtt");
    }
    if (projectData.connections.some((c) => c.type === "relational")) {
      capabilities.push("database");
    }
    if (projectData.queries.length > 0) {
      capabilities.push("query");
    }

    return {
      name: project.name,
      code: project.code,
      version,
      projectId: project.id,
      tenantId: project.tenantId,
      buildTime: new Date().toISOString(),
      schemaVersion: "1.0.0",
      entryConfig: project.entryConfig || {},
      dataRequirements,
      capabilities,
      security: {
        requireAuth: false, // 可配置
      },
    };
  }

  /**
   * 打包 IFP
   */
  async bundleIFP(deploymentId, projectId, manifest, projectData) {
    const fileName = `${projectId}_v${manifest.version}_${Date.now()}.ifp`;
    const ifpPath = path.join(ARTIFACTS_DIR, fileName);
    const runtimeSecuritySnapshot = projectData.runtimeSecuritySnapshot || {
      version: new Date().toISOString(),
      users: [],
      roles: [],
      bindings: [],
      grants: [],
    };
    const manifestWithSecurity = buildManifestWithRuntimeSecurity(
      manifest,
      runtimeSecuritySnapshot,
    );

    return new Promise((resolve, reject) => {
      const output = fs.createWriteStream(ifpPath);
      const archive = archiver("zip", { zlib: { level: 9 } });

      output.on("close", async () => {
        // 计算哈希
        const hash = await this.calculateFileHash(ifpPath);
        const stats = await fs.stat(ifpPath);

        resolve({
          ifpPath,
          hash,
          size: stats.size,
        });
      });
      output.on("error", (err) => {
        reject(err);
      });

      archive.on("error", (err) => {
        reject(err);
      });

      archive.pipe(output);

      // 添加 manifest.json
      archive.append(JSON.stringify(manifestWithSecurity, null, 2), { name: "manifest.json" });

      // 添加 project.json（页面和组件 Schema）
      const projectJson = {
        pages: projectData.pages.map((p) => ({
          id: p.id,
          name: p.name,
          path: p.path,
          type: p.type,
          parentId: p.parentId,
          isHome: p.isHome,
          schemaVersion: p.schemaVersion,
          schemaContent: p.schemaContent,
          pageConfig: p.pageConfig,
          variables: p.variables,
          dataSources: p.dataSources,
          lifecycle: p.lifecycle,
          sortOrder: p.sortOrder,
        })),
      };
      archive.append(JSON.stringify(projectJson, null, 2), { name: "project.json" });

      // 添加 datacenter.json（数据配置）
      const datacenterJson = {
        version: projectData.artifactVersion || "1.0",
        projectId,
        generatedAt: projectData.artifactGeneratedAt || new Date().toISOString(),
        connections: projectData.connections.map((c) => ({
          id: c.id,
          name: c.name,
          type: c.type,
          status: c.status,
          config: c.config,
        })),
        relationalConfigs: projectData.relationalConfigs.map((config) => ({ ...config })),
        mqttConfigs: projectData.mqttConfigs.map((config) => ({ ...config })),
        mqttSubscriptions: projectData.mqttSubscriptions.map((subscription) => ({ ...subscription })),
        mqttTagGroups: projectData.mqttTagGroups.map((group) => ({ ...group })),
        mqttTags: projectData.mqttTags.map((tag) => ({ ...tag })),
        dataPoints: projectData.dataPoints.map((dp) => ({
          id: dp.id,
          path: dp.path,
          name: dp.name,
          sourceType: dp.sourceType,
          sourceId: dp.sourceId,
          sourceConfig: dp.sourceConfig,
          dataType: dp.dataType,
          unit: dp.unit,
          defaultValue: dp.defaultValue,
        })),
        queries: projectData.queries.map((q) => ({
          id: q.id,
          name: q.name,
          connectionId: q.connectionId,
          queryType: q.queryType,
          config: q.config,
          transformer: q.transformer,
          timeoutMs: q.timeoutMs,
          cacheEnabled: q.cacheEnabled,
          cacheTtlSeconds: q.cacheTtlSeconds,
        })),
        protocols: normalizeArtifactProtocols(projectData.protocols),
      };
      archive.append(JSON.stringify(datacenterJson, null, 2), { name: "datacenter.json" });

      // 添加运行态安全快照，供节点侧离线认证和鉴权直接读取。
      archive.append(JSON.stringify(runtimeSecuritySnapshot, null, 2), {
        name: RUNTIME_SECURITY_FILE_NAME,
      });

      // TODO: 添加 assets 目录（如果有资源文件）

      archive.finalize();
    });
  }

  /**
   * 计算文件哈希
   */
  async calculateFileHash(filePath) {
    return new Promise((resolve, reject) => {
      const hash = crypto.createHash("sha256");
      const stream = fs.createReadStream(filePath);
      stream.on("data", (data) => hash.update(data));
      stream.on("end", () => resolve(hash.digest("hex")));
      stream.on("error", reject);
    });
  }

  /**
   * 统计组件数量
   */
  countComponents(pages) {
    let count = 0;
    for (const page of pages) {
      if (page.schemaContent?.components) {
        count += this.countNestedComponents(page.schemaContent.components);
      }
    }
    return count;
  }

  countNestedComponents(components) {
    if (!Array.isArray(components)) return 0;
    let count = components.length;
    for (const comp of components) {
      if (comp.children) {
        count += this.countNestedComponents(comp.children);
      }
    }
    return count;
  }

  /**
   * 获取发布详情
   */
  async getDeployment(deploymentId) {
    return Deployment.findByPk(deploymentId, {
      include: [
        { model: Project, as: "project", attributes: ["id", "name", "code"] },
        { model: User, as: "deployer", attributes: ["id", "username"] },
      ],
    });
  }

  /**
   * 获取工程的发布历史
   */
  async getDeploymentsByProject(projectId, options = {}) {
    const { page = 1, pageSize = 20 } = options;
    const offset = (page - 1) * pageSize;

    const { rows, count } = await Deployment.findAndCountAll({
      where: { projectId },
      include: [
        { model: User, as: "deployer", attributes: ["id", "username"] },
        { model: NodeDeployment, as: "nodeDeployments", attributes: ["id"] },
      ],
      order: [["createdAt", "DESC"]],
      limit: pageSize,
      offset,
    });

    const items = rows.map((item) => {
      const json = item.toJSON();
      const refs = Array.isArray(json.nodeDeployments) ? json.nodeDeployments.length : 0;
      return {
        ...json,
        nodeDeploymentRefCount: refs,
        canDelete: refs === 0,
      };
    });

    return {
      items,
      total: count,
      page,
      pageSize,
    };
  }

  /**
   * 下载制品
   */
  async getArtifactPath(deploymentId) {
    const deployment = await Deployment.findByPk(deploymentId);
    if (!deployment || !deployment.artifactUrl) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { message: "制品不存在" });
    }

    // 新记录优先使用 buildConfig 中固化的 artifactFileName；
    // 旧记录则按 `projectId + version + 时间戳` 的既有命名规则回溯，避免受受控下载 URL 影响。
    const candidates = await resolveArtifactFileCandidates(deployment);
    for (const filePath of candidates) {
      if (await fs.pathExists(filePath)) {
        return filePath;
      }
    }

    throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { message: "制品文件不存在" });
  }

  /**
   * 删除发布记录（软删除）。
   * 仅允许删除未被节点部署引用的发布版本，避免破坏历史关联关系。
   * @param {string} deploymentId - 发布记录ID
   * @returns {Promise<{deleted:boolean,id:string}>}
   */
  async deleteDeployment(deploymentId) {
    const deployment = await Deployment.findByPk(deploymentId);
    if (!deployment) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { message: "发布记录不存在" });
    }

    const referencedCount = await NodeDeployment.count({
      where: { deploymentId },
    });
    if (referencedCount > 0) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: "发布版本已被部署引用，无法删除",
      });
    }

    // 发布版本号受唯一索引约束，删除时必须物理删除以释放版本号。
    await deployment.destroy({ force: true });
    return { deleted: true, id: deploymentId };
  }
}

module.exports = new PublishService();
