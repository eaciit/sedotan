package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/eaciit/dbox"
	_ "github.com/eaciit/dbox/dbc/csv"
	_ "github.com/eaciit/dbox/dbc/json"
	_ "github.com/eaciit/dbox/dbc/mongo"
	"github.com/eaciit/sedotan/sedotan.v2"
	"github.com/eaciit/toolkit"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var (
	_id          string
	configpath   string
	snapshot     string
	snapshotdata Snapshot
	config       toolkit.M
	Log          *toolkit.LogEngine
	thistime     time.Time
)

type Snapshot struct {
	Id             string
	Starttime      string
	Endtime        string
	Grabcount      int
	Rowgrabbed     int
	Errorfound     int
	Lastgrabstatus string //[success|failed]
	Grabstatus     string //[running|done]
}

func init() {
	snapshotdata = Snapshot{}
	thistime = sedotan.TimeNow()
}

func fetchConfig() {
	var result interface{}

	if strings.Contains(configpath, "http") {
		res, err := http.Get(configpath)
		checkfatalerror(err)
		defer res.Body.Close()

		decoder := json.NewDecoder(res.Body)
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
				m, err := toolkit.ToM(each)
				checkfatalerror(err)

				config = m
				isFound = true
			}
		}

		if !isFound {
			checkfatalerror(errors.New(fmt.Sprintf("config with _id %s is not found\n%#v", _id, result)))
		}
	case "map[string]interface {}":
		m, err := toolkit.ToM(result)
		checkfatalerror(err)
		config = m
	default:
		checkfatalerror(errors.New(fmt.Sprintf("invalid config file\n%#v", result)))
	}
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

func getsnapshot() (err error) {
	err = nil

	config := toolkit.M{"newfile": true, "useheader": true, "delimiter": ","}
	conn, err := prepareconnection("csv", snapshot, "", "", "", config)
	if err != nil {
		sedotan.CheckError(errors.New(fmt.Sprintf("Fatal error on get snapshot : %v", err.Error())))
	}
	defer conn.Close()

	csr, err := c.NewQuery().Where(dbox.Eq("_id", _id)).Cursor(nil)
	if err != nil {
		sedotan.CheckError(errors.New(fmt.Sprintf("Fatal error on get snapshot : %v", err.Error())))
		return
	}

	if csr == nil {
		sedotan.CheckError(errors.New(fmt.Sprintf("Fatal error on get snapshot : Cursor not initialized")))
		return
	}
	defer csr.Close()

	err = csr.Fetch(&snapshotdata, 1, false)
	if err != nil {
		sedotan.CheckError(errors.New(fmt.Sprintf("Fatal error on get snapshot : %v", err.Error())))
	}

	return
}

func savesnapshot() (err error) {
	err = nil
	config := toolkit.M{"newfile": true, "useheader": true, "delimiter": ","}
	conn, err := prepareconnection("csv", snapshot, "", "", "", config)
	if err != nil {
		return
	}

	err = conn.NewQuery().SetConfig("multiexec", true).Save().Exec(toolkit.M{}.Set("data", snapshotdata))

	conn.Close()

	return
}

func logexiterror(err error) {
	if err == nil {
		return
	}
	Log.AddLog(fmt.Sprintf("[%v] Found : %v", _id, err.Error()), "ERROR")
	checkfatalerror(err)
}

func checkfatalerror(err error) {
	if err == nil {
		return
	}

	snapshotdata.Errorfound += 1
	snapshotdata.Lastgrabstatus = "failed"
	snapshotdata.Grabstatus = "done"
	e := savesnapshot()

	if e != nil {
		sedotan.CheckError(errors.New(fmt.Sprintf("Fatal error on save snapshot running web grabber : %v", err.Error())))
	}

	sedotan.CheckError(errors.New(fmt.Sprintf("Fatal error on running web grabber : %v", err.Error())))
}

func main() {
	var err error

	flagConfigPath := flag.String("config", "", "config file")
	flagSnapShot := flag.String("snapshot", "", "snapshot filepath")
	flagID := flag.String("id", "", "_id of the config (if array)")

	flag.Parse()
	tconfigPath := toolkit.ToString(*flagConfigPath)
	tsnapshot := toolkit.ToString(*flagSnapShot)
	_id = *flagID

	configpath = strings.Replace(tconfigPath, `"`, "", -1)
	if configpath == "" {
		sedotan.CheckError(errors.New("-config cannot be empty"))
	}

	snapshot := strings.Replace(tsnapshot, `"`, "", -1)
	if snapshot == "" {
		sedotan.CheckError(errors.New("-snapshot cannot be empty"))
	}

	err = getsnapshot()
	if err != nil {
		sedotan.CheckError(errors.New(fmt.Sprintf("get snapshot error found : %v", err.Error())))
	}

	err = fetchConfig()
	checkfatalerror(err)

	logconf := config.Get("logconf", toolkit.M{}).(toolkit.M)
	if !logconf.Has("logpath") || !logconf.Has("filename") || !logconf.Has("filepattern") {
		checkfatalerror(errors.New(fmt.Sprintf("config log is not complete")))
	}

	Log, err = toolkit.NewLog(false, true, logconf["logpath"].(string), (logconf["filename"].(string) + "-%s"), logconf["filepattern"].(string))
	checkfatalerror(err)

	// _, err := sedotan.Process(config)
	// sedotan.CheckError(err)
}
