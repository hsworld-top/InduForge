-- data_service 最终数据库结构基线。
-- 仅用于空数据库初始化；已有数据库的结构调整必须由开发者直接执行 SQL。

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
--
-- PostgreSQL database dump
--


-- Dumped from database version 16.13
-- Dumped by pg_dump version 16.13

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--



SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: collector_dev_agents; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE collector_dev_agents (
    id uuid NOT NULL,
    tenant_id text NOT NULL,
    name text NOT NULL,
    os text NOT NULL,
    arch text NOT NULL,
    version text NOT NULL,
    credential_hash text NOT NULL,
    capabilities jsonb DEFAULT '[]'::jsonb NOT NULL,
    last_seen_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    revoked_at timestamp with time zone,
    last_ip text DEFAULT ''::text NOT NULL,
    machine_id text NOT NULL,
    disconnected_at timestamp with time zone,
    CONSTRAINT collector_dev_agents_capabilities_check CHECK ((jsonb_typeof(capabilities) = 'array'::text))
);


--
-- Name: collector_dev_registration_codes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE collector_dev_registration_codes (
    id uuid NOT NULL,
    tenant_id text NOT NULL,
    code_hash text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone,
    used_by_agent_id uuid,
    created_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: collector_dev_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE collector_dev_tasks (
    id uuid NOT NULL,
    tenant_id text NOT NULL,
    project_id uuid NOT NULL,
    agent_id uuid NOT NULL,
    operation text NOT NULL,
    status text NOT NULL,
    request_payload jsonb NOT NULL,
    result_payload jsonb,
    error_code text,
    error_message text,
    deadline_at timestamp with time zone NOT NULL,
    claimed_at timestamp with time zone,
    finished_at timestamp with time zone,
    created_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    connection_id uuid NOT NULL,
    CONSTRAINT collector_dev_tasks_operation_check CHECK ((operation = ANY (ARRAY['connection.test'::text, 'connection.open'::text, 'connection.close'::text, 'device.browse'::text, 'point.read'::text, 'point.write'::text, 'point.subscribe.preview'::text]))),
    CONSTRAINT collector_dev_tasks_request_payload_check CHECK ((jsonb_typeof(request_payload) = 'object'::text)),
    CONSTRAINT collector_dev_tasks_status_check CHECK ((status = ANY (ARRAY['queued'::text, 'running'::text, 'succeeded'::text, 'failed'::text, 'cancelled'::text, 'expired'::text])))
);


--
-- Name: data_access_source_records; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_access_source_records (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id text NOT NULL,
    connection_id uuid NOT NULL,
    record_type text NOT NULL,
    title text NOT NULL,
    status text NOT NULL,
    protocol text NOT NULL,
    duration_ms integer DEFAULT 0 NOT NULL,
    sample_count integer DEFAULT 0 NOT NULL,
    truncated boolean DEFAULT false NOT NULL,
    error_summary text,
    detail jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_access_source_records_detail_check CHECK ((jsonb_typeof(detail) = 'object'::text)),
    CONSTRAINT data_access_source_records_duration_ms_check CHECK ((duration_ms >= 0)),
    CONSTRAINT data_access_source_records_error_summary_check CHECK (((error_summary IS NULL) OR (char_length(error_summary) <= 1000))),
    CONSTRAINT data_access_source_records_protocol_check CHECK ((char_length(protocol) <= 50)),
    CONSTRAINT data_access_source_records_record_type_check CHECK ((char_length(record_type) <= 100)),
    CONSTRAINT data_access_source_records_sample_count_check CHECK ((sample_count >= 0)),
    CONSTRAINT data_access_source_records_status_check CHECK ((char_length(status) <= 50)),
    CONSTRAINT data_access_source_records_title_check CHECK ((char_length(title) <= 200))
);


--
-- Name: data_alarm_policies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_alarm_policies (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    group_id uuid,
    name character varying(100) NOT NULL,
    description text,
    mode character varying(20) DEFAULT 'per_target'::character varying NOT NULL,
    targets jsonb DEFAULT '[]'::jsonb NOT NULL,
    inputs jsonb DEFAULT '[]'::jsonb NOT NULL,
    derived_expression text DEFAULT ''::text NOT NULL,
    conditions jsonb DEFAULT '[]'::jsonb NOT NULL,
    suppression jsonb DEFAULT '{}'::jsonb NOT NULL,
    message_template text DEFAULT ''::text NOT NULL,
    is_enabled boolean DEFAULT true NOT NULL,
    contract jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_by uuid,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_alarm_policies_conditions_array_check CHECK ((jsonb_typeof(conditions) = 'array'::text)),
    CONSTRAINT data_alarm_policies_contract_object_check CHECK ((jsonb_typeof(contract) = 'object'::text)),
    CONSTRAINT data_alarm_policies_inputs_array_check CHECK ((jsonb_typeof(inputs) = 'array'::text)),
    CONSTRAINT data_alarm_policies_mode_check CHECK (((mode)::text = ANY ((ARRAY['per_target'::character varying, 'derived'::character varying])::text[]))),
    CONSTRAINT data_alarm_policies_suppression_object_check CHECK ((jsonb_typeof(suppression) = 'object'::text)),
    CONSTRAINT data_alarm_policies_targets_array_check CHECK ((jsonb_typeof(targets) = 'array'::text))
);


--
-- Name: data_alarm_policy_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_alarm_policy_groups (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    parent_id uuid,
    name character varying(100) NOT NULL,
    description text,
    is_enabled boolean DEFAULT true NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_by uuid,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: data_alarm_project_settings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_alarm_project_settings (
    project_id uuid NOT NULL,
    escalation_interval_seconds integer DEFAULT 300 NOT NULL,
    repeat_notification_interval_seconds integer DEFAULT 60 NOT NULL,
    created_by uuid,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_alarm_project_settings_escalation_interval_check CHECK ((escalation_interval_seconds >= 30)),
    CONSTRAINT data_alarm_project_settings_repeat_interval_check CHECK ((repeat_notification_interval_seconds >= 10))
);


--
-- Name: data_alarm_rules; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_alarm_rules (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    name text NOT NULL,
    description text,
    target_datapoint_id uuid,
    target_path text NOT NULL,
    rule_type text DEFAULT 'H'::text NOT NULL,
    condition jsonb DEFAULT '{}'::jsonb NOT NULL,
    severity text DEFAULT 'warning'::text NOT NULL,
    hysteresis double precision,
    sample_window_ms integer,
    suppression jsonb DEFAULT '{}'::jsonb NOT NULL,
    message_template text DEFAULT ''::text NOT NULL,
    contract jsonb DEFAULT '{}'::jsonb NOT NULL,
    is_enabled boolean DEFAULT true NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_alarm_rules_condition_check CHECK ((jsonb_typeof(condition) = 'object'::text)),
    CONSTRAINT data_alarm_rules_contract_check CHECK ((jsonb_typeof(contract) = 'object'::text)),
    CONSTRAINT data_alarm_rules_hysteresis_check CHECK (((hysteresis IS NULL) OR (hysteresis >= (0)::double precision))),
    CONSTRAINT data_alarm_rules_name_check CHECK ((char_length(name) <= 100)),
    CONSTRAINT data_alarm_rules_rule_type_check CHECK ((rule_type = ANY (ARRAY['H'::text, 'L'::text, 'HH'::text, 'LL'::text, 'deviation_high'::text, 'deviation_low'::text, 'rate_of_change'::text, 'cel'::text]))),
    CONSTRAINT data_alarm_rules_sample_window_ms_check CHECK (((sample_window_ms IS NULL) OR (sample_window_ms >= 0))),
    CONSTRAINT data_alarm_rules_severity_check CHECK ((severity = ANY (ARRAY['info'::text, 'warning'::text, 'major'::text, 'critical'::text]))),
    CONSTRAINT data_alarm_rules_suppression_check CHECK ((jsonb_typeof(suppression) = 'object'::text)),
    CONSTRAINT data_alarm_rules_target_path_check CHECK ((char_length(target_path) <= 255))
);


--
-- Name: data_collector_connection_secrets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_collector_connection_secrets (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    connection_id uuid NOT NULL,
    secret_key text NOT NULL,
    encrypted_value bytea NOT NULL,
    encryption_key_version text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_collector_connection_secrets_key_check CHECK (((secret_key ~ '^[A-Za-z][A-Za-z0-9_.-]*$'::text) AND (char_length(secret_key) <= 100))),
    CONSTRAINT data_collector_connection_secrets_value_check CHECK ((octet_length(encrypted_value) > 0)),
    CONSTRAINT data_collector_connection_secrets_version_check CHECK (((char_length(encryption_key_version) >= 1) AND (char_length(encryption_key_version) <= 50)))
);


--
-- Name: data_collector_connections; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_collector_connections (
    id uuid NOT NULL,
    project_id uuid NOT NULL,
    protocol_family text NOT NULL,
    driver_id text NOT NULL,
    driver_version text NOT NULL,
    schema_version integer NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    name text NOT NULL,
    code text NOT NULL,
    status text DEFAULT 'unknown'::text NOT NULL,
    display_order integer DEFAULT 0 NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid,
    CONSTRAINT data_collector_connections_code_check CHECK (((char_length(code) >= 1) AND (char_length(code) <= 100))),
    CONSTRAINT data_collector_connections_config_check CHECK ((jsonb_typeof(config) = 'object'::text)),
    CONSTRAINT data_collector_connections_display_order_check CHECK ((display_order >= 0)),
    CONSTRAINT data_collector_connections_driver_id_check CHECK (((driver_id = lower(driver_id)) AND (driver_id ~ '^[a-z0-9][a-z0-9.-]*$'::text))),
    CONSTRAINT data_collector_connections_driver_version_check CHECK (((char_length(driver_version) >= 1) AND (char_length(driver_version) <= 50))),
    CONSTRAINT data_collector_connections_metadata_check CHECK ((jsonb_typeof(metadata) = 'object'::text)),
    CONSTRAINT data_collector_connections_name_check CHECK (((char_length(TRIM(BOTH FROM name)) >= 1) AND (char_length(TRIM(BOTH FROM name)) <= 100))),
    CONSTRAINT data_collector_connections_protocol_family_check CHECK (((protocol_family = lower(protocol_family)) AND (protocol_family ~ '^[a-z0-9][a-z0-9-]*$'::text))),
    CONSTRAINT data_collector_connections_schema_version_check CHECK ((schema_version > 0)),
    CONSTRAINT data_collector_connections_status_check CHECK ((status = ANY (ARRAY['unknown'::text, 'connected'::text, 'disconnected'::text, 'error'::text])))
);


--
-- Name: data_collector_import_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_collector_import_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    driver_id text NOT NULL,
    status text DEFAULT 'preview'::text NOT NULL,
    candidates jsonb DEFAULT '[]'::jsonb NOT NULL,
    errors jsonb DEFAULT '[]'::jsonb NOT NULL,
    total_rows integer DEFAULT 0 NOT NULL,
    valid_rows integer DEFAULT 0 NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_by uuid NOT NULL,
    committed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_collector_import_sessions_candidates_check CHECK ((jsonb_typeof(candidates) = 'array'::text)),
    CONSTRAINT data_collector_import_sessions_check CHECK (((valid_rows >= 0) AND (valid_rows <= total_rows))),
    CONSTRAINT data_collector_import_sessions_errors_check CHECK ((jsonb_typeof(errors) = 'array'::text)),
    CONSTRAINT data_collector_import_sessions_status_check CHECK ((status = ANY (ARRAY['preview'::text, 'committed'::text, 'expired'::text]))),
    CONSTRAINT data_collector_import_sessions_total_rows_check CHECK ((total_rows >= 0))
);


--
-- Name: data_collector_point_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_collector_point_groups (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    parent_id uuid,
    name text NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_collector_point_groups_metadata_check CHECK ((jsonb_typeof(metadata) = 'object'::text)),
    CONSTRAINT data_collector_point_groups_name_check CHECK (((char_length(TRIM(BOTH FROM name)) >= 1) AND (char_length(TRIM(BOTH FROM name)) <= 100)))
);


--
-- Name: data_collector_points; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_collector_points (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    group_id uuid,
    code text NOT NULL,
    name text NOT NULL,
    description text,
    address jsonb NOT NULL,
    address_text text NOT NULL,
    address_schema_version integer NOT NULL,
    data_type text NOT NULL,
    element_count integer DEFAULT 1 NOT NULL,
    read_options jsonb DEFAULT '{}'::jsonb NOT NULL,
    acquisition jsonb DEFAULT '{}'::jsonb NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_collector_points_acquisition_check CHECK ((jsonb_typeof(acquisition) = 'object'::text)),
    CONSTRAINT data_collector_points_address_check CHECK ((jsonb_typeof(address) = 'object'::text)),
    CONSTRAINT data_collector_points_address_schema_version_check CHECK ((address_schema_version > 0)),
    CONSTRAINT data_collector_points_address_text_check CHECK (((char_length(TRIM(BOTH FROM address_text)) >= 1) AND (char_length(TRIM(BOTH FROM address_text)) <= 500))),
    CONSTRAINT data_collector_points_code_check CHECK (((char_length(TRIM(BOTH FROM code)) >= 1) AND (char_length(TRIM(BOTH FROM code)) <= 100))),
    CONSTRAINT data_collector_points_data_type_check CHECK ((data_type = ANY (ARRAY['bool'::text, 'int8'::text, 'uint8'::text, 'int16'::text, 'uint16'::text, 'int32'::text, 'uint32'::text, 'int64'::text, 'uint64'::text, 'float32'::text, 'float64'::text, 'decimal'::text, 'string'::text, 'bytes'::text, 'datetime'::text]))),
    CONSTRAINT data_collector_points_element_count_check CHECK ((element_count >= 1)),
    CONSTRAINT data_collector_points_metadata_check CHECK ((jsonb_typeof(metadata) = 'object'::text)),
    CONSTRAINT data_collector_points_name_check CHECK (((char_length(TRIM(BOTH FROM name)) >= 1) AND (char_length(TRIM(BOTH FROM name)) <= 200))),
    CONSTRAINT data_collector_points_read_options_check CHECK ((jsonb_typeof(read_options) = 'object'::text))
);


--
-- Name: data_collector_point_debug_snapshots; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_collector_point_debug_snapshots (
    point_id uuid NOT NULL,
    value jsonb,
    value_text text,
    data_type text,
    quality text,
    source_timestamp timestamp with time zone,
    server_timestamp timestamp with time zone,
    read_at timestamp with time zone,
    last_attempt_status text NOT NULL,
    last_attempt_at timestamp with time zone NOT NULL,
    last_error_code text,
    last_error_message text,
    CONSTRAINT data_collector_point_debug_snapshots_status_check CHECK ((last_attempt_status = ANY (ARRAY['succeeded'::text, 'failed'::text])))
);


--
-- Name: data_compute_folders; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_compute_folders (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    name text NOT NULL,
    parent_id uuid,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_compute_folders_name_check CHECK ((char_length(name) <= 100))
);


--
-- Name: data_compute_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_compute_runs (
    id bigint NOT NULL,
    project_id uuid NOT NULL,
    compute_unit_id uuid NOT NULL,
    trigger_mode text NOT NULL,
    status text NOT NULL,
    duration_ms integer DEFAULT 0 NOT NULL,
    output jsonb DEFAULT '{}'::jsonb NOT NULL,
    error_message text,
    started_at timestamp with time zone NOT NULL,
    finished_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_compute_runs_duration_ms_check CHECK ((duration_ms >= 0)),
    CONSTRAINT data_compute_runs_status_check CHECK ((status = ANY (ARRAY['success'::text, 'timeout'::text, 'failed'::text]))),
    CONSTRAINT data_compute_runs_trigger_mode_check CHECK ((trigger_mode = ANY (ARRAY['run'::text, 'debug'::text, 'schedule'::text])))
);


--
-- Name: data_compute_runs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE data_compute_runs ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME data_compute_runs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: data_compute_units; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_compute_units (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    name text NOT NULL,
    language text NOT NULL,
    description text,
    folder_id uuid,
    script_code text NOT NULL,
    dependencies jsonb DEFAULT '[]'::jsonb NOT NULL,
    trigger_type text DEFAULT 'manual'::text NOT NULL,
    trigger_config jsonb DEFAULT '{}'::jsonb NOT NULL,
    input_bindings jsonb DEFAULT '{}'::jsonb NOT NULL,
    output_bindings jsonb DEFAULT '{}'::jsonb NOT NULL,
    timeout_ms integer DEFAULT 3000 NOT NULL,
    is_enabled boolean DEFAULT true NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_compute_units_dependencies_check CHECK ((jsonb_typeof(dependencies) = 'array'::text)),
    CONSTRAINT data_compute_units_input_bindings_check CHECK ((jsonb_typeof(input_bindings) = 'object'::text)),
    CONSTRAINT data_compute_units_language_check CHECK ((language = ANY (ARRAY['js'::text, 'python'::text]))),
    CONSTRAINT data_compute_units_name_check CHECK ((char_length(name) <= 100)),
    CONSTRAINT data_compute_units_output_bindings_check CHECK ((jsonb_typeof(output_bindings) = 'object'::text)),
    CONSTRAINT data_compute_units_timeout_ms_check CHECK (((timeout_ms > 0) AND (timeout_ms <= 120000))),
    CONSTRAINT data_compute_units_trigger_config_check CHECK ((jsonb_typeof(trigger_config) = 'object'::text)),
    CONSTRAINT data_compute_units_trigger_type_check CHECK ((trigger_type = ANY (ARRAY['manual'::text, 'timer'::text, 'datapoint_change'::text])))
);


--
-- Name: data_connections; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_connections (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    name text NOT NULL,
    type text NOT NULL,
    category text DEFAULT 'api'::text NOT NULL,
    status text DEFAULT 'unknown'::text NOT NULL,
    is_enabled boolean DEFAULT true NOT NULL,
    retry_count integer DEFAULT 3 NOT NULL,
    retry_interval_ms integer DEFAULT 5000 NOT NULL,
    health_check_interval_ms integer DEFAULT 30000 NOT NULL,
    last_connected_at timestamp with time zone,
    last_error_message text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    display_order integer DEFAULT 0 NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_connections_category_check CHECK ((category = ANY (ARRAY['database'::text, 'message'::text, 'protocol'::text, 'api'::text, 'builtin'::text]))),
    CONSTRAINT data_connections_health_check_interval_ms_check CHECK ((health_check_interval_ms >= 0)),
    CONSTRAINT data_connections_metadata_check CHECK ((jsonb_typeof(metadata) = 'object'::text)),
    CONSTRAINT data_connections_name_check CHECK ((char_length(name) <= 100)),
    CONSTRAINT data_connections_retry_count_check CHECK ((retry_count >= 0)),
    CONSTRAINT data_connections_retry_interval_ms_check CHECK ((retry_interval_ms >= 0)),
    CONSTRAINT data_connections_status_check CHECK ((status = ANY (ARRAY['connected'::text, 'disconnected'::text, 'error'::text, 'unknown'::text]))),
    CONSTRAINT data_connections_type_check CHECK ((type = ANY (ARRAY['relational'::text, 'mqtt'::text, 'websocket'::text, 'http'::text, 'kafka'::text, 'redis'::text, 'tdengine'::text, 'builtin.relation'::text, 'builtin.timeseries'::text, 'builtin.realtime'::text, 'builtin.message'::text])))
);


--
-- Name: data_contract_check_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_contract_check_runs (
    id bigint NOT NULL,
    project_id uuid NOT NULL,
    scope text DEFAULT 'project'::text NOT NULL,
    object_type text,
    object_id text,
    status text NOT NULL,
    summary jsonb DEFAULT '{}'::jsonb NOT NULL,
    result jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_contract_check_runs_result_check CHECK ((jsonb_typeof(result) = 'object'::text)),
    CONSTRAINT data_contract_check_runs_status_check CHECK ((status = ANY (ARRAY['passed'::text, 'warning'::text, 'pending'::text, 'failed'::text]))),
    CONSTRAINT data_contract_check_runs_summary_check CHECK ((jsonb_typeof(summary) = 'object'::text))
);


--
-- Name: data_contract_check_runs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE data_contract_check_runs ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME data_contract_check_runs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: data_http_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_http_configs (
    connection_id uuid NOT NULL,
    base_url text NOT NULL,
    method text DEFAULT 'GET'::text NOT NULL,
    headers jsonb DEFAULT '{}'::jsonb NOT NULL,
    timeout_ms integer DEFAULT 30000 NOT NULL,
    body_template jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_http_configs_base_url_check CHECK ((char_length(base_url) <= 1000)),
    CONSTRAINT data_http_configs_body_template_check CHECK (((body_template IS NULL) OR (jsonb_typeof(body_template) = 'object'::text))),
    CONSTRAINT data_http_configs_headers_check CHECK ((jsonb_typeof(headers) = 'object'::text)),
    CONSTRAINT data_http_configs_method_check CHECK ((method = ANY (ARRAY['GET'::text, 'POST'::text, 'PUT'::text, 'PATCH'::text, 'DELETE'::text]))),
    CONSTRAINT data_http_configs_timeout_ms_check CHECK ((timeout_ms >= 0))
);


--
-- Name: data_http_request_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_http_request_groups (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    parent_id uuid,
    name text NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_http_request_groups_name_check CHECK ((char_length(name) <= 100))
);


--
-- Name: data_http_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_http_requests (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    group_id uuid,
    name text NOT NULL,
    method text DEFAULT 'GET'::text NOT NULL,
    url text NOT NULL,
    params jsonb DEFAULT '[]'::jsonb NOT NULL,
    headers jsonb DEFAULT '[]'::jsonb NOT NULL,
    auth jsonb DEFAULT '{}'::jsonb NOT NULL,
    body_type text DEFAULT 'none'::text NOT NULL,
    body jsonb DEFAULT '{}'::jsonb NOT NULL,
    settings jsonb DEFAULT '{}'::jsonb NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    quality text DEFAULT 'unknown'::text NOT NULL,
    last_sent_at timestamp with time zone,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_http_requests_auth_check CHECK ((jsonb_typeof(auth) = 'object'::text)),
    CONSTRAINT data_http_requests_body_check CHECK ((jsonb_typeof(body) = 'object'::text)),
    CONSTRAINT data_http_requests_body_type_check CHECK ((body_type = ANY (ARRAY['none'::text, 'json'::text, 'raw'::text, 'form-data'::text, 'x-www-form-urlencoded'::text]))),
    CONSTRAINT data_http_requests_headers_check CHECK ((jsonb_typeof(headers) = 'array'::text)),
    CONSTRAINT data_http_requests_method_check CHECK ((method = ANY (ARRAY['GET'::text, 'POST'::text, 'PUT'::text, 'PATCH'::text, 'DELETE'::text]))),
    CONSTRAINT data_http_requests_name_check CHECK ((char_length(name) <= 100)),
    CONSTRAINT data_http_requests_params_check CHECK ((jsonb_typeof(params) = 'array'::text)),
    CONSTRAINT data_http_requests_quality_check CHECK ((quality = ANY (ARRAY['good'::text, 'bad'::text, 'unknown'::text]))),
    CONSTRAINT data_http_requests_settings_check CHECK ((jsonb_typeof(settings) = 'object'::text)),
    CONSTRAINT data_http_requests_url_check CHECK ((char_length(url) <= 2000))
);


--
-- Name: data_kafka_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_kafka_configs (
    connection_id uuid NOT NULL,
    brokers text NOT NULL,
    topic text,
    consumer_group text,
    start_position text DEFAULT 'latest'::text NOT NULL,
    options jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_kafka_configs_brokers_check CHECK ((char_length(brokers) <= 500)),
    CONSTRAINT data_kafka_configs_consumer_group_check CHECK (((consumer_group IS NULL) OR (char_length(consumer_group) <= 200))),
    CONSTRAINT data_kafka_configs_options_check CHECK ((jsonb_typeof(options) = 'object'::text)),
    CONSTRAINT data_kafka_configs_start_position_check CHECK ((start_position = ANY (ARRAY['latest'::text, 'earliest'::text]))),
    CONSTRAINT data_kafka_configs_topic_check CHECK (((topic IS NULL) OR (char_length(topic) <= 500)))
);


--
-- Name: data_kafka_field_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_kafka_field_groups (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    topic_mapping_id uuid NOT NULL,
    parent_id uuid,
    name text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_kafka_field_groups_name_check CHECK ((char_length(name) <= 100))
);


--
-- Name: data_kafka_fields; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_kafka_fields (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    topic_mapping_id uuid NOT NULL,
    group_id uuid,
    name text NOT NULL,
    value_path text NOT NULL,
    key_path text DEFAULT ''::text NOT NULL,
    data_type text NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    last_value jsonb,
    quality text DEFAULT 'unknown'::text NOT NULL,
    last_updated_at timestamp with time zone,
    sort_order integer DEFAULT 0 NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_kafka_fields_data_type_check CHECK ((char_length(data_type) <= 50)),
    CONSTRAINT data_kafka_fields_key_path_check CHECK ((char_length(key_path) <= 500)),
    CONSTRAINT data_kafka_fields_name_check CHECK ((char_length(name) <= 100)),
    CONSTRAINT data_kafka_fields_quality_check CHECK ((quality = ANY (ARRAY['good'::text, 'bad'::text, 'unknown'::text]))),
    CONSTRAINT data_kafka_fields_value_path_check CHECK ((char_length(value_path) <= 500))
);


--
-- Name: data_kafka_topic_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_kafka_topic_groups (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    parent_id uuid,
    name text NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_kafka_topic_groups_name_check CHECK ((char_length(name) <= 100))
);


--
-- Name: data_kafka_topic_mappings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_kafka_topic_mappings (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    group_id uuid,
    name text NOT NULL,
    topic text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    consumer_group text DEFAULT ''::text NOT NULL,
    output_mode text DEFAULT 'field_mapping'::text NOT NULL,
    raw_output_scope text DEFAULT 'value'::text NOT NULL,
    partition_mode text DEFAULT 'all'::text NOT NULL,
    partition integer,
    start_position text DEFAULT 'latest'::text NOT NULL,
    start_offset bigint,
    decode text DEFAULT 'json'::text NOT NULL,
    sample_limit integer DEFAULT 100 NOT NULL,
    timeout_ms integer DEFAULT 5000 NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_kafka_topic_mappings_consumer_group_check CHECK ((char_length(consumer_group) <= 200)),
    CONSTRAINT data_kafka_topic_mappings_decode_check CHECK ((decode = ANY (ARRAY['json'::text, 'string'::text, 'binary'::text]))),
    CONSTRAINT data_kafka_topic_mappings_name_check CHECK ((char_length(name) <= 100)),
    CONSTRAINT data_kafka_topic_mappings_offset_check CHECK (((start_position <> 'offset'::text) OR ((partition_mode = 'single'::text) AND (partition IS NOT NULL) AND (start_offset IS NOT NULL)))),
    CONSTRAINT data_kafka_topic_mappings_output_mode_check CHECK ((output_mode = ANY (ARRAY['raw_message'::text, 'field_mapping'::text]))),
    CONSTRAINT data_kafka_topic_mappings_partition_check CHECK (((partition IS NULL) OR (partition >= 0))),
    CONSTRAINT data_kafka_topic_mappings_partition_mode_check CHECK ((partition_mode = ANY (ARRAY['all'::text, 'single'::text]))),
    CONSTRAINT data_kafka_topic_mappings_raw_output_scope_check CHECK ((raw_output_scope = ANY (ARRAY['value'::text, 'full_message'::text]))),
    CONSTRAINT data_kafka_topic_mappings_sample_limit_check CHECK (((sample_limit >= 1) AND (sample_limit <= 1000))),
    CONSTRAINT data_kafka_topic_mappings_single_partition_check CHECK (((partition_mode <> 'single'::text) OR (partition IS NOT NULL))),
    CONSTRAINT data_kafka_topic_mappings_start_position_check CHECK ((start_position = ANY (ARRAY['latest'::text, 'earliest'::text, 'offset'::text]))),
    CONSTRAINT data_kafka_topic_mappings_timeout_ms_check CHECK (((timeout_ms >= 1000) AND (timeout_ms <= 30000))),
    CONSTRAINT data_kafka_topic_mappings_topic_check CHECK ((char_length(topic) <= 500))
);


--
-- Name: data_mqtt_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_mqtt_configs (
    connection_id uuid NOT NULL,
    broker_url text NOT NULL,
    protocol text DEFAULT 'mqtt'::text NOT NULL,
    port integer DEFAULT 1883 NOT NULL,
    client_id text,
    username text,
    password text,
    keepalive integer DEFAULT 60 NOT NULL,
    clean_session boolean DEFAULT true NOT NULL,
    qos smallint DEFAULT 0 NOT NULL,
    reconnect_period_ms integer DEFAULT 5000 NOT NULL,
    connect_timeout_ms integer DEFAULT 30000 NOT NULL,
    will jsonb,
    ssl_config jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_mqtt_configs_broker_url_check CHECK ((char_length(broker_url) <= 500)),
    CONSTRAINT data_mqtt_configs_client_id_check CHECK (((client_id IS NULL) OR (char_length(client_id) <= 100))),
    CONSTRAINT data_mqtt_configs_connect_timeout_ms_check CHECK ((connect_timeout_ms >= 0)),
    CONSTRAINT data_mqtt_configs_keepalive_check CHECK ((keepalive >= 0)),
    CONSTRAINT data_mqtt_configs_port_check CHECK (((port > 0) AND (port <= 65535))),
    CONSTRAINT data_mqtt_configs_protocol_check CHECK ((protocol = ANY (ARRAY['mqtt'::text, 'mqtts'::text, 'ws'::text, 'wss'::text]))),
    CONSTRAINT data_mqtt_configs_qos_check CHECK ((qos = ANY (ARRAY[0, 1, 2]))),
    CONSTRAINT data_mqtt_configs_reconnect_period_ms_check CHECK ((reconnect_period_ms >= 0)),
    CONSTRAINT data_mqtt_configs_ssl_config_check CHECK (((ssl_config IS NULL) OR (jsonb_typeof(ssl_config) = 'object'::text))),
    CONSTRAINT data_mqtt_configs_username_check CHECK (((username IS NULL) OR (char_length(username) <= 100))),
    CONSTRAINT data_mqtt_configs_will_check CHECK (((will IS NULL) OR (jsonb_typeof(will) = 'object'::text)))
);


--
-- Name: data_mqtt_messages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_mqtt_messages (
    id bigint NOT NULL,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    subscription_id uuid NOT NULL,
    topic text NOT NULL,
    payload text NOT NULL,
    qos smallint DEFAULT 0 NOT NULL,
    received_at timestamp with time zone DEFAULT now() NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT data_mqtt_messages_metadata_check CHECK ((jsonb_typeof(metadata) = 'object'::text)),
    CONSTRAINT data_mqtt_messages_qos_check CHECK ((qos = ANY (ARRAY[0, 1, 2]))),
    CONSTRAINT data_mqtt_messages_topic_check CHECK ((char_length(topic) <= 500))
);


--
-- Name: data_mqtt_messages_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE data_mqtt_messages ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME data_mqtt_messages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: data_mqtt_subscription_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_mqtt_subscription_groups (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    name text NOT NULL,
    parent_id uuid,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_mqtt_subscription_groups_name_check CHECK ((char_length(name) <= 100))
);


--
-- Name: data_mqtt_subscriptions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_mqtt_subscriptions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    name text NOT NULL,
    topic text NOT NULL,
    qos smallint DEFAULT 0 NOT NULL,
    usage_mode text DEFAULT 'single_variable'::text NOT NULL,
    group_id uuid,
    description text,
    display_order integer DEFAULT 0 NOT NULL,
    message_retention integer DEFAULT 5000 NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    default_batch_parse_rule jsonb,
    CONSTRAINT data_mqtt_subscriptions_message_retention_check CHECK ((message_retention > 0)),
    CONSTRAINT data_mqtt_subscriptions_name_check CHECK ((char_length(name) <= 100)),
    CONSTRAINT data_mqtt_subscriptions_qos_check CHECK ((qos = ANY (ARRAY[0, 1, 2]))),
    CONSTRAINT data_mqtt_subscriptions_topic_check CHECK ((char_length(topic) <= 500)),
    CONSTRAINT data_mqtt_subscriptions_usage_mode_check CHECK ((usage_mode = ANY (ARRAY['raw_datapoint'::text, 'single_variable'::text, 'batch_variable'::text])))
);


