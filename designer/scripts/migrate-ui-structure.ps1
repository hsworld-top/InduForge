$ErrorActionPreference = "Stop"
Set-Location "D:\SVNCode\indu-forge\designer\src\ui"

# shell
New-Item -ItemType Directory -Force -Path shell | Out-Null
if (Test-Path "Designer\DesignerView.vue") {
  Move-Item -Path "Designer\DesignerView.vue" -Destination "shell\DesignerView.vue"
  Remove-Item -Path "Designer" -Recurse -Force -ErrorAction SilentlyContinue
}
if (Test-Path "TopToolbar") { Move-Item -Path "TopToolbar" -Destination "shell\TopToolbar" }
if (Test-Path "ToolRail") { Move-Item -Path "ToolRail" -Destination "shell\ToolRail" }
if (Test-Path "DockPanel") { Move-Item -Path "DockPanel" -Destination "shell\DockPanel" }

# editors dirs
@(
  "editors\page\canvas\services",
  "editors\page\canvas\components",
  "editors\page\panels\left",
  "editors\page\panels\right",
  "editors\page\panels\composables",
  "editors\page\preview",
  "editors\diagram",
  "editors\scene",
  "shared\panels",
  "shared\utils"
) | ForEach-Object { New-Item -ItemType Directory -Force -Path $_ | Out-Null }

if (Test-Path "Canvas\components\Diagram2D.vue") {
  Move-Item -Path "Canvas\components\Diagram2D.vue" -Destination "editors\diagram\Diagram2D.vue"
}
if (Test-Path "Canvas\composables") {
  Move-Item -Path "Canvas\composables" -Destination "editors\page\canvas\composables"
}
if (Test-Path "Canvas\use-drag-state.js") {
  Move-Item -Path "Canvas\use-drag-state.js" -Destination "editors\page\canvas\composables\use-drag-state.js"
}
if (Test-Path "Canvas\DragDropManager.js") {
  Move-Item -Path "Canvas\DragDropManager.js" -Destination "editors\page\canvas\services\DragDropManager.js"
}
if (Test-Path "Canvas\GridSnapping.js") {
  Move-Item -Path "Canvas\GridSnapping.js" -Destination "editors\page\canvas\services\GridSnapping.js"
}
if (Test-Path "Canvas\components\EChart.vue") {
  Move-Item -Path "Canvas\components\EChart.vue" -Destination "editors\page\canvas\components\EChart.vue"
}
if (Test-Path "Canvas\components\GridContainer.vue") {
  Move-Item -Path "Canvas\components\GridContainer.vue" -Destination "editors\page\canvas\components\GridContainer.vue"
}

if (Test-Path "Canvas") {
  Get-ChildItem "Canvas" -File | ForEach-Object { Move-Item $_.FullName -Destination "editors\page\canvas\" }
  Remove-Item -Path "Canvas\components" -Recurse -Force -ErrorAction SilentlyContinue
  Remove-Item -Path "Canvas" -Recurse -Force -ErrorAction SilentlyContinue
}

if (Test-Path "Preview\PreviewView.vue") {
  Move-Item -Path "Preview\PreviewView.vue" -Destination "editors\page\preview\PreviewView.vue"
}
if (Test-Path "Preview\previewRuntime.js") {
  Move-Item -Path "Preview\previewRuntime.js" -Destination "editors\page\preview\previewRuntime.js"
}
Remove-Item -Path "Preview" -Recurse -Force -ErrorAction SilentlyContinue

$leftPage = @(
  "ComponentPanel.vue", "OutlineTree.vue", "MaterialPanel.vue", "DataPanel.vue",
  "DatapointPanel.vue", "ResourcePanel.vue", "CanvasToolsPanel.vue", "DrawingPanel.vue",
  "DiagramAreaPanel.vue", "SymbolLibraryPanel.vue"
)
foreach ($name in $leftPage) {
  $src = Join-Path "LeftPanel" $name
  if (Test-Path $src) { Move-Item -Path $src -Destination "editors\page\panels\left\$name" }
}

$sharedLeft = @("PageTree.vue", "ScriptVarsPanel.vue", "AiPanel.vue", "I18nPanel.vue", "RolePanel.vue")
foreach ($name in $sharedLeft) {
  $src = Join-Path "LeftPanel" $name
  if (Test-Path $src) { Move-Item -Path $src -Destination "shared\panels\$name" }
}

if (Test-Path "RightPanel") {
  Get-ChildItem "RightPanel" -File | Where-Object {
    $_.Name -notin @("VariablesPanel.vue", "use-panel-state.js", "use-multi-select.js")
  } | ForEach-Object { Move-Item $_.FullName -Destination "editors\page\panels\right\" }
  if (Test-Path "RightPanel\StylePanel") {
    Move-Item -Path "RightPanel\StylePanel" -Destination "editors\page\panels\right\StylePanel"
  }
  if (Test-Path "RightPanel\VariablesPanel.vue") {
    Move-Item -Path "RightPanel\VariablesPanel.vue" -Destination "shared\panels\VariablesPanel.vue"
  }
  if (Test-Path "RightPanel\use-panel-state.js") {
    Move-Item -Path "RightPanel\use-panel-state.js" -Destination "editors\page\panels\composables\use-panel-state.js"
  }
  if (Test-Path "RightPanel\use-multi-select.js") {
    Move-Item -Path "RightPanel\use-multi-select.js" -Destination "editors\page\panels\composables\use-multi-select.js"
  }
  Remove-Item -Path "RightPanel" -Recurse -Force -ErrorAction SilentlyContinue
}

Remove-Item -Path "LeftPanel" -Recurse -Force -ErrorAction SilentlyContinue

if (Test-Path "utils\component-methods.js") {
  Move-Item -Path "utils\component-methods.js" -Destination "shared\utils\component-methods.js"
}
Remove-Item -Path "utils" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path "DatapointPicker" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path "DiagnosticsPanel" -Recurse -Force -ErrorAction SilentlyContinue

Write-Host "migrate-ui-structure: OK"
