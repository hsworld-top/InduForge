<template>
  <section class="compute-editor">
    <div v-if="loading && !activeDraft" class="compute-editor__loading">
      <el-skeleton :rows="10" animated />
    </div>

    <EmptyState
      v-else-if="error && !activeDraft"
      icon-name="warning"
      :title="ui('计算单元详情不可用', 'Compute Unit Details Unavailable')"
      :description="error"
    />

    <EmptyState
      v-else-if="!activeDraft"
      icon-name="compute"
      :title="ui('选择一个计算单元', 'Select a Compute Unit')"
      :description="ui('从左侧资源树选择计算单元，右侧会打开编辑标签。', 'Select a compute unit from the resource tree to open it in the editor.')"
    />

    <template v-else>
      <header class="compute-editor__tabs">
        <el-tabs
          class="compute-editor__file-tabs"
          type="card"
          :model-value="activeId || ''"
          @tab-change="(name) => $emit('activate-tab', String(name))"
          @tab-remove="(name) => $emit('close-tab', String(name))"
        >
          <el-tab-pane v-for="tab in tabs" :key="tab.id" :name="tab.id" :closable="true">
            <template #label>
              <span class="compute-editor__tab-label">
                <span>{{ tab.name }}</span>
                <em v-if="tab.dirty">*</em>
              </span>
            </template>
          </el-tab-pane>
        </el-tabs>
      </header>

      <div class="compute-editor__toolbar">
        <div class="compute-editor__title-wrap">
          <div class="compute-editor__meta">
            <StatusBadge
              :tone="statusTone(activeDraft.status)"
              :text="statusText(activeDraft.status)"
            />
            <em v-if="activeDraft.dirty">{{ ui('未保存', 'Unsaved') }}</em>
          </div>
          <input
            v-model="activeDraft.name"
            class="compute-editor__name-input"
            :aria-label="ui('计算单元名称', 'Compute unit name')"
            @input="syncAllOutputPaths"
          />
          <div class="compute-editor__subline">
            <span class="compute-editor__lang-pill">{{ langText(activeDraft.lang) }}</span>
            <span class="compute-editor__output-inline">{{ previewOutputPath }}</span>
          </div>
        </div>

        <div class="compute-editor__actions">
          <button
            type="button"
            class="compute-editor__icon-action"
            :title="activeDraft.isEnabled === false ? ui('启用', 'Enable') : ui('停用', 'Disable')"
            :aria-label="activeDraft.isEnabled === false ? ui('启用', 'Enable') : ui('停用', 'Disable')"
            :disabled="saving"
            @click="$emit('toggle-enabled', activeDraft.id, activeDraft.isEnabled === false)"
          >
            <IconTablerPlayerPlay
              v-if="activeDraft.isEnabled === false"
              class="compute-editor__action-icon"
            />
            <IconTablerPlayerPause v-else class="compute-editor__action-icon" />
          </button>
          <button
            type="button"
            class="compute-editor__icon-action"
            :class="{ 'is-active': inspectorOpen }"
            :title="ui('输出配置', 'Output Settings')"
            :aria-label="ui('输出配置', 'Output Settings')"
            @click="inspectorOpen = !inspectorOpen"
          >
            <IconTablerSettings class="compute-editor__action-icon" />
          </button>
          <button
            type="button"
            class="compute-editor__icon-action is-danger"
            :title="ui('删除', 'Delete')"
            :aria-label="ui('删除', 'Delete')"
            :disabled="deleting"
            @click="$emit('delete-unit', activeDraft.id)"
          >
            <IconTablerTrash class="compute-editor__action-icon" />
          </button>
          <button
            type="button"
            class="compute-editor__save"
            :title="ui('保存', 'Save')"
            :aria-label="ui('保存', 'Save')"
            :disabled="saving || !activeDraft.dirty"
            @click="saveAfterSyntaxCheck"
          >
            <IconTablerDeviceFloppy class="compute-editor__action-icon" />
          </button>
        </div>
      </div>

      <div v-if="!sandboxAvailable" class="compute-editor__sandbox-warning">
        <span v-if="capabilitiesLoading">{{ ui('正在检测独立计算沙箱…', 'Checking the isolated compute sandbox…') }}</span>
        <span v-else>
          {{ ui('独立计算沙箱当前不可用：', 'The isolated compute sandbox is unavailable: ') }}{{ sandboxUnavailableReason }}{{ ui('。语法检查和开发态试运行已禁用。', '. Syntax checking and development dry runs are disabled.') }}
        </span>
        <button type="button" :disabled="capabilitiesLoading" @click="$emit('retry-capabilities')">
          {{ capabilitiesLoading ? ui('检测中', 'Checking') : ui('重新检测', 'Retry') }}
        </button>
      </div>

      <main class="compute-editor__main">
        <section class="compute-editor__code">
          <div class="compute-editor__code-head">
            <span class="compute-editor__code-title">
              <IconTablerCode class="compute-editor__head-icon" />
              {{ ui('代码', 'Code') }}
            </span>
            <div class="compute-editor__code-actions">
              <button
                type="button"
                class="compute-editor__tool-btn"
                :title="ui('数据点变量', 'Data Point Variables')"
                :aria-label="ui('数据点变量', 'Data Point Variables')"
                @click="openDatapointPicker('variable')"
              >
                <IconTablerDatabaseImport class="compute-editor__action-icon" />
              </button>
              <button
                type="button"
                class="compute-editor__tool-btn"
                :title="ui('代码模板', 'Code Templates')"
                :aria-label="ui('代码模板', 'Code Templates')"
                @click="templateDialogVisible = true"
              >
                <IconTablerTemplate class="compute-editor__action-icon" />
              </button>
              <span class="compute-editor__tooltip-wrap" :title="dryRunTooltip">
                <button
                  type="button"
                  class="compute-editor__tool-btn"
                  :aria-label="ui('试运行', 'Dry Run')"
                  :disabled="debugRunning || activeDraft.dirty || !sandboxAvailable"
                  @click="quickDryRun"
                >
                  <IconTablerPlayerPlay class="compute-editor__action-icon" />
                </button>
              </span>
              <button
                type="button"
                class="compute-editor__syntax-status"
                :class="`is-${syntaxStatusTone}`"
                :title="syntaxStatusText"
                @click="openSyntaxPanel"
              >
                <IconTablerCircleCheck
                  v-if="syntaxStatus === 'clean'"
                  class="compute-editor__action-icon"
                />
                <IconTablerAlertTriangle
                  v-else-if="syntaxDiagnostics.length"
                  class="compute-editor__action-icon"
                />
                <IconTablerLoader2
                  v-else-if="syntaxStatus === 'checking'"
                  class="compute-editor__action-icon"
                />
                <IconTablerCircleDashed v-else class="compute-editor__action-icon" />
                <span>{{ syntaxStatusText }}</span>
              </button>
              <em>{{ langText(activeDraft.lang) }}</em>
            </div>
          </div>
          <MonacoEditor
            ref="monacoEditorRef"
            v-model="activeDraft.code"
            class="compute-editor__monaco"
            :language="monacoLanguage"
            :type-definitions="computeTypeDefinitions"
            :member-completions="computeMemberCompletions"
            height="100%"
            @change="markDirty"
            @cursor-change="updateCursorInfo"
            @save="saveAfterSyntaxCheck"
          />
        </section>

        <section v-if="inspectorOpen" class="compute-editor__inspector">
          <div class="compute-editor__inspector-head">
            <div class="compute-editor__inspector-title">
              <IconTablerSettings class="compute-editor__head-icon" />
              <h3>{{ ui('输出', 'Outputs') }}</h3>
            </div>
            <button
              type="button"
              class="compute-editor__ghost-icon"
              :title="ui('关闭输出配置', 'Close Output Settings')"
              :aria-label="ui('关闭输出配置', 'Close Output Settings')"
              @click="inspectorOpen = false"
            >
              <IconTablerX class="compute-editor__action-icon" />
            </button>
            <div class="compute-editor__output-path">
              {{ ui(`${activeDraft.outputs.length} 个输出数据点`, `${activeDraft.outputs.length} output data point${activeDraft.outputs.length === 1 ? '' : 's'}`) }}
            </div>
          </div>
          <label class="compute-editor__field">
            <span>{{ ui('超时', 'Timeout') }}</span>
            <input
              v-model.number="activeDraft.timeoutMs"
              type="number"
              min="1"
              max="120000"
              @input="markDirty"
            />
          </label>
          <section class="compute-editor__outputs">
            <header>
              <strong>{{ ui('强类型输出', 'Typed Outputs') }}</strong>
              <button type="button" class="compute-editor__secondary-action" @click="addOutput">
                {{ ui('添加输出', 'Add Output') }}
              </button>
            </header>
            <article
              v-for="(output, index) in activeDraft.outputs"
              :key="String(output.id || index)"
              class="compute-editor__output-item"
            >
              <label class="compute-editor__field">
                <span>{{ ui('输出 Key', 'Output Key') }}</span>
                <input
                  v-model="output.key"
                  placeholder="result"
                  @input="handleOutputKeyInput(output)"
                />
              </label>
              <label class="compute-editor__field">
                <span>{{ ui('显示名称', 'Display Name') }}</span>
                <input v-model="output.name" :placeholder="ui('计算结果', 'Compute Result')" @input="markDirty" />
              </label>
              <label class="compute-editor__field">
                <span>{{ ui('数据类型', 'Data Type') }}</span>
                <select v-model="output.dataType" @change="markDirty">
                  <option v-for="option in outputTypeOptions" :key="option" :value="option">
                    {{ option }}
                  </option>
                </select>
              </label>
              <label class="compute-editor__field">
                <span>{{ ui('单位', 'Unit') }}</span>
                <input v-model="output.unit" :placeholder="ui('可选', 'Optional')" @input="markDirty" />
              </label>
              <label class="compute-editor__field">
                <span>{{ ui('精度', 'Precision') }}</span>
                <input
                  v-model.number="output.precisionNum"
                  type="number"
                  min="0"
                  :placeholder="ui('可选', 'Optional')"
                  @input="markDirty"
                />
              </label>
              <label class="compute-editor__field">
                <span>{{ ui('空值策略', 'Null Policy') }}</span>
                <select v-model="output.nullPolicy" @change="markDirty">
                  <option value="error">{{ ui('本次运行失败', 'Fail This Run') }}</option>
                  <option value="skip">{{ ui('跳过并保留旧值', 'Skip and Keep Previous Value') }}</option>
                </select>
              </label>
              <small>{{ output.path }}</small>
              <button
                type="button"
                class="compute-editor__remove-output"
                :disabled="activeDraft.outputs.length <= 1"
                @click="removeOutput(index)"
              >
                {{ ui('删除', 'Delete') }}
              </button>
            </article>
          </section>
        </section>
      </main>

      <section
        class="compute-editor__bottom"
        :class="{ 'is-collapsed': panelCollapsed }"
        :style="bottomPanelStyle"
      >
        <div
          v-show="!panelCollapsed"
          class="compute-editor__resize-handle"
          :title="ui('拖拽调整高度，双击切换最大高度', 'Drag to resize; double-click to toggle maximum height')"
          @dblclick="togglePanelMaxHeight"
          @pointerdown="startPanelResize"
        >
          <span></span>
        </div>

        <footer class="compute-editor__panel-tabs">
          <div v-if="panelCollapsed" class="compute-editor__panel-summary">
            <button
              v-for="item in panelSummaryItems"
              :key="item.id"
              type="button"
              :title="item.label"
              @click="openPanel(item.id)"
            >
              {{ item.label }} {{ item.value }}
            </button>
          </div>
          <div v-else class="compute-editor__panel-tablist">
            <button
              v-for="tab in panelTabs"
              :key="tab.id"
              type="button"
              class="compute-editor__panel-tab"
              :class="{ 'is-active': activePanel === tab.id }"
              :title="tab.label"
              @click="activePanel = tab.id"
            >
              <component :is="tab.icon" class="compute-editor__panel-tab-icon" />
              <span>{{ tab.label }}</span>
            </button>
          </div>
          <div class="compute-editor__cursor-status">
            <span>{{ ui(`行 ${cursorInfo.line}，列 ${cursorInfo.column}`, `Ln ${cursorInfo.line}, Col ${cursorInfo.column}`) }}</span>
            <span>{{ ui('空格', 'Spaces') }}: {{ cursorInfo.spaces }}</span>
          </div>
          <button
            type="button"
            class="compute-editor__panel-toggle"
            :title="panelCollapsed ? ui('展开', 'Expand') : ui('收起', 'Collapse')"
            :aria-label="panelCollapsed ? ui('展开底部面板', 'Expand bottom panel') : ui('收起底部面板', 'Collapse bottom panel')"
            @click="togglePanelCollapsed"
          >
            <IconTablerChevronUp
              class="compute-editor__panel-toggle-icon"
              :class="{ 'is-collapsed': panelCollapsed }"
            />
          </button>
        </footer>

        <div v-show="!panelCollapsed" class="compute-editor__panel" :class="`is-${activePanel}`">
          <template v-if="activePanel === 'inputs'">
            <div class="compute-editor__panel-head">
              <div>
                <h3>
                  <span>{{ ui('脚本参数', 'Script Parameters') }}</span>
                  <button
                    type="button"
                    class="compute-editor__help-dot"
                    :title="ui('调用方传入的参数，脚本内按 argv[0]、argv[1] 顺序读取。', 'Parameters supplied by the caller and read in order as argv[0], argv[1], and so on.')"
                    :aria-label="ui('脚本参数说明', 'Script parameter help')"
                  >
                    ?
                  </button>
                </h3>
              </div>
              <button
                type="button"
                class="compute-editor__tool-btn"
                :title="ui('添加参数', 'Add Parameter')"
                :aria-label="ui('添加参数', 'Add Parameter')"
                @click="addInput"
              >
                <IconTablerPlus class="compute-editor__action-icon" />
              </button>
            </div>
            <div
              v-if="!activeDraft.parameterRows.length"
              class="compute-editor__empty compute-editor__empty-action"
            >
              <strong>{{ ui('还没有脚本参数', 'No Script Parameters') }}</strong>
              <span>{{ ui('脚本参数由调用方传入，脚本内通过 argv[0]、argv[1] 读取。', 'Parameters are supplied by the caller and read as argv[0], argv[1], and so on.') }}</span>
              <div>
                <button type="button" class="compute-editor__compact-primary" @click="addInput">
                  <IconTablerPlus class="compute-editor__action-icon" />
                  <span>{{ ui('添加参数', 'Add Parameter') }}</span>
                </button>
              </div>
            </div>
            <div v-else class="compute-editor__table-wrap">
              <div class="compute-editor__mapping-head">
                <span>{{ ui('序号', 'Index') }}</span>
                <span>{{ ui('参数名', 'Name') }}</span>
                <span>{{ ui('类型', 'Type') }}</span>
                <span>{{ ui('必填', 'Required') }}</span>
                <span>{{ ui('默认值', 'Default') }}</span>
                <span>{{ ui('说明', 'Description') }}</span>
                <span></span>
              </div>
              <div
                v-for="(row, index) in activeDraft.parameterRows"
                :key="row.uid"
                class="compute-editor__mapping-row"
              >
                <span class="compute-editor__row-index">argv[{{ index }}]</span>
                <input v-model="row.name" :placeholder="ui('参数名', 'Parameter name')" @input="markDirty" />
                <select v-model="row.type" @change="markDirty">
                  <option value="string">string</option>
                  <option value="number">number</option>
                  <option value="boolean">boolean</option>
                  <option value="object">object</option>
                  <option value="array">array</option>
                </select>
                <label class="compute-editor__switch" :class="{ 'is-on': row.required }">
                  <input v-model="row.required" type="checkbox" @change="markDirty" />
                  <span></span>
                </label>
                <input v-model="row.defaultValue" placeholder="-" @input="markDirty" />
                <input v-model="row.description" :placeholder="ui('调用方传参说明', 'Describe this argument')" @input="markDirty" />
                <button
                  type="button"
                  class="compute-editor__row-icon"
                  :title="ui('删除参数', 'Delete Parameter')"
                  :aria-label="ui('删除参数', 'Delete Parameter')"
                  @click="removeInput(index)"
                >
                  <IconTablerTrash class="compute-editor__action-icon" />
                </button>
              </div>
            </div>
          </template>

          <template v-else-if="activePanel === 'variables'">
            <div class="compute-editor__panel-head">
              <div>
                <h3>
                  <span>{{ ui('数据点变量', 'Data Point Variables') }}</span>
                  <button
                    type="button"
                    class="compute-editor__help-dot"
                    :title="ui('变量名对应一个数据点对象，可直接调用 get、read、peek 等方法。', 'Each variable references a data point object and supports methods such as get, read, and peek.')"
                    :aria-label="ui('数据点变量说明', 'Data point variable help')"
                  >
                    ?
                  </button>
                </h3>
              </div>
              <button
                type="button"
                class="compute-editor__tool-btn"
                :title="ui('插入数据点变量', 'Insert Data Point Variable')"
                :aria-label="ui('插入数据点变量', 'Insert Data Point Variable')"
                @click="openDatapointPicker('variable')"
              >
                <IconTablerDatabaseImport class="compute-editor__action-icon" />
              </button>
            </div>
            <div
              v-if="!activeDraft.datapointVariableRows.length"
              class="compute-editor__empty compute-editor__empty-action"
            >
              <strong>{{ ui('还没有数据点变量', 'No Data Point Variables') }}</strong>
              <span>{{ ui('从数据点列表选择后，会在这里管理变量名、路径和类型。', 'Select a data point to manage its variable name, path, and type here.') }}</span>
              <div>
                <button
                  type="button"
                  class="compute-editor__compact-primary"
                  @click="openDatapointPicker('variable')"
                >
                  <IconTablerDatabaseImport class="compute-editor__action-icon" />
                  <span>{{ ui('插入数据点变量', 'Insert Data Point Variable') }}</span>
                </button>
              </div>
            </div>
            <div v-else class="compute-editor__variable-list">
              <div
                v-for="(row, index) in activeDraft.datapointVariableRows"
                :key="row.uid"
                class="compute-editor__variable-row"
              >
                <input
                  class="compute-editor__variable-alias"
                  :value="row.alias"
                  :aria-label="ui(`变量名：${row.alias}`, `Variable name: ${row.alias}`)"
                  :title="ui('修改脚本变量名', 'Rename script variable')"
                  spellcheck="false"
                  @focus="($event.target as HTMLInputElement).select()"
                  @keydown.enter="($event.target as HTMLInputElement).blur()"
                  @blur="renameDatapointVariable(row, $event)"
                />
                <span class="compute-editor__variable-path" :title="row.path">
                  {{ row.path }}
                </span>
                <em>{{ row.dataType || '-' }}</em>
                <span class="compute-editor__variable-state">{{ ui('已引用', 'Referenced') }}</span>
                <button
                  type="button"
                  class="compute-editor__row-icon"
                  :title="ui('删除变量', 'Delete Variable')"
                  :aria-label="ui('删除变量', 'Delete Variable')"
                  @click="removeDatapointVariable(index)"
                >
                  <IconTablerTrash class="compute-editor__action-icon" />
                </button>
              </div>
            </div>
          </template>

          <template v-else-if="activePanel === 'trigger'">
            <div class="compute-editor__trigger-layout">
              <aside class="compute-editor__trigger-rail">
                <button
                  v-for="item in triggerTypes"
                  :key="item.id"
                  type="button"
                  :class="{ 'is-active': isTriggerOptionActive(item.id) }"
                  @click="setTriggerType(item.id)"
                >
                  <component :is="triggerIcon(item.id)" class="compute-editor__panel-tab-icon" />
                  <span>{{ item.label }}</span>
                  <small>{{ triggerDescription(item.id) }}</small>
                </button>
              </aside>
              <section class="compute-editor__trigger-config">
                <div class="compute-editor__panel-head">
                  <div>
                    <h3>
                      <span>{{ triggerText }}</span>
                      <button
                        type="button"
                        class="compute-editor__help-dot"
                        :title="triggerSummary"
                        :aria-label="ui('触发说明', 'Trigger help')"
                      >
                        ?
                      </button>
                    </h3>
                  </div>
                </div>
                <div
                  v-if="activeDraft.triggerType === 'schedule'"
                  class="compute-editor__trigger-card is-schedule"
                >
                  <label class="compute-editor__field">
                    <span v-if="activeDraft.triggerConfig.kind === 'interval'">{{ ui('周期', 'Interval') }}</span>
                    <span v-else>{{ ui('执行时间', 'Run Time') }}</span>
                    <div
                      v-if="activeDraft.triggerConfig.kind === 'interval'"
                      class="compute-editor__field-inline"
                    >
                      <input
                        v-model.number="activeDraft.triggerConfig.every"
                        type="number"
                        min="1"
                        @input="markScheduleDirty"
                      />
                      <select v-model="activeDraft.triggerConfig.unit" @change="markScheduleDirty">
                        <option value="seconds">{{ ui('秒', 'Seconds') }}</option>
                        <option value="minutes">{{ ui('分钟', 'Minutes') }}</option>
                        <option value="hours">{{ ui('小时', 'Hours') }}</option>
                      </select>
                    </div>
                    <input
                      v-else
                      v-model="activeDraft.triggerConfig.time"
                      type="time"
                      step="1"
                      @input="markScheduleDirty"
                    />
                    <small
                      v-if="
                        scheduleFieldError(
                          activeDraft.triggerConfig.kind === 'interval' ? 'every' : 'time',
                        )
                      "
                      class="compute-editor__field-error"
                    >
                      {{
                        scheduleFieldError(
                          activeDraft.triggerConfig.kind === 'interval' ? 'every' : 'time',
                        )
                      }}
                    </small>
                  </label>
                  <label
                    v-if="activeDraft.triggerConfig.kind === 'weekly'"
                    class="compute-editor__field"
                  >
                    <span>{{ ui('执行星期', 'Days of Week') }}</span>
                    <div class="compute-editor__weekday-list">
                      <label v-for="day in weekdayOptions" :key="day.value">
                        <input
                          v-model="activeDraft.triggerConfig.weekdays"
                          type="checkbox"
                          :value="day.value"
                          @change="markScheduleDirty"
                        />
                        {{ day.label }}
                      </label>
                    </div>
                    <small
                      v-if="scheduleFieldError('weekdays')"
                      class="compute-editor__field-error"
                    >
                      {{ scheduleFieldError('weekdays') }}
                    </small>
                  </label>
                  <div
                    v-if="
                      activeDraft.triggerConfig.kind === 'monthly' ||
                      activeDraft.triggerConfig.kind === 'yearly'
                    "
                    class="compute-editor__schedule-rule"
                  >
                    <label
                      v-if="activeDraft.triggerConfig.kind === 'yearly'"
                      class="compute-editor__field"
                    >
                      <span>{{ ui('执行月份', 'Month') }}</span>
                      <select
                        v-model.number="activeDraft.triggerConfig.month"
                        @change="markScheduleDirty"
                      >
                        <option v-for="month in 12" :key="month" :value="month">
                          {{ ui(`${month} 月`, monthName(month)) }}
                        </option>
                      </select>
                      <small v-if="scheduleFieldError('month')" class="compute-editor__field-error">
                        {{ scheduleFieldError('month') }}
                      </small>
                    </label>
                    <label class="compute-editor__field">
                      <span>{{ ui('日期规则', 'Date Rule') }}</span>
                      <select
                        v-model="activeDraft.triggerConfig.dayRule"
                        @change="markScheduleDirty"
                      >
                        <option value="day">{{ ui('指定日期', 'Specific Date') }}</option>
                        <option value="weekday">{{ ui('指定星期', 'Specific Weekday') }}</option>
                      </select>
                      <small
                        v-if="scheduleFieldError('dayRule')"
                        class="compute-editor__field-error"
                      >
                        {{ scheduleFieldError('dayRule') }}
                      </small>
                    </label>
                    <label
                      v-if="activeDraft.triggerConfig.dayRule === 'day'"
                      class="compute-editor__field"
                    >
                      <span>{{ ui('日期', 'Day') }}</span>
                      <input
                        v-model.number="activeDraft.triggerConfig.dayOfMonth"
                        type="number"
                        min="1"
                        max="31"
                        @input="markScheduleDirty"
                      />
                      <small
                        v-if="scheduleFieldError('dayOfMonth')"
                        class="compute-editor__field-error"
                      >
                        {{ scheduleFieldError('dayOfMonth') }}
                      </small>
                    </label>
                    <template v-else>
                      <label class="compute-editor__field">
                        <span>{{ ui('第几周', 'Week of Month') }}</span>
                        <select
                          v-model.number="activeDraft.triggerConfig.weekOfMonth"
                          @change="markScheduleDirty"
                        >
                          <option :value="1">{{ ui('第一周', 'First') }}</option>
                          <option :value="2">{{ ui('第二周', 'Second') }}</option>
                          <option :value="3">{{ ui('第三周', 'Third') }}</option>
                          <option :value="4">{{ ui('第四周', 'Fourth') }}</option>
                          <option :value="5">{{ ui('第五周', 'Fifth') }}</option>
                          <option :value="-1">{{ ui('最后一周', 'Last') }}</option>
                        </select>
                        <small
                          v-if="scheduleFieldError('weekOfMonth')"
                          class="compute-editor__field-error"
                        >
                          {{ scheduleFieldError('weekOfMonth') }}
                        </small>
                      </label>
                      <label class="compute-editor__field">
                        <span>{{ ui('星期', 'Weekday') }}</span>
                        <select
                          v-model.number="activeDraft.triggerConfig.weekday"
                          @change="markScheduleDirty"
                        >
                          <option v-for="day in weekdayOptions" :key="day.value" :value="day.value">
                            {{ ui(`星期${day.label}`, day.label) }}
                          </option>
                        </select>
                        <small
                          v-if="scheduleFieldError('weekday')"
                          class="compute-editor__field-error"
                        >
                          {{ scheduleFieldError('weekday') }}
                        </small>
                      </label>
                    </template>
                  </div>
                  <label
                    v-if="activeDraft.triggerConfig.kind !== 'interval'"
                    class="compute-editor__field"
                  >
                    <span>{{ ui('时区', 'Time Zone') }}</span>
                    <input
                      v-model="activeDraft.triggerConfig.timezone"
                      list="compute-timezones"
                      placeholder="Asia/Shanghai"
                      autocomplete="off"
                      @input="markScheduleDirty"
                    />
                    <datalist id="compute-timezones">
                      <option
                        v-for="timezone in timezoneOptions"
                        :key="timezone"
                        :value="timezone"
                      />
                    </datalist>
                    <small
                      v-if="scheduleFieldError('timezone')"
                      class="compute-editor__field-error"
                    >
                      {{ scheduleFieldError('timezone') }}
                    </small>
                  </label>
                  <div class="compute-editor__schedule-window">
                    <label class="compute-editor__field">
                      <span>{{ ui('开始时间', 'Start Time') }}</span>
                      <input
                        type="datetime-local"
                        step="1"
                        :value="scheduleDateTimeValue('startAt')"
                        @input="updateScheduleDateTime('startAt', $event)"
                      />
                      <small
                        v-if="scheduleFieldError('startAt')"
                        class="compute-editor__field-error"
                      >
                        {{ scheduleFieldError('startAt') }}
                      </small>
                    </label>
                    <label class="compute-editor__field">
                      <span>{{ ui('结束时间', 'End Time') }}</span>
                      <input
                        type="datetime-local"
                        step="1"
                        :value="scheduleDateTimeValue('endAt')"
                        @input="updateScheduleDateTime('endAt', $event)"
                      />
                      <small v-if="scheduleFieldError('endAt')" class="compute-editor__field-error">
                        {{ scheduleFieldError('endAt') }}
                      </small>
                    </label>
                    <label class="compute-editor__field">
                      <span>{{ ui('最多执行', 'Maximum Runs') }}</span>
                      <div class="compute-editor__field-inline">
                        <input
                          v-model.number="activeDraft.triggerConfig.maxRuns"
                          type="number"
                          min="1"
                          max="1000000"
                          :placeholder="ui('不限', 'Unlimited')"
                          @input="markScheduleDirty"
                        />
                        <span>{{ ui('次', 'runs') }}</span>
                      </div>
                      <small
                        v-if="scheduleFieldError('maxRuns')"
                        class="compute-editor__field-error"
                      >
                        {{ scheduleFieldError('maxRuns') }}
                      </small>
                    </label>
                  </div>
                  <p class="compute-editor__trigger-note">
                    {{ ui('该计划由节点运行时执行；开发态不会自动运行。', 'This schedule runs on the node runtime and does not run automatically during development.') }}
                  </p>
                  <div class="compute-editor__trigger-preview">
                    <button
                      type="button"
                      class="compute-editor__preview-button"
                      :title="ui('预览后续执行', 'Preview Upcoming Runs')"
                      :disabled="schedulePreviewLoading"
                      @click="previewSchedule"
                    >
                      {{ schedulePreviewLoading ? ui('预览中…', 'Previewing…') : ui('预览后续执行', 'Preview Upcoming Runs') }}
                    </button>
                    <div
                      v-if="schedulePreviewSummary"
                      class="compute-editor__trigger-preview-result"
                    >
                      <strong>{{ schedulePreviewSummary }}</strong>
                      <ol v-if="schedulePreviewRuns.length">
                        <li v-for="(run, index) in schedulePreviewRuns" :key="run">
                          {{ ui(`第 ${index + 1} 次：`, `Run ${index + 1}: `) }}{{ formatScheduleRun(run) }}
                        </li>
                      </ol>
                    </div>
                    <ul v-if="schedulePreviewErrors.length" class="compute-editor__schedule-errors">
                      <li
                        v-for="item in schedulePreviewErrors"
                        :key="`${item.field}:${item.message}`"
                      >
                        {{ item.message }}
                      </li>
                    </ul>
                  </div>
                </div>
                <div
                  v-else-if="activeDraft.triggerType === 'datapoint_change'"
                  class="compute-editor__trigger-card is-datapoint-change"
                >
                  <div class="compute-editor__datapoint-trigger-source">
                    <label class="compute-editor__field">
                      <span>{{ ui('数据点', 'Data Point') }}</span>
                      <input
                        v-model="activeDraft.triggerConfig.path"
                        :placeholder="ui('选择数据点', 'Select a data point')"
                        readonly
                      />
                    </label>
                    <button
                      type="button"
                      class="compute-editor__inline-icon"
                      :title="ui('选择数据点', 'Select Data Point')"
                      :aria-label="ui('选择数据点', 'Select Data Point')"
                      @click="openDatapointPicker('trigger')"
                    >
                      <IconTablerDatabaseImport class="compute-editor__action-icon" />
                    </button>
                  </div>
                  <label class="compute-editor__field">
                    <span>{{ ui('变化类型', 'Change Type') }}</span>
                    <select
                      v-model="activeDraft.triggerConfig.mode"
                      @change="handlePointChangeMode"
                    >
                      <option value="any">{{ ui('任意更新', 'Any Update') }}</option>
                      <option value="value_change">{{ ui('值发生变化', 'Value Changed') }}</option>
                      <option value="increase" :disabled="!isPointChangeModeCompatible('increase')">
                        {{ ui('数值增大', 'Value Increased') }}
                      </option>
                      <option value="decrease" :disabled="!isPointChangeModeCompatible('decrease')">
                        {{ ui('数值减小', 'Value Decreased') }}
                      </option>
                      <option
                        value="rising_edge"
                        :disabled="!isPointChangeModeCompatible('rising_edge')"
                      >
                        {{ ui('上升沿', 'Rising Edge') }}
                      </option>
                      <option
                        value="falling_edge"
                        :disabled="!isPointChangeModeCompatible('falling_edge')"
                      >
                        {{ ui('下降沿', 'Falling Edge') }}
                      </option>
                    </select>
                  </label>
                  <label class="compute-editor__field">
                    <span>{{ ui('防抖时间', 'Debounce') }}</span>
                    <div class="compute-editor__field-inline">
                      <input
                        v-model.number="activeDraft.triggerConfig.debounceMs"
                        type="number"
                        min="0"
                        max="3600000"
                        @input="markDirty"
                      />
                      <span>ms</span>
                    </div>
                  </label>
                  <label class="compute-editor__field">
                    <span>{{ ui('变化死区', 'Deadband') }}</span>
                    <input
                      v-model.number="activeDraft.triggerConfig.deadband"
                      type="number"
                      min="0"
                      :disabled="!pointChangeDeadbandAvailable"
                      :placeholder="ui('不限制', 'No Limit')"
                      @input="markDirty"
                    />
                  </label>
                  <p class="compute-editor__trigger-note">
                    {{ ui('触发条件由节点运行时判断；开发态仅保存配置。', 'The node runtime evaluates this trigger; development mode only saves the configuration.') }}
                  </p>
                </div>
                <div
                  v-else-if="activeDraft.triggerType === 'condition'"
                  class="compute-editor__trigger-card is-condition"
                >
                  <label class="compute-editor__field compute-editor__condition-expression">
                    <span>{{ ui('条件表达式', 'Condition Expression') }}</span>
                    <textarea
                      v-model="activeDraft.triggerConfig.expression"
                      rows="2"
                      :placeholder="ui('例如：temperature > 80 && enabled', 'Example: temperature > 80 && enabled')"
                      @input="markDirty"
                    />
                  </label>
                  <div class="compute-editor__condition-variables">
                    <span>{{ ui('可用变量', 'Available Variables') }}</span>
                    <div v-if="conditionVariableRows.length">
                      <button
                        v-for="row in conditionVariableRows"
                        :key="row.uid"
                        type="button"
                        :title="`${row.path}（${row.dataType}）`"
                        @click="appendConditionVariable(row.alias)"
                      >
                        {{ row.alias }}
                      </button>
                    </div>
                    <small v-else>{{ ui('请先在“变量”中添加 bool、string 或数值数据点。', 'Add a bool, string, or numeric data point under Variables first.') }}</small>
                  </div>
                  <fieldset class="compute-editor__condition-phases">
                    <legend>{{ ui('触发阶段', 'Trigger Phases') }}</legend>
                    <label>
                      <input
                        v-model="activeDraft.triggerConfig.phases"
                        type="checkbox"
                        value="entered"
                        @change="ensureConditionPhase('entered')"
                      />
                      {{ ui('进入', 'Entered') }}
                    </label>
                    <label>
                      <input
                        v-model="activeDraft.triggerConfig.phases"
                        type="checkbox"
                        value="active"
                        @change="ensureConditionPhase('active')"
                      />
                      {{ ui('持续', 'Active') }}
                    </label>
                    <label>
                      <input
                        v-model="activeDraft.triggerConfig.phases"
                        type="checkbox"
                        value="exited"
                        @change="ensureConditionPhase('exited')"
                      />
                      {{ ui('离开', 'Exited') }}
                    </label>
                  </fieldset>
                  <label class="compute-editor__field">
                    <span>{{ ui('防抖时间', 'Debounce') }}</span>
                    <div class="compute-editor__field-inline">
                      <input
                        v-model.number="activeDraft.triggerConfig.debounceMs"
                        type="number"
                        min="0"
                        max="3600000"
                        @input="markDirty"
                      />
                      <span>ms</span>
                    </div>
                  </label>
                  <p class="compute-editor__trigger-note">
                    {{ ui('进入、离开对应条件状态切换；持续在条件成立期间每次引用数据点变化时执行。节点通过 ctx.trigger.phase 提供 entered、active 或 exited。', 'Entered and exited correspond to condition state changes. Active runs whenever a referenced data point changes while the condition remains true. The node exposes entered, active, or exited through ctx.trigger.phase.') }}
                  </p>
                </div>
                <div v-else class="compute-editor__trigger-card is-note">
                  <strong>{{ ui('手动触发', 'Manual Trigger') }}</strong>
                  <span>{{ ui('脚本只在调用方调用、调试或手动运行时执行。', 'The script runs only when invoked by a caller, debugger, or manual action.') }}</span>
                </div>
              </section>
            </div>
          </template>

          <template v-else-if="activePanel === 'dependencies'">
            <div class="compute-editor__panel-head">
              <div>
                <h3>
                  <span>{{ ui('依赖库', 'Dependencies') }}</span>
                  <button
                    type="button"
                    class="compute-editor__help-dot"
                    :title="ui(`当前语言：${langText(activeDraft.lang)}，已选 ${dependencyCount} 个。`, `Language: ${langText(activeDraft.lang)}. ${dependencyCount} selected.`)"
                    :aria-label="ui('依赖库说明', 'Dependency help')"
                  >
                    ?
                  </button>
                </h3>
              </div>
              <button
                type="button"
                class="compute-editor__tool-btn"
                :title="ui('管理工程依赖', 'Manage Project Dependencies')"
                :aria-label="ui('管理工程依赖', 'Manage Project Dependencies')"
                @click="$emit('manage-dependencies')"
              >
                <IconTablerSettings class="compute-editor__action-icon" />
              </button>
            </div>
            <div v-if="activeDraft.dependencies.length" class="compute-editor__dependency-chips">
              <button
                v-for="item in activeDraft.dependencies"
                :key="item.id"
                type="button"
                :title="dependencyName(item.id)"
                disabled
              >
                <span>{{ dependencyName(item.id) }}</span>
              </button>
            </div>
            <div v-if="dependenciesError" class="compute-editor__empty">
              {{ dependenciesError }}
            </div>
            <div v-else-if="!filteredDependencies.length" class="compute-editor__empty">
              {{ ui('当前语言无可用依赖', 'No dependencies available for this language') }}
            </div>
            <div v-else class="compute-editor__dependency-table">
              <div
                v-for="dep in filteredDependencies"
                :key="dep.id"
                class="compute-editor__dependency-row"
              >
                <strong>{{ dep.name }}</strong>
                <code>{{ dep.importName || dep.name }}</code>
                <span>{{ dep.version || '-' }}</span>
                <em>{{ isDependencyChecked(dep.id) ? ui('当前代码已使用', 'Used by Current Code') : ui('工程已安装', 'Installed in Project') }}</em>
              </div>
            </div>
          </template>

          <template v-else-if="activePanel === 'debug'">
            <div class="compute-editor__panel-head compute-editor__debug-panel-head">
              <div>
                <h3>
                  <span>{{ ui('调试', 'Debug') }}</span>
                  <button
                    type="button"
                    class="compute-editor__help-dot"
                    :title="
                      activeDraft.dirty
                        ? ui('保存后才能试运行。', 'Save before running a dry run.')
                        : ui('试运行只检查脚本返回值，不会保存结果或写入数据点。', 'A dry run only inspects the script return value; it does not save results or write data points.')
                    "
                    :aria-label="ui('调试说明', 'Debug help')"
                  >
                    ?
                  </button>
                </h3>
              </div>
              <div class="compute-editor__debug-actions">
                <span class="compute-editor__tooltip-wrap" :title="dryRunTooltip">
                  <button
                    type="button"
                    class="compute-editor__run-button compute-editor__debug-run-button"
                    :disabled="debugRunning || activeDraft.dirty || !sandboxAvailable"
                    @click="executeDebug"
                  >
                    <IconTablerPlayerPlay class="compute-editor__action-icon" />
                    {{ ui('试运行', 'Dry Run') }}
                  </button>
                </span>
              </div>
            </div>
            <div class="compute-editor__debug-layout">
              <div class="compute-editor__debug-inputs">
                <label class="compute-editor__debug-input">
                  <div class="compute-editor__debug-input-head">
                    <span>{{ ui('调用参数 JSON', 'Arguments JSON') }}</span>
                    <button
                      type="button"
                      :title="ui('按参数定义重新生成', 'Regenerate from parameter definitions')"
                      @click="resetDebugArgvFromDefinition"
                    >
                      {{ ui('按定义重置', 'Reset from Definitions') }}
                    </button>
                  </div>
                  <textarea v-model="debugArgvText" spellcheck="false" />
                  <p
                    v-if="debugArgvHint"
                    class="compute-editor__debug-hint"
                    :class="{ 'is-warning': debugArgvHintTone === 'warning' }"
                  >
                    {{ debugArgvHint }}
                  </p>
                </label>
                <label class="compute-editor__debug-input">
                  <div class="compute-editor__debug-input-head">
                    <span>{{ ui('数据点模拟快照 JSON', 'Data Point Snapshot JSON') }}</span>
                    <button
                      type="button"
                      :title="ui('按数据点变量重新生成', 'Regenerate from data point variables')"
                      @click="resetDebugDatapointsFromDefinition"
                    >
                      {{ ui('按定义重置', 'Reset from Definitions') }}
                    </button>
                  </div>
                  <textarea v-model="debugDatapointText" spellcheck="false" />
                  <p
                    v-if="debugDatapointHint"
                    class="compute-editor__debug-hint"
                    :class="{ 'is-warning': debugDatapointHintTone === 'warning' }"
                  >
                    {{ debugDatapointHint }}
                  </p>
                </label>
              </div>
              <section class="compute-editor__debug-output">
                <div class="compute-editor__debug-result-tabs">
                  <button
                    v-for="tab in debugResultTabs"
                    :key="tab.id"
                    type="button"
                    :class="{
                      'is-active': activeDebugResultTab === tab.id,
                      'has-error': tab.id === 'error' && Boolean(debugErrorText),
                    }"
                    @click="activeDebugResultTab = tab.id"
                  >
                    {{ tab.label }}
                  </button>
                </div>
                <pre>{{ activeDebugResultText }}</pre>
              </section>
            </div>
          </template>

          <template v-else-if="activePanel === 'problems'">
            <div class="compute-editor__panel-head">
              <div>
                <h3>
                  <span>{{ ui('问题', 'Problems') }}</span>
                  <button
                    type="button"
                    class="compute-editor__help-dot"
                    :title="syntaxStatusText"
                    :aria-label="ui('问题说明', 'Problems help')"
                  >
                    ?
                  </button>
                </h3>
              </div>
              <button
                type="button"
                class="compute-editor__tool-btn"
                :title="ui('重新检查', 'Check Again')"
                :aria-label="ui('重新检查', 'Check Again')"
                :disabled="syntaxStatus === 'checking' || !sandboxAvailable"
                @click="runSyntaxCheck"
              >
                <IconTablerRefresh class="compute-editor__action-icon" />
              </button>
            </div>
            <div
              v-if="syntaxStatus === 'clean' && !syntaxDiagnostics.length"
              class="compute-editor__empty"
            >
              {{ ui('没有语法问题', 'No Syntax Problems') }}
            </div>
            <div v-else-if="syntaxStatus === 'failed'" class="compute-editor__empty">
              {{ syntaxErrorText || ui('语法检查失败', 'Syntax Check Failed') }}
            </div>
            <div v-else class="compute-editor__problem-list">
              <button
                v-for="item in syntaxDiagnostics"
                :key="`${item.line}:${item.column}:${item.message}`"
                type="button"
                class="compute-editor__problem-row"
                @click="revealDiagnostic(item)"
              >
                <IconTablerAlertTriangle class="compute-editor__problem-icon" />
                <span>{{ item.message }}</span>
                <code>{{ item.line }}:{{ item.column }}</code>
              </button>
            </div>
          </template>
        </div>
      </section>
    </template>

    <DatapointPickerDialog
      v-model="datapointPickerVisible"
      :project-id="projectId"
      :multiple="false"
      :title="datapointPickerIntent === 'trigger' ? ui('选择变化触发数据点', 'Select Change Trigger Data Point') : ui('选择数据点变量', 'Select Data Point Variable')"
      :confirm-text="datapointPickerIntent === 'trigger' ? ui('选择', 'Select') : ui('插入', 'Insert')"
      :disabled-reason="
        datapointPickerIntent === 'trigger' ? datapointTriggerDisabledReason : undefined
      "
      @apply="confirmSelectedDatapoint"
    />

    <DcDialog
      v-model="templateDialogVisible"
      :title="ui('代码模板', 'Code Templates')"
      width="760px"
      body-max-height="560px"
    >
      <div class="compute-editor__template-dialog">
        <aside class="compute-editor__template-list">
          <button
            v-for="template in codeTemplates"
            :key="template.id"
            type="button"
            :class="{ 'is-active': template.id === activeTemplateId }"
            :disabled="template.disabled"
            @click="activeTemplateId = template.id"
          >
            <strong>{{ template.name }}</strong>
            <span>{{ template.description }}</span>
          </button>
        </aside>
        <section class="compute-editor__template-preview">
          <div class="compute-editor__template-meta">
            <strong>{{ activeTemplate?.name }}</strong>
            <span>{{ activeTemplate?.description }}</span>
          </div>
          <pre>{{ activeTemplate?.code }}</pre>
          <button
            type="button"
            class="compute-editor__picker-confirm"
            :disabled="activeTemplate?.disabled"
            @click="insertActiveTemplate"
          >
            {{ ui('插入到光标', 'Insert at Cursor') }}
          </button>
        </section>
      </div>
    </DcDialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import IconTablerAlertTriangle from '~icons/tabler/alert-triangle'
