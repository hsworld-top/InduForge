package sceneasset

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
)

type ProviderCapabilities struct {
	Kinds            []string          `json:"kinds"`
	SupportsViewer   bool              `json:"supportsViewer"`
	SupportsImport   bool              `json:"supportsImport"`
	SupportsExport   bool              `json:"supportsExport"`
	MaxFileSizeBytes int64             `json:"maxFileSizeBytes"`
	Assets           []AssetCapability `json:"assets"`
}

type AssetCapability struct {
	Type       AssetType `json:"type"`
	Kinds      []string  `json:"kinds"`
	Extensions []string  `json:"extensions"`
	Editable   bool      `json:"editable"`
}

type ProviderFile struct {
	Path        string
	ContentType string
	Content     []byte
	Hash        string
}

type Analysis struct {
	Dependencies          []string
	DatapointRefs         []string
	DatapointRequirements []DatapointRequirement
	ManagedContract       map[string]any
}

type DatapointRequirement struct {
	Path      string `json:"path"`
	ValueType string `json:"valueType,omitempty"`
	Get       bool   `json:"get"`
	Sub       bool   `json:"sub"`
	Set       bool   `json:"set"`
}

type SceneProvider interface {
	Name() string
	Version() string
	Capabilities() ProviderCapabilities
	ValidateEntry(kind, entry string) error
	ValidateFile(path string, content []byte) error
	ValidateDraftFile(kind, entryPath, filename string, content []byte) error
	ExtractMetadata(content []byte) (Analysis, error)
	Analyze(ctx context.Context, kind, entry string, files map[string]ProviderFile) (Analysis, error)
	AnalyzeAsset(ctx context.Context, assetType AssetType, files map[string]ProviderFile) (AssetAnalysis, error)
	MountPath(asset SceneAsset) string
	EditorURL(session EditorSession) string
	AssetEditorURL(session AssetEditorSession) string
	ViewerURL(session ViewerSession) string
}

type Registry struct{ providers map[string]SceneProvider }

func NewRegistry(providers ...SceneProvider) *Registry {
	result := &Registry{providers: make(map[string]SceneProvider, len(providers))}
	for _, provider := range providers {
		result.providers[provider.Name()] = provider
	}
	return result
}

func (r *Registry) Get(name string) (SceneProvider, error) {
	provider, ok := r.providers[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return nil, ErrInvalidProvider
	}
	return provider, nil
}

type HTProvider struct{}

func NewHTProvider() *HTProvider    { return &HTProvider{} }
func (*HTProvider) Name() string    { return ProviderHT }
func (*HTProvider) Version() string { return HTProviderVersion }
func (*HTProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		Kinds: []string{"2d", "3d"}, SupportsViewer: true, SupportsImport: true, SupportsExport: true, MaxFileSizeBytes: MaxFileSize,
		Assets: []AssetCapability{
			{Type: AssetImage, Kinds: []string{"2d", "3d"}, Extensions: []string{".png", ".jpg", ".jpeg", ".webp", ".gif"}},
			{Type: AssetFont, Kinds: []string{"2d", "3d"}, Extensions: []string{".woff", ".woff2"}},
			{Type: AssetModel, Kinds: []string{"3d"}, Extensions: []string{".obj", ".mtl", ".zip"}},
			{Type: AssetMaterial, Kinds: []string{"3d"}, Extensions: []string{".json", ".zip"}},
			{Type: AssetSymbol, Kinds: []string{"2d"}, Extensions: []string{".json", ".zip"}, Editable: true},
			{Type: AssetComponent, Kinds: []string{"2d"}, Extensions: []string{".json", ".zip"}, Editable: true},
		},
	}
}

func (*HTProvider) ValidateEntry(kind, entry string) error {
	entry, err := NormalizePath(entry, false)
	if err != nil || !strings.HasSuffix(strings.ToLower(entry), ".json") {
		return ErrInvalidPath
	}
	prefix := "displays/"
	if kind == "3d" {
		prefix = "scenes/"
	} else if kind != "2d" {
		return ErrInvalidKind
	}
	if !strings.HasPrefix(entry, prefix) {
		return fmt.Errorf("%w: %s 入口必须位于 %s", ErrInvalidPath, kind, prefix)
	}
	return nil
}

