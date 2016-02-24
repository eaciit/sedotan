package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/eaciit/dbox"
	_ "github.com/eaciit/dbox/dbc/csv"
	_ "github.com/eaciit/dbox/dbc/json"
	"github.com/eaciit/sedotan/sedotan.v2"
	"github.com/eaciit/toolkit"
	"io/ioutil"
	"net/http"
	"strings"
	// "sync"
	// "runtime"
	"os"
	"path/filepath"
	"time"
)

var (
	starttime   time.Time
	configPath  string
	config      []toolkit.M
	snapshot    []Snapshot
	mapsnapshot map[string]Snapshot
	debugMode   bool
	configerr   error
	Log         *toolkit.LogEngine
	thistime    time.Time
	validconfig int
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
	Note           string
}

func init() {
	starttime = time.Now().UTC()
}

func initiate() {
	config = make([]toolkit.M, 0, 0)
	snapshot = make([]Snapshot, 0, 0)
	mapsnapshot = make(map[string]Snapshot, 0)
	configerr = nil
	thistime = sedotan.TimeNow()
	validconfig = 0
}

func fetchconfig() {
	var result interface{}

	if strings.Contains(configPath, "http") {
		res, configerr := http.Get(configPath)
		if configerr != nil {
			return
		}
		defer res.Body.Close()

		decoder := json.NewDecoder(res.Body)
		configerr = decoder.Decode(&result)
	} else {
		bytes, configerr := ioutil.ReadFile(configPath)
		if configerr != nil {
			return
		}

		configerr = json.Unmarshal(bytes, &result)
	}

	if configerr != nil {
		return
	}

	switch toolkit.TypeName(result) {
	case "[]interface {}":
		for i, each := range result.([]interface{}) {
			m, err := toolkit.ToM(each)

			if err == nil {
				err = checkconfig(m)
			}

			if err == nil {
				validconfig += 1
				if m.Get("running", false).(bool) {
					config = append(config, m)
				}
			} else {
				// tstring := fmt.Sprintf("%v;(Config index %d. %v)", configerr, i, err)
				// configerr = errors.New(fmt.Sprintf("%v", tstring))
				Log.AddLog(fmt.Sprintf("Config index %d. %v", i, err), "ERROR")
			}
		}
	case "map[string]interface {}":
		m, err := toolkit.ToM(result)
		if err == nil {
			err = checkconfig(m)
		}

		if err == nil {
			validconfig += 1
			if m.Get("running", false).(bool) {
				config = append(config, m)
			}
		} else {
			Log.AddLog(fmt.Sprintf("Fetch Config Error Found : %v", err), "ERROR")
		}
	default:
		Log.AddLog(fmt.Sprintf("invalid config file\n%#v", result), "ERROR")
	}
}

func checkconfig(cm toolkit.M) (err error) {
	err = nil

	if !cm.Has("_id") {
		err = errors.New(fmt.Sprintf("_id is not found"))
		return
	}

	if !cm.Has("sourcetype") {
		err = errors.New(fmt.Sprintf("sourcetype is not found"))
		return
	}

	if cm.Has("grabconf") {
		_, err = toolkit.ToM(cm["grabconf"])
	} else {
		err = errors.New(fmt.Sprintf("grab config is not found"))
	}

	if err != nil {
		return
	}

	if cm.Has("intervalconf") {
		_, err = toolkit.ToM(cm["intervalconf"])
	} else {
		err = errors.New(fmt.Sprintf("interval configuration is not found"))
	}

	if err != nil {
		return
	}

	if cm.Has("logconf") {
		_, err = toolkit.ToM(cm["logconf"])
	} else {
		err = errors.New(fmt.Sprintf("log configuration is not found"))
	}

	if err != nil {
		return
	}

	if cm.Has("datasettings") {
		if toolkit.TypeName(cm["datasettings"]) != "[]interface {}" {
			err = errors.New(fmt.Sprintf("data settings must be []interface {}"))
		}
	} else {
		err = errors.New(fmt.Sprintf("data settings is not found"))
	}

	if err != nil {
		return
	}

	if cm.Has("histconf") {
		_, err = toolkit.ToM(cm["histconf"])
	} else {
		err = errors.New(fmt.Sprintf("history configuration is not found"))
	}

	if err != nil {
		return
	}
	return
}

func prepareConnectionSnapshot(filepathsnapshot string) (dbox.IConnection, error) {
	config := toolkit.M{"newfile": true, "useheader": true, "delimiter": ","} //for create new file, if you dont need just overwrite "config" with "nil"
	ci := &dbox.ConnectionInfo{filepathsnapshot, "", "", "", config}

	c, e := dbox.NewConnection("csv", ci)
	if e != nil {
		return nil, e
	}

	e = c.Connect()
	if e != nil {
		return nil, e
	}

	return c, nil
}

