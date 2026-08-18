param(
    [Parameter(Mandatory = $true)][string]$SourceRoot
)

$ErrorActionPreference = 'Stop'

$source = (Resolve-Path -LiteralPath $SourceRoot).Path
$client = Join-Path $source 'client'
$custom = Join-Path $source 'instance\custom'
$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$designerRoot = Split-Path -Parent $scriptRoot
$destination = Join-Path $designerRoot 'public\ht-editor'
$adapter = Join-Path $scriptRoot 'ht-editor\InduForgeService.js'
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)

function Write-Utf8NoBom {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][AllowEmptyString()][string]$Content
    )

    [System.IO.File]::WriteAllText($Path, $Content, $utf8NoBom)
}

foreach ($required in @(
    (Join-Path $client 'index.html'),
    (Join-Path $client 'index3d.html'),
    (Join-Path $custom 'configs\2d\config.js'),
    (Join-Path $custom 'configs\3d\config.js'),
    $adapter
)) {
    if (-not (Test-Path -LiteralPath $required -PathType Leaf)) {
        throw "HT 源码缺少必要文件: $required"
    }
}

if (Test-Path -LiteralPath $destination) {
    $resolvedDestination = (Resolve-Path -LiteralPath $destination).Path
    if (-not $resolvedDestination.StartsWith($designerRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "拒绝清理 Designer 目录之外的路径: $resolvedDestination"
    }
    Remove-Item -LiteralPath $resolvedDestination -Recurse -Force
}

New-Item -ItemType Directory -Path $destination | Out-Null
Copy-Item -Path (Join-Path $client '*') -Destination $destination -Recurse -Force

$customDestination = Join-Path $destination 'custom'
New-Item -ItemType Directory -Path $customDestination | Out-Null
Copy-Item -Path (Join-Path $custom '*') -Destination $customDestination -Recurse -Force

$previewSource = Join-Path $custom 'previews'
if (Test-Path -LiteralPath $previewSource) {
    Copy-Item -Path (Join-Path $previewSource '*') -Destination $destination -Recurse -Force
}

Copy-Item -LiteralPath $adapter -Destination (Join-Path $customDestination 'libs\InduForgeService.js') -Force
$legacyService = Join-Path $customDestination 'libs\WebSocketService.js'
if (Test-Path -LiteralPath $legacyService) {
    Remove-Item -LiteralPath $legacyService -Force
}

foreach ($relativeConfig in @('configs\2d\config.js', 'configs\3d\config.js')) {
    $configPath = Join-Path $customDestination $relativeConfig
    $content = Get-Content -LiteralPath $configPath -Raw
    $content = $content.Replace('custom/libs/WebSocketService.js', 'custom/libs/InduForgeService.js')
    $content = [regex]::Replace(
        $content,
        'serviceClass\s*:\s*["'']WebSocketService["'']',
        'serviceClass:"InduForgeService"'
    )
    $content = $content.Replace('// 为 socket.io 提供路径映射', '// 保留资源路径前缀配置入口')
    Write-Utf8NoBom -Path $configPath -Content $content
}

foreach ($entryPath in Get-ChildItem -LiteralPath $destination -Recurse -File -Filter '*.html' | Select-Object -ExpandProperty FullName) {
    $content = Get-Content -LiteralPath $entryPath -Raw
    $content = [regex]::Replace($content, '(?im)^.*socket\.io.*(?:\r?\n|$)', '')
    $isRootPreview = (Split-Path -Parent $entryPath) -eq $destination -and (Split-Path -Leaf $entryPath) -notlike 'index*.html'
    if ($isRootPreview -and $content -notmatch 'custom/libs/InduForgeService\.js') {
        $content = [regex]::Replace(
            $content,
            '(?i)</head>',
            "    <script src='custom/libs/InduForgeService.js'></script>`r`n</head>",
            1
        )
    }
    Write-Utf8NoBom -Path $entryPath -Content $content
}

$createdHook = Join-Path $customDestination 'configs\3d\config-onEditor3dCreated.js'
$hookContent = Get-Content -LiteralPath $createdHook -Raw
$focusCall = [regex]::Match(
    $hookContent,
    '([A-Za-z_$][A-Za-z0-9_$]*)\.scenes\.tree\.setFocusDataById\([^)]*\)'
)
if (-not $focusCall.Success) {
    throw 'HT 3D 创建钩子缺少默认场景定位逻辑'
}
$editorVariable = $focusCall.Groups[1].Value
$focusLogic = "(function(){var workspace=new URLSearchParams(window.location.search).get('workspace'),tabs=$editorVariable.leftTopTabView.getTabModel().getDatas(),targetTab=tabs.get(workspace==='model'?1:0);targetTab&&$editorVariable.leftTopTabView.getTabModel().sm().ss(targetTab)})()"
$hookContent = $hookContent.Remove($focusCall.Index, $focusCall.Length).Insert($focusCall.Index, $focusLogic)
Write-Utf8NoBom -Path $createdHook -Content $hookContent

$socketReferences = Get-ChildItem -LiteralPath $destination -Recurse -File -Filter '*.html' | Select-String -Pattern 'socket\.io'
if (@($socketReferences).Count -gt 0) {
    throw 'HT 入口仍包含 Socket.IO 引用'
}
if (Select-String -LiteralPath (Join-Path $customDestination 'configs\2d\config.js'), (Join-Path $customDestination 'configs\3d\config.js') -Pattern 'WebSocketService' -Quiet) {
    throw 'HT 配置仍引用 WebSocketService'
}
$previewPages = Get-ChildItem -LiteralPath $destination -File -Filter '*.html' | Where-Object { $_.Name -notlike 'index*.html' }
$missingPreviewAdapters = $previewPages | Where-Object { -not (Select-String -LiteralPath $_.FullName -Pattern 'custom/libs/InduForgeService\.js' -Quiet) }
if (@($missingPreviewAdapters).Count -gt 0) {
    throw 'HT 根预览页未完整注入 InduForgeService'
}

$files = Get-ChildItem -LiteralPath $destination -Recurse -File
$size = ($files | Measure-Object Length -Sum).Sum
Write-Host ("HT 源码迁移完成: {0} 个文件, {1:N2} MB" -f $files.Count, ($size / 1MB))
