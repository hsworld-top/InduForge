package scenecontract

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/project"
)

var (
	ErrInvalidContract = errors.New("场景公开契约无效")
	ErrNotFound        = errors.New("场景公开契约不存在")
)

var sceneIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)

type ProjectRepository interface {
	Get(context.Context, string, string) (project.Project, error)
}

// Contract 是 HT 场景提供给 Vue 页面和运行时 SDK 的稳定公开接口，不包含画布私有结构。
type Contract struct {
	ID              string   `json:"id"`
	Kind            string   `json:"kind"`
	Name            string   `json:"name"`
	Description     string   `json:"description,omitempty"`
	Route           string   `json:"route,omitempty"`
	EmbedMode       string   `json:"embedMode"`
	Inputs          []Member `json:"inputs,omitempty"`
	Events          []Member `json:"events,omitempty"`
	Commands        []Member `json:"commands,omitempty"`
	PublicObjects   []Member `json:"publicObjects,omitempty"`
	DatapointRefs   []string `json:"datapointRefs,omitempty"`
	PermissionRefs  []string `json:"permissionRefs,omitempty"`
	ContractVersion string   `json:"contractVersion"`
}

// Member 统一描述场景的输入、事件、命令和公开对象，Schema 为开发工具可读取的 JSON Schema 片段。
type Member struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Schema      map[string]any `json:"schema,omitempty"`
}

type Snapshot struct {
	ContractVersion string     `json:"contractVersion"`
	Contracts       []Contract `json:"contracts"`
}

type Service struct{ projects ProjectRepository }

func NewService(projects ProjectRepository) *Service { return &Service{projects: projects} }

func (s *Service) List(ctx context.Context, actor auth.User, projectID string) (Snapshot, error) {
	workspace, err := s.workspace(ctx, actor, projectID, auth.CapabilityProjectRead)
	if err != nil {
		return Snapshot{}, err
	}
	contracts, err := readContracts(workspace)
	if err != nil {
		return Snapshot{}, err
	}
	return snapshot(contracts), nil
}

func (s *Service) Get(ctx context.Context, actor auth.User, projectID, kind, id string) (Contract, error) {
	workspace, err := s.workspace(ctx, actor, projectID, auth.CapabilityProjectRead)
	if err != nil {
		return Contract{}, err
	}
	return readContract(workspace, kind, id)
}

func (s *Service) Put(ctx context.Context, actor auth.User, projectID, kind, id string, input Contract) (Contract, error) {
	workspace, err := s.workspace(ctx, actor, projectID, auth.CapabilityProjectWrite)
	if err != nil {
		return Contract{}, err
	}
	input.ID = id
	input.Kind = kind
	if err := normalizeAndValidate(&input); err != nil {
		return Contract{}, err
	}
	content, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return Contract{}, fmt.Errorf("序列化场景公开契约失败: %w", err)
	}
	path, err := contractPath(workspace, input.Kind, input.ID)
	if err != nil {
		return Contract{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Contract{}, fmt.Errorf("创建场景契约目录失败: %w", err)
	}
	if err := os.WriteFile(path, append(content, '\n'), 0o644); err != nil {
		return Contract{}, fmt.Errorf("保存场景公开契约失败: %w", err)
	}
	return input, nil
}

func (s *Service) Delete(ctx context.Context, actor auth.User, projectID, kind, id string) error {
	workspace, err := s.workspace(ctx, actor, projectID, auth.CapabilityProjectWrite)
	if err != nil {
		return err
	}
	path, err := contractPath(workspace, kind, id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("删除场景公开契约失败: %w", err)
	}
	return nil
}

func (s *Service) workspace(ctx context.Context, actor auth.User, projectID string, capability auth.Capability) (string, error) {
	item, err := s.projects.Get(ctx, actor.TenantID, projectID)
	if err != nil {
		return "", err
	}
	if err := project.RequireCapability(actor, item, capability); err != nil {
		return "", err
	}
	path, err := filepath.Abs(strings.TrimSpace(item.WorkspacePath))
	if err != nil || path == "" {
		return "", ErrInvalidContract
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", ErrInvalidContract
	}
	return path, nil
}

