package router

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/http/handler"
	"github.com/indu-forge/data_service/internal/http/middleware"
)

type options struct {
	accessSourceHandler       *handler.AccessSourceHandler
	alarmPolicyHandler        *handler.AlarmPolicyHandler
	alarmRuleHandler          *handler.AlarmRuleHandler
	builtinRuntimeHandler     *handler.BuiltinRuntimeHandler
	connectionHandler         *handler.ConnectionHandler
	contractCheckHandler      *handler.ContractCheckHandler
	queryHandler              *handler.QueryHandler
	workbenchGroupHandler     *handler.WorkbenchGroupHandler
	realtimeStoreHandler      *handler.RealtimeStoreHandler
	dataPointHandler          *handler.DataPointHandler
	modbusModelingHandler     *handler.ModbusModelingHandler
	mqttHandler               *handler.MqttHandler
	kafkaWorkbenchHandler     *handler.KafkaWorkbenchHandler
	httpWorkbenchHandler      *handler.HTTPWorkbenchHandler
	websocketWorkbenchHandler *handler.WebSocketWorkbenchHandler
	opcuaModelingHandler      *handler.OpcuaModelingHandler
	s7ModelingHandler         *handler.S7ModelingHandler
	protocolDevSessionHandler *handler.ProtocolDevSessionHandler
	protocolWave1Handler      *handler.ProtocolWave1Handler
	protocolWave2Handler      *handler.ProtocolWave2Handler
	previewHandler            *handler.PreviewHandler
	previewSocketHandler      http.Handler
	computeHandler            *handler.ComputeHandler
	projectSnapshotHandler    *handler.ProjectSnapshotHandler
	jwtValidator              *auth.JWTValidator
}

// WithBuiltinRuntimeRoutes wires IF builtin runtime store routes.
func WithBuiltinRuntimeRoutes(builtinRuntimeHandler *handler.BuiltinRuntimeHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.builtinRuntimeHandler = builtinRuntimeHandler
		opts.jwtValidator = jwtValidator
	}
}

// Option defines route wiring dependencies.
type Option func(*options)

