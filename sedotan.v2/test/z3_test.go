package main

import (
	"github.com/eaciit/toolkit"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"fmt"
	"errors"
	"encoding/json"
	"github.com/eaciit/dbox"
	_ "github.com/eaciit/dbox/dbc/csv"
	_ "github.com/eaciit/dbox/dbc/json"
	_ "github.com/eaciit/dbox/dbc/mongo"
	_ "github.com/eaciit/dbox/dbc/xlsx"
	"github.com/eaciit/sedotan/sedotan.v2"
	"io/ioutil"
	_ "math"
	"net/http"
	"flag"
)

type SourceTypeEnum int

const (
	SourceType_DocExcel SourceTypeEnum = iota
	SourceType_DocMongo
)

type DestInfo struct {
	dbox.IConnection
	collection string
	desttype   string
}

var (
	basePath string = func(dir string, err error) string { return dir }(os.Getwd())
	cmd      *exec.Cmd
	histpath	string
	recpath		string
	filename	string
	filepattern	string
	config       toolkit.M
	SourceType      SourceTypeEnum
	sGrabber        *sedotan.GetDatabase
	destDboxs       map[string]*DestInfo
	histConf        toolkit.M
	extCommand      toolkit.M
	configpath   string
	_id          string
	pid          int
	snapshotdata Snapshot
	thistime     time.Time
	snapshot     string
	EC_APP_PATH  string = os.Getenv("EC_APP_PATH")
	EC_DATA_PATH string = os.Getenv("EC_DATA_PATH")
	tbasepath = strings.Replace(basePath, " ", toolkit.PathSeparator+" ", -1)
)

type Snapshot struct {
	Id             string
	Starttime      string
	Laststartgrab  string
	Lastupdate     string
	Grabcount      int
	Rowgrabbed     int
	Errorfound     int
	Lastgrabstatus string //[success|failed]
	Grabstatus     string //[running|done]
	Cgtotal        int
	Cgprocess      int
	Note           string
	Pid            int
}

func TestBuild(t *testing.T) {
	t.Skip("Skip : Comment this line to do test")
	cmd = exec.Command("cmd", "/C", "go", "build", "../daemon")
	e := cmd.Run()
	// cmd.Wait()

	if e != nil {
		t.Errorf("Error, %s \n", e)
	} else {
		t.Logf("RUN, %s \n", "success")
	}
	// cmd = exec.Command("rm", "-rf", "cli")
	// cmd.Run()
	// cmd.Wait()
}

func TestRunDaemon(t *testing.T) {
	t.Skip("Skip : Comment this line to do test")

	// tbasepath := strings.Replace(basePath, " ", toolkit.PathSeparator+" ", -1)

	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", "..\\daemon\\sedotandaemon.exe", `-config="`+tbasepath+`\config-daemon.json"`, `-logpath="`+tbasepath+`\log"`)
	} else {
		cmd = exec.Command("sudo", "../daemon/sedotandaemon", `-config="`+tbasepath+`\config-daemon.json"`, `-logpath="`+tbasepath+`\log"`)
	}

	err := cmd.Start()
	if err != nil {
		t.Errorf("Error, %s \n", err)
	}

	go func(cmd *exec.Cmd) {
		daemoninterval := 60 * time.Second
		<-time.After(daemoninterval)

		err := cmd.Process.Signal(os.Kill)
		if err != nil {
			toolkit.Println("Error, %s \n", err.Error())
		}

		if runtime.GOOS == "windows" {
			err = exec.Command("cmd", "/C", "taskkill", "/F", "/IM", "sedotandaemon.exe", "/T").Run()
		} else {
			err = exec.Command("sudo", "pkill", "sedotandaemon").Run()
		}

		if err != nil {
			toolkit.Println("Error, %s \n", err.Error())
		}

	}(cmd)

	err = cmd.Wait()
	if err != nil {
		t.Errorf("Error, %s \n", err.Error())
	}
}

