<template>
  <section class="compute-editor">
    <div v-if="loading && !activeDraft" class="compute-editor__loading">
      <el-skeleton :rows="10" animated />
    </div>

    <EmptyState
      v-else-if="error && !activeDraft"
      icon-name="warning"
      title="计算单元详情不可用"
      :description="error"
    />

    <EmptyState
      v-else-if="!activeDraft"
      icon-name="compute"
      title="选择一个计算单元"
      description="从左侧资源树选择计算单元，右侧会打开编辑标签。"
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
            <em v-if="activeDraft.dirty">未保存</em>
          </div>
          <input
            v-model="activeDraft.name"
            class="compute-editor__name-input"
            aria-label="计算单元名称"
            @input="markDirty"
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
            :title="activeDraft.isEnabled === false ? '启用' : '停用'"
            :aria-label="activeDraft.isEnabled === false ? '启用' : '停用'"
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
            title="输出配置"
            aria-label="输出配置"
            @click="inspectorOpen = !inspectorOpen"
          >
            <IconTablerSettings class="compute-editor__action-icon" />
          </button>
          <button
            type="button"
            class="compute-editor__icon-action is-danger"
            title="删除"
            aria-label="删除"
            :disabled="deleting"
            @click="$emit('delete-unit', activeDraft.id)"
          >
            <IconTablerTrash class="compute-editor__action-icon" />
          </button>
          <button
            type="button"
            class="compute-editor__save"
            title="保存"
            aria-label="保存"
            :disabled="saving || !activeDraft.dirty"
            @click="saveAfterSyntaxCheck"
          >
            <IconTablerDeviceFloppy class="compute-editor__action-icon" />
          </button>
        </div>
      </div>

      <div v-if="!sandboxAvailable" class="compute-editor__sandbox-warning">
        独立计算沙箱当前不可用，语法检查和开发态试运行已禁用；不会回退到 data_service 宿主执行。
      </div>

      <main class="compute-editor__main">
        <section class="compute-editor__code">
          <div class="compute-editor__code-head">
            <span class="compute-editor__code-title">
              <IconTablerCode class="compute-editor__head-icon" />
              代码
            </span>
            <div class="compute-editor__code-actions">
              <button
                type="button"
                class="compute-editor__tool-btn"
                title="数据点变量"
                aria-label="数据点变量"
                @click="openDatapointPicker('variable')"
              >
                <IconTablerDatabaseImport class="compute-editor__action-icon" />
              </button>
              <button
                type="button"
                class="compute-editor__tool-btn"
                title="代码模板"
                aria-label="代码模板"
                @click="templateDialogVisible = true"
              >
                <IconTablerTemplate class="compute-editor__action-icon" />
              </button>
              <span class="compute-editor__tooltip-wrap" :title="dryRunTooltip">
                <button
                  type="button"
                  class="compute-editor__tool-btn"
                  aria-label="试运行"
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
              <h3>输出</h3>
            </div>
            <button
              type="button"
              class="compute-editor__ghost-icon"
              title="关闭输出配置"
              aria-label="关闭输出配置"
              @click="inspectorOpen = false"
            >
              <IconTablerX class="compute-editor__action-icon" />
            </button>
            <div class="compute-editor__output-path">{{ previewOutputPath }}</div>
          </div>
          <label class="compute-editor__field">
            <span>超时</span>
            <input
              v-model.number="activeDraft.timeoutMs"
              type="number"
              min="1"
              max="120000"
              @input="markDirty"
            />
          </label>
          <label class="compute-editor__field">
            <span>输出名</span>
            <input v-model="outputName" placeholder="result" @input="updateOutputBindings" />
          </label>
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
          title="拖拽调整高度，双击切换最大高度"
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
            <span>行 {{ cursorInfo.line }}，列 {{ cursorInfo.column }}</span>
            <span>空格: {{ cursorInfo.spaces }}</span>
          </div>
          <button
            type="button"
            class="compute-editor__panel-toggle"
            :title="panelCollapsed ? '展开' : '收起'"
            :aria-label="panelCollapsed ? '展开底部面板' : '收起底部面板'"
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
                  <span>脚本参数</span>
                  <button
                    type="button"
                    class="compute-editor__help-dot"
                    title="调用方传入的参数，脚本内按 argv[0]、argv[1] 顺序读取。"
                    aria-label="脚本参数说明"
                  >
                    ?
                  </button>
                </h3>
              </div>
              <button
                type="button"
                class="compute-editor__tool-btn"
                title="添加参数"
                aria-label="添加参数"
                @click="addInput"
              >
                <IconTablerPlus class="compute-editor__action-icon" />
              </button>
            </div>
            <div
              v-if="!activeDraft.parameterRows.length"
              class="compute-editor__empty compute-editor__empty-action"
            >
              <strong>还没有脚本参数</strong>
              <span>脚本参数由调用方传入，脚本内通过 argv[0]、argv[1] 读取。</span>
              <div>
                <button type="button" class="compute-editor__compact-primary" @click="addInput">
                  <IconTablerPlus class="compute-editor__action-icon" />
                  <span>添加参数</span>
                </button>
              </div>
            </div>
            <div v-else class="compute-editor__table-wrap">
              <div class="compute-editor__mapping-head">
                <span>序号</span>
                <span>参数名</span>
                <span>类型</span>
                <span>必填</span>
                <span>默认值</span>
                <span>说明</span>
                <span></span>
              </div>
              <div
                v-for="(row, index) in activeDraft.parameterRows"
                :key="row.uid"
                class="compute-editor__mapping-row"
              >
                <span class="compute-editor__row-index">argv[{{ index }}]</span>
                <input v-model="row.name" placeholder="参数名" @input="markDirty" />
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
                <input v-model="row.description" placeholder="调用方传参说明" @input="markDirty" />
                <button
                  type="button"
                  class="compute-editor__row-icon"
                  title="删除参数"
                  aria-label="删除参数"
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
                  <span>数据点变量</span>
                  <button
                    type="button"
                    class="compute-editor__help-dot"
                    title="插入到脚本中的数据点变量，脚本内可直接按变量名读取。"
                    aria-label="数据点变量说明"
                  >
                    ?
                  </button>
                </h3>
              </div>
              <button
                type="button"
                class="compute-editor__tool-btn"
                title="插入数据点变量"
                aria-label="插入数据点变量"
                @click="openDatapointPicker('variable')"
              >
                <IconTablerDatabaseImport class="compute-editor__action-icon" />
              </button>
              <button
                v-if="unusedDatapointVariableCount"
                type="button"
                class="compute-editor__tool-btn"
                title="清理未引用变量"
                aria-label="清理未引用变量"
                @click="removeUnusedDatapointVariables"
              >
                <IconTablerTrash class="compute-editor__action-icon" />
              </button>
            </div>
            <div
              v-if="!activeDraft.datapointVariableRows.length"
              class="compute-editor__empty compute-editor__empty-action"
            >
              <strong>还没有数据点变量</strong>
              <span>从数据点列表选择后，会在这里管理变量名、路径和类型。</span>
              <div>
                <button
                  type="button"
                  class="compute-editor__compact-primary"
                  @click="openDatapointPicker('variable')"
                >
                  <IconTablerDatabaseImport class="compute-editor__action-icon" />
                  <span>插入数据点变量</span>
                </button>
              </div>
            </div>
            <div v-else class="compute-editor__variable-list">
              <div
                v-for="(row, index) in activeDraft.datapointVariableRows"
                :key="row.uid"
                class="compute-editor__variable-row"
                :class="{ 'is-unused': !isDatapointVariableReferenced(row.alias) }"
              >
                <button
                  type="button"
                  class="compute-editor__variable-alias"
                  :title="`插入变量：${row.alias}`"
                  @click="insertVariableAlias(row.alias)"
                >
                  {{ row.alias }}
                </button>
                <span class="compute-editor__variable-path" :title="row.path">
                  {{ row.path }}
                </span>
                <em>{{ row.dataType || '-' }}</em>
                <span
                  class="compute-editor__variable-state"
                  :class="{ 'is-unused': !isDatapointVariableReferenced(row.alias) }"
                >
                  {{ isDatapointVariableReferenced(row.alias) ? '已引用' : '未引用' }}
                </span>
                <button
                  type="button"
                  class="compute-editor__row-icon"
                  title="删除变量"
                  aria-label="删除变量"
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
                        aria-label="触发说明"
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
                    <span v-if="activeDraft.triggerConfig.kind === 'interval'">周期</span>
                    <span v-else>执行时间</span>
                    <div
                      v-if="activeDraft.triggerConfig.kind === 'interval'"
                      class="compute-editor__field-inline"
                    >
                      <input
                        v-model.number="activeDraft.triggerConfig.every"
                        type="number"
                        min="1"
                        @input="markDirty"
                      />
                      <select v-model="activeDraft.triggerConfig.unit" @change="markDirty">
                        <option value="seconds">秒</option>
                        <option value="minutes">分钟</option>
                        <option value="hours">小时</option>
                      </select>
                    </div>
                    <input
                      v-else
                      v-model="activeDraft.triggerConfig.time"
                      type="time"
                      step="1"
                      @input="markDirty"
                    />
                  </label>
                  <label
                    v-if="activeDraft.triggerConfig.kind === 'weekly'"
                    class="compute-editor__field"
                  >
                    <span>执行星期</span>
                    <div class="compute-editor__weekday-list">
                      <label v-for="day in weekdayOptions" :key="day.value">
                        <input
                          v-model="activeDraft.triggerConfig.weekdays"
                          type="checkbox"
                          :value="day.value"
                          @change="markDirty"
                        />
                        {{ day.label }}
                      </label>
                    </div>
                  </label>
                  <label
                    v-if="activeDraft.triggerConfig.kind !== 'interval'"
                    class="compute-editor__field"
                  >
                    <span>时区</span>
                    <input
                      v-model="activeDraft.triggerConfig.timezone"
                      placeholder="Asia/Shanghai"
                      @input="markDirty"
                    />
                  </label>
                  <p class="compute-editor__trigger-note">
                    该计划由节点运行时执行；开发态不会自动运行。
                  </p>
                  <div class="compute-editor__trigger-preview">
                    <button
                      type="button"
                      class="compute-editor__inline-icon"
                      title="预览后续执行"
                      @click="previewSchedule"
                    >
                      预览后续执行
                    </button>
                    <span v-if="schedulePreviewText">{{ schedulePreviewText }}</span>
                  </div>
                </div>
                <div
                  v-else-if="activeDraft.triggerType === 'datapoint_change'"
                  class="compute-editor__trigger-card"
                >
                  <label class="compute-editor__field">
                    <span>数据点</span>
                    <input
                      v-model="activeDraft.triggerConfig.path"
                      placeholder="选择数据点"
                      readonly
                    />
                  </label>
                  <button
                    type="button"
                    class="compute-editor__inline-icon"
                    title="选择数据点"
                    aria-label="选择数据点"
                    @click="openDatapointPicker('trigger')"
                  >
                    <IconTablerDatabaseImport class="compute-editor__action-icon" />
                  </button>
                </div>
                <div v-else class="compute-editor__trigger-card is-note">
                  <strong>手动触发</strong>
                  <span>脚本只在调用方调用、调试或手动运行时执行。</span>
                </div>
              </section>
            </div>
          </template>

          <template v-else-if="activePanel === 'dependencies'">
            <div class="compute-editor__panel-head">
              <div>
                <h3>
                  <span>依赖库</span>
                  <button
                    type="button"
                    class="compute-editor__help-dot"
                    :title="`当前语言：${langText(activeDraft.lang)}，已选 ${dependencyCount} 个。`"
                    aria-label="依赖库说明"
                  >
                    ?
                  </button>
                </h3>
              </div>
              <button
                type="button"
                class="compute-editor__tool-btn"
                title="刷新依赖"
                aria-label="刷新依赖"
                @click="$emit('refresh-dependencies')"
              >
                <IconTablerRefresh class="compute-editor__action-icon" />
              </button>
            </div>
            <div v-if="activeDraft.dependencies.length" class="compute-editor__dependency-chips">
              <button
                v-for="item in activeDraft.dependencies"
                :key="item.id"
                type="button"
                :title="`取消依赖：${dependencyName(item.id)}`"
                @click="toggleDependency(item.id)"
              >
                <span>{{ dependencyName(item.id) }}</span>
                <IconTablerX class="compute-editor__chip-icon" />
              </button>
            </div>
            <div v-if="dependenciesError" class="compute-editor__empty">
              {{ dependenciesError }}
            </div>
            <div v-else-if="!filteredDependencies.length" class="compute-editor__empty">
              当前语言无可用依赖
            </div>
            <div v-else class="compute-editor__dependency-table">
              <label
                v-for="dep in filteredDependencies"
                :key="dep.id"
                class="compute-editor__dependency-row"
              >
                <input
                  type="checkbox"
                  :checked="isDependencyChecked(dep.id)"
                  @change="toggleDependency(dep.id)"
                />
                <strong>{{ dep.name }}</strong>
                <code>{{ dep.importName || dep.name }}</code>
                <span>{{ dep.version || '-' }}</span>
                <em>{{ dep.description || '内置依赖' }}</em>
              </label>
            </div>
          </template>

          <template v-else-if="activePanel === 'debug'">
            <div class="compute-editor__panel-head compute-editor__debug-panel-head">
              <div>
                <h3>
                  <span>调试</span>
                  <button
                    type="button"
                    class="compute-editor__help-dot"
                    :title="
                      activeDraft.dirty
                        ? '保存后才能试运行。'
                        : '试运行只检查脚本返回值，不会保存结果或写入数据点。'
                    "
                    aria-label="调试说明"
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
                    试运行
                  </button>
                </span>
              </div>
            </div>
            <div class="compute-editor__debug-layout">
              <div class="compute-editor__debug-inputs">
                <label class="compute-editor__debug-input">
                  <div class="compute-editor__debug-input-head">
                    <span>调用参数 JSON</span>
                    <button
                      type="button"
                      title="按参数定义重新生成"
                      @click="resetDebugArgvFromDefinition"
                    >
                      按定义重置
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
                    <span>变量模拟值 JSON</span>
                    <button
                      type="button"
                      title="按数据点变量重新生成"
                      @click="resetDebugDatapointsFromDefinition"
                    >
                      按定义重置
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
                  <span>问题</span>
                  <button
                    type="button"
                    class="compute-editor__help-dot"
                    :title="syntaxStatusText"
                    aria-label="问题说明"
                  >
                    ?
                  </button>
                </h3>
              </div>
              <button
                type="button"
                class="compute-editor__tool-btn"
                title="重新检查"
                aria-label="重新检查"
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
              没有语法问题
            </div>
            <div v-else-if="syntaxStatus === 'failed'" class="compute-editor__empty">
              {{ syntaxErrorText || '语法检查失败' }}
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

    <DcDialog
      v-model="datapointPickerVisible"
      title="数据点变量"
      width="860px"
      body-max-height="620px"
    >
      <div class="compute-editor__picker">
        <div class="compute-editor__picker-toolbar">
          <el-input
            v-model="datapointPickerKeyword"
            size="small"
            clearable
            placeholder="搜索名称或路径"
            @keyup.enter="reloadPickerDatapoints"
          />
          <select
            v-model="datapointPickerSource"
            class="compute-editor__picker-select"
            aria-label="来源类型"
            @change="reloadPickerDatapoints"
          >
            <option value="">全部来源</option>
            <option value="mqtt.subscription">MQTT</option>
            <option value="db.query">数据库</option>
            <option value="http">HTTP</option>
            <option value="manual">手动</option>
          </select>
          <select
            v-model="datapointPickerDataType"
            class="compute-editor__picker-select"
            aria-label="数据类型"
            @change="reloadPickerDatapoints"
          >
            <option value="">全部类型</option>
            <option value="object">object</option>
            <option value="number">number</option>
            <option value="string">string</option>
            <option value="boolean">boolean</option>
          </select>
          <select
            v-model="datapointPickerStatus"
            class="compute-editor__picker-select"
            aria-label="状态"
            @change="reloadPickerDatapoints"
          >
            <option value="">全部状态</option>
            <option value="active">正常</option>
            <option value="inactive">停用</option>
            <option value="error">异常</option>
            <option value="unknown">未知</option>
          </select>
          <button type="button" class="compute-editor__small" @click="reloadPickerDatapoints">
            搜索
          </button>
        </div>
        <div v-if="datapointPickerLoading" class="compute-editor__picker-loading">
          <el-skeleton :rows="5" animated />
        </div>
        <div v-else class="compute-editor__picker-table-wrap">
          <table v-if="datapointPickerOptions.length" class="compute-editor__picker-table">
            <thead>
              <tr>
                <th>名称</th>
                <th>路径</th>
                <th>类型</th>
                <th>来源</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="point in datapointPickerOptions"
                :key="point.id"
                :class="{ 'is-selected': selectedDatapointId === String(point.id) }"
                @click="selectedDatapointId = String(point.id)"
                @dblclick="confirmSelectedDatapoint(point)"
              >
                <td>
                  <strong>{{ point.name }}</strong>
                </td>
                <td>
                  <code>{{ point.path }}</code>
                </td>
                <td>{{ point.dataType || '-' }}</td>
                <td>{{ sourceTypeText(point.sourceType) }}</td>
                <td>
                  <span
                    class="compute-editor__picker-status"
                    :class="`is-${point.status || 'unknown'}`"
                  >
                    {{ datapointStatusText(point.status) }}
                  </span>
                </td>
                <td>
                  <span
                    class="compute-editor__picker-row-action"
                    role="button"
                    tabindex="0"
                    @click.stop="confirmSelectedDatapoint(point)"
                    @keydown.enter.stop.prevent="confirmSelectedDatapoint(point)"
                  >
                    {{ datapointPickerIntent === 'trigger' ? '选择' : '插入' }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
          <div
            v-if="datapointPickerOptions.length === 0"
            class="compute-editor__empty compute-editor__picker-empty"
          >
            <strong>没有匹配的数据点</strong>
            <span>换个关键词，或放宽来源、类型、状态筛选。</span>
          </div>
        </div>
        <div class="compute-editor__picker-footer">
          <div class="compute-editor__picker-count">
            共 {{ datapointPickerTotal }} 条
            <span v-if="selectedDatapoint">已选 {{ selectedDatapoint.name }}</span>
          </div>
          <label
            v-if="selectedDatapoint && datapointPickerIntent === 'variable'"
            class="compute-editor__alias-field"
          >
            <span>变量名</span>
            <input v-model="selectedDatapointAlias" />
          </label>
          <div class="compute-editor__picker-pager">
            <button
              type="button"
              :disabled="datapointPickerPage <= 1 || datapointPickerLoading"
              @click="changePickerPage(datapointPickerPage - 1)"
            >
              上一页
            </button>
            <span>{{ datapointPickerPage }} / {{ datapointPickerTotalPages }}</span>
            <button
              type="button"
              :disabled="datapointPickerPage >= datapointPickerTotalPages || datapointPickerLoading"
              @click="changePickerPage(datapointPickerPage + 1)"
            >
              下一页
            </button>
            <button
              type="button"
              class="compute-editor__picker-confirm"
              :disabled="!selectedDatapoint"
              @click="confirmSelectedDatapoint()"
              @mousedown.prevent="confirmSelectedDatapoint()"
            >
              {{ datapointPickerIntent === 'trigger' ? '选择' : '插入到光标' }}
            </button>
          </div>
        </div>
      </div>
    </DcDialog>

    <DcDialog
      v-model="templateDialogVisible"
      title="代码模板"
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
            @click="insertActiveTemplate"
          >
            插入到光标
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
import type { Datapoint } from '@/api/schemas/datapoint.schema'
import type {
  ComputeDependency,
  ComputeCapabilities,
  ComputeLang,
  ComputeRunResult,
  ComputeSyntaxDiagnostic,
} from '@/api/schemas/compute.schema'
import { checkComputeSyntax, debugComputeUnit, previewComputeSchedule } from '@/api/compute.api'
import { getDatapoints } from '@/api/datapoint.api'
import { getApiErrorMessage } from '@/utils/request'
import MonacoEditor from '@/components/MonacoEditor.vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import type { ComputeDraft, ComputeEditorTab } from './computeEditorModel'

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
    activeDraft: ComputeDraft | null
    loading?: boolean
    saving?: boolean
    deleting?: boolean
    error?: string
    dependencies?: ComputeDependency[]
    dependenciesLoading?: boolean
    dependenciesError?: string
    capabilities?: ComputeCapabilities | null
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
  },
)