import IconTablerChevronUp from '~icons/tabler/chevron-up'
import IconTablerCircleCheck from '~icons/tabler/circle-check'
import IconTablerCircleDashed from '~icons/tabler/circle-dashed'
import IconTablerCode from '~icons/tabler/code'
import IconTablerDatabaseImport from '~icons/tabler/database-import'
import IconTablerDeviceFloppy from '~icons/tabler/device-floppy'
import IconTablerGitFork from '~icons/tabler/git-fork'
import IconTablerManualGearbox from '~icons/tabler/manual-gearbox'
import IconTablerPlugConnected from '~icons/tabler/plug-connected'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerPlayerPause from '~icons/tabler/player-pause'
import IconTablerPlayerPlay from '~icons/tabler/player-play'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerSettings from '~icons/tabler/settings'
import IconTablerTemplate from '~icons/tabler/template'
import IconTablerTerminal2 from '~icons/tabler/terminal-2'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerClock from '~icons/tabler/clock'
import IconTablerX from '~icons/tabler/x'
import type {
  ComputeDependency,
  ComputeCapabilities,
  ComputeLang,
  ComputeRunResult,
  ComputeSyntaxDiagnostic,
} from '@/api/schemas/compute.schema'
import { checkComputeSyntax, debugComputeUnit, previewComputeSchedule } from '@/api/compute.api'
import { getApiErrorMessage } from '@/utils/request'
import MonacoEditor from '@/components/MonacoEditor.vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import DatapointPickerDialog, {
  type DatapointPickerSelection,
} from '@/components/shared/DatapointPickerDialog.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import { datacenterLocale } from '@/i18n/runtime'
import type {
  ComputeDatapointVariableRow,
  ComputeDraft,
  ComputeEditorTab,
} from './computeEditorModel'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)
const monthName = (month: number) =>
  new Intl.DateTimeFormat(datacenterLocale.value === 'en' ? 'en-US' : 'zh-CN', {
    month: 'long',
  }).format(new Date(2024, month - 1, 1))

