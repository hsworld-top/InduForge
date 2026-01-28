@echo off
chcp 65001 >nul
cd /d "%~dp0"

REM MinIO root user/password (keep consistent with .env)
set MINIO_ROOT_USER=minioadmin
set MINIO_ROOT_PASSWORD=minioadmin

echo Starting MinIO Server...
echo Data Directory: %cd%\data
echo Access Key: %MINIO_ROOT_USER%
echo Secret Key: %MINIO_ROOT_PASSWORD%
echo.

bin\minio.exe server data --address ":9000" --console-address ":9001"

pause