--
-- Name: data_mqtt_tags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_mqtt_tags (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    subscription_id uuid NOT NULL,
    name text NOT NULL,
    code text NOT NULL,
    description text,
    data_type text DEFAULT 'string'::text NOT NULL,
    parse_type text DEFAULT 'jsonpath'::text NOT NULL,
    parse_rule text NOT NULL,
    default_value text,
    unit text,
    transform text,
    validation jsonb,
    display_order integer DEFAULT 0 NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_mqtt_tags_code_check CHECK ((char_length(code) <= 100)),
    CONSTRAINT data_mqtt_tags_data_type_check CHECK ((data_type = ANY (ARRAY['string'::text, 'number'::text, 'boolean'::text, 'object'::text, 'array'::text]))),
    CONSTRAINT data_mqtt_tags_name_check CHECK ((char_length(name) <= 100)),
    CONSTRAINT data_mqtt_tags_parse_type_check CHECK ((parse_type = ANY (ARRAY['jsonpath'::text, 'regex'::text, 'script'::text, 'fixed'::text, 'batch_jsonpath'::text]))),
    CONSTRAINT data_mqtt_tags_unit_check CHECK (((unit IS NULL) OR (char_length(unit) <= 50))),
    CONSTRAINT data_mqtt_tags_validation_check CHECK (((validation IS NULL) OR (jsonb_typeof(validation) = 'object'::text)))
);


