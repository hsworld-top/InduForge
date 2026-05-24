const { QueryTypes } = require('sequelize')

/**
 * 统一封装 design_project_settings 的读取与写入。
 * 这里显式使用 PostgreSQL 风格占位符与 ON CONFLICT，避免路由层散落方言差异。
 */
const SELECT_PROJECT_SETTINGS_SQL = `
  SELECT "globalVariables", "globalScripts", "i18n"
  FROM design_project_settings
  WHERE "projectId" = $1
  LIMIT 1
`

const UPSERT_PROJECT_SETTINGS_SQL = `
  INSERT INTO design_project_settings (
    "projectId",
    "schemaVersion",
    "globalVariables",
    "globalScripts",
    "i18n",
    "updatedBy",
    "updatedAt"
  )
  VALUES ($1, $2, $3::jsonb, $4::jsonb, $5::jsonb, $6, $7)
  ON CONFLICT ("projectId") DO UPDATE
  SET
    "globalVariables" = EXCLUDED."globalVariables",
    "globalScripts" = EXCLUDED."globalScripts",
    "i18n" = EXCLUDED."i18n",
    "updatedBy" = EXCLUDED."updatedBy",
    "updatedAt" = EXCLUDED."updatedAt"
`

async function getProjectSettingsRow(sequelize, projectId) {
  const rows = await sequelize.query(SELECT_PROJECT_SETTINGS_SQL, {
    bind: [projectId],
    type: QueryTypes.SELECT,
  })
  return Array.isArray(rows) && rows.length ? rows[0] : null
}

async function upsertProjectSettings(
  sequelize,
  { projectId, schemaVersion, globalVariables, globalScripts, i18n, updatedBy, updatedAt },
) {
  await sequelize.query(UPSERT_PROJECT_SETTINGS_SQL, {
    bind: [
      projectId,
      schemaVersion,
      JSON.stringify(globalVariables || {}),
      JSON.stringify(globalScripts || {}),
      JSON.stringify(i18n || {}),
      updatedBy,
      updatedAt,
    ],
  })
}

module.exports = {
  SELECT_PROJECT_SETTINGS_SQL,
  UPSERT_PROJECT_SETTINGS_SQL,
  getProjectSettingsRow,
  upsertProjectSettings,
}
