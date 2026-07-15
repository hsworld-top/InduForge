<template>
  <img
    v-if="iconSource"
    class="collector-driver-icon"
    :src="iconSource"
    alt=""
    aria-hidden="true"
  />
  <IconTablerCpu v-else class="collector-driver-icon collector-driver-icon--fallback" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import beckhoffIcon from '@/assets/collector-drivers/beckhoff.png'
import fatekIcon from '@/assets/collector-drivers/fatek.png'
import inovanceIcon from '@/assets/collector-drivers/inovance.png'
import keyenceIcon from '@/assets/collector-drivers/keyence.png'
import melsecIcon from '@/assets/collector-drivers/melsec.png'
import modbusIcon from '@/assets/collector-drivers/modbus.png'
import omronIcon from '@/assets/collector-drivers/omron.png'
import panasonicIcon from '@/assets/collector-drivers/panasonic.png'
import schneiderIcon from '@/assets/collector-drivers/schneider.png'
import siemensIcon from '@/assets/collector-drivers/siemens.png'
import {
  resolveCollectorDriverIconKey,
  type CollectorDriverIconKey,
} from './collector-workbench-model'
import IconTablerCpu from '~icons/tabler/cpu'

const props = defineProps<{
  protocolFamily: string
  driverId?: string
}>()
const iconSources: Record<CollectorDriverIconKey, string> = {
  beckhoff: beckhoffIcon,
  fatek: fatekIcon,
  inovance: inovanceIcon,
  keyence: keyenceIcon,
  melsec: melsecIcon,
  modbus: modbusIcon,
  omron: omronIcon,
  panasonic: panasonicIcon,
  schneider: schneiderIcon,
  siemens: siemensIcon,
}
const iconSource = computed(() => {
  const key = resolveCollectorDriverIconKey(props.protocolFamily, props.driverId)
  return key ? iconSources[key] : ''
})
</script>

<style scoped>
.collector-driver-icon {
  width: 18px;
  height: 18px;
  flex: 0 0 18px;
  object-fit: contain;
}
.collector-driver-icon--fallback {
  color: currentcolor;
}
</style>