type MonacoEditorExpose = InstanceType<typeof MonacoEditor> & {
  insertText?: (text: string) => void
  revealPosition?: (line: number, column?: number) => void
  setDiagnostics?: (diagnostics: ComputeSyntaxDiagnostic[]) => void
}

type EditorCursorInfo = {
  line: number
  column: number
  spaces: number
}

const props = withDefaults(
  defineProps<{
    projectId: string
    tabs: ComputeEditorTab[]
    activeId: string | null
    loading?: boolean
    saving?: boolean
    deleting?: boolean
    error?: string
    dependencies?: ComputeDependency[]
    dependenciesLoading?: boolean
    dependenciesError?: string
    capabilities?: ComputeCapabilities | null
    capabilitiesLoading?: boolean
  }>(),
  {
    loading: false,
    saving: false,
    deleting: false,
    error: '',
    dependencies: () => [],
    dependenciesLoading: false,
    dependenciesError: '',
    capabilities: null,
    capabilitiesLoading: false,
  },
)

// 编辑草稿由工作区持有，显式声明为双向模型，避免子组件绕过组件契约修改 props。
const activeDraft = defineModel<ComputeDraft | null>('activeDraft', { required: true })

const emit = defineEmits<{
  (event: 'activate-tab', id: string): void
  (event: 'close-tab', id: string): void
  (event: 'save', id: string): void
  (event: 'toggle-enabled', id: string, enabled: boolean): void
  (event: 'delete-unit', id: string): void
  (event: 'mark-dirty', id: string): void
  (event: 'refresh-dependencies'): void
  (event: 'manage-dependencies'): void
  (event: 'retry-capabilities'): void
}>()

const panelDefaultHeight = 260
const panelMaxHeight = 520
const panelMinHeight = 180
const activePanel = ref('inputs')
const panelCollapsed = ref(true)
const panelHeight = ref(panelDefaultHeight)
const inspectorOpen = ref(false)
const monacoEditorRef = ref<MonacoEditorExpose | null>(null)
const datapointPickerVisible = ref(false)
const datapointPickerIntent = ref<'variable' | 'trigger'>('variable')
const debugArgvText = ref('[]')
const debugDatapointText = ref('{}')
const debugRunning = ref(false)
const debugResult = ref<ComputeRunResult | null>(null)
const debugError = ref('')
const syntaxStatus = ref<'idle' | 'checking' | 'clean' | 'dirty' | 'failed'>('idle')
const syntaxDiagnostics = ref<ComputeSyntaxDiagnostic[]>([])
const syntaxErrorText = ref('')
let syntaxCheckTimer: ReturnType<typeof window.setTimeout> | null = null
let syntaxCheckSeq = 0
let syntaxCheckController: AbortController | null = null
let debugController: AbortController | null = null
const templateDialogVisible = ref(false)
const activeTemplateId = ref('argv')
type SchedulePreviewError = { field: string; message: string }
const schedulePreviewSummary = ref('')
const schedulePreviewRuns = ref<string[]>([])
const schedulePreviewErrors = ref<SchedulePreviewError[]>([])
const schedulePreviewLoading = ref(false)
let schedulePreviewSeq = 0
const activeDebugResultTab = ref('output')
const cursorInfo = ref<EditorCursorInfo>({
  line: 1,
  column: 1,
  spaces: 2,
})

const panelTabs = computed(() => [
  { id: 'inputs', label: ui('参数', 'Parameters'), icon: IconTablerPlugConnected },
  { id: 'variables', label: ui('变量', 'Variables'), icon: IconTablerDatabaseImport },
  { id: 'trigger', label: ui('触发', 'Trigger'), icon: IconTablerPlayerPlay },
  { id: 'dependencies', label: ui('依赖', 'Dependencies'), icon: IconTablerGitFork },
  { id: 'debug', label: ui('调试', 'Debug'), icon: IconTablerTerminal2 },
  { id: 'problems', label: ui('问题', 'Problems'), icon: IconTablerAlertTriangle },
])