func TestRunSedotanw(t *testing.T) {
	t.Skip("Skip : Comment this line to do test")

	tbasepath := strings.Replace(basePath, " ", toolkit.PathSeparator+" ", -1)
	snapshotpath := filepath.Join(tbasepath, "log", "daemonsnapshot.csv")

	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", "..\\sedotanw\\sedotanw.exe", `-config="`+tbasepath+`\config-daemon.json"`, `-snapshot="`+snapshotpath+`"`, `-id="irondcecomcn"`)
	} else {
		cmd = exec.Command("sudo", "../sedotanw/sedotanw", `-config="`+tbasepath+`/config-daemon.json"`, `-snapshot="`+snapshotpath+`"`, `-id="irondcecomcn"`)
	}

	err := cmd.Start()
	if err != nil {
		t.Errorf("Error, %s \n", err)
	}

	// go func(cmd *exec.Cmd) {
	// 	daemoninterval := 60 * time.Second
	// 	<-time.After(daemoninterval)

	// 	err := cmd.Process.Signal(os.Kill)
	// 	if err != nil {
	// 		toolkit.Println("Error, %s \n", err.Error())
	// 	}

	// 	if runtime.GOOS == "windows" {
	// 		err = exec.Command("cmd", "/C", "taskkill", "/F", "/IM", "sedotanw.exe", "/T").Run()
	// 	} else {
	// 		err = exec.Command("sudo", "pkill", "sedotanw").Run()
	// 	}

	// 	if err != nil {
	// 		toolkit.Println("Error, %s \n", err.Error())
	// 	}

	// }(cmd)

	err = cmd.Wait()
	if err != nil {
		t.Errorf("Error, %s \n", err.Error())
	}
}

