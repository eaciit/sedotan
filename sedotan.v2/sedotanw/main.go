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
	"path/filepath"
	"strings"
	"time"
)

type SourceTypeEnum int

const (
	SourceType_HttpHtml SourceTypeEnum = iota
	SourceType_HttpJson
)

var (
	_id          string
	configpath   string
	snapshot     string
	snapshotdata Snapshot
	config       toolkit.M
	thistime     time.Time

	histConf   toolkit.M
	SourceType SourceTypeEnum
	wGrabber   *sedotan.Grabber
	destDboxs  map[string]*DestInfo
	Log        *toolkit.LogEngine
)

type DestInfo struct {
	dbox.IConnection
	collection string
	desttype   string
}

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
	snapshotdata = Snapshot{}
	thistime = sedotan.TimeNow()
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

func fetchConfig() (err error) {

	switch toolkit.ToString(config.Get("sourcetype", "")) {
	case "SourceType_HttpHtml":
		SourceType = SourceType_HttpHtml
	case "SourceType_HttpJson":
		SourceType = SourceType_HttpJson
	default:
		err = errors.New(fmt.Sprintf("Fetch Config, Source type is not defined : %v", config.Get("sourcetype", "")))
		return
	}

	Log.AddLog("Start fetch grabconf", "INFO")
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

	grabconfig := sedotan.Config{}
	if tconfgrab.Has("formvalues") {
		tfromvalues := toolkit.M{}
		tfromvalues, err = toolkit.ToM(tconfgrab["formvalues"])
		if err != nil {
			err = errors.New(fmt.Sprintf("Fetch Config, formvalues found error : %v", err.Error()))
			return
		}
		grabconfig.SetFormValues(tfromvalues)
	}

	if tconfgrab.Has("loginvalues") {
		grabconfig.LoginValues, err = toolkit.ToM(tconfgrab["loginvalues"])
	}

	if err != nil {
		err = errors.New(fmt.Sprintf("Fetch Config, loginvalues found error : %v", err.Error()))
		return
	}

	grabconfig.URL = toolkit.ToString(tconfgrab.Get("url", ""))
	grabconfig.CallType = toolkit.ToString(tconfgrab.Get("calltype", ""))

	grabconfig.AuthType = toolkit.ToString(tconfgrab.Get("authtype", ""))
	grabconfig.AuthUserId = toolkit.ToString(tconfgrab.Get("authuserid", ""))
	grabconfig.AuthPassword = toolkit.ToString(tconfgrab.Get("authpassword", ""))

	grabconfig.LoginUrl = toolkit.ToString(tconfgrab.Get("loginurl", ""))
	grabconfig.LogoutUrl = toolkit.ToString(tconfgrab.Get("logouturl", ""))

	Log.AddLog(fmt.Sprintf("Done fetch grabconf : %v", toolkit.JsonString(grabconfig)), "INFO")

	wGrabber = sedotan.NewGrabber(grabconfig.URL, grabconfig.CallType, &grabconfig)

	Log.AddLog("Start fetch datasettings", "INFO")
	if !config.Has("datasettings") || !(toolkit.TypeName(config["datasettings"]) == "[]interface {}") {
		err = errors.New("Fetch Config, datasettings is not found or have wrong format")
		return
	}

	wGrabber.DataSettings = make(map[string]*sedotan.DataSetting)
	destDboxs = make(map[string]*DestInfo)

	for i, xVal := range config["datasettings"].([]interface{}) {
		err = nil
		tDataSetting := sedotan.DataSetting{}
		tDestDbox := DestInfo{}

		mVal := toolkit.M{}
		mVal, err = toolkit.ToM(xVal)
		if err != nil {
			Log.AddLog(fmt.Sprintf("[Fetch.Ds.%d] Found : %v", i, err.Error()), "ERROR")
			continue
		}

		t_id := toolkit.ToString(mVal.Get("_id", ""))
		if t_id == "" {
			Log.AddLog(fmt.Sprintf("[Fetch.Ds.%d] Data Setting Id is not found", i), "ERROR")
			continue
		}
		tDataSetting.RowSelector = toolkit.ToString(mVal.Get("rowselector", ""))

		// Fetch columnsettings
		if !mVal.Has("columnsettings") || !(toolkit.TypeName(mVal["columnsettings"]) == "[]interface {}") {
			Log.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v] Found : columnsettings is not found or incorrect", i, t_id), "ERROR")
			continue
		}

		tDataSetting.ColumnSettings = make([]*sedotan.GrabColumn, 0, 0)
		for xi, Valcs := range mVal["columnsettings"].([]interface{}) {
			mValcs := toolkit.M{}
			mValcs, err = toolkit.ToM(Valcs)
			if err != nil {
				Log.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v.%v] Found : columnsettings is not found or incorrect", i, t_id, xi), "ERROR")
				continue
			}

			tgrabcolumn := sedotan.GrabColumn{}
			tgrabcolumn.Alias = toolkit.ToString(mValcs.Get("alias", ""))
			tgrabcolumn.Selector = toolkit.ToString(mValcs.Get("selector", ""))
			tgrabcolumn.ValueType = toolkit.ToString(mValcs.Get("valuetype", ""))
			tgrabcolumn.AttrName = toolkit.ToString(mValcs.Get("attrname", ""))

			tindex := toolkit.ToInt(mValcs.Get("index", 0), toolkit.RoundingAuto)
			tDataSetting.Column(tindex, &tgrabcolumn)
		}

		//Fetch Filter Condition
		if mVal.Has("filtercond") {
			tfiltercond := toolkit.M{}
			tfiltercond, err = toolkit.ToM(mVal.Get("filtercond", toolkit.M{}))
			if err != nil {
				Log.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v] Found : filter cond is incorrect, %v", i, t_id, err.Error()), "ERROR")
			} else {
				tDataSetting.SetFilterCond(tfiltercond)
			}
		}

		//Fetch Connection Info
		tConnInfo := toolkit.M{}
		tConnInfo, err = toolkit.ToM(mVal.Get("connectioninfo", toolkit.M{}))
		if err != nil {
			Log.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v] Found : %v", i, t_id, err.Error()), "ERROR")
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
			Log.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v] Connection Setting Found : %v", i, t_id, err.Error()), "ERROR")
			continue
		}

		tDestDbox.IConnection, err = prepareconnection(tDestDbox.desttype, tHost, tDatabase, tUserName, tPassword, tSettings)
		if err != nil {
			Log.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v] Create connection found : %v", i, t_id, err.Error()), "ERROR")
			continue
		}
		tDestDbox.IConnection.Close()

		destDboxs[t_id] = &tDestDbox
		wGrabber.DataSettings[t_id] = &tDataSetting

	}
	err = nil

	if len(destDboxs) == 0 || len(wGrabber.DataSettings) == 0 {
		err = errors.New("Fetch Config, datasettings is not found or have wrong format")
		return
	}

	if !config.Has("histconf") {
		err = errors.New("Fetch Config, history configuration is not found or have wrong format")
		return
	}

	Log.AddLog("Start fetch histconf", "INFO")
	histConf, err = toolkit.ToM(config.Get("histconf", nil))
	if err != nil || len(histConf) == 0 || !histConf.Has("histpath") || !histConf.Has("recpath") || !histConf.Has("filename") || !histConf.Has("filepattern") {
		err = errors.New("Fetch Config, history configuration is not found or have wrong format")
		return
	}

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

