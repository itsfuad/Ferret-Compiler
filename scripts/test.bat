@echo off
cd ..\compiler
setlocal EnableDelayedExpansion

set "passed=0"
set "failed=0"
set "skipped=0"

:: Run go test and parse each output line
for /f "delims=" %%A in ('go test ./... -v 2^>nul') do (
    set "line=%%A"
    call :CheckLine
)

set /a total=passed+failed+skipped

echo.
echo Passed : %passed%
echo Failed : %failed%
echo Skipped: %skipped%
echo Total  : %total%

if %total%==0 (
    echo.
    echo No subtests were run.
    exit /b
)

:: Calculate percentage of passed tests
set /a percent=100 * passed / total
echo Success: %percent%%%

exit /b

:: Subroutine to check and count test results
:CheckLine
setlocal EnableDelayedExpansion
set "check=!line!"

if not "!check:--- PASS=!"=="!check!" (
    endlocal & set /a passed+=1 & goto :eof
)
if not "!check:--- FAIL=!"=="!check!" (
    endlocal & set /a failed+=1 & goto :eof
)
if not "!check:--- SKIP=!"=="!check!" (
    endlocal & set /a skipped+=1 & goto :eof
)

endlocal
goto :eof
