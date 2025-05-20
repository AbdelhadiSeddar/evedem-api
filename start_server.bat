@echo off
REM Script to ensure the .env file is loaded before starting the server

REM Get the directory of the running script (installation directory)
SET APP_DIR=%~dp0

REM Check if the .env file exists in the installation directory
IF NOT EXIST "%APP_DIR%.env" (
    echo [ERROR] .env file not found in %APP_DIR%.
    echo Please ensure the .env file exists in the installation directory.
    exit /b 1
)

REM Load the .env file variables into the environment without including quotes
FOR /F "usebackq tokens=1,2 delims==" %%A IN (`findstr /V "#" "%APP_DIR%.env"`) DO (
    SET %%A=%%B
)

REM Start the server
echo Starting the server from %APP_DIR% with .env variables loaded...
"%APP_DIR%evedem-api.exe"

pause