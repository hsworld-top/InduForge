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
	Dependencies  []string
	DatapointRefs []string
}

type SceneProvider interface {
	Name() string
	Version() string
	Capabilities() ProviderCapabilities
	ValidateEntry(kind, entry string) error
	ValidateFile(path string, content []byte) error
	ValidateDraftFile(kind, entryPath, filename string, content []byte) error
	Analyze(ctx context.Context, kind, entry string, files map[string]ProviderFile) (Analysis, error)
	AnalyzeAsset(ctx context.Context, assetType AssetType, files map[string]ProviderFile) (AssetAnalysis, error)
	MountPath(asset SceneAsset) string
	EditorURL(session EditorSession) string
	AssetEditorURL(session AssetEditorSession) string
	ViewerURL(scene Scene, revision Revision) string
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

func (*HTProvider) Analyze(_ context.Context, kind, entry string, files map[string]ProviderFile) (Analysis, error) {
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
	refList := mapKeys(refs)
	return Analysis{Dependencies: dependencyList, DatapointRefs: refList}, nil
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
	return "/designer/scene-studio/" + entry + "?" + query.Encode()
}

func (*HTProvider) AssetEditorURL(session AssetEditorSession) string {
	return "/designer/scene-studio/index.html?assetSessionId=" + session.ID
}

func (*HTProvider) ViewerURL(scene Scene, revision Revision) string {
	return fmt.Sprintf("/api/v1/projects/%s/scenes/%s/viewer?kind=%s&revision=%d", scene.ProjectID, scene.SceneID, scene.Kind, revision.Revision)
}
