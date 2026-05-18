<template>
  <DcDialog
    :model-value="visible"
    :title="dialogTitle"
    width="600px"
    @close="handleClose"
  >
    <el-form ref="formRef" :model="formData" :rules="rules" label-width="100px">
      <el-form-item label="分组名称" prop="name">
        <el-input
          v-model="formData.name"
          placeholder="请输入分组名称"
          clearable
        />
      </el-form-item>

      <el-form-item label="分组描述" prop="description">
        <el-input
          v-model="formData.description"
          type="textarea"
          :rows="3"
          placeholder="请输入分组描述"
        />
      </el-form-item>

      <el-form-item label="分组颜色" prop="color">
        <div class="color-options">
          <button
            v-for="color in presetColors"
            :key="color"
            type="button"
            class="color-option"
            :class="{ active: formData.color === color }"
            :style="{ backgroundColor: color }"
            :aria-label="`选择颜色 ${color}`"
            @click="handleColorSelect(color)"
          />
        </div>
        <span class="ml-2 text-xs text-gray-500">从预设颜色中选择</span>
      </el-form-item>

      <el-form-item label="显示顺序" prop="order">
        <el-input-number
          v-model="formData.order"
          :min="0"
          :max="999"
          controls-position="right"
        />
        <span class="ml-2 text-xs text-gray-500">数字越小越靠前</span>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" @click="handleSubmit" :loading="submitting">
        {{ mode === "create" ? "创建" : "更新" }}
      </el-button>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import DcDialog from "@/components/shared/DcDialog.vue";
import { ElMessage } from "element-plus";
import { createMqttTagGroup, updateMqttTagGroup } from "@/api/data.api";
import { getApiErrorMessage } from "@/utils/request";

const props = defineProps({
  visible: {
    type: Boolean,
    default: false,
  },
  mode: {
    type: String,
    default: "create", // 'create' | 'edit'
  },
  group: {
    type: Object,
    default: null,
  },
  projectId: {
    type: String,
    required: true,
  },
  subscriptionId: {
    type: String,
    required: true,
  },
});

const emit = defineEmits(["close", "success"]);

// 状态
const formRef = ref(null);
const submitting = ref(false);
const presetColors = [
  "#3b82f6",
  "#10b981",
  "#f59e0b",
  "#ef4444",
  "#6366f1",
  "#8b5cf6",
  "#06b6d4",
  "#6b7280",
];
const formData = ref({
  name: "",
  code: "",
  description: "",
  color: "#409EFF",
  order: 0,
});

// 校验规则
const rules = {
  name: [
    { required: true, message: "请输入分组名称", trigger: "blur" },
    { min: 2, max: 50, message: "长度在 2 到 50 个字符", trigger: "blur" },
  ],
};

// 计算属性
const dialogTitle = computed(() => {
  return props.mode === "create" ? "新建变量组" : "编辑变量组";
});

/**
 * 选择分组颜色
 * @param {string} color - 预设颜色值
 */
const handleColorSelect = (color) => {
  formData.value.color = color;
};

/**
 * 自动生成分组标识符
 * @param {string} name - 分组名称
 * @returns {string} 分组标识符
 */
const generateCode = (name) => {
  const rawName = String(name || "");
  let code = rawName
    .toLowerCase()
    .replace(/[\u4e00-\u9fa5]/g, "")
    .replace(/[^a-z0-9_]/g, "_")
    .replace(/^_+|_+$/g, "")
    .replace(/_+/g, "_");

  if (!code) {
    code = "group_" + Date.now();
  }

  if (!/^[a-z]/.test(code)) {
    code = "g_" + code;
  }

  return code;
};

// 监听 group 变化
watch(
  () => props.group,
  (newGroup) => {
    if (newGroup && props.mode === "edit") {
      formData.value = {
        name: newGroup.name || "",
        code: newGroup.code || "",
        description: newGroup.description || "",
        color: newGroup.color || "#409EFF",
        icon: newGroup.icon || "el-icon-folder",
        order: newGroup.order || 0,
      };
    } else {
      formData.value = {
        name: "",
        code: "",
        description: "",
        color: "#409EFF",
        icon: "el-icon-folder",
        order: 0,
      };
    }
  },
  { immediate: true },
);

/**
 * 关闭对话框
 * @returns {void}
 */
const handleClose = () => {
  emit("close");
};

/**
 * 提交表单
 * @returns {Promise<void>}
 * @throws 表单校验失败或接口调用异常
 */
const handleSubmit = async () => {
  try {
    if (!formData.value.code) {
      formData.value.code = generateCode(formData.value.name);
    }
    await formRef.value.validate();

    submitting.value = true;

    if (props.mode === "create") {
      await createMqttTagGroup(
        props.projectId,
        props.subscriptionId,
        formData.value,
      );
    } else {
      await updateMqttTagGroup(props.projectId, props.group.id, formData.value);
    }

    emit("success");
  } catch (error) {
    if (error !== false) {
      // 非表单校验错误
      console.error("Failed to submit group:", error);
      ElMessage.error(
        getApiErrorMessage(
          error,
          props.mode === "create" ? "创建失败" : "更新失败",
        ),
      );
    }
  } finally {
    submitting.value = false;
  }
};
</script>

<style scoped>
.el-select {
  width: 100%;
}

.color-options {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.color-option {
  width: 22px;
  height: 22px;
  border-radius: 6px;
  border: 2px solid transparent;
  cursor: pointer;
  outline: none;
}

.color-option.active {
  border-color: #111827;
  box-shadow: 0 0 0 1px rgba(17, 24, 39, 0.3);
}
</style>
