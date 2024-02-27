
cd /d %~dp0
cd

set GOOS=linux
go build -ldflags "-s -w" -o ./socket_monitor

::set GOOS=windows
::go build -ldflags "-s -w" -o ./socket_monitor.exe
pause