$timeout = 10
$job = Start-Job -ScriptBlock {
    Set-Location "d:\dev\Golang\Ferret-Compiler\compiler\cmd"
    go run . "./../../app/cmd/start.fer" -debug
}

if (Wait-Job $job -Timeout $timeout) {
    $result = Receive-Job $job
    Write-Host $result
} else {
    Write-Host "Script timed out after $timeout seconds - likely infinite loop prevented!"
    Stop-Job $job
}

Remove-Job $job