func getDataSnapShot(filepathsnapshot string) (err error) {
	conn, err := prepareConnectionSnapshot(filepathsnapshot)
	if err != nil {
		err = errors.New(fmt.Sprintf("snapshot connection failed : %v", err.Error()))
		return
	}

	csr, err := conn.NewQuery().Cursor(nil)
	if err != nil {
		err = errors.New(fmt.Sprintf("snapshot connection failed : %v", err.Error()))
		return
	}

	if csr == nil {
		err = errors.New(fmt.Sprintf("Cursor is nil"))
		return
	}

	if csr.Count() > 0 {
		err = csr.Fetch(&snapshot, 0, false)
		if err != nil {
			err = errors.New(fmt.Sprintf("Unable to fetch all: %s \n", err.Error()))
			return
		}

		for _, val := range snapshot {
			mapsnapshot[val.Id] = val
		}
	}

	csr.Close()
	conn.Close()

	return
}

func savesnapshot(id, filepathsnapshot string) (err error) {
	conn, err := prepareConnectionSnapshot(filepathsnapshot)
	if err != nil {
		err = errors.New(fmt.Sprintf("snapshot connection failed : %v", err.Error()))
		return
	}

	if _, f := mapsnapshot[id]; f {
		err = conn.NewQuery().SetConfig("multiexec", true).Save().Exec(toolkit.M{}.Set("data", mapsnapshot[id]))
	}

	conn.Close()

	return
}

//Check Time run and record to snapshot
func checkistimerun(id string, intervalconf toolkit.M) (cond bool) {
	cond = false
	tempss := Snapshot{Id: id,
		Starttime:      sedotan.DateToString(thistime),
		Endtime:        "",
		Grabcount:      0,
		Rowgrabbed:     0,
		Errorfound:     0,
		Lastgrabstatus: "",
		Grabstatus:     "running",
		Note:           ""}

	strintervalconf := intervalconf.Get("starttime", "").(string)
	intervalstart := sedotan.StringToDate(strintervalconf)
	strcronconf := intervalconf.Get("cronconf", "").(string)
	strintervaltype := intervalconf.Get("intervaltype", "").(string)
	grabinterval := toolkit.ToInt(intervalconf.Get("grabinterval", 0), toolkit.RoundingAuto)

	mtkdata := Snapshot{}
	if _, f := mapsnapshot[id]; f {
		mtkdata = mapsnapshot[id]
	}
	// strmtkdatastarttime := mtkdata.Get("starttime", "").(string)
	mtkstarttime := sedotan.StringToDate(mtkdata.Starttime)
	if mtkdata.Lastgrabstatus == "failed" {
		grabinterval = toolkit.ToInt(intervalconf.Get("timeoutinterval", 0), toolkit.RoundingAuto)
	}

	minutetime := sedotan.DateMinutePress(thistime)

	if strintervalconf != "" && intervalstart.Before(thistime) {
		_, fcond := mapsnapshot[id]
		switch {
		case !fcond:
			cond = true
		case intervalstart.After(mtkstarttime):
			cond = true
		case intervalconf.Get("grabinterval", 0).(float64) > 0:
			var durationgrab time.Duration

			switch strintervaltype {
			case "seconds":
				durationgrab = time.Second * time.Duration(grabinterval)
			case "minutes":
				durationgrab = time.Minute * time.Duration(grabinterval)
			case "hours":
				durationgrab = time.Hour * time.Duration(grabinterval)
			}
			nextgrab := mtkstarttime.Add(durationgrab)

			if nextgrab.Before(thistime) {
				cond = true
				tempss.Grabcount = mtkdata.Grabcount + 1
				tempss.Rowgrabbed = mtkdata.Rowgrabbed
				tempss.Errorfound = mtkdata.Errorfound
				tempss.Lastgrabstatus = mtkdata.Lastgrabstatus
			}
		}
	}

	if strcronconf != "" {
		//min hour dayofmonth month dayofweek
		strsplit := strings.Split(strcronconf, " ")
		cond = true

		for i, vtime := range strsplit {
			ivtime := toolkit.ToInt(vtime, toolkit.RoundingAuto)
			switch i {
			case 0:
				if vtime == "*" || thistime.Minute() == ivtime {
					cond = cond && true
				} else {
					cond = false
				}
			case 1:
				if vtime == "*" || thistime.Hour() == ivtime {
					cond = cond && true
				} else {
					cond = false
				}
			case 2:
				if vtime == "*" || thistime.Day() == ivtime {
					cond = cond && true
				} else {
					cond = false
				}
			case 3:
				if vtime == "*" || toolkit.ToInt(thistime.Month(), toolkit.RoundingAuto) == ivtime {
					cond = cond && true
				} else {
					cond = false
				}
			case 4:
				if vtime == "*" || toolkit.ToInt(thistime.Weekday(), toolkit.RoundingAuto) == ivtime {
					cond = cond && true
				} else {
					cond = false
				}
			}
		}

		if mtkdata.Starttime != "" {
			cond = cond && minutetime.After(sedotan.StringToDate(mtkdata.Starttime))
		}

		if cond {
			tempss.Grabcount = mtkdata.Grabcount + 1
			tempss.Rowgrabbed = mtkdata.Rowgrabbed
			tempss.Errorfound = mtkdata.Errorfound
			tempss.Lastgrabstatus = mtkdata.Lastgrabstatus
		}
	}

	if cond {
		mapsnapshot[id] = tempss
	}

	return
}

