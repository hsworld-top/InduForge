<template>
  <div class="global-vars">
    <!-- 顶部操作 -->
    <div class="toolbar">
      <el-button class="toolbar-button" type="primary" size="small" @click="openCreate">新增变量</el-button>
      <el-button class="toolbar-button toolbar-button--ghost" size="small" @click="openQuickAdd">快速添加数据源变量</el-button>
    </div>

    <!-- 变量列表 -->
    <el-table :data="varList" size="small" border style="width:100%">
      <el-table-column prop="name" label="变量名" width="180" />
      <el-table-column prop="type" label="类型" width="120" />
      <el-table-column label="初值">
        <template #default="{ row }">
          <el-tag v-if="row.mappedTo" type="success" effect="plain">{{ row.value }}</el-tag>
          <span v-else>{{ row.value }}</span>
        </template>
      </el-table-column>
      <el-table-column label="映射">
        <template #default="{ row }">
          <span v-if="row.mappedTo">{{ row.mappedTo.dsName }}.{{ row.mappedTo.field }}</span>
        </template>
      </el-table-column>
      <el-table-column width="140">
        <template #default="{ row }">
          <el-button link size="small" @click="openEdit(row)">编辑</el-button>
          <el-button link size="small" type="danger" @click="removeVar(row.name)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 新建 / 编辑变量 -->
    <el-dialog v-model="editVisible" :title="editMode ? '编辑变量' : '新增变量'" width="520px">
      <el-form label-width="80px">
        <el-form-item label="变量名">
          <el-input v-model="editName" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="editType" @change="resetEditValue">
            <el-option v-for="t in types" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>
        <el-form-item label="初值">
          <el-input v-if="isTextType" v-model="editValue" type="textarea" :rows="6" />
          <el-input-number v-else-if="editType==='number'" v-model="editValue" style="width:100%" />
          <el-switch v-else-if="editType==='boolean'" v-model="editValue" />
          <el-date-picker v-else-if="editType==='date'" v-model="editValue" type="datetime" style="width:100%" />
        </el-form-item>
        <el-form-item label="映射">
          <el-switch v-model="mapped" />
        </el-form-item>
        <template v-if="mapped">
          <el-form-item label="数据源">
            <el-select v-model="mappedDs">
              <el-option v-for="ds in dataSources" :key="ds.id" :label="ds.name" :value="ds" />
            </el-select>
          </el-form-item>
          <el-form-item label="字段">
            <el-input v-model="mappedField" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="editVisible=false">取消</el-button>
        <el-button type="primary" @click="saveEdit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 快速添加变量 -->
    <el-dialog v-model="quickVisible" title="快速添加数据源变量" width="900px">
      <el-form :inline="true" class="quick-form" label-width="60px" size="small">
        <el-row :gutter="12" class="quick-form-row">
          <el-col :span="6">
            <el-form-item label="数据源">
              <el-select v-model="quickDs" placeholder="请选择数据源" @change="loadFields">
                <el-option v-for="ds in dataSources" :key="ds.id" :label="ds.name" :value="ds" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="搜索">
              <el-input v-model="searchKey" placeholder="字段名搜索" />
            </el-form-item>
          </el-col>
          <el-col :span="3">
            <el-form-item label="前缀">
              <el-input v-model="prefix" placeholder="前缀" />
            </el-form-item>
          </el-col>
          <el-col :span="3">
            <el-form-item label="后缀">
              <el-input v-model="suffix" placeholder="后缀" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="替换" class="quick-replace">
              <el-input v-model="replaceFrom" placeholder="替换" />
              <span class="quick-arrow">→</span>
              <el-input v-model="replaceTo" placeholder="为" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>

      <el-table
        :data="filteredFields"
        border
        height="420"
        @selection-change="onSelectFields"
      >
        <el-table-column type="selection" width="50" />
        <el-table-column prop="name" label="字段名" sortable width="150" />
        <el-table-column prop="type" label="类型" width="120" sortable />
        <el-table-column label="变量名">
          <template #default="{ row }">
            {{ buildVarName(row.name) }}
          </template>
        </el-table-column>
        <el-table-column label="映射">
          <template #default="{ row }">
            {{ buildMappedExpression(row.name) }}
          </template>
        </el-table-column>
      </el-table>

      <template #footer>
        <el-button @click="quickVisible=false">取消</el-button>
        <el-button type="primary" @click="confirmQuickAdd">添加</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { useDesignStore } from '@/store/design'
import { designAPI } from '@/api/design.api'

const store = useDesignStore()

const types = ['string','number','boolean','array','object','set','map','date','regexp','function']

if(!store.projectVariables) store.projectVariables = {}

const varList = computed(() => Object.entries(store.projectVariables).map(([name,v])=>({name,...v})))

const editVisible = ref(false)
const editMode = ref(false)
const originalName = ref('')
const editName = ref('')
const editType = ref('string')
const editValue = ref('')
const mapped = ref(false)
const mappedDs = ref(null)
const mappedField = ref('')
const dataSources = ref([])

const isTextType = computed(()=>['string','array','object','regexp','function','set','map'].includes(editType.value))

function defaultEditValue(type){
  switch(type){
    case 'string': return ''
    case 'number': return 0
    case 'boolean': return false
    case 'array': return '[]'
    case 'object': return '{}'
    case 'set': return '[]'
    case 'map': return '[]'
    case 'date': return null
    case 'regexp': return '/pattern/g'
    case 'function': return 'function(){}'
    default: return ''
  }
}

function resetEditValue(){ editValue.value = defaultEditValue(editType.value) }

