cd daemon
go build -o %EC_APP_PATH%\daemon\sedotand.exe
echo "sedotand installed on %EC_APP_PATH%\daemon"

cd ..\sedotans
go build -o %EC_APP_PATH%\cli\sedotans.exe
echo "sedotans installed on %EC_APP_PATH%\cli"

cd ..\sedotanw
go build -o %EC_APP_PATH%\cli\sedotanw.exe
echo "sedotanw installed on %EC_APP_PATH%\cli"

copy ..\test\log\daemonsnapshot.csv %EC_DATA_PATH%\daemon\
echo "daemonsnapshot.csv copied to %EC_DATA_PATH%\daemon"
cd ..\