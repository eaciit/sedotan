package modules

import (
	// "bufio"
	"fmt"
	"github.com/eaciit/cast"
	"github.com/eaciit/dbox"
	_ "github.com/eaciit/dbox/dbc/csv"
	_ "github.com/eaciit/dbox/dbc/json"
	"github.com/eaciit/knot/knot.v1"
	sdt "github.com/eaciit/sedotan/sedotan.v1"
	"github.com/eaciit/toolkit"
	"os"
	f "path/filepath"
	"reflect"
	"strconv"
	// "strings"
	"time"
)

type GrabModule struct {
	nameId, lastGrabTime string
	// gi, ti               time.Time
}

type StatService struct {
	name   string
	status bool
}

var (
	wd = func(dir string, err error) string { return dir + "/../" }(os.Getwd())

	HistoryPath    = wd + f.Join("data", "History") + string(os.PathSeparator)
	HistoryRecPath = wd + f.Join("data", "HistoryRec") + string(os.PathSeparator)
	grabs          *sdt.GrabService
	grabber        *sdt.Grabber
)

func NewGrabService() *GrabModule {
	g := new(GrabModule)
	return g
}

func Process(datas []interface{}) (error, bool) {
	var (
		e error
		// isServRun bool
	)
	for _, v := range datas {
		vToMap, e := toolkit.ToM(v)
		if e != nil {
			return e, false
		}

		if vToMap["sourcetype"].(string) == "SourceType_Http" {
			grabs, e = GrabHtmlConfig(vToMap)
			if e != nil {
				return e, false
			}
		} else if vToMap["sourcetype"].(string) == "SourceType_DocExcel" {
			grabs, e = GrabDocConfig(vToMap)
			if e != nil {
				return e, false
			}
		}

	}

	e, isServRun := StartService(grabs)
	if e != nil {
		return e, false
	}

	return nil, isServRun
}

