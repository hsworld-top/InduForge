-- Kafka 接入源只保存 Broker 与客户端参数，Topic/消费组由工作台 Topic 映射承载。
ALTER TABLE data_kafka_configs
    ALTER COLUMN topic DROP NOT NULL,
    ALTER COLUMN consumer_group DROP NOT NULL;
