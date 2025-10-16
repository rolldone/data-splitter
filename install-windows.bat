@echo off
REM Data Splitter - Windows Batch Installation Script
REM This script installs data-splitter to a user directory and adds to PATH

echo Installing data-splitter on Windows...

REM Check if binary path is provided
if "%~1"=="" (
    echo Usage: install-windows.bat "path\to\data-splitter-windows-amd64.exe"
    echo Example: install-windows.bat "C:\Downloads\data-splitter-windows-amd64.exe"
    exit /b 1
)

set BINARY_PATH=%~1
set INSTALL_DIR=%USERPROFILE%\bin
set BINARY_NAME=data-splitter.exe

echo Binary path: %BINARY_PATH%
echo Install dir: %INSTALL_DIR%

REM Check if binary exists
if not exist "%BINARY_PATH%" (
    echo Error: Binary not found at %BINARY_PATH%
    exit /b 1
)

REM Create installation directory
if not exist "%INSTALL_DIR%" (
    mkdir "%INSTALL_DIR%"
    echo Created directory: %INSTALL_DIR%
)

REM Copy binary
copy "%BINARY_PATH%" "%INSTALL_DIR%\%BINARY_NAME%" >nul
if errorlevel 1 (
    echo Error: Failed to copy binary
    exit /b 1
)
echo Copied binary to: %INSTALL_DIR%\%BINARY_NAME%

REM Check if already in PATH
echo %PATH% | find /i "%INSTALL_DIR%" >nul
if errorlevel 1 (
    echo Adding to user PATH...
    setx PATH "%PATH%;%INSTALL_DIR%" >nul
    if errorlevel 1 (
        echo Warning: Failed to add to PATH. You may need to add %INSTALL_DIR% to your PATH manually.
    ) else (
        echo Added to PATH successfully.
    )
) else (
    echo Already in PATH.
)

echo.
echo Installation complete!
echo.
echo You can now run 'data-splitter' from anywhere:
echo   data-splitter --info
echo   data-splitter --config C:\path\to\config.yaml
echo.
echo Make sure config.yaml and .env are in your working directory.
echo.
echo Note: Restart your command prompt for PATH changes to take effect.

pause