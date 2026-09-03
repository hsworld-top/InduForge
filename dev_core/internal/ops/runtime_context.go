package ops

import (
	"context"
	"encoding/json"
	"fmt"
)

// ProjectRuntimeContext 是调和器按 deployment 加载的受信运行快照；不含任何 Secret 值。
type ProjectRuntimeContext struct {
	TenantID, ProjectID, EnvironmentID, DeploymentID, Mode string
	Release                                                releaseMetadata
	BindingRevision                                        int
	Support                                                RuntimeSupportResources
	// RuntimeEngines 是本 deployment 当前期望运行的共享 JetStream 消费角色。
	// 它来自 deployment_services，而不是发布制品能力，避免未被部署的角色
	// 进入拓扑声明。
	RuntimeEngines []string
	// CollectorSourceSnapshot 是 Release 冻结的 collector artifact 完整性快照；
	// 它由发布编排注入，调和器不得以当前开发态配置替代。
	CollectorSourceSnapshot json.RawMessage
}

// LoadProjectRuntimeContext 保持 pending service 查询轻量：制品与环境支撑在这里
// 单独读取。生产只接受 ready application_version；开发只读取 binding descriptor。
func (r *PostgreSQLRepository) LoadProjectRuntimeContext(ctx context.Context, deploymentID string) (ProjectRuntimeContext, error) {
	var out ProjectRuntimeContext
	var descriptor []byte
	var releaseID string
	var revision int
	err := r.pool.QueryRow(ctx, `SELECT d.tenant_id::text,d.project_id::text,d.environment_id::text,d.mode,COALESCE(v.id::text,''),COALESCE(v.version,''),COALESCE(v.artifact_key,''),COALESCE(v.artifact_hash,''),COALESCE(v.manifest_hash,''),COALESCE(v.checksums_hash,''),COALESCE(v.signing_key_id,''),COALESCE(v.artifact_size,0),COALESCE(v.manifest,'{}'::jsonb),COALESCE(b.artifact_descriptor,'{}'::jsonb),COALESCE(b.revision,0) FROM project_deployments d LEFT JOIN application_versions v ON v.id=d.application_version_id AND v.tenant_id=d.tenant_id AND v.status='ready' AND v.deleted_at IS NULL LEFT JOIN LATERAL (SELECT artifact_descriptor,revision FROM deployment_bindings WHERE project_deployment_id=d.id AND artifact_mode=CASE WHEN d.mode='development' THEN 'development' ELSE 'release' END ORDER BY revision DESC LIMIT 1) b ON true WHERE d.id=$1 AND (d.desired_status='running' OR d.deletion_requested_at IS NOT NULL)`, deploymentID).Scan(&out.TenantID, &out.ProjectID, &out.EnvironmentID, &out.Mode, &releaseID, &out.Release.Version, &out.Release.ArtifactKey, &out.Release.ArtifactHash, &out.Release.ManifestHash, &out.Release.ChecksumsHash, &out.Release.SigningKeyID, &out.Release.ArtifactSize, &out.Release.Manifest, &descriptor, &revision)
	if err != nil {
		return ProjectRuntimeContext{}, mapNotFound(err)
	}
	out.DeploymentID = deploymentID
	out.BindingRevision = revision
	if revision < 1 {
		return ProjectRuntimeContext{}, fmt.Errorf("运行绑定不存在")
	}
	if out.Mode == "development" {
		var d DevelopmentArtifact
		if json.Unmarshal(descriptor, &d) != nil {
			return ProjectRuntimeContext{}, fmt.Errorf("开发制品描述损坏")
		}
		m, e := developmentReleaseMetadata(&d, out.ProjectID)
		if e != nil {
			return ProjectRuntimeContext{}, e
		}
		out.Release = m
	} else if out.Mode == "release" {
		out.Release.ID = releaseID
		if releaseID == "" {
			return ProjectRuntimeContext{}, fmt.Errorf("生产版本未就绪")
		}
		if e := validateReleaseMetadata(out.Release, out.ProjectID, false); e != nil {
			return ProjectRuntimeContext{}, e
		}
	} else {
		return ProjectRuntimeContext{}, fmt.Errorf("部署模式非法")
	}
	collectorSnapshot := collectorSourceSnapshotFromManifest
	if out.Mode == "development" {
		collectorSnapshot = developmentCollectorSourceSnapshotFromManifest
	}
	if source, required, e := collectorSnapshot(out.Release.Manifest, out.ProjectID); e != nil {
		return ProjectRuntimeContext{}, e
	} else if required {
		out.CollectorSourceSnapshot = source
	}
	// resource_refs 在基础服务异常时保留，便于诊断和恢复；只有 observed running
	// 的服务才能成为新工程部署的 resolver 支撑，避免把已知异常资源继续下发。
	rows, e := r.pool.Query(ctx, `SELECT service_type,resource_refs FROM runtime_environment_services WHERE environment_id=$1 AND desired_status='running' AND observed_status='running'`, out.EnvironmentID)
	if e != nil {
		return ProjectRuntimeContext{}, e
	}
	defer rows.Close()
	refs := map[string]map[string]string{}
	for rows.Next() {
		var kind string
		var raw []byte
		if e = rows.Scan(&kind, &raw); e != nil {
			return ProjectRuntimeContext{}, e
		}
		var v map[string]string
		if json.Unmarshal(raw, &v) != nil {
			return ProjectRuntimeContext{}, fmt.Errorf("环境资源引用损坏")
		}
		refs[kind] = v
	}
	if e = rows.Err(); e != nil {
		return ProjectRuntimeContext{}, e
	}
	out.Support, e = ResolveRuntimeSupportResources(refs)
	if e != nil {
		return ProjectRuntimeContext{}, e
	}
	engineRows, e := r.pool.Query(ctx, `SELECT service_type FROM deployment_services WHERE project_deployment_id=$1 AND desired_status='running' AND service_type IN ('base','compute','alarm') ORDER BY service_type`, deploymentID)
	if e != nil {
		return ProjectRuntimeContext{}, e
	}
	defer engineRows.Close()
	for engineRows.Next() {
		var engine string
		if e = engineRows.Scan(&engine); e != nil {
			return ProjectRuntimeContext{}, e
		}
		out.RuntimeEngines = append(out.RuntimeEngines, engine)
	}
	if e = engineRows.Err(); e != nil {
		return ProjectRuntimeContext{}, e
	}
	return out, nil
}