const emit = defineEmits<{
  (event: 'activate-tab', id: string): void
  (event: 'close-tab', id: string): void
  (event: 'save', id: string): void
  (event: 'toggle-enabled', id: string, enabled: boolean): void
  (event: 'delete-unit', id: string): void
  (event: 'mark-dirty', id: string): void
  (event: 'refresh-dependencies'): void
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
const datapointPickerKeyword = ref('')
const datapointPickerSource = ref('')
const datapointPickerDataType = ref('')
const datapointPickerStatus = ref('')
const datapointPickerPage = ref(1)
const datapointPickerPageSize = 12
const datapointPickerTotal = ref(0)
const datapointPickerOptions = ref<Datapoint[]>([])
const datapointPickerLoading = ref(false)
const selectedDatapointId = ref<string | null>(null)
const selectedDatapointAlias = ref('')
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
const templateDialogVisible = ref(false)
const activeTemplateId = ref('argv')
const schedulePreviewText = ref('')
const activeDebugResultTab = ref('output')
const cursorInfo = ref<EditorCursorInfo>({
  line: 1,
  column: 1,
  spaces: 2,
})

const panelTabs = [
  { id: 'inputs', label: '参数', icon: IconTablerPlugConnected },
  { id: 'variables', label: '变量', icon: IconTablerDatabaseImport },
  { id: 'trigger', label: '触发', icon: IconTablerPlayerPlay },
  { id: 'dependencies', label: '依赖', icon: IconTablerGitFork },
  { id: 'debug', label: '调试', icon: IconTablerTerminal2 },
  { id: 'problems', label: '问题', icon: IconTablerAlertTriangle },
]

const debugResultTabs = [
  { id: 'output', label: '返回值' },
  { id: 'logs', label: '日志' },
  { id: 'error', label: '错误' },
]

const sandboxAvailable = computed(() => props.capabilities?.sandboxStatus === 'available')

const codeTemplates = computed(() => {
  const lang = props.activeDraft?.lang === 'python' ? 'python' : 'javascript'
  const templates = lang === 'python' ? pythonTemplates : javascriptTemplates
  return templates
})

const triggerTypes = [
  { id: 'manual', label: '手动' },
  { id: 'schedule_interval', label: '周期执行' },
  { id: 'schedule_daily', label: '每日定时' },
  { id: 'schedule_weekly', label: '每周定时' },
  { id: 'datapoint_change', label: '数据点变化' },
]

const weekdayOptions = [
  { value: 1, label: '一' },
  { value: 2, label: '二' },
  { value: 3, label: '三' },
  { value: 4, label: '四' },
  { value: 5, label: '五' },
  { value: 6, label: '六' },
  { value: 7, label: '日' },
]

type CodeTemplate = {
  id: string
  name: string
  description: string
  code: string
}

const javascriptTemplates: CodeTemplate[] = [
  {
    id: 'argv',
    name: '获取脚本参数',
    description: '读取调用方传入的 argv 参数。',
    code: 'const firstArg = argv[0];\nconst secondArg = argv[1];\n',
  },
  {
    id: 'function',
    name: '定义函数',
    description: '封装一段可复用计算逻辑。',
    code: 'function calculate(value) {\n  return value;\n}\n',
  },
  {
    id: 'class',
    name: '定义类',
    description: '组织复杂对象或业务模型。',
    code: 'class Model {\n  constructor(value) {\n    this.value = value;\n  }\n}\n',
  },
  {
    id: 'datapoint',
    name: '读取数据点变量',
    description: '读取已插入的数据点变量和 dp 命名空间。',
    code: 'const value = dp.tag1;\nreturn value;\n',
  },
  {
    id: 'output',
    name: '输出结果',
    description: '把脚本结果返回给调用方和输出数据点。',
    code: 'return {\n  value: null,\n  updatedAt: new Date().toISOString()\n};\n',
  },
  {
    id: 'try-catch',
    name: '错误处理',
    description: '捕获异常并返回明确错误。',
    code: 'try {\n  return null;\n} catch (error) {\n  return { error: String(error && error.message ? error.message : error) };\n}\n',
  },
]

const pythonTemplates: CodeTemplate[] = [
  {
    id: 'argv',
    name: '获取脚本参数',
    description: '读取调用方传入的 argv 参数。',
    code: 'def main(argv, dp, ctx):\n    first_arg = argv[0] if len(argv) > 0 else None\n    second_arg = argv[1] if len(argv) > 1 else None\n    return first_arg\n',
  },
  {
    id: 'function',
    name: '定义函数',
    description: '封装一段可复用计算逻辑。',
    code: 'def calculate(value):\n    return value\n\ndef main(argv, dp, ctx):\n    return calculate(argv[0] if len(argv) > 0 else None)\n',
  },
  {
    id: 'class',
    name: '定义类',
    description: '组织复杂对象或业务模型。',
    code: 'class Model:\n    def __init__(self, value):\n        self.value = value\n',
  },
  {
    id: 'datapoint',
    name: '读取数据点变量',
    description: '读取已插入的数据点变量和 dp 命名空间。',
    code: 'def main(argv, dp, ctx):\n    value = dp.get("tag1")\n    return value\n',
  },
  {
    id: 'output',
    name: '输出结果',
    description: '把脚本结果返回给调用方和输出数据点。',
    code: 'def main(argv, dp, ctx):\n    return {\n        "value": None\n    }\n',
  },
  {
    id: 'try-catch',
    name: '错误处理',
    description: '捕获异常并返回明确错误。',
    code: 'def main(argv, dp, ctx):\n    try:\n        return None\n    except Exception as error:\n        return {"error": str(error)}\n',
  },
]

const monacoLanguage = computed(() => {
  if (props.activeDraft?.lang === 'python') return 'python'
  return 'javascript'
})

const previewOutputPath = computed(() => {
  const name = props.activeDraft?.name?.trim() || 'unnamed'
  return `calc.${name.replace(/\s+/g, '_')}.${outputName.value || 'result'}`
})

const outputName = computed({
  get() {
    const outputs = props.activeDraft?.outputBindings.outputs
    if (Array.isArray(outputs) && outputs[0]) {
      const first = outputs[0] as Record<string, unknown>
      return String(first.name || 'result')
    }
    return 'result'
  },
  set(value: string) {
    if (!props.activeDraft) return
    props.activeDraft.outputBindings = {
      ...props.activeDraft.outputBindings,
      outputs: [{ name: value || 'result', dataType: 'object' }],
    }
  },
})

const filteredDependencies = computed(() => {
  const runtime = props.activeDraft?.lang === 'python' ? 'python' : 'javascript'
  return props.dependencies.filter((item) => item.runtime === runtime)
})

const activeTemplate = computed(
  () =>
    codeTemplates.value.find((item) => item.id === activeTemplateId.value) ||
    codeTemplates.value[0],
)

const parameterCount = computed(() => props.activeDraft?.parameterRows.length || 0)
const datapointVariableCount = computed(() => props.activeDraft?.datapointVariableRows.length || 0)
const unusedDatapointVariableCount = computed(
  () =>
    props.activeDraft?.datapointVariableRows.filter(
      (row) => !isDatapointVariableReferenced(row.alias),
    ).length || 0,
)
const dependencyCount = computed(() => props.activeDraft?.dependencies.length || 0)
const triggerText = computed(() => {
  const triggerType = props.activeDraft?.triggerType || 'manual'
  if (triggerType === 'schedule') {
    const kind = String(props.activeDraft?.triggerConfig.kind || 'interval')
    return kind === 'daily' ? '每日定时' : kind === 'weekly' ? '每周定时' : '周期执行'
  }
  return triggerTypes.find((item) => item.id === triggerType)?.label || '手动'
})

const triggerSummary = computed(() => {
  if (!props.activeDraft) return '未选择计算单元'
  if (props.activeDraft.triggerType === 'schedule') {
    const config = props.activeDraft.triggerConfig
    if (config.kind === 'daily')
      return `每天 ${config.time || '00:00:00'}（${config.timezone || '未设置时区'}）`
    if (config.kind === 'weekly')
      return `每周指定日期 ${config.time || '00:00:00'}（${config.timezone || '未设置时区'}）`
    return `每 ${config.every || 1} ${config.unit === 'hours' ? '小时' : config.unit === 'minutes' ? '分钟' : '秒'}执行一次`
  }
  if (props.activeDraft.triggerType === 'datapoint_change') {
    return props.activeDraft.triggerConfig.path
      ? `数据点变化时执行：${props.activeDraft.triggerConfig.path}`
      : '选择一个数据点作为变化触发源'
  }
  return '由调用方或调试动作主动执行'
})

const bottomPanelStyle = computed(() => {
  if (panelCollapsed.value) return {}
  return { height: `${panelHeight.value}px` }
})

const debugStateText = computed(() => {
  if (debugRunning.value) return '运行中'
  if (debugErrorText.value) return '异常'
  if (debugResult.value) return '完成'
  return '未运行'
})

const dryRunTooltip = computed(() => {
  if (!sandboxAvailable.value) return '独立计算沙箱不可用'
  if (props.activeDraft?.dirty) return '请先保存后再试运行'
  if (debugRunning.value) return '正在试运行'
  return '试运行'
})

const panelSummaryItems = computed(() =>
  panelTabs
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
  debugArgvHint.value.includes('不一致') ? 'warning' : 'muted',
)
const debugDatapointHintTone = computed(() =>
  debugDatapointHint.value.includes('不一致') ? 'warning' : 'muted',
)

const debugArgvHint = computed(() => {
  const parsed = debugArgvParsed.value
  const expected = props.activeDraft?.parameterRows.length || 0
  if (!parsed.valid) return 'JSON 格式无效，试运行前需要修正。'
  if (!Array.isArray(parsed.value)) return '调用参数 JSON 必须是数组。'
  const actual = parsed.value.length
  if (actual === expected) return `本次试运行传入 ${actual} 个参数。`
  return `本次试运行参数数量和定义不一致：定义 ${expected} 个，实际 ${actual} 个。`
})

const debugDatapointHint = computed(() => {
  const parsed = debugDatapointParsed.value
  const expectedAliases = (props.activeDraft?.datapointVariableRows || [])
    .map((row) => row.alias.trim())
    .filter(Boolean)
  if (!parsed.valid) return 'JSON 格式无效，试运行前需要修正。'
  if (!isPlainRecord(parsed.value)) return '变量模拟值 JSON 必须是对象。'
  const keys = Object.keys(parsed.value)
  const missingCount = expectedAliases.filter((alias) => !(alias in parsed.value)).length
  const extraCount = keys.filter((key) => !expectedAliases.includes(key)).length
  if (!missingCount && !extraCount) return `本次试运行模拟 ${keys.length} 个变量。`
  return `本次试运行变量和定义不一致：缺少 ${missingCount} 个，多出 ${extraCount} 个。`
})

const syntaxStatusTone = computed(() => {
  if (syntaxStatus.value === 'clean') return 'success'
  if (syntaxDiagnostics.value.length || syntaxStatus.value === 'failed') return 'danger'
  if (syntaxStatus.value === 'checking') return 'checking'
  return 'muted'
})

const syntaxStatusText = computed(() => {
  if (syntaxStatus.value === 'checking') return '检查中'
  if (syntaxStatus.value === 'failed') return '检查失败'
  if (syntaxDiagnostics.value.length) return `${syntaxDiagnostics.value.length} 个语法问题`
  if (syntaxStatus.value === 'clean') return '语法正常'
  if (syntaxStatus.value === 'dirty') return '待检查'
  return '未检查'
})

const selectedDatapoint = computed(() =>
  datapointPickerOptions.value.find((point) => String(point.id) === selectedDatapointId.value),
)

const datapointPickerTotalPages = computed(() =>
  Math.max(1, Math.ceil(datapointPickerTotal.value / datapointPickerPageSize)),
)

watch(selectedDatapoint, (point) => {
  selectedDatapointAlias.value = point
    ? uniqueDatapointAlias(point.name || point.path.split('.').pop() || 'tag')
    : ''
})

watch(codeTemplates, (templates) => {
  if (!templates.some((item) => item.id === activeTemplateId.value)) {
    activeTemplateId.value = templates[0]?.id || 'argv'
  }
})

watch(
  () => props.activeId,
  () => {
    activePanel.value = 'inputs'
    panelCollapsed.value = true
    panelHeight.value = panelDefaultHeight
    inspectorOpen.value = false
    debugResult.value = null
    debugError.value = ''
    syntaxStatus.value = 'idle'
    syntaxDiagnostics.value = []
    syntaxErrorText.value = ''
    monacoEditorRef.value?.setDiagnostics?.([])
    resetDebugArgvFromDefinition(false)
    resetDebugDatapointsFromDefinition(false)
    scheduleSyntaxCheck()
  },
)

onBeforeUnmount(() => {
  stopPanelResize()
  clearSyntaxCheckTimer()
})

watch(
  () => [props.activeDraft?.code, props.activeDraft?.lang],
  () => {
    if (!props.activeDraft) return
    syntaxStatus.value = 'dirty'
    scheduleSyntaxCheck()
  },
)

function markDirty() {
  if (props.activeDraft) {
    emit('mark-dirty', props.activeDraft.id)
  }
}

function clearSyntaxCheckTimer() {
  if (syntaxCheckTimer) {
    window.clearTimeout(syntaxCheckTimer)
    syntaxCheckTimer = null
  }
}

function scheduleSyntaxCheck() {
  clearSyntaxCheckTimer()
  if (!props.activeDraft) return
  syntaxCheckTimer = window.setTimeout(() => {
    void runSyntaxCheck()
  }, 600)
}

async function runSyntaxCheck() {
  if (!sandboxAvailable.value) {
    syntaxStatus.value = 'failed'
    syntaxErrorText.value = '独立计算沙箱不可用'
    return false
  }
  if (!props.activeDraft) return false
  clearSyntaxCheckTimer()
  const seq = ++syntaxCheckSeq
  syntaxStatus.value = 'checking'
  syntaxErrorText.value = ''
  try {
    const result = await checkComputeSyntax(props.projectId, {
      lang: props.activeDraft.lang,
      code: props.activeDraft.code,
    })
    if (seq !== syntaxCheckSeq) return false
    syntaxDiagnostics.value = result.diagnostics || []
    syntaxStatus.value = syntaxDiagnostics.value.length ? 'failed' : 'clean'
    monacoEditorRef.value?.setDiagnostics?.(syntaxDiagnostics.value)
    return syntaxDiagnostics.value.length === 0
  } catch (error) {
    if (seq !== syntaxCheckSeq) return false
    syntaxDiagnostics.value = []
    monacoEditorRef.value?.setDiagnostics?.([])
    syntaxStatus.value = 'failed'
    syntaxErrorText.value = getApiErrorMessage(error, '语法检查失败')
    return false
  }
}

async function ensureSyntaxClean() {
  const ok = await runSyntaxCheck()
  if (!ok) {
    activePanel.value = 'problems'
    panelCollapsed.value = false
    panelHeight.value = Math.max(panelHeight.value, 300)
    ElMessage.warning(syntaxDiagnostics.value.length ? '请先修复语法问题' : syntaxErrorText.value)
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

function updateOutputBindings() {
  markDirty()
}

async function saveAfterSyntaxCheck() {
  if (!props.activeDraft) return
  if (!(await ensureSyntaxClean())) return
  emit('save', props.activeDraft.id)
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
  if (!props.activeDraft) return
  props.activeDraft.parameterRows.push({
    uid: crypto.randomUUID(),
    name: `arg${props.activeDraft.parameterRows.length + 1}`,
    type: 'string',
    required: true,
    defaultValue: '',
    description: '',
  })
  markDirty()
}

function addDatapointVariable(point: Datapoint, alias: string) {
  if (!props.activeDraft) return
  props.activeDraft.datapointVariableRows.push({
    uid: crypto.randomUUID(),
    alias,
    path: point.path,
    datapointId: point.id,
    dataType: point.dataType,
  })
  markDirty()
}

function removeDatapointVariable(index: number) {
  if (!props.activeDraft) return
  props.activeDraft.datapointVariableRows.splice(index, 1)
  markDirty()
}

function removeUnusedDatapointVariables() {
  if (!props.activeDraft) return
  const nextRows = props.activeDraft.datapointVariableRows.filter((row) =>
    isDatapointVariableReferenced(row.alias),
  )
  if (nextRows.length === props.activeDraft.datapointVariableRows.length) return
  props.activeDraft.datapointVariableRows.splice(
    0,
    props.activeDraft.datapointVariableRows.length,
    ...nextRows,
  )
  markDirty()
  ElMessage.success('已清理未引用变量')
}

function removeInput(index: number) {
  if (!props.activeDraft) return
  props.activeDraft.parameterRows.splice(index, 1)
  markDirty()
}

async function openDatapointPicker(intent: 'variable' | 'trigger') {
  datapointPickerIntent.value = intent
  datapointPickerVisible.value = true
  selectedDatapointId.value = null
  selectedDatapointAlias.value = ''
  await reloadPickerDatapoints()
}

async function reloadPickerDatapoints() {
  datapointPickerPage.value = 1
  await loadPickerDatapoints()
}

async function loadPickerDatapoints() {
  if (!props.projectId) return
  datapointPickerLoading.value = true
  try {
    const params: Record<string, unknown> = {
      search: datapointPickerKeyword.value,
      page: datapointPickerPage.value,
      pageSize: datapointPickerPageSize,
    }
    if (datapointPickerSource.value) {
      params.type = datapointPickerSource.value
      params.sourceType = datapointPickerSource.value
    }
    if (datapointPickerDataType.value) {
      params.dataType = datapointPickerDataType.value
    }
    if (datapointPickerStatus.value) {
      params.status = datapointPickerStatus.value
    }
    const result = await getDatapoints(props.projectId, params)
    datapointPickerOptions.value = result.list
    datapointPickerTotal.value = Number(result.pagination?.total ?? result.list.length)
    if (
      selectedDatapointId.value &&
      !result.list.some((point) => String(point.id) === selectedDatapointId.value)
    ) {
      selectedDatapointId.value = null
    }
  } finally {
    datapointPickerLoading.value = false
  }
}

async function changePickerPage(page: number) {
  const nextPage = Math.min(Math.max(1, page), datapointPickerTotalPages.value)
  if (nextPage === datapointPickerPage.value) return
  datapointPickerPage.value = nextPage
  await loadPickerDatapoints()
}

function confirmSelectedDatapoint(point?: Datapoint) {
  const target = point || selectedDatapoint.value
  if (!target) return
  if (datapointPickerIntent.value === 'trigger') {
    if (props.activeDraft) {
      props.activeDraft.triggerConfig = {
        ...props.activeDraft.triggerConfig,
        datapointId: target.id,
        path: target.path,
      }
      markDirty()
    }
    datapointPickerVisible.value = false
    return
  }
  const alias = selectedDatapointAlias.value.trim() || uniqueDatapointAlias(target.name)
  if (!isValidVariableName(alias, props.activeDraft.lang)) {
    ElMessage.warning('变量名必须是当前脚本语言的合法变量名')
    return
  }
  if (isDatapointAliasUsed(alias)) {
    ElMessage.warning('变量名已存在')
    return
  }
  datapointPickerVisible.value = false
  try {
    addDatapointVariable(target, alias)
    insertVariableAlias(alias)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '插入数据点失败'))
  }
}

function insertVariableAlias(alias: string) {
  monacoEditorRef.value?.insertText?.(alias)
  markDirty()
}

async function executeDebug() {
  if (!sandboxAvailable.value) {
    debugError.value = '独立计算沙箱不可用'
    return
  }
  if (!props.activeDraft) return
  if (props.activeDraft.dirty) {
    ElMessage.warning('请先保存后再试运行')
    return
  }
  if (!(await ensureSyntaxClean())) return
  let input: Record<string, unknown>
  try {
    input = parseDebugInput()
  } catch (error) {
    debugError.value = error instanceof Error ? error.message : '试运行输入 JSON 格式无效'
    return
  }

  debugRunning.value = true
  debugResult.value = null
  debugError.value = ''
  try {
    debugResult.value = await debugComputeUnit(props.projectId, props.activeDraft.id, input, true)
    activeDebugResultTab.value = debugErrorText.value ? 'error' : 'output'
    ElMessage.success('试运行完成')
  } catch (error) {
    debugError.value = getApiErrorMessage(error, '试运行失败')
    activeDebugResultTab.value = 'error'
  } finally {
    debugRunning.value = false
  }
}

async function quickDryRun() {
  activePanel.value = 'debug'
  panelCollapsed.value = false
  panelHeight.value = Math.max(panelHeight.value, 360)
  await executeDebug()
}

function setTriggerType(type: string) {
  if (!props.activeDraft) return
  if (type === 'schedule_interval') {
    props.activeDraft.triggerType = 'schedule'
    props.activeDraft.triggerConfig = {
      kind: 'interval',
      every: 1,
      unit: 'minutes',
    }
  } else if (type === 'schedule_daily') {
    props.activeDraft.triggerType = 'schedule'
    props.activeDraft.triggerConfig = {
      kind: 'daily',
      time: '00:00:00',
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai',
    }
  } else if (type === 'schedule_weekly') {
    props.activeDraft.triggerType = 'schedule'
    props.activeDraft.triggerConfig = {
      kind: 'weekly',
      weekdays: [1],
      time: '00:00:00',
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai',
    }
  } else if (type === 'datapoint_change') {
    props.activeDraft.triggerType = type
    props.activeDraft.triggerConfig = {
      datapointId: '',
      path: '',
    }
  } else {
    props.activeDraft.triggerType = 'manual'
    props.activeDraft.triggerConfig = {}
  }
  markDirty()
  schedulePreviewText.value = ''
}

async function previewSchedule() {
  if (!props.activeDraft || props.activeDraft.triggerType !== 'schedule') return
  try {
    const preview = await previewComputeSchedule(props.projectId, {
      triggerType: props.activeDraft.triggerType,
      triggerConfig: props.activeDraft.triggerConfig,
    })
    if (preview.errors.length > 0) {
      schedulePreviewText.value = preview.errors.map((item) => item.message).join('；')
      return
    }
    const next = preview.nextRuns[0]
      ? new Date(preview.nextRuns[0]).toLocaleString()
      : '暂无后续执行时间'
    schedulePreviewText.value = `${preview.summary}；下次：${next}`
  } catch (error) {
    schedulePreviewText.value = getApiErrorMessage(error, '定时配置无效')
  }
}

function isDependencyChecked(id: string) {
  return props.activeDraft?.dependencies.some((item) => item.id === id) || false
}

function toggleDependency(id: string) {
  if (!props.activeDraft) return
  if (isDependencyChecked(id)) {
    props.activeDraft.dependencies = props.activeDraft.dependencies.filter((item) => item.id !== id)
  } else {
    props.activeDraft.dependencies.push({ id })
  }
  markDirty()
}

function dependencyName(id: string) {
  return props.dependencies.find((item) => item.id === id)?.name || id
}

function triggerIcon(id: string) {
  if (id.startsWith('schedule_')) return IconTablerClock
  if (id === 'datapoint_change') return IconTablerDatabaseImport
  return IconTablerManualGearbox
}

function triggerDescription(id: string) {
  if (id === 'schedule_interval') return '每隔一段时间执行'
  if (id === 'schedule_daily') return '每天指定时间执行'
  if (id === 'schedule_weekly') return '每周指定日期执行'
  if (id === 'datapoint_change') return '数据点变化时执行'
  return '调用方主动执行'
}

function isTriggerOptionActive(id: string) {
  if (!props.activeDraft) return false
  if (id.startsWith('schedule_'))
    return (
      props.activeDraft.triggerType === 'schedule' &&
      props.activeDraft.triggerConfig.kind === id.replace('schedule_', '')
    )
  return props.activeDraft.triggerType === id
}

function parseDebugInput(): Record<string, unknown> {
  let argv: unknown
  let datapoints: unknown
  try {
    argv = JSON.parse(debugArgvText.value || '[]')
  } catch {
    throw new Error('argv JSON 格式无效')
  }
  try {
    datapoints = JSON.parse(debugDatapointText.value || '{}')
  } catch {
    throw new Error('数据点变量模拟 JSON 格式无效')
  }
  if (!Array.isArray(argv)) {
    throw new Error('argv JSON 必须是数组')
  }
  if (!datapoints || typeof datapoints !== 'object' || Array.isArray(datapoints)) {
    throw new Error('数据点变量模拟 JSON 必须是对象')
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

function resetDebugArgvFromDefinition(showMessage = true) {
  debugArgvText.value = buildDefaultDebugArgv()
  if (showMessage) ElMessage.success('已按参数定义重置')
}

function resetDebugDatapointsFromDefinition(showMessage = true) {
  debugDatapointText.value = buildDefaultDebugDatapoints()
  if (showMessage) ElMessage.success('已按数据点变量重置')
}

function buildDefaultDebugArgv() {
  const argv = (props.activeDraft?.parameterRows || []).map((row) =>
    defaultValueByType(row.type, row.defaultValue),
  )
  return JSON.stringify(argv, null, 2)
}

function buildDefaultDebugDatapoints() {
  const values = Object.fromEntries(
    (props.activeDraft?.datapointVariableRows || [])
      .filter((row) => row.alias.trim())
      .map((row) => [row.alias.trim(), defaultValueByType(row.dataType || 'string', '')]),
  )
  return JSON.stringify(values, null, 2)
}

function defaultValueByType(type: string, defaultValue: string) {
  if (defaultValue !== '') {
    if (type === 'number') return Number(defaultValue)
    if (type === 'boolean') return defaultValue === 'true'
    if (type === 'object' || type === 'array') {
      try {
        return JSON.parse(defaultValue)
      } catch {
        return type === 'array' ? [] : {}
      }
    }
    return defaultValue
  }
  if (type === 'number') return 0
  if (type === 'boolean') return false
  if (type === 'object') return {}
  if (type === 'array') return []
  return ''
}

function uniqueDatapointAlias(name: string) {
  const base = normalizeVariableName(name, props.activeDraft?.lang)
  const used = new Set((props.activeDraft?.datapointVariableRows || []).map((row) => row.alias))
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

function isVariableNameStart(char: string, lang?: ComputeLang | string) {
  if (!char) return false
  if (char === '_') return true
  if (lang !== 'python' && char === '$') return true
  return /\p{ID_Start}/u.test(char)
}

function isVariableNameChar(char: string, isStart: boolean, lang?: ComputeLang | string) {
  if (isStart) return isVariableNameStart(char, lang)
  if (char === '_') return true
  if (lang !== 'python' && char === '$') return true
  return /\p{ID_Continue}/u.test(char)
}

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
    return hasValidChars && !name.startsWith('__') && !pythonReservedVariableNames.has(name)
  }
  return hasValidChars && !jsReservedVariableNames.has(name)
}

function isDatapointAliasUsed(name: string) {
  return Boolean(props.activeDraft?.datapointVariableRows.some((row) => row.alias === name))
}

function isDatapointVariableReferenced(alias: string) {
  if (!props.activeDraft) return false
  const name = alias.trim()
  if (!name) return false
  const pattern = new RegExp(
    `(?<![\\p{ID_Continue}$])${escapeRegExp(name)}(?![\\p{ID_Continue}$])`,
    'u',
  )
  return pattern.test(props.activeDraft.code || '')
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function insertActiveTemplate() {
  if (!activeTemplate.value) return
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
  return map[lang || ''] || '未知语言'
}

const statusText = (status?: string) => {
  const map: Record<string, string> = {
    enabled: '启用',
    idle: '空闲',
    running: '运行中',
    error: '异常',
    disabled: '停用',
  }
  return map[status || ''] || '未知'
}

const datapointStatusText = (status?: string) => {
  const map: Record<string, string> = {
    active: '正常',
    inactive: '停用',
    error: '异常',
    unknown: '未知',
  }
  return map[status || ''] || '未知'
}

const sourceTypeText = (sourceType?: string) => {
  const map: Record<string, string> = {
    'mqtt.subscription': 'MQTT',
    'db.query': '数据库',
    http: 'HTTP',
    manual: '手动',
  }
  return map[sourceType || ''] || sourceType || '-'
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
  padding: 8px 16px;
  color: var(--el-color-warning-dark-2);
  font-size: 12px;
  line-height: 20px;
  background: var(--el-color-warning-light-9);
  border-bottom: 1px solid var(--el-color-warning-light-7);
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
  color: var(--dc-surface-raised);
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
  color: var(--dc-surface-raised);
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
.compute-editor__mapping-row input,
.compute-editor__mapping-row select,
.compute-editor__alias-field input {
  height: 26px;
  min-width: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text);
  padding: 0 8px;
  font-size: 12px;
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

.compute-editor__trigger-layout {
  min-width: 0;
  display: grid;
  grid-template-columns: 210px minmax(0, 1fr);
  gap: 10px;
}

.compute-editor__trigger-rail {
  min-width: 0;
  display: grid;
  align-content: start;
  gap: 6px;
  padding-right: 10px;
  border-right: 1px solid var(--dc-border);
}

.compute-editor__trigger-rail button {
  min-height: 34px;
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
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__trigger-rail small {
  overflow: hidden;
  color: var(--dc-text-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__trigger-config {
  min-width: 0;
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
  max-width: 620px;
  align-items: stretch;
  flex-direction: column;
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

.compute-editor__trigger-note,
.compute-editor__trigger-preview span {
  margin: 0;
  color: var(--dc-text-muted);
  font-size: 11px;
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

.compute-editor__variable-row.is-unused {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
}

.compute-editor__variable-alias {
  min-width: 0;
  height: 26px;
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

.compute-editor__variable-state.is-unused {
  background: rgba(245, 158, 11, 0.12);
  color: var(--dc-warning);
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

.compute-editor__picker {
  display: grid;
  gap: 10px;
}

.compute-editor__picker-toolbar {
  display: grid;
  grid-template-columns: minmax(200px, 1fr) 120px 112px 112px auto;
  gap: 8px;
  align-items: center;
}

.compute-editor__picker-select {
  height: 30px;
  min-width: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text-secondary);
  padding: 0 8px;
  font-size: 12px;
  font-weight: 700;
}

.compute-editor__picker-loading {
  padding: 10px;
}

.compute-editor__picker-table-wrap {
  min-height: 300px;
  max-height: 390px;
  overflow: auto;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
}

.compute-editor__picker-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.compute-editor__picker-table th,
.compute-editor__picker-table td {
  min-width: 0;
  padding: 8px 10px;
  border-bottom: 1px solid var(--dc-border);
  text-align: left;
  vertical-align: middle;
}

.compute-editor__picker-table th {
  position: sticky;
  top: 0;
  z-index: 1;
  background: var(--dc-surface-raised);
  color: var(--dc-text-muted);
  font-size: 12px;
  font-weight: 800;
}

.compute-editor__picker-table th:nth-child(1) {
  width: 20%;
}

.compute-editor__picker-table th:nth-child(2) {
  width: 32%;
}

.compute-editor__picker-table th:nth-child(3) {
  width: 12%;
}

.compute-editor__picker-table th:nth-child(4) {
  width: 14%;
}

.compute-editor__picker-table th:nth-child(5) {
  width: 12%;
}

.compute-editor__picker-table th:nth-child(6) {
  width: 10%;
}

.compute-editor__picker-table tbody tr {
  cursor: pointer;
}

.compute-editor__picker-table tbody tr:hover,
.compute-editor__picker-table tbody tr.is-selected {
  background: var(--dc-primary-soft);
}

.compute-editor__picker-table strong,
.compute-editor__picker-table code,
.compute-editor__picker-table td {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__picker-table strong {
  display: block;
  color: var(--dc-text);
  font-size: 13px;
}

.compute-editor__picker-table code {
  display: block;
  color: var(--dc-text-muted);
  font-family: Consolas, 'Courier New', monospace;
  font-size: 12px;
}

.compute-editor__picker-status {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 12px;
  font-weight: 800;
}

.compute-editor__picker-status.is-active {
  background: var(--dc-success-soft);
  color: var(--dc-success);
}

.compute-editor__picker-status.is-error {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.compute-editor__picker-status.is-inactive {
  color: var(--dc-text-muted);
}

.compute-editor__picker-row-action {
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 9px;
  border: 1px solid rgba(29, 78, 216, 0.28);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 800;
}

.compute-editor__picker-empty {
  margin: 10px;
  display: grid;
  gap: 5px;
}

.compute-editor__picker-empty strong {
  color: var(--dc-text);
}

.compute-editor__picker-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.compute-editor__picker-count {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.compute-editor__picker-count span {
  max-width: 240px;
  overflow: hidden;
  color: var(--dc-primary);
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__alias-field {
  display: inline-grid;
  grid-template-columns: auto 120px;
  align-items: center;
  gap: 6px;
  color: var(--dc-text-muted);
  font-size: 12px;
  font-weight: 800;
}

.compute-editor__picker-pager {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.compute-editor__picker-pager button {
  height: 30px;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 800;
}

.compute-editor__picker-pager button:disabled {
  cursor: not-allowed;
  opacity: 0.48;
}

.compute-editor__picker-pager span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.compute-editor__picker-confirm,
.compute-editor__picker-pager .compute-editor__picker-confirm {
  border-color: var(--dc-primary);
  background: var(--dc-primary);
  color: var(--dc-surface-raised);
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
  color: var(--dc-surface-raised);
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

  .compute-editor__trigger-layout {
    grid-template-columns: 1fr;
  }

  .compute-editor__trigger-rail {
    grid-template-columns: repeat(3, minmax(0, 1fr));
    padding-right: 0;
    padding-bottom: 8px;
    border-right: 0;
    border-bottom: 1px solid var(--dc-border);
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

  .compute-editor__picker-toolbar {
    grid-template-columns: 1fr 1fr;
  }

  .compute-editor__picker-footer {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