--
-- Name: data_points; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_points (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    path text NOT NULL,
    name text NOT NULL,
    description text,
    source_type text NOT NULL,
    source_id uuid,
    source_config jsonb DEFAULT '{}'::jsonb NOT NULL,
    data_type text NOT NULL,
    unit text,
    precision_num integer,
    default_value text,
    min_value numeric(20,6),
    max_value numeric(20,6),
    alarm_low numeric(20,6),
    alarm_high numeric(20,6),
    tags jsonb DEFAULT '[]'::jsonb NOT NULL,
    refresh_mode text DEFAULT 'auto'::text NOT NULL,
    refresh_interval_ms integer,
    status text DEFAULT 'active'::text NOT NULL,
    display_order integer DEFAULT 0 NOT NULL,
    runtime_permissions jsonb DEFAULT '{"write": {"inherit": true, "denyRoles": [], "allowRoles": []}}'::jsonb NOT NULL,
    created_by uuid,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_points_alarm_range_check CHECK ((((alarm_low IS NULL) OR (alarm_high IS NULL) OR (alarm_low <= alarm_high)) AND ((min_value IS NULL) OR (alarm_low IS NULL) OR (min_value <= alarm_low)) AND ((max_value IS NULL) OR (alarm_high IS NULL) OR (alarm_high <= max_value)))),
    CONSTRAINT data_points_data_type_check CHECK ((char_length(data_type) <= 20)),
    CONSTRAINT data_points_min_max_check CHECK (((min_value IS NULL) OR (max_value IS NULL) OR (min_value <= max_value))),
    CONSTRAINT data_points_name_check CHECK ((char_length(name) <= 100)),
    CONSTRAINT data_points_path_check CHECK ((char_length(path) <= 255)),
    CONSTRAINT data_points_precision_num_check CHECK (((precision_num IS NULL) OR (precision_num >= 0))),
    CONSTRAINT data_points_refresh_interval_ms_check CHECK (((refresh_interval_ms IS NULL) OR (refresh_interval_ms >= 0))),
    CONSTRAINT data_points_refresh_mode_check CHECK ((refresh_mode = ANY (ARRAY['auto'::text, 'manual'::text, 'subscription'::text]))),
    CONSTRAINT data_points_runtime_permissions_check CHECK ((jsonb_typeof(runtime_permissions) = 'object'::text)),
    CONSTRAINT data_points_source_config_check CHECK ((jsonb_typeof(source_config) = 'object'::text)),
    CONSTRAINT data_points_source_type_check CHECK ((char_length(source_type) <= 50)),
    CONSTRAINT data_points_status_check CHECK ((status = ANY (ARRAY['active'::text, 'inactive'::text, 'invalid'::text]))),
    CONSTRAINT data_points_tags_check CHECK ((jsonb_typeof(tags) = 'array'::text)),
    CONSTRAINT data_points_unit_check CHECK (((unit IS NULL) OR (char_length(unit) <= 20)))
);