func GrabHtmlConfig(data toolkit.M) (*sdt.GrabService, error) {
	var e error
	var gi, ti time.Duration
	xGrabService := sdt.NewGrabService()
	xGrabService.Name = data["nameid"].(string) //"irondcecom"
	xGrabService.Url = data["url"].(string)     //"http://www.dce.com.cn/PublicWeb/MainServlet"

	xGrabService.SourceType = sdt.SourceType_HttpHtml

	grabintervalToInt := toolkit.ToInt(data["grabinterval"], toolkit.RoundingAuto)
	timeintervalToInt := toolkit.ToInt(data["timeoutinterval"], toolkit.RoundingAuto)
	if data["intervaltype"].(string) == "seconds" {
		gi = time.Duration(grabintervalToInt) * time.Second
		ti = time.Duration(timeintervalToInt) * time.Second
	} else if data["intervaltype"].(string) == "minutes" {
		gi = time.Duration(grabintervalToInt) * time.Minute
		ti = time.Duration(timeintervalToInt) * time.Minute
	} else if data["intervaltype"].(string) == "hours" {
		gi = time.Duration(grabintervalToInt) * time.Hour
		ti = time.Duration(timeintervalToInt) * time.Hour
	}

	xGrabService.GrabInterval = gi    //* time.Minute
	xGrabService.TimeOutInterval = ti //* time.Minute //time.Hour, time.Minute, time.Second

	xGrabService.TimeOutIntervalInfo = fmt.Sprintf("%v %s", timeintervalToInt, data["intervaltype"] /*"seconds"*/)

	grabConfig := sdt.Config{}

	if data["calltype"].(string) == "POST" {
		dataurl := toolkit.M{}
		for _, grabconf := range data["grabconf"].(map[string]interface{}) {
			grabDataConf, e := toolkit.ToM(grabconf)
			if e != nil {
				return nil, e
			}
			for key, subGrabDataConf := range grabDataConf {
				if reflect.ValueOf(subGrabDataConf).Kind() == reflect.Float64 {
					i := toolkit.ToInt(subGrabDataConf, toolkit.RoundingAuto)
					toString := strconv.Itoa(i)
					dataurl[key] = toString
				} else {
					dataurl[key] = subGrabDataConf
				}
			}
		}

		grabConfig.SetFormValues(dataurl)
	}

	grabDataConf, e := toolkit.ToM(data["grabconf"])
	if e != nil {
		return nil, e
	}

	isAuthType := grabDataConf.Has("authtype")
	if isAuthType {
		grabConfig.AuthType = grabDataConf["authtype"].(string)
		grabConfig.LoginUrl = grabDataConf["loginurl"].(string)   //"http://localhost:8000/login"
		grabConfig.LogoutUrl = grabDataConf["logouturl"].(string) //"http://localhost:8000/logout"

		grabConfig.LoginValues = toolkit.M{}.
			Set("name", grabDataConf["loginvalues"].(map[string]interface{})["name"].(string)).
			Set("password", grabDataConf["loginvalues"].(map[string]interface{})["password"].(string))

	}

	xGrabService.ServGrabber = sdt.NewGrabber(xGrabService.Url, data["calltype"].(string), &grabConfig)

	logconfToMap, e := toolkit.ToM(data["logconf"])
	if e != nil {
		return nil, e
	}
	logpath := logconfToMap["logpath"].(string)           //"E:\\data\\vale\\log"
	filename := logconfToMap["filename"].(string) + "-%s" //"LOG-GRABDCETEST"
	filepattern := logconfToMap["filepattern"].(string)   //"20060102"

	logconf, e := toolkit.NewLog(false, true, logpath, filename, filepattern)
	if e != nil {
		return nil, e
	}

	xGrabService.Log = logconf

	xGrabService.ServGrabber.DataSettings = make(map[string]*sdt.DataSetting)
	xGrabService.DestDbox = make(map[string]*sdt.DestInfo)

	tempDataSetting := sdt.DataSetting{}
	tempDestInfo := sdt.DestInfo{}
	// isCondition := []interface{}{}
	tempFilterCond := toolkit.M{}
	// var condition string

	for _, dataSet := range data["datasettings"].([]interface{}) {
		dataToMap, _ := toolkit.ToM(dataSet)
		tempDataSetting.RowSelector = dataToMap["rowselector"].(string)

		for _, columnSet := range dataToMap["columnsettings"].([]interface{}) {
			columnToMap, e := toolkit.ToM(columnSet)
			if e != nil {
				return nil, e
			}
			i := toolkit.ToInt(columnToMap["index"], toolkit.RoundingAuto)
			tempDataSetting.Column(i, &sdt.GrabColumn{Alias: columnToMap["alias"].(string), Selector: columnToMap["selector"].(string)})
		}

		/*if data["calltype"].(string) == "POST" {
			// orCondition := []interface{}{}
			isRowdeletecond := dataToMap.Has("rowdeletecond")
			fmt.Println("isRowdeletecond>", isRowdeletecond)
			if hasRowdeletecond := dataToMap.Has("rowdeletecond"); hasRowdeletecond {
				for key, rowDeleteMap := range dataToMap["rowdeletecond"].(map[string]interface{}) {
					if key == "$or" || key == "$and" {
						for _, subDataRowDelete := range rowDeleteMap.([]interface{}) {
							for subIndex, getValueRowDelete := range subDataRowDelete.(map[string]interface{}) {
								orCondition = append(orCondition, map[string]interface{}{subIndex: getValueRowDelete})
							}
						}
						condition = key
					}
				}
			}
			// tempDataSetting.RowDeleteCond = toolkit.M{}.Set(condition, orCondition)
			tempFilterCond := toolkit.M{}.Set(condition, orCondition)
			tempDataSetting.SetFilterCond(tempFilterCond)
		}*/

		if hasRowdeletecond := dataToMap.Has("rowdeletecond"); hasRowdeletecond {
			rowToM, e := toolkit.ToM(dataToMap["rowdeletecond"])
			if e != nil {
				return nil, e
			}
			tempFilterCond, e = toolkit.ToM(rowToM.Get("filtercond", nil))
			tempDataSetting.SetFilterCond(tempFilterCond)
		}

		if hasRowincludecond := dataToMap.Has("rowincludecond"); hasRowincludecond {
			rowToM, e := toolkit.ToM(dataToMap["rowincludecond"])
			if e != nil {
				return nil, e
			}

			tempFilterCond, e = toolkit.ToM(rowToM.Get("filtercond", nil))
			tempDataSetting.SetFilterCond(tempFilterCond)
		}

		xGrabService.ServGrabber.DataSettings[dataToMap["name"].(string)] = &tempDataSetting //DATA01 use name in datasettings

		connToMap, e := toolkit.ToM(dataToMap["connectioninfo"])
		if e != nil {
			return nil, e
		}
		var db, usr, pwd string

		if hasDb := connToMap.Has("database"); !hasDb {
			db = ""
		} else {
			db = connToMap["database"].(string)
		}

		if hasUser := connToMap.Has("username"); !hasUser {
			usr = ""
		} else {
			usr = connToMap["username"].(string)
		}

		if hasPwd := connToMap.Has("password"); !hasPwd {
			pwd = ""
		} else {
			pwd = connToMap["username"].(string)
		}
		ci := dbox.ConnectionInfo{}
		ci.Host = connToMap["host"].(string) //"E:\\data\\vale\\Data_GrabIronTest.csv"
		ci.Database = db
		ci.UserName = usr
		ci.Password = pwd

		if hasSettings := connToMap.Has("settings"); !hasSettings {
			ci.Settings = nil
		} else {
			settingToMap, e := toolkit.ToM(connToMap["settings"])
			if e != nil {
				return nil, e
			}
			ci.Settings = settingToMap //toolkit.M{}.Set("useheader", settingToMap["useheader"].(bool)).Set("delimiter", settingToMap["delimiter"])
		}

		if hasCollection := connToMap.Has("collection"); !hasCollection {
			tempDestInfo.Collection = ""
		} else {
			tempDestInfo.Collection = connToMap["collection"].(string)
		}

		tempDestInfo.Desttype = dataToMap["desttype"].(string)

		tempDestInfo.IConnection, e = dbox.NewConnection(tempDestInfo.Desttype, &ci)
		if e != nil {
			return nil, e
		}

		xGrabService.DestDbox[dataToMap["name"].(string)] = &tempDestInfo

		//=History===========================================================
		xGrabService.HistoryPath = HistoryPath       //"E:\\data\\vale\\history\\"
		xGrabService.HistoryRecPath = HistoryRecPath //"E:\\data\\vale\\historyrec\\"
		//===================================================================
	}

	return xGrabService, nil
}

