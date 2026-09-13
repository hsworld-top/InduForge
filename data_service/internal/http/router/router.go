package router

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/http/handler"
	"github.com/indu-forge/data_service/internal/http/middleware"
)

type options struct {
	alarmHandler                *handler.AlarmHandler
	builtinRuntimeHandler       *handler.BuiltinRuntimeHandler
	connectionHandler           *handler.ConnectionHandler
	contractCheckHandler        *handler.ContractCheckHandler
	collectorDevHandler         *handler.CollectorDevHandler
	collectorCatalogHandler     *handler.CollectorCatalogHandler
	collectorHandler            *handler.CollectorHandler
	collectorPointHandler       *handler.CollectorPointHandler
	collectorImportHandler      *handler.CollectorImportHandler
	collectorAuthenticator      middleware.CollectorAgentAuthenticator
	historyStorageHandler       *handler.HistoryStorageHandler
	queryHandler                *handler.QueryHandler
	workbenchGroupHandler       *handler.WorkbenchGroupHandler
	realtimeStoreHandler        *handler.RealtimeStoreHandler
	dataPointHandler            *handler.DataPointHandler
	mqttHandler                 *handler.MqttHandler
	kafkaWorkbenchHandler       *handler.KafkaWorkbenchHandler
	httpWorkbenchHandler        *handler.HTTPWorkbenchHandler
	websocketWorkbenchHandler   *handler.WebSocketWorkbenchHandler
	protocolConnectionHandler   *handler.ProtocolConnectionHandler
	tdengineOPCHandler          *handler.TDengineOPCHandler
	previewHandler              *handler.PreviewHandler
	previewSocketHandler        http.Handler
	computeHandler              *handler.ComputeHandler
	projectSnapshotHandler      *handler.ProjectSnapshotHandler
	projectTenantBindingHandler *handler.ProjectTenantBindingHandler
	collectorBindingHandler     *handler.CollectorBindingBundleHandler
	authoringFenceHandler       *handler.AuthoringFenceHandler
	authoringFenceChecker       middleware.AuthoringFenceChecker
	internalToken               string
	jwtValidator                *auth.JWTValidator
}

func WithAuthoringFenceRoutes(fenceHandler *handler.AuthoringFenceHandler, checker middleware.AuthoringFenceChecker, jwtValidator *auth.JWTValidator, internalToken string) Option {
	return func(opts *options) {
		opts.authoringFenceHandler = fenceHandler
		opts.authoringFenceChecker = checker
		opts.jwtValidator = jwtValidator
		opts.internalToken = internalToken
	}
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

// WithAlarmRoutes wires alarm policy routes.
func WithAlarmRoutes(alarmHandler *handler.AlarmHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.alarmHandler = alarmHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithHistoryStorageRoutes wires development-time history storage routes.
func WithHistoryStorageRoutes(historyStorageHandler *handler.HistoryStorageHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.historyStorageHandler = historyStorageHandler
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

// WithCollectorDevRoutes wires Collector Dev 用户 API 与 Agent API。
func WithCollectorDevRoutes(collectorHandler *handler.CollectorDevHandler, authenticator middleware.CollectorAgentAuthenticator, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.collectorDevHandler = collectorHandler
		opts.collectorAuthenticator = authenticator
		opts.jwtValidator = jwtValidator
	}
}

// WithCollectorCatalogRoutes wires统一工业采集驱动目录路由。
func WithCollectorCatalogRoutes(catalogHandler *handler.CollectorCatalogHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.collectorCatalogHandler = catalogHandler
		opts.jwtValidator = jwtValidator
	}
}

func WithCollectorRoutes(collectorHandler *handler.CollectorHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) { opts.collectorHandler = collectorHandler; opts.jwtValidator = jwtValidator }
}

func WithCollectorPointRoutes(pointHandler *handler.CollectorPointHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) { opts.collectorPointHandler = pointHandler; opts.jwtValidator = jwtValidator }
}

func WithCollectorImportRoutes(importHandler *handler.CollectorImportHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) { opts.collectorImportHandler = importHandler; opts.jwtValidator = jwtValidator }
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

