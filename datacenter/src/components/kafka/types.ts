import type {
  KafkaField,
  KafkaFieldGroup,
  KafkaPreview,
  KafkaPreviewSample,
  KafkaSchemaField,
  KafkaTopicGroup,
  KafkaTopicMapping,
} from '@/api/schemas/kafka-workbench.schema'

export type {
  KafkaField,
  KafkaFieldGroup,
  KafkaPreview,
  KafkaPreviewSample,
  KafkaSchemaField,
  KafkaTopicGroup,
  KafkaTopicMapping,
}

export type KafkaTopicGroupNode = KafkaTopicGroup & {
  children: KafkaTopicGroupNode[]
  mappings: KafkaTopicMapping[]
}

export type KafkaWorkbenchConnection = {
  id: string
  name?: string
  type?: string
  status?: string
  config?: Record<string, unknown>
}