func getsnapshot() (err error) {
	err = nil

	config := toolkit.M{"useheader": true, "delimiter": ","}
	conn, err := prepareconnection("csv", snapshot, "", "", "", config)
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

	snapshotdata.Endtime = sedotan.DateToString(sedotan.TimeNow())
	err = conn.NewQuery().SetConfig("multiexec", true).Save().Exec(toolkit.M{}.Set("data", snapshotdata))

	conn.Close()

	return
}

func savehistory(dt toolkit.M) (err error) {
	err = nil
	filename := fmt.Sprintf("%s-%s.csv", toolkit.ToString(histConf.Get("filename", "")), toolkit.Date2String(sedotan.TimeNow(), toolkit.ToString(histConf.Get("filepattern", ""))))
	fullfilename := filepath.Join(toolkit.ToString(histConf.Get("histpath", "")), filename)

	cconfig := toolkit.M{"newfile": true, "useheader": true, "delimiter": ","}
	conn, err := prepareconnection("csv", fullfilename, "", "", "", cconfig)
	if err != nil {
		return
	}

	err = conn.NewQuery().SetConfig("multiexec", true).Insert().Exec(toolkit.M{}.Set("data", dt))

	conn.Close()

	return
}

func saverechistory(key string, dts []toolkit.M) (fullfilename string, err error) {
	err = nil
	filename := fmt.Sprintf("%s.%s-%s.csv", _id, key, toolkit.Date2String(sedotan.TimeNow(), "YYYYMMddHHmmss"))
	fullfilename = filepath.Join(toolkit.ToString(histConf.Get("recpath", "")), filename)

	cconfig := toolkit.M{"newfile": true, "useheader": true, "delimiter": ","}
	conn, err := prepareconnection("csv", fullfilename, "", "", "", cconfig)
	if err != nil {
		return
	}

	q := conn.NewQuery().SetConfig("multiexec", true).Insert()
	for _, dt := range dts {
		err = q.Exec(toolkit.M{}.Set("data", dt))
	}

	conn.Close()

	return
}

