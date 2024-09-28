@echo off
setlocal

for /f "delims=" %%i in ('python --version 2^>nul') do set "PYTHON_VERSION=%%i"
echo %PYTHON_VERSION% | find "Python 3.10" >nul
if %errorlevel% neq 0 (
    echo Python 3.10 is not installed. Downloading and installing Python 3.10...
    curl -o python-3.10.11.exe https://www.python.org/ftp/python/3.10.10/python-3.10.10-amd64.exe
    start /wait python-3.10.11.exe /quiet InstallAllUsers=1 PrependPath=1
) else (
    echo Python 3.10 is already installed.
)

where pip >nul 2>nul
if %errorlevel% neq 0 (
    echo pip is not installed, Downloading and installing pip...
    curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py

    if exist get-pip.py (
        python get-pip.py
        if %errorlevel%==0 (
            echo pip install successfully
        ) else (
            echo pip install failed
            exit /b 1
        )
    ) else (
        echo pip download failed
        exit /b 1
    )
) else (
    echo pip is already installed
)

python -c "import colorama" 2>nul
if %errorlevel% neq 0 (
    echo colorama is not installed. Installing colorama...
    pip install colorama
) else (
    echo colorama is already installed.
)

redis-cli FLUSHALL

python start.py
pause