const debugResultTabs = computed(() => [
  { id: 'output', label: ui('返回值', 'Return Value') },
  { id: 'logs', label: ui('日志', 'Logs') },
  { id: 'error', label: ui('错误', 'Errors') },
])

const sandboxAvailable = computed(() => props.capabilities?.sandboxStatus === 'available')
const sandboxUnavailableReason = computed(
  () => props.capabilities?.sandboxReason?.trim() || ui('隔离执行检查未通过', 'Isolation check failed'),
)

const codeTemplates = computed(() => {
  const lang = activeDraft.value?.lang === 'python' ? 'python' : 'javascript'
  const pointAlias = activeDraft.value?.datapointVariableRows[0]?.alias.trim() || ''
  const templates =
    lang === 'python' ? pythonTemplates(pointAlias) : javascriptTemplates(pointAlias)
  return templates
})

const computeTypeDefinitions = computed(() => {
  if (activeDraft.value?.lang === 'python') return ''
  const declarations = (activeDraft.value?.datapointVariableRows || [])
    .filter((row) => row.alias.trim())
    .map(
      (row) =>
        `declare const ${row.alias.trim()}: InduForgeDataPoint<${monacoTypeForDatapoint(row.dataType)}>;`,
    )
    .join('\n')
  return `
type InduForgeSDKResult<T> = Readonly<{ code: number; msg: string; data: T | null }>;
type InduForgeDataPointSample<T> = Readonly<{
  path: string;
  value: T;
  quality: string;
  timestamp: string | null;
  observedAt: string | null;
  sourceTimestamp: string | null;
  status: string | null;
}>;
type InduForgeDataPointCapabilities = Readonly<{
  get: boolean; read: boolean; peek: boolean; set: boolean; subscribe: boolean;
  history: boolean; refresh: boolean; run: boolean; execute: boolean; publish: boolean;
}>;
interface InduForgeDataPoint<T = unknown> {
  readonly id: string | null;
  readonly ref: string;
  readonly path: string;
  readonly name: string | null;
  readonly displayName: string | null;
  readonly dataType: string | null;
  readonly schema: unknown;
  readonly source: Readonly<{ type: string | null; id: string | null }>;
  readonly status: string | null;
  readonly unit: string | null;
  readonly precision: number | null;
  readonly min: number | null;
  readonly max: number | null;
  readonly defaultValue: T | null;
  readonly tags: readonly unknown[];
  readonly attributes: Readonly<Record<string, string>>;
  readonly capabilities: InduForgeDataPointCapabilities;
  get(options?: unknown): Promise<InduForgeSDKResult<InduForgeDataPointSample<T>>>;
  read(options?: unknown): Promise<InduForgeSDKResult<InduForgeDataPointSample<T>>>;
  peek(): Promise<InduForgeSDKResult<InduForgeDataPointSample<T>>>;
  set(value: T, options?: unknown): Promise<InduForgeSDKResult<unknown>>;
  subscribe(handler: (sample: InduForgeDataPointSample<T>) => void, options?: unknown): Promise<InduForgeSDKResult<unknown>>;
  history(query?: unknown): Promise<InduForgeSDKResult<unknown>>;
  refresh(options?: unknown): Promise<InduForgeSDKResult<unknown>>;
  run(input?: unknown): Promise<InduForgeSDKResult<unknown>>;
  execute(input?: unknown): Promise<InduForgeSDKResult<unknown>>;
  publish(payload?: unknown, options?: unknown): Promise<InduForgeSDKResult<unknown>>;
}
${declarations}
`
})

const computeMemberCompletions = computed(() => {
  if (activeDraft.value?.lang !== 'python') return {}
  return Object.fromEntries(
    (activeDraft.value?.datapointVariableRows || [])
      .filter((row) => row.alias.trim())
      .map((row) => [row.alias.trim(), datapointMemberCompletions.value]),
  )
})

function monacoTypeForDatapoint(dataType?: string) {
  const type = String(dataType || '').toLowerCase()
  if (numericDatapointTypes.has(type) || type === 'number') return 'number'
  if (type === 'bool' || type === 'boolean') return 'boolean'
  if (type === 'array') return 'unknown[]'
  if (type === 'object') return 'Record<string, unknown>'
  return 'string'
}

const datapointMemberCompletions = computed(() => [
  { label: 'get', insertText: 'get()', kind: 'method', detail: ui('获取业务值快照', 'Get the business-value snapshot') },
  { label: 'read', insertText: 'read()', kind: 'method', detail: ui('读取完整当前样本', 'Read the complete current sample') },
  { label: 'peek', insertText: 'peek()', kind: 'method', detail: ui('读取已物化当前样本', 'Read the materialized current sample') },
  { label: 'set', insertText: 'set(value)', kind: 'method', detail: ui('写入数据点', 'Write the data point') },
  { label: 'subscribe', insertText: 'subscribe(handler)', kind: 'method', detail: ui('订阅数据变化', 'Subscribe to data changes') },
  { label: 'history', insertText: 'history(query)', kind: 'method', detail: ui('查询历史样本', 'Query historical samples') },
  { label: 'refresh', insertText: 'refresh()', kind: 'method', detail: ui('主动刷新来源', 'Refresh the source') },
  { label: 'run', insertText: 'run()', kind: 'method', detail: ui('运行计算数据点', 'Run a compute data point') },
  { label: 'execute', insertText: 'execute()', kind: 'method', detail: ui('执行按需数据点', 'Execute an on-demand data point') },
  { label: 'publish', insertText: 'publish(payload)', kind: 'method', detail: ui('发布消息', 'Publish a message') },
  ...[
    'id',
    'ref',
    'path',
    'name',
    'displayName',
    'dataType',
    'schema',
    'source',
    'status',
    'unit',
    'precision',
    'min',
    'max',
    'defaultValue',
    'tags',
    'attributes',
    'capabilities',
  ].map((label) => ({ label, insertText: label, kind: 'property', detail: ui('数据点属性', 'Data point property') })),
])

const allTriggerTypes = computed(() => [
  { id: 'manual', label: ui('手动', 'Manual') },
  { id: 'schedule_interval', label: ui('周期执行', 'Interval') },
  { id: 'schedule_daily', label: ui('每日定时', 'Daily') },
  { id: 'schedule_weekly', label: ui('每周定时', 'Weekly') },
  { id: 'schedule_monthly', label: ui('每月定时', 'Monthly') },
  { id: 'schedule_yearly', label: ui('每年定时', 'Yearly') },
  { id: 'datapoint_change', label: ui('数据点变化', 'Data Point Change') },
  { id: 'condition', label: ui('条件触发', 'Condition') },
])

const triggerTypes = computed(() => {
  const supported = props.capabilities?.triggerTypes
  if (!supported?.length) return allTriggerTypes.value
  return allTriggerTypes.value.filter((item) => {
    const capability = item.id.startsWith('schedule_') ? 'schedule' : item.id
    return supported.includes(capability)
  })
})

const timezoneOptions = computed(() => {
  const current = String(activeDraft.value?.triggerConfig.timezone || '')
  const intlWithValues = Intl as typeof Intl & {
    supportedValuesOf?: (key: 'timeZone') => string[]
  }
  const supported = intlWithValues.supportedValuesOf?.('timeZone') || []
  return Array.from(new Set(['Asia/Shanghai', current, ...supported].filter(Boolean)))
})

const weekdayOptions = computed(() => [
  { value: 1, label: ui('一', 'Monday') },
  { value: 2, label: ui('二', 'Tuesday') },
  { value: 3, label: ui('三', 'Wednesday') },
  { value: 4, label: ui('四', 'Thursday') },
  { value: 5, label: ui('五', 'Friday') },
  { value: 6, label: ui('六', 'Saturday') },
  { value: 7, label: ui('日', 'Sunday') },
])

type CodeTemplate = {
  id: string
  name: string
  description: string
  code: string
  disabled?: boolean
}

function javascriptTemplates(pointAlias: string): CodeTemplate[] {
  const disabled = !pointAlias
  const point = pointAlias || 'point'
  const qualityError = ui('数据质量不可用', 'Data quality is unavailable')
  const writeError = ui('数据点不支持写入', 'The data point does not support writes')
  return [
    {
      id: 'argv',
      name: ui('获取脚本参数', 'Read Script Arguments'),
      description: ui('读取调用方传入的 argv 参数。', 'Read argv arguments supplied by the caller.'),
      code: 'const firstArg = argv[0];\nconst secondArg = argv[1];\n',
    },
    {
      id: 'datapoint-read',
      name: ui('读取数据点样本', 'Read Data Point Sample'),
      description: disabled ? ui('请先在变量面板添加数据点。', 'Add a data point in Variables first.') : ui(`读取 ${point} 的值、质量和时间。`, `Read the value, quality, and timestamp of ${point}.`),
      code: `const result = await ${point}.read();
if (result.code !== 0) return result;
if (result.data.quality !== 'good') {
  return { error: ${JSON.stringify(qualityError)}, sample: result.data };
}
return result.data.value;
`,
      disabled,
    },
    {
      id: 'datapoint-get',
      name: ui('获取数据点业务值', 'Get Data Point Value'),
      description: disabled ? ui('请先在变量面板添加数据点。', 'Add a data point in Variables first.') : ui(`调用 ${point}.get() 获取业务值。`, `Call ${point}.get() to obtain its business value.`),
      code: `const result = await ${point}.get();
if (result.code !== 0) return result;
return result.data.value;
`,
      disabled,
    },
    {
      id: 'datapoint-set',
      name: ui('写入数据点', 'Write Data Point'),
      description: disabled ? ui('请先在变量面板添加数据点。', 'Add a data point in Variables first.') : ui(`检查能力后写入 ${point}。`, `Check capabilities before writing ${point}.`),
      code: `if (!${point}.capabilities.set) {
  return { error: ${JSON.stringify(writeError)} };
}
const result = await ${point}.set(argv[0]);
if (result.code !== 0) return result;
return result.data;
`,
      disabled,
    },
    {
      id: 'output',
      name: ui('输出结果', 'Return Result'),
      description: ui('把脚本结果返回给调用方和输出数据点。', 'Return the script result to the caller and output data points.'),
      code: 'return {\n  value: null,\n  updatedAt: new Date().toISOString()\n};\n',
    },
    {
      id: 'try-catch',
      name: ui('错误处理', 'Error Handling'),
      description: ui('捕获异常并返回明确错误。', 'Catch exceptions and return a clear error.'),
      code: 'try {\n  return null;\n} catch (error) {\n  return { error: String(error && error.message ? error.message : error) };\n}\n',
    },
  ]
}

function pythonTemplates(pointAlias: string): CodeTemplate[] {
  const disabled = !pointAlias
  const point = pointAlias || 'point'
  const qualityError = ui('数据质量不可用', 'Data quality is unavailable')
  const writeError = ui('数据点不支持写入', 'The data point does not support writes')
  return [
    {
      id: 'argv',
      name: ui('获取脚本参数', 'Read Script Arguments'),
      description: ui('读取调用方传入的 argv 参数。', 'Read argv arguments supplied by the caller.'),
      code: 'def main(argv, dp, ctx):\n    first_arg = argv[0] if len(argv) > 0 else None\n    second_arg = argv[1] if len(argv) > 1 else None\n    return first_arg\n',
    },
    {
      id: 'datapoint-read',
      name: ui('读取数据点样本', 'Read Data Point Sample'),
      description: disabled ? ui('请先在变量面板添加数据点。', 'Add a data point in Variables first.') : ui(`读取 ${point} 的值、质量和时间。`, `Read the value, quality, and timestamp of ${point}.`),
      code: `def main(argv, dp, ctx):
    result = ${point}.read()
    if result.code != 0:
        return result
    if result.data["quality"] != "good":
        return {"error": ${JSON.stringify(qualityError)}, "sample": result.data}
    return result.data["value"]
`,
      disabled,
    },
    {
      id: 'datapoint-get',
      name: ui('获取数据点业务值', 'Get Data Point Value'),
      description: disabled ? ui('请先在变量面板添加数据点。', 'Add a data point in Variables first.') : ui(`调用 ${point}.get() 获取业务值。`, `Call ${point}.get() to obtain its business value.`),
      code: `def main(argv, dp, ctx):
    result = ${point}.get()
    if result.code != 0:
        return result
    return result.data["value"]
`,
      disabled,
    },
    {
      id: 'datapoint-set',
      name: ui('写入数据点', 'Write Data Point'),
      description: disabled ? ui('请先在变量面板添加数据点。', 'Add a data point in Variables first.') : ui(`检查能力后写入 ${point}。`, `Check capabilities before writing ${point}.`),
      code: `def main(argv, dp, ctx):
    if not ${point}.capabilities["set"]:
        return {"error": ${JSON.stringify(writeError)}}
    value = argv[0] if len(argv) > 0 else None
    result = ${point}.set(value)
    if result.code != 0:
        return result
    return result.data
`,
      disabled,
    },
    {
      id: 'output',
      name: ui('输出结果', 'Return Result'),
      description: ui('把脚本结果返回给调用方和输出数据点。', 'Return the script result to the caller and output data points.'),
      code: 'def main(argv, dp, ctx):\n    return {\n        "value": None\n    }\n',
    },
    {
      id: 'try-catch',
      name: ui('错误处理', 'Error Handling'),
      description: ui('捕获异常并返回明确错误。', 'Catch exceptions and return a clear error.'),
      code: 'def main(argv, dp, ctx):\n    try:\n        return None\n    except Exception as error:\n        return {"error": str(error)}\n',
    },
  ]
}

const monacoLanguage = computed(() => {
  if (activeDraft.value?.lang === 'python') return 'python'
  return 'javascript'
})

const outputTypeOptions = [
  'bool',
  'int8',
  'uint8',
  'int16',
  'uint16',
  'int32',
  'uint32',
  'int64',
  'uint64',
  'float32',
  'float64',
  'decimal',
  'string',
  'bytes',
  'datetime',
  'object',
  'array',
]

const normalizedUnitPath = () =>
  (activeDraft.value?.name?.trim() || 'unnamed').replace(/[^a-zA-Z0-9_\-\u4e00-\u9fa5]+/g, '_')

const syncOutputPath = (output: { key: string; path: string }) => {
  output.path = `calc.${normalizedUnitPath()}.${output.key.trim() || 'result'}`
}

const handleOutputKeyInput = (output: { key: string; path: string }) => {
  syncOutputPath(output)
  markDirty()
}

const syncAllOutputPaths = () => {
  activeDraft.value?.outputs.forEach(syncOutputPath)
  markDirty()
}

const addOutput = () => {
  if (!activeDraft.value) return
  const index = activeDraft.value.outputs.length + 1
  const output = {
    key: `output${index}`,
    name: ui(`输出 ${index}`, `Output ${index}`),
    path: '',
    dataType: 'object' as const,
    unit: null,
    precisionNum: null,
    nullPolicy: 'error' as const,
    description: null,
  }
  syncOutputPath(output)
  activeDraft.value.outputs.push(output)
  markDirty()
}

const removeOutput = (index: number) => {
  if (!activeDraft.value || activeDraft.value.outputs.length <= 1) return
  activeDraft.value.outputs.splice(index, 1)
  markDirty()
}

const filteredDependencies = computed(() => {
  const runtime = activeDraft.value?.lang === 'python' ? 'python' : 'javascript'
  return props.dependencies.filter((item) => item.runtime === runtime)
})

const activeTemplate = computed(
  () =>
    codeTemplates.value.find((item) => item.id === activeTemplateId.value) ||
    codeTemplates.value[0],
)

const parameterCount = computed(() => activeDraft.value?.parameterRows.length || 0)
const datapointVariableCount = computed(() => activeDraft.value?.datapointVariableRows.length || 0)
const numericDatapointTypes = new Set([
  'int8',
  'uint8',
  'int16',
  'uint16',
  'int32',
  'uint32',
  'int64',
  'uint64',
  'float32',
  'float64',
  'decimal',
])
const conditionVariableRows = computed(
  () =>
    activeDraft.value?.datapointVariableRows.filter((row) => isTriggerScalarType(row.dataType)) ||
    [],
)
const pointChangeDeadbandAvailable = computed(() => {
  const config = activeDraft.value?.triggerConfig
  if (!config || !numericDatapointTypes.has(String(config.dataType || ''))) return false
  return ['value_change', 'increase', 'decrease'].includes(String(config.mode || ''))
})
const dependencyCount = computed(() => activeDraft.value?.dependencies.length || 0)
const triggerText = computed(() => {
  const triggerType = activeDraft.value?.triggerType || 'manual'
  if (triggerType === 'schedule') {
    const kind = String(activeDraft.value?.triggerConfig.kind || 'interval')
    if (kind === 'daily') return ui('每日定时', 'Daily')
    if (kind === 'weekly') return ui('每周定时', 'Weekly')
    if (kind === 'monthly') return ui('每月定时', 'Monthly')
    if (kind === 'yearly') return ui('每年定时', 'Yearly')
    return ui('周期执行', 'Interval')
  }
  return triggerTypes.value.find((item) => item.id === triggerType)?.label || ui('手动', 'Manual')
})

