<template>
  <div class="role-panel-guide">
    <div class="role-panel-guide__title">角色维护已迁移</div>
    <p class="role-panel-guide__desc">
      工程运行态用户与角色请到工程卡片「成员与权限」中维护，设计器内仅保留页面与动作的就近授权配置。
    </p>
  </div>
</template>

<style scoped>
.role-panel-guide {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  color: var(--designer-text-secondary);
  line-height: 1.6;
}

.role-panel-guide__title {
  font-size: var(--designer-font-md);
  font-weight: 600;
  color: var(--designer-text-primary);
}

.role-panel-guide__desc {
  margin: 0;
  font-size: var(--designer-font-sm);
}
</style>
