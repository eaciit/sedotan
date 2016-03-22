package main

import (
	_ "encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/eaciit/dbox"
	_ "github.com/eaciit/dbox/dbc/csv"
	_ "github.com/eaciit/dbox/dbc/json"
	"github.com/eaciit/toolkit"
	"time"
	"github.com/eaciit/cast"
	"strconv"
	"io/ioutil"
	"os"
	"strings"
	"bufio"
	"github.com/eaciit/colony-manager/helper"
)

var (
	fLocation = flag.String("pathfile", "", "Full path file location include filename and pattern") //support using environment variable EC_DATA_PATH
	fReadType = flag.String("readtype", "", "read type sedotan file")
	fNameid = flag.String("nameid", "", "read name id for snapshot and log file") 
	fDateTime = flag.String("datetime", "", "Date time for log file")	 
	tLocation string
	tNameid string
	tDateTime string
)

type Grabber struct {
	filepathName, nameid, logPath string
	humanDate					 string
	rowgrabbed, rowsaved		  float64
}

func main() {
	var err error
	var datastring string
	var datatemp []interface{}
	var logs interface{}
	container := toolkit.M{}
	dataset := make([]toolkit.M, 0, 0)
	// fReadType := flag.String("readtype", "", "read type sedotan file")							   //snapshot,history,rechistory,logfile,[daemonlog]
	// fLocation := flag.String("pathfile", "", "Full path file location include filename and pattern") //support using environment variable EC_DATA_PATH

	// fDateTime := flag.String("datetime", "", "Date time for log file")
	// fTake := flag.Int("take", 0, "take for limit data")
	// fSkip := flag.Int("skip", 0, "skip for limit data")

	flag.Parse()

	tReadType := toolkit.ToString(*fReadType)
	tLocation = toolkit.ToString(*fLocation)
	tNameid = toolkit.ToString(*fNameid)
	tDateTime = toolkit.ToString(*fDateTime)

	//snapshot,history,rechistory,logfile,[daemonlog]

	//=========== Parse other flag ===========
	// HERE
	//========================================

	switch tReadType {
	case "snapshot":
		module := GetDirSnapshot(tNameid)
		SnapShot, err := module.OpenSnapShot(tNameid)
		if err != nil {
			fmt.Sprintf("ERROR: %s", err)
		}
		datastring = toolkit.JsonString(SnapShot)
		err = toolkit.UnjsonFromString(datastring, &dataset)
		container.Set("DATA", dataset)
	case "history":
		module := NewHistory(tLocation)
		datatemp, err = module.OpenHistory()
		if err != nil {
			fmt.Sprintf("ERROR: %s", err)
		}
		datastring = toolkit.JsonString(datatemp)
		err = toolkit.UnjsonFromString(datastring, &dataset)
		container.Set("DATA", dataset)
	case "rechistory":
		var data []toolkit.M
		config := toolkit.M{"useheader": true, "delimiter": ","}
		query := helper.Query("csv", tLocation, "", "", "", config)
		data, err = query.SelectAll("")
		container.Set("DATA", data)
	case "logfile":
		history := NewHistory(tNameid)
		logs = history.GetLogHistory(tLocation, tDateTime)	
		container.Set("DATA", logs)
	case "daemonlog":
		container.Set("DATA", dataset)
		err = errors.New(fmt.Sprintf("-readtype cannot empty or get wrong format"))
	default:
		container.Set("DATA", dataset)
		err = errors.New(fmt.Sprintf("-readtype cannot empty or get wrong format"))
	}

	container.Set("ERROR", err)
	outputstring := toolkit.JsonString(container)

	fmt.Printf("%s", outputstring)
}

func GetLogs() (interface{}, error) {
	filepath := toolkit.ToString(*fLocation)

	bytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	logs := string(bytes)

	return logs, err
}

func (w *Grabber) OpenHistory() ([]interface{}, error) {
	var history = []interface{}{} //toolkit.M{}
	var config = map[string]interface{}{"useheader": true, "delimiter": ",", "dateformat": "MM-dd-YYYY"}

	ci := &dbox.ConnectionInfo{tLocation, "", "", "", config}
	c, err := dbox.NewConnection("csv", ci)
	if err != nil {
		return history, err
	}

	err = c.Connect()
	if err != nil {
		return history, err
	}
	defer c.Close()

	csr, err := c.NewQuery().Select("*").Cursor(nil)
	if err != nil {
		return history, err
	}
	if csr == nil {
		return history, errors.New("Cursor not initialized")
	}
	defer csr.Close()
	ds := []toolkit.M{}
	err = csr.Fetch(&ds, 0, false)
	if err != nil {
		return history, err
	}

	for i, v := range ds {
		castDate, _ := time.Parse(time.RFC3339, v.Get("grabdate").(string))

		w.humanDate = cast.Date2String(castDate, "YYYY/MM/dd HH:mm:ss")
		w.rowgrabbed, _ = strconv.ParseFloat(fmt.Sprintf("%v", v.Get("rowgrabbed")), 64)
		w.rowsaved, _ = strconv.ParseFloat(fmt.Sprintf("%v", v.Get("rowgrabbed")), 64)

		var addToMap = toolkit.M{}
		addToMap.Set("id", i+1)
		addToMap.Set("datasettingname", v.Get("datasettingname"))
		addToMap.Set("grabdate", w.humanDate)
		addToMap.Set("grabstatus", v.Get("grabstatus"))
		addToMap.Set("rowgrabbed", w.rowgrabbed)
		addToMap.Set("rowsaved", w.rowsaved)
		addToMap.Set("notehistory", v.Get("note"))
		addToMap.Set("recfile", v.Get("recfile"))
		addToMap.Set("nameid", w.nameid)

		history = append(history, addToMap)
	}
	return history, nil
}
																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																										
