package scenecontract

// Contract 是 2D/3D 场景提供给工程页面、AI 上下文和 Runtime SDK 的稳定公开接口。
type Contract struct {
	ID              string      `json:"id"`
	Kind            string      `json:"kind"`
	Name            string      `json:"name"`
	Description     string      `json:"description,omitempty"`
	Parameters      []Parameter `json:"parameters,omitempty"`
	Events          []Member    `json:"events,omitempty"`
	Commands        []Command   `json:"commands,omitempty"`
	DatapointRefs   []string    `json:"datapointRefs,omitempty"`
	ContractVersion string      `json:"contractVersion"`
}

type Member struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Schema      map[string]any `json:"schema,omitempty"`
}

type Parameter struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Required    bool           `json:"required"`
	Schema      map[string]any `json:"schema"`
}

type Command struct {
	Name         string         `json:"name"`
	Description  string         `json:"description,omitempty"`
	InputSchema  map[string]any `json:"inputSchema"`
	OutputSchema map[string]any `json:"outputSchema"`
}

type Snapshot struct {
	ContractVersion string     `json:"contractVersion"`
	Contracts       []Contract `json:"contracts"`
}
