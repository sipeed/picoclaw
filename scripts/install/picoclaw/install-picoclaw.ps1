[CmdletBinding()]
param(
    [ValidateSet('system', 'user')]
    [string]$InstallMode = 'user',

    [ValidateSet('github', 'cdn')]
    [string]$Source = 'github',

    [string]$Arch
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$BaseUrls = @{
    github = 'https://github.com/sipeed/picoclaw/releases/latest/download/'
    cdn    = 'https://picoclaw-downloads.tos-cn-beijing.volces.com/latest/'
}

$Assets = @{
    windows = @{
        x86_64 = @{ file = 'picoclaw_Windows_x86_64.zip' }
        arm64  = @{ file = 'picoclaw_Windows_arm64.zip' }
    }
    linux = @{
        x86_64  = @{ file = 'picoclaw_Linux_x86_64.tar.gz'; deb = 'picoclaw_x86_64.deb'; rpm = 'picoclaw_x86_64.rpm' }
        arm64   = @{ file = 'picoclaw_Linux_arm64.tar.gz'; deb = 'picoclaw_aarch64.deb'; rpm = 'picoclaw_aarch64.rpm' }
        armv7   = @{ file = 'picoclaw_Linux_armv7.tar.gz'; deb = 'picoclaw_armv7.deb'; rpm = 'picoclaw_armv7.rpm' }
        armv6   = @{ file = 'picoclaw_Linux_armv6.tar.gz'; deb = 'picoclaw_armv6.deb'; rpm = 'picoclaw_armv6.rpm' }
        riscv64 = @{ file = 'picoclaw_Linux_riscv64.tar.gz'; deb = 'picoclaw_riscv64.deb'; rpm = 'picoclaw_riscv64.rpm' }
        mipsle  = @{ file = 'picoclaw_Linux_mipsle.tar.gz'; deb = 'picoclaw_mipsle.deb'; rpm = 'picoclaw_mipsle.rpm' }
        s390x   = @{ file = 'picoclaw_Linux_s390x.tar.gz'; deb = 'picoclaw_s390x.deb'; rpm = 'picoclaw_s390x.rpm' }
    }
    macos = @{
        x86_64 = @{ file = 'picoclaw_Darwin_x86_64.tar.gz' }
        arm64  = @{ file = 'picoclaw_Darwin_arm64.tar.gz' }
    }
    freebsd = @{
        x86_64 = @{ file = 'picoclaw_Freebsd_x86_64.tar.gz' }
        arm64  = @{ file = 'picoclaw_Freebsd_arm64.tar.gz' }
        armv7  = @{ file = 'picoclaw_Freebsd_armv7.tar.gz' }
    }
    netbsd = @{
        x86_64 = @{ file = 'picoclaw_Netbsd_x86_64.tar.gz' }
        arm64  = @{ file = 'picoclaw_Netbsd_arm64.tar.gz' }
    }
}

$ExpectedBins = @('picoclaw', 'picoclaw-launcher', 'picoclaw-launcher-tui')

function Get-CommandExists {
    param([Parameter(Mandatory = $true)][string]$Name)
    return $null -ne (Get-Command -Name $Name -ErrorAction SilentlyContinue)
}

function Get-OsId {
    if ($IsWindows) {
        return 'windows'
    }
    if ($IsLinux) {
        return 'linux'
    }
    if ($IsMacOS) {
        return 'macos'
    }

    $uname = (& uname -s).Trim().ToLowerInvariant()
    switch ($uname) {
        'freebsd' { return 'freebsd' }
        'netbsd' { return 'netbsd' }
        default {
            throw "Unsupported operating system: $uname"
        }
    }
}

function Resolve-Arch {
    param([Parameter(Mandatory = $true)][string]$OsId)

    if (-not [string]::IsNullOrWhiteSpace($Arch)) {
        return $Arch.ToLowerInvariant()
    }

    $machine = if ($IsWindows) {
        [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
    }
    else {
        (& uname -m).Trim().ToLowerInvariant()
    }

    switch ($machine) {
        'x64' { return 'x86_64' }
        'amd64' { return 'x86_64' }
        'x86_64' { return 'x86_64' }
        'arm64' { return 'arm64' }
        'aarch64' { return 'arm64' }
        'armv7l' { return 'armv7' }
        'armv7' { return 'armv7' }
        'armv6l' { return 'armv6' }
        'armv6' { return 'armv6' }
        'riscv64' { return 'riscv64' }
        'mipsel' { return 'mipsle' }
        'mipsle' { return 'mipsle' }
        's390x' { return 's390x' }
        default {
            throw "Unsupported architecture '$machine' for OS '$OsId'. Use -Arch to override."
        }
    }
}

function Assert-SystemInstallPrivileges {
    param([Parameter(Mandatory = $true)][string]$OsId)

    if ($InstallMode -ne 'system') {
        return
    }

    if ($OsId -eq 'windows') {
        $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
        $principal = New-Object Security.Principal.WindowsPrincipal($identity)
        $isAdmin = $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
        if (-not $isAdmin) {
            throw 'System install on Windows requires Administrator privileges.'
        }
        return
    }

    $uid = (& id -u).Trim()
    if ($uid -ne '0') {
        throw 'System install requires root privileges (sudo/root).'
    }
}

function Get-UnixSystemLayout {
    if (Test-Path -LiteralPath '/usr/local/share') {
        return @{
            installRoot = '/usr/local/share/picoclaw'
            linkDir     = '/usr/local/bin'
            mode        = 'symlink'
        }
    }

    if (Test-Path -LiteralPath '/usr/share') {
        return @{
            installRoot = '/usr/share/picoclaw'
            linkDir     = '/usr/bin'
            mode        = 'symlink'
        }
    }

    $tempInstallRoot = "/tmp/picoclaw-$([DateTimeOffset]::UtcNow.ToUnixTimeSeconds())"

    try {
        Get-ChildItem -Path '/tmp' -Filter 'picoclaw-*' -Directory -ErrorAction SilentlyContinue |
            ForEach-Object {
                if ($_.FullName -ne $tempInstallRoot) {
                    Remove-Item -LiteralPath $_.FullName -Recurse -Force -ErrorAction SilentlyContinue
                }
            }
    } catch {
        # Best-effort cleanup; ignore any failures to avoid breaking installation.
    }

    return @{
        installRoot = $tempInstallRoot
        linkDir     = '/usr/bin'
        mode        = 'copy-executables'
    }
}

function Get-InstallLayout {
    param([Parameter(Mandatory = $true)][string]$OsId)

    if ($InstallMode -eq 'user') {
        if ($OsId -eq 'windows') {
            $base = [Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)
            if ([string]::IsNullOrWhiteSpace($base) -or -not (Test-Path -LiteralPath $base)) {
                throw 'Failed to resolve a valid LocalApplicationData path.'
            }
            return @{
                installRoot = (Join-Path $base 'picoclaw')
                linkDir     = $null
                mode        = 'user'
            }
        }

        return @{
            installRoot = (Join-Path $HOME '.local/share/picoclaw')
            linkDir     = (Join-Path $HOME '.local/bin')
            mode        = 'user'
        }
    }

    if ($OsId -eq 'windows') {
        $programFiles = [Environment]::GetFolderPath([Environment+SpecialFolder]::ProgramFiles)
        if ([string]::IsNullOrWhiteSpace($programFiles) -or -not (Test-Path -LiteralPath $programFiles)) {
            throw 'ProgramFiles path from .NET API is invalid or missing. Installation terminated.'
        }
        return @{
            installRoot = (Join-Path $programFiles 'picoclaw')
            linkDir     = $null
            mode        = 'system'
        }
    }

    return Get-UnixSystemLayout
}

function New-DirectoryIfMissing {
    param([Parameter(Mandatory = $true)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

function Get-DownloadUrl {
    param(
        [Parameter(Mandatory = $true)][string]$FileName,
        [Parameter(Mandatory = $true)][string]$SourceName
    )
    return "$($BaseUrls[$SourceName])$FileName"
}

function Invoke-ArtifactDownload {
    param(
        [Parameter(Mandatory = $true)][string]$Url,
        [Parameter(Mandatory = $true)][string]$OutFile
    )
    Write-Host "Downloading: $Url"
    Invoke-WebRequest -Uri $Url -OutFile $OutFile
}

function Expand-Artifact {
    param(
        [Parameter(Mandatory = $true)][string]$ArchivePath,
        [Parameter(Mandatory = $true)][string]$Destination
    )

    New-DirectoryIfMissing -Path $Destination

    if ($ArchivePath.EndsWith('.zip', [StringComparison]::OrdinalIgnoreCase)) {
        Expand-Archive -LiteralPath $ArchivePath -DestinationPath $Destination -Force
        return
    }

    if ($ArchivePath.EndsWith('.tar.gz', [StringComparison]::OrdinalIgnoreCase)) {
        & tar -xzf $ArchivePath -C $Destination
        if ($LASTEXITCODE -ne 0) {
            throw "Failed to extract tar archive: $ArchivePath"
        }
        return
    }

    throw "Unsupported archive type: $ArchivePath"
}

function Get-BinaryPath {
    param(
        [Parameter(Mandatory = $true)][string]$Root,
        [Parameter(Mandatory = $true)][string]$Name
    )

    $direct = Join-Path $Root $Name
    if (Test-Path -LiteralPath $direct -PathType Leaf) {
        return $direct
    }

    $directExe = Join-Path $Root ("$Name.exe")
    if (Test-Path -LiteralPath $directExe -PathType Leaf) {
        return $directExe
    }

    return $null
}

function Set-UnixUserPathExport {
    param(
        [Parameter(Mandatory = $true)][string]$HomeDir,
        [Parameter(Mandatory = $true)][string]$BinDir
    )

    $line = 'export PATH="$HOME/.local/bin:$PATH"'
    $rcFiles = @(
        (Join-Path $HomeDir '.bashrc'),
        (Join-Path $HomeDir '.zshrc'),
        (Join-Path $HomeDir '.profile')
    )

    $updated = @()
    foreach ($rcPath in $rcFiles) {
        if (-not (Test-Path -LiteralPath $rcPath)) {
            New-Item -ItemType File -Path $rcPath -Force | Out-Null
        }

        $hasLine = Select-String -Path $rcPath -Pattern [regex]::Escape($line) -SimpleMatch -Quiet -ErrorAction SilentlyContinue
        if (-not $hasLine) {
            Add-Content -Path $rcPath -Value "`n$line"
            $updated += $rcPath
        }
    }

    return $updated
}

function Add-PathVariable {
    param(
        [Parameter(Mandatory = $true)][string]$PathToAdd,
        [Parameter(Mandatory = $true)][ValidateSet('User', 'Machine')][string]$Scope
    )

    $current = [Environment]::GetEnvironmentVariable('Path', $Scope)
    $parts = @()
    if (-not [string]::IsNullOrWhiteSpace($current)) {
        $parts = $current.Split(';') | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
    }

    if ($parts -contains $PathToAdd) {
        return
    }

    $newValue = if ($parts.Count -eq 0) { $PathToAdd } else { "$current;$PathToAdd" }
    [Environment]::SetEnvironmentVariable('Path', $newValue, $Scope)
}

function Install-LinuxPackageFromAsset {
    param(
        [Parameter(Mandatory = $true)][hashtable]$Asset,
        [Parameter(Mandatory = $true)][string]$TempDir
    )

    $debCmd = Get-CommandExists -Name 'dpkg'
    $rpmCmd = Get-CommandExists -Name 'rpm'

    if (-not $debCmd -and -not $rpmCmd) {
        return $null
    }

    if ($debCmd -and $Asset.ContainsKey('deb')) {
        $pkg = $Asset.deb
        $pkgPath = Join-Path $TempDir $pkg
        Invoke-ArtifactDownload -Url (Get-DownloadUrl -FileName $pkg -SourceName $Source) -OutFile $pkgPath
        & dpkg -i $pkgPath
        if ($LASTEXITCODE -ne 0) {
            throw 'dpkg install failed.'
        }
        return @{
            manager = 'dpkg'
            package = $pkgPath
        }
    }

    if ($rpmCmd -and $Asset.ContainsKey('rpm')) {
        $pkg = $Asset.rpm
        $pkgPath = Join-Path $TempDir $pkg
        Invoke-ArtifactDownload -Url (Get-DownloadUrl -FileName $pkg -SourceName $Source) -OutFile $pkgPath
        & rpm -Uvh --replacepkgs $pkgPath
        if ($LASTEXITCODE -ne 0) {
            throw 'rpm install failed.'
        }
        return @{
            manager = 'rpm'
            package = $pkgPath
        }
    }

    return $null
}

function Install-LinksOrCopies {
    param(
        [Parameter(Mandatory = $true)][string]$InstallRoot,
        [Parameter(Mandatory = $true)][string]$LinkDir,
        [Parameter(Mandatory = $true)][string]$Mode
    )

    New-DirectoryIfMissing -Path $LinkDir

    if ($Mode -eq 'copy-executables') {
        $copied = @()
        foreach ($name in $ExpectedBins) {
            $binPath = Get-BinaryPath -Root $InstallRoot -Name $name
            if ($null -eq $binPath) {
                continue
            }

            $dest = Join-Path $LinkDir (Split-Path -Path $binPath -Leaf)
            Copy-Item -LiteralPath $binPath -Destination $dest -Force
            $copied += $dest
        }

        return @{
            type  = 'copied'
            paths = $copied
        }
    }

    $linked = @()
    foreach ($name in $ExpectedBins) {
        $binPath = Get-BinaryPath -Root $InstallRoot -Name $name
        if ($null -eq $binPath) {
            continue
        }

        $target = Join-Path $LinkDir $name
        if (Test-Path -LiteralPath $target) {
            Remove-Item -LiteralPath $target -Force
        }

        if ($IsWindows) {
            Copy-Item -LiteralPath $binPath -Destination $target -Force
        }
        else {
            & ln -s $binPath $target
            if ($LASTEXITCODE -ne 0) {
                throw "Failed to create symlink: $target"
            }
        }
        $linked += $target
    }

    return @{
        type  = 'linked'
        paths = $linked
    }
}

function Write-MissingUserLocalWarning {
    param([Parameter(Mandatory = $true)][string]$OsId)

    if ($InstallMode -ne 'user' -or $OsId -eq 'windows') {
        return
    }

    $localRoot = Join-Path $HOME '.local'
    if (-not (Test-Path -LiteralPath $localRoot)) {
        Write-Host "Warning: $localRoot does not exist. The installer will create it automatically." -ForegroundColor Yellow
    }
}

$osId = Get-OsId
$archId = Resolve-Arch -OsId $osId
Assert-SystemInstallPrivileges -OsId $osId
Write-MissingUserLocalWarning -OsId $osId

if (-not $Assets.ContainsKey($osId)) {
    throw "OS '$osId' is not included in assets map."
}

$osAssets = $Assets[$osId]
if (-not $osAssets.ContainsKey($archId)) {
    $available = ($osAssets.Keys | Sort-Object) -join ', '
    throw "Architecture '$archId' is not available for '$osId'. Available: $available"
}

$asset = $osAssets[$archId]
$layout = Get-InstallLayout -OsId $osId

$tempDir = Join-Path ([IO.Path]::GetTempPath()) ("picoclaw-install-$([Guid]::NewGuid().ToString('N'))")
New-DirectoryIfMissing -Path $tempDir

try {
    Write-Host "Install mode: $InstallMode"
    Write-Host "OS/Arch: $osId/$archId"

    if ($osId -eq 'linux' -and $InstallMode -eq 'system') {
        $pkgInfo = Install-LinuxPackageFromAsset -Asset $asset -TempDir $tempDir
        if ($null -ne $pkgInfo) {
            Write-Host "Installed via package manager: $($pkgInfo.manager)"
            Write-Host "Package file: $($pkgInfo.package)"
            Write-Host 'Install location is managed by the package manager.'
            return
        }
    }

    $archiveName = $asset.file
    $archivePath = Join-Path $tempDir $archiveName
    $downloadUrl = Get-DownloadUrl -FileName $archiveName -SourceName $Source
    Invoke-ArtifactDownload -Url $downloadUrl -OutFile $archivePath

    New-DirectoryIfMissing -Path $layout.installRoot
    Expand-Artifact -ArchivePath $archivePath -Destination $layout.installRoot

    if ($osId -eq 'windows') {
        $pathScope = if ($InstallMode -eq 'system') { 'Machine' } else { 'User' }
        Add-PathVariable -PathToAdd $layout.installRoot -Scope $pathScope
        Write-Host "Installed to: $($layout.installRoot)"
        Write-Host "PATH scope updated: $pathScope"
        Write-Host 'Please restart your terminal or log out and back in for PATH changes to take effect.'
        return
    }

    $linkSummary = Install-LinksOrCopies -InstallRoot $layout.installRoot -LinkDir $layout.linkDir -Mode $layout.mode

    Write-Host "Extracted to: $($layout.installRoot)"
    if ($linkSummary.paths.Count -gt 0) {
        if ($linkSummary.type -eq 'copied') {
            Write-Host "Copied executables to: $($layout.linkDir)"
        }
        else {
            Write-Host "Created links in: $($layout.linkDir)"
        }
        $linkSummary.paths | ForEach-Object { Write-Host " - $_" }
    }
    else {
        Write-Host 'No launcher binaries were found for link/copy step.'
    }

    if ($InstallMode -eq 'user' -and $layout.linkDir) {
        if ($osId -ne 'windows') {
            $shellRcUpdated = Set-UnixUserPathExport -HomeDir $HOME -BinDir $layout.linkDir
            if ($shellRcUpdated.Count -gt 0) {
                Write-Host 'Updated shell profile files for PATH:'
                $shellRcUpdated | ForEach-Object { Write-Host " - $_" }
            }
        }
        Write-Host "User-local binary directory: $($layout.linkDir)"
        Write-Host 'Ensure your shell PATH contains the directory above.' -ForegroundColor Yellow
    }
}
finally {
    if (Test-Path -LiteralPath $tempDir) {
        try {
            Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction Stop
        }
        catch {
            Write-Host "Warning: failed to remove temp directory $tempDir . Please remove it manually." -ForegroundColor Yellow
        }
    }
}