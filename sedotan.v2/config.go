package sedotan

import (
	"fmt"
	"github.com/eaciit/dbox"
	_ "github.com/eaciit/dbox/dbc/csv"
	_ "github.com/eaciit/dbox/dbc/json"
	_ "github.com/eaciit/dbox/dbc/mongo"
	"github.com/eaciit/toolkit"
	// "io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type DestInfo struct {
	dbox.IConnection
	Collection string
	Desttype   string
}

func Process(config toolkit.M) (toolkit.M, error) {
	data, err := GetData(config)
	if err != nil {
		return nil, err
	}

	err = SaveData(config, data)
	if err != nil {
		return data, err
	}

	return data, nil
}

func GetData(config toolkit.M) (toolkit.M, error) {
	res := toolkit.M{}

	if config["sourcetype"].(string) == "SourceType_Http" {
		grabs, e := PrepareGrabHtmlConfig(config)
		if e != nil {
			return nil, e
		}

		e = grabs.Grab(nil)
		if e != nil {
			return nil, e
		}

		for key, _ := range grabs.Config.DataSettings {
			docs := []toolkit.M{}
			e = grabs.ResultFromHtml(key, &docs)
			if e != nil {
				return nil, e
			}
			res[key] = docs
		}
	} else if config["sourcetype"].(string) == "SourceType_DocExcel" {
		grabs, e := PrepareGrabDocConfig(config)
		if e != nil {
			return nil, e
		}

		for key, _ := range grabs.CollectionSettings {
			docs := []toolkit.M{}
			e = grabs.ResultFromDatabase(config["nameid"].(string), &docs)
			if e != nil {
				return nil, e
			}
			res[key] = docs
		}
	}

	return res, nil
}

func SaveData(config toolkit.M, data toolkit.M) error {
	for _, each := range config["datasettings"].([]interface{}) {
		dataSet, e := toolkit.ToM(each)
		if e != nil {
			return e
		}

		docs := data[dataSet.GetString("name")].([]toolkit.M)

		destInfo, e := PrepareOutputConfig(dataSet)
		if e != nil {
			return e
		}

		// =======================================================
		if destInfo.Desttype == "csv" {
			outputCSVpath := destInfo.IConnection.Info().Host
			if toolkit.IsFileExist(outputCSVpath) {
				// outputCSVcontentBytes, e := ioutil.ReadFile(outputCSVpath)
				// if e != nil {
				// 	return e
				// 	// g.ErrorNotes = fmt.Sprintf("[%s-%s] Connect to destination failed [%s-%s]:%s", g.Name, key, destInfo.Desttype, destInfo.IConnection.Info().Host, e)
				// 	// g.Log.AddLog(g.ErrorNotes, "ERROR")
				// }

				// if strings.TrimSpace(string(outputCSVcontentBytes)) == "" {
				e = os.Remove(outputCSVpath)
				if e != nil {
					return e
				}
				// }
			}
		}

		e = destInfo.IConnection.Connect()
		if e != nil {
			return e
			// g.ErrorNotes = fmt.Sprintf("[%s-%s] Connect to destination failed [%s-%s]:%s", g.Name, key, destInfo.Desttype, destInfo.IConnection.Info().Host, e)
			// g.Log.AddLog(g.ErrorNotes, "ERROR")
		}

		var q dbox.IQuery
		if destInfo.Collection == "" {
			q = destInfo.IConnection.NewQuery().SetConfig("multiexec", true).Save()
		} else {
			q = destInfo.IConnection.NewQuery().SetConfig("multiexec", true).From(destInfo.Collection).Save()
		}

		if destInfo.Desttype == "csv" {
			q = q.Insert()
		}

		xN := 0
		iN := 0
		for _, doc := range docs {
			for key, val := range doc {
				doc[key] = strings.TrimSpace(fmt.Sprintf("%s", val))
			}

			if destInfo.Desttype == "mongo" {
				doc["_id"] = toolkit.GenerateRandomString("1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnpqrstuvwxyz", 32)
			}

			e = q.Exec(toolkit.M{"data": doc})

			if destInfo.Desttype == "mongo" {
				delete(doc, "_id")
			}

			if e != nil {
				return e
				// g.ErrorNotes = fmt.Sprintf("[%s-%s] Unable to insert [%s-%s]:%s", g.Name, key, destInfo.Desttype, destInfo.IConnection.Info().Host, e)
				// g.Log.AddLog(g.ErrorNotes, "ERROR")
				// g.ErrorFound += 1
			} else {
				iN += 1
			}
			xN++
		}
		// g.RowGrabbed += xN
		q.Close()
		destInfo.IConnection.Close()

		// g.Log.AddLog(fmt.Sprintf("[%s-%s] Fetch Data to destination finished with %d record fetch", g.Name, key, xN), "INFO")

		// if g.HistoryPath != "" && g.HistoryRecPath != "" {
		// 	recfile := g.AddRecHistory(key, docs)
		// 	historyservice := toolkit.M{}.Set("datasettingname", key).Set("grabdate", g.LastGrabExe).Set("rowgrabbed", g.RowGrabbed).
		// 		Set("rowsaved", iN).Set("note", g.ErrorNotes).Set("grabstatus", "SUCCESS").Set("recfile", recfile)
		// 	if !(g.LastGrabStat) {
		// 		historyservice.Set("grabstatus", "FAILED")
		// 	}
		// 	g.AddHistory(historyservice)
		// }
		// =======================================================

	}

	return nil
}