--
-- Name: data_preview_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_preview_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    user_id uuid NOT NULL,
    status text DEFAULT 'active'::text NOT NULL,
    started_at timestamp with time zone DEFAULT now() NOT NULL,
    last_active_at timestamp with time zone DEFAULT now() NOT NULL,
    expired_at timestamp with time zone NOT NULL,
    meta jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_preview_sessions_meta_check CHECK ((jsonb_typeof(meta) = 'object'::text)),
    CONSTRAINT data_preview_sessions_status_check CHECK ((status = ANY (ARRAY['active'::text, 'expired'::text, 'closed'::text, 'error'::text])))
);


--
-- Name: data_queries; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_queries (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    group_id uuid,
    name text NOT NULL,
    description text,
    category text,
    query_type text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    transformer text,
    is_enabled boolean DEFAULT true NOT NULL,
    timeout_ms integer DEFAULT 30000 NOT NULL,
    cache_enabled boolean DEFAULT false NOT NULL,
    cache_ttl_seconds integer DEFAULT 300 NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_queries_cache_ttl_seconds_check CHECK ((cache_ttl_seconds >= 0)),
    CONSTRAINT data_queries_config_check CHECK ((jsonb_typeof(config) = 'object'::text)),
    CONSTRAINT data_queries_name_check CHECK ((char_length(name) <= 200)),
    CONSTRAINT data_queries_query_type_check CHECK ((query_type = ANY (ARRAY['sql'::text, 'tags'::text, 'http'::text, 'mqtt_pub'::text, 'mqtt_sub'::text]))),
    CONSTRAINT data_queries_timeout_ms_check CHECK ((timeout_ms >= 0))
);


--
-- Name: data_realtime_keys; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_realtime_keys (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    provider text NOT NULL,
    key_path text NOT NULL,
    redis_type text DEFAULT 'string'::text NOT NULL,
    value_type text DEFAULT 'object'::text NOT NULL,
    default_ttl_seconds integer DEFAULT 0 NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_realtime_keys_provider_check CHECK ((provider = ANY (ARRAY['redis'::text, 'builtin'::text])))
);


--
-- Name: data_redis_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_redis_configs (
    connection_id uuid NOT NULL,
    address text NOT NULL,
    db integer DEFAULT 0 NOT NULL,
    username text,
    password text,
    key_pattern text DEFAULT '*'::text NOT NULL,
    mode text DEFAULT 'standalone'::text NOT NULL,
    options jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_redis_configs_address_check CHECK ((char_length(address) <= 255)),
    CONSTRAINT data_redis_configs_db_check CHECK ((db >= 0)),
    CONSTRAINT data_redis_configs_key_pattern_check CHECK ((char_length(key_pattern) <= 255)),
    CONSTRAINT data_redis_configs_mode_check CHECK ((mode = ANY (ARRAY['standalone'::text, 'sentinel'::text, 'cluster'::text]))),
    CONSTRAINT data_redis_configs_options_check CHECK ((jsonb_typeof(options) = 'object'::text)),
    CONSTRAINT data_redis_configs_username_check CHECK (((username IS NULL) OR (char_length(username) <= 100)))
);


--
-- Name: data_relational_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_relational_configs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    connection_id uuid NOT NULL,
    db_type text NOT NULL,
    host text NOT NULL,
    port integer NOT NULL,
    database text NOT NULL,
    username text NOT NULL,
    password text NOT NULL,
    schema text,
    charset text DEFAULT 'utf8mb4'::text NOT NULL,
    timezone text,
    ssl boolean DEFAULT false NOT NULL,
    ssl_config jsonb,
    pool_min integer DEFAULT 2 NOT NULL,
    pool_max integer DEFAULT 10 NOT NULL,
    acquire_timeout_ms integer DEFAULT 60000 NOT NULL,
    idle_timeout_ms integer DEFAULT 30000 NOT NULL,
    query_timeout_ms integer DEFAULT 60000 NOT NULL,
    options jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_relational_configs_acquire_timeout_ms_check CHECK ((acquire_timeout_ms >= 0)),
    CONSTRAINT data_relational_configs_check CHECK (((pool_max > 0) AND (pool_max >= pool_min))),
    CONSTRAINT data_relational_configs_database_check CHECK ((char_length(database) <= 100)),
    CONSTRAINT data_relational_configs_db_type_check CHECK ((db_type = ANY (ARRAY['mysql'::text, 'postgresql'::text, 'sqlserver'::text, 'oracle'::text, 'sqlite'::text, 'clickhouse'::text]))),
    CONSTRAINT data_relational_configs_host_check CHECK ((char_length(host) <= 255)),
    CONSTRAINT data_relational_configs_idle_timeout_ms_check CHECK ((idle_timeout_ms >= 0)),
    CONSTRAINT data_relational_configs_options_check CHECK ((jsonb_typeof(options) = 'object'::text)),
    CONSTRAINT data_relational_configs_pool_min_check CHECK ((pool_min >= 0)),
    CONSTRAINT data_relational_configs_port_check CHECK (((port > 0) AND (port <= 65535))),
    CONSTRAINT data_relational_configs_query_timeout_ms_check CHECK ((query_timeout_ms >= 0)),
    CONSTRAINT data_relational_configs_ssl_config_check CHECK (((ssl_config IS NULL) OR (jsonb_typeof(ssl_config) = 'object'::text))),
    CONSTRAINT data_relational_configs_username_check CHECK ((char_length(username) <= 100))
);


--
-- Name: data_history_storage_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_history_storage_configs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    access_source_id uuid,
    collector_connection_id uuid,
    datapoint_id uuid,
    is_enabled boolean DEFAULT true NOT NULL,
    write_mode text DEFAULT 'on_change'::text NOT NULL,
    interval_ms bigint,
    deadband numeric(20,6),
    max_silence_ms bigint,
    offline_behavior text DEFAULT 'store_stale'::text NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_history_storage_configs_scope_check CHECK ((num_nonnulls(access_source_id, collector_connection_id, datapoint_id) = 1)),
    CONSTRAINT data_history_storage_configs_write_mode_check CHECK ((write_mode = ANY (ARRAY['every_sample'::text, 'interval_latest'::text, 'on_change'::text, 'periodic_snapshot'::text]))),
    CONSTRAINT data_history_storage_configs_interval_check CHECK (((interval_ms IS NULL) OR (interval_ms > 0))),
    CONSTRAINT data_history_storage_configs_deadband_check CHECK (((deadband IS NULL) OR (deadband >= (0)::numeric))),
    CONSTRAINT data_history_storage_configs_max_silence_check CHECK (((max_silence_ms IS NULL) OR (max_silence_ms > 0))),
    CONSTRAINT data_history_storage_configs_offline_behavior_check CHECK ((offline_behavior = ANY (ARRAY['store_stale'::text, 'skip'::text])))
);


--
-- Name: data_history_storage_targets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_history_storage_targets (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    config_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    is_primary boolean DEFAULT false NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    retention_days bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_history_storage_targets_retention_check CHECK (((retention_days IS NULL) OR (retention_days > 0))),
    CONSTRAINT data_history_storage_targets_sort_order_check CHECK ((sort_order >= 0))
);


--
-- Name: data_table_group_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_table_group_members (
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    table_name text NOT NULL,
    group_id uuid NOT NULL,
    updated_by uuid,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_table_group_members_table_name_check CHECK (((char_length(table_name) >= 1) AND (char_length(table_name) <= 255)))
);


--
-- Name: data_tdengine_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_tdengine_configs (
    connection_id uuid NOT NULL,
    dsn text NOT NULL,
    database_name text NOT NULL,
    timezone text,
    options jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_tdengine_configs_database_name_check CHECK ((char_length(database_name) <= 128)),
    CONSTRAINT data_tdengine_configs_dsn_check CHECK ((char_length(dsn) <= 1000)),
    CONSTRAINT data_tdengine_configs_options_check CHECK ((jsonb_typeof(options) = 'object'::text))
);


--
-- Name: data_websocket_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_websocket_configs (
    connection_id uuid NOT NULL,
    url text,
    topic text,
    headers jsonb DEFAULT '{}'::jsonb NOT NULL,
    heartbeat_interval_ms integer DEFAULT 30000 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_websocket_configs_headers_check CHECK ((jsonb_typeof(headers) = 'object'::text)),
    CONSTRAINT data_websocket_configs_heartbeat_interval_ms_check CHECK ((heartbeat_interval_ms >= 0)),
    CONSTRAINT data_websocket_configs_topic_check CHECK (((topic IS NULL) OR (char_length(topic) <= 500))),
    CONSTRAINT data_websocket_configs_url_check CHECK ((char_length(url) <= 1000))
);


--
-- Name: data_websocket_session_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_websocket_session_groups (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    parent_id uuid,
    name text NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_websocket_session_groups_name_check CHECK ((char_length(name) <= 100))
);


