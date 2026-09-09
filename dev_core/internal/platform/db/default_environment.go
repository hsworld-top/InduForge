package db

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"strings"
)

const DefaultRuntimeEnvironmentCode = "default-runtime"

// EnsureDefaultRuntimeEnvironment 在调用方的初始化事务中创建组织默认环境。
// 不关联节点或部署服务；重复初始化保留已有环境，事件仅记录一次。
func EnsureDefaultRuntimeEnvironment(ctx context.Context, tx pgx.Tx, tenantID, actorID string) (string, error) {
	var environmentID string
	err := tx.QueryRow(ctx, `INSERT INTO runtime_environments(tenant_id,name,code,is_default,created_by) VALUES($1,'默认环境',$2,true,$3) ON CONFLICT (tenant_id,code) DO NOTHING RETURNING id::text`, tenantID, DefaultRuntimeEnvironmentCode, actorID).Scan(&environmentID)
	if err == nil {
		_, err = tx.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,message,created_by) VALUES($1,$2,'default_environment_created','默认环境已初始化','平台','success','默认环境已创建，请关联节点',$3)`, tenantID, environmentID, actorID)
		return environmentID, err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	err = tx.QueryRow(ctx, `SELECT id::text FROM runtime_environments WHERE tenant_id=$1 AND code=$2 AND is_default AND deleted_at IS NULL FOR UPDATE`, tenantID, DefaultRuntimeEnvironmentCode).Scan(&environmentID)
	return environmentID, err
}

type BuiltInRuntimeConfig struct {
	NodeName     string
	NodeIP       string
	Architecture string
	K3sVersion   string
	K3sAPIPort   int
}

// EnsureBuiltInRuntime 在组织初始化事务中登记中心自带的 K3s 节点和运行集群。
// 该节点由中心控制面直接观测和调度，不具备 NodeAgent 接入身份。
func EnsureBuiltInRuntime(ctx context.Context, tx pgx.Tx, tenantID, actorID string, config BuiltInRuntimeConfig) (string, string, error) {
	nodeName := strings.TrimSpace(config.NodeName)
	if nodeName == "" {
		nodeName = "induframe-center"
	}
	architecture := strings.TrimSpace(config.Architecture)
	if architecture == "" {
		architecture = "unknown"
	}
	if strings.TrimSpace(config.K3sVersion) == "" || config.K3sAPIPort < 1 || config.K3sAPIPort > 65535 {
		return "", "", fmt.Errorf("中心内置节点运行配置无效")
	}
	environmentID, err := EnsureDefaultRuntimeEnvironment(ctx, tx, tenantID, actorID)
	if err != nil {
		return "", "", err
	}
	var nodeID string
	err = tx.QueryRow(ctx, `INSERT INTO host_nodes(tenant_id,node_source,display_name,hostname,platform,architecture,ip_address,desired_status,observed_status,capabilities,approved_at,approved_by)
		VALUES($1,'built_in','中心内置节点',$2,'linux',$3,NULLIF($4,''),'active','online','["project_entry","data_runtime"]'::jsonb,now(),$5)
		ON CONFLICT (tenant_id) WHERE node_source='built_in' DO UPDATE SET hostname=EXCLUDED.hostname,architecture=EXCLUDED.architecture,ip_address=EXCLUDED.ip_address,updated_at=now()
		RETURNING id::text`, tenantID, nodeName, architecture, strings.TrimSpace(config.NodeIP), actorID).Scan(&nodeID)
	if err != nil {
		return "", "", err
	}
	var clusterID string
	var persistedVersion string
	var persistedPort int
	err = tx.QueryRow(ctx, `INSERT INTO runtime_clusters(tenant_id,k3s_version,api_port) VALUES($1,$2,$3)
		ON CONFLICT (tenant_id) DO UPDATE SET updated_at=runtime_clusters.updated_at
		RETURNING id::text,k3s_version,api_port`, tenantID, config.K3sVersion, config.K3sAPIPort).Scan(&clusterID, &persistedVersion, &persistedPort)
	if err != nil {
		return "", "", err
	}
	if persistedVersion != config.K3sVersion || persistedPort != config.K3sAPIPort {
		return "", "", fmt.Errorf("中心 K3s 配置与组织运行集群不一致")
	}
	tag, err := tx.Exec(ctx, `INSERT INTO runtime_cluster_nodes(tenant_id,cluster_id,node_id,node_kind,desired_action,desired_generation,observed_generation,cluster_status,cluster_message,cluster_observed_at)
		VALUES($1,$2,$3,'center','active',1,1,'ready','中心 K3s 已就绪',now()) ON CONFLICT (node_id) DO NOTHING`, tenantID, clusterID, nodeID)
	if err != nil {
		return "", "", err
	}
	if tag.RowsAffected() > 0 {
		if _, err = tx.Exec(ctx, `INSERT INTO runtime_cluster_events(tenant_id,cluster_id,node_id,event_type,name,target,result,message,created_by)
			VALUES($1,$2,$3,'center_initialized','中心内置节点已初始化','中心内置节点','success','中心安装自带的 K3s 节点已纳入运行集群',$4)`, tenantID, clusterID, nodeID, actorID); err != nil {
			return "", "", err
		}
	}
	tag, err = tx.Exec(ctx, `INSERT INTO runtime_environment_nodes(tenant_id,environment_id,node_id,created_by) VALUES($1,$2,$3,$4) ON CONFLICT (environment_id,node_id) DO NOTHING`, tenantID, environmentID, nodeID, actorID)
	if err != nil {
		return "", "", err
	}
	if tag.RowsAffected() > 0 {
		if _, err = tx.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,message,created_by)
			VALUES($1,$2,'built_in_node_assigned','中心内置节点已关联','中心内置节点','success','默认环境可按需在中心节点部署基础服务或工程',$3)`, tenantID, environmentID, actorID); err != nil {
			return "", "", err
		}
	}
	return environmentID, nodeID, nil
}