func PrepareOutputConfig(dataSet toolkit.M) (DestInfo, error) {
	tempDestInfo := DestInfo{}
	dataToMap, e := toolkit.ToM(dataSet)
	if e != nil {
		return tempDestInfo, e
	}

	connToMap, e := toolkit.ToM(dataToMap["connectioninfo"])
	if e != nil {
		return tempDestInfo, e
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
			return tempDestInfo, e
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
		return tempDestInfo, e
	}

	return tempDestInfo, nil
}

func PrepareGrabHtmlConfig(data toolkit.M) (*Grabber, error) {
	// var e error
	// var gi, ti time.Duration
	// xGrabService := /*sdt.*/ NewGrabService()
	// /*xGrabService.*/ Name = data["nameid"].(string) //"irondcecom"
	// /*xGrabService.*/ Url = data["url"].(string)     //"http://www.dce.com.cn/PublicWeb/MainServlet"

	// /*xGrabService.*/ SourceType = /*sdt.*/ SourceType_HttpHtml

	// grabintervalToInt := toolkit.ToInt(data["grabinterval"], toolkit.RoundingAuto)
	// timeintervalToInt := toolkit.ToInt(data["timeoutinterval"], toolkit.RoundingAuto)
	// if data["intervaltype"].(string) == "seconds" {
	// 	gi = time.Duration(grabintervalToInt) * time.Second
	// 	ti = time.Duration(timeintervalToInt) * time.Second
	// } else if data["intervaltype"].(string) == "minutes" {
	// 	gi = time.Duration(grabintervalToInt) * time.Minute
	// 	ti = time.Duration(timeintervalToInt) * time.Minute
	// } else if data["intervaltype"].(string) == "hours" {
	// 	gi = time.Duration(grabintervalToInt) * time.Hour
	// 	ti = time.Duration(timeintervalToInt) * time.Hour
	// }

	// /*xGrabService.*/ GrabInterval = gi    //* time.Minute
	// /*xGrabService.*/ TimeOutInterval = ti //* time.Minute //time.Hour, time.Minute, time.Second

	// /*xGrabService.*/ TimeOutIntervalInfo = fmt.Sprintf("%v %s", timeintervalToInt, data["intervaltype"] /*"seconds"*/)

	grabConfig := /*sdt.*/ Config{}

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

	/*xGrabService.*/ ServGrabber := /*sdt.*/ NewGrabber( /*xGrabService.*/ data["url"].(string), data["calltype"].(string), &grabConfig)

	// logconfToMap, e := toolkit.ToM(data["logconf"])
	// if e != nil {
	// 	return nil, e
	// }
	// logpath := logconfToMap["logpath"].(string)           //"E:\\data\\vale\\log"
	// filename := logconfToMap["filename"].(string) + "-%s" //"LOG-GRABDCETEST"
	// filepattern := logconfToMap["filepattern"].(string)   //"20060102"

	// logconf, e := toolkit.NewLog(false, true, logpath, filename, filepattern)
	// if e != nil {
	// 	return nil, e
	// }

	// /*xGrabService.*/ Log = logconf

	/*xGrabService.*/ ServGrabber.DataSettings = make(map[string]* /*sdt.*/ DataSetting)
	// /*xGrabService.*/ DestDbox = make(map[string]* /*sdt.*/ DestInfo)

	tempDataSetting := /*sdt.*/ DataSetting{}
	// tempDestInfo := /*sdt.*/ DestInfo{}
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
			tempDataSetting.Column(i, & /*sdt.*/ GrabColumn{Alias: columnToMap["alias"].(string), Selector: columnToMap["selector"].(string)})
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

		/*xGrabService.*/ ServGrabber.DataSettings[dataToMap["name"].(string)] = &tempDataSetting //DATA01 use name in datasettings

		// connToMap, e := toolkit.ToM(dataToMap["connectioninfo"])
		// if e != nil {
		// 	return nil, e
		// }
		// var db, usr, pwd string

		// if hasDb := connToMap.Has("database"); !hasDb {
		// 	db = ""
		// } else {
		// 	db = connToMap["database"].(string)
		// }

		// if hasUser := connToMap.Has("username"); !hasUser {
		// 	usr = ""
		// } else {
		// 	usr = connToMap["username"].(string)
		// }

		// if hasPwd := connToMap.Has("password"); !hasPwd {
		// 	pwd = ""
		// } else {
		// 	pwd = connToMap["username"].(string)
		// }
		// ci := dbox.ConnectionInfo{}
		// ci.Host = connToMap["host"].(string) //"E:\\data\\vale\\Data_GrabIronTest.csv"
		// ci.Database = db
		// ci.UserName = usr
		// ci.Password = pwd

		// if hasSettings := connToMap.Has("settings"); !hasSettings {
		// 	ci.Settings = nil
		// } else {
		// 	settingToMap, e := toolkit.ToM(connToMap["settings"])
		// 	if e != nil {
		// 		return nil, e
		// 	}
		// 	ci.Settings = settingToMap //toolkit.M{}.Set("useheader", settingToMap["useheader"].(bool)).Set("delimiter", settingToMap["delimiter"])
		// }

		// if hasCollection := connToMap.Has("collection"); !hasCollection {
		// 	tempDestInfo.Collection = ""
		// } else {
		// 	tempDestInfo.Collection = connToMap["collection"].(string)
		// }

		// tempDestInfo.Desttype = dataToMap["desttype"].(string)

		// tempDestInfo.IConnection, e = dbox.NewConnection(tempDestInfo.Desttype, &ci)
		// if e != nil {
		// 	return nil, e
		// }

		// /*xGrabService.*/ DestDbox[dataToMap["name"].(string)] = &tempDestInfo

		// //=History===========================================================
		// /*xGrabService.*/ HistoryPath = HistoryPath //"E:\\data\\vale\\history\\"
		// /*xGrabService.*/ HistoryRecPath = HistoryRecPath //"E:\\data\\vale\\historyrec\\"
		// //===================================================================
	}

	return ServGrabber, nil
}

