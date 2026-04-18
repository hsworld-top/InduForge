@echo off
chcp 65001 >nul
cd /d "%~dp0"

set "SEAWEEDFS_DATA_DIR=%cd%\data"
set "SEAWEEDFS_MASTER_DIR=%SEAWEEDFS_DATA_DIR%\master"
set "SEAWEEDFS_VOLUME_DIR=%SEAWEEDFS_DATA_DIR%\volume"
set "SEAWEEDFS_FILER_DIR=%SEAWEEDFS_DATA_DIR%\filer"
set "SEAWEEDFS_LOG_DIR=%cd%\logs"
set "SEAWEEDFS_S3_CONFIG=%cd%\s3.conf"

if "%SEAWEEDFS_S3_ACCESS_KEY%"=="" set "SEAWEEDFS_S3_ACCESS_KEY=minioadmin"
if "%SEAWEEDFS_S3_SECRET_KEY%"=="" set "SEAWEEDFS_S3_SECRET_KEY=minioadmin"
if "%SEAWEEDFS_S3_PORT%"=="" set "SEAWEEDFS_S3_PORT=25000"
if "%SEAWEEDFS_MASTER_PORT%"=="" set "SEAWEEDFS_MASTER_PORT=9333"
if "%SEAWEEDFS_MASTER_GRPC_PORT%"=="" set "SEAWEEDFS_MASTER_GRPC_PORT=19333"
if "%SEAWEEDFS_FILER_PORT%"=="" set "SEAWEEDFS_FILER_PORT=8888"
if "%SEAWEEDFS_FILER_GRPC_PORT%"=="" set "SEAWEEDFS_FILER_GRPC_PORT=18888"
if "%SEAWEEDFS_VOLUME_PORT%"=="" set "SEAWEEDFS_VOLUME_PORT=8080"
if "%SEAWEEDFS_VOLUME_GRPC_PORT%"=="" set "SEAWEEDFS_VOLUME_GRPC_PORT=18080"
if "%SEAWEEDFS_S3_GRPC_PORT%"=="" set "SEAWEEDFS_S3_GRPC_PORT=35000"

if not exist "%SEAWEEDFS_DATA_DIR%" mkdir "%SEAWEEDFS_DATA_DIR%"
if not exist "%SEAWEEDFS_MASTER_DIR%" mkdir "%SEAWEEDFS_MASTER_DIR%"
if not exist "%SEAWEEDFS_VOLUME_DIR%" mkdir "%SEAWEEDFS_VOLUME_DIR%"
if not exist "%SEAWEEDFS_FILER_DIR%" mkdir "%SEAWEEDFS_FILER_DIR%"
if not exist "%SEAWEEDFS_LOG_DIR%" mkdir "%SEAWEEDFS_LOG_DIR%"

call :check_port %SEAWEEDFS_MASTER_PORT% || goto :port_conflict
call :check_port %SEAWEEDFS_MASTER_GRPC_PORT% || goto :port_conflict
call :check_port %SEAWEEDFS_VOLUME_PORT% || goto :port_conflict
call :check_port %SEAWEEDFS_VOLUME_GRPC_PORT% || goto :port_conflict
call :check_port %SEAWEEDFS_FILER_PORT% || goto :port_conflict
call :check_port %SEAWEEDFS_FILER_GRPC_PORT% || goto :port_conflict
call :check_port %SEAWEEDFS_S3_PORT% || goto :port_conflict
call :check_port %SEAWEEDFS_S3_GRPC_PORT% || goto :port_conflict

(
echo {
echo   "identities": [
echo     {
echo       "name": "local-admin",
echo       "credentials": [
echo         {
echo           "accessKey": "%SEAWEEDFS_S3_ACCESS_KEY%",
echo           "secretKey": "%SEAWEEDFS_S3_SECRET_KEY%"
echo         }
echo       ],
echo       "actions": ["Admin", "Read", "Write", "List", "Tagging"]
echo     }
echo   ]
echo }
) > "%SEAWEEDFS_S3_CONFIG%"

echo Starting SeaweedFS (Windows split mode)...
echo Data Directory: %SEAWEEDFS_DATA_DIR%
echo Log Directory: %SEAWEEDFS_LOG_DIR%
echo S3 Endpoint: http://127.0.0.1:%SEAWEEDFS_S3_PORT%
echo Master UI: http://127.0.0.1:%SEAWEEDFS_MASTER_PORT%
echo Filer UI: http://127.0.0.1:%SEAWEEDFS_FILER_PORT%
echo Access Key: %SEAWEEDFS_S3_ACCESS_KEY%
echo Secret Key: %SEAWEEDFS_S3_SECRET_KEY%
echo S3 Config: %SEAWEEDFS_S3_CONFIG%
echo.

start "seaweed-master" /B cmd /c ""%~dp0weed.exe" master -ip=127.0.0.1 -ip.bind=127.0.0.1 -mdir="%SEAWEEDFS_MASTER_DIR%" -port=%SEAWEEDFS_MASTER_PORT% -port.grpc=%SEAWEEDFS_MASTER_GRPC_PORT% -peers=none >>"%SEAWEEDFS_LOG_DIR%\master.log" 2>&1"
timeout /t 1 /nobreak >nul

start "seaweed-volume" /B cmd /c ""%~dp0weed.exe" volume -ip=127.0.0.1 -ip.bind=127.0.0.1 -dir="%SEAWEEDFS_VOLUME_DIR%" -port=%SEAWEEDFS_VOLUME_PORT% -port.grpc=%SEAWEEDFS_VOLUME_GRPC_PORT% -mserver=127.0.0.1:%SEAWEEDFS_MASTER_PORT% >>"%SEAWEEDFS_LOG_DIR%\volume.log" 2>&1"
timeout /t 1 /nobreak >nul

start "seaweed-filer-s3" /B cmd /c ""%~dp0weed.exe" filer -ip=127.0.0.1 -ip.bind=127.0.0.1 -port=%SEAWEEDFS_FILER_PORT% -port.grpc=%SEAWEEDFS_FILER_GRPC_PORT% -defaultStoreDir="%SEAWEEDFS_FILER_DIR%" -master=127.0.0.1:%SEAWEEDFS_MASTER_PORT% -s3 -s3.port=%SEAWEEDFS_S3_PORT% -s3.port.grpc=%SEAWEEDFS_S3_GRPC_PORT% -s3.ip.bind=127.0.0.1 -s3.config="%SEAWEEDFS_S3_CONFIG%" >>"%SEAWEEDFS_LOG_DIR%\filer-s3.log" 2>&1"
timeout /t 2 /nobreak >nul

echo SeaweedFS started in background.
echo master log: %SEAWEEDFS_LOG_DIR%\master.log
echo volume log: %SEAWEEDFS_LOG_DIR%\volume.log
echo filer+s3 log: %SEAWEEDFS_LOG_DIR%\filer-s3.log
echo.
echo Stop command: .\stop.bat

pause
goto :eof

:check_port
netstat -ano | findstr /R /C:":%~1 .*LISTENING" >nul
if %errorlevel%==0 (
  echo [ERROR] Port %~1 is already in use.
  exit /b 1
)
exit /b 0

:port_conflict
echo.
echo Start aborted. Please free the ports above, or run .\stop.bat first.
pause
exit /b 1
