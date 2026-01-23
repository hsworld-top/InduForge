@echo off
chcp 65001 >nul
echo ============================================================
echo   NodeAgent 锁定文件清理工具
echo ============================================================
echo.

set "lockFile=%temp%\node_agent.lock"

if exist "%lockFile%" (
    echo 检测到锁定文件: %lockFile%
    echo.
    
    type "%lockFile%"
    echo.
    
    echo 正在清理...
    del /q "%lockFile%"
    
    if exist "%lockFile%" (
        echo 清理失败！
        echo 请手动删除: %lockFile%
    ) else (
        echo 清理成功！
    )
) else (
    echo 未检测到锁定文件。
    echo NodeAgent 应该可以正常启动。
)

echo.
echo 按任意键退出...
pause >nul