--
-- Name: data_websocket_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_websocket_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    group_id uuid,
    name text NOT NULL,
    url text NOT NULL,
    headers jsonb DEFAULT '[]'::jsonb NOT NULL,
    auth jsonb DEFAULT '{}'::jsonb NOT NULL,
    protocols jsonb DEFAULT '[]'::jsonb NOT NULL,
    messages jsonb DEFAULT '[]'::jsonb NOT NULL,
    settings jsonb DEFAULT '{}'::jsonb NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    last_message jsonb,
    last_diagnostic text,
    quality text DEFAULT 'unknown'::text NOT NULL,
    last_message_at timestamp with time zone,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_websocket_sessions_auth_check CHECK ((jsonb_typeof(auth) = 'object'::text)),
    CONSTRAINT data_websocket_sessions_headers_check CHECK ((jsonb_typeof(headers) = 'array'::text)),
    CONSTRAINT data_websocket_sessions_messages_check CHECK ((jsonb_typeof(messages) = 'array'::text)),
    CONSTRAINT data_websocket_sessions_name_check CHECK ((char_length(name) <= 100)),
    CONSTRAINT data_websocket_sessions_protocols_check CHECK ((jsonb_typeof(protocols) = 'array'::text)),
    CONSTRAINT data_websocket_sessions_quality_check CHECK ((quality = ANY (ARRAY['good'::text, 'bad'::text, 'unknown'::text]))),
    CONSTRAINT data_websocket_sessions_settings_check CHECK ((jsonb_typeof(settings) = 'object'::text)),
    CONSTRAINT data_websocket_sessions_url_check CHECK ((char_length(url) <= 2000))
);


--
-- Name: data_workbench_object_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE data_workbench_object_groups (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    scope text NOT NULL,
    name text NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT data_workbench_object_groups_name_check CHECK (((char_length(name) >= 1) AND (char_length(name) <= 100))),
    CONSTRAINT data_workbench_object_groups_scope_check CHECK ((scope = ANY (ARRAY['query'::text, 'table'::text])))
);


--
-- Name: collector_dev_agents collector_dev_agents_credential_hash_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY collector_dev_agents
    ADD CONSTRAINT collector_dev_agents_credential_hash_key UNIQUE (credential_hash);


--
-- Name: collector_dev_agents collector_dev_agents_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY collector_dev_agents
    ADD CONSTRAINT collector_dev_agents_pkey PRIMARY KEY (id);


--
-- Name: collector_dev_registration_codes collector_dev_registration_codes_code_hash_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY collector_dev_registration_codes
    ADD CONSTRAINT collector_dev_registration_codes_code_hash_key UNIQUE (code_hash);


--
-- Name: collector_dev_registration_codes collector_dev_registration_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY collector_dev_registration_codes
    ADD CONSTRAINT collector_dev_registration_codes_pkey PRIMARY KEY (id);


--
-- Name: collector_dev_tasks collector_dev_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY collector_dev_tasks
    ADD CONSTRAINT collector_dev_tasks_pkey PRIMARY KEY (id);


--
-- Name: data_access_source_records data_access_source_records_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_access_source_records
    ADD CONSTRAINT data_access_source_records_pkey PRIMARY KEY (id);


--
-- Name: data_alarm_policies data_alarm_policies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_alarm_policies
    ADD CONSTRAINT data_alarm_policies_pkey PRIMARY KEY (id);


--
-- Name: data_alarm_policies data_alarm_policies_project_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_alarm_policies
    ADD CONSTRAINT data_alarm_policies_project_name_key UNIQUE (project_id, name);


--
-- Name: data_alarm_policy_groups data_alarm_policy_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_alarm_policy_groups
    ADD CONSTRAINT data_alarm_policy_groups_pkey PRIMARY KEY (id);


--
-- Name: data_alarm_project_settings data_alarm_project_settings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_alarm_project_settings
    ADD CONSTRAINT data_alarm_project_settings_pkey PRIMARY KEY (project_id);


--
-- Name: data_alarm_rules data_alarm_rules_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_alarm_rules
    ADD CONSTRAINT data_alarm_rules_pkey PRIMARY KEY (id);


--
-- Name: data_alarm_rules data_alarm_rules_project_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_alarm_rules
    ADD CONSTRAINT data_alarm_rules_project_name_key UNIQUE (project_id, name);


--
-- Name: data_collector_connection_secrets data_collector_connection_secrets_identity_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_connection_secrets
    ADD CONSTRAINT data_collector_connection_secrets_identity_key UNIQUE (connection_id, secret_key);


--
-- Name: data_collector_connection_secrets data_collector_connection_secrets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_connection_secrets
    ADD CONSTRAINT data_collector_connection_secrets_pkey PRIMARY KEY (id);


--
-- Name: data_collector_connections data_collector_connections_identity_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_connections
    ADD CONSTRAINT data_collector_connections_identity_key UNIQUE (id, project_id);


--
-- Name: data_collector_connections data_collector_connections_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_connections
    ADD CONSTRAINT data_collector_connections_pkey PRIMARY KEY (id);

ALTER TABLE ONLY data_collector_connections
    ADD CONSTRAINT data_collector_connections_project_code_key UNIQUE (project_id, code);

CREATE UNIQUE INDEX data_collector_connections_project_name_key ON data_collector_connections USING btree (project_id, lower(name));


--
-- Name: data_collector_import_sessions data_collector_import_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_import_sessions
    ADD CONSTRAINT data_collector_import_sessions_pkey PRIMARY KEY (id);


--
-- Name: data_collector_point_groups data_collector_point_groups_identity_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_point_groups
    ADD CONSTRAINT data_collector_point_groups_identity_key UNIQUE (id, project_id, connection_id);


--
-- Name: data_collector_point_groups data_collector_point_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_point_groups
    ADD CONSTRAINT data_collector_point_groups_pkey PRIMARY KEY (id);


--
-- Name: data_collector_point_debug_snapshots data_collector_point_debug_snapshots_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_point_debug_snapshots
    ADD CONSTRAINT data_collector_point_debug_snapshots_pkey PRIMARY KEY (point_id);


--
-- Name: data_collector_points data_collector_points_connection_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_points
    ADD CONSTRAINT data_collector_points_connection_code_key UNIQUE (project_id, connection_id, code);


--
-- Name: data_collector_points data_collector_points_connection_address_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_points
    ADD CONSTRAINT data_collector_points_connection_address_key UNIQUE (project_id, connection_id, address_text);


--
-- Name: data_collector_points data_collector_points_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_points
    ADD CONSTRAINT data_collector_points_pkey PRIMARY KEY (id);


--
-- Name: data_compute_folders data_compute_folders_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_compute_folders
    ADD CONSTRAINT data_compute_folders_pkey PRIMARY KEY (id);


--
-- Name: data_compute_runs data_compute_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_compute_runs
    ADD CONSTRAINT data_compute_runs_pkey PRIMARY KEY (id);


--
-- Name: data_compute_units data_compute_units_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_compute_units
    ADD CONSTRAINT data_compute_units_pkey PRIMARY KEY (id);


--
-- Name: data_compute_units data_compute_units_project_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_compute_units
    ADD CONSTRAINT data_compute_units_project_name_key UNIQUE (project_id, name);


--
-- Name: data_connections data_connections_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_connections
    ADD CONSTRAINT data_connections_pkey PRIMARY KEY (id);


--
-- Name: data_contract_check_runs data_contract_check_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_contract_check_runs
    ADD CONSTRAINT data_contract_check_runs_pkey PRIMARY KEY (id);


--
-- Name: data_http_configs data_http_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_http_configs
    ADD CONSTRAINT data_http_configs_pkey PRIMARY KEY (connection_id);


--
-- Name: data_http_request_groups data_http_request_groups_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_http_request_groups
    ADD CONSTRAINT data_http_request_groups_name_key UNIQUE (project_id, connection_id, parent_id, name);


--
-- Name: data_http_request_groups data_http_request_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_http_request_groups
    ADD CONSTRAINT data_http_request_groups_pkey PRIMARY KEY (id);


--
-- Name: data_http_requests data_http_requests_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_http_requests
    ADD CONSTRAINT data_http_requests_name_key UNIQUE (project_id, connection_id, group_id, name);


--
-- Name: data_http_requests data_http_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_http_requests
    ADD CONSTRAINT data_http_requests_pkey PRIMARY KEY (id);


--
-- Name: data_kafka_configs data_kafka_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_configs
    ADD CONSTRAINT data_kafka_configs_pkey PRIMARY KEY (connection_id);


--
-- Name: data_kafka_field_groups data_kafka_field_groups_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_field_groups
    ADD CONSTRAINT data_kafka_field_groups_name_key UNIQUE (project_id, topic_mapping_id, parent_id, name);


--
-- Name: data_kafka_field_groups data_kafka_field_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_field_groups
    ADD CONSTRAINT data_kafka_field_groups_pkey PRIMARY KEY (id);


--
-- Name: data_kafka_fields data_kafka_fields_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_fields
    ADD CONSTRAINT data_kafka_fields_pkey PRIMARY KEY (id);


--
-- Name: data_kafka_fields data_kafka_fields_value_path_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_fields
    ADD CONSTRAINT data_kafka_fields_value_path_key UNIQUE (project_id, topic_mapping_id, value_path);


--
-- Name: data_kafka_topic_groups data_kafka_topic_groups_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_topic_groups
    ADD CONSTRAINT data_kafka_topic_groups_name_key UNIQUE (project_id, connection_id, parent_id, name);


--
-- Name: data_kafka_topic_groups data_kafka_topic_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_topic_groups
    ADD CONSTRAINT data_kafka_topic_groups_pkey PRIMARY KEY (id);


--
-- Name: data_kafka_topic_mappings data_kafka_topic_mappings_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_topic_mappings
    ADD CONSTRAINT data_kafka_topic_mappings_name_key UNIQUE (project_id, connection_id, name);


--
-- Name: data_kafka_topic_mappings data_kafka_topic_mappings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_topic_mappings
    ADD CONSTRAINT data_kafka_topic_mappings_pkey PRIMARY KEY (id);


--
-- Name: data_mqtt_configs data_mqtt_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_mqtt_configs
    ADD CONSTRAINT data_mqtt_configs_pkey PRIMARY KEY (connection_id);


--
-- Name: data_mqtt_messages data_mqtt_messages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_mqtt_messages
    ADD CONSTRAINT data_mqtt_messages_pkey PRIMARY KEY (id);


--
-- Name: data_mqtt_subscription_groups data_mqtt_subscription_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_mqtt_subscription_groups
    ADD CONSTRAINT data_mqtt_subscription_groups_pkey PRIMARY KEY (id);


--
-- Name: data_mqtt_subscriptions data_mqtt_subscriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_mqtt_subscriptions
    ADD CONSTRAINT data_mqtt_subscriptions_pkey PRIMARY KEY (id);


--
-- Name: data_mqtt_subscriptions data_mqtt_subscriptions_project_connection_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_mqtt_subscriptions
    ADD CONSTRAINT data_mqtt_subscriptions_project_connection_name_key UNIQUE (project_id, connection_id, name);


--
-- Name: data_mqtt_tags data_mqtt_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_mqtt_tags
    ADD CONSTRAINT data_mqtt_tags_pkey PRIMARY KEY (id);


--
-- Name: data_mqtt_tags data_mqtt_tags_project_subscription_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_mqtt_tags
    ADD CONSTRAINT data_mqtt_tags_project_subscription_code_key UNIQUE (project_id, subscription_id, code);


--
-- Name: data_points data_points_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_points
    ADD CONSTRAINT data_points_pkey PRIMARY KEY (id);


--
-- Name: data_points data_points_project_path_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_points
    ADD CONSTRAINT data_points_project_path_key UNIQUE (project_id, path);


