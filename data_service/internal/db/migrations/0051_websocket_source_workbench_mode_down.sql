UPDATE data_websocket_configs
SET url = ''
WHERE url IS NULL;

DROP INDEX IF EXISTS data_websocket_configs_url_idx;

ALTER TABLE data_websocket_configs
    ALTER COLUMN url SET NOT NULL;

CREATE INDEX IF NOT EXISTS data_websocket_configs_url_idx
    ON data_websocket_configs (url);
