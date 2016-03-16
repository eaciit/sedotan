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
	_ "github.com/eaciit/dbox/dbc/xlsx"
	"github.com/eaciit/sedotan/sedotan.v2"
	"github.com/eaciit/toolkit"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type SourceTypeEnum int

const (
	SourceType_DocExcel SourceTypeEnum = iota
	SourceType_DocMongo
)

var (
	_id          string
	pid          int
	configpath   string
	snapshot     string
	snapshotdata Snapshot
	config       toolkit.M
	thistime     time.Time

	histConf        toolkit.M
	extCommand      toolkit.M
	SourceType      SourceTypeEnum
	sGrabber        *sedotan.GetDatabase
	destDboxs       map[string]*DestInfo
	mapRecHistory   map[string]string
	historyFileName string
	Log             *toolkit.LogEngine

	mutex = &sync.Mutex{}

	EC_APP_PATH  string = os.Getenv("EC_APP_PATH")
	EC_DATA_PATH string = os.Getenv("EC_DATA_PATH")
)

type DestInfo struct {
	dbox.IConnection
	collection string
	desttype   string
}

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
	case "SourceType_DocExcel":
		SourceType = SourceType_DocExcel
	case "SourceType_DocMongo":
		SourceType = SourceType_DocMongo
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

	Log.AddLog("Start fetch datasettings", "INFO")
	if !config.Has("datasettings") || !(toolkit.TypeName(config["datasettings"]) == "[]interface {}") {
		err = errors.New("Fetch Config, datasettings is not found or have wrong format")
		return
	}

	sGrabber.CollectionSettings = make(map[string]*sedotan.CollectionSetting)
	destDboxs = make(map[string]*DestInfo)

	for i, xVal := range config["datasettings"].([]interface{}) {
		err = nil
		tCollectionSetting := sedotan.CollectionSetting{}
		tDestDbox := DestInfo{}

		mVal := toolkit.M{}
		mVal, err = toolkit.ToM(xVal)
		if err != nil {
			Log.AddLog(fmt.Sprintf("[Fetch.Ds.%d] Found : %v", i, err.Error()), "ERROR")
			continue
		}

		tnameid := toolkit.ToString(mVal.Get("nameid", ""))
		if tnameid == "" {
			Log.AddLog(fmt.Sprintf("[Fetch.Ds.%d] Data Setting Id is not found", i), "ERROR")
			continue
		}
		tCollectionSetting.Collection = toolkit.ToString(mVal.Get("collection", ""))

		// Fetch mapssettings
		if !mVal.Has("mapssettings") || !(toolkit.TypeName(mVal["mapssettings"]) == "[]interface {}") {
			Log.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v] Found : mapssettings is not found or incorrect", i, tnameid), "ERROR")
			continue
		}

		tCollectionSetting.MapsColumns = make([]*sedotan.MapColumn, 0, 0)
		for xi, Valcs := range mVal["mapssettings"].([]interface{}) {
			mValcs := toolkit.M{}
			mValcs, err = toolkit.ToM(Valcs)
			if err != nil {
				Log.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v.%v] Found : mapssettings is not found or incorrect", i, tnameid, xi), "ERROR")
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
				Log.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v] Found : filter cond is incorrect, %v", i, tnameid, err.Error()), "ERROR")
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
			Log.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v] Found : %v", i, tnameid, err.Error()), "ERROR")
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
			Log.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v] Connection Setting Found : %v", i, tnameid, err.Error()), "ERROR")
			continue
		}

		tDestDbox.IConnection, err = prepareconnection(tDestDbox.desttype, tHost, tDatabase, tUserName, tPassword, tSettings)
		if err != nil {
			Log.AddLog(fmt.Sprintf("[Fetch.Ds.%d.%v] Create connection found : %v", i, tnameid, err.Error()), "ERROR")
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

	Log.AddLog("Start fetch histconf", "INFO")
	histConf, err = toolkit.ToM(config.Get("histconf", nil))
	if err != nil || len(histConf) == 0 || !histConf.Has("histpath") || !histConf.Has("recpath") || !histConf.Has("filename") || !histConf.Has("filepattern") {
		err = errors.New("Fetch Config, history configuration is not found or have wrong format")
		return
	}

	if !config.Has("extcommand") {
		return
	}

	Log.AddLog("Start fetch extcommand", "INFO")
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

	snapshotdata.Lastupdate = sedotan.DateToString(sedotan.TimeNow())
	err = conn.NewQuery().SetConfig("multiexec", true).Save().Exec(toolkit.M{}.Set("data", snapshotdata))

	conn.Close()

	return
}

