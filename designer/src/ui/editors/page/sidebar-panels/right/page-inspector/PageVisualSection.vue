<script setup lang="ts">
import type { PageInspectorFormState } from "./page-inspector-types";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import FriendlyColorPicker from "@/ui/shared/widgets/base/FriendlyColorPicker.vue";

const props = defineProps<{
  form: PageInspectorFormState;
}>();

defineEmits<{
  updateConfig: [];
}>();

const { t } = useI18n();

const backgroundKindOptions = computed(() => [
  { label: t("pageInspector.backgroundKinds.color"), value: "color" },
  { label: t("pageInspector.backgroundKinds.image"), value: "image" },
  { label: t("pageInspector.backgroundKinds.gradient"), value: "gradient" },
]);

const backgroundSizeOptions = computed(() => [
  { label: t("pageInspector.backgroundSizes.cover"), value: "cover" },
  { label: t("pageInspector.backgroundSizes.contain"), value: "contain" },
  { label: t("pageInspector.backgroundSizes.stretch"), value: "stretch" },
  { label: t("pageInspector.backgroundSizes.auto"), value: "auto" },
]);

const backgroundRepeatOptions = computed(() => [
  { label: t("pageInspector.backgroundRepeats.noRepeat"), value: "no-repeat" },
  { label: t("pageInspector.backgroundRepeats.repeat"), value: "repeat" },
  { label: t("pageInspector.backgroundRepeats.repeatX"), value: "repeat-x" },
  { label: t("pageInspector.backgroundRepeats.repeatY"), value: "repeat-y" },
]);

const transitionOptions = computed(() => [
  { label: t("pageInspector.transitionTypes.none"), value: "none" },
  { label: t("pageInspector.transitionTypes.fade"), value: "fade" },
  { label: t("pageInspector.transitionTypes.slide"), value: "slide" },
  { label: t("pageInspector.transitionTypes.zoom"), value: "zoom" },
]);

const showImageBackgroundOptions = computed(() => props.form.backgroundType === "image");
</script>

<template>
  <div class="page-section-fields">
    <div class="page-prop-item">
      <div class="page-prop-label">{{ t("pageInspector.labels.backgroundType") }}</div>
      <div class="page-prop-editor">
        <el-select v-model="form.backgroundType" size="small" @change="$emit('updateConfig')">
          <el-option
            v-for="item in backgroundKindOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </div>
    </div>

    <div class="page-prop-item page-prop-item--stacked">
      <div class="page-prop-label">{{ t("pageInspector.labels.backgroundValue") }}</div>
      <div class="page-prop-editor">
        <FriendlyColorPicker
          v-if="form.backgroundType === 'color'"
          v-model="form.backgroundValue"
          @change="$emit('updateConfig')"
        />
        <el-input
          v-else
          v-model="form.backgroundValue"
          size="small"
          :placeholder="t('pageInspector.placeholders.backgroundValue')"
          @blur="$emit('updateConfig')"
        />
      </div>
    </div>

    <template v-if="showImageBackgroundOptions">
      <div class="page-prop-item">
        <div class="page-prop-label">{{ t("pageInspector.labels.backgroundSize") }}</div>
        <div class="page-prop-editor">
          <el-select v-model="form.backgroundSize" size="small" @change="$emit('updateConfig')">
            <el-option
              v-for="item in backgroundSizeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </div>
      </div>

      <div class="page-prop-item">
        <div class="page-prop-label">{{ t("pageInspector.labels.backgroundPosition") }}</div>
        <div class="page-prop-editor">
          <el-input
            v-model="form.backgroundPosition"
            size="small"
            :placeholder="t('pageInspector.placeholders.backgroundPosition')"
            @blur="$emit('updateConfig')"
          />
        </div>
      </div>

      <div class="page-prop-item">
        <div class="page-prop-label">{{ t("pageInspector.labels.backgroundRepeat") }}</div>
        <div class="page-prop-editor">
          <el-select
            v-model="form.backgroundRepeat"
            size="small"
            @change="$emit('updateConfig')"
          >
            <el-option
              v-for="item in backgroundRepeatOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </div>
      </div>
    </template>

    <div class="page-prop-item">
      <div class="page-prop-label">{{ t("pageInspector.labels.transitionType") }}</div>
      <div class="page-prop-editor">
        <el-select v-model="form.transitionType" size="small" @change="$emit('updateConfig')">
          <el-option
            v-for="item in transitionOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page-section-fields {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
}

.page-prop-item {
  display: flex;
  align-items: center;
  gap: var(--designer-gap-sm);
  min-height: 32px;
  padding: 4px 6px;
  border-radius: var(--designer-radius-sm);
}

.page-prop-item--stacked {
  align-items: flex-start;
}

.page-prop-label {
  width: 92px;
  min-width: 92px;
  font-size: var(--designer-font-label);
  color: var(--designer-text-regular);
}

.page-prop-editor {
  flex: 1;
  min-width: 0;
}

.page-prop-editor :deep(.el-select),
.page-prop-editor :deep(.el-input) {
  width: 100%;
}
</style>
