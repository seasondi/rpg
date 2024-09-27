@echo off
setlocal
for /f "delims=" %%i in ('python --version 2^>nul') do set "PYTHON_VERSION=%%i"
echo %PYTHON_VERSION% | find "Python 3.10" >nul
if %errorlevel% neq 0 (
    echo Python 3.10 is not installed. Downloading and installing Python 3.10...
    curl -o python-3.10.11.exe https://www.python.org/ftp/python/3.10.11/python-3.10.11-amd64.exe
    start /wait python-3.10.11.exe /quiet InstallAllUsers=1 PrependPath=1
) else (
    echo Python 3.10 is already installed.
)

@REM for /f "delims=" %%i in ('pip --version 2^>nul') do set "PIP_VERSION=%%i"
@REM if %PIP_VERSION%=="" (
@REM     echo pip is not installed. Installing pip...
@REM     python -m ensurepip --upgrade
@REM ) else (
@REM     echo pip is already installed.
@REM )

python -c "import colorama" 2>nul
if %errorlevel% neq 0 (
    echo colorama is not installed. Installing colorama...
    pip install colorama
) else (
    echo colorama is already installed.
)

@echo off reg add HKEY_CURRENT_USERConsole /v QuickEdit /t REG_DWORD /d 00000000 /f

python start.py
pause