func fetchConfig() (err error) {

	switch toolkit.ToString(config.Get("sourcetype", "")) {
	case "SourceType_DocExcel":
		SourceType = SourceType_DocExcel
	case "SourceType_DocMongo":
		SourceType = SourceType_DocMongo
	default:
		err = errors.New(fmt.Sprintf("Fetch Config, Source type is not defined : %v", config.Get("sourcetype", "")))
		return
	}

	// LOg.AddLog("Start fetch grabconf", "INFO")
	if !config.Has("grabconf") {
		err = errors.New(fmt.Sprintf("Fetch Config, grabconf not found error"))
		return
	}

	tconfgrab := toolkit.M{}
	tconfgrab, err = toolkit.ToM(config["grabconf"])
	if err != nil {
		err = errors.New(fmt.Sprintf("Fetch Config, grabconf found error : %v", err.Error()))
		return
	}

	if !tconfgrab.Has("doctype") {
		err = errors.New("Fetch Config, doctype not found")
		return
	}

	ci := dbox.ConnectionInfo{}
	mapconninfo := toolkit.M{}

	mapconninfo, err = toolkit.ToM(tconfgrab.Get("connectioninfo", nil))
	if err != nil {
		err = errors.New(fmt.Sprintf("Fetch Config, load connectioninfo found error : %v", err.Error()))
		return
	}

	ci.Host = toolkit.ToString(mapconninfo.Get("host", ""))
	ci.Database = toolkit.ToString(mapconninfo.Get("database", ""))
	ci.UserName = toolkit.ToString(mapconninfo.Get("userName", ""))
	ci.Password = toolkit.ToString(mapconninfo.Get("password", ""))
	ci.Settings, err = toolkit.ToM(mapconninfo.Get("settings", nil))
	if err != nil {
		err = errors.New(fmt.Sprintf("Fetch Config, load connectioninfo.settings found error : %v", err.Error()))
		return
	}

	sGrabber, err = sedotan.NewGetDatabase(ci.Host, toolkit.ToString(tconfgrab.Get("doctype", "")), &ci)
	if err != nil {
		err = errors.New(fmt.Sprintf("Fetch Config, create new get database found error : %v", err.Error()))
	}

	// LOg.AddLog("Start fetch datasettings", "INFO")
	if !config.Has("datasettings") || !(toolkit.TypeName(config["datasettings"]) == "[]interface {}") {
		err = errors.New("Fetch Config, datasettings is not found or have wrong format")
		return
	}

	sGrabber.CollectionSettings = make(map[string]*sedotan.CollectionSetting)
	destDboxs = make(map[string]*DestInfo)

	for _, xVal := range config["datasettings"].([]interface{}) {
		err = nil
		tCollectionSetting := sedotan.CollectionSetting{}
		tDestDbox := DestInfo{}

		mVal := toolkit.M{}
		mVal, err = toolkit.ToM(xVal)
		if err != nil {
			// LOg.AddLog(fmt.Sprintf("[Fetch.Ds.%d] Found : %v", i, err.Error()), "ERROR")
			continue
		}

		tnameid := toolkit.ToString(mVal.Get("nameid", ""))
		if tnameid == "" {
			// LOg.AddLog(fmt.Sprintf("[Fetch.Ds.%d] Data Setting Id is not found", i), "ERROR")
			continue
		}
		tCollectionSetting.Collection = toolkit.ToString(mVal.Get("collection", ""))

		// Fetch mapssettings
		if !mVal.Has("mapssettings") || !(toolkit.TypeName(mVal["mapssettings"]) == "[]interface {}") {
			// LOg.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v] Found : mapssettings is not found or incorrect", i, tnameid), "ERROR")
			continue
		}

		tCollectionSetting.MapsColumns = make([]*sedotan.MapColumn, 0, 0)
		for _, Valcs := range mVal["mapssettings"].([]interface{}) {
			mValcs := toolkit.M{}
			mValcs, err = toolkit.ToM(Valcs)
			if err != nil {
				// LOg.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v.%v] Found : mapssettings is not found or incorrect", i, tnameid, xi), "ERROR")
				continue
			}

			tgrabcolumn := sedotan.MapColumn{}

			tgrabcolumn.Source = toolkit.ToString(mValcs.Get("source", ""))
			tgrabcolumn.SType = toolkit.ToString(mValcs.Get("sourcetype", ""))
			tgrabcolumn.Destination = toolkit.ToString(mValcs.Get("destination", ""))
			tgrabcolumn.DType = toolkit.ToString(mValcs.Get("destinationtype", ""))

			// tindex := toolkit.ToInt(mValcs.Get("index", 0), toolkit.RoundingAuto)
			tCollectionSetting.MapsColumns = append(tCollectionSetting.MapsColumns, &tgrabcolumn)
		}

		//Fetch Filter Condition
		if mVal.Has("filtercond") {
			tfiltercond := toolkit.M{}
			tfiltercond, err = toolkit.ToM(mVal.Get("filtercond", toolkit.M{}))
			if err != nil {
				// LOg.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v] Found : filter cond is incorrect, %v", i, tnameid, err.Error()), "ERROR")
			} else {
				tCollectionSetting.SetFilterCond(tfiltercond)
			}
		}

		//Fetch limit data
		if mVal.Has("limitrow") {
			tLimitrow, err := toolkit.ToM(mVal["limitrow"])
			if err == nil {
				tCollectionSetting.Take = toolkit.ToInt(tLimitrow.Get("take", 0), toolkit.RoundingAuto)
				tCollectionSetting.Skip = toolkit.ToInt(tLimitrow.Get("skip", 0), toolkit.RoundingAuto)
			}
		}

		//Fetch Connection Info
		tConnInfo := toolkit.M{}
		tConnInfo, err = toolkit.ToM(mVal.Get("connectioninfo", toolkit.M{}))
		if err != nil {
			// LOg.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v] Found : %v", i, tnameid, err.Error()), "ERROR")
			continue
		}
		tDestDbox.desttype = toolkit.ToString(mVal.Get("desttype", ""))
		tDestDbox.collection = toolkit.ToString(tConnInfo.Get("collection", ""))

		tHost := toolkit.ToString(tConnInfo.Get("host", ""))
		tDatabase := toolkit.ToString(tConnInfo.Get("database", ""))
		tUserName := toolkit.ToString(tConnInfo.Get("username", ""))
		tPassword := toolkit.ToString(tConnInfo.Get("password", ""))
		tSettings := toolkit.M{}
		tSettings, err = toolkit.ToM(tConnInfo.Get("settings", nil))
		if err != nil {
			// LOg.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v] Connection Setting Found : %v", i, tnameid, err.Error()), "ERROR")
			continue
		}

		tDestDbox.IConnection, err = prepareconnection(tDestDbox.desttype, tHost, tDatabase, tUserName, tPassword, tSettings)
		if err != nil {
			// LOg.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v] Create connection found : %v", i, tnameid, err.Error()), "ERROR")
			continue
		}
		tDestDbox.IConnection.Close()

		destDboxs[tnameid] = &tDestDbox
		sGrabber.CollectionSettings[tnameid] = &tCollectionSetting

	}
	err = nil

	if len(destDboxs) == 0 || len(sGrabber.CollectionSettings) == 0 {
		err = errors.New("Fetch Config, datasettings is not found or have wrong format")
		return
	}

	if !config.Has("histconf") {
		err = errors.New("Fetch Config, history configuration is not found or have wrong format")
		return
	}

	// LOg.AddLog("Start fetch histconf", "INFO")
	histConf, err = toolkit.ToM(config.Get("histconf", nil))
	if err != nil || len(histConf) == 0 || !histConf.Has("histpath") || !histConf.Has("recpath") || !histConf.Has("filename") || !histConf.Has("filepattern") {
		err = errors.New("Fetch Config, history configuration is not found or have wrong format")
		return
	}

	if !config.Has("extcommand") {
		return
	}

	// LOg.AddLog("Start fetch extcommand", "INFO")
	extCommand, _ = toolkit.ToM(config.Get("extcommand", nil))

	return
}