func (*HTProvider) ValidateFile(filename string, content []byte) error {
	if int64(len(content)) > MaxFileSize {
		return ErrFileTooLarge
	}
	if strings.EqualFold(path.Ext(filename), ".json") && !json.Valid(content) {
		return ErrInvalidJSON
	}
	return nil
}

// ValidateDraftFile 保证平台场景与 HT 根作品一一对应，资源 JSON 仍由独立资源会话管理。
func (provider *HTProvider) ValidateDraftFile(kind, entryPath, filename string, content []byte) error {
	if err := provider.ValidateFile(filename, content); err != nil {
		return err
	}
	root := "displays/"
	if kind == "3d" {
		root = "scenes/"
	} else if kind != "2d" {
		return ErrInvalidKind
	}
	if strings.HasPrefix(filename, root) && strings.EqualFold(path.Ext(filename), ".json") && filename != entryPath {
		return fmt.Errorf("%w: 当前 %s 场景只允许写入根入口 %s", ErrProtectedPath, kind, entryPath)
	}
	return nil
}

var datapointKeyPattern = regexp.MustCompile(`(?i)(?:dataPoint|datapoint|pointId|tagId|bindId|property)`)
var fileReferenceKeyPattern = regexp.MustCompile(`(?i)(?:url|src|image|texture|font|model|material|symbol|component|display|scene|file|path|icon|background)`)

var providerDependencyExtensions = map[string]struct{}{
	".json": {}, ".obj": {}, ".mtl": {}, ".png": {}, ".jpg": {}, ".jpeg": {}, ".webp": {}, ".gif": {},
	".woff": {}, ".woff2": {},
}

type providerMetadataCollector struct {
	refs          map[string]struct{}
	events        []any
	commands      []any
	eventByName   map[string]map[string]any
	commandByName map[string]map[string]any
	requirements  map[string]*DatapointRequirement
}

func newProviderMetadataCollector() *providerMetadataCollector {
	return &providerMetadataCollector{
		refs: map[string]struct{}{}, eventByName: map[string]map[string]any{}, commandByName: map[string]map[string]any{},
		requirements: map[string]*DatapointRequirement{},
		events:       []any{}, commands: []any{},
	}
}