const triggerSummary = computed(() => {
  if (!activeDraft.value) return ui('未选择计算单元', 'No compute unit selected')
  if (activeDraft.value.triggerType === 'schedule') {
    const config = activeDraft.value.triggerConfig
    if (config.kind === 'daily')
      return ui(`每天 ${config.time || '00:00:00'}（${config.timezone || '未设置时区'}）`, `Every day at ${config.time || '00:00:00'} (${config.timezone || 'time zone not set'})`)
    if (config.kind === 'weekly')
      return ui(`每周指定日期 ${config.time || '00:00:00'}（${config.timezone || '未设置时区'}）`, `Weekly at ${config.time || '00:00:00'} (${config.timezone || 'time zone not set'})`)
    if (config.kind === 'monthly')
      return ui(`每月按日期规则 ${config.time || '00:00:00'}（${config.timezone || '未设置时区'}）`, `Monthly at ${config.time || '00:00:00'} (${config.timezone || 'time zone not set'})`)
    if (config.kind === 'yearly')
      return ui(`每年按日期规则 ${config.time || '00:00:00'}（${config.timezone || '未设置时区'}）`, `Yearly at ${config.time || '00:00:00'} (${config.timezone || 'time zone not set'})`)
    const unit = config.unit === 'hours' ? ui('小时', 'hour(s)') : config.unit === 'minutes' ? ui('分钟', 'minute(s)') : ui('秒', 'second(s)')
    return ui(`每 ${config.every || 1} ${unit}执行一次`, `Run every ${config.every || 1} ${unit}`)
  }
  if (activeDraft.value.triggerType === 'datapoint_change') {
    return activeDraft.value.triggerConfig.path
      ? ui(`数据点变化时执行：${activeDraft.value.triggerConfig.path}`, `Run when this data point changes: ${activeDraft.value.triggerConfig.path}`)
      : ui('选择一个数据点作为变化触发源', 'Select a data point as the change trigger source')
  }
  if (activeDraft.value.triggerType === 'condition') {
    return activeDraft.value.triggerConfig.expression
      ? ui(`条件成立状态变化时执行：${activeDraft.value.triggerConfig.expression}`, `Run when the condition state changes: ${activeDraft.value.triggerConfig.expression}`)
      : ui('使用数据点变量配置条件表达式', 'Build a condition expression from data point variables')
  }
  return ui('由调用方或调试动作主动执行', 'Run explicitly by a caller or debug action')
})

const bottomPanelStyle = computed(() => {
  if (panelCollapsed.value) return {}
  return { height: `${panelHeight.value}px` }
})

const debugStateText = computed(() => {
  if (debugRunning.value) return ui('运行中', 'Running')
  if (debugErrorText.value) return ui('异常', 'Error')
  if (debugResult.value) return ui('完成', 'Completed')
  return ui('未运行', 'Not Run')
})

const dryRunTooltip = computed(() => {
  if (!sandboxAvailable.value) return ui('独立计算沙箱不可用', 'Isolated compute sandbox unavailable')
  if (activeDraft.value?.dirty) return ui('请先保存后再试运行', 'Save before running a dry run')
  if (debugRunning.value) return ui('正在试运行', 'Dry run in progress')
  return ui('试运行', 'Dry Run')
})

const panelSummaryItems = computed(() =>
  panelTabs.value
    .filter((tab) => tab.id !== 'problems')
    .map((tab) => ({
      id: tab.id,
      label: tab.label,
      value: panelSummaryValue(tab.id),
    })),
)

const debugOutputText = computed(() => formatDebugValue(debugResult.value?.output))

const debugLogsText = computed(() => {
  const logs = debugResult.value?.logs || []
  return logs.length ? logs.join('\n') : '-'
})

const debugErrorText = computed(
  () => debugError.value || debugResult.value?.errorMessage || debugResult.value?.error || '',
)

const activeDebugResultText = computed(() => {
  if (activeDebugResultTab.value === 'logs') return debugLogsText.value
  if (activeDebugResultTab.value === 'error') return debugErrorText.value || '-'
  return debugOutputText.value
})

const debugArgvParsed = computed(() => parseJsonForHint(debugArgvText.value, []))
const debugDatapointParsed = computed(() => parseJsonForHint(debugDatapointText.value, {}))

const debugArgvHintTone = computed(() =>
  debugArgvHint.value.includes(ui('不一致', 'does not match')) ? 'warning' : 'muted',
)
const debugDatapointHintTone = computed(() =>
  debugDatapointHint.value.includes(ui('不一致', 'does not match')) ? 'warning' : 'muted',
)

const debugArgvHint = computed(() => {
  const parsed = debugArgvParsed.value
  const expected = activeDraft.value?.parameterRows.length || 0
  if (!parsed.valid) return ui('JSON 格式无效，试运行前需要修正。', 'Invalid JSON. Fix it before the dry run.')
  if (!Array.isArray(parsed.value)) return ui('调用参数 JSON 必须是数组。', 'Arguments JSON must be an array.')
  const actual = parsed.value.length
  if (actual === expected) return ui(`本次试运行传入 ${actual} 个参数。`, `This dry run supplies ${actual} argument${actual === 1 ? '' : 's'}.`)
  return ui(`本次试运行参数数量和定义不一致：定义 ${expected} 个，实际 ${actual} 个。`, `The dry-run argument count does not match the definitions: ${expected} defined, ${actual} supplied.`)
})

const debugDatapointHint = computed(() => {
  const parsed = debugDatapointParsed.value
  const expectedAliases = (activeDraft.value?.datapointVariableRows || [])
    .map((row) => row.alias.trim())
    .filter(Boolean)
  if (!parsed.valid) return ui('JSON 格式无效，试运行前需要修正。', 'Invalid JSON. Fix it before the dry run.')
  if (!isPlainRecord(parsed.value)) return ui('数据点模拟快照 JSON 必须是对象。', 'Data point snapshot JSON must be an object.')
  const keys = Object.keys(parsed.value)
  const invalidKey = keys.find((key) => !isDebugDatapointSnapshot(parsed.value[key]))
  if (invalidKey) return ui(`“${invalidKey}”必须使用包含 value 的快照对象。`, `“${invalidKey}” must be a snapshot object containing value.`)
  const missingCount = expectedAliases.filter((alias) => !(alias in parsed.value)).length
  const extraCount = keys.filter((key) => !expectedAliases.includes(key)).length
  if (!missingCount && !extraCount) return ui(`本次试运行模拟 ${keys.length} 个数据点快照。`, `This dry run simulates ${keys.length} data point snapshot${keys.length === 1 ? '' : 's'}.`)
  return ui(`本次试运行快照和变量定义不一致：缺少 ${missingCount} 个，多出 ${extraCount} 个。`, `The dry-run snapshots do not match the variable definitions: ${missingCount} missing and ${extraCount} extra.`)
})

const syntaxStatusTone = computed(() => {
  if (syntaxStatus.value === 'clean') return 'success'
  if (syntaxDiagnostics.value.length || syntaxStatus.value === 'failed') return 'danger'
  if (syntaxStatus.value === 'checking') return 'checking'
  return 'muted'
})

const syntaxStatusText = computed(() => {
  if (syntaxStatus.value === 'checking') return ui('检查中', 'Checking')
  if (syntaxStatus.value === 'failed') return ui('检查失败', 'Check Failed')
  if (syntaxDiagnostics.value.length) return ui(`${syntaxDiagnostics.value.length} 个语法问题`, `${syntaxDiagnostics.value.length} syntax problem${syntaxDiagnostics.value.length === 1 ? '' : 's'}`)
  if (syntaxStatus.value === 'clean') return ui('语法正常', 'Syntax Valid')
  if (syntaxStatus.value === 'dirty') return ui('待检查', 'Check Pending')
  return ui('未检查', 'Not Checked')
})

watch(codeTemplates, (templates) => {
  if (!templates.some((item) => item.id === activeTemplateId.value)) {
    activeTemplateId.value = templates[0]?.id || 'argv'
  }
})

watch(
  () => props.activeId,
  () => {
    cancelSandboxRequests()
    activePanel.value = 'inputs'
    panelCollapsed.value = true
    panelHeight.value = panelDefaultHeight
    inspectorOpen.value = false
    debugResult.value = null
    debugError.value = ''
    syntaxStatus.value = 'idle'
    syntaxDiagnostics.value = []
    syntaxErrorText.value = ''
    clearSchedulePreview()
    monacoEditorRef.value?.setDiagnostics?.([])
    resetDebugArgvFromDefinition(false)
    resetDebugDatapointsFromDefinition(false)
    scheduleSyntaxCheck()
  },
)

onBeforeUnmount(() => {
  stopPanelResize()
  clearSyntaxCheckTimer()
  cancelSandboxRequests()
})

watch(
  () => [activeDraft.value?.code, activeDraft.value?.lang],
  () => {
    if (!activeDraft.value) return
    syntaxStatus.value = 'dirty'
    scheduleSyntaxCheck()
  },
)

watch(sandboxAvailable, (available, wasAvailable) => {
  if (!available || wasAvailable || !activeDraft.value) return
  syntaxStatus.value = 'dirty'
  syntaxErrorText.value = ''
  scheduleSyntaxCheck()
})

function markDirty() {
  if (activeDraft.value) {
    activeDraft.value.dirty = true
    emit('mark-dirty', activeDraft.value.id)
  }
}

function clearSchedulePreview() {
  schedulePreviewSeq += 1
  schedulePreviewLoading.value = false
  schedulePreviewSummary.value = ''
  schedulePreviewRuns.value = []
  schedulePreviewErrors.value = []
}

function markScheduleDirty() {
  markDirty()
  clearSchedulePreview()
}

function scheduleDateTimeValue(field: 'startAt' | 'endAt') {
  const value = String(activeDraft.value?.triggerConfig[field] || '')
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 19)
}

function updateScheduleDateTime(field: 'startAt' | 'endAt', event: Event) {
  if (!activeDraft.value) return
  const value = (event.target as HTMLInputElement).value
  activeDraft.value.triggerConfig[field] = value ? new Date(value).toISOString() : ''
  markScheduleDirty()
}

function scheduleFieldError(field: string) {
  return schedulePreviewErrors.value.find((item) => item.field.endsWith(field))?.message || ''
}

function clearSyntaxCheckTimer() {
  if (syntaxCheckTimer) {
    window.clearTimeout(syntaxCheckTimer)
    syntaxCheckTimer = null
  }
}

// 标签切换、关闭或组件卸载时中止 HTTP 请求；取消会沿 Go 请求上下文传到沙箱并终止整个进程组。
function cancelSandboxRequests() {
  syntaxCheckSeq += 1
  syntaxCheckController?.abort()
  syntaxCheckController = null
  debugController?.abort()
  debugController = null
  debugRunning.value = false
}

function scheduleSyntaxCheck() {
  clearSyntaxCheckTimer()
  if (!activeDraft.value) return
  syntaxCheckTimer = window.setTimeout(() => {
    void runSyntaxCheck()
  }, 600)
}

async function runSyntaxCheck() {
  if (!sandboxAvailable.value) {
    syntaxStatus.value = 'failed'
    syntaxErrorText.value = ui('独立计算沙箱不可用', 'Isolated compute sandbox unavailable')
    return false
  }
  if (!activeDraft.value) return false
  clearSyntaxCheckTimer()
  const seq = ++syntaxCheckSeq
  syntaxCheckController?.abort()
  const controller = new AbortController()
  syntaxCheckController = controller
  syntaxStatus.value = 'checking'
  syntaxErrorText.value = ''
  try {
    const result = await checkComputeSyntax(
      props.projectId,
      {
        lang: activeDraft.value.lang,
        code: activeDraft.value.code,
      },
      controller.signal,
    )
    if (seq !== syntaxCheckSeq) return false
    syntaxDiagnostics.value = result.diagnostics || []
    syntaxStatus.value = syntaxDiagnostics.value.length ? 'failed' : 'clean'
    monacoEditorRef.value?.setDiagnostics?.(syntaxDiagnostics.value)
    return syntaxDiagnostics.value.length === 0
  } catch (error) {
    if (controller.signal.aborted) return false
    if (seq !== syntaxCheckSeq) return false
    syntaxDiagnostics.value = []
    monacoEditorRef.value?.setDiagnostics?.([])
    syntaxStatus.value = 'failed'
    syntaxErrorText.value = getApiErrorMessage(error, ui('语法检查失败', 'Syntax check failed'))
    return false
  } finally {
    if (syntaxCheckController === controller) syntaxCheckController = null
  }
}

async function ensureSyntaxClean() {
  const ok = await runSyntaxCheck()
  if (!ok) {
    activePanel.value = 'problems'
    panelCollapsed.value = false
    panelHeight.value = Math.max(panelHeight.value, 300)
    ElMessage.warning(syntaxDiagnostics.value.length ? ui('请先修复语法问题', 'Fix the syntax problems first') : syntaxErrorText.value)
  }
  return ok
}

function openSyntaxPanel() {
  activePanel.value = 'problems'
  panelCollapsed.value = false
  panelHeight.value = Math.max(panelHeight.value, 300)
}

function revealDiagnostic(item: ComputeSyntaxDiagnostic) {
  monacoEditorRef.value?.revealPosition?.(item.line, item.column)
}

async function saveAfterSyntaxCheck() {
  if (!activeDraft.value) return
  if (!validateDatapointVariableAliases()) return
  // 沙箱离线只影响开发态语法检查和试运行，不能阻断配置本身的保存。
  if (!sandboxAvailable.value) {
    emit('save', activeDraft.value.id)
    return
  }
  if (!(await ensureSyntaxClean())) return
  emit('save', activeDraft.value.id)
}

function togglePanelCollapsed() {
  panelCollapsed.value = !panelCollapsed.value
  if (panelCollapsed.value) {
    stopPanelResize()
    return
  }
  if (!panelCollapsed.value && panelHeight.value < panelMinHeight) {
    panelHeight.value = panelDefaultHeight
  }
}

function openPanel(panelId: string) {
  activePanel.value = panelId
  panelCollapsed.value = false
  if (panelHeight.value < panelMinHeight) {
    panelHeight.value = panelDefaultHeight
  }
}

function updateCursorInfo(payload: EditorCursorInfo) {
  cursorInfo.value = {
    line: Math.max(1, Number(payload.line) || 1),
    column: Math.max(1, Number(payload.column) || 1),
    spaces: Math.max(1, Number(payload.spaces) || 2),
  }
}

function panelSummaryValue(panelId: string) {
  if (panelId === 'inputs') return String(parameterCount.value)
  if (panelId === 'variables') return String(datapointVariableCount.value)
  if (panelId === 'trigger') return triggerText.value
  if (panelId === 'dependencies') return String(dependencyCount.value)
  if (panelId === 'debug') return debugStateText.value
  return ''
}

function togglePanelMaxHeight() {
  if (panelCollapsed.value) {
    panelCollapsed.value = false
    panelHeight.value = panelDefaultHeight
    return
  }
  panelHeight.value = panelHeight.value >= panelMaxHeight - 20 ? panelDefaultHeight : panelMaxHeight
}

function startPanelResize(event: PointerEvent) {
  if (panelCollapsed.value) return
  event.preventDefault()
  window.addEventListener('pointermove', resizePanel)
  window.addEventListener('pointerup', stopPanelResize, { once: true })
}

function resizePanel(event: PointerEvent) {
  const viewportHeight = window.innerHeight || 900
  const maxHeight = Math.min(panelMaxHeight, Math.floor(viewportHeight * 0.64))
  const nextHeight = viewportHeight - event.clientY
  panelHeight.value = Math.min(maxHeight, Math.max(panelMinHeight, nextHeight))
}

function stopPanelResize() {
  window.removeEventListener('pointermove', resizePanel)
}

function addInput() {
  if (!activeDraft.value) return
  activeDraft.value.parameterRows.push({
    uid: crypto.randomUUID(),
    name: `arg${activeDraft.value.parameterRows.length + 1}`,
    type: 'string',
    required: true,
    defaultValue: '',
    description: '',
  })
  markDirty()
}

function addDatapointVariable(point: DatapointPickerSelection, alias: string) {
  if (!activeDraft.value) return
  activeDraft.value.datapointVariableRows.push({
    uid: crypto.randomUUID(),
    alias,
    path: point.path,
    datapointId: point.id,
    dataType: point.dataType,
  })
  syncDebugDatapointAlias('', alias, point.dataType)
  markDirty()
}

function removeDatapointVariable(index: number) {
  if (!activeDraft.value) return
  const removed = activeDraft.value.datapointVariableRows[index]
  activeDraft.value.datapointVariableRows.splice(index, 1)
  if (removed) syncDebugDatapointAlias(removed.alias, '', removed.dataType)
  markDirty()
}

