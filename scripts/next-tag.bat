@echo off
setlocal EnableExtensions EnableDelayedExpansion

set "BUMP=patch"
set "REMOTE=origin"

if "%~1"=="" goto :args_done
if "%~1"=="-p" set "BUMP=patch"
if "%~1"=="-m" set "BUMP=minor"
if "%~1"=="-M" set "BUMP=major"
if not "%~1"=="-p" if not "%~1"=="-m" if not "%~1"=="-M" goto :usage
if not "%~2"=="" set "REMOTE=%~2"
if not "%~3"=="" goto :usage

:args_done

set "REPO_DIR=%~dp0.."
set "LATEST_TAG="
set "NEXT_TAG="

git -C "%REPO_DIR%" rev-parse --is-inside-work-tree >nul 2>&1
if errorlevel 1 (
    echo Not a git repository: %REPO_DIR% 1>&2
    exit /b 1
)

git -C "%REPO_DIR%" remote get-url "%REMOTE%" >nul 2>&1
if errorlevel 1 (
    echo Remote not found: %REMOTE% 1>&2
    exit /b 1
)

git -C "%REPO_DIR%" fetch "%REMOTE%" --tags
if errorlevel 1 (
    echo Failed to fetch tags from %REMOTE% 1>&2
    exit /b 1
)

for /f "usebackq delims=" %%T in (`git -C "%REPO_DIR%" tag --sort=-v:refname`) do (
    call :normalize_tag "%%~T"
    if not errorlevel 1 (
        set "LATEST_TAG=%%~T"
        goto :found_tag
    )
)

:found_tag
if not defined LATEST_TAG (
    if /I "%BUMP%"=="major" (
        set "NEXT_TAG=v1.0.0"
    ) else if /I "%BUMP%"=="minor" (
        set "NEXT_TAG=v0.1.0"
    ) else (
        set "NEXT_TAG=v0.0.1"
    )
    goto :create_tag
)

call :normalize_tag "%LATEST_TAG%"
if errorlevel 1 (
    echo Failed to parse latest git tag: %LATEST_TAG% 1>&2
    exit /b 1
)

for /f "tokens=1-3 delims=." %%A in ("!NORMALIZED_TAG!") do (
    set /a MAJOR=%%A
    set /a MINOR=%%B
    set /a PATCH=%%C
)

if /I "%BUMP%"=="major" (
    set /a MAJOR+=1
    set /a MINOR=0
    set /a PATCH=0
) else if /I "%BUMP%"=="minor" (
    set /a MINOR+=1
    set /a PATCH=0
) else (
    set /a PATCH+=1
)

set "NEXT_TAG=!TAG_PREFIX!!MAJOR!.!MINOR!.!PATCH!"

:create_tag
git -C "%REPO_DIR%" rev-parse -q --verify "refs/tags/%NEXT_TAG%" >nul 2>&1
if not errorlevel 1 (
    echo Tag already exists: %NEXT_TAG% 1>&2
    exit /b 1
)

git -C "%REPO_DIR%" tag -a "%NEXT_TAG%" -m "Release %NEXT_TAG%"
if errorlevel 1 (
    echo Failed to create tag: %NEXT_TAG% 1>&2
    exit /b 1
)

git -C "%REPO_DIR%" push "%REMOTE%" "%NEXT_TAG%"
if errorlevel 1 (
    echo Failed to push tag %NEXT_TAG% to %REMOTE% 1>&2
    echo Local tag %NEXT_TAG% has been created. Delete it manually if needed. 1>&2
    exit /b 1
)

echo %NEXT_TAG%
exit /b 0

:normalize_tag
set "RAW_TAG=%~1"
set "NORMALIZED_TAG=%RAW_TAG%"
set "TAG_PREFIX="
set "PART1="
set "PART2="
set "PART3="
set "PART4="

if /I "!NORMALIZED_TAG:~0,1!"=="v" (
    set "TAG_PREFIX=v"
    set "NORMALIZED_TAG=!NORMALIZED_TAG:~1!"
)

for /f "tokens=1-4 delims=." %%A in ("!NORMALIZED_TAG!") do (
    set "PART1=%%A"
    set "PART2=%%B"
    set "PART3=%%C"
    set "PART4=%%D"
)

if not defined PART1 exit /b 1
if not defined PART2 exit /b 1
if not defined PART3 exit /b 1
if defined PART4 exit /b 1

echo(!PART1!| findstr /r "^[0-9][0-9]*$" >nul || exit /b 1
echo(!PART2!| findstr /r "^[0-9][0-9]*$" >nul || exit /b 1
echo(!PART3!| findstr /r "^[0-9][0-9]*$" >nul || exit /b 1
exit /b 0

:usage
echo Usage: %~nx0 [-p ^| -m ^| -M] [remote] 1>&2
echo   no args  create and push the next patch tag to origin 1>&2
echo   -p       create and push the next patch tag 1>&2
echo   -m       create and push the next minor tag 1>&2
echo   -M       create and push the next major tag 1>&2
echo   remote   optional remote name, default is origin 1>&2
exit /b 1