func collectorSourceSnapshotFromManifest(raw []byte, projectID string) (json.RawMessage, bool, error) {
	var manifest deployableReleaseManifest
	if json.Unmarshal(raw, &manifest) != nil {
		return nil, false, fmt.Errorf("Release Manifest 无效")
	}
	required := false
	for _, capability := range manifest.Capabilities {
		if capability == ServiceCollector {
			required = true
			break
		}
	}
	if !required {
		return nil, false, nil
	}
	if manifest.Artifacts.Collector == nil || !validCollectorSourceSnapshot(manifest.Artifacts.Collector.SourceSnapshot, manifest.Artifacts.Collector.SourceSnapshotSHA256, projectID) {
		return nil, true, fmt.Errorf("collector Release sourceSnapshot 缺失或无效")
	}
	return append(json.RawMessage(nil), manifest.Artifacts.Collector.SourceSnapshot...), true, nil
}

// developmentCollectorSourceSnapshotFromManifest 读取服务端开发制品中的采集快照。
// 开发制品没有 production Release 的外层冻结摘要，但仍必须校验其内部工件摘要及
// 项目身份；数据域构造 collector binding 时还会使用当前权威快照再次验证。
func developmentCollectorSourceSnapshotFromManifest(raw []byte, projectID string) (json.RawMessage, bool, error) {
	var manifest deployableReleaseManifest
	if json.Unmarshal(raw, &manifest) != nil {
		return nil, false, fmt.Errorf("开发制品 Manifest 无效")
	}
	required := false
	for _, capability := range manifest.Capabilities {
		if capability == ServiceCollector {
			required = true
			break
		}
	}
	if !required {
		return nil, false, nil
	}
	if manifest.Artifacts.Collector == nil || !validDevelopmentCollectorSourceSnapshot(manifest.Artifacts.Collector.SourceSnapshot, projectID) {
		return nil, true, fmt.Errorf("collector 开发 sourceSnapshot 缺失或无效")
	}
	return append(json.RawMessage(nil), manifest.Artifacts.Collector.SourceSnapshot...), true, nil
}

func validDevelopmentCollectorSourceSnapshot(raw []byte, projectID string) bool {
	if len(raw) == 0 || len(raw) > 16<<20 {
		return false
	}
	var source struct {
		SchemaVersion    string          `json:"schemaVersion"`
		ProjectID        string          `json:"projectId"`
		ArtifactRevision int64           `json:"artifactRevision"`
		SHA256           string          `json:"sha256"`
		Size             int             `json:"size"`
		Artifact         json.RawMessage `json:"artifact"`
	}
	return json.Unmarshal(raw, &source) == nil && source.SchemaVersion == "collector-runtime-artifact.v1" && source.ProjectID == projectID && source.ArtifactRevision > 0 && source.Size == len(source.Artifact) && source.SHA256 == "sha256:"+sha256Hex(source.Artifact) && len(source.Artifact) > 0 && json.Valid(source.Artifact)
}
