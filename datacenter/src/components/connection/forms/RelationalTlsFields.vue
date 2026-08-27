<template>
  <section class="relational-tls-fields">
    <el-form-item :label="t('connection.sslConnection')">
      <el-select
        :model-value="tlsConfig.mode"
        :placeholder="t('connection.sslModePlaceholder')"
        class="w-full"
        @update:model-value="updateMode"
      >
        <el-option
          v-for="option in modeOptions"
          :key="option.value"
          :label="option.label"
          :value="option.value"
        >
          <span>{{ option.label }}</span>
          <span class="text-xs text-gray-400 ml-2">- {{ option.hint }}</span>
        </el-option>
      </el-select>
    </el-form-item>

    <template v-if="showsCertificates">
      <el-divider content-position="left">{{ t('connection.sslCertificateConfig') }}</el-divider>

      <el-form-item :label="t('connection.caCertificate')">
        <el-input
          :model-value="tlsConfig.ca"
          type="textarea"
          :rows="4"
          placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
          @update:model-value="updateField('ca', $event)"
        />
        <span class="text-xs text-gray-500 ml-2">
          {{
            savedSecrets.ca
              ? t('connection.savedCertificateHint')
              : t('connection.caCertificateHint')
          }}
        </span>
      </el-form-item>

      <template v-if="databaseType !== 'sqlserver'">
        <el-form-item :label="t('connection.clientCertificate')">
          <el-input
            :model-value="tlsConfig.cert"
            type="textarea"
            :rows="4"
            placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
            @update:model-value="updateField('cert', $event)"
          />
          <span class="text-xs text-gray-500 ml-2">
            {{
              savedSecrets.cert
                ? t('connection.savedCertificateHint')
                : t('connection.clientCertificateHint')
            }}
          </span>
        </el-form-item>

        <el-form-item :label="t('connection.clientKey')">
          <el-input
            :model-value="tlsConfig.key"
            type="textarea"
            :rows="4"
            placeholder="-----BEGIN PRIVATE KEY-----&#10;...&#10;-----END PRIVATE KEY-----"
            @update:model-value="updateField('key', $event)"
          />
          <span class="text-xs text-gray-500 ml-2">
            {{
              savedSecrets.key
                ? t('connection.savedCertificateHint')
                : t('connection.clientKeyHint')
            }}
          </span>
        </el-form-item>
      </template>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { t } from '@/i18n/runtime'

type TLSMode = 'disable' | 'prefer' | 'require' | 'verify-ca' | 'verify-full'
type TLSConfig = { mode: TLSMode; ca: string; cert: string; key: string }

const props = withDefaults(
  defineProps<{
    modelValue?: Partial<TLSConfig>
    databaseType: 'mysql' | 'postgresql' | 'sqlserver'
    savedSecrets?: Partial<Record<'ca' | 'cert' | 'key', boolean>>
  }>(),
  {
    modelValue: () => ({}),
    savedSecrets: () => ({}),
  },
)

const emit = defineEmits<{ 'update:modelValue': [value: TLSConfig] }>()

const tlsConfig = computed<TLSConfig>(() => ({
  mode: props.modelValue.mode || 'disable',
  ca: props.modelValue.ca || '',
  cert: props.modelValue.cert || '',
  key: props.modelValue.key || '',
}))

const allModeOptions = computed(() => [
  {
    value: 'disable' as const,
    label: t('connection.sslDisabled'),
    hint: t('connection.sslDisabledHint'),
  },
  {
    value: 'prefer' as const,
    label: t('connection.sslPrefer'),
    hint: t('connection.sslPreferHint'),
  },
  {
    value: 'require' as const,
    label: t('connection.sslRequire'),
    hint: t('connection.sslRequireHint'),
  },
  {
    value: 'verify-ca' as const,
    label: t('connection.sslVerifyCA'),
    hint: t('connection.sslVerifyCAHint'),
  },
  {
    value: 'verify-full' as const,
    label: t('connection.sslVerifyFull'),
    hint: t('connection.sslVerifyFullHint'),
  },
])

const modeOptions = computed(() => {
  if (props.databaseType === 'sqlserver') {
    return allModeOptions.value.filter((option) =>
      ['disable', 'require', 'verify-full'].includes(option.value),
    )
  }
  return allModeOptions.value
})

const showsCertificates = computed(() =>
  ['verify-ca', 'verify-full'].includes(tlsConfig.value.mode),
)

const updateMode = (mode: TLSMode) => {
  emit('update:modelValue', { ...tlsConfig.value, mode })
}

const updateField = (field: 'ca' | 'cert' | 'key', value: string) => {
  emit('update:modelValue', { ...tlsConfig.value, [field]: value })
}
</script>
