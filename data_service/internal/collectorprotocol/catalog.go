package collectorprotocol

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

const jsonSchema202012 = "https://json-schema.org/draft/2020-12/schema"

var allowedDataTypes = map[string]struct{}{
	"bool": {}, "int8": {}, "uint8": {}, "int16": {}, "uint16": {}, "int32": {}, "uint32": {},
	"int64": {}, "uint64": {}, "float32": {}, "float64": {}, "decimal": {}, "string": {}, "bytes": {}, "datetime": {},
}

var allowedOperations = map[string]struct{}{
	"connection.test": {}, "connection.open": {}, "connection.close": {}, "device.browse": {}, "point.read": {}, "point.write": {}, "point.subscribe.preview": {},
}

var allowedFeatures = map[string]struct{}{
	"point.elementCount": {},
}

var allowedExtensions = map[string]struct{}{
	"x-induforge-secret": {}, "x-induforge-sensitive-log": {}, "x-induforge-unit": {},
	"x-induforge-advanced": {}, "x-induforge-address-text": {}, "x-induforge-enum-labels": {},
}

type Manifest struct {
	ProtocolFamily   string              `json:"protocolFamily"`
	DriverID         string              `json:"driverId"`
	DriverVersion    string              `json:"driverVersion"`
	SchemaVersion    int                 `json:"schemaVersion"`
	DisplayName      string              `json:"displayName"`
	AddressHelper    string              `json:"addressHelper,omitempty"`
	Category         string              `json:"category"`
	Transports       []string            `json:"transports"`
	Operations       []string            `json:"operations"`
	Features         []string            `json:"features"`
	DataTypes        []string            `json:"dataTypes"`
	AcquisitionModes []string            `json:"acquisitionModes"`
	Platforms        map[string][]string `json:"platforms"`
}

type DriverDefinition struct {
	Manifest         Manifest
	ConnectionSchema json.RawMessage
	AddressSchema    json.RawMessage
	UISchema         json.RawMessage
}

type Catalog struct {
	drivers map[string]DriverDefinition
}

func LoadCatalog(source fs.FS) (*Catalog, error) {
	entries, err := fs.ReadDir(source, ".")
	if err != nil {
		return nil, fmt.Errorf("读取工业采集协议目录失败: %w", err)
	}
	drivers := make(map[string]DriverDefinition)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		definition, err := loadDriver(source, entry.Name())
		if err != nil {
			return nil, err
		}
		if _, exists := drivers[definition.Manifest.DriverID]; exists {
			return nil, fmt.Errorf("驱动标识重复: %s", definition.Manifest.DriverID)
		}
		drivers[definition.Manifest.DriverID] = definition
	}
	return &Catalog{drivers: drivers}, nil
}

func (c *Catalog) Driver(driverID string) (DriverDefinition, bool) {
	driver, ok := c.drivers[driverID]
	return driver, ok
}

func (c *Catalog) Drivers() []DriverDefinition {
	result := make([]DriverDefinition, 0, len(c.drivers))
	for _, driver := range c.drivers {
		result = append(result, driver)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Manifest.DriverID < result[j].Manifest.DriverID })
	return result
}

