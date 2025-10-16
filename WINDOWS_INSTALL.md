# Windows Installation Guide for Data Splitter

## Option 1: Using PowerShell Script (Recommended)

1. Download the Windows binary: `data-splitter-windows-amd64.exe`
2. Open PowerShell as Administrator
3. Run the installation script:

```powershell
# Run the PowerShell script
.\install-windows.ps1 -BinaryPath "path\to\data-splitter-windows-amd64.exe"
```

## Option 2: Using Batch Script (No Admin Required)

1. Download the Windows binary: `data-splitter-windows-amd64.exe`
2. Open Command Prompt
3. Run the batch script:

```cmd
install-windows.bat "path\to\data-splitter-windows-amd64.exe"
```

This installs to your user directory (`%USERPROFILE%\bin\`) and adds to user PATH.

## Option 3: Manual Installation

1. Download `data-splitter-windows-amd64.exe`
2. Create a folder like `C:\Program Files\data-splitter\`
3. Copy the `.exe` file to that folder
4. Add the folder to your system PATH:
   - Right-click "This PC" → Properties → Advanced system settings
   - Click "Environment Variables"
   - Under "System variables", find "Path" and click "Edit"
   - Add `C:\Program Files\data-splitter\` to the list
   - Click OK and restart your command prompt/PowerShell

## Option 3: User Installation (No Admin Required)

1. Download `data-splitter-windows-amd64.exe`
2. Create a folder in your user directory: `%USERPROFILE%\bin\`
3. Copy the `.exe` file to that folder
4. Add to user PATH:
   - Open Environment Variables (search in Start menu)
   - Under "User variables", edit "Path"
   - Add `%USERPROFILE%\bin\` to the list

## Uninstallation

### Using PowerShell Script

```powershell
# Run as Administrator for system installation
.\uninstall-windows.ps1

# Or specify custom directory
.\uninstall-windows.ps1 -InstallDir "C:\custom\path"
```

### Using Batch Script

```cmd
uninstall-windows.bat
```

### Manual Uninstallation

1. Delete the `data-splitter.exe` file from your installation directory
2. Remove the installation directory if empty
3. Remove the directory from your PATH:
   - Right-click "This PC" → Properties → Advanced system settings
   - Click "Environment Variables"
   - Under "System/User variables", find "Path" and click "Edit"
   - Remove the data-splitter directory from the list
   - Click OK and restart your command prompt/PowerShell

## Usage

After installation, you can use data-splitter from any directory:

```cmd
# Command Prompt
data-splitter --info
data-splitter --config C:\path\to\config.yaml

# PowerShell
data-splitter --info
data-splitter --config "C:\path\to\config.yaml"
```

## Notes

- Make sure `config.yaml` and `.env` files are in your working directory
- The tool will create `logs\` folder in your working directory for log files
- Use forward slashes `/` or double backslashes `\\` in config file paths