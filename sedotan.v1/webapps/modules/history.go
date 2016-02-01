package modules

import (
	"bufio"
	"fmt"
	"github.com/eaciit/cast"
	"github.com/eaciit/dbox"
	_ "github.com/eaciit/dbox/dbc/csv"
	"github.com/eaciit/toolkit"
	"os"
	f "path/filepath"
	"strconv"
	"strings"
	"time"
)

type HistoryModule struct {
	filepathName, nameid, logPath string
	humanDate                     string
	rowgrabbed, rowsaved          float64
}

var (
	filepath = wd + f.Join("data", "History") + string(os.PathSeparator)
)

func NewHistory(nameid string) *HistoryModule {
	h := new(HistoryModule)

	dateNow := cast.Date2String(time.Now(), "YYYYMM") //time.Now()
	path := filepath + nameid + "-" + dateNow + ".csv"
	h.filepathName = path
	h.nameid = nameid
	return h
}

func (h *HistoryModule) OpenHistory() interface{} {
	var config = map[string]interface{}{"useheader": true, "delimiter": ",", "dateformat": "MM-dd-YYYY"}
	fmt.Println("======", h.filepathName)
	ci := &dbox.ConnectionInfo{h.filepathName, "", "", "", config}
	c, e := dbox.NewConnection("csv", ci)
	if e != nil {
		return e.Error()
	}

	e = c.Connect()
	if e != nil {
		return e.Error()
	}
	defer c.Close()

	csr, e := c.NewQuery().Select("*").Cursor(nil)
	if e != nil {
		return e.Error()
	}
	if csr == nil {
		return "Cursor not initialized"
	}
	defer csr.Close()
	ds := []toolkit.M{}
	e = csr.Fetch(&ds, 0, false)
	if e != nil {
		return e.Error()
	}

	var history = []interface{}{} //toolkit.M{}
	for i, v := range ds {
		// layout := "2006/01/02 15:04:05"
		castDate := time.Now()
		if v.Has("grabdata") {
			castDate, _ = time.Parse(time.RFC3339, v.Get("grabdate").(string))
		}

		h.humanDate = cast.Date2String(castDate, "YYYY/MM/dd HH:mm:ss")
		h.rowgrabbed, _ = strconv.ParseFloat(fmt.Sprintf("%v", v.Get("rowgrabbed")), 64)
		h.rowsaved, _ = strconv.ParseFloat(fmt.Sprintf("%v", v.Get("rowgrabbed")), 64)

		var addToMap = toolkit.M{}
		addToMap.Set("id", i+1)
		addToMap.Set("datasettingname", v.Get("datasettingname"))
		addToMap.Set("grabdate", h.humanDate)
		addToMap.Set("grabstatus", v.Get("grabstatus"))
		addToMap.Set("rowgrabbed", h.rowgrabbed)
		addToMap.Set("rowsaved", h.rowsaved)
		addToMap.Set("notehistory", v.Get("note"))
		addToMap.Set("recfile", v.Get("recfile"))
		addToMap.Set("nameid", h.nameid)

		history = append(history, addToMap)
	}

	return history
}

func (h *HistoryModule) GetLog(datas []interface{}, date string) interface{} {
	for _, v := range datas {
		vMap, _ := toolkit.ToM(v)

		logConf := vMap["logconf"].(map[string]interface{})
		dateNow := time.Now()
		dateNowFormat := dateNow.Format(logConf["filepattern"].(string))
		fileName := fmt.Sprintf("%s-%s", logConf["filename"], dateNowFormat)
		h.logPath = f.Join(logConf["logpath"].(string), fileName)
	}

	file, err := os.Open(h.logPath)
	if err != nil {
		return err.Error()
	}
	defer file.Close()

	getHours := strings.Split(date, ":")
	containString := getHours[0] + ":" + getHours[1]
	scanner := bufio.NewScanner(file)
	lines := 0
	containLines := 0

	var logs []interface{}
	for scanner.Scan() {
		lines++
		contains := strings.Contains(scanner.Text(), containString)
		if contains {
			containLines = lines
		}

		if lines == containLines {
			logs = append(logs, "<li>"+scanner.Text()+"</li>")
		}
	}

	if err := scanner.Err(); err != nil {
		return err.Error()
	}

	var addSlice = toolkit.M{}
	addSlice.Set("logs", logs)

	return addSlice
}