func readContracts(workspace string) ([]Contract, error) {
	contracts := make([]Contract, 0)
	for _, kind := range []string{"2d", "3d"} {
		directory := filepath.Join(workspace, ".induforge", "scenes", kind)
		entries, err := os.ReadDir(directory)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("读取场景公开契约失败: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			contract, err := readContract(workspace, kind, strings.TrimSuffix(entry.Name(), ".json"))
			if err != nil {
				return nil, err
			}
			contracts = append(contracts, contract)
		}
	}
	sort.Slice(contracts, func(left, right int) bool {
		if contracts[left].Kind != contracts[right].Kind {
			return contracts[left].Kind < contracts[right].Kind
		}
		return contracts[left].ID < contracts[right].ID
	})
	return contracts, nil
}

func readContract(workspace, kind, id string) (Contract, error) {
	path, err := contractPath(workspace, kind, id)
	if err != nil {
		return Contract{}, err
	}
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Contract{}, ErrNotFound
	}
	if err != nil {
		return Contract{}, fmt.Errorf("读取场景公开契约失败: %w", err)
	}
	var contract Contract
	if err := json.Unmarshal(content, &contract); err != nil {
		return Contract{}, fmt.Errorf("解析场景公开契约失败: %w", err)
	}
	if contract.ID != id || contract.Kind != kind || normalizeAndValidate(&contract) != nil {
		return Contract{}, ErrInvalidContract
	}
	return contract, nil
}

func contractPath(workspace, kind, id string) (string, error) {
	if !validKind(kind) || !sceneIDPattern.MatchString(id) {
		return "", ErrInvalidContract
	}
	return filepath.Join(workspace, ".induforge", "scenes", kind, id+".json"), nil
}

func normalizeAndValidate(contract *Contract) error {
	contract.ID = strings.TrimSpace(contract.ID)
	contract.Kind = strings.ToLower(strings.TrimSpace(contract.Kind))
	contract.Name = strings.TrimSpace(contract.Name)
	contract.Description = strings.TrimSpace(contract.Description)
	contract.Route = strings.TrimSpace(contract.Route)
	contract.EmbedMode = strings.ToLower(strings.TrimSpace(contract.EmbedMode))
	if contract.EmbedMode == "" {
		contract.EmbedMode = "both"
	}
	if !sceneIDPattern.MatchString(contract.ID) || !validKind(contract.Kind) || contract.Name == "" || !validEmbedMode(contract.EmbedMode) {
		return ErrInvalidContract
	}
	for _, group := range [][]Member{contract.Inputs, contract.Events, contract.Commands, contract.PublicObjects} {
		if err := validateMembers(group); err != nil {
			return err
		}
	}
	contract.DatapointRefs = normalizeReferences(contract.DatapointRefs)
	contract.PermissionRefs = normalizeReferences(contract.PermissionRefs)
	contract.ContractVersion = calculateVersion(*contract)
	return nil
}

func validateMembers(members []Member) error {
	seen := make(map[string]struct{}, len(members))
	for _, member := range members {
		name := strings.TrimSpace(member.Name)
		if name == "" {
			return ErrInvalidContract
		}
		if _, exists := seen[name]; exists {
			return ErrInvalidContract
		}
		seen[name] = struct{}{}
	}
	return nil
}

func normalizeReferences(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func calculateVersion(contract Contract) string {
	contract.ContractVersion = ""
	content, _ := json.Marshal(contract)
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

func snapshot(contracts []Contract) Snapshot {
	versions := make([]string, 0, len(contracts))
	for _, contract := range contracts {
		versions = append(versions, contract.Kind+":"+contract.ID+":"+contract.ContractVersion)
	}
	hash := sha256.Sum256([]byte(strings.Join(versions, "\n")))
	return Snapshot{ContractVersion: hex.EncodeToString(hash[:]), Contracts: contracts}
}

func validKind(kind string) bool { return kind == "2d" || kind == "3d" }

func validEmbedMode(mode string) bool {
	return mode == "standalone" || mode == "embedded" || mode == "both"
}
