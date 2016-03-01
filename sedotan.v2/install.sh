if [ -z "$EC_APP_PATH" ]; then
	echo "\$EC_APP_PATH is not defined!";
	return
fi

cd daemon
go build -o $EC_APP_PATH/daemon/sedotand
echo "sedotand installed on $EC_APP_PATH/daemon"

cd ../sedotans
go build -o $EC_APP_PATH/cli/sedotans
echo "sedotans installed on $EC_APP_PATH/cli"

cd ../sedotanw
go build -o $EC_APP_PATH/cli/sedotanw
echo "sedotanw installed on $EC_APP_PATH/cli"

cp ../test/log/daemonsnapshot.csv $EC_DATA_PATH/daemon/
echo "daemonsnapshot.csv copied to $EC_DATA_PATH/daemon"
cd ../