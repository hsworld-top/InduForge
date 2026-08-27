package sceneasset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/indu-forge/dev_core/internal/scenecontract"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const maxContractMembers = 100

var contractMemberNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.:-]{0,63}$`)

type publicContractDefinition struct {
	Description string                    `json:"description"`
	Parameters  []scenecontract.Parameter `json:"parameters"`
	Events      []scenecontract.Member    `json:"events"`
	Commands    []scenecontract.Command   `json:"commands"`
}

func emptyPublicContract() map[string]any {
	return map[string]any{
		"description": "",
		"parameters":  []any{},
		"events":      []any{},
		"commands":    []any{},
	}
}

// validatePublicContract 只接受平台公开的交互字段，并提前编译全部 JSON Schema。
func validatePublicContract(value map[string]any) (map[string]any, error) {
	if value == nil {
		value = emptyPublicContract()
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrContractInvalid, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var contract publicContractDefinition
	if err := decoder.Decode(&contract); err != nil {
		return nil, fmt.Errorf("%w: 仅支持 description、parameters、events 和 commands", ErrContractInvalid)
	}
	if len([]rune(contract.Description)) > 2000 {
		return nil, fmt.Errorf("%w: 场景说明不能超过 2000 个字符", ErrContractInvalid)
	}
	if err := validateParameterMembers(contract.Parameters); err != nil {
		return nil, err
	}
	if err := validateEventMembers(contract.Events); err != nil {
		return nil, err
	}
	if err := validateCommandMembers(contract.Commands); err != nil {
		return nil, err
	}
	normalized, err := json.Marshal(contract)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(normalized, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// mergeSceneContracts 将用户契约与 Provider 生成契约合并为 revision 快照。
// 同名成员只有在 Schema 完全一致时才允许合并，避免页面与场景对同一名称产生不同理解。
func mergeSceneContracts(publicValue, managedValue map[string]any) (map[string]any, error) {
	publicContract, err := decodeContractDefinition(publicValue)
	if err != nil {
		return nil, err
	}
	managedContract, err := decodeContractDefinition(managedValue)
	if err != nil {
		return nil, err
	}

	events := append([]scenecontract.Member{}, publicContract.Events...)
	eventSchemas := make(map[string]map[string]any, len(events))
	for _, member := range events {
		eventSchemas[member.Name] = member.Schema
	}
	for _, member := range managedContract.Events {
		if schema, exists := eventSchemas[member.Name]; exists {
			if !jsonValuesEqual(schema, member.Schema) {
				return nil, fmt.Errorf("%w: 事件 %q 的用户 Schema 与 Provider Schema 冲突", ErrContractInvalid, member.Name)
			}
			continue
		}
		eventSchemas[member.Name] = member.Schema
		events = append(events, member)
	}

	commands := append([]scenecontract.Command{}, publicContract.Commands...)
	commandSchemas := make(map[string]scenecontract.Command, len(commands))
	for _, command := range commands {
		commandSchemas[command.Name] = command
	}
	for _, command := range managedContract.Commands {
		if previous, exists := commandSchemas[command.Name]; exists {
			if !jsonValuesEqual(previous.InputSchema, command.InputSchema) || !jsonValuesEqual(previous.OutputSchema, command.OutputSchema) {
				return nil, fmt.Errorf("%w: 命令 %q 的用户 Schema 与 Provider Schema 冲突", ErrContractInvalid, command.Name)
			}
			continue
		}
		commandSchemas[command.Name] = command
		commands = append(commands, command)
	}

	return validatePublicContract(contractDefinitionMap(publicContract.Description, publicContract.Parameters, events, commands))
}

func decodeContractDefinition(value map[string]any) (publicContractDefinition, error) {
	normalized, err := validatePublicContract(value)
	if err != nil {
		return publicContractDefinition{}, err
	}
	payload, _ := json.Marshal(normalized)
	var result publicContractDefinition
	if err := json.Unmarshal(payload, &result); err != nil {
		return publicContractDefinition{}, err
	}
	return result, nil
}

func contractDefinitionMap(description string, parameters []scenecontract.Parameter, events []scenecontract.Member, commands []scenecontract.Command) map[string]any {
	payload, _ := json.Marshal(publicContractDefinition{Description: description, Parameters: parameters, Events: events, Commands: commands})
	var result map[string]any
	_ = json.Unmarshal(payload, &result)
	return result
}

func jsonValuesEqual(left, right any) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return bytes.Equal(leftJSON, rightJSON)
}

func validateParameterMembers(items []scenecontract.Parameter) error {
	if len(items) > maxContractMembers {
		return fmt.Errorf("%w: 参数最多 %d 项", ErrContractInvalid, maxContractMembers)
	}
	seen := map[string]struct{}{}
	for _, item := range items {
		if err := validateMemberName("参数", item.Name, seen); err != nil {
			return err
		}
		if err := compileContractSchema("参数 "+item.Name, item.Schema); err != nil {
			return err
		}
	}
	return nil
}

func validateEventMembers(items []scenecontract.Member) error {
	if len(items) > maxContractMembers {
		return fmt.Errorf("%w: 事件最多 %d 项", ErrContractInvalid, maxContractMembers)
	}
	seen := map[string]struct{}{}
	for _, item := range items {
		if err := validateMemberName("事件", item.Name, seen); err != nil {
			return err
		}
		if err := compileContractSchema("事件 "+item.Name, item.Schema); err != nil {
			return err
		}
	}
	return nil
}

func validateCommandMembers(items []scenecontract.Command) error {
	if len(items) > maxContractMembers {
		return fmt.Errorf("%w: 命令最多 %d 项", ErrContractInvalid, maxContractMembers)
	}
	seen := map[string]struct{}{}
	for _, item := range items {
		if err := validateMemberName("命令", item.Name, seen); err != nil {
			return err
		}
		if err := compileContractSchema("命令 "+item.Name+" 输入", item.InputSchema); err != nil {
			return err
		}
		if err := compileContractSchema("命令 "+item.Name+" 输出", item.OutputSchema); err != nil {
			return err
		}
	}
	return nil
}

func validateMemberName(kind, name string, seen map[string]struct{}) error {
	name = strings.TrimSpace(name)
	if !contractMemberNamePattern.MatchString(name) {
		return fmt.Errorf("%w: %s名称 %q 格式无效", ErrContractInvalid, kind, name)
	}
	if _, exists := seen[name]; exists {
		return fmt.Errorf("%w: %s名称 %q 重复", ErrContractInvalid, kind, name)
	}
	seen[name] = struct{}{}
	return nil
}

func compileContractSchema(label string, schema map[string]any) error {
	if schema == nil {
		return fmt.Errorf("%w: %s缺少 Schema", ErrContractInvalid, label)
	}
	if containsRemoteSchemaReference(schema) {
		return fmt.Errorf("%w: %s不允许使用 $ref 或 $dynamicRef", ErrContractInvalid, label)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	if err := compiler.AddResource("contract.json", schema); err != nil {
		return fmt.Errorf("%w: %s Schema 无效: %v", ErrContractInvalid, label, err)
	}
	if _, err := compiler.Compile("contract.json"); err != nil {
		return fmt.Errorf("%w: %s Schema 无效: %v", ErrContractInvalid, label, err)
	}
	return nil
}

func containsRemoteSchemaReference(value any) bool {
	switch item := value.(type) {
	case map[string]any:
		for key, child := range item {
			if key == "$ref" || key == "$dynamicRef" || containsRemoteSchemaReference(child) {
				return true
			}
		}
	case []any:
		for _, child := range item {
			if containsRemoteSchemaReference(child) {
				return true
			}
		}
	}
	return false
}
