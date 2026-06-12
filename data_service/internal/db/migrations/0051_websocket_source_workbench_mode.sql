ALTER TABLE data_websocket_configs
    ALTER COLUMN url DROP NOT NULL;

DROP INDEX IF EXISTS data_websocket_configs_url_idx;

CREATE INDEX IF NOT EXISTS data_websocket_configs_url_idx
    ON data_websocket_configs (url)
    WHERE url IS NOT NULL;