func updatesnapshot(iN int, key string) (err error) {
	mutex.Lock()
	err = getsnapshot()
	if err != nil {
		note := fmt.Sprintf("[savedatagrab.%s] Unable to get last snapshot :%s", key, err.Error())
		Log.AddLog(note, "ERROR")
	}

	if pid == snapshotdata.Pid {
		snapshotdata.Cgprocess += iN
	}

	snapshotdata.Rowgrabbed += iN
	err = savesnapshot()

	if err != nil {
		note := fmt.Sprintf("[savedatagrab.%s] Unable to update process in snapshot : %s", key, err.Error())
		Log.AddLog(note, "ERROR")
	}
	mutex.Unlock()

	return
}

func savehistory(dt toolkit.M) (err error) {
	err = nil
	// filename := fmt.Sprintf("%s-%s.csv", toolkit.ToString(histConf.Get("filename", "")), toolkit.Date2String(sedotan.TimeNow(), toolkit.ToString(histConf.Get("filepattern", ""))))
	fullfilename := filepath.Join(toolkit.ToString(histConf.Get("histpath", "")), historyFileName)
	if EC_DATA_PATH != "" {
		fullfilename = filepath.Join(EC_DATA_PATH, "datagrabber", "history", historyFileName)
	}

	cconfig := toolkit.M{"newfile": true, "useheader": true, "delimiter": ","}
	conn, err := prepareconnection("csv", fullfilename, "", "", "", cconfig)
	if err != nil {
		return
	}

	err = conn.NewQuery().SetConfig("multiexec", true).Insert().Exec(toolkit.M{}.Set("data", dt))

	conn.Close()

	return
}

func saverechistory(key string, dt toolkit.M) (err error) {
	err = nil
	fullfilename := filepath.Join(toolkit.ToString(histConf.Get("recpath", "")), mapRecHistory[key])
	if EC_DATA_PATH != "" {
		fullfilename = filepath.Join(EC_DATA_PATH, "datagrabber", "historyrec", mapRecHistory[key])
	}
	// fmt.Println(fullfilename, " - Key - ", key, " - filename - ", mapRecHistory)
	cconfig := toolkit.M{"newfile": true, "useheader": true, "delimiter": ","}
	conn, err := prepareconnection("csv", fullfilename, "", "", "", cconfig)
	if err != nil {
		return
	}

	q := conn.NewQuery().SetConfig("multiexec", true).Insert()
	for k, v := range dt {
		if toolkit.TypeName(v) == "toolkit.M" {
			dt.Set(k, fmt.Sprintf("%v", v))
		}
	}

	err = q.Exec(toolkit.M{}.Set("data", dt))
	conn.Close()

	return
}

func errlogsavehistory(note string, dt toolkit.M) {
	Log.AddLog(note, "ERROR")
	dt = dt.Set("note", note)
	_ = savehistory(dt)
	return
}

