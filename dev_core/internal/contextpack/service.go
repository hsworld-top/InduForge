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
	ContractVersion string             `json:"contractVersion"`
	ProjectID       string             `json:"projectId"`
	GeneratedAt     time.Time          `json:"generatedAt"`
	DataPoints      []dataPointContext `json:"datapoints"`
}

// dataPointContext 必须与 data_service 的开发契约保持同构，避免上下文生成时丢失
// 属性、方法说明和更新时间。这里不包含实时值、连接参数等运行期私有数据。
type dataPointContext struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	DataType    string            `json:"dataType"`
	Status      string            `json:"status"`
	Attributes  map[string]string `json:"attributes"`
	Methods     []dataPointMethod `json:"methods"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

type dataPointMethod struct {
	Name        string         `json:"name"`
	Parameters  map[string]any `json:"parameters"`
	Result      map[string]any `json:"result,omitempty"`
	Event       map[string]any `json:"event,omitempty"`
	Description string         `json:"description"`
}

type roleContext struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"projectId"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Status       string    `json:"status"`
	IsBuiltin    bool      `json:"isBuiltin"`
	Capabilities []string  `json:"capabilities"`
	UserCount    int64     `json:"userCount"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type dataServiceEnvelope struct {
	Code int           `json:"code"`
	Msg  string        `json:"msg"`
	Data pointSnapshot `json:"data"`
}

type resourceContext struct {
	Items     []map[string]any
	Available bool
	Error     string
}

// ObjectLibrarySource 只返回工程对象库的公开元数据，不允许把对象存储 key、
// 访问凭据或内部 Provider 信息带入开发工作区。
type ObjectLibrarySource func(context.Context, auth.User, string) ([]map[string]any, error)

type listEnvelope struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		List       []map[string]any `json:"list"`
		Pagination struct {
			Page       int `json:"page"`
			TotalPages int `json:"totalPages"`
		} `json:"pagination"`
	} `json:"data"`
}

type Service struct {
	projects       *project.Service
	runtimeAccess  *runtimeaccess.Service
	workspace      *project.FileWorkspace
	dataServiceURL string
	httpClient     *http.Client
	scenes         SceneContractSource
	objectLibrary  ObjectLibrarySource
}

type SceneContractSource interface {
	List(context.Context, auth.User, string) (scenecontract.Snapshot, error)
}

func NewService(projects *project.Service, runtime *runtimeaccess.Service, workspace *project.FileWorkspace, dataServiceURL string) *Service {
	return &Service{projects: projects, runtimeAccess: runtime, workspace: workspace, dataServiceURL: strings.TrimRight(dataServiceURL, "/"), httpClient: &http.Client{Timeout: 20 * time.Second}}
}

func (s *Service) SetSceneContracts(scenes SceneContractSource) { s.scenes = scenes }

func (s *Service) SetObjectLibrary(source ObjectLibrarySource) { s.objectLibrary = source }

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
	alarms := s.fetchOptionalList(ctx, projectID, authorization, "/alarm-items", "报警")
	computes := s.fetchOptionalList(ctx, projectID, authorization, "/compute-units", "计算单元")
	objects := resourceContext{Items: []map[string]any{}}
	if s.objectLibrary == nil {
		objects.Error = "当前服务未接入工程对象库"
	} else if items, objectErr := s.objectLibrary(ctx, actor, projectID); objectErr != nil {
		objects.Error = fmt.Sprintf("读取对象库上下文失败: %v", objectErr)
	} else {
		objects.Items = items
		objects.Available = true
	}
	files, err := buildFiles(item, roles, points, scenes, alarms, computes, objects)
	if err != nil {
		return nil, err
	}
	if err := s.workspace.SyncContext(projectID, files); err != nil {
		return nil, err
	}
	return map[string]any{"schemaVersion": "workspace-context.v2", "contractVersion": points.ContractVersion, "pointCount": len(points.DataPoints), "roleCount": len(roles), "alarmCount": len(alarms.Items), "computeCount": len(computes.Items), "objectLibraryCount": len(objects.Items), "sceneCount": len(scenes.Contracts), "sceneContractVersion": scenes.ContractVersion, "updatedAt": time.Now().UTC().Format("2006-01-02 15:04:05")}, nil
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

func (s *Service) fetchOptionalList(ctx context.Context, projectID, authorization, suffix, resource string) resourceContext {
	result := resourceContext{Items: []map[string]any{}}
	for page := 1; ; page++ {
		separator := "?"
		if strings.Contains(suffix, "?") {
			separator = "&"
		}
		url := s.dataServiceURL + "/api/v1/data/projects/" + projectID + suffix + separator + fmt.Sprintf("page=%d&pageSize=100", page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			result.Error = err.Error()
			return result
		}
		req.Header.Set("Authorization", authorization)
		response, err := s.httpClient.Do(req)
		if err != nil {
			result.Error = fmt.Sprintf("读取%s上下文失败: %v", resource, err)
			return result
		}
		var payload listEnvelope
		decodeErr := json.NewDecoder(response.Body).Decode(&payload)
		response.Body.Close()
		if decodeErr != nil {
			result.Error = fmt.Sprintf("解析%s上下文失败: %v", resource, decodeErr)
			return result
		}
		if response.StatusCode/100 != 2 || payload.Code != 0 {
			result.Error = fmt.Sprintf("读取%s上下文失败: %s", resource, payload.Msg)
			return result
		}
		result.Items = append(result.Items, payload.Data.List...)
		if payload.Data.Pagination.TotalPages <= page || len(payload.Data.List) == 0 {
			break
		}
	}
	result.Available = true
	return result
}