func checkisonprocess(id string, intervalconf toolkit.M) (cond bool) {
	cond = false
	if _, f := mapsnapshot[id]; !f {
		return
	}

	mtkdata := mapsnapshot[id]
	if mtkdata.Grabstatus == "running" && intervalconf.Get("cronconf", "").(string) == "" {
		cond = true
	}

	return
}

func main() {
	// runtime.GOMAXPROCS(runtime.NumCPU())
	var err error

	flagConfigPath := flag.String("config", "", "config file")
	flagDebugMode := flag.Bool("debug", false, "debug mode")
	flagLogPath := flag.String("logpath", "", "log path")

	flag.Parse()
	tconfigPath := toolkit.ToString(*flagConfigPath)
	tlogPath := toolkit.ToString(*flagLogPath)
	debugMode = *flagDebugMode

	configPath = strings.Replace(tconfigPath, `"`, "", -1)
	if tconfigPath == "" {
		sedotan.CheckError(errors.New("-config cannot be empty"))
	}

	logstdout := false
	logfile := true

	logPath := strings.Replace(tlogPath, `"`, "", -1)
	fmt.Println("Log Path, ", logPath)
	if logPath == "" {
		logPath, err = os.Getwd()
		if err != nil {
			logstdout = true
			logfile = false
			fmt.Println("cannot get log path")
		}
	}

	//Temporary :
	var snapshotpath string = filepath.Join(logPath, "daemonsnapshot.csv")
	// err = getDataSnapShot(snapshotpath)
	// sedotan.CheckError(err)

	Log, err = toolkit.NewLog(logstdout, logfile, logPath, "daemonlog-%s", "20060102")
	sedotan.CheckError(err)

	Log.AddLog(fmt.Sprintf("Start daemon grabbing, config path : %v", configPath), "INFO")

	for {
		err = nil

		daemoninterval := 1 * time.Second
		<-time.After(daemoninterval)

		Log.AddLog(fmt.Sprintf("Run daemon"), "INFO")
		initiate()

		Log.AddLog(fmt.Sprintf("Fetch config grabbing started"), "INFO")
		fetchconfig()
		if configerr != nil {
			Log.AddLog(configerr.Error(), "ERROR")
			configerr = nil
		}

		Log.AddLog(fmt.Sprintf("Get data snapshot"), "INFO")
		err = getDataSnapShot(snapshotpath)
		if err != nil {
			Log.AddLog(fmt.Sprintf("Failed to start grabbing, snapshot error : %v", err.Error()), "ERROR")
			continue
		}

		if len(config) > 0 {
			Log.AddLog(fmt.Sprintf("Ready to start grabbing, found %v valid config and %v active config", validconfig, len(config)), "INFO")
		} else {
			Log.AddLog(fmt.Sprintf("Skip to start grabbing, found %v valid config and 0 active config", validconfig), "ERROR")
		}

		for _, econfig := range config {
			err = nil
			eid := econfig.Get("_id", "").(string)
			Log.AddLog(fmt.Sprintf("Check config for id : %v", eid), "INFO")
			intervalconf, _ := toolkit.ToM(econfig["intervalconf"])

			var isonprocess bool = checkisonprocess(eid, intervalconf)
			var isconfrun bool = econfig.Get("running", false).(bool) //check config status run/stop (isconfrun)
			var istimerun bool = checkistimerun(eid, intervalconf)

			egrabconf, _ := toolkit.ToM(econfig["grabconf"]) // check error in check config function
			etype := egrabconf.Get("sourcetype", "").(string)

			//check grab status onprocess/done/na/error -> conf file / snapshot file ? (isonprocess)
			//check interval+time start/corn schedulling and check last running for interval(istimerun)
			fmt.Printf("!%v && %v && %v \n", isonprocess, isconfrun, istimerun)
			if !isonprocess && isconfrun && istimerun {
				Log.AddLog(fmt.Sprintf("Start grabbing for id : %v", eid), "INFO")
				// save data snapshot using dbox save
				err = savesnapshot(eid, snapshotpath)
				if err != nil {
					Log.AddLog(fmt.Sprintf("Save snapshot id : %v, error found : %v", eid, err), "INFO")
					continue
				}
				// run grabbing
				go func(id string, etype string) {
					etype = strings.ToLower(etype)
					// Check Type [SourceType_HttpHtml|SourceType_HttpJson|SourceType_DocExcel]
					switch {
					case strings.Contains(etype, "http"):
						//Do Get Web
					case strings.Contains(etype, "doc"):
						//Do Get Database
					}
				}(eid, etype)
			} else {
				Log.AddLog(fmt.Sprintf("Skip grabbing for id : %v", eid), "INFO")
			}
		}
	}
	// _, err := sedotan.Process(config)
	// sedotan.CheckError(err)
}
