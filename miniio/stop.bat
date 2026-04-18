@echo off
chcp 65001 >nul
cd /d "%~dp0"

if "%SEAWEEDFS_MASTER_PORT%"=="" set "SEAWEEDFS_MASTER_PORT=9333"
if "%SEAWEEDFS_MASTER_GRPC_PORT%"=="" set "SEAWEEDFS_MASTER_GRPC_PORT=19333"
if "%SEAWEEDFS_VOLUME_PORT%"=="" set "SEAWEEDFS_VOLUME_PORT=8080"
if "%SEAWEEDFS_VOLUME_GRPC_PORT%"=="" set "SEAWEEDFS_VOLUME_GRPC_PORT=18080"
if "%SEAWEEDFS_FILER_PORT%"=="" set "SEAWEEDFS_FILER_PORT=8888"
if "%SEAWEEDFS_FILER_GRPC_PORT%"=="" set "SEAWEEDFS_FILER_GRPC_PORT=18888"
if "%SEAWEEDFS_S3_PORT%"=="" set "SEAWEEDFS_S3_PORT=25000"
if "%SEAWEEDFS_S3_GRPC_PORT%"=="" set "SEAWEEDFS_S3_GRPC_PORT=35000"

echo Stopping SeaweedFS processes by listening ports...

call :kill_port %SEAWEEDFS_S3_GRPC_PORT%
call :kill_port %SEAWEEDFS_S3_PORT%
call :kill_port %SEAWEEDFS_FILER_GRPC_PORT%
call :kill_port %SEAWEEDFS_FILER_PORT%
call :kill_port %SEAWEEDFS_VOLUME_GRPC_PORT%
call :kill_port %SEAWEEDFS_VOLUME_PORT%
call :kill_port %SEAWEEDFS_MASTER_GRPC_PORT%
call :kill_port %SEAWEEDFS_MASTER_PORT%

echo Done.
pause
exit /b 0

:kill_port
set "TARGET_PORT=%~1"
set "FOUND=0"
for /f "tokens=5" %%i in ('netstat -ano ^| findstr /R /C:":%TARGET_PORT% .*LISTENING"') do (
  set "FOUND=1"
  echo - Port %TARGET_PORT% -> PID %%i
  taskkill /PID %%i /F >nul 2>&1
)
if "%FOUND%"=="0" (
  echo - Port %TARGET_PORT% has no active listener
)
exit /b 0