// WithProtocolConnectionRoutes wires protocol wave 1 routes.
func WithProtocolConnectionRoutes(protocolConnectionHandler *handler.ProtocolConnectionHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.protocolConnectionHandler = protocolConnectionHandler
		opts.jwtValidator = jwtValidator
	}
}

// WithTDengineOPCRoutes wires protocol wave 2 routes.
func WithTDengineOPCRoutes(tdengineOPCHandler *handler.TDengineOPCHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.tdengineOPCHandler = tdengineOPCHandler
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

// WithProjectSnapshotInternalRoutes wires control 面快照发布与恢复接口。
func WithProjectSnapshotInternalRoutes(projectSnapshotHandler *handler.ProjectSnapshotHandler, internalToken string) Option {
	return func(opts *options) {
		opts.projectSnapshotHandler = projectSnapshotHandler
		opts.internalToken = internalToken
	}
}

// WithProjectTenantBindingInternalRoutes wires control 面专用的项目租户绑定内部路由。
func WithProjectTenantBindingInternalRoutes(bindingHandler *handler.ProjectTenantBindingHandler, internalToken string) Option {
	return func(opts *options) {
		opts.projectTenantBindingHandler = bindingHandler
		opts.internalToken = internalToken
	}
}

// WithCollectorBindingBundleInternalRoutes wires control-plane-only collector bundle assembly.
func WithCollectorBindingBundleInternalRoutes(bundleHandler *handler.CollectorBindingBundleHandler, internalToken string) Option {
	return func(opts *options) {
		opts.collectorBindingHandler = bundleHandler
		opts.internalToken = internalToken
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

	mountAlarmRoutes(mux, opts)
	mountHistoryStorageRoutes(mux, opts)
	mountBuiltinRuntimeRoutes(mux, opts)
	mountContractCheckRoutes(mux, opts)
	mountCollectorDevRoutes(mux, opts)
	mountCollectorCatalogRoutes(mux, opts)
	mountCollectorRoutes(mux, opts)
	mountCollectorPointRoutes(mux, opts)
	mountCollectorImportRoutes(mux, opts)
	mountConnectionRoutes(mux, opts)
	mountDataRoutes(mux, opts)
	mountWorkbenchGroupRoutes(mux, opts)
	mountMqttRoutes(mux, opts)
	mountKafkaWorkbenchRoutes(mux, opts)
	mountHTTPWorkbenchRoutes(mux, opts)
	mountWebSocketWorkbenchRoutes(mux, opts)
	mountRealtimeStoreRoutes(mux, opts)
	mountProtocolConnectionRoutes(mux, opts)
	mountTDengineOPCRoutes(mux, opts)
	mountPreviewSocketRoutes(mux, opts)
	mountPreviewRoutes(mux, opts)
	mountComputeRoutes(mux, opts)
	mountProjectSnapshotRoutes(mux, opts)
	mountProjectSnapshotInternalRoutes(mux, opts)
	mountAuthoringFenceInternalRoutes(mux, opts)
	mountProjectTenantBindingInternalRoutes(mux, opts)
	mountCollectorBindingBundleInternalRoutes(mux, opts)
	return middleware.AuthoringFenceGuard(opts.jwtValidator, opts.authoringFenceChecker)(mux)
}

func mountAuthoringFenceInternalRoutes(mux *http.ServeMux, opts options) {
	if opts.authoringFenceHandler == nil {
		return
	}
	internal := func(fn func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.RequireInternalToken(opts.internalToken)(middleware.ErrorHandler(fn))
	}
	base := "/api/v1/internal/data/projects/{projectId}/authoring-fence"
	mux.Handle("POST "+base, internal(opts.authoringFenceHandler.Acquire))
	mux.Handle("PUT "+base, internal(opts.authoringFenceHandler.Renew))
	mux.Handle("DELETE "+base, internal(opts.authoringFenceHandler.Release))
}

func mountProjectSnapshotInternalRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.projectSnapshotHandler == nil {
		return
	}
	internal := func(fn func(http.ResponseWriter, *http.Request) error) http.Handler {
		return middleware.RequireInternalToken(opts.internalToken)(middleware.ErrorHandler(fn))
	}
	base := "/api/v1/internal/data/projects/{projectId}"
	mux.Handle("GET "+base+"/snapshot", internal(opts.projectSnapshotHandler.GetInternal))
	mux.Handle("PUT "+base+"/snapshot", internal(opts.projectSnapshotHandler.PutInternal))
	mux.Handle("POST "+base+"/artifact", internal(opts.projectSnapshotHandler.BuildArtifactInternal))
	mux.Handle("POST "+base+"/snapshot/artifacts", internal(opts.projectSnapshotHandler.BuildArtifactsInternal))
}

func mountCollectorBindingBundleInternalRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.collectorBindingHandler == nil {
		return
	}
	mux.Handle("POST /api/v1/internal/data/projects/{projectId}/collector-binding", middleware.RequireInternalToken(opts.internalToken)(
		middleware.ErrorHandler(opts.collectorBindingHandler.Build),
	))
}

func mountProjectTenantBindingInternalRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.projectTenantBindingHandler == nil {
		return
	}
	mux.Handle("PUT /api/v1/internal/data/project-bindings/{projectId}", middleware.RequireInternalToken(opts.internalToken)(
		middleware.ErrorHandler(opts.projectTenantBindingHandler.Put),
	))
	mux.Handle("GET /api/v1/internal/data/project-bindings/{projectId}", middleware.RequireInternalToken(opts.internalToken)(
		middleware.ErrorHandler(opts.projectTenantBindingHandler.Get),
	))
}

func mountHistoryStorageRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.historyStorageHandler == nil || opts.jwtValidator == nil {
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

	base := "/api/v1/data/projects/{projectId}/history-storage"
	mux.Handle("GET "+base+"/sources", read(opts.historyStorageHandler.ListSources))
	mux.Handle("GET "+base+"/sources/{scopeType}/{scopeId}", read(opts.historyStorageHandler.GetSource))
	mux.Handle("PUT "+base+"/sources/{scopeType}/{scopeId}", write(opts.historyStorageHandler.SaveSource))
	mux.Handle("GET "+base+"/targets", read(opts.historyStorageHandler.ListTargets))
	mux.Handle("GET "+base+"/datapoints/{datapointId}", read(opts.historyStorageHandler.GetDatapoint))
	mux.Handle("PUT "+base+"/datapoints/{datapointId}", write(opts.historyStorageHandler.SaveDatapoint))
	mux.Handle("POST "+base+"/datapoints/batch-configure", write(opts.historyStorageHandler.BatchConfigure))
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

func mountAlarmRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.alarmHandler == nil || opts.jwtValidator == nil {
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

	mux.Handle("GET /api/v1/data/projects/{projectId}/alarm-groups", read(opts.alarmHandler.ListGroups))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-groups", write(opts.alarmHandler.CreateGroup))
	mux.Handle("PUT /api/v1/data/projects/{projectId}/alarm-groups/{id}", write(opts.alarmHandler.UpdateGroup))
	mux.Handle("DELETE /api/v1/data/projects/{projectId}/alarm-groups/{id}", write(opts.alarmHandler.DeleteGroup))

	mux.Handle("GET /api/v1/data/projects/{projectId}/alarm-settings", read(opts.alarmHandler.GetSettings))
	mux.Handle("PUT /api/v1/data/projects/{projectId}/alarm-settings", write(opts.alarmHandler.UpdateSettings))
	mux.Handle("GET /api/v1/data/projects/{projectId}/alarm-level-settings", read(opts.alarmHandler.GetLevelSettings))
	mux.Handle("PUT /api/v1/data/projects/{projectId}/alarm-level-settings", write(opts.alarmHandler.UpdateLevelSettings))
	mux.Handle("GET /api/v1/data/projects/{projectId}/alarm-history-settings", read(opts.alarmHandler.GetHistorySettings))
	mux.Handle("PUT /api/v1/data/projects/{projectId}/alarm-history-settings", write(opts.alarmHandler.UpdateHistorySettings))
	mux.Handle("GET /api/v1/data/projects/{projectId}/alarm-channels", read(opts.alarmHandler.ListChannels))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-channels", write(opts.alarmHandler.CreateChannel))
	mux.Handle("PUT /api/v1/data/projects/{projectId}/alarm-channels/{id}", write(opts.alarmHandler.UpdateChannel))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-channels/{id}/test", write(opts.alarmHandler.TestChannel))
	mux.Handle("DELETE /api/v1/data/projects/{projectId}/alarm-channels/{id}", write(opts.alarmHandler.DeleteChannel))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-config-sync", write(opts.alarmHandler.SyncConfig))

	mux.Handle("GET /api/v1/data/projects/{projectId}/alarm-items", read(opts.alarmHandler.List))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-items", write(opts.alarmHandler.Create))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-items/batch-create", write(opts.alarmHandler.BatchCreate))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-items/batch-create/validate", read(opts.alarmHandler.ValidateBatchCreate))
	mux.Handle("PUT /api/v1/data/projects/{projectId}/alarm-items/preset-config", write(opts.alarmHandler.SavePresetConfiguration))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-items/preset-config/validate", read(opts.alarmHandler.ValidatePresetConfiguration))
	mux.Handle("PATCH /api/v1/data/projects/{projectId}/alarm-items/batch", write(opts.alarmHandler.BatchUpdate))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-items/batch-delete", write(opts.alarmHandler.BatchDelete))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-items/export", write(opts.alarmHandler.Export))
	mux.Handle("GET /api/v1/data/projects/{projectId}/alarm-items/import-template", read(opts.alarmHandler.ImportTemplate))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-items/import/preview", write(opts.alarmHandler.ImportPreview))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-items/import/error-workbook", write(opts.alarmHandler.ImportErrorWorkbook))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-items/import/apply", write(opts.alarmHandler.ImportApply))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-items/validate-draft", read(opts.alarmHandler.ValidateDraft))
	mux.Handle("POST /api/v1/data/projects/{projectId}/alarm-items/test-draft", read(opts.alarmHandler.TestDraft))
	mux.Handle("GET /api/v1/data/projects/{projectId}/alarm-items/{id}", read(opts.alarmHandler.Get))
	mux.Handle("PUT /api/v1/data/projects/{projectId}/alarm-items/{id}", write(opts.alarmHandler.Update))
	mux.Handle("PATCH /api/v1/data/projects/{projectId}/alarm-items/{id}/enabled", write(opts.alarmHandler.ToggleEnabled))
	mux.Handle("DELETE /api/v1/data/projects/{projectId}/alarm-items/{id}", write(opts.alarmHandler.Delete))
	mux.Handle("GET /api/v1/data/projects/{projectId}/alarm-items/{id}/contract", read(opts.alarmHandler.Contract))
	mux.Handle("GET /api/v1/data/projects/{projectId}/datapoints/{datapointId}/alarms", read(opts.alarmHandler.DatapointSummary))
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
		"GET /api/v1/data/projects/{projectId}/connections/{connectionId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.connectionHandler.Get),
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
		"POST /api/v1/data/projects/{projectId}/connections/{connectionId}/secrets/reveal",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.connectionHandler.RevealSecret),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/connections/{connectionId}/delete-impact",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.connectionHandler.DeleteImpact),
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
		"POST /api/v1/data/projects/{projectId}/connections/{connectionId}/test",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.connectionHandler.TestSavedConnection),
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
			"GET /api/v1/data/queries/{id}",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:read")(
					middleware.ErrorHandler(opts.queryHandler.Get),
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
			"GET /api/v1/data/projects/{projectId}/datapoint-source-options",
			middleware.Authenticate(opts.jwtValidator)(middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.dataPointHandler.ListSourceOptions),
			)),
		)
		mux.Handle(
			"GET /api/v1/data/projects/{projectId}/datapoints/development-contract",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:read")(
					middleware.ErrorHandler(opts.dataPointHandler.DevelopmentContract),
				),
			),
		)
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
			"POST /api/v1/data/projects/{projectId}/datapoints",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:write")(
					middleware.ErrorHandler(opts.dataPointHandler.Create),
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
			"GET /api/v1/data/projects/{projectId}/datapoints/tags",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:read")(
					middleware.ErrorHandler(opts.dataPointHandler.ListTags),
				),
			),
		)
		mux.Handle(
			"POST /api/v1/data/projects/{projectId}/datapoints/remove-tag",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:write")(
					middleware.ErrorHandler(opts.dataPointHandler.RemoveTag),
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
			"POST /api/v1/data/projects/{projectId}/datapoints/write-by-path",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:write")(
					middleware.ErrorHandler(opts.dataPointHandler.WriteValueByPath),
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
			"GET /api/v1/data/projects/{projectId}/datapoints/{id}/custom-attributes",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:read")(
					middleware.ErrorHandler(opts.dataPointHandler.GetCustomAttributes),
				),
			),
		)
		mux.Handle(
			"PUT /api/v1/data/projects/{projectId}/datapoints/{id}/custom-attributes",
			middleware.Authenticate(opts.jwtValidator)(
				middleware.RequireCapability("project:write")(
					middleware.ErrorHandler(opts.dataPointHandler.UpdateCustomAttributes),
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
		"POST /api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}/tags/sync",
		middleware.Authenticate(opts.jwtValidator)(middleware.RequireCapability("project:write")(middleware.ErrorHandler(opts.mqttHandler.SyncTagsBatch))),
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
		"POST /api/v1/data/projects/{projectId}/collector-artifact",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.projectSnapshotHandler.BuildCollectorArtifact),
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

func mountProtocolConnectionRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.jwtValidator == nil || opts.protocolConnectionHandler == nil {
		return
	}

	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/kafka/configs",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.protocolConnectionHandler.CreateKafkaConfig),
			),
		),
	)
	mux.Handle("PUT /api/v1/data/projects/{projectId}/kafka/configs/{connectionId}", middleware.Authenticate(opts.jwtValidator)(middleware.RequireCapability("project:write")(middleware.ErrorHandler(opts.protocolConnectionHandler.UpdateKafkaConfig))))
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/protocols/{connectionId}/preview",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.protocolConnectionHandler.PreviewProtocol),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/http/configs",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.protocolConnectionHandler.CreateHTTPConfig),
			),
		),
	)
	mux.Handle("PUT /api/v1/data/projects/{projectId}/http/configs/{connectionId}", middleware.Authenticate(opts.jwtValidator)(middleware.RequireCapability("project:write")(middleware.ErrorHandler(opts.protocolConnectionHandler.UpdateHTTPConfig))))
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/websocket/configs",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.protocolConnectionHandler.CreateWebSocketConfig),
			),
		),
	)
	mux.Handle("PUT /api/v1/data/projects/{projectId}/websocket/configs/{connectionId}", middleware.Authenticate(opts.jwtValidator)(middleware.RequireCapability("project:write")(middleware.ErrorHandler(opts.protocolConnectionHandler.UpdateWebSocketConfig))))
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/redis/configs",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.protocolConnectionHandler.CreateRedisConfig),
			),
		),
	)
	mux.Handle("PUT /api/v1/data/projects/{projectId}/redis/configs/{connectionId}", middleware.Authenticate(opts.jwtValidator)(middleware.RequireCapability("project:write")(middleware.ErrorHandler(opts.protocolConnectionHandler.UpdateRedisConfig))))
}

func mountTDengineOPCRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.jwtValidator == nil || opts.tdengineOPCHandler == nil {
		return
	}

	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/tdengine/configs",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.tdengineOPCHandler.CreateTdengineConfig),
			),
		),
	)
	mux.Handle(
		"PUT /api/v1/data/projects/{projectId}/tdengine/configs/{connectionId}",
		middleware.Authenticate(opts.jwtValidator)(middleware.RequireCapability("project:write")(middleware.ErrorHandler(opts.tdengineOPCHandler.UpdateTdengineConfig))),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/opcda/contracts/validate",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.tdengineOPCHandler.ValidateOpcdaContract),
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
			middleware.RequireCapability("project:read")(
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
			middleware.RequireCapability("project:read")(
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
		"POST /api/v1/data/projects/{projectId}/compute-units/dependencies",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.computeHandler.InstallDependency),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/compute-units/dependencies/import",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.computeHandler.ImportDependency),
			),
		),
	)
	mux.Handle(
		"DELETE /api/v1/data/projects/{projectId}/compute-units/dependencies/{dependencyId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.computeHandler.UninstallDependency),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/compute-units/capabilities",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.computeHandler.Capabilities),
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
		"POST /api/v1/data/projects/{projectId}/compute-units/schedule-preview",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.computeHandler.SchedulePreview),
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

func mountCollectorCatalogRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.collectorCatalogHandler == nil || opts.jwtValidator == nil {
		return
	}
	read := func(handlerFunc middleware.ErrorHandlerFunc) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("platform:read")(
				middleware.ErrorHandler(handlerFunc),
			),
		)
	}
	mux.Handle("GET /api/v1/data/collector/drivers", read(opts.collectorCatalogHandler.List))
	mux.Handle("GET /api/v1/data/collector/drivers/{driverId}", read(opts.collectorCatalogHandler.Get))
}

func mountCollectorRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.collectorHandler == nil || opts.jwtValidator == nil {
		return
	}
	read := func(handlerFunc middleware.ErrorHandlerFunc) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(middleware.RequireCapability("project:read")(middleware.ErrorHandler(handlerFunc)))
	}
	write := func(handlerFunc middleware.ErrorHandlerFunc) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(middleware.RequireCapability("project:write")(middleware.ErrorHandler(handlerFunc)))
	}
	base := "/api/v1/data/projects/{projectId}/collector/connections"
	mux.Handle("GET "+base, read(opts.collectorHandler.ListConnections))
	mux.Handle("POST "+base, write(opts.collectorHandler.CreateConnection))
	mux.Handle("GET "+base+"/{connectionId}", read(opts.collectorHandler.GetConnection))
	mux.Handle("PUT "+base+"/{connectionId}", write(opts.collectorHandler.UpdateConnection))
	mux.Handle("GET "+base+"/{connectionId}/diagnostic", read(opts.collectorHandler.GetConnectionDiagnostic))
	mux.Handle("GET "+base+"/{connectionId}/diagnostic/export", read(opts.collectorHandler.ExportConnectionDiagnostic))
	mux.Handle("GET "+base+"/{connectionId}/delete-impact", read(opts.collectorHandler.DeleteConnectionImpact))
	mux.Handle("DELETE "+base+"/{connectionId}", write(opts.collectorHandler.DeleteConnection))
}

func mountCollectorPointRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.collectorPointHandler == nil || opts.jwtValidator == nil {
		return
	}
	read := func(handlerFunc middleware.ErrorHandlerFunc) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(middleware.RequireCapability("project:read")(middleware.ErrorHandler(handlerFunc)))
	}
	write := func(handlerFunc middleware.ErrorHandlerFunc) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(middleware.RequireCapability("project:write")(middleware.ErrorHandler(handlerFunc)))
	}
	base := "/api/v1/data/projects/{projectId}/collector/connections/{connectionId}"
	mux.Handle("GET "+base+"/point-groups", read(opts.collectorPointHandler.ListGroups))
	mux.Handle("POST "+base+"/point-groups", write(opts.collectorPointHandler.CreateGroup))
	mux.Handle("PUT "+base+"/point-groups/{groupId}", write(opts.collectorPointHandler.UpdateGroup))
	mux.Handle("DELETE "+base+"/point-groups/{groupId}", write(opts.collectorPointHandler.DeleteGroup))
	mux.Handle("GET "+base+"/points", read(opts.collectorPointHandler.ListPoints))
	mux.Handle("POST "+base+"/points/export", read(opts.collectorPointHandler.ExportPoints))
	mux.Handle("POST "+base+"/points/check-addresses", read(opts.collectorPointHandler.CheckAddresses))
	mux.Handle("POST "+base+"/points/normalize-address", read(opts.collectorPointHandler.NormalizeAddress))
	mux.Handle("POST "+base+"/points/batch", write(opts.collectorPointHandler.CreateBatch))
	mux.Handle("POST "+base+"/points/update-batch", write(opts.collectorPointHandler.UpdateBatch))
	mux.Handle("POST "+base+"/points/delete-batch", write(opts.collectorPointHandler.DeleteBatch))
	mux.Handle("POST "+base+"/points/move-batch", write(opts.collectorPointHandler.MoveBatch))
}

func mountCollectorImportRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.collectorImportHandler == nil || opts.jwtValidator == nil {
		return
	}
	platformRead := func(handlerFunc middleware.ErrorHandlerFunc) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(middleware.RequireCapability("platform:read")(middleware.ErrorHandler(handlerFunc)))
	}
	projectWrite := func(handlerFunc middleware.ErrorHandlerFunc) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(middleware.RequireCapability("project:write")(middleware.ErrorHandler(handlerFunc)))
	}
	mux.Handle("GET /api/v1/data/collector/drivers/{driverId}/points/import-template", platformRead(opts.collectorImportHandler.Template))
	base := "/api/v1/data/projects/{projectId}/collector/connections/{connectionId}/points"
	mux.Handle("POST "+base+"/import-preview", projectWrite(opts.collectorImportHandler.Preview))
	mux.Handle("POST "+base+"/import-commit", projectWrite(opts.collectorImportHandler.Commit))
}

func mountCollectorDevRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.collectorDevHandler == nil || opts.jwtValidator == nil || opts.collectorAuthenticator == nil {
		return
	}
	platformRead := func(next middleware.ErrorHandlerFunc) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(middleware.ErrorHandler(next))
	}
	platformProjectRead := func(next middleware.ErrorHandlerFunc) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(middleware.RequireCapability("project:read")(middleware.ErrorHandler(next)))
	}
	platformProjectWrite := func(next middleware.ErrorHandlerFunc) http.Handler {
		return middleware.Authenticate(opts.jwtValidator)(middleware.RequireCapability("project:write")(middleware.ErrorHandler(next)))
	}
	agent := func(next middleware.ErrorHandlerFunc) http.Handler {
		return middleware.AuthenticateCollectorAgent(opts.collectorAuthenticator)(middleware.ErrorHandler(next))
	}

	mux.Handle("POST /api/v1/data/collector-dev/registration-codes", platformRead(opts.collectorDevHandler.CreateRegistrationCode))
	mux.Handle("GET /api/v1/data/collector-dev/agents", platformRead(opts.collectorDevHandler.ListAgents))
	mux.Handle("DELETE /api/v1/data/collector-dev/agents/{agentId}", platformRead(opts.collectorDevHandler.DeleteAgent))
	mux.Handle("POST /api/v1/data/projects/{projectId}/collector-dev/tasks", platformProjectWrite(opts.collectorDevHandler.CreateTask))
	mux.Handle("GET /api/v1/data/projects/{projectId}/collector-dev/tasks/{taskId}", platformProjectRead(opts.collectorDevHandler.GetTask))
	mux.Handle("DELETE /api/v1/data/projects/{projectId}/collector-dev/tasks/{taskId}", platformProjectWrite(opts.collectorDevHandler.CancelTask))
	mux.Handle("POST /api/v1/data/collector-dev/agent/register", middleware.ErrorHandler(opts.collectorDevHandler.RegisterAgent))
	mux.Handle("POST /api/v1/data/collector-dev/agent/disconnect", agent(opts.collectorDevHandler.Disconnect))
	mux.Handle("POST /api/v1/data/collector-dev/agent/revoke", agent(opts.collectorDevHandler.Revoke))
	mux.Handle("POST /api/v1/data/collector-dev/agent/heartbeat", agent(opts.collectorDevHandler.Heartbeat))
	mux.Handle("POST /api/v1/data/collector-dev/agent/tasks/claim", agent(opts.collectorDevHandler.ClaimTask))
	mux.Handle("POST /api/v1/data/collector-dev/agent/tasks/{taskId}/complete", agent(opts.collectorDevHandler.CompleteTask))
}
