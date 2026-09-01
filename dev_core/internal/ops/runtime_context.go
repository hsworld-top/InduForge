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
	err := r.pool.QueryRow(ctx, `SELECT d.tenant_id::text,d.project_id::text,d.environment_id::text,d.mode,COALESCE(v.id::text,''),COALESCE(v.version,''),COALESCE(v.artifact_key,''),COALESCE(v.artifact_hash,''),COALESCE(v.manifest_hash,''),COALESCE(v.checksums_hash,''),COALESCE(v.signing_key_id,''),COALESCE(v.artifact_size,0),COALESCE(v.manifest,'{}'::jsonb),COALESCE(b.artifact_descriptor,'{}'::jsonb),COALESCE(b.revision,0) FROM project_deployments d LEFT JOIN application_versions v ON v.id=d.application_version_id AND v.tenant_id=d.tenant_id AND v.status='ready' AND v.deleted_at IS NULL LEFT JOIN LATERAL (SELECT artifact_descriptor,revision FROM deployment_bindings WHERE project_deployment_id=d.id AND artifact_mode=CASE WHEN d.mode='development' THEN 'development' ELSE 'release' END ORDER BY revision DESC LIMIT 1) b ON true WHERE d.id=$1 AND d.desired_status='running'`, deploymentID).Scan(&out.TenantID, &out.ProjectID, &out.EnvironmentID, &out.Mode, &releaseID, &out.Release.Version, &out.Release.ArtifactKey, &out.Release.ArtifactHash, &out.Release.ManifestHash, &out.Release.ChecksumsHash, &out.Release.SigningKeyID, &out.Release.ArtifactSize, &out.Release.Manifest, &descriptor, &revision)
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
	if source, required, e := collectorSourceSnapshotFromManifest(out.Release.Manifest, out.ProjectID); e != nil {
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
