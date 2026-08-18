package contextpack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/project"
	"github.com/indu-forge/dev_core/internal/runtimeaccess"
	"github.com/indu-forge/dev_core/internal/scenecontract"
)

type pointSnapshot struct {
	ContractVersion string `json:"contractVersion"`
	DataPoints      []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		DataType    string `json:"dataType"`
		Status      string `json:"status"`
		Methods     []struct {
			Name       string         `json:"name"`
			Parameters map[string]any `json:"parameters"`
			Result     map[string]any `json:"result"`
			Event      map[string]any `json:"event"`
		} `json:"methods"`
	} `json:"datapoints"`
}

type dataServiceEnvelope struct {
	Code int           `json:"code"`
	Msg  string        `json:"msg"`
	Data pointSnapshot `json:"data"`
}

type Service struct {
	projects       *project.Service
	runtimeAccess  *runtimeaccess.Service
	workspace      *project.FileWorkspace
	dataServiceURL string
	httpClient     *http.Client
	scenes         SceneContractSource
}

type SceneContractSource interface {
	List(context.Context, auth.User, string) (scenecontract.Snapshot, error)
}

func NewService(projects *project.Service, runtime *runtimeaccess.Service, workspace *project.FileWorkspace, dataServiceURL string) *Service {
	return &Service{projects: projects, runtimeAccess: runtime, workspace: workspace, dataServiceURL: strings.TrimRight(dataServiceURL, "/"), httpClient: &http.Client{Timeout: 20 * time.Second}}
}

func (s *Service) SetSceneContracts(scenes SceneContractSource) { s.scenes = scenes }

func (s *Service) Sync(ctx context.Context, actor auth.User, projectID, authorization string) (map[string]any, error) {
	if s == nil || s.projects == nil || s.runtimeAccess == nil || s.workspace == nil {
		return nil, fmt.Errorf("工程上下文服务未初始化")
	}
	if s.dataServiceURL == "" {
		return nil, fmt.Errorf("未配置 DATA_SERVICE_URL，无法生成数据点上下文")
	}
	item, err := s.projects.Get(ctx, actor, projectID)
	if err != nil {
		return nil, err
	}
	roles, err := s.runtimeAccess.ListRoles(ctx, actor, projectID)
	if err != nil {
		return nil, err
	}
	points, err := s.fetchPoints(ctx, projectID, authorization)
	if err != nil {
		return nil, err
	}
	scenes := scenecontract.Snapshot{}
	if s.scenes != nil {
		scenes, err = s.scenes.List(ctx, actor, projectID)
		if err != nil {
			return nil, err
		}
	}
	files, err := buildFiles(item, roles, points, scenes)
	if err != nil {
		return nil, err
	}
	if err := s.workspace.SyncContext(projectID, files); err != nil {
		return nil, err
	}
	return map[string]any{"contractVersion": points.ContractVersion, "pointCount": len(points.DataPoints), "roleCount": len(roles), "sceneCount": len(scenes.Contracts), "sceneContractVersion": scenes.ContractVersion, "updatedAt": time.Now().UTC().Format("2006-01-02 15:04:05")}, nil
}

func (s *Service) fetchPoints(ctx context.Context, projectID, authorization string) (pointSnapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.dataServiceURL+"/api/v1/data/projects/"+projectID+"/datapoints/development-contract", nil)
	if err != nil {
		return pointSnapshot{}, err
	}
	req.Header.Set("Authorization", authorization)
	response, err := s.httpClient.Do(req)
	if err != nil {
		return pointSnapshot{}, fmt.Errorf("读取数据点开发契约失败: %w", err)
	}
	defer response.Body.Close()
	var payload dataServiceEnvelope
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return pointSnapshot{}, fmt.Errorf("解析数据点开发契约失败: %w", err)
	}
	if response.StatusCode/100 != 2 || payload.Code != 0 {
		return pointSnapshot{}, fmt.Errorf("读取数据点开发契约失败: %s", payload.Msg)
	}
	return payload.Data, nil
}