func (collector *providerMetadataCollector) walk(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			switch key {
			case "induforge.bindings":
				if err := collector.collectBindings(child); err != nil {
					return err
				}
			case "induforge.interactions":
				if err := collector.collectInteractions(child); err != nil {
					return err
				}
			}
			if err := collector.walk(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := collector.walk(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func (collector *providerMetadataCollector) collectBindings(value any) error {
	bindingMap, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("%w: induforge.bindings 必须为对象", ErrContractInvalid)
	}
	for _, raw := range bindingMap {
		binding, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if pointPath, ok := binding["pointPath"].(string); ok && strings.TrimSpace(pointPath) != "" {
			pointPath = strings.TrimSpace(pointPath)
			collector.refs[pointPath] = struct{}{}
			requirement := collector.requirements[pointPath]
			if requirement == nil {
				requirement = &DatapointRequirement{Path: pointPath}
				collector.requirements[pointPath] = requirement
			}
			direction, _ := binding["direction"].(string)
			if direction == "" {
				direction = "read"
			}
			readMode, _ := binding["readMode"].(string)
			requirement.Get = requirement.Get || direction != "write"
			requirement.Sub = requirement.Sub || (direction != "write" && readMode == "subscribe")
			requirement.Set = requirement.Set || direction != "read"
			valueType, _ := binding["valueType"].(string)
			valueType = strings.ToLower(strings.TrimSpace(valueType))
			if valueType != "" {
				if requirement.ValueType != "" && requirement.ValueType != valueType {
					return fmt.Errorf("%w: 数据点 %s 的绑定类型不一致", ErrContractInvalid, pointPath)
				}
				requirement.ValueType = valueType
			}
		}
	}
	return nil
}

func (collector *providerMetadataCollector) datapointRequirements() []DatapointRequirement {
	paths := make([]string, 0, len(collector.requirements))
	for pointPath := range collector.requirements {
		paths = append(paths, pointPath)
	}
	sort.Strings(paths)
	result := make([]DatapointRequirement, 0, len(paths))
	for _, pointPath := range paths {
		result = append(result, *collector.requirements[pointPath])
	}
	return result
}

func (collector *providerMetadataCollector) collectInteractions(value any) error {
	interactions, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("%w: induforge.interactions 必须为对象", ErrContractInvalid)
	}
	if values, ok := interactions["events"].([]any); ok {
		for _, raw := range values {
			mapping, ok := raw.(map[string]any)
			if !ok {
				return fmt.Errorf("%w: 事件映射必须为对象", ErrContractInvalid)
			}
			name, _ := mapping["name"].(string)
			if strings.TrimSpace(name) == "" {
				continue
			}
			member := map[string]any{
				"name":        strings.TrimSpace(name),
				"description": providerEventDescription(mapping),
				"schema":      providerEventSchema(mapping),
			}
			if err := collector.addManagedMember("事件", member, collector.eventByName, &collector.events); err != nil {
				return err
			}
		}
	}
	if values, ok := interactions["commands"].([]any); ok {
		for _, raw := range values {
			mapping, ok := raw.(map[string]any)
			if !ok {
				return fmt.Errorf("%w: 命令映射必须为对象", ErrContractInvalid)
			}
			name, _ := mapping["name"].(string)
			if strings.TrimSpace(name) == "" {
				continue
			}
			member := map[string]any{
				"name":         strings.TrimSpace(name),
				"description":  providerCommandDescription(mapping),
				"inputSchema":  providerCommandInputSchema(mapping),
				"outputSchema": map[string]any{"type": "null"},
			}
			if err := collector.addManagedMember("命令", member, collector.commandByName, &collector.commands); err != nil {
				return err
			}
		}
	}
	return nil
}

func (collector *providerMetadataCollector) addManagedMember(kind string, member map[string]any, existing map[string]map[string]any, target *[]any) error {
	name := member["name"].(string)
	if previous, ok := existing[name]; ok {
		previousJSON, _ := json.Marshal(previous)
		currentJSON, _ := json.Marshal(member)
		if !bytes.Equal(previousJSON, currentJSON) {
			return fmt.Errorf("%w: Provider %s %q 的 Schema 冲突", ErrContractInvalid, kind, name)
		}
		return nil
	}
	existing[name] = member
	*target = append(*target, member)
	return nil
}

func providerEventDescription(mapping map[string]any) string {
	trigger, _ := mapping["trigger"].(string)
	return "场景对象触发：" + strings.TrimSpace(trigger)
}

func providerEventSchema(mapping map[string]any) map[string]any {
	trigger, _ := mapping["trigger"].(string)
	switch strings.TrimSpace(trigger) {
	case "change", "input":
		valueType, _ := mapping["valueType"].(string)
		if valueType != "string" && valueType != "number" && valueType != "integer" && valueType != "boolean" {
			valueType = "string"
		}
		return map[string]any{"type": "object", "properties": map[string]any{"value": map[string]any{"type": valueType}}, "required": []any{"value"}, "additionalProperties": false}
	case "flowStateChange":
		return map[string]any{"type": "object", "properties": map[string]any{
			"mode":    map[string]any{"type": "string", "enum": []any{"off", "continuous", "segment", "particle"}},
			"running": map[string]any{"type": "boolean"}, "reverse": map[string]any{"type": "boolean"}, "speed": map[string]any{"type": "number"},
		}, "required": []any{"mode", "running", "reverse", "speed"}, "additionalProperties": false}
	default:
		return map[string]any{"type": "object", "additionalProperties": false}
	}
}

func providerCommandDescription(mapping map[string]any) string {
	action, _ := mapping["action"].(string)
	return "场景对象操作：" + strings.TrimSpace(action)
}

func providerCommandInputSchema(mapping map[string]any) map[string]any {
	action, _ := mapping["action"].(string)
	propertyType := "boolean"
	propertyName := "value"
	switch strings.TrimSpace(action) {
	case "setValue":
		propertyType = "number"
		if valueType, ok := mapping["valueType"].(string); ok && (valueType == "string" || valueType == "number" || valueType == "integer" || valueType == "boolean") {
			propertyType = valueType
		}
	case "setPipeState":
		return map[string]any{"type": "object", "properties": map[string]any{
			"mode":    map[string]any{"type": "string", "enum": []any{"off", "continuous", "segment", "particle"}},
			"running": map[string]any{"type": "boolean"}, "reverse": map[string]any{"type": "boolean"}, "speed": map[string]any{"type": "number"},
		}, "additionalProperties": false}
	case "show", "hide", "enable", "disable", "focus", "highlight":
		return map[string]any{"type": "object", "additionalProperties": false}
	default:
		propertyName = "value"
	}
	return map[string]any{"type": "object", "properties": map[string]any{propertyName: map[string]any{"type": propertyType}}, "required": []any{propertyName}, "additionalProperties": false}
}

func (provider *HTProvider) Analyze(_ context.Context, kind, entry string, files map[string]ProviderFile) (Analysis, error) {
	if _, ok := files[entry]; !ok {
		return Analysis{}, ErrEntryMissing
	}
	dependencies := map[string]struct{}{entry: {}}
	refs := map[string]struct{}{}
	queue := []string{entry}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		file := files[current]
		switch strings.ToLower(path.Ext(current)) {
		case ".json":
			var value any
			decoder := json.NewDecoder(bytes.NewReader(file.Content))
			decoder.UseNumber()
			if err := decoder.Decode(&value); err != nil {
				return Analysis{}, ErrInvalidJSON
			}
			if err := walkHTValue(value, path.Dir(current), files, dependencies, refs, &queue, ""); err != nil {
				return Analysis{}, err
			}
		case ".obj":
			if err := analyzeOBJDependencies(current, file.Content, files, dependencies, &queue); err != nil {
				return Analysis{}, err
			}
		case ".mtl":
			if err := analyzeMTLDependencies(current, file.Content, files, dependencies, &queue); err != nil {
				return Analysis{}, err
			}
		}
	}
	dependencyList := mapKeys(dependencies)
	metadata, err := provider.ExtractMetadata(files[entry].Content)
	if err != nil {
		return Analysis{}, err
	}
	return Analysis{Dependencies: dependencyList, DatapointRefs: metadata.DatapointRefs,
		DatapointRequirements: metadata.DatapointRequirements, ManagedContract: metadata.ManagedContract}, nil
}

