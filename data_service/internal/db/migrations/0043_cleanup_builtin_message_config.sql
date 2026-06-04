UPDATE data_connections
SET metadata = metadata - 'topic' - 'defaultTopic' - 'samplePayload' - 'topicPrefix',
    updated_at = now()
WHERE type = 'builtin.message';
