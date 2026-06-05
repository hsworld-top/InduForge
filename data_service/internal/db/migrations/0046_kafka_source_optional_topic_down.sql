-- 回滚到旧结构前需要保证历史 Kafka 配置均已补齐 Topic 与消费组。
UPDATE data_kafka_configs
SET topic = COALESCE(topic, ''),
    consumer_group = COALESCE(consumer_group, '')
WHERE topic IS NULL
   OR consumer_group IS NULL;

ALTER TABLE data_kafka_configs
    ALTER COLUMN topic SET NOT NULL,
    ALTER COLUMN consumer_group SET NOT NULL;
