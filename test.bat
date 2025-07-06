@echo off
cd compiler
setlocal EnableDelayedExpansion

set "passed=0"
set "failed=0"
set "skipped=0"

:: Run go test -v and capture output
> test_output.tmp (
    go test ./... -v
)

:: Display test output to console
type test_output.tmp

:: Count only indented lines (subtests)
for /f "delims=" %%A in ('findstr /R "^[ ]*--- " test_output.tmp') do (
    set "line=%%A"
    echo !line! | findstr /C:"--- PASS" >nul && set /a passed+=1
    echo !line! | findstr /C:"--- FAIL" >nul && set /a failed+=1
    echo !line! | findstr /C:"--- SKIP" >nul && set /a skipped+=1
)

set /a total=passed+failed+skipped

if "%total%"=="0" (
    echo.
    echo No subtests were run.
    del test_output.tmp
    exit /b
)

set /a percent=100 * passed / total

echo.
echo Passed: %passed% / %total% (%percent%%%)
del test_output.tmp
