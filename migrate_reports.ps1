# PowerShell script for migrating report syntax
# Run this from the compiler directory

Write-Host "Starting report syntax migration..." -ForegroundColor Green

# Function to replace patterns in Go files
function Replace-InGoFiles {
    param(
        [string]$Pattern,
        [string]$Replacement,
        [string]$Description
    )
    
    Write-Host "Replacing $Description..." -ForegroundColor Yellow
    
    Get-ChildItem -Path . -Filter "*.go" -Recurse | ForEach-Object {
        $content = Get-Content $_.FullName -Raw
        if ($content -match "Reports\.Add.*SetLevel") {
            $newContent = $content -replace $Pattern, $Replacement
            if ($content -ne $newContent) {
                # Create backup
                Copy-Item $_.FullName "$($_.FullName).bak"
                # Write new content
                Set-Content -Path $_.FullName -Value $newContent -NoNewline
                Write-Host "  Updated: $($_.FullName)" -ForegroundColor Cyan
            }
        }
    }
}

# Replace SEMANTIC_ERROR
Replace-InGoFiles `
    "\.Reports\.Add\(([^)]+)\)\.SetLevel\(report\.SEMANTIC_ERROR\)" `
    ".Reports.AddSemanticError(`$1)" `
    "SEMANTIC_ERROR patterns"

# Replace CRITICAL_ERROR
Replace-InGoFiles `
    "\.Reports\.Add\(([^)]+)\)\.SetLevel\(report\.CRITICAL_ERROR\)" `
    ".Reports.AddCriticalError(`$1)" `
    "CRITICAL_ERROR patterns"

# Replace SYNTAX_ERROR
Replace-InGoFiles `
    "\.Reports\.Add\(([^)]+)\)\.SetLevel\(report\.SYNTAX_ERROR\)" `
    ".Reports.AddSyntaxError(`$1)" `
    "SYNTAX_ERROR patterns"

# Replace NORMAL_ERROR
Replace-InGoFiles `
    "\.Reports\.Add\(([^)]+)\)\.SetLevel\(report\.NORMAL_ERROR\)" `
    ".Reports.AddError(`$1)" `
    "NORMAL_ERROR patterns"

# Replace WARNING
Replace-InGoFiles `
    "\.Reports\.Add\(([^)]+)\)\.SetLevel\(report\.WARNING\)" `
    ".Reports.AddWarning(`$1)" `
    "WARNING patterns"

# Replace INFO
Replace-InGoFiles `
    "\.Reports\.Add\(([^)]+)\)\.SetLevel\(report\.INFO\)" `
    ".Reports.AddInfo(`$1)" `
    "INFO patterns"

Write-Host "`nMigration complete!" -ForegroundColor Green
Write-Host "Backup files (.bak) have been created." -ForegroundColor Yellow
Write-Host "Run 'Get-ChildItem -Filter `"*.bak`" -Recurse | Remove-Item' to delete backups when satisfied." -ForegroundColor Yellow