func buildFiles(item project.Project, roles []runtimeaccess.Role, points pointSnapshot, scenes scenecontract.Snapshot) (map[string][]byte, error) {
	roleLines := []string{"# 运行角色", "", "角色和能力仅用于开发参考，运行时仍由服务端校验。", ""}
	for _, role := range roles {
		roleLines = append(roleLines, "## "+role.Code+"（"+role.Name+"）", "", role.Description, "", "能力：`"+strings.Join(role.Capabilities, "`, `")+"`", "")
	}
	overview := "# 工程信息\n\n- 名称：" + item.Name + "\n- 编码：`" + item.Code + "`\n- 工程 ID：`" + item.ID + "`\n"
	readme := "# InduForge 工程上下文包\n\n这是数据库和场景公开契约的只读开发镜像。先搜索 `points/README.md`，再使用 `@induforge/runtime-sdk` 调用数据点、权限和场景能力。\n"
	files := map[string][]byte{"README.md": []byte(readme), "project/overview.md": []byte(overview), "project/development-rules.md": []byte("# 开发规则\n\n普通页面使用 Vue 3 + JavaScript；路由源码是唯一事实。\n"), "runtime-api.md": []byte(runtimeAPIMarkdown()), "authorization/roles.md": []byte(strings.Join(roleLines, "\n")), "authorization/permissions.md": []byte("# 权限\n\n请通过运行时 SDK 判断角色和权限。\n"), "authorization/usage.md": []byte(authorizationUsageMarkdown()), "points/README.md": []byte("# 数据点开发契约\n\n数据点按 500 条分片写入 `catalog/`，请先搜索 ID，再按需读取对应文件。\n"), "scenes/README.md": []byte("# 场景公开契约\n\n这里仅描述 2D、3D 场景向 Vue 页面和运行时 SDK 公开的接口，不包含 HT 私有画布结构。\n")}
	const pointsPerShard = 500
	chunkCount := 0
	for start := 0; start < len(points.DataPoints); start += pointsPerShard {
		end := start + pointsPerShard
		if end > len(points.DataPoints) {
			end = len(points.DataPoints)
		}
		chunkCount++
		lines := []string{"# 数据点契约分片 " + fmt.Sprintf("%04d", chunkCount), ""}
		for _, point := range points.DataPoints[start:end] {
			lines = append(lines, "## `"+point.ID+"`", "", point.Name+" · "+point.DataType+" · "+point.Status, "", point.Description, "")
			for _, method := range point.Methods {
				parameters, _ := json.Marshal(method.Parameters)
				result, _ := json.Marshal(method.Result)
				event, _ := json.Marshal(method.Event)
				lines = append(lines, "- `"+method.Name+"` 参数：`"+string(parameters)+"` 返回：`"+string(result)+"` 事件：`"+string(event)+"`")
			}
			lines = append(lines, "")
		}
		files[fmt.Sprintf("points/catalog/%04d.md", chunkCount)] = []byte(strings.Join(lines, "\n"))
	}
	pointManifest, err := json.MarshalIndent(map[string]any{"contractVersion": points.ContractVersion, "pointCount": len(points.DataPoints), "chunkCount": chunkCount, "pointsPerChunk": pointsPerShard}, "", "  ")
	if err != nil {
		return nil, err
	}
	files["points/manifest.json"] = pointManifest
	sceneIndex := []string{"# 场景索引", ""}
	for _, scene := range scenes.Contracts {
		path := fmt.Sprintf("%s/%s.md", scene.Kind, scene.ID)
		sceneIndex = append(sceneIndex, "- [`"+scene.ID+"` · "+scene.Name+"](./"+path+")")
		files["scenes/"+path] = []byte(sceneMarkdown(scene))
	}
	if len(scenes.Contracts) == 0 {
		sceneIndex = append(sceneIndex, "当前工程尚未声明 2D 或 3D 场景公开契约。")
	}
	files["scenes/index.md"] = []byte(strings.Join(sceneIndex, "\n") + "\n")
	manifest, err := json.MarshalIndent(map[string]any{"projectId": item.ID, "generatedAt": time.Now().UTC().Format(time.RFC3339), "pointContractVersion": points.ContractVersion, "pointCount": len(points.DataPoints), "roleCount": len(roles), "pointChunkCount": chunkCount, "sceneContractVersion": scenes.ContractVersion, "sceneCount": len(scenes.Contracts)}, "", "  ")
	if err != nil {
		return nil, err
	}
	files["manifest.json"] = manifest
	return files, nil
}

func sceneMarkdown(scene scenecontract.Contract) string {
	lines := []string{"# " + scene.Name, "", "- 场景 ID：`" + scene.ID + "`", "- 类型：`" + scene.Kind + "`", "- 嵌入方式：`" + scene.EmbedMode + "`", "- 契约版本：`" + scene.ContractVersion + "`"}
	if scene.Description != "" {
		lines = append(lines, "", scene.Description)
	}
	if scene.Route != "" {
		lines = append(lines, "", "- 路由：`"+scene.Route+"`")
	}
	appendMembers := func(title string, members []scenecontract.Member) {
		if len(members) == 0 {
			return
		}
		lines = append(lines, "", "## "+title, "")
		for _, member := range members {
			schema, _ := json.Marshal(member.Schema)
			line := "- `" + member.Name + "`"
			if member.Description != "" {
				line += "：" + member.Description
			}
			if len(schema) > 0 && string(schema) != "null" {
				line += "，Schema：`" + string(schema) + "`"
			}
			lines = append(lines, line)
		}
	}
	appendMembers("输入", scene.Inputs)
	appendMembers("事件", scene.Events)
	appendMembers("命令", scene.Commands)
	appendMembers("公开对象", scene.PublicObjects)
	if len(scene.DatapointRefs) > 0 {
		lines = append(lines, "", "## 数据点引用", "", "`"+strings.Join(scene.DatapointRefs, "`, `")+"`")
	}
	if len(scene.PermissionRefs) > 0 {
		lines = append(lines, "", "## 权限引用", "", "`"+strings.Join(scene.PermissionRefs, "`, `")+"`")
	}
	return strings.Join(lines, "\n") + "\n"
}

func runtimeAPIMarkdown() string {
	return `# 运行时 SDK

客户页面使用 JavaScript，导入工程模板内置的 @induforge/runtime-sdk：

~~~js
import { points, access, scenes } from '@induforge/runtime-sdk'

const result = await points.production.orders.get({ workshopId: 'A01' })
await points.production.command.set({ command: 'start' })
const unsubscribe = points.production.status.sub((event) => console.log(event))
unsubscribe()

if (access.hasRole('operator')) {
  scenes.open2D('line-overview')
}
~~~

- 数据点路径以 points/catalog/ 中的稳定 ID 为准。
- get、set、sub、pub 是否可用和参数结构以对应数据点契约为准。
- 场景只通过 scenes.open2D()、scenes.open3D() 和场景公开契约使用，不读取 HT 私有文件。
- 运行时仍执行真实权限、参数和数据校验；上下文文件只用于开发提示。
`
}

func authorizationUsageMarkdown() string {
	return `# 权限使用

~~~js
import { access } from '@induforge/runtime-sdk'

const canOperate = access.hasRole('operator')
const canManage = access.hasAnyRole(['admin', 'engineer'])
~~~

角色和权限结果取自当前运行用户。不要把上下文中的角色文档当作授权依据，也不要在页面中保存 Token、用户名或角色绑定关系。
`
}
