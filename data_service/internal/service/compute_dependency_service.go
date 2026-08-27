package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/indu-forge/data_service/internal/auth"
	enginecompute "github.com/indu-forge/data_service/internal/engine/compute"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

var (
	jsRequirePattern = regexp.MustCompile(`(?m)\brequire\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	jsImportPattern  = regexp.MustCompile(`(?m)\bfrom\s+['"]([^'"]+)['"]|\bimport\s+['"]([^'"]+)['"]`)
	pyImportPattern  = regexp.MustCompile(`(?m)^\s*(?:from|import)\s+([A-Za-z_][A-Za-z0-9_.]*)`)
)

type InstallComputeDependencyInput struct {
	Language    string `json:"language"`
	PackageName string `json:"packageName"`
	Version     string `json:"version"`
}

type ImportComputeDependencyInput struct {
	Language string
	Filename string
	Content  []byte
}

func (s *ComputeService) InstallComputeDependency(ctx context.Context, claims *auth.Claims, projectID string, input InstallComputeDependencyInput) (*ComputeDependency, error) {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return nil, err
	}
	input.Language = strings.ToLower(strings.TrimSpace(input.Language))
	input.PackageName = strings.TrimSpace(input.PackageName)
	input.Version = strings.TrimSpace(input.Version)
	if input.Language != "js" && input.Language != "python" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "依赖语言仅支持 JavaScript 或 Python")
	}
	if input.PackageName == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "包名不能为空")
	}
	manager, ok := s.nodeRunner.(enginecompute.DependencyManager)
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusServiceUnavailable, "计算沙箱不支持依赖安装")
	}
	requested := enginecompute.RuntimeDependency{Language: input.Language, PackageName: input.PackageName, Version: input.Version}
	runtimeDependency, err := manager.InstallDependency(ctx, enginecompute.DependencyInstallRequest{ProjectID: projectID, RuntimeDependency: requested})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "安装工程依赖失败："+err.Error(), err)
	}
	record, err := s.repository.SaveDependency(ctx, repository.SaveComputeDependencyParams{ProjectID: projectID, UserID: claims.UserID, Language: runtimeDependency.Language, PackageName: runtimeDependency.PackageName, ImportName: runtimeDependency.ImportName, Version: runtimeDependency.Version})
	if err != nil {
		_ = manager.UninstallDependency(context.WithoutCancel(ctx), enginecompute.DependencyInstallRequest{ProjectID: projectID, RuntimeDependency: runtimeDependency})
		return nil, err
	}
	item := computeDependencyFromRecord(*record, 0)
	return &item, nil
}

func (s *ComputeService) ImportComputeDependency(ctx context.Context, claims *auth.Claims, projectID string, input ImportComputeDependencyInput) (*ComputeDependency, error) {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return nil, err
	}
	input.Language = strings.ToLower(strings.TrimSpace(input.Language))
	input.Filename = strings.TrimSpace(input.Filename)
	if input.Language != "js" && input.Language != "python" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "依赖语言仅支持 JavaScript 或 Python")
	}
	if input.Filename == "" || len(input.Content) == 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请选择离线依赖文件")
	}
	manager, ok := s.nodeRunner.(enginecompute.DependencyManager)
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusServiceUnavailable, "计算沙箱不支持依赖导入")
	}
	runtimeDependency, err := manager.ImportDependency(ctx, enginecompute.DependencyImportRequest{ProjectID: projectID, Language: input.Language, Filename: input.Filename, Content: input.Content})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "导入工程依赖失败："+err.Error(), err)
	}
	record, err := s.repository.SaveDependency(ctx, repository.SaveComputeDependencyParams{ProjectID: projectID, UserID: claims.UserID, Language: runtimeDependency.Language, PackageName: runtimeDependency.PackageName, ImportName: runtimeDependency.ImportName, Version: runtimeDependency.Version})
	if err != nil {
		_ = manager.UninstallDependency(context.WithoutCancel(ctx), enginecompute.DependencyInstallRequest{ProjectID: projectID, RuntimeDependency: runtimeDependency})
		return nil, err
	}
	item := computeDependencyFromRecord(*record, 0)
	return &item, nil
}

func (s *ComputeService) UninstallComputeDependency(ctx context.Context, claims *auth.Claims, projectID, dependencyID string) error {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return err
	}
	record, err := s.repository.GetDependency(ctx, projectID, dependencyID)
	if err != nil {
		return err
	}
	references, err := s.repository.CountDependencyReferences(ctx, projectID, dependencyID)
	if err != nil {
		return err
	}
	if references > 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("仍有 %d 个计算单元使用该依赖，请先移除代码引用", references))
	}
	manager, ok := s.nodeRunner.(enginecompute.DependencyManager)
	if !ok {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusServiceUnavailable, "计算沙箱不支持依赖卸载")
	}
	request := enginecompute.DependencyInstallRequest{ProjectID: projectID, RuntimeDependency: runtimeDependencyFromRecord(*record)}
	if err := manager.UninstallDependency(ctx, request); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "卸载工程依赖失败："+err.Error(), err)
	}
	return s.repository.DeleteDependency(ctx, projectID, dependencyID)
}

func (s *ComputeService) listInstalledComputeDependencies(ctx context.Context, projectID string) ([]repository.ComputeDependencyRecord, error) {
	return s.repository.ListDependencies(ctx, projectID)
}

func (s *ComputeService) detectComputeDependencies(ctx context.Context, projectID, language, script string) ([]any, error) {
	installed, err := s.listInstalledComputeDependencies(ctx, projectID)
	if err != nil {
		return nil, err
	}
	requested := detectedImports(language, script)
	byImport := make(map[string]repository.ComputeDependencyRecord, len(installed))
	for _, item := range installed {
		if item.Language == language {
			byImport[item.ImportName] = item
		}
	}
	result := make([]any, 0, len(requested))
	for _, name := range requested {
		item, ok := byImport[name]
		if !ok {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "代码引用了尚未安装的工程依赖："+name)
		}
		result = append(result, map[string]any{"id": item.ID, "language": item.Language, "packageName": item.PackageName, "importName": item.ImportName, "version": item.Version})
	}
	return result, nil
}

func detectedImports(language, script string) []string {
	seen := map[string]struct{}{}
	add := func(name string) {
		name = strings.TrimSpace(name)
		if language == "python" {
			name = strings.Split(name, ".")[0]
		}
		if name != "" && !strings.HasPrefix(name, ".") {
			seen[name] = struct{}{}
		}
	}
	if language == "python" {
		for _, match := range pyImportPattern.FindAllStringSubmatch(script, -1) {
			add(match[1])
		}
	} else {
		for _, match := range jsRequirePattern.FindAllStringSubmatch(script, -1) {
			add(match[1])
		}
		for _, match := range jsImportPattern.FindAllStringSubmatch(script, -1) {
			name := match[1]
			if name == "" {
				name = match[2]
			}
			add(name)
		}
	}
	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func runtimeDependencyFromRecord(item repository.ComputeDependencyRecord) enginecompute.RuntimeDependency {
	return enginecompute.RuntimeDependency{Language: item.Language, PackageName: item.PackageName, ImportName: item.ImportName, Version: item.Version}
}

func computeDependencyFromRecord(item repository.ComputeDependencyRecord, references int) ComputeDependency {
	return ComputeDependency{ID: item.ID, Name: item.PackageName, Runtime: computeLanguageForFrontend(item.Language), Version: item.Version, Description: "工程依赖", Status: "installed", ImportName: item.ImportName, ReferenceCount: references}
}

func runtimeDependenciesFromUnit(items []any) []enginecompute.RuntimeDependency {
	result := make([]enginecompute.RuntimeDependency, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, enginecompute.RuntimeDependency{Language: toString(item["language"]), PackageName: toString(item["packageName"]), ImportName: toString(item["importName"]), Version: toString(item["version"])})
	}
	return result
}
