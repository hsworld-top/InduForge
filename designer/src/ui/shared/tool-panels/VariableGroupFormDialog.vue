<!--
  新建/编辑变量分组（脚本侧与数据点侧共用表单壳）
-->
<script setup lang="ts">
interface VariableGroupOptionLike {
  id: string;
  name: string;
}

defineProps<{
  editMode?: boolean;
  parentOptions?: VariableGroupOptionLike[];
}>();

const emit = defineEmits<{
  (event: "confirm"): void;
}>();

const visible = defineModel<boolean>({ default: false });
const name = defineModel<string>("name", { default: "" });
const parentId = defineModel<string | null>("parentId", { default: null });

function handleCancel() {
  visible.value = false;
}

function handleConfirm() {
  emit("confirm");
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="editMode ? '编辑分组' : '新建分组'"
    width="420px"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <el-form label-width="70px">
      <el-form-item label="名称">
        <el-input v-model="name" />
      </el-form-item>
      <el-form-item label="父级">
        <el-select v-model="parentId" placeholder="根目录" :clearable="false">
          <el-option label="根目录" :value="null" />
          <el-option
            v-for="group in parentOptions"
            :key="group.id"
            :label="group.name"
            :value="group.id"
          />
        </el-select>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="handleCancel">取消</el-button>
      <el-button type="primary" @click="handleConfirm">确定</el-button>
    </template>
  </el-dialog>
</template>
