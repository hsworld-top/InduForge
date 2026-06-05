<template>
  <aside class="kafka-inspector">
    <section class="kafka-inspector__section">
      <h3>接入源</h3>
      <dl>
        <div>
          <dt>名称</dt>
          <dd>{{ connection.name || '未命名 Kafka 接入源' }}</dd>
        </div>
        <div>
          <dt>Brokers</dt>
          <dd>{{ brokersText }}</dd>
        </div>
        <div>
          <dt>状态</dt>
          <dd>{{ connection.status || 'unknown' }}</dd>
        </div>
      </dl>
    </section>

    <section class="kafka-inspector__section">
      <h3>消费规则</h3>
      <dl v-if="mapping">
        <div>
          <dt>名称</dt>
          <dd>{{ mapping.name || '-' }}</dd>
        </div>
        <div>
          <dt>Topic</dt>
          <dd>{{ mapping.topic }}</dd>
        </div>
        <div>
          <dt>消费组</dt>
          <dd>{{ mapping.consumerGroup || '默认生成' }}</dd>
        </div>
        <div>
          <dt>输出</dt>
          <dd>{{ outputModeText }}</dd>
        </div>
        <div v-if="mapping.outputMode === 'raw_message'">
          <dt>数据点</dt>
          <dd>{{ mapping.rawDataPointPath || '-' }}</dd>
        </div>
        <div>
          <dt>分区</dt>
          <dd>{{ partitionText }}</dd>
        </div>
        <div>
          <dt>解码</dt>
          <dd>{{ mapping.decode }}</dd>
        </div>
      </dl>
      <p v-else>选择左侧消费规则后查看配置。</p>
    </section>

    <section class="kafka-inspector__section">
      <h3>最近预览</h3>
      <dl v-if="preview">
        <div>
          <dt>状态</dt>
          <dd>{{ preview.status }}</dd>
        </div>
        <div>
          <dt>样本</dt>
          <dd>{{ preview.samples.length }}</dd>
        </div>
        <div>
          <dt>耗时</dt>
          <dd>{{ preview.durationMs || 0 }} ms</dd>
        </div>
      </dl>
      <p v-else>拉取一次样本后展示诊断摘要。</p>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { KafkaPreview, KafkaTopicMapping, KafkaWorkbenchConnection } from './types'

const props = defineProps<{
  connection: KafkaWorkbenchConnection
  mapping?: KafkaTopicMapping | null
  preview?: KafkaPreview | null
}>()

const config = computed(() => props.connection.config || {})
const brokersText = computed(() => String(config.value.brokers || '未配置'))
const partitionText = computed(() => {
  if (!props.mapping) return '-'
  if (props.mapping.partitionMode === 'single') return `partition ${props.mapping.partition ?? 0}`
  return '全部分区'
})
const outputModeText = computed(() => {
  if (!props.mapping) return '-'
  return props.mapping.outputMode === 'raw_message' ? '整包数据点' : '字段数据点'
})
</script>

<style scoped>
.kafka-inspector {
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
  border-left: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
  padding: 12px;
}

.kafka-inspector__section {
  display: grid;
  gap: 8px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--dc-border);
}

.kafka-inspector__section + .kafka-inspector__section {
  margin-top: 14px;
}

.kafka-inspector h3 {
  margin: 0;
  color: var(--dc-text);
  font-size: 12px;
}

.kafka-inspector dl {
  margin: 0;
  display: grid;
  gap: 6px;
}

.kafka-inspector dl div {
  min-width: 0;
  display: grid;
  grid-template-columns: 58px minmax(0, 1fr);
  gap: 8px;
  align-items: start;
}

.kafka-inspector dt,
.kafka-inspector dd {
  min-width: 0;
  margin: 0;
  font-size: 12px;
}

.kafka-inspector dt {
  color: var(--dc-text-muted);
}

.kafka-inspector dd {
  overflow-wrap: anywhere;
  color: var(--dc-text-secondary);
  font-weight: 700;
}

.kafka-inspector p {
  margin: 0;
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 1.6;
}
</style>