func buildFiles(item project.Project, roles []runtimeaccess.Role, points pointSnapshot, scenes scenecontract.Snapshot, resources ...resourceContext) (map[string][]byte, error) {
	alarms := resourceContext{Items: []map[string]any{}, Available: true}
	computes := resourceContext{Items: []map[string]any{}, Available: true}
	objects := resourceContext{Items: []map[string]any{}, Available: true}
	if len(resources) > 0 {
		alarms = resources[0]
	}
	if len(resources) > 1 {
		computes = resources[1]
	}
	if len(resources) > 2 {
		objects = resources[2]
	}
	roleLines := []string{"# 运行角色", "", "角色和能力仅用于开发参考，运行时仍由服务端校验。", ""}
	roleItems := make([]roleContext, 0, len(roles))
	for _, role := range roles {
		roleLines = append(roleLines, "## "+role.Code+"（"+role.Name+"）", "", role.Description, "", "能力：`"+strings.Join(role.Capabilities, "`, `")+"`", "")
		roleItems = append(roleItems, roleContext{ID: role.ID, ProjectID: role.ProjectID, Code: role.Code, Name: role.Name, Description: role.Description, Status: role.Status, IsBuiltin: role.IsBuiltin, Capabilities: append([]string(nil), role.Capabilities...), UserCount: role.UserCount, CreatedAt: role.CreatedAt, UpdatedAt: role.UpdatedAt})
	}
	overview := "# 工程信息\n\n- 名称：" + item.Name + "\n- 编码：`" + item.Code + "`\n- 工程 ID：`" + item.ID + "`\n"
	overviewJSON, err := json.MarshalIndent(map[string]any{"schemaVersion": "project-context.v1", "id": item.ID, "name": item.Name, "code": item.Code, "description": item.Description, "status": item.Status, "visibility": item.Visibility, "updatedAt": item.UpdatedAt}, "", "  ")
	if err != nil {
		return nil, err
	}
	rolesJSON, err := json.MarshalIndent(map[string]any{"schemaVersion": "authorization-context.v1", "projectId": item.ID, "roles": roleItems, "runtimeNote": "角色和能力仅用于开发参考，运行时仍由服务端校验。"}, "", "  ")
	if err != nil {
		return nil, err
	}
	// 静态开发规则随 SDK 发布；此处仅生成当前工程业务数据，避免两处说明漂移。
	readme := "# 工程上下文\n\n此目录是平台生成的只读业务数据。JSON 文件是机器读取的权威契约，Markdown 文件用于快速浏览和检索；两者由同一次同步生成，不应单独修改。开发规则、能力说明、Runtime SDK 和对象库使用示例见 /opt/induforge/project-sdk。先读取实际工程 package.json 和 .workspace/project.json 确定技术栈，再读取 manifest.json；开发静态资源前先读取 object-library/README.md、object-library/manifest.json 和 object-library/catalog.json，必要时查看同名 Markdown。\n"
	files := map[string][]byte{
		"README.md":                []byte(readme),
		"project/overview.md":      []byte(overview),
		"project/overview.json":    append(overviewJSON, '\n'),
		"authorization/roles.md":   []byte(strings.Join(roleLines, "\n")),
		"authorization/roles.json": append(rolesJSON, '\n'),
		"points/README.md":         []byte("# 数据点开发契约\n\nJSON 分片是机器读取的权威契约，Markdown 分片用于快速阅读。数据点按 500 条分片写入 catalog/，请先读取 manifest.json 再按需检索。\n"),
		"scenes/README.md":         []byte("# 场景公开契约\n\n此处描述工程页面可用的场景参数、事件和命令，不包含私有画布结构。\n"),
		"object-library/README.md": []byte("# 工程对象库\n\n这里列出当前工程可复用的图片、字体、模型、材质、Symbol 和 Component 元数据。对象字节由平台对象库托管，不能把内部 object key 写入源码，也不要把同一资源复制到 public。当前上下文提供资源 ID、类型、适用场景、入口文件和缩略图；页面运行时请优先使用场景 Provider 已声明的资源入口。当前 Runtime SDK 没有通用对象库读取方法，若当前环境没有对应能力，应先说明缺口，不要猜测 URL。\n"),
	}
	addResourceFiles(files, "alarms", "报警", alarms, nil)
	addResourceFiles(files, "computes", "计算单元", computes, []string{"scriptCode", "code"})
	addResourceFiles(files, "object-library", "对象库", objects, []string{"objectKey", "provider", "tenantId"})
	const pointsPerShard = 500
	chunkCount := 0
	for start := 0; start < len(points.DataPoints); start += pointsPerShard {
		end := start + pointsPerShard
		if end > len(points.DataPoints) {
			end = len(points.DataPoints)
		}
		chunkCount++
		lines := []string{"# 数据点契约分片 " + fmt.Sprintf("%04d", chunkCount), ""}
		chunkJSON, err := json.MarshalIndent(map[string]any{"schemaVersion": "datapoint-context.v1", "contractVersion": points.ContractVersion, "chunk": chunkCount, "points": points.DataPoints[start:end]}, "", "  ")
		if err != nil {
			return nil, err
		}
		for _, point := range points.DataPoints[start:end] {
			lines = append(lines, "## `"+point.ID+"`", "", point.Name+" · "+point.DataType+" · "+point.Status, "", point.Description, "")
			for _, method := range point.Methods {
				parameters, _ := json.Marshal(method.Parameters)
				result, _ := json.Marshal(method.Result)
				event, _ := json.Marshal(method.Event)
				line := "- `" + method.Name + "`：" + method.Description + " 参数：`" + string(parameters) + "` 返回：`" + string(result) + "` 事件：`" + string(event) + "`"
				lines = append(lines, line)
			}
			lines = append(lines, "")
		}
		files[fmt.Sprintf("points/catalog/%04d.md", chunkCount)] = []byte(strings.Join(lines, "\n"))
		files[fmt.Sprintf("points/catalog/%04d.json", chunkCount)] = append(chunkJSON, '\n')
	}
	pointManifest, err := json.MarshalIndent(map[string]any{"schemaVersion": "datapoint-context.v1", "contractVersion": points.ContractVersion, "projectId": points.ProjectID, "generatedAt": points.GeneratedAt, "pointCount": len(points.DataPoints), "chunkCount": chunkCount, "pointsPerChunk": pointsPerShard, "formats": []string{"json", "md"}}, "", "  ")
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
	sceneManifestJSON, err := json.MarshalIndent(map[string]any{"schemaVersion": "scene-context.v1", "contractVersion": scenes.ContractVersion, "formats": []string{"json", "md"}, "scenes": sceneManifest}, "", "  ")
	if err != nil {
		return nil, err
	}
	files["scenes/manifest.json"] = sceneManifestJSON
	missing := []string{"runtime-users"}
	if !alarms.Available {
		missing = append(missing, "alarm-context")
	}
	if !computes.Available {
		missing = append(missing, "compute-context")
	}
	if !objects.Available {
		missing = append(missing, "object-library")
	}
	manifest, err := json.MarshalIndent(map[string]any{"schemaVersion": "workspace-context.v2", "projectId": item.ID, "generatedAt": time.Now().UTC().Format(time.RFC3339), "pointContractVersion": points.ContractVersion, "pointCount": len(points.DataPoints), "roleCount": len(roles), "alarmCount": len(alarms.Items), "computeCount": len(computes.Items), "objectLibraryCount": len(objects.Items), "pointChunkCount": chunkCount, "sceneContractVersion": scenes.ContractVersion, "sceneCount": len(scenes.Contracts), "formats": []string{"json", "md"}, "missing": missing}, "", "  ")
	if err != nil {
		return nil, err
	}
	files["manifest.json"] = manifest
	return files, nil
}

