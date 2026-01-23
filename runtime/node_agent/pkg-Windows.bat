@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

echo ============================================================
echo   NodeAgent 打包脚本
echo ============================================================
echo.

REM 获取脚本所在目录
set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

REM 切换到脚本目录
cd /d "%SCRIPT_DIR%"

echo 工作目录: %SCRIPT_DIR%
echo.

REM 检查 Go 是否安装
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo 错误: Go 未安装或未在 PATH 中
    echo 请先安装 Go: https://golang.org/dl/
    echo.
    pause
    exit /b 1
)

echo Go 版本:
go version
echo.

REM 设置前端目录（使用更直接的方式）
set "FRONTEND_DIR=%SCRIPT_DIR%\..\node_agent_front"
set "FRONTEND_DIR=%FRONTEND_DIR:\=\%"

REM 设置静态目录
set "STATIC_DIR=%SCRIPT_DIR%\internal\web\static\dist"

echo 前端目录: %FRONTEND_DIR%
echo 静态目录: %STATIC_DIR%
echo.

echo ============================================================
echo   构建前端资源
echo ============================================================
echo.

REM 检查前端目录是否存在
if not exist "%FRONTEND_DIR%" (
    echo 警告: 前端目录不存在，跳过前端构建
    echo 前端目录: %FRONTEND_DIR%
    echo.
) else (
    REM 检查 pnpm 是否安装
    where pnpm >nul 2>&1
    if %errorlevel% neq 0 (
        echo 警告: pnpm 未安装，跳过前端构建
        echo 请先安装 pnpm: https://pnpm.io/installation
        echo.
    ) else (
        echo 正在构建前端...

        REM 进入前端目录并执行 pnpm 命令（使用 call 以确保变量正确展开）
        echo 当前目录: %FRONTEND_DIR%

        pushd "%FRONTEND_DIR%" >nul
        if %errorlevel% neq 0 (
            echo 错误: 无法进入前端目录！
            echo.
            pause
            exit /b 1
        )

        echo 已切换到目录: %CD%

        REM 安装依赖
        echo 安装依赖...
        call pnpm install

        if %errorlevel% neq 0 (
            echo 错误: 前端依赖安装失败！
            echo.
            popd >nul
            pause
            exit /b 1
        )

        REM 构建前端
        echo 构建前端...
        call pnpm build

        if %errorlevel% neq 0 (
            echo 错误: 前端构建失败！
            echo.
            popd >nul
            pause
            exit /b 1
        )

        echo 前端构建成功！

        REM 回到原目录
        popd >nul

        REM 创建静态目录并复制文件
        echo 复制前端资源...
        if not exist "%STATIC_DIR%" mkdir "%STATIC_DIR%"
        xcopy "%FRONTEND_DIR%\dist\*" "%STATIC_DIR%" /E /Y /Q

        echo 前端资源复制完成！
    )
)

echo.

REM 创建构建目录
if not exist "build" mkdir build

echo ============================================================
echo   构建 Windows 版本
echo ============================================================
echo.

REM 清理旧的构建产物
del /q build\node_agent.exe 2>nul
del /q build\node_agent.pdb 2>nul

REM 构建
echo 正在构建 Windows 版本...
go build -ldflags "-X main.version=1.0.0" -o build\node_agent.exe cmd\main.go

if %errorlevel% neq 0 (
    echo.
    echo 错误: 构建失败！
    echo.
    pause
    exit /b 1
)

echo.
echo 构建成功！
echo.

REM 显示构建产物
if exist "build\node_agent.exe" (
    echo 构建产物:
    for %%F in (build\node_agent.exe) do (
        echo   - %%~nxF
        echo     大小: %%~zF 字节
    )
)

REM 检查是否包含前端资源
if exist "%STATIC_DIR%\index.html" (
    echo   - Web 管理界面 (包含前端资源)
) else (
    echo   - 仅后端 API (无前端资源)
)

echo.
echo ============================================================
echo   打包完成！
echo ============================================================
echo.
echo 下一步:
echo   1. 复制 build\node_agent.exe 到目标机器
echo   2. 复制 configs\config.yaml 到同目录
echo   3. 运行 node_agent.exe 进行配置
echo.
echo 按任意键退出...
pause >nul