async function loadDataSourcesForMapping(){ dataSources.value = store.dataCenterConfig || [] }

async function persistProjectVariables() {
  try {
    await store.saveProjectVariables()
  } catch (error) {
    const message = error?.message ? String(error.message) : 'Unknown error'
    ElMessage.error(`Failed to save project variables: ${message}`)
  }
}

async function saveEdit(){
  const name = editName.value.trim()
  if(!name) return ElMessage.warning('变量名不能为空')
  if(editMode.value && name!==originalName.value) delete store.projectVariables[originalName.value]
  store.projectVariables[name] = {
    type: editType.value,
    value: editValue.value,
    mappedTo: mapped.value && mappedDs.value ? { dsId: mappedDs.value.id, dsName: mappedDs.value.name, field: mappedField.value } : null
  }
  store.isDirty = true
  editVisible.value=false
  await persistProjectVariables()
}

async function removeVar(name){
  delete store.projectVariables[name]
  store.isDirty=true
  await persistProjectVariables()
}

async function openCreate(){
  editMode.value=false; editName.value=''; editType.value='string'; resetEditValue()
  mapped.value=false; mappedDs.value=null; mappedField.value=''
  await loadDataSourcesForMapping()
  editVisible.value=true
}

async function openEdit(row){
  editMode.value=true; originalName.value=row.name
  editName.value=row.name; editType.value=row.type; editValue.value=row.value
  mapped.value=!!row.mappedTo
  await loadDataSourcesForMapping()
  if(row.mappedTo){
    mappedDs.value=dataSources.value.find(ds=>ds.id===row.mappedTo.dsId)||null
    mappedField.value=row.mappedTo.field
  } else { mappedDs.value=null; mappedField.value='' }
  editVisible.value=true
}

// 快速添加变量
const quickVisible = ref(false)
const quickDs = ref(null)
const fields = ref([])
const selectedFields = ref([])
const searchKey = ref('')
const prefix = ref('')
const suffix = ref('')
const replaceFrom = ref('')
const replaceTo = ref('')

async function openQuickAdd(){
  quickVisible.value=true
  await loadDataSourcesForMapping()
  quickDs.value=null
  fields.value=[]
  selectedFields.value=[]
}

async function loadFields(ds){
  quickDs.value = ds
  if(!ds) {
    fields.value = []
    return
  }
  // relational 类型通过 API 查询变量
  if(ds.type === 'relational'){
    const res = await designAPI.getQueries(store.projectId,{ connectionId: ds.id })
    const variables = res.data?.queries || []

    // 如果 dbType 是 mysql，类型全设为 object
    const isMysql = ds.relationalConfig?.dbType === 'mysql'

    fields.value = variables.map(v => ({
      name: v.name,
      type: isMysql ? 'object' : 'string'  // mysql -> object, 否则 string
    }))
  } else {
    // 其他类型可根据原来的 getFieldsFromDs
    fields.value = []
  }
}

const filteredFields = computed(()=>fields.value.filter(f=>f.name.toLowerCase().includes(searchKey.value.toLowerCase())))

function onSelectFields(rows){ selectedFields.value=rows }

function buildVarName(field){
  let name=field
  if(replaceFrom.value) name=name.replace(replaceFrom.value,replaceTo.value)
  return `${prefix.value}${name}${suffix.value}`
}

function buildMappedExpression(field){
  if(!quickDs.value) return ''
  return `{{${quickDs.value.name}.${field}}}`
}

async function confirmQuickAdd(){
  selectedFields.value.forEach(f=>{
    const name=buildVarName(f.name)
    if(store.projectVariables[name]) return
    store.projectVariables[name]={
      type: f.type||'string',
      value: defaultEditValue(f.type),
      mappedTo:{ dsId: quickDs.value.id, dsName: quickDs.value.name, field: f.name }
    }
  })
  store.isDirty=true
  quickVisible.value=false
  await persistProjectVariables()
}
</script>

<style scoped>
.global-vars{padding:12px;}
.toolbar{margin-bottom:12px; display:flex; align-items:center; gap:10px;}
.toolbar-button{height:28px; padding:0 12px; border-radius:4px; font-weight:600;}
.toolbar-button--ghost{color:var(--el-color-primary); border-color:var(--el-color-primary-light-5); background-color:var(--el-color-primary-light-9);}
.toolbar-button--ghost:hover{color:var(--el-color-primary); border-color:var(--el-color-primary); background-color:var(--el-color-primary-light-8);}
.quick-form{--quick-control-height:var(--el-component-size-small, 28px);}
.quick-form-row{margin-bottom:10px;}
.quick-form :deep(.el-form-item){width:100%; margin-bottom:8px; align-items:center;}
.quick-form :deep(.el-form-item__label){line-height:var(--quick-control-height);}
.quick-form :deep(.el-form-item__content){flex:1; min-width:0;}
.quick-form :deep(.el-input), .quick-form :deep(.el-select){width:100%;}
.quick-form :deep(.el-input__wrapper), .quick-form :deep(.el-select .el-input__wrapper){height:var(--quick-control-height); min-height:var(--quick-control-height);}
.quick-form :deep(.el-input__inner), .quick-form :deep(.el-select .el-input__inner){height:var(--quick-control-height); line-height:var(--quick-control-height);}
.quick-replace :deep(.el-form-item__content){display:flex; align-items:center; gap:6px;}
.quick-replace :deep(.el-input){flex:1;}
.quick-arrow{color:#909399; flex:0 0 auto;}
</style>
