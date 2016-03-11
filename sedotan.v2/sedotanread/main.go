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
)

var (
	fLocation = flag.String("pathfile", "", "Full path file location include filename and pattern") //support using environment variable EC_DATA_PATH
	fReadType = flag.String("readtype", "", "read type sedotan file")   
	tLocation string
)

type Grabber struct {
	filepathName, nameid, logPath string
	humanDate                     string
	rowgrabbed, rowsaved          float64
}

func main() {
	// run test
	// go run main.go -readtype="history" -pathfile="E:\EACIIT\src\github.com\eaciit\sedotan\sedotan.v2\test\hist\HIST-GRABDCE-20160225.csv"

	var err error
	var datastring string
	// var data toolkit.M

	container := toolkit.M{}
	dataset := make([]toolkit.M, 0, 0)
	var datatemp []interface{}

	// fReadType := flag.String("readtype", "", "read type sedotan file")                               //snapshot,history,rechistory,logfile,[daemonlog]
	// fLocation := flag.String("pathfile", "", "Full path file location include filename and pattern") //support using environment variable EC_DATA_PATH

	// fDateTime := flag.String("datetime", "", "Date time for log file")
	// fTake := flag.Int("take", 0, "take for limit data")
	// fSkip := flag.Int("skip", 0, "skip for limit data")

	flag.Parse()

	tReadType := toolkit.ToString(*fReadType)
	//snapshot,history,rechistory,logfile,[daemonlog]

	//=========== Parse other flag ===========
	// HERE
	//========================================

	switch tReadType {
	case "snapshot":
		//snapshot function include dataset
	case "history":
		module := NewHistory(tLocation)
		datatemp, err = module.OpenHistory()
		datastring = toolkit.JsonString(datatemp)
		err = toolkit.UnjsonFromString(datastring, &dataset)
	case "rechistory":
		//rechistory function include dataset
	case "logfile":
		//logfile function include dataset
	case "daemonlog":
		//daemonlog function include dataset
	default:
		err = errors.New(fmt.Sprintf("-readtype cannot empty or get wrong format"))
	}

	container.Set("ERROR", err)
	container.Set("DATA", dataset)

	outputstring := toolkit.JsonString(container)

	fmt.Printf("%s", outputstring)
}

func (w *Grabber) OpenHistory() ([]interface{}, error) {
	var history = []interface{}{} //toolkit.M{}
	var config = map[string]interface{}{"useheader": true, "delimiter": ",", "dateformat": "MM-dd-YYYY"}
	tLocation = toolkit.ToString(*fLocation)
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