func savedatagrab() (err error) {
	for key, _ := range wGrabber.Config.DataSettings {

		err = nil
		note := ""
		dt := toolkit.M{}.Set("datasettingname", key).Set("grabdate", thistime).Set("rowgrabbed", 0).
			Set("rowsaved", 0).Set("note", note).Set("grabstatus", "fail").Set("recfile", "")

		Log.AddLog(fmt.Sprintf("[savedatagrab.%s] start save data", key), "INFO")
		docs := []toolkit.M{}
		err = wGrabber.ResultFromHtml(key, &docs)
		if err != nil {
			note = fmt.Sprintf("[savedatagrab.%s] Unable to get data : %s", key, err.Error())
			Log.AddLog(note, "ERROR")
			dt = dt.Set("note", note)
			_ = savehistory(dt)
			continue
		}

		dt = dt.Set("rowgrabbed", len(docs))

		err = destDboxs[key].IConnection.Connect()
		if err != nil {
			note = fmt.Sprintf("[savedatagrab.%s] Unable to connect [%s-%s]:%s", key, destDboxs[key].desttype, destDboxs[key].IConnection.Info().Host, err.Error())
			Log.AddLog(note, "ERROR")
			dt = dt.Set("note", note)
			_ = savehistory(dt)
			continue
		}

		q := destDboxs[key].IConnection.NewQuery().SetConfig("multiexec", true).Save()
		if destDboxs[key].collection != "" {
			q = q.From(destDboxs[key].collection)
		}

		iN := 0
		for _, doc := range docs {
			if destDboxs[key].desttype == "mongo" {
				doc["_id"] = toolkit.GenerateRandomString("", 32)
			}

			err = q.Exec(toolkit.M{
				"data": doc,
			})

			if err != nil {
				note = "Error in insert data"
				Log.AddLog(fmt.Sprintf("[savedatagrab.%s] Unable to insert [%s-%s]:%s", key, destDboxs[key].desttype, destDboxs[key].IConnection.Info().Host, err.Error()), "ERROR")
				continue
			} else {
				iN += 1
			}
		}

		dt = dt.Set("rowsaved", iN)

		filerec, err := saverechistory(key, docs)
		if err != nil {
			if note != "" {
				note = note + ";" + fmt.Sprintf("[savedatagrab.%s] Unable to save rec history : %s", key, err.Error())
			} else {
				note = fmt.Sprintf("[savedatagrab.%s] Unable to save rec history : %s", key, err.Error())
			}

			Log.AddLog(fmt.Sprintf("[savedatagrab.%s] Unable to save rec history : %s", key, err.Error()), "ERROR")
			dt = dt.Set("note", note)
			_ = savehistory(dt)
			continue
		}

		dt = dt.Set("note", note).Set("grabstatus", "done").Set("recfile", filerec)
		err = savehistory(dt)
		if err != nil {
			Log.AddLog(fmt.Sprintf("[savedatagrab.%s] Unable to save history : %s", key), "ERROR")
		}
		snapshotdata.Rowgrabbed += iN
		Log.AddLog(fmt.Sprintf("[savedatagrab.%s] Finish save data", key), "INFO")
	}
	return
}

func checkexiterror(err error) {
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
	snapshotdata.Note = fmt.Sprintf("Fatal error on running web grabber : %v", err.Error())

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

	_id = strings.Replace(_id, `"`, "", -1)
	if _id == "" {
		sedotan.CheckError(errors.New("-id cannot be empty"))
	}

	configpath = strings.Replace(tconfigPath, `"`, "", -1)
	if configpath == "" {
		sedotan.CheckError(errors.New("-config cannot be empty"))
	}

	snapshot = strings.Replace(tsnapshot, `"`, "", -1)
	if snapshot == "" {
		sedotan.CheckError(errors.New("-snapshot cannot be empty"))
	}

	err = getsnapshot()
	if err != nil {
		sedotan.CheckError(errors.New(fmt.Sprintf("get snapshot error found : %v", err.Error())))
	}

	err = getConfig()
	checkfatalerror(err)

	logconf, _ := toolkit.ToM(config.Get("logconf", toolkit.M{}))
	if !logconf.Has("logpath") || !logconf.Has("filename") || !logconf.Has("filepattern") {
		checkfatalerror(errors.New(fmt.Sprintf("config log is not complete")))
	}

	Log, err = toolkit.NewLog(false, true, logconf["logpath"].(string), (logconf["filename"].(string) + "-%s"), logconf["filepattern"].(string))
	checkfatalerror(err)
	Log.AddLog("Starting web grab data", "INFO")

	Log.AddLog("Start fetch the config", "INFO")
	err = fetchConfig()
	checkexiterror(err)
	Log.AddLog(fmt.Sprintf("Data grabber created : %v", toolkit.JsonString(wGrabber)), "INFO")
	Log.AddLog("Fetch the config success", "INFO")

	Log.AddLog("Get the data", "INFO")
	err = wGrabber.Grab(nil)
	if err != nil {
		checkexiterror(errors.New(fmt.Sprintf("Grab Failed : %v", err.Error())))
	}
	Log.AddLog("Get data grab success", "INFO")

	Log.AddLog("Start save data grab", "INFO")
	err = savedatagrab()
	if err != nil {
		checkexiterror(errors.New(fmt.Sprintf("Save data finish with error : %v", err.Error())))
	}
	Log.AddLog("Save data grab success", "INFO")

	snapshotdata.Note = ""
	snapshotdata.Lastgrabstatus = "success"
	snapshotdata.Grabstatus = "done"

	err = savesnapshot()
	if err != nil {
		checkexiterror(errors.New(fmt.Sprintf("Save snapshot error : %v", err.Error())))
	}

	Log.AddLog("Finish grab data", "INFO")
}