func PrepareGrabDocConfig(data toolkit.M) (*GetDatabase, error) {
	// var e error
	// var gi, ti time.Duration

	// GrabService := /*sdt.*/ NewGrabService()
	// /*GrabService.*/Name = data["nameid"].(string) //"iopriceindices"
	// /*GrabService.*/SourceType = /*sdt.*/ SourceType_DocExcel

	// grabintervalToInt := toolkit.ToInt(data["grabinterval"], toolkit.RoundingAuto)
	// timeintervalToInt := toolkit.ToInt(data["timeoutinterval"], toolkit.RoundingAuto)
	// if data["intervaltype"].(string) == "seconds" {
	// 	gi = time.Duration(grabintervalToInt) * time.Second
	// 	ti = time.Duration(timeintervalToInt) * time.Second
	// } else if data["intervaltype"].(string) == "minutes" {
	// 	gi = time.Duration(grabintervalToInt) * time.Minute
	// 	ti = time.Duration(timeintervalToInt) * time.Minute
	// } else if data["intervaltype"].(string) == "hours" {
	// 	gi = time.Duration(grabintervalToInt) * time.Hour
	// 	ti = time.Duration(timeintervalToInt) * time.Hour
	// }
	// /*GrabService.*/GrabInterval = gi
	// /*GrabService.*/TimeOutInterval = ti //time.Hour, time.Minute, time.Second
	// /*GrabService.*/TimeOutIntervalInfo = fmt.Sprintf("%v %s", timeintervalToInt, data["intervaltype"])

	ServGetData := new(GetDatabase)
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
		/*GrabService.*/ ServGetData, e = /*sdt.*/ NewGetDatabase(ci.Host, grabDataConf["doctype"].(string), &ci)
		if e != nil {
			return nil, e
		}
	} else {
		return nil, e
	}

	// logconfToMap, e := toolkit.ToM(data["logconf"])
	// if e != nil {
	// 	return nil, e
	// }
	// logpath := logconfToMap["logpath"].(string)           //"E:\\data\\vale\\log"
	// filename := logconfToMap["filename"].(string) + "-%s" //"LOG-LOCALXLSX-%s"
	// filepattern := logconfToMap["filepattern"].(string)   //"20060102"

	// logconf, e := toolkit.NewLog(false, true, logpath, filename, filepattern)
	// if e != nil {
	// 	return nil, e
	// }

	// /*GrabService.*/ Log = logconf

	/*GrabService.*/ ServGetData.CollectionSettings = make(map[string]* /*sdt.*/ CollectionSetting)
	// /*GrabService.*/ DestDbox = make(map[string]* /*sdt.*/ DestInfo)

	tempDataSetting := /*sdt.*/ CollectionSetting{}
	// tempDestInfo := /*sdt.*/ DestInfo{}

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
			tempDataSetting.SelectColumn = append(tempDataSetting.SelectColumn, & /*sdt.*/ GrabColumn{Alias: columnToMap["alias"].(string), Selector: columnToMap["selector"].(string)})
		}
		/*GrabService.*/ ServGetData.CollectionSettings[dataToMap["name"].(string)] = &tempDataSetting //DATA01 use name in datasettings

		// // fmt.Println("doctype>", grabDataConf["doctype"])
		// connToMap, e := toolkit.ToM(dataToMap["connectioninfo"])
		// if e != nil {
		// 	return nil, e
		// }
		// var db, usr, pwd string

		// if hasDb := connToMap.Has("database"); !hasDb {
		// 	db = ""
		// } else {
		// 	db = connToMap["database"].(string)
		// }
		// if hasUser := connToMap.Has("username"); !hasUser {
		// 	usr = ""
		// } else {
		// 	usr = connToMap["username"].(string)
		// }

		// if hasPwd := connToMap.Has("password"); !hasPwd {
		// 	pwd = ""
		// } else {
		// 	pwd = connToMap["username"].(string)
		// }
		// ci = dbox.ConnectionInfo{}
		// ci.Host = connToMap["host"].(string) //"localhost:27017"
		// ci.Database = db                     //"valegrab"
		// ci.UserName = usr                    //""
		// ci.Password = pwd                    //""

		// //tempDestInfo.Collection = "iopriceindices"
		// if hasCollection := connToMap.Has("collection"); !hasCollection {
		// 	tempDestInfo.Collection = ""
		// } else {
		// 	tempDestInfo.Collection = connToMap["collection"].(string)
		// }
		// tempDestInfo.Desttype = dataToMap["desttype"].(string) //"mongo"

		// tempDestInfo.IConnection, e = dbox.NewConnection(tempDestInfo.Desttype, &ci)
		// if e != nil {
		// 	return nil, e
		// }

		// /*GrabService.*/ DestDbox[dataToMap["name"].(string)] = &tempDestInfo
		// //=History===========================================================
		// /*GrabService.*/ HistoryPath = HistoryPath //"E:\\data\\vale\\history\\"
		// /*GrabService.*/ HistoryRecPath = HistoryRecPath //"E:\\data\\vale\\historyrec\\"
		// //===================================================================
	}

	return ServGetData, e
}