--
-- Name: data_preview_sessions data_preview_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_preview_sessions
    ADD CONSTRAINT data_preview_sessions_pkey PRIMARY KEY (id);


--
-- Name: data_queries data_queries_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_queries
    ADD CONSTRAINT data_queries_pkey PRIMARY KEY (id);


--
-- Name: data_queries data_queries_project_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_queries
    ADD CONSTRAINT data_queries_project_name_key UNIQUE (project_id, name);


--
-- Name: data_realtime_keys data_realtime_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_realtime_keys
    ADD CONSTRAINT data_realtime_keys_pkey PRIMARY KEY (id);


--
-- Name: data_realtime_keys data_realtime_keys_project_id_connection_id_key_path_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_realtime_keys
    ADD CONSTRAINT data_realtime_keys_project_id_connection_id_key_path_key UNIQUE (project_id, connection_id, key_path);


--
-- Name: data_redis_configs data_redis_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_redis_configs
    ADD CONSTRAINT data_redis_configs_pkey PRIMARY KEY (connection_id);


--
-- Name: data_relational_configs data_relational_configs_connection_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_relational_configs
    ADD CONSTRAINT data_relational_configs_connection_id_key UNIQUE (connection_id);


--
-- Name: data_relational_configs data_relational_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_relational_configs
    ADD CONSTRAINT data_relational_configs_pkey PRIMARY KEY (id);


--
-- Name: data_history_storage_configs data_history_storage_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_history_storage_configs
    ADD CONSTRAINT data_history_storage_configs_pkey PRIMARY KEY (id);


--
-- Name: data_history_storage_targets data_history_storage_targets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_history_storage_targets
    ADD CONSTRAINT data_history_storage_targets_pkey PRIMARY KEY (id);


--
-- Name: data_history_storage_targets data_history_storage_targets_config_connection_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_history_storage_targets
    ADD CONSTRAINT data_history_storage_targets_config_connection_key UNIQUE (config_id, connection_id);


--
-- Name: data_table_group_members data_table_group_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_table_group_members
    ADD CONSTRAINT data_table_group_members_pkey PRIMARY KEY (project_id, connection_id, table_name);


--
-- Name: data_tdengine_configs data_tdengine_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_tdengine_configs
    ADD CONSTRAINT data_tdengine_configs_pkey PRIMARY KEY (connection_id);


--
-- Name: data_websocket_configs data_websocket_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_websocket_configs
    ADD CONSTRAINT data_websocket_configs_pkey PRIMARY KEY (connection_id);


--
-- Name: data_websocket_session_groups data_websocket_session_groups_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_websocket_session_groups
    ADD CONSTRAINT data_websocket_session_groups_name_key UNIQUE (project_id, connection_id, parent_id, name);


--
-- Name: data_websocket_session_groups data_websocket_session_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_websocket_session_groups
    ADD CONSTRAINT data_websocket_session_groups_pkey PRIMARY KEY (id);


--
-- Name: data_websocket_sessions data_websocket_sessions_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_websocket_sessions
    ADD CONSTRAINT data_websocket_sessions_name_key UNIQUE (project_id, connection_id, group_id, name);


--
-- Name: data_websocket_sessions data_websocket_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_websocket_sessions
    ADD CONSTRAINT data_websocket_sessions_pkey PRIMARY KEY (id);


--
-- Name: data_workbench_object_groups data_workbench_object_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_workbench_object_groups
    ADD CONSTRAINT data_workbench_object_groups_pkey PRIMARY KEY (id);


--
-- Name: collector_dev_agents_tenant_machine_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX collector_dev_agents_tenant_machine_idx ON collector_dev_agents USING btree (tenant_id, machine_id);


--
-- Name: collector_dev_agents_tenant_updated_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX collector_dev_agents_tenant_updated_idx ON collector_dev_agents USING btree (tenant_id, updated_at DESC);


--
-- Name: collector_dev_registration_codes_tenant_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX collector_dev_registration_codes_tenant_created_idx ON collector_dev_registration_codes USING btree (tenant_id, created_at DESC);


--
-- Name: collector_dev_tasks_agent_claim_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX collector_dev_tasks_agent_claim_idx ON collector_dev_tasks USING btree (agent_id, status, created_at) WHERE (status = ANY (ARRAY['queued'::text, 'running'::text]));


--
-- Name: collector_dev_tasks_connection_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX collector_dev_tasks_connection_created_idx ON collector_dev_tasks USING btree (project_id, connection_id, created_at DESC);


--
-- Name: collector_dev_tasks_finished_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX collector_dev_tasks_finished_idx ON collector_dev_tasks USING btree (finished_at) WHERE (finished_at IS NOT NULL);


--
-- Name: collector_dev_tasks_project_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX collector_dev_tasks_project_created_idx ON collector_dev_tasks USING btree (tenant_id, project_id, created_at DESC);


--
-- Name: data_access_source_records_connection_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_access_source_records_connection_created_idx ON data_access_source_records USING btree (connection_id, created_at DESC, id DESC);


--
-- Name: data_access_source_records_project_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_access_source_records_project_created_idx ON data_access_source_records USING btree (project_id, created_at DESC, id DESC);


--
-- Name: data_alarm_policies_project_enabled_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_alarm_policies_project_enabled_idx ON data_alarm_policies USING btree (project_id, is_enabled);


--
-- Name: data_alarm_policies_project_group_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_alarm_policies_project_group_idx ON data_alarm_policies USING btree (project_id, group_id);


--
-- Name: data_alarm_policies_project_updated_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_alarm_policies_project_updated_idx ON data_alarm_policies USING btree (project_id, updated_at DESC);


--
-- Name: data_alarm_policy_groups_project_parent_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_alarm_policy_groups_project_parent_idx ON data_alarm_policy_groups USING btree (project_id, parent_id, sort_order, created_at);


--
-- Name: data_alarm_policy_groups_project_parent_name_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_alarm_policy_groups_project_parent_name_key ON data_alarm_policy_groups USING btree (project_id, parent_id, name) WHERE (parent_id IS NOT NULL);


--
-- Name: data_alarm_policy_groups_project_root_name_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_alarm_policy_groups_project_root_name_key ON data_alarm_policy_groups USING btree (project_id, name) WHERE (parent_id IS NULL);


--
-- Name: data_alarm_policy_groups_project_sort_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_alarm_policy_groups_project_sort_idx ON data_alarm_policy_groups USING btree (project_id, sort_order, created_at);


--
-- Name: data_alarm_rules_condition_gin_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_alarm_rules_condition_gin_idx ON data_alarm_rules USING gin (condition);


--
-- Name: data_alarm_rules_project_enabled_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_alarm_rules_project_enabled_idx ON data_alarm_rules USING btree (project_id, is_enabled);


--
-- Name: data_alarm_rules_project_target_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_alarm_rules_project_target_path_idx ON data_alarm_rules USING btree (project_id, target_path);


--
-- Name: data_alarm_rules_project_updated_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_alarm_rules_project_updated_idx ON data_alarm_rules USING btree (project_id, updated_at DESC);


--
-- Name: data_alarm_rules_target_datapoint_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_alarm_rules_target_datapoint_idx ON data_alarm_rules USING btree (target_datapoint_id);


--
-- Name: data_collector_connection_secrets_connection_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_collector_connection_secrets_connection_idx ON data_collector_connection_secrets USING btree (connection_id);


--
-- Name: data_collector_connections_config_gin_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_collector_connections_config_gin_idx ON data_collector_connections USING gin (config);


--
-- Name: data_collector_connections_project_driver_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_collector_connections_project_driver_idx ON data_collector_connections USING btree (project_id, protocol_family, driver_id, created_at DESC);


--
-- Name: data_collector_import_sessions_connection_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_collector_import_sessions_connection_created_idx ON data_collector_import_sessions USING btree (project_id, connection_id, created_at DESC);


--
-- Name: data_collector_import_sessions_expiry_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_collector_import_sessions_expiry_idx ON data_collector_import_sessions USING btree (status, expires_at) WHERE (status = 'preview'::text);


--
-- Name: data_collector_point_groups_parent_name_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_collector_point_groups_parent_name_key ON data_collector_point_groups USING btree (project_id, connection_id, parent_id, name) WHERE (parent_id IS NOT NULL);


--
-- Name: data_collector_point_groups_parent_order_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_collector_point_groups_parent_order_idx ON data_collector_point_groups USING btree (project_id, connection_id, parent_id, sort_order, created_at, id);


--
-- Name: data_collector_point_groups_root_name_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_collector_point_groups_root_name_key ON data_collector_point_groups USING btree (project_id, connection_id, name) WHERE (parent_id IS NULL);


--
-- Name: data_collector_points_address_gin_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_collector_points_address_gin_idx ON data_collector_points USING gin (address);


--
-- Name: data_collector_points_address_text_trgm_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_collector_points_address_text_trgm_idx ON data_collector_points USING gin (address_text gin_trgm_ops);


--
-- Name: data_collector_points_connection_name_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_collector_points_connection_name_key ON data_collector_points USING btree (project_id, connection_id, lower(name));


--
-- Name: data_collector_points_data_type_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_collector_points_data_type_idx ON data_collector_points USING btree (project_id, connection_id, data_type);


--
-- Name: data_collector_points_group_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_collector_points_group_idx ON data_collector_points USING btree (project_id, connection_id, group_id, sort_order, created_at, id);


--
-- Name: data_collector_points_list_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_collector_points_list_idx ON data_collector_points USING btree (project_id, connection_id, enabled, sort_order, created_at, id);


--
-- Name: data_compute_folders_project_parent_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_compute_folders_project_parent_idx ON data_compute_folders USING btree (project_id, parent_id, created_at DESC);


--
-- Name: data_compute_folders_project_parent_name_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_compute_folders_project_parent_name_key ON data_compute_folders USING btree (project_id, parent_id, name) WHERE (parent_id IS NOT NULL);


--
-- Name: data_compute_folders_project_root_name_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_compute_folders_project_root_name_key ON data_compute_folders USING btree (project_id, name) WHERE (parent_id IS NULL);


--
-- Name: data_compute_runs_project_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_compute_runs_project_created_idx ON data_compute_runs USING btree (project_id, created_at DESC, id DESC);


--
-- Name: data_compute_runs_unit_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_compute_runs_unit_created_idx ON data_compute_runs USING btree (compute_unit_id, created_at DESC, id DESC);


--
-- Name: data_compute_units_project_enabled_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_compute_units_project_enabled_idx ON data_compute_units USING btree (project_id, is_enabled);


--
-- Name: data_compute_units_project_folder_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_compute_units_project_folder_idx ON data_compute_units USING btree (project_id, folder_id, created_at DESC);


--
-- Name: data_compute_units_project_language_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_compute_units_project_language_idx ON data_compute_units USING btree (project_id, language);


--
-- Name: data_connections_project_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_connections_project_created_idx ON data_connections USING btree (project_id, created_at DESC);


--
-- Name: data_connections_project_name_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_connections_project_name_key ON data_connections USING btree (project_id, name);


--
-- Name: data_connections_project_order_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_connections_project_order_idx ON data_connections USING btree (project_id, display_order, created_at DESC);


--
-- Name: data_connections_project_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_connections_project_status_idx ON data_connections USING btree (project_id, status);


--
-- Name: data_connections_project_type_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_connections_project_type_idx ON data_connections USING btree (project_id, type);


--
-- Name: data_contract_check_runs_project_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_contract_check_runs_project_created_idx ON data_contract_check_runs USING btree (project_id, created_at DESC, id DESC);