func GrabDocConfig(data toolkit.M) (*sdt.GrabService, error) {
	var e error
	var gi, ti time.Duration

	GrabService := sdt.NewGrabService()
	GrabService.Name = data["nameid"].(string) //"iopriceindices"
	GrabService.SourceType = sdt.SourceType_DocExcel

	grabintervalToInt := toolkit.ToInt(data["grabinterval"], toolkit.RoundingAuto)
	timeintervalToInt := toolkit.ToInt(data["timeoutinterval"], toolkit.RoundingAuto)
	if data["intervaltype"].(string) == "seconds" {
		gi = time.Duration(grabintervalToInt) * time.Second
		ti = time.Duration(timeintervalToInt) * time.Second
	} else if data["intervaltype"].(string) == "minutes" {
		gi = time.Duration(grabintervalToInt) * time.Minute
		ti = time.Duration(timeintervalToInt) * time.Minute
	} else if data["intervaltype"].(string) == "hours" {
		gi = time.Duration(grabintervalToInt) * time.Hour
		ti = time.Duration(timeintervalToInt) * time.Hour
	}
	GrabService.GrabInterval = gi
	GrabService.TimeOutInterval = ti //time.Hour, time.Minute, time.Second
	GrabService.TimeOutIntervalInfo = fmt.Sprintf("%v %s", timeintervalToInt, data["intervaltype"])

	ci := dbox.ConnectionInfo{}

	grabDataConf, e := toolkit.ToM(data["grabconf"])
	if e != nil {
		return nil, e
	}
	isDoctype := grabDataConf.Has("doctype")
	if isDoctype {
		connToMap, e := toolkit.ToM(grabDataConf["connectioninfo"])
		if e != nil {
			return nil, e
		}
		ci.Host = connToMap["host"].(string) //"E:\\data\\sample\\IO Price Indices.xlsm"
		if hasSettings := connToMap.Has("settings"); !hasSettings {
			ci.Settings = nil
		} else {
			settingToMap, e := toolkit.ToM(connToMap["settings"])
			if e != nil {
				return nil, e
			}
			ci.Settings = settingToMap //toolkit.M{}.Set("useheader", settingToMap["useheader"].(bool)).Set("delimiter", settingToMap["delimiter"])
		}
		GrabService.ServGetData, e = sdt.NewGetDatabase(ci.Host, grabDataConf["doctype"].(string), &ci)
	}

	logconfToMap, e := toolkit.ToM(data["logconf"])
	if e != nil {
		return nil, e
	}
	logpath := logconfToMap["logpath"].(string)           //"E:\\data\\vale\\log"
	filename := logconfToMap["filename"].(string) + "-%s" //"LOG-LOCALXLSX-%s"
	filepattern := logconfToMap["filepattern"].(string)   //"20060102"

	logconf, e := toolkit.NewLog(false, true, logpath, filename, filepattern)
	if e != nil {
		return nil, e
	}

	GrabService.Log = logconf

	GrabService.ServGetData.CollectionSettings = make(map[string]*sdt.CollectionSetting)
	GrabService.DestDbox = make(map[string]*sdt.DestInfo)

	tempDataSetting := sdt.CollectionSetting{}
	tempDestInfo := sdt.DestInfo{}

	for _, dataSet := range data["datasettings"].([]interface{}) {
		dataToMap, e := toolkit.ToM(dataSet)
		if e != nil {
			return nil, e
		}
		tempDataSetting.Collection = dataToMap["rowselector"].(string) //"HIST"
		for _, columnSet := range dataToMap["columnsettings"].([]interface{}) {
			columnToMap, e := toolkit.ToM(columnSet)
			if e != nil {
				return nil, e
			}
			tempDataSetting.SelectColumn = append(tempDataSetting.SelectColumn, &sdt.GrabColumn{Alias: columnToMap["alias"].(string), Selector: columnToMap["selector"].(string)})
		}
		GrabService.ServGetData.CollectionSettings[dataToMap["name"].(string)] = &tempDataSetting //DATA01 use name in datasettings

		// fmt.Println("doctype>", grabDataConf["doctype"])
		connToMap, e := toolkit.ToM(dataToMap["connectioninfo"])
		if e != nil {
			return nil, e
		}
		var db, usr, pwd string

		if hasDb := connToMap.Has("database"); !hasDb {
			db = ""
		} else {
			db = connToMap["database"].(string)
		}
		if hasUser := connToMap.Has("username"); !hasUser {
			usr = ""
		} else {
			usr = connToMap["username"].(string)
		}

		if hasPwd := connToMap.Has("password"); !hasPwd {
			pwd = ""
		} else {
			pwd = connToMap["username"].(string)
		}
		ci = dbox.ConnectionInfo{}
		ci.Host = connToMap["host"].(string) //"localhost:27017"
		ci.Database = db                     //"valegrab"
		ci.UserName = usr                    //""
		ci.Password = pwd                    //""

		//tempDestInfo.Collection = "iopriceindices"
		if hasCollection := connToMap.Has("collection"); !hasCollection {
			tempDestInfo.Collection = ""
		} else {
			tempDestInfo.Collection = connToMap["collection"].(string)
		}
		tempDestInfo.Desttype = dataToMap["desttype"].(string) //"mongo"

		tempDestInfo.IConnection, e = dbox.NewConnection(tempDestInfo.Desttype, &ci)
		if e != nil {
			return nil, e
		}

		GrabService.DestDbox[dataToMap["name"].(string)] = &tempDestInfo
		//=History===========================================================
		GrabService.HistoryPath = HistoryPath       //"E:\\data\\vale\\history\\"
		GrabService.HistoryRecPath = HistoryRecPath //"E:\\data\\vale\\historyrec\\"
		//===================================================================
	}

	return GrabService, e
}