// ExtractMetadata 仅识别平台定义的显式绑定和交互结构，避免把 HT 私有字段误判为数据点。
func (*HTProvider) ExtractMetadata(content []byte) (Analysis, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return Analysis{}, ErrInvalidJSON
	}
	collector := newProviderMetadataCollector()
	if err := collector.walk(value); err != nil {
		return Analysis{}, err
	}
	contract, err := validatePublicContract(map[string]any{
		"description": "", "parameters": []any{}, "events": collector.events, "commands": collector.commands,
	})
	if err != nil {
		return Analysis{}, err
	}
	return Analysis{DatapointRefs: mapKeys(collector.refs), DatapointRequirements: collector.datapointRequirements(), ManagedContract: contract}, nil
}

var htAssetExtensions = map[AssetType]map[string]struct{}{
	AssetImage:     {".png": {}, ".jpg": {}, ".jpeg": {}, ".webp": {}, ".gif": {}},
	AssetFont:      {".woff": {}, ".woff2": {}},
	AssetModel:     {".obj": {}, ".mtl": {}, ".png": {}, ".jpg": {}, ".jpeg": {}, ".webp": {}, ".gif": {}},
	AssetMaterial:  {".json": {}, ".png": {}, ".jpg": {}, ".jpeg": {}, ".webp": {}, ".gif": {}},
	AssetSymbol:    {".json": {}, ".png": {}, ".jpg": {}, ".jpeg": {}, ".webp": {}, ".gif": {}},
	AssetComponent: {".json": {}, ".png": {}, ".jpg": {}, ".jpeg": {}, ".webp": {}, ".gif": {}},
}

