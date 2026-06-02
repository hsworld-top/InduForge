ALTER TABLE data_mqtt_subscriptions
    ALTER COLUMN message_retention SET DEFAULT 5000;

UPDATE data_mqtt_subscriptions
SET message_retention = 5000
WHERE message_retention = 100;
