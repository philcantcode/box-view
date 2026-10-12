param([string]$Version = '0.1.0')
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$repoRoot = Split-Path $PSScriptRoot -Parent
Push-Location $repoRoot
$originalGoos = $env:GOOS
$originalGoarch = $env:GOARCH
$originalCgo = $env:CGO_ENABLED
try {
    if ($Version -notmatch '^\d+\.\d+\.\d+$') { throw 'Version must be semver without a prefix' }
    $escapedVersion = [regex]::Escape($Version)
    if ((Get-Content -LiteralPath model.go -Raw) -notmatch "const Version = `"$escapedVersion`"") { throw 'Release version must match model.Version' }
    if ((Get-Content -LiteralPath flutter/pubspec.yaml -Raw) -notmatch "(?m)^version: $escapedVersion\r?$" ) { throw 'Release version must match Flutter package' }
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go tests failed' }
    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go vet failed' }
    New-Item -ItemType Directory -Force -Path dist | Out-Null
    $env:CGO_ENABLED = '0'
    foreach ($platform in @('windows/amd64','windows/arm64','linux/amd64','linux/arm64','darwin/amd64','darwin/arm64')) {
        $parts = $platform.Split('/')
        $env:GOOS = $parts[0]
        $env:GOARCH = $parts[1]
        $name = "box-view_${Version}_$($parts[0])_$($parts[1])"
        $stage = Join-Path 'dist' $name
        New-Item -ItemType Directory -Force -Path $stage | Out-Null
        $binary = 'box-view'
        if ($parts[0] -eq 'windows') { $binary = 'box-view.exe' }
        go build -trimpath -ldflags '-s -w' -o (Join-Path $stage $binary) ./cmd/box-view
        if ($LASTEXITCODE -ne 0) { throw "Build failed for $platform" }
        Copy-Item -LiteralPath README.md,LICENSE,CHANGELOG.md -Destination $stage
        Copy-Item -LiteralPath docs -Destination $stage -Recurse -Force
        if ($parts[0] -eq 'windows') {
            Compress-Archive -Path "$stage/*" -DestinationPath "dist/$name.zip" -Force
        } else {
            tar -czf "dist/$name.tar.gz" -C $stage .
            if ($LASTEXITCODE -ne 0) { throw "Archive failed for $platform" }
        }
    }
    tar -czf "dist/box-view_${Version}_flutter.tar.gz" -C flutter lib pubspec.yaml README.md LICENSE CHANGELOG.md analysis_options.yaml example/lib example/web example/pubspec.yaml example/pubspec.lock example/README.md
    if ($LASTEXITCODE -ne 0) { throw 'Flutter archive failed' }
    Get-ChildItem -LiteralPath dist -File | Where-Object { $_.Extension -in '.zip','.gz' } |
        Sort-Object Name | ForEach-Object {
            $hash = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
            "$hash  $($_.Name)"
        } | Set-Content -LiteralPath dist/SHA256SUMS -Encoding ascii
    Write-Output "Release artifacts prepared in $repoRoot/dist"
} finally {
    $env:GOOS = $originalGoos
    $env:GOARCH = $originalGoarch
    $env:CGO_ENABLED = $originalCgo
    Pop-Location
}