--
-- Name: data_contract_check_runs_project_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_contract_check_runs_project_status_idx ON data_contract_check_runs USING btree (project_id, status, created_at DESC);


--
-- Name: data_http_configs_method_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_http_configs_method_idx ON data_http_configs USING btree (method);


--
-- Name: data_http_request_groups_root_name_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_http_request_groups_root_name_key ON data_http_request_groups USING btree (project_id, connection_id, name) WHERE (parent_id IS NULL);


--
-- Name: data_http_request_groups_tree_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_http_request_groups_tree_idx ON data_http_request_groups USING btree (project_id, connection_id, parent_id, sort_order, updated_at DESC);


--
-- Name: data_http_requests_connection_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_http_requests_connection_idx ON data_http_requests USING btree (project_id, connection_id, updated_at DESC);


--
-- Name: data_http_requests_group_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_http_requests_group_idx ON data_http_requests USING btree (project_id, connection_id, group_id, sort_order, updated_at DESC);


--
-- Name: data_http_requests_root_name_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_http_requests_root_name_key ON data_http_requests USING btree (project_id, connection_id, name) WHERE (group_id IS NULL);


--
-- Name: data_kafka_configs_topic_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_kafka_configs_topic_idx ON data_kafka_configs USING btree (topic);


--
-- Name: data_kafka_field_groups_root_name_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_kafka_field_groups_root_name_key ON data_kafka_field_groups USING btree (project_id, topic_mapping_id, name) WHERE (parent_id IS NULL);


--
-- Name: data_kafka_field_groups_tree_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_kafka_field_groups_tree_idx ON data_kafka_field_groups USING btree (project_id, topic_mapping_id, parent_id, sort_order, created_at);


--
-- Name: data_kafka_fields_connection_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_kafka_fields_connection_idx ON data_kafka_fields USING btree (project_id, connection_id);


--
-- Name: data_kafka_fields_group_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_kafka_fields_group_idx ON data_kafka_fields USING btree (project_id, topic_mapping_id, group_id, sort_order, created_at);


--
-- Name: data_kafka_topic_groups_root_name_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_kafka_topic_groups_root_name_key ON data_kafka_topic_groups USING btree (project_id, connection_id, name) WHERE (parent_id IS NULL);


--
-- Name: data_kafka_topic_groups_tree_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_kafka_topic_groups_tree_idx ON data_kafka_topic_groups USING btree (project_id, connection_id, parent_id, sort_order, created_at);


--
-- Name: data_kafka_topic_mappings_group_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_kafka_topic_mappings_group_idx ON data_kafka_topic_mappings USING btree (project_id, connection_id, group_id, sort_order, created_at);


--
-- Name: data_mqtt_configs_protocol_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_mqtt_configs_protocol_idx ON data_mqtt_configs USING btree (protocol);


--
-- Name: data_mqtt_messages_project_subscription_received_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_mqtt_messages_project_subscription_received_idx ON data_mqtt_messages USING btree (project_id, subscription_id, received_at DESC, id DESC);


--
-- Name: data_mqtt_messages_subscription_received_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_mqtt_messages_subscription_received_idx ON data_mqtt_messages USING btree (subscription_id, received_at DESC, id DESC);


--
-- Name: data_mqtt_subscription_groups_parent_name_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_mqtt_subscription_groups_parent_name_key ON data_mqtt_subscription_groups USING btree (project_id, connection_id, parent_id, name) WHERE (parent_id IS NOT NULL);


--
-- Name: data_mqtt_subscription_groups_project_connection_parent_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_mqtt_subscription_groups_project_connection_parent_idx ON data_mqtt_subscription_groups USING btree (project_id, connection_id, parent_id, created_at);


--
-- Name: data_mqtt_subscription_groups_root_name_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_mqtt_subscription_groups_root_name_key ON data_mqtt_subscription_groups USING btree (project_id, connection_id, name) WHERE (parent_id IS NULL);


--
-- Name: data_mqtt_subscriptions_connection_order_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_mqtt_subscriptions_connection_order_idx ON data_mqtt_subscriptions USING btree (connection_id, display_order, created_at DESC);


--
-- Name: data_mqtt_subscriptions_project_connection_group_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_mqtt_subscriptions_project_connection_group_idx ON data_mqtt_subscriptions USING btree (project_id, connection_id, group_id, display_order, created_at);


--
-- Name: data_mqtt_subscriptions_project_connection_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_mqtt_subscriptions_project_connection_idx ON data_mqtt_subscriptions USING btree (project_id, connection_id);


--
-- Name: data_mqtt_subscriptions_project_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_mqtt_subscriptions_project_created_idx ON data_mqtt_subscriptions USING btree (project_id, created_at DESC);


--
-- Name: data_mqtt_tags_project_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_mqtt_tags_project_idx ON data_mqtt_tags USING btree (project_id, display_order, created_at DESC);


--
-- Name: data_mqtt_tags_subscription_order_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_mqtt_tags_subscription_order_idx ON data_mqtt_tags USING btree (subscription_id, display_order, created_at DESC);


--
-- Name: data_mqtt_tags_validation_gin_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_mqtt_tags_validation_gin_idx ON data_mqtt_tags USING gin (validation);


--
-- Name: data_points_project_created_display_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_points_project_created_display_idx ON data_points USING btree (project_id, created_at DESC, display_order, id DESC);


--
-- Name: data_points_project_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_points_project_created_idx ON data_points USING btree (project_id, created_at DESC);


--
-- Name: data_points_project_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_points_project_path_idx ON data_points USING btree (project_id, path);


--
-- Name: data_points_project_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_points_project_status_idx ON data_points USING btree (project_id, status);


--
-- Name: data_points_source_config_gin_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_points_source_config_gin_idx ON data_points USING gin (source_config);


--
-- Name: data_points_source_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_points_source_idx ON data_points USING btree (source_type, source_id);


--
-- Name: data_points_tags_gin_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_points_tags_gin_idx ON data_points USING gin (tags);


--
-- Name: data_preview_sessions_last_active_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_preview_sessions_last_active_at_idx ON data_preview_sessions USING btree (last_active_at);


--
-- Name: data_preview_sessions_project_user_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_preview_sessions_project_user_status_idx ON data_preview_sessions USING btree (project_id, user_id, status);


--
-- Name: data_queries_config_gin_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_queries_config_gin_idx ON data_queries USING gin (config);


--
-- Name: data_queries_connection_project_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_queries_connection_project_idx ON data_queries USING btree (connection_id, project_id);


--
-- Name: data_queries_project_connection_group_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_queries_project_connection_group_idx ON data_queries USING btree (project_id, connection_id, group_id, created_at DESC);


--
-- Name: data_queries_project_connection_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_queries_project_connection_idx ON data_queries USING btree (project_id, connection_id);


--
-- Name: data_queries_project_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_queries_project_created_idx ON data_queries USING btree (project_id, created_at DESC);


--
-- Name: data_queries_type_enabled_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_queries_type_enabled_idx ON data_queries USING btree (query_type, is_enabled);


--
-- Name: data_realtime_keys_connection_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_realtime_keys_connection_idx ON data_realtime_keys USING btree (project_id, connection_id, provider, key_path);


--
-- Name: data_redis_configs_mode_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_redis_configs_mode_idx ON data_redis_configs USING btree (mode);


--
-- Name: data_relational_configs_db_type_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_relational_configs_db_type_idx ON data_relational_configs USING btree (db_type);


--
-- Name: data_relational_configs_ssl_config_gin_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_relational_configs_ssl_config_gin_idx ON data_relational_configs USING gin (ssl_config);


--
-- Name: data_history_storage_configs_access_source_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_history_storage_configs_access_source_key ON data_history_storage_configs USING btree (project_id, access_source_id) WHERE (access_source_id IS NOT NULL);


--
-- Name: data_history_storage_configs_collector_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_history_storage_configs_collector_key ON data_history_storage_configs USING btree (project_id, collector_connection_id) WHERE (collector_connection_id IS NOT NULL);


--
-- Name: data_history_storage_configs_datapoint_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_history_storage_configs_datapoint_key ON data_history_storage_configs USING btree (project_id, datapoint_id) WHERE (datapoint_id IS NOT NULL);


--
-- Name: data_history_storage_targets_config_order_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_history_storage_targets_config_order_idx ON data_history_storage_targets USING btree (project_id, config_id, is_primary DESC, sort_order, id);


--
-- Name: data_history_storage_targets_primary_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_history_storage_targets_primary_key ON data_history_storage_targets USING btree (config_id) WHERE is_primary;


--
-- Name: data_table_group_members_group_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_table_group_members_group_idx ON data_table_group_members USING btree (project_id, connection_id, group_id);


--
-- Name: data_tdengine_configs_database_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_tdengine_configs_database_idx ON data_tdengine_configs USING btree (database_name);


--
-- Name: data_websocket_configs_url_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_websocket_configs_url_idx ON data_websocket_configs USING btree (url) WHERE (url IS NOT NULL);


--
-- Name: data_websocket_session_groups_root_name_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_websocket_session_groups_root_name_key ON data_websocket_session_groups USING btree (project_id, connection_id, name) WHERE (parent_id IS NULL);


--
-- Name: data_websocket_session_groups_tree_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_websocket_session_groups_tree_idx ON data_websocket_session_groups USING btree (project_id, connection_id, parent_id, sort_order, updated_at DESC);


--
-- Name: data_websocket_sessions_connection_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_websocket_sessions_connection_idx ON data_websocket_sessions USING btree (project_id, connection_id, updated_at DESC);


--
-- Name: data_websocket_sessions_group_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_websocket_sessions_group_idx ON data_websocket_sessions USING btree (project_id, connection_id, group_id, sort_order, updated_at DESC);


--
-- Name: data_websocket_sessions_root_name_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_websocket_sessions_root_name_key ON data_websocket_sessions USING btree (project_id, connection_id, name) WHERE (group_id IS NULL);


--
-- Name: data_workbench_object_groups_connection_scope_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX data_workbench_object_groups_connection_scope_idx ON data_workbench_object_groups USING btree (project_id, connection_id, scope, sort_order, created_at);


--
-- Name: data_workbench_object_groups_name_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX data_workbench_object_groups_name_key ON data_workbench_object_groups USING btree (project_id, connection_id, scope, name);


--
-- Name: collector_dev_registration_codes collector_dev_registration_codes_used_by_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY collector_dev_registration_codes
    ADD CONSTRAINT collector_dev_registration_codes_used_by_agent_id_fkey FOREIGN KEY (used_by_agent_id) REFERENCES collector_dev_agents(id) ON DELETE SET NULL;


--
-- Name: collector_dev_tasks collector_dev_tasks_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY collector_dev_tasks
    ADD CONSTRAINT collector_dev_tasks_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES collector_dev_agents(id);


--
-- Name: collector_dev_tasks collector_dev_tasks_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY collector_dev_tasks
    ADD CONSTRAINT collector_dev_tasks_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_collector_connections(id) ON DELETE CASCADE;


--
-- Name: data_access_source_records data_access_source_records_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_access_source_records
    ADD CONSTRAINT data_access_source_records_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_alarm_policies data_alarm_policies_group_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_alarm_policies
    ADD CONSTRAINT data_alarm_policies_group_id_fkey FOREIGN KEY (group_id) REFERENCES data_alarm_policy_groups(id) ON DELETE SET NULL;


--
-- Name: data_alarm_policy_groups data_alarm_policy_groups_parent_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_alarm_policy_groups
    ADD CONSTRAINT data_alarm_policy_groups_parent_fkey FOREIGN KEY (parent_id) REFERENCES data_alarm_policy_groups(id) ON DELETE CASCADE;


