# PgVoyager Windows Installer
param(
    [int]$Port = 5137
)

$ErrorActionPreference = "Stop"

$InstallDir = "$env:LOCALAPPDATA\PgVoyager"
$StartMenuDir = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs"
$ConfigDir = "$env:LOCALAPPDATA\PgVoyager"
$Binary = "pgvoyager-windows-amd64.exe"
$DownloadUrl = "https://github.com/thelinuxer/pgvoyager/releases/latest/download/$Binary"

Write-Host "Installing PgVoyager..." -ForegroundColor Green
Write-Host "Port: $Port"

# Create install directory
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# Download binary if not present
$BinaryPath = Join-Path $InstallDir "pgvoyager.exe"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

if (Test-Path (Join-Path $ScriptDir "pgvoyager.exe")) {
    Write-Host "Copying pgvoyager.exe..."
    Copy-Item (Join-Path $ScriptDir "pgvoyager.exe") $BinaryPath -Force
} elseif (Test-Path (Join-Path $ScriptDir $Binary)) {
    Write-Host "Copying $Binary..."
    Copy-Item (Join-Path $ScriptDir $Binary) $BinaryPath -Force
} else {
    Write-Host "Downloading $Binary..."
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $BinaryPath
}

# Copy launcher
$LauncherPath = Join-Path $InstallDir "pgvoyager-launcher.bat"
if (Test-Path (Join-Path $ScriptDir "pgvoyager-launcher.bat")) {
    Copy-Item (Join-Path $ScriptDir "pgvoyager-launcher.bat") $LauncherPath -Force
}

# Copy icon if present
$IconPath = Join-Path $InstallDir "pgvoyager.ico"
if (Test-Path (Join-Path $ScriptDir "pgvoyager.ico")) {
    Copy-Item (Join-Path $ScriptDir "pgvoyager.ico") $IconPath -Force
}

# Create Start Menu shortcut
$ShortcutPath = Join-Path $StartMenuDir "PgVoyager.lnk"
$WshShell = New-Object -ComObject WScript.Shell
$Shortcut = $WshShell.CreateShortcut($ShortcutPath)
$Shortcut.TargetPath = $LauncherPath
$Shortcut.WorkingDirectory = $InstallDir
$Shortcut.Description = "PostgreSQL database explorer with AI assistant"
if (Test-Path $IconPath) {
    $Shortcut.IconLocation = $IconPath
}
$Shortcut.Save()

# Add to PATH (optional - user can uncomment)
# $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
# if ($UserPath -notlike "*$InstallDir*") {
#     [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
# }

# Save port configuration
$ConfigPath = Join-Path $ConfigDir "config.txt"
"PGVOYAGER_PORT=$Port" | Out-File -FilePath $ConfigPath -Encoding UTF8

Write-Host ""
Write-Host "PgVoyager installed successfully!" -ForegroundColor Green
Write-Host "Location: $InstallDir"
Write-Host "Server will run on port: $Port"
Write-Host "You can now launch it from the Start Menu or run: $LauncherPath"
Write-Host ""
Write-Host "To use a different port, reinstall with: .\install.ps1 -Port <port>"