func StartService(grabConfig *sdt.GrabService) (error, bool) {

	e := grabConfig.StartService()

	if e != nil {
		return e, grabConfig.ServiceRunningStat
	} else {
		if grabConfig.ServiceRunningStat {
			//s := Insertstatservice(grabConfig.ServiceRunningStat, grabConfig.Name)

			ks := new(knot.Server)
			ks.Log().Info(fmt.Sprintf("==Start '%s' grab service==", grabConfig.Name))
			knot.SharedObject().Set(grabConfig.Name, grabConfig)

			return nil, grabConfig.ServiceRunningStat //s.(*StatService).status
		}

		err, isStopService := StopService(grabConfig)
		if err != nil {
			return err, isStopService
		}
	}

	return nil, grabConfig.ServiceRunningStat

}

/*func Insertstatservice(isServiceRun bool, nameid string) interface{} {
	var addserv interface{}
	if addserv == nil {
		addserv = &StatService{name: nameid, status: isServiceRun}
	} else {

	}
	return addserv
}*/

func StopProcess(datas []interface{}) (error, bool) {
	var (
		e          error
		isStopServ bool
	)
	for _, v := range datas {
		vToMap, _ := toolkit.ToM(v)
		if knot.SharedObject().Get(vToMap["nameid"].(string)) != nil {
			e, isStopServ = StopService(knot.SharedObject().Get(vToMap["nameid"].(string)).(*sdt.GrabService))
			if e != nil {
				return e, false
			}
		} else {
			return e, false
		}
	}

	if !isStopServ {
		isStopServ = true
	} else {
		isStopServ = false
	}

	return nil, isStopServ
}

