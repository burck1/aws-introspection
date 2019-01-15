#requires -Version 4.0
[cmdletbinding()]
param ()

$ErrorActionPreference = 'Stop'

$workDir = $PSScriptRoot
$buildRoot = Join-Path $workDir "build"
if (-not (Test-Path $buildRoot)) {
    New-Item -ItemType Directory -Force -Path $buildRoot | Out-Null
}

Push-Location $workDir

$architectures = @(
    'linux/amd64',
    'windows/amd64',
    'darwin/amd64'
)

foreach ($arch in $architectures) {
    Write-Host "Building $arch"
    $env:GOOS, $env:GOARCH = $arch -split '/'
    $out = "build/$arch/introspect"
    if ($env:GOOS -eq 'windows') { $out += '.exe' }
    $sw = [Diagnostics.Stopwatch]::StartNew()
    go build -o $out
    $sw.Stop()
    Write-Host "Took $($sw.Elapsed)"
    if ($LASTEXITCODE -ne 0) { throw "Error calling go build" }
}

# cleanup env vars
Remove-Item env:\GOOS
Remove-Item env:\GOARCH

Pop-Location