func savedatagrab() (err error) {
	var wg, wgstream sync.WaitGroup
	// if len(sGrabber.CollectionSettings) > 0 {
	// 	wg.Add(len(sGrabber.CollectionSettings))
	// }

	mapRecHistory = make(map[string]string, 0)
	historyFileName = fmt.Sprintf("%s-%s.csv", toolkit.ToString(histConf.Get("filename", "")), toolkit.Date2String(sedotan.TimeNow(), toolkit.ToString(histConf.Get("filepattern", ""))))

	for key, _ := range sGrabber.CollectionSettings {
		//set history name
		mapRecHistory[key] = fmt.Sprintf("%s.%s-%s.csv", _id, key, toolkit.Date2String(sedotan.TimeNow(), "YYYYMMddHHmmss"))
		//================
		err = nil
		note := ""
		dt := toolkit.M{}.Set("datasettingname", key).Set("grabdate", thistime).Set("rowgrabbed", 0).
			Set("rowsaved", 0).Set("note", note).Set("grabstatus", "fail").Set("recfile", mapRecHistory[key])

		Log.AddLog(fmt.Sprintf("[savedatagrab.%s] start save data", key), "INFO")
		Log.AddLog(fmt.Sprintf("[savedatagrab.%s] prepare data source", key), "INFO")
		iQ, err := sGrabber.GetQuery(key)
		if err != nil {
			note = fmt.Sprintf("[savedatagrab.%s] Unable to get query data : %s", key, err.Error())
			errlogsavehistory(note, dt)
			continue
		}

		defer sGrabber.CloseConn()
		csr, err := iQ.Cursor(nil)
		if err != nil || csr == nil {
			note = fmt.Sprintf("[savedatagrab.%s] Unable to create cursor or cursor nil to get data : %s", key, err.Error())
			errlogsavehistory(note, dt)
			continue
		}

		Log.AddLog(fmt.Sprintf("[savedatagrab.%s] prepare data souce done", key), "INFO")
		Log.AddLog(fmt.Sprintf("[savedatagrab.%s] prepare destination save", key), "INFO")
		err = destDboxs[key].IConnection.Connect()
		if err != nil {
			note = fmt.Sprintf("[savedatagrab.%s] Unable to connect [%s-%s]:%s", key, destDboxs[key].desttype, destDboxs[key].IConnection.Info().Host, err.Error())
			errlogsavehistory(note, dt)
			continue
		}

		q := destDboxs[key].IConnection.NewQuery().SetConfig("multiexec", true).Save()
		if destDboxs[key].collection != "" {
			q = q.From(destDboxs[key].collection)
		}
		Log.AddLog(fmt.Sprintf("[savedatagrab.%s] prepare destination save done", key), "INFO")

		//Update Total Process
		mutex.Lock()
		Log.AddLog(fmt.Sprintf("[savedatagrab.%s] get snapshot for update total process", key), "INFO")
		err = getsnapshot()
		if err != nil {
			note = fmt.Sprintf("[savedatagrab.%s] Unable to get last snapshot :%s", key, err.Error())
			Log.AddLog(note, "ERROR")
		}

		if pid == snapshotdata.Pid {
			Log.AddLog(fmt.Sprintf("[savedatagrab.%s] update total process data : %v", key, csr.Count()), "INFO")
			snapshotdata.Cgtotal += csr.Count()
			err = savesnapshot()
		}

		if err != nil {
			note = fmt.Sprintf("[savedatagrab.%s] Unable to get last snapshot :%s", key, err.Error())
			Log.AddLog(note, "ERROR")
		}
		mutex.Unlock()

		outtm := make(chan toolkit.M)
		wgstream.Add(1)
		go func(intm chan toolkit.M, sQ dbox.IQuery, key string, dt toolkit.M) {
			streamsavedata(intm, sQ, key, dt)
			wgstream.Done()
		}(outtm, q, key, dt)

		wg.Add(1)
		go func(outtm chan toolkit.M, csr dbox.ICursor) {
			condition := true

			for condition {
				results := make([]toolkit.M, 0)
				err = csr.Fetch(&results, 1000, false)
				if err != nil {
					note = fmt.Sprintf("[savedatagrab.%s] Unable to fetch data :%s", key, err.Error())
					Log.AddLog(note, "ERROR")
					condition = false
					continue
				}

				for _, result := range results {
					xresult := toolkit.M{}
					for _, column := range sGrabber.CollectionSettings[key].MapsColumns {
						var val, tval interface{}
						strsplits := strings.Split(column.Source, "|")

						ttm := toolkit.M{}
						lskey := ""
						for i, strsplit := range strsplits {
							lskey = strsplit
							if i == 0 && result.Has(strsplit) {
								ttm.Set(strsplit, result[strsplit])
							} else if ttm.Has(strsplit) {
								ttm = toolkit.M{}.Set(strsplit, ttm[strsplit])
							}
						}

						strsplits = strings.Split(column.Destination, "|")
						if ttm.Has(lskey) {
							tval = ttm[lskey]
						} else {
							tval = sedotan.DefaultValue(column.DType)
						}

						// if len(strsplits) > 1 {
						// 	if xresult.Has(strsplits[0]) {
						// 		val = xre
						// 	} else {

						// 	}
						// }

						tm := toolkit.M{}
						if xresult.Has(strsplits[0]) {
							tm, _ = toolkit.ToM(xresult[strsplits[0]])
						}
						// xval = val.Set(strsplits[1], getresultobj(strsplits[1:], tval, tm))
						switch column.DType {
						case "int":
							tval = toolkit.ToInt(tval, toolkit.RoundingAuto)
						case "float":
							tval = toolkit.ToFloat64(tval, 2, toolkit.RoundingAuto)
						case "string":
							tval = toolkit.ToString(tval)
						default:
							tval = tval
						}

						val = getresultobj(strsplits, tval, tm)

						// if len(strsplits) == 1 {
						// 	val = tval
						// } else {
						// 	val = getresultobj(strsplits[1:], tval)
						// }

						// if len(strsplits) > 1 && {

						// }
						xresult.Set(strsplits[0], val)

						// for i, strsplit := range strsplits {
						// 	if i == 0 && ttm.Has(lskey) {
						// 		xresult.Set(strsplit, ttm[lskey])
						// 	} else if (i + 1) == len(strsplits) {

						// 	}
						// 	// else if ttm.Has(strsplit) {
						// 	// 	ttm = toolkit.M{}.Set(strsplit, ttm[strsplit])
						// 	// }
						// }

					}
					outtm <- xresult
				}

				if len(results) < 1000 || err != nil {
					condition = false
				}
			}
			//
			//

			// ms := []toolkit.M{}
			// for _, val := range results {
			// 	m := toolkit.M{}
			// 	for _, column := range g.CollectionSettings[dataSettingId].MapsColumns {
			// 		m.Set(column.Source, "")
			// 		if val.Has(column.Destination) {
			// 			m.Set(column.Source, val[column.Destination])
			// 		}
			// 	}
			// 	ms = append(ms, m)
			// }

			close(outtm)
			wg.Done()
		}(outtm, csr)

	}

	wgstream.Wait()
	wg.Wait()

	return
}