func (*HTProvider) AnalyzeAsset(_ context.Context, assetType AssetType, files map[string]ProviderFile) (AssetAnalysis, error) {
	allowed, ok := htAssetExtensions[assetType]
	if !ok {
		return AssetAnalysis{}, ErrInvalidAssetType
	}
	if len(files) == 0 {
		return AssetAnalysis{}, ErrInvalidArchive
	}
	paths := make([]string, 0, len(files))
	for filename, file := range files {
		extension := strings.ToLower(path.Ext(filename))
		if _, valid := allowed[extension]; !valid {
			return AssetAnalysis{}, fmt.Errorf("%w: %s", ErrInvalidPath, filename)
		}
		if extension == ".json" && !json.Valid(file.Content) {
			return AssetAnalysis{}, ErrInvalidJSON
		}
		paths = append(paths, filename)
	}
	sort.Strings(paths)
	entry := ""
	wanted := map[AssetType]string{AssetModel: ".obj", AssetMaterial: ".json", AssetSymbol: ".json", AssetComponent: ".json"}[assetType]
	if wanted == "" {
		if len(paths) != 1 {
			return AssetAnalysis{}, ErrInvalidArchive
		}
		entry = paths[0]
	} else if wanted == ".json" {
		if _, generatedSelection := files["selection.json"]; generatedSelection {
			entry = "selection.json"
		}
		for _, filename := range paths {
			if entry == "selection.json" || !strings.EqualFold(path.Ext(filename), wanted) {
				continue
			}
			if entry != "" {
				return AssetAnalysis{}, fmt.Errorf("%w: 资源包只能包含一个入口文件", ErrInvalidArchive)
			}
			entry = filename
		}
		if entry == "" {
			return AssetAnalysis{}, fmt.Errorf("%w: 缺少 %s 入口", ErrInvalidArchive, wanted)
		}
	} else {
		for _, filename := range paths {
			if strings.EqualFold(path.Ext(filename), wanted) {
				if entry != "" {
					return AssetAnalysis{}, fmt.Errorf("%w: 资源包只能包含一个入口文件", ErrInvalidArchive)
				}
				entry = filename
			}
		}
		if entry == "" {
			return AssetAnalysis{}, fmt.Errorf("%w: 缺少 %s 入口", ErrInvalidArchive, wanted)
		}
	}
	dependencies := map[string]struct{}{entry: {}}
	queue := []string{entry}
	refs := map[string]struct{}{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		file := files[current]
		switch strings.ToLower(path.Ext(current)) {
		case ".json":
			var value any
			decoder := json.NewDecoder(bytes.NewReader(file.Content))
			decoder.UseNumber()
			if err := decoder.Decode(&value); err != nil {
				return AssetAnalysis{}, ErrInvalidJSON
			}
			if err := walkHTValue(value, path.Dir(current), files, dependencies, refs, &queue, ""); err != nil {
				return AssetAnalysis{}, err
			}
		case ".obj":
			if err := analyzeOBJDependencies(current, file.Content, files, dependencies, &queue); err != nil {
				return AssetAnalysis{}, err
			}
		case ".mtl":
			if err := analyzeMTLDependencies(current, file.Content, files, dependencies, &queue); err != nil {
				return AssetAnalysis{}, err
			}
		}
	}
	// 资源代次保存完整上传包；依赖闭包校验用于拒绝引用缺失，不丢弃入口未直接引用的附属文件。
	compatibleKind := "both"
	if assetType == AssetModel || assetType == AssetMaterial {
		compatibleKind = "3d"
	} else if assetType == AssetSymbol || assetType == AssetComponent {
		compatibleKind = "2d"
	}
	return AssetAnalysis{EntryPath: entry, CompatibleKind: compatibleKind, Dependencies: paths,
		Manifest: map[string]any{"fileCount": len(paths), "type": string(assetType)}}, nil
}

func (*HTProvider) MountPath(asset SceneAsset) string {
	root := map[AssetType]string{
		AssetImage: "assets", AssetFont: "assets", AssetModel: "models", AssetMaterial: "materials",
		AssetSymbol: "symbols", AssetComponent: "components",
	}[asset.Type]
	return path.Join(root, "library", asset.ID)
}

