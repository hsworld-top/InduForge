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
	// 静态开发规则随 SDK 发布；此处仅生成当前工程业务数据，避免两处说明漂移。
	readme := "# 工程上下文\n\n此目录是平台生成的只读业务数据。开发规则、能力说明和示例见 /opt/induforge/project-sdk。先读取实际工程 package.json 和 .induforge/project.json 确定技术栈，再按需查询本目录的数据点和场景。\n"
	files := map[string][]byte{
		"README.md":              []byte(readme),
		"project/overview.md":    []byte(overview),
		"authorization/roles.md": []byte(strings.Join(roleLines, "\n")),
		"points/README.md":       []byte("# 数据点开发契约\n\n数据点按 500 条分片写入 catalog/，请按需检索读取。\n"),
		"scenes/README.md":       []byte("# 场景公开契约\n\n此处描述工程页面可用的场景参数、事件和命令，不包含私有画布结构。\n"),
	}
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
	sceneManifest := make([]scenecontract.Contract, 0, len(scenes.Contracts))
	for _, scene := range scenes.Contracts {
		path := fmt.Sprintf("%s/%s.md", scene.Kind, scene.ID)
		sceneIndex = append(sceneIndex, "- [`"+scene.ID+"` · "+scene.Name+"](./"+path+")")
		files["scenes/"+path] = []byte(sceneMarkdown(scene))
		sceneManifest = append(sceneManifest, scene)
	}
	if len(scenes.Contracts) == 0 {
		sceneIndex = append(sceneIndex, "当前工程尚未声明 2D 或 3D 场景公开契约。")
	}
	files["scenes/index.md"] = []byte(strings.Join(sceneIndex, "\n") + "\n")
	sceneManifestJSON, err := json.MarshalIndent(map[string]any{"contractVersion": scenes.ContractVersion, "scenes": sceneManifest}, "", "  ")
	if err != nil {
		return nil, err
	}
	files["scenes/manifest.json"] = sceneManifestJSON
	manifest, err := json.MarshalIndent(map[string]any{"projectId": item.ID, "generatedAt": time.Now().UTC().Format(time.RFC3339), "pointContractVersion": points.ContractVersion, "pointCount": len(points.DataPoints), "roleCount": len(roles), "pointChunkCount": chunkCount, "sceneContractVersion": scenes.ContractVersion, "sceneCount": len(scenes.Contracts)}, "", "  ")
	if err != nil {
		return nil, err
	}
	files["manifest.json"] = manifest
	return files, nil
}

func sceneMarkdown(scene scenecontract.Contract) string {
	lines := []string{"# " + scene.Name, "", "- 场景 ID：`" + scene.ID + "`", "- 类型：`" + scene.Kind + "`", "- 契约版本：`" + scene.ContractVersion + "`"}
	if scene.Description != "" {
		lines = append(lines, "", scene.Description)
	}
	tag := "induforge-scene-2d"
	if scene.Kind == "3d" {
		tag = "induforge-scene-3d"
	}
	lines = append(lines, "", "## 页面使用", "", "~~~html", "<"+tag+" scene-id=\""+scene.ID+"\"></"+tag+">", "~~~")
	selector := tag + "[scene-id=\\\"" + scene.ID + "\\\"]"
	lines = append(lines, "", "~~~js", "import '@induforge/runtime-sdk/scene-elements'", "", "const scene = document.querySelector('"+selector+"')")
	if len(scene.Parameters) > 0 {
		params := make(map[string]any, len(scene.Parameters))
		for _, parameter := range scene.Parameters {
			params[parameter.Name] = schemaExample(parameter.Schema)
		}
		encoded, _ := json.MarshalIndent(params, "", "  ")
		lines = append(lines, "await scene.setParams("+string(encoded)+")")
	}
	if len(scene.Events) > 0 {
		lines = append(lines, "scene.addEventListener('scene-event', (event) => {", "  const { name, payload } = event.detail", "  console.log(name, payload)", "})")
	}
	if len(scene.Commands) > 0 {
		command := scene.Commands[0]
		input, _ := json.MarshalIndent(schemaExample(command.InputSchema), "", "  ")
		lines = append(lines, "const result = await scene.invoke('"+command.Name+"', "+string(input)+")", "console.log(result)")
	}
	lines = append(lines, "~~~")
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
	parameters := make([]scenecontract.Member, 0, len(scene.Parameters))
	for _, parameter := range scene.Parameters {
		description := parameter.Description
		if parameter.Required {
			description = strings.TrimSpace(description + "（必填）")
		}
		parameters = append(parameters, scenecontract.Member{Name: parameter.Name, Description: description, Schema: parameter.Schema})
	}
	appendMembers("参数", parameters)
	appendMembers("事件", scene.Events)
	if len(scene.Commands) > 0 {
		lines = append(lines, "", "## 命令", "")
		for _, command := range scene.Commands {
			input, _ := json.Marshal(command.InputSchema)
			output, _ := json.Marshal(command.OutputSchema)
			line := "- `" + command.Name + "`"
			if command.Description != "" {
				line += "：" + command.Description
			}
			line += "，输入：`" + string(input) + "`，输出：`" + string(output) + "`"
			lines = append(lines, line)
		}
	}
	if len(scene.DatapointRefs) > 0 {
		lines = append(lines, "", "## 数据点引用", "", "`"+strings.Join(scene.DatapointRefs, "`, `")+"`")
	}
	return strings.Join(lines, "\n") + "\n"
}

func schemaExample(schema map[string]any) any {
	if value, ok := schema["const"]; ok {
		return value
	}
	if values, ok := schema["enum"].([]any); ok && len(values) > 0 {
		return values[0]
	}
	switch schema["type"] {
	case "string":
		return "example"
	case "integer", "number":
		return 0
	case "boolean":
		return false
	case "array":
		if items, ok := schema["items"].(map[string]any); ok {
			return []any{schemaExample(items)}
		}
		return []any{}
	case "object":
		result := map[string]any{}
		if properties, ok := schema["properties"].(map[string]any); ok {
			for name, value := range properties {
				if property, ok := value.(map[string]any); ok {
					result[name] = schemaExample(property)
				}
			}
		}
		return result
	default:
		return nil
	}
}