function renameDatapointVariable(row: ComputeDatapointVariableRow, event: Event) {
  if (!activeDraft.value) return
  const input = event.target as HTMLInputElement
  const previous = row.alias
  const next = input.value.trim()
  if (next === previous) return
  if (!isValidVariableName(next, activeDraft.value.lang)) {
    input.value = previous
    ElMessage.warning(ui('变量名必须是当前脚本语言的合法且非保留名称', 'The variable name must be valid and not reserved in the current language'))
    return
  }
  if (isDatapointAliasUsed(next, row.uid)) {
    input.value = previous
    ElMessage.warning(ui('当前计算单元中已存在同名变量', 'A variable with this name already exists in this compute unit'))
    return
  }
  row.alias = next
  replaceVariableReferences(previous, next)
  syncDebugDatapointAlias(previous, next, row.dataType)
  markDirty()
}

function replaceVariableReferences(previous: string, next: string) {
  if (!activeDraft.value || !previous || previous === next) return
  const pattern = variableReferencePattern(previous)
  activeDraft.value.code = activeDraft.value.code.replace(pattern, next)
  if (typeof activeDraft.value.triggerConfig.expression === 'string') {
    activeDraft.value.triggerConfig.expression = activeDraft.value.triggerConfig.expression.replace(
      pattern,
      next,
    )
  }
}

function syncDebugDatapointAlias(previous: string, next: string, dataType?: string) {
  if (!activeDraft.value) return
  const parsed = parseJsonForHint(debugDatapointText.value, {})
  if (!parsed.valid || !isPlainRecord(parsed.value)) return
  const values = { ...parsed.value }
  const preserved = previous && previous in values ? values[previous] : undefined
  if (previous) delete values[previous]
  if (next && !(next in values)) {
    values[next] = preserved ?? defaultDebugDatapointSnapshot(dataType)
  }
  const synchronized = Object.fromEntries(
    activeDraft.value.datapointVariableRows
      .filter((row) => row.alias.trim())
      .map((row) => {
        const alias = row.alias.trim()
        return [
          alias,
          alias in values ? values[alias] : defaultDebugDatapointSnapshot(row.dataType),
        ]
      }),
  )
  debugDatapointText.value = JSON.stringify(synchronized, null, 2)
}

function removeInput(index: number) {
  if (!activeDraft.value) return
  activeDraft.value.parameterRows.splice(index, 1)
  markDirty()
}

function openDatapointPicker(intent: 'variable' | 'trigger') {
  datapointPickerIntent.value = intent
  datapointPickerVisible.value = true
}

function isTriggerScalarType(dataType?: string) {
  const normalized = String(dataType || '').toLowerCase()
  return normalized === 'bool' || normalized === 'string' || numericDatapointTypes.has(normalized)
}

function datapointTriggerDisabledReason(point: DatapointPickerSelection) {
  if (point.status && point.status !== 'active') return ui('变化触发只能选择状态正常的数据点', 'Change triggers require an active data point')
  if (!isTriggerScalarType(point.dataType)) return ui('变化触发仅支持 bool、string 和数值数据点', 'Change triggers support only bool, string, and numeric data points')
  return ''
}

function isPointChangeModeCompatible(mode: string) {
  const dataType = String(activeDraft.value?.triggerConfig.dataType || '').toLowerCase()
  if (!dataType) return true
  if (mode === 'increase' || mode === 'decrease') return numericDatapointTypes.has(dataType)
  if (mode === 'rising_edge' || mode === 'falling_edge') return dataType === 'bool'
  return isTriggerScalarType(dataType)
}

function handlePointChangeMode() {
  if (!activeDraft.value) return
  if (!pointChangeDeadbandAvailable.value) delete activeDraft.value.triggerConfig.deadband
  markDirty()
}

function appendConditionVariable(alias: string) {
  if (!activeDraft.value) return
  const current = String(activeDraft.value.triggerConfig.expression || '').trimEnd()
  activeDraft.value.triggerConfig.expression = current ? `${current} ${alias}` : alias
  markDirty()
}

function ensureConditionPhase(fallback: 'entered' | 'active' | 'exited') {
  if (!activeDraft.value) return
  const phases = activeDraft.value.triggerConfig.phases
  if (!Array.isArray(phases) || phases.length === 0) {
    activeDraft.value.triggerConfig.phases = [fallback]
    ElMessage.warning(ui('至少保留一个触发阶段', 'Keep at least one trigger phase'))
  }
  markDirty()
}

function confirmSelectedDatapoint(points: DatapointPickerSelection[]) {
  const target = points[0]
  if (!target) return
  if (datapointPickerIntent.value === 'trigger') {
    if (activeDraft.value) {
      activeDraft.value.triggerConfig = {
        ...activeDraft.value.triggerConfig,
        datapointId: target.id,
        path: target.path,
        dataType: target.dataType,
        mode: 'any',
      }
      markDirty()
    }
    datapointPickerVisible.value = false
    return
  }
  if (
    activeDraft.value?.datapointVariableRows.some(
      (row) => row.datapointId === target.id || row.path === target.path,
    )
  ) {
    ElMessage.warning(ui('当前计算单元已经添加该数据点', 'This data point has already been added to the compute unit'))
    return
  }
  const pathName = target.path.split('.').pop() || 'tag'
  const preferredName = target.name || pathName
  const normalizedName = normalizeVariableName(preferredName, activeDraft.value.lang)
  const alias = uniqueDatapointAlias(/^_+$/.test(normalizedName) ? pathName : preferredName)
  if (!isValidVariableName(alias, activeDraft.value.lang)) {
    ElMessage.warning(ui('变量名必须是当前脚本语言的合法变量名', 'The variable name must be valid in the current language'))
    return
  }
  if (isDatapointAliasUsed(alias)) {
    ElMessage.warning(ui('当前计算单元中已存在同名变量', 'A variable with this name already exists in this compute unit'))
    return
  }
  datapointPickerVisible.value = false
  try {
    addDatapointVariable(target, alias)
    insertVariableAlias(alias)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('插入数据点失败', 'Failed to insert data point')))
  }
}

function insertVariableAlias(alias: string) {
  monacoEditorRef.value?.insertText?.(alias)
  markDirty()
}

async function executeDebug() {
  if (!sandboxAvailable.value) {
    debugError.value = ui('独立计算沙箱不可用', 'Isolated compute sandbox unavailable')
    return
  }
  if (!activeDraft.value) return
  if (!validateDatapointVariableAliases()) return
  if (activeDraft.value.dirty) {
    ElMessage.warning(ui('请先保存后再试运行', 'Save before running a dry run'))
    return
  }
  if (!(await ensureSyntaxClean())) return
  let input: Record<string, unknown>
  try {
    input = parseDebugInput()
  } catch (error) {
    debugError.value = error instanceof Error ? error.message : ui('试运行输入 JSON 格式无效', 'Invalid dry-run input JSON')
    return
  }

  debugRunning.value = true
  debugResult.value = null
  debugError.value = ''
  debugController?.abort()
  const controller = new AbortController()
  debugController = controller
  try {
    debugResult.value = await debugComputeUnit(
      props.projectId,
      activeDraft.value.id,
      input,
      true,
      controller.signal,
    )
    activeDebugResultTab.value = debugErrorText.value ? 'error' : 'output'
    ElMessage.success(ui('试运行完成', 'Dry run completed'))
  } catch (error) {
    if (controller.signal.aborted) return
    debugError.value = getApiErrorMessage(error, ui('试运行失败', 'Dry run failed'))
    activeDebugResultTab.value = 'error'
  } finally {
    if (debugController === controller) {
      debugController = null
      debugRunning.value = false
    }
  }
}

async function quickDryRun() {
  activePanel.value = 'debug'
  panelCollapsed.value = false
  panelHeight.value = Math.max(panelHeight.value, 360)
  await executeDebug()
}

function setTriggerType(type: string) {
  if (!activeDraft.value) return
  if (type.startsWith('schedule_')) {
    panelHeight.value = Math.max(panelHeight.value, 340)
  }
  if (type === 'schedule_interval') {
    activeDraft.value.triggerType = 'schedule'
    activeDraft.value.triggerConfig = {
      kind: 'interval',
      every: 1,
      unit: 'minutes',
    }
  } else if (type === 'schedule_daily') {
    activeDraft.value.triggerType = 'schedule'
    activeDraft.value.triggerConfig = {
      kind: 'daily',
      time: '00:00:00',
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai',
    }
  } else if (type === 'schedule_weekly') {
    activeDraft.value.triggerType = 'schedule'
    activeDraft.value.triggerConfig = {
      kind: 'weekly',
      weekdays: [1],
      time: '00:00:00',
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai',
    }
  } else if (type === 'schedule_monthly') {
    activeDraft.value.triggerType = 'schedule'
    activeDraft.value.triggerConfig = {
      kind: 'monthly',
      dayRule: 'day',
      dayOfMonth: 1,
      time: '00:00:00',
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai',
    }
  } else if (type === 'schedule_yearly') {
    activeDraft.value.triggerType = 'schedule'
    activeDraft.value.triggerConfig = {
      kind: 'yearly',
      month: 1,
      dayRule: 'day',
      dayOfMonth: 1,
      time: '00:00:00',
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai',
    }
  } else if (type === 'datapoint_change') {
    activeDraft.value.triggerType = type
    activeDraft.value.triggerConfig = {
      datapointId: '',
      path: '',
      mode: 'any',
      debounceMs: 0,
    }
  } else if (type === 'condition') {
    activeDraft.value.triggerType = type
    activeDraft.value.triggerConfig = {
      expression: '',
      phases: ['entered', 'active', 'exited'],
      debounceMs: 0,
    }
  } else {
    activeDraft.value.triggerType = 'manual'
    activeDraft.value.triggerConfig = {}
  }
  markDirty()
  clearSchedulePreview()
}

async function previewSchedule() {
  if (!activeDraft.value || activeDraft.value.triggerType !== 'schedule') return
  const seq = ++schedulePreviewSeq
  schedulePreviewLoading.value = true
  schedulePreviewSummary.value = ''
  schedulePreviewRuns.value = []
  schedulePreviewErrors.value = []
  try {
    const preview = await previewComputeSchedule(props.projectId, {
      triggerType: activeDraft.value.triggerType,
      triggerConfig: activeDraft.value.triggerConfig,
    })
    if (seq !== schedulePreviewSeq) return
    schedulePreviewSummary.value = preview.summary
    schedulePreviewRuns.value = preview.nextRuns
    schedulePreviewErrors.value = preview.errors
  } catch (error) {
    if (seq !== schedulePreviewSeq) return
    schedulePreviewErrors.value = [
      { field: '', message: getApiErrorMessage(error, ui('定时配置无效', 'Invalid schedule configuration')) },
    ]
  } finally {
    if (seq === schedulePreviewSeq) schedulePreviewLoading.value = false
  }
}

function formatScheduleRun(value: string) {
  const timezone = String(activeDraft.value?.triggerConfig.timezone || '')
  try {
    return new Intl.DateTimeFormat(datacenterLocale.value === 'en' ? 'en-US' : 'zh-CN', {
      dateStyle: 'medium',
      timeStyle: 'medium',
      ...(timezone ? { timeZone: timezone } : {}),
    }).format(new Date(value))
  } catch {
    return new Date(value).toLocaleString()
  }
}

function isDependencyChecked(id: string) {
  return activeDraft.value?.dependencies.some((item) => item.id === id) || false
}

function dependencyName(id: string) {
  return props.dependencies.find((item) => item.id === id)?.name || id
}

function triggerIcon(id: string) {
  if (id.startsWith('schedule_')) return IconTablerClock
  if (id === 'datapoint_change') return IconTablerDatabaseImport
  if (id === 'condition') return IconTablerGitFork
  return IconTablerManualGearbox
}

function triggerDescription(id: string) {
  if (id === 'schedule_interval') return ui('每隔一段时间执行', 'Run at a fixed interval')
  if (id === 'schedule_daily') return ui('每天指定时间执行', 'Run daily at a specified time')
  if (id === 'schedule_weekly') return ui('每周指定日期执行', 'Run on specified weekdays')
  if (id === 'schedule_monthly') return ui('每月按日期规则执行', 'Run monthly using a date rule')
  if (id === 'schedule_yearly') return ui('每年按日期规则执行', 'Run yearly using a date rule')
  if (id === 'datapoint_change') return ui('数据点变化时执行', 'Run when a data point changes')
  if (id === 'condition') return ui('条件状态变化时执行', 'Run when a condition state changes')
  return ui('调用方主动执行', 'Run explicitly by a caller')
}

function isTriggerOptionActive(id: string) {
  if (!activeDraft.value) return false
  if (id.startsWith('schedule_'))
    return (
      activeDraft.value.triggerType === 'schedule' &&
      activeDraft.value.triggerConfig.kind === id.replace('schedule_', '')
    )
  return activeDraft.value.triggerType === id
}

function parseDebugInput(): Record<string, unknown> {
  let argv: unknown
  let datapoints: unknown
  try {
    argv = JSON.parse(debugArgvText.value || '[]')
  } catch {
    throw new Error(ui('argv JSON 格式无效', 'Invalid argv JSON'))
  }
  try {
    datapoints = JSON.parse(debugDatapointText.value || '{}')
  } catch {
    throw new Error(ui('数据点变量模拟 JSON 格式无效', 'Invalid data point snapshot JSON'))
  }
  if (!Array.isArray(argv)) {
    throw new Error(ui('argv JSON 必须是数组', 'argv JSON must be an array'))
  }
  if (!datapoints || typeof datapoints !== 'object' || Array.isArray(datapoints)) {
    throw new Error(ui('数据点模拟快照 JSON 必须是对象', 'Data point snapshot JSON must be an object'))
  }
  for (const [alias, snapshot] of Object.entries(datapoints)) {
    if (!isDebugDatapointSnapshot(snapshot)) {
      throw new Error(ui(`数据点“${alias}”必须使用包含 value 的快照对象`, `Data point “${alias}” must use a snapshot object containing value`))
    }
  }
  return { argv, datapoints }
}

function parseJsonForHint(text: string, fallback: unknown) {
  try {
    return {
      valid: true,
      value: JSON.parse(text || JSON.stringify(fallback)),
    }
  } catch {
    return {
      valid: false,
      value: fallback,
    }
  }
}

function isPlainRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function isDebugDatapointSnapshot(value: unknown) {
  return isPlainRecord(value) && Object.prototype.hasOwnProperty.call(value, 'value')
}

function resetDebugArgvFromDefinition(showMessage = true) {
  debugArgvText.value = buildDefaultDebugArgv()
  if (showMessage) ElMessage.success(ui('已按参数定义重置', 'Reset from parameter definitions'))
}

function resetDebugDatapointsFromDefinition(showMessage = true) {
  debugDatapointText.value = buildDefaultDebugDatapoints()
  if (showMessage) ElMessage.success(ui('已按数据点变量重置', 'Reset from data point variables'))
}

function buildDefaultDebugArgv() {
  const argv = (activeDraft.value?.parameterRows || []).map((row) =>
    defaultValueByType(row.type, row.defaultValue),
  )
  return JSON.stringify(argv, null, 2)
}

function buildDefaultDebugDatapoints() {
  const values = Object.fromEntries(
    (activeDraft.value?.datapointVariableRows || [])
      .filter((row) => row.alias.trim())
      .map((row) => [row.alias.trim(), defaultDebugDatapointSnapshot(row.dataType)]),
  )
  return JSON.stringify(values, null, 2)
}

function defaultDebugDatapointSnapshot(dataType?: string) {
  return {
    value: defaultValueByType(dataType || 'string', ''),
    quality: 'good',
    timestamp: null,
    observedAt: null,
    sourceTimestamp: null,
  }
}

function defaultValueByType(type: string, defaultValue: string) {
  const normalizedType = type.toLowerCase()
  if (defaultValue !== '') {
    if (normalizedType === 'number' || numericDatapointTypes.has(normalizedType))
      return Number(defaultValue)
    if (normalizedType === 'boolean' || normalizedType === 'bool') return defaultValue === 'true'
    if (normalizedType === 'object' || normalizedType === 'array') {
      try {
        return JSON.parse(defaultValue)
      } catch {
        return normalizedType === 'array' ? [] : {}
      }
    }
    return defaultValue
  }
  if (normalizedType === 'number' || numericDatapointTypes.has(normalizedType)) return 0
  if (normalizedType === 'boolean' || normalizedType === 'bool') return false
  if (normalizedType === 'object') return {}
  if (normalizedType === 'array') return []
  return ''
}

function uniqueDatapointAlias(name: string) {
  const base = normalizeVariableName(name, activeDraft.value?.lang)
  const used = new Set((activeDraft.value?.datapointVariableRows || []).map((row) => row.alias))
  if (!used.has(base)) return base
  let index = 2
  while (used.has(`${base}${index}`)) {
    index += 1
  }
  return `${base}${index}`
}

