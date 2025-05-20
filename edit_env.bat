@echo off
:: Check if the script is running as an administrator
NET SESSION >nul 2>&1
IF %ERRORLEVEL% NEQ 0 (
    :: Not running as admin, so relaunch with administrative privileges
    echo This script requires administrative privileges.
    echo Please approve the UAC prompt.
    powershell -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)

:: Script continues here with administrative privileges
echo Running with elevated rights...
REM Script to open the .env file in Notepad
SET APP_DIR=%~dp0
notepad "%APP_DIR%.env"