func addResourceFiles(files map[string][]byte, directory, label string, resource resourceContext, omit []string) {
	items := make([]map[string]any, 0, len(resource.Items))
	for _, item := range resource.Items {
		copyItem := make(map[string]any, len(item))
		for key, value := range item {
			if containsString(omit, key) {
				continue
			}
			copyItem[key] = value
		}
		items = append(items, copyItem)
	}
	payload := map[string]any{"schemaVersion": directory + "-context.v1", "available": resource.Available, "count": len(items), "items": items}
	if resource.Error != "" {
		payload["error"] = resource.Error
	}
	jsonData, _ := json.MarshalIndent(payload, "", "  ")
	files[directory+"/manifest.json"] = append(jsonData, '\n')
	lines := []string{"# " + label + "开发契约", ""}
	if !resource.Available {
		lines = append(lines, "当前上下文未能读取该资源："+resource.Error, "")
	} else if len(items) == 0 {
		lines = append(lines, "当前工程没有已配置的"+label+"。", "")
	} else {
		for _, item := range items {
			name := stringValue(item["name"])
			if name == "" {
				name = stringValue(item["displayName"])
			}
			id := stringValue(item["id"])
			if id == "" {
				id = stringValue(item["assetId"])
			}
			lines = append(lines, "## `"+id+"` "+name, "", "```json", mustMarshal(item), "```", "")
		}
	}
	files[directory+"/catalog.md"] = []byte(strings.Join(lines, "\n"))
	files[directory+"/catalog.json"] = append(jsonData, '\n')
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func mustMarshal(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
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