function normalizeVariableName(name: string, lang?: ComputeLang | string) {
  const cleaned = Array.from(name.trim())
    .map((char, index) => (isVariableNameChar(char, index === 0, lang) ? char : '_'))
    .join('')
    .replace(/_+/g, '_')
  const withPrefix = isVariableNameStart(cleaned[0] || '', lang) ? cleaned : `tag_${cleaned}`
  const normalized = withPrefix || 'tag'
  return isValidVariableName(normalized, lang) ? normalized : `tag_${normalized}`
}

function isVariableNameStart(char: string, _lang?: ComputeLang | string) {
  if (!char) return false
  if (char === '_') return true
  return /^[A-Za-z]$/.test(char)
}

function isVariableNameChar(char: string, isStart: boolean, lang?: ComputeLang | string) {
  if (isStart) return isVariableNameStart(char, lang)
  if (char === '_') return true
  return /^[A-Za-z0-9]$/.test(char)
}

const platformReservedVariableNames = new Set(['ctx', 'dp', 'argv', 'console', 'require'])

const jsReservedVariableNames = new Set([
  'arguments',
  'eval',
  'await',
  'break',
  'case',
  'catch',
  'class',
  'const',
  'continue',
  'debugger',
  'default',
  'delete',
  'do',
  'else',
  'enum',
  'export',
  'extends',
  'false',
  'finally',
  'for',
  'function',
  'if',
  'import',
  'in',
  'instanceof',
  'let',
  'new',
  'null',
  'return',
  'super',
  'switch',
  'this',
  'throw',
  'true',
  'try',
  'typeof',
  'var',
  'void',
  'while',
  'with',
  'yield',
])

const pythonReservedVariableNames = new Set([
  'False',
  'None',
  'True',
  'and',
  'as',
  'assert',
  'async',
  'await',
  'break',
  'class',
  'continue',
  'def',
  'del',
  'elif',
  'else',
  'except',
  'finally',
  'for',
  'from',
  'global',
  'if',
  'import',
  'in',
  'is',
  'lambda',
  'nonlocal',
  'not',
  'or',
  'pass',
  'raise',
  'return',
  'try',
  'while',
  'with',
  'yield',
])

function isValidVariableName(name: string, lang?: ComputeLang | string) {
  const chars = Array.from(name)
  const hasValidChars =
    chars.length > 0 && chars.every((char, index) => isVariableNameChar(char, index === 0, lang))
  if (lang === 'python') {
    return (
      hasValidChars &&
      !name.startsWith('__') &&
      !platformReservedVariableNames.has(name) &&
      !pythonReservedVariableNames.has(name)
    )
  }
  return (
    hasValidChars &&
    !name.startsWith('__') &&
    !platformReservedVariableNames.has(name) &&
    !jsReservedVariableNames.has(name)
  )
}

function isDatapointAliasUsed(name: string, excludeUid = '') {
  return Boolean(
    activeDraft.value?.datapointVariableRows.some(
      (row) => row.uid !== excludeUid && row.alias === name,
    ),
  )
}

function validateDatapointVariableAliases() {
  if (!activeDraft.value) return false
  const seen = new Set<string>()
  for (const row of activeDraft.value.datapointVariableRows) {
    const alias = row.alias.trim()
    if (!isValidVariableName(alias, activeDraft.value.lang)) {
      activePanel.value = 'variables'
      panelCollapsed.value = false
      ElMessage.warning(ui(`数据点变量“${alias || row.path}”名称无效`, `Invalid data point variable name: “${alias || row.path}”`))
      return false
    }
    if (seen.has(alias)) {
      activePanel.value = 'variables'
      panelCollapsed.value = false
      ElMessage.warning(ui(`当前计算单元中存在重复变量名“${alias}”`, `Duplicate variable name in this compute unit: “${alias}”`))
      return false
    }
    seen.add(alias)
  }
  return true
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function variableReferencePattern(alias: string) {
  return new RegExp(`(?<![\\p{ID_Continue}$])${escapeRegExp(alias)}(?![\\p{ID_Continue}$])`, 'gu')
}

function insertActiveTemplate() {
  if (!activeTemplate.value || activeTemplate.value.disabled) return
  monacoEditorRef.value?.insertText?.(activeTemplate.value.code)
  templateDialogVisible.value = false
  markDirty()
}

function formatDebugValue(value: unknown) {
  if (value === undefined || value === null) return '-'
  if (typeof value === 'string') return value
  return JSON.stringify(value, null, 2)
}

const langText = (lang?: ComputeLang | string) => {
  const map: Record<string, string> = {
    js: 'JavaScript',
    javascript: 'JavaScript',
    python: 'Python',
  }
  return map[lang || ''] || ui('未知语言', 'Unknown Language')
}

const statusText = (status?: string) => {
  const map: Record<string, string> = {
    enabled: ui('启用', 'Enabled'),
    idle: ui('空闲', 'Idle'),
    running: ui('运行中', 'Running'),
    error: ui('异常', 'Error'),
    disabled: ui('停用', 'Disabled'),
  }
  return map[status || ''] || ui('未知', 'Unknown')
}

const statusTone = (status?: string) => {
  if (status === 'running' || status === 'enabled' || status === 'idle') return 'success'
  if (status === 'error') return 'danger'
  if (status === 'disabled') return 'muted'
  return 'info'
}
</script>

<style scoped>
.compute-editor {
  min-width: 0;
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--dc-surface);
}

.compute-editor__loading {
  padding: 20px;
}

.compute-editor__tabs {
  height: 40px;
  min-height: 40px;
  display: flex;
  align-items: stretch;
  padding: 0;
  background: var(--dc-surface-raised);
  overflow-x: auto;
}

.compute-editor__file-tabs {
  min-width: 0;
  flex: 1;
}

.compute-editor__file-tabs :deep(.el-tabs__header) {
  margin: 0;
  border-bottom-color: var(--dc-border);
}

.compute-editor__file-tabs :deep(.el-tabs__item) {
  min-width: 106px;
  max-width: 220px;
  padding-right: 34px;
  position: relative;
  border-radius: 0;
}

.compute-editor__file-tabs :deep(.el-tabs__nav) {
  border-radius: 0;
}

.compute-editor__file-tabs :deep(.el-tabs__item .is-icon-close) {
  position: absolute;
  top: 50%;
  right: 10px;
  width: 14px;
  height: 14px;
  margin-left: 0;
  transform: translateY(-50%);
}

.compute-editor__file-tabs :deep(.el-tabs__content) {
  display: none;
}

.compute-editor__tab-label {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  overflow: hidden;
}

.compute-editor__tab-label span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__tab-label em {
  color: var(--dc-warning);
  font-style: normal;
}

.compute-editor__toolbar {
  min-height: 44px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface);
}

.compute-editor__sandbox-warning {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 16px;
  color: var(--el-color-warning-dark-2);
  font-size: 12px;
  line-height: 20px;
  background: var(--el-color-warning-light-9);
  border-bottom: 1px solid var(--el-color-warning-light-7);
}

.compute-editor__sandbox-warning span {
  min-width: 0;
}

.compute-editor__sandbox-warning button {
  flex: 0 0 auto;
  height: 26px;
  padding: 0 10px;
  border: 1px solid var(--el-color-warning-light-5);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--el-color-warning-dark-2);
  font-size: 12px;
  font-weight: 700;
}

.compute-editor__sandbox-warning button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.compute-editor__title-wrap {
  min-width: 0;
  display: grid;
  grid-template-columns: auto minmax(150px, 340px) minmax(0, 1fr);
  align-items: center;
  gap: 8px;
}

.compute-editor__meta,
.compute-editor__subline {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.compute-editor__meta > span {
  color: var(--dc-primary);
  font-size: 12px;
  font-weight: 800;
}

.compute-editor__meta em {
  height: 22px;
  display: inline-flex;
  align-items: center;
  border: 1px solid rgba(217, 119, 6, 0.26);
  border-radius: 999px;
  background: rgba(217, 119, 6, 0.08);
  color: #b45309;
  font-size: 12px;
  font-style: normal;
  font-weight: 700;
  padding: 0 8px;
  white-space: nowrap;
}

.compute-editor__name-input {
  width: min(340px, 46vw);
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text);
  font-size: 14px;
  font-weight: 800;
  line-height: 1.3;
}

.compute-editor__name-input:focus {
  border-color: var(--dc-primary);
  background: var(--dc-surface-raised);
  outline: none;
}

.compute-editor__subline span {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text-muted);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__lang-pill {
  padding: 2px 6px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  font-weight: 800;
}

.compute-editor__output-inline {
  font-family: Consolas, 'Courier New', monospace;
}

.compute-editor__actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.compute-editor__icon-action,
.compute-editor__save,
.compute-editor__danger,
.compute-editor__small,
.compute-editor__run-button,
.compute-editor__compact-primary,
.compute-editor__inline-icon,
.compute-editor__tool-btn,
.compute-editor__ghost-icon {
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--dc-radius-sm);
  font-weight: 700;
}