func walkHTValue(value any, base string, files map[string]ProviderFile, dependencies, refs map[string]struct{}, queue *[]string, key string) error {
	switch typed := value.(type) {
	case map[string]any:
		for childKey, child := range typed {
			if err := walkHTValue(child, base, files, dependencies, refs, queue, childKey); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := walkHTValue(child, base, files, dependencies, refs, queue, key); err != nil {
				return err
			}
		}
	case string:
		trimmed := strings.TrimSpace(typed)
		if datapointKeyPattern.MatchString(key) && trimmed != "" {
			refs[trimmed] = struct{}{}
		}
		candidate := strings.ReplaceAll(strings.TrimPrefix(trimmed, "./"), `\`, "/")
		if strings.Contains(candidate, "://") || strings.HasPrefix(candidate, "data:") || candidate == "" {
			return nil
		}
		resolved := candidate
		if _, exists := files[resolved]; !exists && base != "." {
			var err error
			resolved, err = NormalizePath(path.Join(base, candidate), false)
			if err != nil {
				return err
			}
		}
		if _, exists := files[resolved]; exists {
			if _, seen := dependencies[resolved]; !seen {
				dependencies[resolved] = struct{}{}
				*queue = append(*queue, resolved)
			}
		} else if fileReferenceKeyPattern.MatchString(key) {
			if _, supported := providerDependencyExtensions[strings.ToLower(path.Ext(resolved))]; supported {
				return fmt.Errorf("%w: %s", ErrDependencyMissing, resolved)
			}
		}
	}
	return nil
}

func analyzeOBJDependencies(filename string, content []byte, files map[string]ProviderFile, dependencies map[string]struct{}, queue *[]string) error {
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 || !strings.EqualFold(fields[0], "mtllib") {
			continue
		}
		for _, reference := range fields[1:] {
			if err := addProviderDependency(filename, reference, files, dependencies, queue); err != nil {
				return err
			}
		}
	}
	return nil
}

func analyzeMTLDependencies(filename string, content []byte, files map[string]ProviderFile, dependencies map[string]struct{}, queue *[]string) error {
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 {
			continue
		}
		command := strings.ToLower(fields[0])
		if command != "map_ka" && command != "map_kd" && command != "map_ks" && command != "map_d" &&
			command != "map_bump" && command != "bump" && command != "disp" && command != "decal" && command != "refl" {
			continue
		}
		// MTL 贴图参数位于文件名之前，取最后一个字段覆盖 HT 常用导出格式。
		if err := addProviderDependency(filename, fields[len(fields)-1], files, dependencies, queue); err != nil {
			return err
		}
	}
	return nil
}

func addProviderDependency(source, reference string, files map[string]ProviderFile, dependencies map[string]struct{}, queue *[]string) error {
	reference = strings.TrimSpace(strings.Trim(reference, `"'`))
	if reference == "" || strings.Contains(reference, "://") || strings.HasPrefix(reference, "data:") {
		return nil
	}
	candidate, err := NormalizePath(path.Join(path.Dir(source), strings.ReplaceAll(reference, `\`, "/")), false)
	if err != nil {
		return err
	}
	if _, exists := files[candidate]; !exists {
		return fmt.Errorf("%w: %s", ErrDependencyMissing, candidate)
	}
	if _, seen := dependencies[candidate]; !seen {
		dependencies[candidate] = struct{}{}
		*queue = append(*queue, candidate)
	}
	return nil
}

func mapKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func (*HTProvider) EditorURL(session EditorSession) string {
	entry := "index.html"
	if session.Kind == "3d" {
		entry = "index3d.html"
	}
	query := url.Values{}
	query.Set("sessionId", session.ID)
	query.Set("open", session.EntryPath)
	query.Set("sceneName", session.SceneName)
	return "/scene-studio/" + entry + "?" + query.Encode()
}

func (*HTProvider) AssetEditorURL(session AssetEditorSession) string {
	return "/scene-studio/index.html?assetSessionId=" + session.ID
}

func (*HTProvider) ViewerURL(session ViewerSession) string {
	entry := "display.html"
	if session.Kind == "3d" {
		entry = "scene.html"
	}
	query := url.Values{}
	query.Set("viewerSessionId", session.ID)
	return "/scene-studio/" + entry + "?" + query.Encode()
}