--
-- Name: data_alarm_rules data_alarm_rules_target_datapoint_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_alarm_rules
    ADD CONSTRAINT data_alarm_rules_target_datapoint_fkey FOREIGN KEY (target_datapoint_id) REFERENCES data_points(id) ON DELETE SET NULL;


--
-- Name: data_collector_connection_secrets data_collector_connection_secrets_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_connection_secrets
    ADD CONSTRAINT data_collector_connection_secrets_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_collector_connections(id) ON DELETE CASCADE;


--
-- Name: data_collector_import_sessions data_collector_import_sessions_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_import_sessions
    ADD CONSTRAINT data_collector_import_sessions_connection_fkey FOREIGN KEY (connection_id, project_id) REFERENCES data_collector_connections(id, project_id) ON DELETE CASCADE;


--
-- Name: data_collector_point_groups data_collector_point_groups_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_point_groups
    ADD CONSTRAINT data_collector_point_groups_connection_fkey FOREIGN KEY (connection_id, project_id) REFERENCES data_collector_connections(id, project_id) ON DELETE CASCADE;


--
-- Name: data_collector_point_groups data_collector_point_groups_parent_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_point_groups
    ADD CONSTRAINT data_collector_point_groups_parent_fkey FOREIGN KEY (parent_id, project_id, connection_id) REFERENCES data_collector_point_groups(id, project_id, connection_id) ON DELETE CASCADE;


--
-- Name: data_collector_point_debug_snapshots data_collector_point_debug_snapshots_point_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_point_debug_snapshots
    ADD CONSTRAINT data_collector_point_debug_snapshots_point_fkey FOREIGN KEY (point_id) REFERENCES data_collector_points(id) ON DELETE CASCADE;


--
-- Name: data_collector_points data_collector_points_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_points
    ADD CONSTRAINT data_collector_points_connection_fkey FOREIGN KEY (connection_id, project_id) REFERENCES data_collector_connections(id, project_id) ON DELETE CASCADE;


--
-- Name: data_collector_points data_collector_points_group_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_collector_points
    ADD CONSTRAINT data_collector_points_group_fkey FOREIGN KEY (group_id, project_id, connection_id) REFERENCES data_collector_point_groups(id, project_id, connection_id) ON DELETE RESTRICT;


--
-- Name: data_compute_folders data_compute_folders_parent_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_compute_folders
    ADD CONSTRAINT data_compute_folders_parent_fkey FOREIGN KEY (parent_id) REFERENCES data_compute_folders(id) ON DELETE CASCADE;


--
-- Name: data_compute_runs data_compute_runs_compute_unit_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_compute_runs
    ADD CONSTRAINT data_compute_runs_compute_unit_fkey FOREIGN KEY (compute_unit_id) REFERENCES data_compute_units(id) ON DELETE CASCADE;


--
-- Name: data_compute_units data_compute_units_folder_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_compute_units
    ADD CONSTRAINT data_compute_units_folder_fkey FOREIGN KEY (folder_id) REFERENCES data_compute_folders(id) ON DELETE SET NULL;


--
-- Name: data_http_configs data_http_configs_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_http_configs
    ADD CONSTRAINT data_http_configs_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_http_request_groups data_http_request_groups_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_http_request_groups
    ADD CONSTRAINT data_http_request_groups_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_http_request_groups data_http_request_groups_parent_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_http_request_groups
    ADD CONSTRAINT data_http_request_groups_parent_fkey FOREIGN KEY (parent_id) REFERENCES data_http_request_groups(id) ON DELETE SET NULL;


--
-- Name: data_http_requests data_http_requests_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_http_requests
    ADD CONSTRAINT data_http_requests_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_http_requests data_http_requests_group_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_http_requests
    ADD CONSTRAINT data_http_requests_group_fkey FOREIGN KEY (group_id) REFERENCES data_http_request_groups(id) ON DELETE SET NULL;


--
-- Name: data_kafka_configs data_kafka_configs_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_configs
    ADD CONSTRAINT data_kafka_configs_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_kafka_field_groups data_kafka_field_groups_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_field_groups
    ADD CONSTRAINT data_kafka_field_groups_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_kafka_field_groups data_kafka_field_groups_mapping_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_field_groups
    ADD CONSTRAINT data_kafka_field_groups_mapping_fkey FOREIGN KEY (topic_mapping_id) REFERENCES data_kafka_topic_mappings(id) ON DELETE CASCADE;


--
-- Name: data_kafka_field_groups data_kafka_field_groups_parent_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_field_groups
    ADD CONSTRAINT data_kafka_field_groups_parent_fkey FOREIGN KEY (parent_id) REFERENCES data_kafka_field_groups(id) ON DELETE SET NULL;


--
-- Name: data_kafka_fields data_kafka_fields_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_fields
    ADD CONSTRAINT data_kafka_fields_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_kafka_fields data_kafka_fields_group_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_fields
    ADD CONSTRAINT data_kafka_fields_group_fkey FOREIGN KEY (group_id) REFERENCES data_kafka_field_groups(id) ON DELETE SET NULL;


--
-- Name: data_kafka_fields data_kafka_fields_mapping_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_fields
    ADD CONSTRAINT data_kafka_fields_mapping_fkey FOREIGN KEY (topic_mapping_id) REFERENCES data_kafka_topic_mappings(id) ON DELETE CASCADE;


--
-- Name: data_kafka_topic_groups data_kafka_topic_groups_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_topic_groups
    ADD CONSTRAINT data_kafka_topic_groups_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_kafka_topic_groups data_kafka_topic_groups_parent_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_topic_groups
    ADD CONSTRAINT data_kafka_topic_groups_parent_fkey FOREIGN KEY (parent_id) REFERENCES data_kafka_topic_groups(id) ON DELETE SET NULL;


--
-- Name: data_kafka_topic_mappings data_kafka_topic_mappings_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_topic_mappings
    ADD CONSTRAINT data_kafka_topic_mappings_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_kafka_topic_mappings data_kafka_topic_mappings_group_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_kafka_topic_mappings
    ADD CONSTRAINT data_kafka_topic_mappings_group_fkey FOREIGN KEY (group_id) REFERENCES data_kafka_topic_groups(id) ON DELETE SET NULL;


--
-- Name: data_mqtt_configs data_mqtt_configs_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_mqtt_configs
    ADD CONSTRAINT data_mqtt_configs_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_mqtt_messages data_mqtt_messages_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_mqtt_messages
    ADD CONSTRAINT data_mqtt_messages_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_mqtt_messages data_mqtt_messages_subscription_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_mqtt_messages
    ADD CONSTRAINT data_mqtt_messages_subscription_fkey FOREIGN KEY (subscription_id) REFERENCES data_mqtt_subscriptions(id) ON DELETE CASCADE;


--
-- Name: data_mqtt_subscription_groups data_mqtt_subscription_groups_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_mqtt_subscription_groups
    ADD CONSTRAINT data_mqtt_subscription_groups_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_mqtt_subscription_groups data_mqtt_subscription_groups_parent_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_mqtt_subscription_groups
    ADD CONSTRAINT data_mqtt_subscription_groups_parent_fkey FOREIGN KEY (parent_id) REFERENCES data_mqtt_subscription_groups(id) ON DELETE CASCADE;


--
-- Name: data_mqtt_subscriptions data_mqtt_subscriptions_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_mqtt_subscriptions
    ADD CONSTRAINT data_mqtt_subscriptions_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_mqtt_subscriptions data_mqtt_subscriptions_group_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_mqtt_subscriptions
    ADD CONSTRAINT data_mqtt_subscriptions_group_fkey FOREIGN KEY (group_id) REFERENCES data_mqtt_subscription_groups(id) ON DELETE SET NULL;


--
-- Name: data_mqtt_tags data_mqtt_tags_subscription_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_mqtt_tags
    ADD CONSTRAINT data_mqtt_tags_subscription_fkey FOREIGN KEY (subscription_id) REFERENCES data_mqtt_subscriptions(id) ON DELETE CASCADE;


--
-- Name: data_queries data_queries_connection_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_queries
    ADD CONSTRAINT data_queries_connection_id_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_queries data_queries_group_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_queries
    ADD CONSTRAINT data_queries_group_fkey FOREIGN KEY (group_id) REFERENCES data_workbench_object_groups(id) ON DELETE SET NULL;


--
-- Name: data_realtime_keys data_realtime_keys_connection_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_realtime_keys
    ADD CONSTRAINT data_realtime_keys_connection_id_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_redis_configs data_redis_configs_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_redis_configs
    ADD CONSTRAINT data_redis_configs_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_relational_configs data_relational_configs_connection_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_relational_configs
    ADD CONSTRAINT data_relational_configs_connection_id_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_history_storage_configs data_history_storage_configs_access_source_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_history_storage_configs
    ADD CONSTRAINT data_history_storage_configs_access_source_fkey FOREIGN KEY (access_source_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_history_storage_configs data_history_storage_configs_collector_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_history_storage_configs
    ADD CONSTRAINT data_history_storage_configs_collector_fkey FOREIGN KEY (collector_connection_id) REFERENCES data_collector_connections(id) ON DELETE CASCADE;


--
-- Name: data_history_storage_configs data_history_storage_configs_datapoint_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_history_storage_configs
    ADD CONSTRAINT data_history_storage_configs_datapoint_fkey FOREIGN KEY (datapoint_id) REFERENCES data_points(id) ON DELETE CASCADE;


--
-- Name: data_history_storage_targets data_history_storage_targets_config_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_history_storage_targets
    ADD CONSTRAINT data_history_storage_targets_config_fkey FOREIGN KEY (config_id) REFERENCES data_history_storage_configs(id) ON DELETE CASCADE;


--
-- Name: data_history_storage_targets data_history_storage_targets_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_history_storage_targets
    ADD CONSTRAINT data_history_storage_targets_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE RESTRICT;


--
-- Name: data_table_group_members data_table_group_members_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_table_group_members
    ADD CONSTRAINT data_table_group_members_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_table_group_members data_table_group_members_group_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_table_group_members
    ADD CONSTRAINT data_table_group_members_group_fkey FOREIGN KEY (group_id) REFERENCES data_workbench_object_groups(id) ON DELETE CASCADE;


--
-- Name: data_tdengine_configs data_tdengine_configs_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_tdengine_configs
    ADD CONSTRAINT data_tdengine_configs_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_websocket_configs data_websocket_configs_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_websocket_configs
    ADD CONSTRAINT data_websocket_configs_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_websocket_session_groups data_websocket_session_groups_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_websocket_session_groups
    ADD CONSTRAINT data_websocket_session_groups_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_websocket_session_groups data_websocket_session_groups_parent_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_websocket_session_groups
    ADD CONSTRAINT data_websocket_session_groups_parent_fkey FOREIGN KEY (parent_id) REFERENCES data_websocket_session_groups(id) ON DELETE SET NULL;


--
-- Name: data_websocket_sessions data_websocket_sessions_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_websocket_sessions
    ADD CONSTRAINT data_websocket_sessions_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- Name: data_websocket_sessions data_websocket_sessions_group_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_websocket_sessions
    ADD CONSTRAINT data_websocket_sessions_group_fkey FOREIGN KEY (group_id) REFERENCES data_websocket_session_groups(id) ON DELETE SET NULL;


--
-- Name: data_workbench_object_groups data_workbench_object_groups_connection_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY data_workbench_object_groups
    ADD CONSTRAINT data_workbench_object_groups_connection_fkey FOREIGN KEY (connection_id) REFERENCES data_connections(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--
