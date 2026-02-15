@echo off

wsl.exe -e python3 -m http.server -d schemas