func StopService(grabConfig *sdt.GrabService) (error, bool) {
	e := grabConfig.StopService()
	if e != nil {
		return e, grabConfig.ServiceRunningStat
	}
	ks := new(knot.Server)
	ks.Log().Info(fmt.Sprintf("==Stop '%s' grab service==", grabConfig.Name))
	return nil, grabConfig.ServiceRunningStat
}

func (g *GrabModule) CheckStat(datas []interface{}) interface{} {
	var (
		grabStatus         interface{}
		lastDate, nextDate string
		//toolkit.M
	)
	var summaryNotes = toolkit.M{} //map[string]interface{}{}
	for _, v := range datas {
		vToMap, e := toolkit.ToM(v)
		if e != nil {
			return e.Error()
		}

		if knot.SharedObject().Get(vToMap["nameid"].(string)) != nil {
			i := knot.SharedObject().Get(vToMap["nameid"].(string)).(*sdt.GrabService)
			tLast := cast.Date2String(i.LastGrabExe, "YYYY/MM/dd HH:mm:ss") //i.LastGrabExe.Format("2006/01/02 15:04:05")
			if tLast != "0001/01/01 00:00:00" {
				lastDate = tLast
			}
			tNext := cast.Date2String(i.NextGrabExe, "YYYY/MM/dd HH:mm:ss") //i.NextGrabExe.Format("2006/01/02 15:04:05")
			if tNext != "0001/01/01 00:00:00" {
				nextDate = tNext
			}

			startdate := cast.Date2String(i.StartDate, "YYYY/MM/dd HH:mm:ss") //i.StartDate.Format("2006/01/02 15:04:05")
			enddate := cast.Date2String(i.EndDate, "YYYY/MM/dd HH:mm:ss")     //i.EndDate.Format("2006/01/02 15:04:05")
			summaryNotes.Set("startDate", startdate)
			summaryNotes.Set("endDate", enddate)
			summaryNotes.Set("grabCount", i.GrabCount)
			summaryNotes.Set("rowGrabbed", i.RowGrabbed)
			summaryNotes.Set("errorFound", i.ErrorFound)

			grabStatus = ReadLog(vToMap["logconf"], i.ServiceRunningStat, i.Name, lastDate, nextDate, i.LastGrabStat, summaryNotes)
		} else {
			summaryNotes.Set("errorFound", 0)
			grabStatus = ReadLog(vToMap["logconf"], false, vToMap["nameid"].(string), "", "", false, summaryNotes)
		}
	}

	return grabStatus
}

func ReadLog(logConf interface{}, isRun bool, name string, lastDate string, nextDate string, lastGrab bool, summaryNotes toolkit.M) interface{} {
	var grabsStatus = map[string]interface{}{}

	grabsStatus["note"] = summaryNotes
	grabsStatus["lastDate"] = lastDate
	grabsStatus["nextDate"] = nextDate
	grabsStatus["grabStat"] = lastGrab
	grabsStatus["name"] = name
	grabsStatus["isRun"] = isRun
	return grabsStatus
}