func getresultobj(strsplits []string, tval interface{}, val toolkit.M) interface{} {
	var xval interface{}
	if val == nil {
		val = toolkit.M{}
	}

	switch {
	case len(strsplits) == 1:
		xval = tval
	case len(strsplits) > 1:
		tm := toolkit.M{}
		if val.Has(strsplits[1]) {
			tm, _ = toolkit.ToM(val[strsplits[1]])
		}
		xval = val.Set(strsplits[1], getresultobj(strsplits[1:], tval, tm))
	}

	return xval
	// if  {
	// 	val = tval
	// } else {
	// 	val = val.Set(k, v)
	// 	if val == nil {

	// 	} else {

	// 	}

	// 	val = getresultobj(strsplits[1:], tval)
	// }

	// return val
}

func streamsavedata(intms <-chan toolkit.M, sQ dbox.IQuery, key string, dt toolkit.M) {
	var err error
	iN, note := 0, ""

	for intm := range intms {
		if destDboxs[key].desttype == "mongo" {
			intm.Set("_id", toolkit.GenerateRandomString("", 32))
		}

		if len(intm) == 0 {
			continue
		}
		//Pre Execute Program
		if extCommand.Has("pre") && toolkit.ToString(extCommand["pre"]) != "" {
			sintm := toolkit.JsonString(intm)
			arrcmd := make([]string, 0, 0)

			// if runtime.GOOS == "windows" {
			// 	arrcmd = append(arrcmd, "cmd")
			// 	arrcmd = append(arrcmd, "/C")
			// }

			arrcmd = append(arrcmd, toolkit.ToString(extCommand["pre"]))
			arrcmd = append(arrcmd, sintm)

			// output, err := toolkit.RunCommand(arrcmd[0], arrcmd[1:])
			output, err := toolkit.RunCommand(arrcmd[0], arrcmd[1])
			if err != nil {
				Log.AddLog(fmt.Sprintf("[savedatagrab.%s] Unable to execute pre external command :%s", key, err.Error()), "ERROR")
				note = "Error Found"
				continue
			}

			err = toolkit.UnjsonFromString(output, &intm)
			if err != nil {
				Log.AddLog(fmt.Sprintf("[savedatagrab.%s] Unable to get pre external command output :%s", key, err.Error()), "ERROR")
				note = "Error Found"
				continue
			}
		}

		err = sQ.Exec(toolkit.M{
			"data": intm,
		})

		if err != nil {
			Log.AddLog(fmt.Sprintf("[savedatagrab.%s] Unable to insert data [%s-%s]:%s", key, "csv", destDboxs[key].IConnection.Info().Host, err.Error()), "ERROR")
			note = "Error Found"
			continue
		}

		err = saverechistory(key, intm)
		if err != nil {
			Log.AddLog(fmt.Sprintf("[savedatagrab.%s] Unable to insert record data [%s-%s]:%s", key, "csv", destDboxs[key].IConnection.Info().Host, err.Error()), "ERROR")
			note = "Error Found"
		}

		iN += 1
		if math.Mod(float64(iN), 100) == 0 {
			_ = updatesnapshot(iN, key)
			dt = dt.Set("rowsaved", (toolkit.ToInt(dt.Get("rowsaved", 0), toolkit.RoundingAuto) + iN))
			iN = 0
		}

		//Post Execute Program
		if extCommand.Has("post") {
			sintm := toolkit.JsonString(intm)
			arrcmd := make([]string, 0, 0)

			// if runtime.GOOS == "windows" {
			// 	arrcmd = append(arrcmd, "cmd")
			// 	arrcmd = append(arrcmd, "/C")
			// }

			arrcmd = append(arrcmd, toolkit.ToString(extCommand["post"]))
			arrcmd = append(arrcmd, sintm)

			// output, err := toolkit.RunCommand(arrcmd[0], arrcmd[1:])
			output, err := toolkit.RunCommand(arrcmd[0], arrcmd[1])
			if err != nil {
				Log.AddLog(fmt.Sprintf("[savedatagrab.%s] Unable to execute post external command :%s", key, err.Error()), "ERROR")
				note = "Error Found"
				continue
			}

			err = toolkit.UnjsonFromString(output, &intm)
			if err != nil {
				Log.AddLog(fmt.Sprintf("[savedatagrab.%s] Unable to get post external command output :%s", key, err.Error()), "ERROR")
				note = "Error Found"
				continue
			}
		}
	}
	dt = dt.Set("note", note).
		Set("grabstatus", "done").
		Set("rowsaved", (toolkit.ToInt(dt.Get("rowsaved", 0), toolkit.RoundingAuto) + iN))
	_ = updatesnapshot(iN, key)
	err = savehistory(dt)
	if err != nil {
		Log.AddLog(fmt.Sprintf("[savedatagrab.%s] Unable to save history : %s", key), "ERROR")
	}
	Log.AddLog(fmt.Sprintf("[savedatagrab.%s] Finish save data", key), "INFO")
	destDboxs[key].IConnection.Close()
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

func main() {
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

	pid = *flagPID
	if pid == 0 {
		sedotan.CheckError(errors.New("-pid cannot be empty or zero value"))
	}

	err = getConfig()
	checkfatalerror(err)

	logconf, _ := toolkit.ToM(config.Get("logconf", toolkit.M{}))
	if !logconf.Has("logpath") || !logconf.Has("filename") || !logconf.Has("filepattern") {
		checkfatalerror(errors.New(fmt.Sprintf("config log is not complete")))
	}

	logpath := logconf["logpath"].(string)
	if EC_DATA_PATH != "" {
		logpath = filepath.Join(EC_DATA_PATH, "datagrabber", "log")
	}

	Log, err = toolkit.NewLog(false, true, logpath, (logconf["filename"].(string) + "-%s"), logconf["filepattern"].(string))
	checkfatalerror(err)
	Log.AddLog("Starting web grab data", "INFO")

	Log.AddLog("Start fetch the config", "INFO")
	err = fetchConfig()
	checkexiterror(err)
	Log.AddLog(fmt.Sprintf("Data grabber created : %v", toolkit.JsonString(sGrabber)), "INFO")
	Log.AddLog("Fetch the config success", "INFO")

	// Log.AddLog("Get the data", "INFO")
	// err = sGrabber.Grab(nil)
	// if err != nil {
	// 	checkexiterror(errors.New(fmt.Sprintf("Grab Failed : %v", err.Error())))
	// }
	// Log.AddLog("Get data grab success", "INFO")

	Log.AddLog("Start save data grab", "INFO")
	err = savedatagrab()
	if err != nil {
		checkexiterror(errors.New(fmt.Sprintf("Save data finish with error : %v", err.Error())))
	}
	Log.AddLog("Save data grab success", "INFO")

	snapshotdata.Note = ""
	snapshotdata.Lastgrabstatus = "success"
	snapshotdata.Grabstatus = "done"

	if pid == snapshotdata.Pid {
		err = savesnapshot()
		if err != nil {
			checkexiterror(errors.New(fmt.Sprintf("Save snapshot error : %v", err.Error())))
		}
	}

	Log.AddLog("Finish grab data", "INFO")
}
