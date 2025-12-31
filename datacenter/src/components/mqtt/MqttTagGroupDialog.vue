<template>
  <el-dialog
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

      <el-form-item label="分组标识符" prop="code">
        <el-input
          v-model="formData.code"
          placeholder="请输入分组标识符（英文、数字、下划线）"
          clearable
        >
          <template #append>
            <el-button @click="generateCode">自动生成</el-button>
          </template>
        </el-input>
        <span class="text-xs text-gray-500">
          用于程序引用，创建后不可修改
        </span>
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
        <el-color-picker v-model="formData.color" show-alpha />
        <span class="ml-2 text-xs text-gray-500">用于UI显示的颜色标识</span>
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
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch } from "vue";
import { ElMessage } from "element-plus";
import { createMqttTagGroup, updateMqttTagGroup } from "@/api/data.api";

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
  code: [
    { required: true, message: "请输入分组标识符", trigger: "blur" },
    {
      pattern: /^[a-zA-Z][a-zA-Z0-9_]*$/,
      message: "只能包含字母、数字、下划线，且以字母开头",
      trigger: "blur",
    },
    { min: 2, max: 50, message: "长度在 2 到 50 个字符", trigger: "blur" },
  ],
};

// 计算属性
const dialogTitle = computed(() => {
  return props.mode === "create" ? "新建变量组" : "编辑变量组";
});

// 自动生成 code
const generateCode = () => {
  if (!formData.value.name) {
    ElMessage.warning("请先输入分组名称");
    return;
  }

  // 简单转换：移除特殊字符，保留字母、数字、下划线
  let code = formData.value.name
    .toLowerCase()
    // 移除中文字符
    .replace(/[\u4e00-\u9fa5]/g, "")
    // 移除特殊字符，只保留字母、数字、下划线
    .replace(/[^a-z0-9_]/g, "_")
    // 移除开头和结尾的下划线
    .replace(/^_+|_+$/g, "")
    // 合并多个连续下划线
    .replace(/_+/g, "_");

  // 如果结果为空，使用默认值
  if (!code) {
    code = "group_" + Date.now();
  }

  // 确保以字母开头
  if (!/^[a-z]/.test(code)) {
    code = "g_" + code;
  }

  formData.value.code = code;
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

// 关闭对话框
const handleClose = () => {
  emit("close");
};

// 提交表单
const handleSubmit = async () => {
  try {
    await formRef.value.validate();

    submitting.value = true;

    if (props.mode === "create") {
      await createMqttTagGroup(
        props.projectId,
        props.subscriptionId,
        formData.value,
      );
    } else {
      await updateMqttTagGroup(props.group.id, formData.value);
    }

    emit("success");
  } catch (error) {
    if (error !== false) {
      // 非表单校验错误
      console.error("Failed to submit group:", error);
      ElMessage.error(
        error.response?.data?.message ||
          (props.mode === "create" ? "创建失败" : "更新失败"),
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
</style>