.compute-editor__icon-action {
  width: 30px;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.compute-editor__icon-action.is-active {
  border-color: rgba(29, 78, 216, 0.32);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.compute-editor__icon-action.is-danger {
  color: var(--dc-danger);
}

.compute-editor__save {
  width: 30px;
  border: 1px solid var(--dc-primary);
  background: var(--dc-primary);
  color: var(--dc-on-primary);
}

.compute-editor__danger {
  min-width: 72px;
  border: 1px solid var(--dc-danger);
  background: transparent;
  color: var(--dc-danger);
}

.compute-editor__small {
  min-width: 0;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.compute-editor__run-button {
  gap: 5px;
  min-width: 72px;
  padding: 0 12px;
  border: 1px solid rgba(29, 78, 216, 0.32);
  background: var(--dc-primary);
  color: var(--dc-on-primary);
  font-size: 12px;
}

.compute-editor__run-button.is-danger {
  min-width: 78px;
  border-color: var(--dc-danger);
  background: transparent;
  color: var(--dc-danger);
}

.compute-editor__compact-primary {
  gap: 5px;
  padding: 0 10px;
  border: 1px solid rgba(29, 78, 216, 0.32);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-size: 12px;
}

.compute-editor__inline-icon {
  width: 28px;
  min-width: 28px;
  padding: 0;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.compute-editor__icon-action:disabled,
.compute-editor__save:disabled,
.compute-editor__danger:disabled,
.compute-editor__run-button:disabled,
.compute-editor__tool-btn:disabled {
  cursor: not-allowed;
  opacity: 0.48;
}

.compute-editor__action-icon {
  width: 16px;
  height: 16px;
}

.compute-editor__main {
  min-height: 0;
  flex: 1;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 6px;
  padding: 6px;
  overflow: hidden;
}

.compute-editor__code,
.compute-editor__inspector,
.compute-editor__panel {
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.compute-editor__code {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.compute-editor__code-head {
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 0 8px;
  border-bottom: 1px solid var(--dc-border);
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 800;
}

.compute-editor__code-title,
.compute-editor__inspector-title {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.compute-editor__code-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.compute-editor__syntax-status {
  height: 26px;
  max-width: 128px;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 0 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text-muted);
  font-size: 12px;
  font-weight: 800;
}

.compute-editor__syntax-status span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__syntax-status.is-success {
  border-color: rgba(22, 163, 74, 0.3);
  background: var(--dc-success-soft);
  color: var(--dc-success);
}

.compute-editor__syntax-status.is-danger {
  border-color: rgba(220, 38, 38, 0.32);
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.compute-editor__syntax-status.is-checking .compute-editor__action-icon {
  animation: compute-editor-spin 0.8s linear infinite;
}

@keyframes compute-editor-spin {
  to {
    transform: rotate(360deg);
  }
}

.compute-editor__code-head em {
  color: var(--dc-text-muted);
  font-size: 12px;
  font-style: normal;
  font-weight: 600;
}

.compute-editor__monaco {
  min-height: 0;
  flex: 1;
}

.compute-editor__inspector {
  width: 238px;
  min-width: 0;
  padding: 8px;
}

.compute-editor__inspector h3,
.compute-editor__panel h3 {
  margin: 0;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 800;
}

.compute-editor__inspector-head {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 6px;
  margin-bottom: 8px;
}

.compute-editor__inspector-title {
  grid-column: 1;
}

.compute-editor__ghost-icon,
.compute-editor__tool-btn {
  width: 28px;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface);
  color: var(--dc-text-secondary);
}

.compute-editor__tooltip-wrap {
  display: inline-flex;
  align-items: center;
}

.compute-editor__output-path {
  grid-column: 1 / -1;
  min-width: 0;
  padding: 5px 7px;
  overflow: hidden;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-family: Consolas, 'Courier New', monospace;
  font-size: 11px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__field {
  display: grid;
  gap: 3px;
  margin-bottom: 7px;
}

.compute-editor__field span {
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}

.compute-editor__field input,
.compute-editor__field select,
.compute-editor__mapping-row input,
.compute-editor__mapping-row select {
  height: 26px;
  min-width: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text);
  padding: 0 8px;
  font-size: 12px;
}

.compute-editor__outputs {
  display: grid;
  gap: 8px;
  min-height: 0;
  overflow: auto;
}

.compute-editor__outputs > header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.compute-editor__output-item {
  position: relative;
  display: grid;
  gap: 2px;
  padding: 9px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
}

.compute-editor__output-item > small {
  overflow: hidden;
  color: var(--dc-text-muted);
  font-family: Consolas, monospace;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__remove-output {
  justify-self: end;
  border: 0;
  background: transparent;
  color: var(--el-color-danger);
  cursor: pointer;
}

.compute-editor__remove-output:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.compute-editor__inspector p,
.compute-editor__panel p {
  margin: 0;
  color: var(--dc-text-muted);
  font-size: 13px;
  line-height: 1.6;
}

.compute-editor__bottom {
  position: relative;
  flex: 0 0 auto;
  min-height: 40px;
  display: flex;
  flex-direction: column;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.compute-editor__bottom.is-collapsed {
  height: 40px;
}

.compute-editor__resize-handle {
  height: 8px;
  flex: 0 0 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: ns-resize;
  touch-action: none;
}

.compute-editor__resize-handle span {
  width: 42px;
  height: 3px;
  border-radius: 999px;
  background: var(--dc-border-strong, var(--dc-border));
}

.compute-editor__resize-handle:hover span {
  background: var(--dc-primary);
}

.compute-editor__panel-tabs {
  min-height: 34px;
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 8px 4px;
  background: var(--dc-surface-raised);
}

.compute-editor__bottom.is-collapsed .compute-editor__panel-tabs {
  min-height: 40px;
  padding-top: 4px;
  padding-bottom: 4px;
}

.compute-editor__panel-tablist {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.compute-editor__panel-tab {
  height: 30px;
  min-width: 58px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  padding: 0 8px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.compute-editor__panel-tab.is-active {
  border-color: var(--dc-border);
  background: var(--dc-surface);
  color: var(--dc-primary);
}

.compute-editor__panel-tab-icon,
.compute-editor__head-icon {
  width: 15px;
  height: 15px;
}

.compute-editor__cursor-status {
  min-width: 0;
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 10px;
  overflow: hidden;
  color: var(--dc-text-muted);
  font-family: Consolas, 'Courier New', monospace;
  font-size: 12px;
  white-space: nowrap;
}

.compute-editor__cursor-status span {
  overflow: hidden;
  text-overflow: ellipsis;
}

.compute-editor__panel {
  min-height: 0;
  flex: 1;
  margin: 0 6px 6px;
  padding: 7px 8px;
  overflow: auto;
  box-shadow: none;
}

.compute-editor__panel.is-debug {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.compute-editor__panel-toggle {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
}

.compute-editor__panel-summary {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-left: 4px;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.compute-editor__panel-summary button {
  height: 26px;
  padding: 0 8px;
  border: 1px solid transparent;
  border-radius: 999px;
  background: var(--dc-surface);
  color: var(--dc-text-muted);
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}

.compute-editor__panel-summary button:hover {
  border-color: rgba(29, 78, 216, 0.24);
  color: var(--dc-primary);
  background: var(--dc-primary-soft);
}

.compute-editor__panel-toggle:hover {
  color: var(--dc-primary);
  background: var(--dc-surface);
}

.compute-editor__panel-toggle-icon {
  width: 16px;
  height: 16px;
  transition: transform 0.16s ease;
}

.compute-editor__panel-toggle-icon.is-collapsed {
  transform: rotate(180deg);
}

.compute-editor__panel-head,
.compute-editor__mapping-row,
.compute-editor__mapping-head,
.compute-editor__debug-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.compute-editor__panel-head {
  justify-content: space-between;
  min-height: 28px;
  margin-bottom: 6px;
}

.compute-editor__panel-head > div {
  min-width: 0;
}

.compute-editor__panel-head h3 {
  margin: 0;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 800;
  line-height: 1.3;
}

.compute-editor__help-dot {
  width: 16px;
  height: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 1px solid var(--dc-border);
  border-radius: 999px;
  background: var(--dc-surface);
  color: var(--dc-text-muted);
  font-size: 10px;
  font-weight: 800;
  line-height: 1;
  cursor: help;
}

.compute-editor__help-dot:hover {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.compute-editor__panel-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.compute-editor__mapping-row button {
  height: 24px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text-secondary);
}

.compute-editor__empty {
  padding: 9px 10px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 12px;
}

.compute-editor__empty-action {
  display: grid;
  gap: 5px;
  padding: 10px;
}

.compute-editor__empty-action strong {
  color: var(--dc-text);
  font-size: 12px;
}

.compute-editor__empty-action span {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.compute-editor__empty-action div {
  display: inline-flex;
  gap: 8px;
}

.compute-editor__empty-action button {
  height: 26px;
}

.compute-editor__mapping-row {
  margin-top: 5px;
}

.compute-editor__table-wrap {
  min-width: 0;
  overflow: auto;
}

.compute-editor__mapping-head {
  display: grid;
  grid-template-columns:
    72px minmax(110px, 0.8fr) 96px 38px minmax(100px, 0.8fr) minmax(140px, 1fr)
    28px;
  gap: 5px;
  min-width: 680px;
  margin-top: 3px;
  padding: 0 4px 3px;
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}

.compute-editor__mapping-row {
  display: grid;
  grid-template-columns:
    72px minmax(110px, 0.8fr) 96px 38px minmax(100px, 0.8fr) minmax(140px, 1fr)
    28px;
  gap: 5px;
  min-width: 680px;
  align-items: center;
}

.compute-editor__mapping-row input,
.compute-editor__mapping-row select {
  width: 100%;
}

.compute-editor__row-index {
  height: 26px;
  display: inline-flex;
  align-items: center;
  padding: 0 7px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-family: Consolas, 'Courier New', monospace;
  font-size: 11px;
  font-weight: 800;
}

.compute-editor__row-icon {
  width: 24px;
  min-width: 24px;
  padding: 0;
}

.compute-editor__switch,
.compute-editor__checkbox {
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.compute-editor__switch {
  position: relative;
  width: 30px;
  cursor: pointer;
}

.compute-editor__switch input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.compute-editor__switch span {
  position: relative;
  width: 30px;
  height: 16px;
  border-radius: 999px;
  background: var(--dc-border);
  transition: background 0.16s ease;
}

.compute-editor__switch span::after {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 10px;
  height: 10px;
  border-radius: 999px;
  background: var(--dc-surface-raised);
  content: '';
  transition: transform 0.16s ease;
}

.compute-editor__switch.is-on span {
  background: var(--dc-primary);
}

.compute-editor__switch.is-on span::after {
  transform: translateX(14px);
}

.compute-editor__segmented {
  display: inline-flex;
  gap: 4px;
  padding: 3px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}

.compute-editor__segmented button {
  height: 26px;
  padding: 0 10px;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.compute-editor__segmented button.is-active {
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
}

.compute-editor__form-grid {
  max-width: 360px;
  margin-top: 12px;
}

.compute-editor__panel.is-trigger {
  overflow: hidden;
}

.compute-editor__trigger-layout {
  height: 100%;
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-columns: 210px minmax(0, 1fr);
  gap: 10px;
}

.compute-editor__trigger-rail {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-auto-rows: max-content;
  align-content: start;
  gap: 6px;
  padding-right: 10px;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
  border-right: 1px solid var(--dc-border);
}

.compute-editor__trigger-rail button {
  width: 100%;
  height: auto;
  min-height: 50px;
  min-width: 0;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  gap: 2px 8px;
  padding: 6px 8px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  text-align: left;
}

.compute-editor__trigger-rail button.is-active {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.compute-editor__trigger-rail svg {
  grid-row: 1 / 3;
  align-self: center;
}

.compute-editor__trigger-rail span {
  overflow: hidden;
  font-size: 13px;
  font-weight: 800;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__trigger-rail small {
  overflow: hidden;
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 16px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__trigger-config {
  min-width: 0;
  min-height: 0;
  padding-right: 4px;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
}

.compute-editor__trigger-config > .compute-editor__panel-head {
  position: sticky;
  z-index: 2;
  top: 0;
  padding-bottom: 4px;
  background: var(--dc-surface-raised);
}

.compute-editor__trigger-card {
  min-width: 0;
  max-width: 460px;
  display: flex;
  align-items: flex-end;
  gap: 8px;
  padding: 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
}

.compute-editor__trigger-card.is-note {
  max-width: none;
  align-items: center;
  padding: 7px 9px;
  border-style: dashed;
  background: var(--dc-surface-muted);
}

.compute-editor__trigger-card.is-schedule {
  width: 100%;
  max-width: 760px;
  align-items: stretch;
  flex-direction: column;
}

.compute-editor__schedule-rule,
.compute-editor__schedule-window {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.compute-editor__schedule-rule > .compute-editor__field,
.compute-editor__schedule-window > .compute-editor__field {
  min-width: 0;
  margin: 0;
}

.compute-editor__trigger-card.is-datapoint-change {
  width: 100%;
  max-width: 760px;
  display: grid;
  grid-template-columns: minmax(260px, 1.5fr) minmax(180px, 1fr);
  align-items: end;
}

.compute-editor__datapoint-trigger-source {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: end;
  gap: 8px;
}

.compute-editor__datapoint-trigger-source > .compute-editor__field,
.compute-editor__trigger-card.is-datapoint-change > .compute-editor__field {
  min-width: 0;
  margin: 0;
}

.compute-editor__trigger-card.is-datapoint-change > .compute-editor__trigger-note {
  grid-column: 1 / -1;
}

.compute-editor__trigger-card.is-condition {
  width: 100%;
  max-width: 760px;
  display: grid;
  grid-template-columns: minmax(260px, 1.5fr) minmax(180px, 1fr);
  align-items: end;
}

.compute-editor__condition-expression,
.compute-editor__condition-variables,
.compute-editor__trigger-card.is-condition > .compute-editor__trigger-note {
  grid-column: 1 / -1;
}

.compute-editor__condition-expression textarea {
  width: 100%;
  min-height: 54px;
  box-sizing: border-box;
  padding: 7px 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text);
  font: inherit;
  resize: vertical;
}

.compute-editor__condition-variables {
  display: grid;
  gap: 5px;
}

.compute-editor__condition-variables > span {
  color: var(--dc-text-muted);
  font-size: 12px;
  font-weight: 700;
}

.compute-editor__condition-variables > div {
  display: flex;
  gap: 5px;
  flex-wrap: wrap;
}

.compute-editor__condition-variables button {
  height: 24px;
  padding: 0 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
  font-size: 11px;
  cursor: pointer;
}

.compute-editor__condition-variables small {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.compute-editor__condition-phases {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 0;
  padding: 0;
  border: 0;
}

.compute-editor__condition-phases legend {
  margin-bottom: 5px;
  padding: 0;
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}

.compute-editor__condition-phases label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.compute-editor__condition-phases input {
  width: 14px;
  height: 14px;
  margin: 0;
}

.compute-editor__trigger-card > .compute-editor__field {
  flex: 1;
  margin-bottom: 0;
}

.compute-editor__trigger-card > .compute-editor__inline-icon {
  flex: 0 0 auto;
}

.compute-editor__trigger-card strong,
.compute-editor__trigger-card span {
  font-size: 13px;
}

.compute-editor__field-inline,
.compute-editor__weekday-list,
.compute-editor__trigger-preview {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.compute-editor__field-inline input {
  min-width: 120px;
  flex: 1;
}

.compute-editor__field-inline select {
  width: 96px;
}

.compute-editor__weekday-list label {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.compute-editor__trigger-note {
  margin: 0;
  color: var(--dc-text-muted);
  font-size: 11px;
}

.compute-editor__trigger-preview {
  display: grid;
  grid-template-columns: max-content minmax(0, 1fr);
  align-items: flex-start;
  padding-top: 8px;
  border-top: 1px solid var(--dc-border);
}

.compute-editor__preview-button {
  min-width: 112px;
  height: 30px;
  padding: 0 12px;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 32%, var(--dc-border));
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}

.compute-editor__preview-button:disabled {
  cursor: wait;
  opacity: 0.65;
}

.compute-editor__trigger-preview-result {
  min-width: min(100%, 340px);
  display: grid;
  gap: 5px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.compute-editor__trigger-preview-result ol,
.compute-editor__schedule-errors {
  display: grid;
  gap: 3px;
  margin: 0;
  padding-left: 20px;
}

.compute-editor__schedule-errors,
.compute-editor__field-error {
  color: var(--dc-danger);
  font-size: 12px;
}

@media (max-width: 980px) {
  .compute-editor__trigger-layout {
    grid-template-columns: 180px minmax(0, 1fr);
  }

  .compute-editor__trigger-preview {
    grid-template-columns: 1fr;
  }

  .compute-editor__preview-button {
    width: max-content;
  }

  .compute-editor__schedule-rule,
  .compute-editor__schedule-window,
  .compute-editor__trigger-card.is-datapoint-change,
  .compute-editor__trigger-card.is-condition {
    grid-template-columns: 1fr 1fr;
  }
}

.compute-editor__trigger-card strong {
  color: var(--dc-text);
}

.compute-editor__trigger-card span {
  color: var(--dc-text-muted);
}

.compute-editor__variable-list {
  min-width: 0;
  display: grid;
  gap: 6px;
}

.compute-editor__variable-row {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(140px, 0.8fr) minmax(180px, 1.2fr) 72px 54px 28px;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
}

.compute-editor__variable-alias {
  min-width: 0;
  width: 100%;
  height: 26px;
  box-sizing: border-box;
  display: inline-flex;
  align-items: center;
  padding: 0 8px;
  overflow: hidden;
  border: 1px solid rgba(29, 78, 216, 0.22);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-family: Consolas, 'Courier New', monospace;
  font-size: 12px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__variable-alias:focus {
  border-color: var(--dc-primary);
  outline: 2px solid rgba(29, 78, 216, 0.12);
}

.compute-editor__variable-path {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text-secondary);
  font-family: Consolas, 'Courier New', monospace;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__variable-row em {
  color: var(--dc-text-muted);
  font-size: 12px;
  font-style: normal;
  text-align: right;
}

.compute-editor__variable-state {
  height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 7px;
  border-radius: 999px;
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-size: 11px;
  font-weight: 800;
  white-space: nowrap;
}

.compute-editor__dependency-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
}

.compute-editor__dependency-chips button {
  max-width: 180px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 0 8px;
  overflow: hidden;
  border: 1px solid rgba(29, 78, 216, 0.2);
  border-radius: 999px;
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-size: 12px;
  font-weight: 800;
}

.compute-editor__dependency-chips button span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__chip-icon {
  width: 13px;
  height: 13px;
  flex: 0 0 auto;
  color: var(--dc-text-muted);
}

.compute-editor__dependency-table {
  min-width: 0;
  display: grid;
  gap: 5px;
}

.compute-editor__dependency-row {
  min-width: 0;
  display: grid;
  grid-template-columns: 22px minmax(120px, 0.8fr) minmax(120px, 0.8fr) 90px minmax(180px, 1.4fr);
  align-items: center;
  gap: 8px;
  min-height: 32px;
  padding: 4px 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  cursor: pointer;
}

.compute-editor__dependency-row:hover {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
}

.compute-editor__dependency-row strong,
.compute-editor__dependency-row code,
.compute-editor__dependency-row span,
.compute-editor__dependency-row em {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__dependency-row strong {
  color: var(--dc-text);
  font-size: 13px;
}

.compute-editor__dependency-row code {
  color: var(--dc-primary);
  font-family: Consolas, 'Courier New', monospace;
  font-size: 12px;
}

.compute-editor__dependency-row span,
.compute-editor__dependency-row em {
  color: var(--dc-text-muted);
  font-size: 12px;
  font-style: normal;
}

.compute-editor__debug-layout {
  min-width: 0;
  min-height: 0;
  flex: 1;
  display: grid;
  grid-template-columns: minmax(360px, 0.92fr) minmax(360px, 1fr);
  align-items: stretch;
  gap: 12px;
}

.compute-editor__debug-panel-head {
  margin-bottom: 8px;
}

.compute-editor__debug-run-button {
  height: 28px;
  min-width: 82px;
  padding: 0 12px;
  border-color: transparent;
  border-radius: var(--dc-radius-sm);
  box-shadow: none;
}

.compute-editor__debug-inputs {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.compute-editor__debug-input {
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  display: grid;
  grid-template-rows: 34px minmax(0, 1fr) 22px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
}

.compute-editor__debug-input-head {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 8px 0 10px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.compute-editor__debug-input-head span {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__debug-input-head button {
  height: 24px;
  flex: 0 0 auto;
  padding: 0 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text-secondary);
  font-size: 11px;
  font-weight: 800;
}

.compute-editor__debug-input-head button:hover {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.compute-editor__debug-input textarea {
  min-height: 126px;
  height: 100%;
  resize: none;
  border: 0;
  border-radius: 0;
  background: var(--dc-surface);
  color: var(--dc-text);
  padding: 12px 14px;
  font-family: Consolas, 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.55;
  outline: none;
}

.compute-editor__debug-hint {
  min-height: 22px;
  margin: 0;
  overflow: hidden;
  padding: 0 10px;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-muted);
  font-size: 11px;
  line-height: 22px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__debug-hint.is-warning {
  color: var(--dc-warning);
}

.compute-editor__debug-output {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: 34px minmax(0, 1fr);
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  overflow: hidden;
}

.compute-editor__debug-result-tabs {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 4px 6px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.compute-editor__debug-result-tabs button {
  height: 26px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 800;
}

.compute-editor__debug-result-tabs button.is-active {
  border-color: var(--dc-border);
  background: var(--dc-surface);
  color: var(--dc-primary);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.compute-editor__debug-result-tabs button.has-error {
  color: var(--dc-danger);
}

.compute-editor__debug-output pre {
  min-height: 110px;
  min-width: 0;
  margin: 0;
  overflow: auto;
  padding: 14px 16px;
  color: var(--dc-text);
  font-family: Consolas, 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
}

.compute-editor__problem-list {
  display: grid;
  gap: 5px;
}

.compute-editor__problem-row {
  min-height: 32px;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 5px 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text);
  text-align: left;
}

.compute-editor__problem-row:hover {
  border-color: rgba(220, 38, 38, 0.32);
  background: var(--dc-danger-soft);
}

.compute-editor__problem-row span {
  min-width: 0;
  overflow: hidden;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__problem-row code {
  color: var(--dc-text-muted);
  font-family: Consolas, 'Courier New', monospace;
  font-size: 12px;
}

.compute-editor__problem-icon {
  width: 15px;
  height: 15px;
  color: var(--dc-danger);
}

.compute-editor__picker-confirm {
  border-color: var(--dc-primary);
  background: var(--dc-primary);
  color: var(--dc-on-primary);
}

.compute-editor__template-dialog {
  min-height: 420px;
  display: grid;
  grid-template-columns: 230px minmax(0, 1fr);
  gap: 10px;
}

.compute-editor__template-list {
  min-width: 0;
  display: grid;
  align-content: start;
  gap: 6px;
  padding-right: 10px;
  border-right: 1px solid var(--dc-border);
}

.compute-editor__template-list button {
  min-width: 0;
  display: grid;
  gap: 4px;
  padding: 8px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  text-align: left;
}

.compute-editor__template-list button.is-active {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.compute-editor__template-list button:disabled {
  opacity: 0.48;
  cursor: not-allowed;
}

.compute-editor__template-list strong,
.compute-editor__template-meta strong {
  color: var(--dc-text);
  font-size: 13px;
}

.compute-editor__template-list span,
.compute-editor__template-meta span {
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 1.4;
}

.compute-editor__template-preview {
  min-width: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  gap: 10px;
}

.compute-editor__template-meta {
  display: grid;
  gap: 4px;
}

.compute-editor__template-preview pre {
  min-height: 280px;
  margin: 0;
  overflow: auto;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text);
  font-family: Consolas, 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
}

.compute-editor__template-preview > button {
  justify-self: end;
  height: 30px;
  padding: 0 12px;
  border: 1px solid var(--dc-primary);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary);
  color: var(--dc-on-primary);
  font-size: 12px;
  font-weight: 800;
}

@media (max-width: 1100px) {
  .compute-editor__main {
    grid-template-columns: 1fr;
    overflow-y: auto;
  }

  .compute-editor__debug-layout {
    grid-template-columns: 1fr;
  }

  .compute-editor__debug-inputs {
    grid-template-columns: 1fr;
  }

  .compute-editor__dependency-row {
    grid-template-columns: 22px minmax(120px, 1fr) minmax(120px, 1fr);
  }

  .compute-editor__dependency-row span,
  .compute-editor__dependency-row em {
    display: none;
  }

  .compute-editor__mapping-head,
  .compute-editor__mapping-row {
    grid-template-columns: 1fr;
  }

  .compute-editor__template-dialog {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 800px) {
  .compute-editor__trigger-layout {
    grid-template-columns: 1fr;
    grid-template-rows: minmax(96px, 0.45fr) minmax(0, 1fr);
  }

  .compute-editor__trigger-rail {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    padding-right: 0;
    padding-bottom: 8px;
    border-right: 0;
    border-bottom: 1px solid var(--dc-border);
  }
}
</style>