func loadDriver(source fs.FS, directory string) (DriverDefinition, error) {
	manifestPayload, err := readContractFile(source, directory+"/manifest.json")
	if err != nil {
		return DriverDefinition{}, err
	}
	if err := rejectForbiddenPublicTerms(manifestPayload); err != nil {
		return DriverDefinition{}, fmt.Errorf("驱动 %s: %w", directory, err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestPayload, &manifest); err != nil {
		return DriverDefinition{}, fmt.Errorf("解析驱动 %s Manifest 失败: %w", directory, err)
	}
	if manifest.DriverID != directory {
		return DriverDefinition{}, fmt.Errorf("驱动目录名必须等于 driverId: directory=%s driverId=%s", directory, manifest.DriverID)
	}
	if err := validateManifest(manifest); err != nil {
		return DriverDefinition{}, fmt.Errorf("驱动 %s Manifest 无效: %w", directory, err)
	}
	connectionSchema, err := readAndValidateSchema(source, directory+"/connection.schema.json")
	if err != nil {
		return DriverDefinition{}, err
	}
	addressSchema, err := readAndValidateSchema(source, directory+"/address.schema.json")
	if err != nil {
		return DriverDefinition{}, err
	}
	uiSchema, err := readContractFile(source, directory+"/ui.schema.json")
	if err != nil {
		return DriverDefinition{}, err
	}
	if err := validateJSONDocument(uiSchema, directory+"/ui.schema.json"); err != nil {
		return DriverDefinition{}, err
	}
	if err := validateAddressHelper(manifest, uiSchema); err != nil {
		return DriverDefinition{}, fmt.Errorf("驱动 %s UI Schema 无效: %w", directory, err)
	}
	return DriverDefinition{Manifest: manifest, ConnectionSchema: connectionSchema, AddressSchema: addressSchema, UISchema: uiSchema}, nil
}

func validateManifest(manifest Manifest) error {
	if strings.TrimSpace(manifest.ProtocolFamily) == "" || strings.TrimSpace(manifest.DriverID) == "" || strings.TrimSpace(manifest.DriverVersion) == "" || strings.TrimSpace(manifest.DisplayName) == "" || strings.TrimSpace(manifest.Category) == "" {
		return fmt.Errorf("存在空的必填字段")
	}
	if manifest.SchemaVersion < 1 {
		return fmt.Errorf("schemaVersion 必须大于 0")
	}
	for _, operation := range manifest.Operations {
		if _, ok := allowedOperations[operation]; !ok {
			return fmt.Errorf("不支持的操作: %s", operation)
		}
	}
	for _, feature := range manifest.Features {
		if _, ok := allowedFeatures[feature]; !ok {
			return fmt.Errorf("不支持的驱动特性: %s", feature)
		}
	}
	for _, dataType := range manifest.DataTypes {
		if _, ok := allowedDataTypes[dataType]; !ok {
			return fmt.Errorf("不支持的数据类型: %s", dataType)
		}
	}
	if manifest.AddressHelper != "" {
		allowed := map[string]struct{}{"siemens": {}, "modbus": {}, "melsec": {}, "omron": {}, "allen_bradley": {}}
		if _, ok := allowed[manifest.AddressHelper]; !ok {
			return fmt.Errorf("不支持的地址助手: %s", manifest.AddressHelper)
		}
	}
	return nil
}

func validateAddressHelper(manifest Manifest, payload []byte) error {
	var document map[string]any
	if err := json.Unmarshal(payload, &document); err != nil {
		return err
	}
	uiHelper, _ := document["addressHelper"].(string)
	if strings.TrimSpace(uiHelper) != strings.TrimSpace(manifest.AddressHelper) {
		return fmt.Errorf("addressHelper 必须与 Manifest 一致")
	}
	return nil
}

func readAndValidateSchema(source fs.FS, path string) (json.RawMessage, error) {
	payload, err := readContractFile(source, path)
	if err != nil {
		return nil, err
	}
	if err := rejectForbiddenPublicTerms(payload); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var document map[string]any
	if err := json.Unmarshal(payload, &document); err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", path, err)
	}
	if document["$schema"] != jsonSchema202012 || document["type"] != "object" {
		return nil, fmt.Errorf("%s 必须使用 JSON Schema 2020-12 且根类型为 object", path)
	}
	if err := validateExtensionKeywords(document); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return payload, nil
}

func validateExtensionKeywords(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if strings.HasPrefix(key, "x-induforge-") {
				if _, ok := allowedExtensions[key]; !ok {
					return fmt.Errorf("不支持的平台扩展关键字: %s", key)
				}
			}
			if err := validateExtensionKeywords(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := validateExtensionKeywords(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func readContractFile(source fs.FS, path string) ([]byte, error) {
	payload, err := fs.ReadFile(source, path)
	if err != nil {
		return nil, fmt.Errorf("读取协议契约 %s 失败: %w", path, err)
	}
	return payload, nil
}

func validateJSONDocument(payload []byte, path string) error {
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		return fmt.Errorf("解析 %s 失败: %w", path, err)
	}
	return nil
}

func rejectForbiddenPublicTerms(payload []byte) error {
	content := strings.ToLower(string(payload))
	for _, forbidden := range []string{"hslcommunication", "hsl ", "system.boolean", "system.int", "ireadwritenet"} {
		if strings.Contains(content, forbidden) {
			return fmt.Errorf("公共协议契约包含禁止词: %s", forbidden)
		}
	}
	return nil
}