func NewHistory(nameid string) *Grabber {
	w := new(Grabber)

	dateNow := cast.Date2String(time.Now(), "YYYYMMdd") //time.Now()
	path := tLocation + nameid + "-" + dateNow + ".csv"
	w.filepathName = path
	w.nameid = nameid
	return w
}

func GetDirSnapshot(nameid string) *Grabber {
	w := new(Grabber)
	w.filepathName = tLocation
	w.nameid = nameid
	return w
}

func (w *Grabber) OpenSnapShot(Nameid string) ([]interface{}, error) {
	var snapShot = []interface{}{} //toolkit.M{}
	var config = map[string]interface{}{"useheader": true, "delimiter": ",", "dateformat": "MM-dd-YYYY"}
	ci := &dbox.ConnectionInfo{w.filepathName, "", "", "", config}
	c, err := dbox.NewConnection("csv", ci)
	if err != nil {
		return snapShot, err
	}

	err = c.Connect()
	if err != nil {
		return snapShot, err
	}
	defer c.Close()

	csr, err := c.NewQuery().Select("*").Where(dbox.Eq("Id", Nameid)).Cursor(nil)
	if err != nil {
		return snapShot, err
	}
	if csr == nil {
		return snapShot, errors.New("Cursor not initialized")
	}
	defer csr.Close()
	ds := []toolkit.M{}
	err = csr.Fetch(&ds, 0, false)
	if err != nil {
		return snapShot, err
	}
	for _, v := range ds {
		var addToMap = toolkit.M{}
		addToMap.Set("id", v.Get("Id"))
		addToMap.Set("starttime", v.Get("Starttime"))
		addToMap.Set("endtime", v.Get("Endtime"))
		addToMap.Set("grabcount", v.Get("Grabcount"))
		addToMap.Set("rowgrabbed", v.Get("Rowgrabbed"))
		addToMap.Set("errorfound", v.Get("Errorfound"))
		addToMap.Set("lastgrabstatus", v.Get("Lastgrabstatus"))
		addToMap.Set("grabstatus", v.Get("Grabstatus"))
		addToMap.Set("note", v.Get("Note"))

		snapShot = append(snapShot, addToMap)
	}
	return snapShot, nil
}

func (w *Grabber) GetLogHistory(logpath string, date string) interface{} {

	file, err := os.Open(logpath)

	if err != nil {
		return err
	}
	defer file.Close()
	addMinute := toolkit.String2Date(date, "YYYY/MM/dd HH:mm:ss").Add(1 * time.Minute)
	dateAddMinute := toolkit.Date2String(addMinute, "YYYY/MM/dd HH:mm:ss")
	getHours := strings.Split(date, ":")
	getAddMinute := strings.Split(dateAddMinute, ":")
	containString := getHours[0] + ":" + getHours[1]
	containString2 := getAddMinute[0] + ":" + getAddMinute[1]
	scanner := bufio.NewScanner(file)
	lines := 0
	containLines := 0
	logsseparator := ""

	add7Hours := func(s string) string {
		t, _ := time.Parse("2006/01/02 15:04", s)
		t = t.Add(time.Hour * 7)
		return t.Format("2006/01/02 15:04")
	}

	containString = add7Hours(containString)
	containString2 = add7Hours(containString2)

	var logs []interface{}
	for scanner.Scan() {
		lines++
		contains := strings.Contains(scanner.Text(), containString)
		contains2 := strings.Contains(scanner.Text(), containString2)

		if contains {
			containLines = lines
			logsseparator = containString
		}

		if contains2 {
			containLines = lines
			logsseparator = containString2
		}
		result := toolkit.M{}
		if lines == containLines {
			arrlogs := strings.Split(scanner.Text(), logsseparator)
			result.Set("Type", arrlogs[0])
			result.Set("Date", logsseparator+":"+arrlogs[1][1:3])
			result.Set("Desc", arrlogs[1][4:len(arrlogs[1])])
			logs = append(logs, result)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	var addSlice = toolkit.M{}
	addSlice.Set("logs", logs)
	return addSlice
}