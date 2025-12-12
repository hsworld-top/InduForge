<template>
  <span
    :class="['w-2 h-2 rounded-full', dotClass]"
    :title="label"
  ></span>
</template>

<script setup>
import { computed } from 'vue'
import { getStatusInfo } from '@/composables/useConnectionStatus'

const props = defineProps({
  status: {
    type: String,
    required: true,
    validator: (value) => ['connected', 'disconnected', 'error', 'unknown'].includes(value)
  }
})

const statusInfo = computed(() => getStatusInfo(props.status))
const label = computed(() => statusInfo.value.label)
const dotClass = computed(() => statusInfo.value.dotClass)
</script>