func prepareconnection(driver string, host string, database string, username string, password string, config toolkit.M) (dbox.IConnection, error) {
	ci := &dbox.ConnectionInfo{host, database, username, password, config}

	c, e := dbox.NewConnection(driver, ci)
	if e != nil {
		return nil, e
	}

	e = c.Connect()
	if e != nil {
		return nil, e
	}

	return c, nil
}

func getConfig() (err error) {
	var result interface{}

	if strings.Contains(configpath, "http") {
		res, err := http.Get(configpath)
		checkfatalerror(err)
		defer res.Body.Close()

		decoder := json.NewDecoder(res.Body)
		err = decoder.Decode(&result)
		checkfatalerror(err)
	} else {
		bytes, err := ioutil.ReadFile(configpath)
		checkfatalerror(err)

		err = json.Unmarshal(bytes, &result)
		checkfatalerror(err)
	}

	switch toolkit.TypeName(result) {
	case "[]interface {}":
		isFound := false
		for _, eachRaw := range result.([]interface{}) {
			each := eachRaw.(map[string]interface{})
			if each["_id"].(string) == _id {
				m := toolkit.M{}
				m, err = toolkit.ToM(each)
				checkfatalerror(err)

				config = m
				isFound = true
			}
		}

		if !isFound {
			checkfatalerror(errors.New(fmt.Sprintf("config with _id %s is not found\n%#v", _id, result)))
		}
	case "map[string]interface {}":
		m := toolkit.M{}
		m, err = toolkit.ToM(result)
		checkfatalerror(err)
		config = m
	default:
		checkfatalerror(errors.New(fmt.Sprintf("invalid config file\n%#v", result)))
	}

	return
}

func init() {
	snapshotdata = Snapshot{}
	thistime = sedotan.TimeNow()
}

func checkfatalerror(err error) {

	if err == nil {
		return
	}

	snapshotdata.Errorfound += 1
	if snapshotdata.Pid == pid {
		snapshotdata.Lastgrabstatus = "failed"
		snapshotdata.Grabstatus = "done"
		snapshotdata.Note = fmt.Sprintf("Fatal error on running data grabber : %v", err.Error())
	}

	e := savesnapshot()
	if e != nil {
		sedotan.CheckError(errors.New(fmt.Sprintf("Fatal error on save snapshot running web grabber : %v", err.Error())))
	}

	sedotan.CheckError(errors.New(fmt.Sprintf("Fatal error on running web grabber : %v", err.Error())))
}

func getsnapshot() (err error) {
	err = nil

	config := toolkit.M{"useheader": true, "delimiter": ","}
	conn, err := prepareconnection("csv", snapshot, "", "", "", config)
	fmt.Println(snapshot)
	if err != nil {
		sedotan.CheckError(errors.New(fmt.Sprintf("Fatal error on get snapshot : %v", err.Error())))
	}
	defer conn.Close()

	csr, err := conn.NewQuery().Where(dbox.Eq("Id", _id)).Cursor(nil)
	if err != nil {
		sedotan.CheckError(errors.New(fmt.Sprintf("Fatal error on get snapshot : %v", err.Error())))
		return
	}

	if csr == nil {
		sedotan.CheckError(errors.New(fmt.Sprintf("Fatal error on get snapshot : Cursor not initialized")))
		return
	}
	defer csr.Close()
	// aa := toolkit.M{}
	err = csr.Fetch(&snapshotdata, 1, false)
	if err != nil {
		sedotan.CheckError(errors.New(fmt.Sprintf("Fatal error on get snapshot : %v", err.Error())))
	}

	return
}

func savesnapshot() (err error) {
	err = nil
	cconfig := toolkit.M{"newfile": true, "useheader": true, "delimiter": ","}
	conn, err := prepareconnection("csv", snapshot, "", "", "", cconfig)
	if err != nil {
		return
	}

	snapshotdata.Lastupdate = sedotan.DateToString(sedotan.TimeNow())
	err = conn.NewQuery().SetConfig("multiexec", true).Save().Exec(toolkit.M{}.Set("data", snapshotdata))

	conn.Close()

	return
}