// WithAlarmRuleRoutes wires alarm rule routes.
func WithAlarmRuleRoutes(alarmRuleHandler *handler.AlarmRuleHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.alarmRuleHandler = alarmRuleHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithAlarmPolicyRoutes wires alarm policy routes.
func WithAlarmPolicyRoutes(alarmPolicyHandler *handler.AlarmPolicyHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.alarmPolicyHandler = alarmPolicyHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithContractCheckRoutes wires contract check routes.
func WithContractCheckRoutes(contractCheckHandler *handler.ContractCheckHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.contractCheckHandler = contractCheckHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithAccessSourceRoutes wires access source routes.
func WithAccessSourceRoutes(accessSourceHandler *handler.AccessSourceHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.accessSourceHandler = accessSourceHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithConnectionRoutes wires connection routes.
func WithConnectionRoutes(connectionHandler *handler.ConnectionHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.connectionHandler = connectionHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithDataRoutes wires query and datapoint routes.
func WithDataRoutes(queryHandler *handler.QueryHandler, dataPointHandler *handler.DataPointHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.queryHandler = queryHandler
		opts.dataPointHandler = dataPointHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithWorkbenchGroupRoutes wires SQL workbench object group routes.
func WithWorkbenchGroupRoutes(workbenchGroupHandler *handler.WorkbenchGroupHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.workbenchGroupHandler = workbenchGroupHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithRealtimeStoreRoutes wires Redis/IF 实时库统一工作台 routes.
func WithRealtimeStoreRoutes(realtimeStoreHandler *handler.RealtimeStoreHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.realtimeStoreHandler = realtimeStoreHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithMqttRoutes wires MQTT routes.
func WithMqttRoutes(mqttHandler *handler.MqttHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.mqttHandler = mqttHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithKafkaWorkbenchRoutes wires Kafka 工作台 routes.
func WithKafkaWorkbenchRoutes(kafkaHandler *handler.KafkaWorkbenchHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.kafkaWorkbenchHandler = kafkaHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithHTTPWorkbenchRoutes wires HTTP 工作台 routes.
func WithHTTPWorkbenchRoutes(httpHandler *handler.HTTPWorkbenchHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.httpWorkbenchHandler = httpHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithWebSocketWorkbenchRoutes wires WebSocket 工作台 routes.
func WithWebSocketWorkbenchRoutes(websocketHandler *handler.WebSocketWorkbenchHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.websocketWorkbenchHandler = websocketHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithModbusModelingRoutes wires Modbus 寄存器建模 routes.
func WithModbusModelingRoutes(modbusModelingHandler *handler.ModbusModelingHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.modbusModelingHandler = modbusModelingHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithOpcuaModelingRoutes wires OPC UA 点位建模 routes.
func WithOpcuaModelingRoutes(opcuaModelingHandler *handler.OpcuaModelingHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.opcuaModelingHandler = opcuaModelingHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithS7ModelingRoutes wires S7 变量建模 routes.
func WithS7ModelingRoutes(s7ModelingHandler *handler.S7ModelingHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.s7ModelingHandler = s7ModelingHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithProtocolDevSessionRoutes wires OPC UA / Modbus 开发态会话 routes.
func WithProtocolDevSessionRoutes(protocolDevSessionHandler *handler.ProtocolDevSessionHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.protocolDevSessionHandler = protocolDevSessionHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithProtocolWave1Routes wires protocol wave 1 routes.
func WithProtocolWave1Routes(protocolWave1Handler *handler.ProtocolWave1Handler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.protocolWave1Handler = protocolWave1Handler
		opts.jwtValidator = jwtValidator
	}
}

// WithProtocolWave2Routes wires protocol wave 2 routes.
func WithProtocolWave2Routes(protocolWave2Handler *handler.ProtocolWave2Handler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.protocolWave2Handler = protocolWave2Handler
		opts.jwtValidator = jwtValidator
	}
}

// WithPreviewRoutes wires preview session routes.
func WithPreviewRoutes(previewHandler *handler.PreviewHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.previewHandler = previewHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithPreviewSocketHandler wires the preview socket transport layer.
func WithPreviewSocketHandler(previewSocketHandler http.Handler) Option {
	return func(opts *options) {
		opts.previewSocketHandler = previewSocketHandler
	}
}

// WithComputeRoutes wires compute routes.
func WithComputeRoutes(computeHandler *handler.ComputeHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.computeHandler = computeHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithProjectSnapshotRoutes wires project snapshot routes.
func WithProjectSnapshotRoutes(projectSnapshotHandler *handler.ProjectSnapshotHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.projectSnapshotHandler = projectSnapshotHandler
		opts.jwtValidator = jwtValidator
	}
}

// NewRouter builds the base HTTP router for data_service.
func NewRouter(routeOptions ...Option) http.Handler {
	opts := options{}
	for _, option := range routeOptions {
		option(&opts)
	}

	mux := http.NewServeMux()
	mux.Handle("/health", middleware.ErrorHandler(func(w http.ResponseWriter, r *http.Request) error {
		handler.NewHealthHandler().ServeHTTP(w, r)
		return nil
	}))

	mountAlarmRuleRoutes(mux, opts)
	mountAlarmPolicyRoutes(mux, opts)
	mountBuiltinRuntimeRoutes(mux, opts)
	mountAccessSourceRoutes(mux, opts)
	mountContractCheckRoutes(mux, opts)
	mountConnectionRoutes(mux, opts)
	mountDataRoutes(mux, opts)
	mountWorkbenchGroupRoutes(mux, opts)
	mountMqttRoutes(mux, opts)
	mountKafkaWorkbenchRoutes(mux, opts)
	mountHTTPWorkbenchRoutes(mux, opts)
	mountWebSocketWorkbenchRoutes(mux, opts)
	mountModbusModelingRoutes(mux, opts)
	mountOpcuaModelingRoutes(mux, opts)
	mountS7ModelingRoutes(mux, opts)
	mountProtocolDevSessionRoutes(mux, opts)
	mountRealtimeStoreRoutes(mux, opts)
	mountProtocolWave1Routes(mux, opts)
	mountProtocolWave2Routes(mux, opts)
	mountPreviewSocketRoutes(mux, opts)
	mountPreviewRoutes(mux, opts)
	mountComputeRoutes(mux, opts)
	mountProjectSnapshotRoutes(mux, opts)
	return mux
}

func mountBuiltinRuntimeRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.builtinRuntimeHandler == nil || opts.jwtValidator == nil {
		return
	}

	read := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}
	write := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}

	mux.Handle("POST /api/v1/data/projects/{projectId}/builtin/relation/sql/execute", read(opts.builtinRuntimeHandler.ExecuteRelationSQL))
	mux.Handle("POST /api/v1/data/projects/{projectId}/builtin/timeseries/query", read(opts.builtinRuntimeHandler.QueryTimeseries))
	mux.Handle("POST /api/v1/data/projects/{projectId}/builtin/timeseries/sample", write(opts.builtinRuntimeHandler.SampleTimeseries))
}

func mountRealtimeStoreRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.realtimeStoreHandler == nil || opts.jwtValidator == nil {
		return
	}
	read := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}
	write := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}

	mux.Handle("GET /api/v1/data/projects/{projectId}/realtime-stores/{connectionId}/keys", read(opts.realtimeStoreHandler.ListKeys))
	mux.Handle("GET /api/v1/data/projects/{projectId}/realtime-stores/{connectionId}/key", read(opts.realtimeStoreHandler.GetKey))
	mux.Handle("PUT /api/v1/data/projects/{projectId}/realtime-stores/{connectionId}/key", write(opts.realtimeStoreHandler.SaveKey))
	mux.Handle("PATCH /api/v1/data/projects/{projectId}/realtime-stores/{connectionId}/key/rename", write(opts.realtimeStoreHandler.RenameKey))
	mux.Handle("DELETE /api/v1/data/projects/{projectId}/realtime-stores/{connectionId}/key", write(opts.realtimeStoreHandler.DeleteKey))
	mux.Handle("POST /api/v1/data/projects/{projectId}/realtime-stores/{connectionId}/key/datapoint", write(opts.realtimeStoreHandler.CreateDataPoint))
	mux.Handle("POST /api/v1/data/projects/{projectId}/realtime-stores/{connectionId}/keys/datapoints", write(opts.realtimeStoreHandler.BatchCreateDataPoints))
}

func mountAlarmPolicyRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.alarmPolicyHandler == nil || opts.jwtValidator == nil {
		return
	}

	read := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}
	write := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}

	mux.Handle("GET /api/v1/data/projects/{projectId}/alarm-policy-groups", read(opts.alarmPolicyHandler.ListGroups))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-policy-groups", write(opts.alarmPolicyHandler.CreateGroup))
	mux.Handle("PUT /api/v1/data/projects/{projectId}/alarm-policy-groups/{id}", write(opts.alarmPolicyHandler.UpdateGroup))
	mux.Handle("PATCH /api/v1/data/projects/{projectId}/alarm-policy-groups/{id}/enabled", write(opts.alarmPolicyHandler.ToggleGroupEnabled))
	mux.Handle("DELETE /api/v1/data/projects/{projectId}/alarm-policy-groups/{id}", write(opts.alarmPolicyHandler.DeleteGroup))

	mux.Handle("GET /api/v1/data/projects/{projectId}/alarm-settings", read(opts.alarmPolicyHandler.GetSettings))
	mux.Handle("PUT /api/v1/data/projects/{projectId}/alarm-settings", write(opts.alarmPolicyHandler.UpdateSettings))

	mux.Handle("GET /api/v1/data/projects/{projectId}/alarm-policies", read(opts.alarmPolicyHandler.List))
	mux.Handle("GET /api/v1/data/projects/{projectId}/alarm-policies/tree", read(opts.alarmPolicyHandler.Tree))
	mux.Handle("GET /api/v1/data/projects/{projectId}/alarm-policies/coverage", read(opts.alarmPolicyHandler.Coverage))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-policies", write(opts.alarmPolicyHandler.Create))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-policies/validate-draft", read(opts.alarmPolicyHandler.ValidateDraft))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-policies/batch-enable", write(opts.alarmPolicyHandler.BatchEnable))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-policies/batch-disable", write(opts.alarmPolicyHandler.BatchDisable))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-policies/batch-move", write(opts.alarmPolicyHandler.BatchMove))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-policies/batch-apply-conditions", write(opts.alarmPolicyHandler.BatchApplyConditions))
	mux.Handle("GET /api/v1/data/projects/{projectId}/alarm-policies/{id}", read(opts.alarmPolicyHandler.Get))
	mux.Handle("PUT /api/v1/data/projects/{projectId}/alarm-policies/{id}", write(opts.alarmPolicyHandler.Update))
	mux.Handle("PATCH /api/v1/data/projects/{projectId}/alarm-policies/{id}/enabled", write(opts.alarmPolicyHandler.ToggleEnabled))
	mux.Handle("DELETE /api/v1/data/projects/{projectId}/alarm-policies/{id}", write(opts.alarmPolicyHandler.Delete))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-policies/{id}/test", read(opts.alarmPolicyHandler.Test))
	mux.Handle("GET /api/v1/data/projects/{projectId}/alarm-policies/{id}/contract", read(opts.alarmPolicyHandler.Contract))
}

func mountAlarmRuleRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.alarmRuleHandler == nil || opts.jwtValidator == nil {
		return
	}

	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/alarm-rules",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.alarmRuleHandler.List),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/alarm-rules",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.alarmRuleHandler.Create),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/alarm-rules/{id}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.alarmRuleHandler.Get),
			),
		),
	)
	mux.Handle(
		"PUT /api/v1/data/projects/{projectId}/alarm-rules/{id}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.alarmRuleHandler.Update),
			),
		),
	)
	mux.Handle(
		"DELETE /api/v1/data/projects/{projectId}/alarm-rules/{id}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.alarmRuleHandler.Delete),
			),
		),
	)
	mux.Handle(
		"PATCH /api/v1/data/projects/{projectId}/alarm-rules/{id}/enabled",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.alarmRuleHandler.ToggleEnabled),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/alarm-rules/validate-draft",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.alarmRuleHandler.ValidateDraft),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/alarm-rules/{id}/validate-target",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.alarmRuleHandler.ValidateTarget),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/alarm-rules/{id}/test",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.alarmRuleHandler.Test),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/alarm-rules/{id}/contract",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.alarmRuleHandler.Contract),
			),
		),
	)
}

func mountAccessSourceRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.accessSourceHandler == nil || opts.jwtValidator == nil {
		return
	}

	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/access-sources",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.accessSourceHandler.List),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/access-sources/{sourceId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.accessSourceHandler.Get),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/access-sources/{sourceId}/records",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.accessSourceHandler.Records),
			),
		),
	)
}

func mountContractCheckRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.contractCheckHandler == nil || opts.jwtValidator == nil {
		return
	}

	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/contract-checks/run",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.contractCheckHandler.Run),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/contract-checks/latest",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.contractCheckHandler.Latest),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/contract-checks/runs",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.contractCheckHandler.Runs),
			),
		),
	)
}

func mountConnectionRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.connectionHandler == nil || opts.jwtValidator == nil {
		return
	}

	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/connections",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.connectionHandler.List),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/connections",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.connectionHandler.Create),
			),
		),
	)
	mux.Handle(
		"PUT /api/v1/data/projects/{projectId}/connections/{connectionId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.connectionHandler.Update),
			),
		),
	)
	mux.Handle(
		"DELETE /api/v1/data/projects/{projectId}/connections/{connectionId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.connectionHandler.Delete),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/connections/test",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.connectionHandler.TestConnection),
			),
		),
	)
	mux.Handle(
		"PATCH /api/v1/data/projects/{projectId}/connections/{connectionId}/status",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.connectionHandler.UpdateStatus),
			),
		),
	)
	mux.Handle(
		"PATCH /api/v1/data/projects/{projectId}/connections/order",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.connectionHandler.UpdateOrder),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/connections/{connectionId}/tables",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.connectionHandler.ListTables),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/connections/{connectionId}/tables",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.connectionHandler.CreateTable),
			),
		),
	)
	mux.Handle(
		"PUT /api/v1/data/projects/{projectId}/connections/{connectionId}/tables/{tableName}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.connectionHandler.RenameTable),
			),
		),
	)
	mux.Handle(
		"DELETE /api/v1/data/projects/{projectId}/connections/{connectionId}/tables/{tableName}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.connectionHandler.DeleteTable),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/connections/{connectionId}/tables/{tableName}/structure",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.connectionHandler.GetTableStructure),
			),
		),
	)
	mux.Handle(
		"PATCH /api/v1/data/projects/{projectId}/connections/{connectionId}/tables/{tableName}/structure",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.connectionHandler.UpdateTableStructure),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/connections/{connectionId}/tables/{tableName}/data",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.connectionHandler.GetTableData),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/connections/{connectionId}/execute-sql",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.connectionHandler.ExecuteSQL),
			),
		),
	)
}

func mountDataRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.jwtValidator == nil {
		return
	}

	if opts.queryHandler != nil {
		mux.Handle(
			"GET /api/v1/data/projects/{projectId}/queries",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:read")(
					middleware.ErrorHandler(opts.queryHandler.List),
				),
			),
		)
		mux.Handle(
			"POST /api/v1/data/projects/{projectId}/queries",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:write")(
					middleware.ErrorHandler(opts.queryHandler.Create),
				),
			),
		)
		mux.Handle(
			"POST /api/v1/data/queries/{id}/execute",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:read")(
					middleware.ErrorHandler(opts.queryHandler.Execute),
				),
			),
		)
		mux.Handle(
			"PUT /api/v1/data/queries/{id}",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:write")(
					middleware.ErrorHandler(opts.queryHandler.Update),
				),
			),
		)
		mux.Handle(
			"DELETE /api/v1/data/queries/{id}",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:write")(
					middleware.ErrorHandler(opts.queryHandler.Delete),
				),
			),
		)
	}

	if opts.dataPointHandler != nil {
		mux.Handle(
			"GET /api/v1/data/projects/{projectId}/datapoints",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:read")(
					middleware.ErrorHandler(opts.dataPointHandler.List),
				),
			),
		)
		mux.Handle(
			"GET /api/v1/data/projects/{projectId}/datapoints/{id}",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:read")(
					middleware.ErrorHandler(opts.dataPointHandler.Get),
				),
			),
		)
		mux.Handle(
			"GET /api/v1/data/projects/{projectId}/datapoints/value",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:read")(
					middleware.ErrorHandler(opts.dataPointHandler.GetValue),
				),
			),
		)
		mux.Handle(
			"PUT /api/v1/data/projects/{projectId}/datapoints/{id}",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:write")(
					middleware.ErrorHandler(opts.dataPointHandler.Update),
				),
			),
		)
		mux.Handle(
			"DELETE /api/v1/data/projects/{projectId}/datapoints/{id}",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:write")(
					middleware.ErrorHandler(opts.dataPointHandler.Delete),
				),
			),
		)
		mux.Handle(
			"POST /api/v1/data/projects/{projectId}/datapoints/delete-batch",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:write")(
					middleware.ErrorHandler(opts.dataPointHandler.DeleteBatch),
				),
			),
		)
		mux.Handle(
			"POST /api/v1/data/projects/{projectId}/datapoints/delete-by-filter",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:write")(
					middleware.ErrorHandler(opts.dataPointHandler.DeleteBatchByFilter),
				),
			),
		)
		mux.Handle(
			"POST /api/v1/data/projects/{projectId}/datapoints/tags-by-filter",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:write")(
					middleware.ErrorHandler(opts.dataPointHandler.AppendTagsByFilter),
				),
			),
		)
		mux.Handle(
			"POST /api/v1/data/projects/{projectId}/datapoints/status",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:read")(
					middleware.ErrorHandler(opts.dataPointHandler.BatchStatus),
				),
			),
		)
		mux.Handle(
			"POST /api/v1/data/projects/{projectId}/datapoints/values",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:read")(
					middleware.ErrorHandler(opts.dataPointHandler.BatchValues),
				),
			),
		)
		mux.Handle(
			"POST /api/v1/data/projects/{projectId}/datapoints/{id}/write",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:write")(
					middleware.ErrorHandler(opts.dataPointHandler.WriteValue),
				),
			),
		)
		mux.Handle(
			"PUT /api/v1/data/projects/{projectId}/datapoints/{id}/runtime-permissions",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:write")(
					middleware.ErrorHandler(opts.dataPointHandler.UpdateRuntimePermissions),
				),
			),
		)
		mux.Handle(
			"GET /api/v1/data/projects/{projectId}/datapoints/{id}/usages",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:read")(
					middleware.ErrorHandler(opts.dataPointHandler.Usages),
				),
			),
		)
	}
}

func mountWorkbenchGroupRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.workbenchGroupHandler == nil || opts.jwtValidator == nil {
		return
	}

	read := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}
	write := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}

	mux.Handle("GET /api/v1/data/projects/{projectId}/connections/{connectionId}/workbench-groups", read(opts.workbenchGroupHandler.ListGroups))
	mux.Handle("POST /api/v1/data/projects/{projectId}/connections/{connectionId}/workbench-groups", write(opts.workbenchGroupHandler.CreateGroup))
	mux.Handle("PUT /api/v1/data/projects/{projectId}/workbench-groups/{groupId}", write(opts.workbenchGroupHandler.UpdateGroup))
	mux.Handle("DELETE /api/v1/data/projects/{projectId}/workbench-groups/{groupId}", write(opts.workbenchGroupHandler.DeleteGroup))
	mux.Handle("PATCH /api/v1/data/projects/{projectId}/queries/{queryId}/group", write(opts.workbenchGroupHandler.MoveQuery))
	mux.Handle("GET /api/v1/data/projects/{projectId}/connections/{connectionId}/table-group-members", read(opts.workbenchGroupHandler.ListTableMembers))
	mux.Handle("PATCH /api/v1/data/projects/{projectId}/connections/{connectionId}/tables/{tableName}/group", write(opts.workbenchGroupHandler.MoveTable))
}

func mountMqttRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.jwtValidator == nil || opts.mqttHandler == nil {
		return
	}

	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/mqtt/connections",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.mqttHandler.ListConnections),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/mqtt/connections",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.CreateConnection),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.mqttHandler.GetConnection),
			),
		),
	)
	mux.Handle(
		"PUT /api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.UpdateConnection),
			),
		),
	)
	mux.Handle(
		"DELETE /api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.DeleteConnection),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/mqtt/connections/test",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.mqttHandler.TestConnectionConfig),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}/start",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.StartConnection),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}/stop",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.StopConnection),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}/publish",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.PublishMessage),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}/status",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.mqttHandler.GetConnectionStatus),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}/subscriptions",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.mqttHandler.ListSubscriptions),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}/subscription-groups",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.mqttHandler.ListSubscriptionGroups),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}/subscription-groups",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.CreateSubscriptionGroup),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}/subscriptions",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.CreateSubscription),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.mqttHandler.GetSubscription),
			),
		),
	)
	mux.Handle(
		"PUT /api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.UpdateSubscription),
			),
		),
	)
	mux.Handle(
		"PUT /api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}/default-batch-parse-rule",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.UpdateSubscriptionDefaultBatchRule),
			),
		),
	)
	mux.Handle(
		"DELETE /api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.DeleteSubscription),
			),
		),
	)
	mux.Handle(
		"PUT /api/v1/data/projects/{projectId}/mqtt/subscription-groups/{groupId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.UpdateSubscriptionGroup),
			),
		),
	)
	mux.Handle(
		"DELETE /api/v1/data/projects/{projectId}/mqtt/subscription-groups/{groupId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.DeleteSubscriptionGroup),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}/messages",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.mqttHandler.ListMessages),
			),
		),
	)
	mux.Handle(
		"DELETE /api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}/messages",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.ClearMessages),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}/tags",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.mqttHandler.ListTagsBySubscription),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}/tags",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.CreateTag),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}/tags/batch",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.CreateTagsBatch),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}/tags/delete-filtered",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.DeleteTagsBySubscriptionFilter),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/mqtt/tags",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.mqttHandler.ListTagsByProject),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/mqtt/tags/delete-batch",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.DeleteTagsBatch),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/mqtt/tags/{tagId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.mqttHandler.GetTag),
			),
		),
	)
	mux.Handle(
		"PUT /api/v1/data/projects/{projectId}/mqtt/tags/{tagId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.UpdateTag),
			),
		),
	)
	mux.Handle(
		"DELETE /api/v1/data/projects/{projectId}/mqtt/tags/{tagId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.DeleteTag),
			),
		),
	)
	mux.Handle(
		"PUT /api/v1/data/projects/{projectId}/mqtt/tags/order",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.mqttHandler.UpdateTagsOrder),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/mqtt/tags/{tagId}/value",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.mqttHandler.GetTagValue),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/mqtt/tags/values",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.mqttHandler.GetTagValues),
			),
		),
	)
}

func mountKafkaWorkbenchRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.kafkaWorkbenchHandler == nil || opts.jwtValidator == nil {
		return
	}

	read := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}
	write := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}

	sourceBase := "/api/v1/data/projects/{projectId}/kafka/sources/{connectionId}"
	mappingBase := "/api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}"
	fieldBase := "/api/v1/data/projects/{projectId}/kafka/fields/{fieldId}"
	fieldGroupBase := "/api/v1/data/projects/{projectId}/kafka/field-groups/{groupId}"

	mux.Handle("GET "+sourceBase+"/topic-groups", read(opts.kafkaWorkbenchHandler.ListTopicGroups))
	mux.Handle("POST "+sourceBase+"/topic-groups", write(opts.kafkaWorkbenchHandler.CreateTopicGroup))
	mux.Handle("PUT /api/v1/data/projects/{projectId}/kafka/topic-groups/{groupId}", write(opts.kafkaWorkbenchHandler.UpdateTopicGroup))
	mux.Handle("DELETE /api/v1/data/projects/{projectId}/kafka/topic-groups/{groupId}", write(opts.kafkaWorkbenchHandler.DeleteTopicGroup))

	mux.Handle("GET "+sourceBase+"/topic-mappings", read(opts.kafkaWorkbenchHandler.ListTopicMappings))
	mux.Handle("POST "+sourceBase+"/topic-mappings", write(opts.kafkaWorkbenchHandler.CreateTopicMapping))
	mux.Handle("GET "+mappingBase, read(opts.kafkaWorkbenchHandler.GetTopicMapping))
	mux.Handle("PUT "+mappingBase, write(opts.kafkaWorkbenchHandler.UpdateTopicMapping))
	mux.Handle("DELETE "+mappingBase, write(opts.kafkaWorkbenchHandler.DeleteTopicMapping))

	mux.Handle("POST "+sourceBase+"/preview", read(opts.kafkaWorkbenchHandler.PreviewConnection))
	mux.Handle("POST "+mappingBase+"/preview", read(opts.kafkaWorkbenchHandler.PreviewTopicMapping))

	mux.Handle("GET "+mappingBase+"/field-groups", read(opts.kafkaWorkbenchHandler.ListFieldGroups))
	mux.Handle("POST "+mappingBase+"/field-groups", write(opts.kafkaWorkbenchHandler.CreateFieldGroup))
	mux.Handle("PUT "+fieldGroupBase, write(opts.kafkaWorkbenchHandler.UpdateFieldGroup))
	mux.Handle("DELETE "+fieldGroupBase, write(opts.kafkaWorkbenchHandler.DeleteFieldGroup))

	mux.Handle("GET "+mappingBase+"/fields", read(opts.kafkaWorkbenchHandler.ListFields))
	mux.Handle("POST "+mappingBase+"/fields", write(opts.kafkaWorkbenchHandler.CreateField))
	mux.Handle("POST "+mappingBase+"/fields/batch", write(opts.kafkaWorkbenchHandler.CreateFieldsBatch))
	mux.Handle("PUT "+fieldBase, write(opts.kafkaWorkbenchHandler.UpdateField))
	mux.Handle("DELETE "+fieldBase, write(opts.kafkaWorkbenchHandler.DeleteField))
	mux.Handle("PATCH "+fieldBase+"/toggle", write(opts.kafkaWorkbenchHandler.ToggleField))
}

func mountHTTPWorkbenchRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.httpWorkbenchHandler == nil || opts.jwtValidator == nil {
		return
	}

	read := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}
	write := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}

	sourceBase := "/api/v1/data/projects/{projectId}/http/sources/{connectionId}"
	requestBase := "/api/v1/data/projects/{projectId}/http/requests/{requestId}"
	groupBase := "/api/v1/data/projects/{projectId}/http/request-groups/{groupId}"

	mux.Handle("GET "+sourceBase+"/request-groups", read(opts.httpWorkbenchHandler.ListGroups))
	mux.Handle("POST "+sourceBase+"/request-groups", write(opts.httpWorkbenchHandler.CreateGroup))
	mux.Handle("PUT "+groupBase, write(opts.httpWorkbenchHandler.UpdateGroup))
	mux.Handle("DELETE "+groupBase, write(opts.httpWorkbenchHandler.DeleteGroup))

	mux.Handle("GET "+sourceBase+"/requests", read(opts.httpWorkbenchHandler.ListRequests))
	mux.Handle("POST "+sourceBase+"/requests", write(opts.httpWorkbenchHandler.CreateRequest))
	mux.Handle("GET "+requestBase, read(opts.httpWorkbenchHandler.GetRequest))
	mux.Handle("PUT "+requestBase, write(opts.httpWorkbenchHandler.UpdateRequest))
	mux.Handle("DELETE "+requestBase, write(opts.httpWorkbenchHandler.DeleteRequest))
	mux.Handle("POST "+requestBase+"/send", write(opts.httpWorkbenchHandler.SendRequest))
}

func mountWebSocketWorkbenchRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.websocketWorkbenchHandler == nil || opts.jwtValidator == nil {
		return
	}

	read := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}
	write := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}

	sourceBase := "/api/v1/data/projects/{projectId}/websocket/sources/{connectionId}"
	sessionBase := "/api/v1/data/projects/{projectId}/websocket/sessions/{sessionId}"
	groupBase := "/api/v1/data/projects/{projectId}/websocket/session-groups/{groupId}"

	mux.Handle("GET "+sourceBase+"/session-groups", read(opts.websocketWorkbenchHandler.ListGroups))
	mux.Handle("POST "+sourceBase+"/session-groups", write(opts.websocketWorkbenchHandler.CreateGroup))
	mux.Handle("PUT "+groupBase, write(opts.websocketWorkbenchHandler.UpdateGroup))
	mux.Handle("DELETE "+groupBase, write(opts.websocketWorkbenchHandler.DeleteGroup))

	mux.Handle("GET "+sourceBase+"/sessions", read(opts.websocketWorkbenchHandler.ListSessions))
	mux.Handle("POST "+sourceBase+"/sessions", write(opts.websocketWorkbenchHandler.CreateSession))
	mux.Handle("GET "+sessionBase, read(opts.websocketWorkbenchHandler.GetSession))
	mux.Handle("PUT "+sessionBase, write(opts.websocketWorkbenchHandler.UpdateSession))
	mux.Handle("DELETE "+sessionBase, write(opts.websocketWorkbenchHandler.DeleteSession))
	mux.Handle("POST "+sessionBase+"/connect-preview", write(opts.websocketWorkbenchHandler.ConnectPreview))
	mux.Handle("GET "+sessionBase+"/stream", opts.websocketWorkbenchHandler.StreamSessionWithQueryToken(opts.jwtValidator))
}

func mountOpcuaModelingRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.jwtValidator == nil || opts.opcuaModelingHandler == nil {
		return
	}

	read := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}
	write := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}

	base := "/api/v1/data/projects/{projectId}/opcua/{connectionId}"
	mux.Handle("GET "+base+"/node-groups", read(opts.opcuaModelingHandler.ListGroups))
	mux.Handle("POST "+base+"/node-groups", write(opts.opcuaModelingHandler.CreateGroup))
	mux.Handle("PUT "+base+"/node-groups/{groupId}", write(opts.opcuaModelingHandler.UpdateGroup))
	mux.Handle("DELETE "+base+"/node-groups/{groupId}", write(opts.opcuaModelingHandler.DeleteGroup))
	mux.Handle("GET "+base+"/nodes", read(opts.opcuaModelingHandler.ListNodes))
	mux.Handle("POST "+base+"/nodes", write(opts.opcuaModelingHandler.CreateNode))
	mux.Handle("POST "+base+"/nodes/batch-import", write(opts.opcuaModelingHandler.BatchImportNodes))
	mux.Handle("PUT "+base+"/nodes/{nodeId}", write(opts.opcuaModelingHandler.UpdateNode))
	mux.Handle("DELETE "+base+"/nodes/{nodeId}", write(opts.opcuaModelingHandler.DeleteNode))
	mux.Handle("POST "+base+"/validate-model", read(opts.opcuaModelingHandler.ValidateModel))
	mux.Handle("POST "+base+"/preview", read(opts.opcuaModelingHandler.PreviewNodes))
}

func mountModbusModelingRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.jwtValidator == nil || opts.modbusModelingHandler == nil {
		return
	}

	read := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}
	write := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}

	base := "/api/v1/data/projects/{projectId}/modbus/{connectionId}"
	mux.Handle("GET "+base+"/register-groups", read(opts.modbusModelingHandler.ListGroups))
	mux.Handle("POST "+base+"/register-groups", write(opts.modbusModelingHandler.CreateGroup))
	mux.Handle("PUT "+base+"/register-groups/{groupId}", write(opts.modbusModelingHandler.UpdateGroup))
	mux.Handle("DELETE "+base+"/register-groups/{groupId}", write(opts.modbusModelingHandler.DeleteGroup))
	mux.Handle("GET "+base+"/registers", read(opts.modbusModelingHandler.ListRegisters))
	mux.Handle("POST "+base+"/registers", write(opts.modbusModelingHandler.CreateRegister))
	mux.Handle("POST "+base+"/registers/batch-import", write(opts.modbusModelingHandler.BatchImportRegisters))
	mux.Handle("PUT "+base+"/registers/{registerId}", write(opts.modbusModelingHandler.UpdateRegister))
	mux.Handle("DELETE "+base+"/registers/{registerId}", write(opts.modbusModelingHandler.DeleteRegister))
	mux.Handle("POST "+base+"/validate-model", read(opts.modbusModelingHandler.ValidateModel))
	mux.Handle("POST "+base+"/preview", read(opts.modbusModelingHandler.PreviewRegisters))
	mux.Handle("GET "+base+"/read-plan-estimate", read(opts.modbusModelingHandler.EstimateReadPlans))
}

func mountS7ModelingRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.jwtValidator == nil || opts.s7ModelingHandler == nil {
		return
	}

	read := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}
	write := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}

	base := "/api/v1/data/projects/{projectId}/s7/{connectionId}"
	mux.Handle("GET "+base+"/profile", read(opts.s7ModelingHandler.GetProfile))
	mux.Handle("PUT "+base+"/profile", write(opts.s7ModelingHandler.UpsertProfile))
	mux.Handle("GET "+base+"/variable-groups", read(opts.s7ModelingHandler.ListGroups))
	mux.Handle("POST "+base+"/variable-groups", write(opts.s7ModelingHandler.CreateGroup))
	mux.Handle("PUT "+base+"/variable-groups/{groupId}", write(opts.s7ModelingHandler.UpdateGroup))
	mux.Handle("DELETE "+base+"/variable-groups/{groupId}", write(opts.s7ModelingHandler.DeleteGroup))
	mux.Handle("GET "+base+"/variables", read(opts.s7ModelingHandler.ListVariables))
	mux.Handle("POST "+base+"/variables", write(opts.s7ModelingHandler.CreateVariable))
	mux.Handle("POST "+base+"/variables/batch-import", write(opts.s7ModelingHandler.BatchImportVariables))
	mux.Handle("PUT "+base+"/variables/{variableId}", write(opts.s7ModelingHandler.UpdateVariable))
	mux.Handle("DELETE "+base+"/variables/{variableId}", write(opts.s7ModelingHandler.DeleteVariable))
	mux.Handle("POST "+base+"/validate-model", read(opts.s7ModelingHandler.ValidateModel))
	mux.Handle("POST "+base+"/preview", read(opts.s7ModelingHandler.PreviewVariables))
	mux.Handle("GET "+base+"/read-plan-estimate", read(opts.s7ModelingHandler.EstimateReadPlans))
}

func mountProtocolDevSessionRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.jwtValidator == nil || opts.protocolDevSessionHandler == nil {
		return
	}

	read := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}
	write := func(handlerFunc func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}

	opcuaBase := "/api/v1/data/projects/{projectId}/opcua/{connectionId}/sessions"
	mux.Handle("POST "+opcuaBase, read(opts.protocolDevSessionHandler.CreateOpcua))
	mux.Handle("DELETE "+opcuaBase+"/{sessionId}", write(opts.protocolDevSessionHandler.CloseOpcua))
	mux.Handle("GET "+opcuaBase+"/{sessionId}/browse", read(opts.protocolDevSessionHandler.BrowseOpcua))
	mux.Handle("POST "+opcuaBase+"/{sessionId}/read", read(opts.protocolDevSessionHandler.ReadOpcua))
	mux.Handle("POST "+opcuaBase+"/{sessionId}/subscribe", read(opts.protocolDevSessionHandler.SubscribeOpcua))
	mux.Handle("DELETE "+opcuaBase+"/{sessionId}/subscribe", write(opts.protocolDevSessionHandler.StopSubscribeOpcua))

	modbusBase := "/api/v1/data/projects/{projectId}/modbus/{connectionId}/sessions"
	mux.Handle("POST "+modbusBase, read(opts.protocolDevSessionHandler.CreateModbus))
	mux.Handle("DELETE "+modbusBase+"/{sessionId}", write(opts.protocolDevSessionHandler.CloseModbus))
	mux.Handle("POST "+modbusBase+"/{sessionId}/read", read(opts.protocolDevSessionHandler.ReadModbus))
	mux.Handle("POST "+modbusBase+"/{sessionId}/poll", read(opts.protocolDevSessionHandler.PollModbus))
	mux.Handle("DELETE "+modbusBase+"/{sessionId}/poll", write(opts.protocolDevSessionHandler.StopPollModbus))

	s7Base := "/api/v1/data/projects/{projectId}/s7/{connectionId}/sessions"
	mux.Handle("POST "+s7Base, read(opts.protocolDevSessionHandler.CreateS7))
	mux.Handle("DELETE "+s7Base+"/{sessionId}", write(opts.protocolDevSessionHandler.CloseS7))
	mux.Handle("POST "+s7Base+"/{sessionId}/read", read(opts.protocolDevSessionHandler.ReadS7))
	mux.Handle("POST "+s7Base+"/{sessionId}/poll", read(opts.protocolDevSessionHandler.PollS7))
	mux.Handle("DELETE "+s7Base+"/{sessionId}/poll", write(opts.protocolDevSessionHandler.StopPollS7))
}

func mountProjectSnapshotRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.jwtValidator == nil || opts.projectSnapshotHandler == nil {
		return
	}

	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/snapshot",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.projectSnapshotHandler.Get),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/artifact",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.projectSnapshotHandler.GetArtifact),
			),
		),
	)
	mux.Handle(
		"PUT /api/v1/data/projects/{projectId}/snapshot",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.projectSnapshotHandler.Replace),
			),
		),
	)
}

func mountProtocolWave1Routes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.jwtValidator == nil || opts.protocolWave1Handler == nil {
		return
	}

	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/kafka/configs",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.protocolWave1Handler.CreateKafkaConfig),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/kafka/configs/{connectionId}/preview",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.protocolWave1Handler.PreviewKafkaTopic),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/protocols/{connectionId}/preview",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.protocolWave1Handler.PreviewProtocol),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/http/configs",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.protocolWave1Handler.CreateHTTPConfig),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/websocket/configs",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.protocolWave1Handler.CreateWebSocketConfig),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/redis/configs",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.protocolWave1Handler.CreateRedisConfig),
			),
		),
	)
}

func mountProtocolWave2Routes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.jwtValidator == nil || opts.protocolWave2Handler == nil {
		return
	}

	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/opcua/configs",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.protocolWave2Handler.CreateOpcuaConfig),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/s7/configs",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.protocolWave2Handler.CreateS7Config),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/modbus/configs",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.protocolWave2Handler.CreateModbusConfig),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/tdengine/configs",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.protocolWave2Handler.CreateTdengineConfig),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/opcda/contracts/validate",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.protocolWave2Handler.ValidateOpcdaContract),
			),
		),
	)
}

func mountPreviewRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.jwtValidator == nil || opts.previewHandler == nil {
		return
	}

	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/preview/sessions",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.previewHandler.Create),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/preview/diagnostics",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.previewHandler.Diagnose),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/preview/sessions/{sessionId}/heartbeat",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.previewHandler.Heartbeat),
			),
		),
	)
	mux.Handle(
		"DELETE /api/v1/data/preview/sessions/{sessionId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.previewHandler.Delete),
			),
		),
	)
}

func mountPreviewSocketRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.previewSocketHandler == nil {
		return
	}

	mux.Handle("/socket.io", opts.previewSocketHandler)
	mux.Handle("/socket.io/", opts.previewSocketHandler)
}

func mountComputeRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.jwtValidator == nil || opts.computeHandler == nil {
		return
	}

	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/compute-units",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.computeHandler.List),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/compute-units",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.computeHandler.Create),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/compute-units/dependencies",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.computeHandler.Dependencies),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/compute-units/syntax-check",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.computeHandler.CheckSyntax),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/compute-units/folders",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.computeHandler.ListFolders),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/compute-units/folders",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.computeHandler.CreateFolder),
			),
		),
	)
	mux.Handle(
		"PUT /api/v1/data/projects/{projectId}/compute-units/folders/{folderId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.computeHandler.UpdateFolder),
			),
		),
	)
	mux.Handle(
		"DELETE /api/v1/data/projects/{projectId}/compute-units/folders/{folderId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.computeHandler.DeleteFolder),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/compute-units/{id}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.computeHandler.Get),
			),
		),
	)
	mux.Handle(
		"PUT /api/v1/data/projects/{projectId}/compute-units/{id}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.computeHandler.Update),
			),
		),
	)
	mux.Handle(
		"DELETE /api/v1/data/projects/{projectId}/compute-units/{id}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.computeHandler.Delete),
			),
		),
	)
	mux.Handle(
		"PATCH /api/v1/data/projects/{projectId}/compute-units/{id}/enabled",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.computeHandler.ToggleEnabled),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/compute-units/{id}/runs",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.computeHandler.Runs),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/compute-units/{id}/run",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.computeHandler.Run),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/compute-units/{id}/debug",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.computeHandler.Debug),
			),
		),
	)
}
