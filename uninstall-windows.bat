@echo off
REM Data Splitter - Windows Batch Uninstallation Script
REM This script removes data-splitter and cleans up PATH

echo Uninstalling data-splitter from Windows...

REM Default installation directories
set "SYSTEM_DIR=%ProgramFiles%\data-splitter"
set "USER_DIR=%USERPROFILE%\bin"

REM Check which installation exists
if exist "%SYSTEM_DIR%" (
    set "INSTALL_DIR=%SYSTEM_DIR%"
    echo Found system installation: %INSTALL_DIR%
    goto :uninstall
)

if exist "%USER_DIR%" (
    set "INSTALL_DIR=%USER_DIR%"
    echo Found user installation: %INSTALL_DIR%
    goto :uninstall
)

echo No installation found. Nothing to uninstall.
goto :end

:uninstall
REM Remove binary
set "BINARY=%INSTALL_DIR%\data-splitter.exe"
if exist "%BINARY%" (
    echo Removing binary: %BINARY%
    del "%BINARY%"
) else (
    echo Binary not found: %BINARY%
)

REM Remove directory if empty
dir /b "%INSTALL_DIR%" 2>nul | findstr "." >nul
if errorlevel 1 (
    echo Removing empty directory: %INSTALL_DIR%
    rmdir "%INSTALL_DIR%"
) else (
    echo Directory not empty, keeping: %INSTALL_DIR%
)

REM Remove from PATH (this is tricky in batch, we'll provide instructions)
echo.
echo PATH cleanup:
echo If you want to remove data-splitter from PATH:
echo 1. Press Win+R, type 'sysdm.cpl', press Enter
echo 2. Go to 'Advanced' tab ^> 'Environment Variables'
echo 3. Under 'System/User variables', find 'Path' and click 'Edit'
echo 4. Remove '%INSTALL_DIR%' from the list
echo 5. Click OK and restart your command prompt

echo.
echo Uninstallation complete!
echo.
echo Note: Your config files and logs remain untouched.

:end
pause