func runconfig() {
	var err error

	flagConfigPath := flag.String("config", "", "config file")
	flagSnapShot := flag.String("snapshot", "", "snapshot filepath")
	flagID := flag.String("id", "", "_id of the config (if array)")
	flagPID := flag.Int("pid", 0, "process id number for identify snapshot active")

	flag.Parse()
	tconfigPath := toolkit.ToString(*flagConfigPath)
	tsnapshot := toolkit.ToString(*flagSnapShot)
	_id = *flagID

	_id = strings.Replace(_id, `"`, "", -1)
	if _id == "" {
		_id = "iopriceindices"
		// sedotan.CheckError(errors.New("-id cannot be empty"))
	}

	configpath = strings.Replace(tconfigPath, `"`, "", -1)
	if configpath == "" {
		configpath = tbasepath+`\config-daemon.json`
		// sedotan.CheckError(errors.New("-config cannot be empty"))
	}

	snapshot = strings.Replace(tsnapshot, `"`, "", -1)
	if snapshot == "" {
		snapshot = tbasepath+"\\log\\daemonsnapshot.csv"
		// sedotan.CheckError(errors.New("-snapshot cannot be empty"))
	}

	err = getsnapshot()
	if err != nil {
		sedotan.CheckError(errors.New(fmt.Sprintf("get snapshot error found : %v", err.Error())))
	}

	pid = *flagPID
	pid = 692
	if pid == 0 {
		sedotan.CheckError(errors.New("-pid cannot be empty or zero value"))
	}

	err = getConfig()
	checkfatalerror(err)

	logconf, _ := toolkit.ToM(config.Get("logconf", toolkit.M{}))
	if !logconf.Has("logpath") || !logconf.Has("filename") || !logconf.Has("filepattern") {
		checkfatalerror(errors.New(fmt.Sprintf("config log is not complete")))
	}

	// logpath := logconf["logpath"].(string)
	// if EC_DATA_PATH != "" {
	// 	logpath = filepath.Join(EC_DATA_PATH, "datagrabber", "log")
	// }

	
	err = fetchConfig()
	checkexiterror(err)
	

	// Log.AddLog("Get the data", "INFO")
	// err = sGrabber.Grab(nil)
	// if err != nil {
	// 	checkexiterror(errors.New(fmt.Sprintf("Grab Failed : %v", err.Error())))
	// }
	// Log.AddLog("Get data grab success", "INFO")


	snapshotdata.Note = ""
	snapshotdata.Lastgrabstatus = "success"
	snapshotdata.Grabstatus = "done"

	if pid == snapshotdata.Pid {
		err = savesnapshot()
		if err != nil {
			checkexiterror(errors.New(fmt.Sprintf("Save snapshot error : %v", err.Error())))
		}
	}

	// Log.AddLog("Finish grab data", "INFO")
}

func checkexiterror(err error) {
	if err == nil {
		return
	}
	// Log.AddLog(fmt.Sprintf("[%v] Found : %v", _id, err.Error()), "ERROR")
	checkfatalerror(err)
}

func TestRunTesting(t *testing.T) {
	// t.Skip("Skip : Comment this line to do test")
	
	runconfig()

 	histConf, err := toolkit.ToM(config.Get("histconf", nil))

 	fmt.Println(toolkit.ToString(histConf["histpath"]))
 	fmt.Println(toolkit.ToString(histConf["recpath"]))
 	fmt.Println(toolkit.ToString(histConf["filename"]))
 	fmt.Println(toolkit.ToString(histConf["filepattern"]))

 	// fullfilename := filepath.Join(toolkit.ToString(histConf["histpath"]), toolkit.ToString(histConf["filename"]))

	if runtime.GOOS == "windows" {
		// cmd = exec.Command("cmd", "/C", "E:\\EACIIT\\src\\github.com\\eaciit\\sedotan\\sedotan.v2\\sedotanread\\main.exe", `-config="`+tbasepath+`\config-daemon.json"`, `-logpath="`+tbasepath+`\log"`, `-readtype="history"`)
		cmd = exec.Command("cmd", "/C", "E:\\EACIIT\\src\\github.com\\eaciit\\sedotan\\sedotan.v2\\sedotanread\\main.exe", `-config="`+tbasepath+`\config-daemon.json"`, `-readtype="history"`, `-pathfile="E:\EACIIT\src\github.com\eaciit\sedotan\sedotan.v2\test\hist\HIST-GRABDCE-20160225.csv"`)
	} else {
		cmd = exec.Command("sudo", "../daemon/sedotandaemon", `-config="`+tbasepath+`\config-daemon.json"`, `-logpath="`+tbasepath+`\log"`)
	}
	// cmd = exec.Command("go run main.go -readtype=history -pathfile=E:/EACIIT\\src\\github.com\\eaciit\\sedotan\\sedotan.v2\\test\\hist\\HIST-GRABDCE-20160225.csv")
	err = cmd.Start()
	if err != nil {
		t.Errorf("Error, %s \n", err)
	}
}

