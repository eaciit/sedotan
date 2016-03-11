package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/eaciit/dbox"
	_ "github.com/eaciit/dbox/dbc/csv"
	_ "github.com/eaciit/dbox/dbc/json"
	"github.com/eaciit/toolkit"
)

func main() {
	var err error
	var container toolkit.M

	dataset := make([]toolkit.M, 0, 0)

	fReadType := flag.String("readtype", "", "read type sedotan file")                               //snapshot,history,rechistory,logfile,[daemonlog]
	fLocation := flag.String("pathfile", "", "Full path file location include filename and pattern") //support using environment variable EC_DATA_PATH

	fDateTime := flag.String("datetime", "", "Date time for log file")
	fTake := flag.Int("take", 0, "take for limit data")
	fSkip := flag.Int("skip", 0, "skip for limit data")

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
		//snapshot function include dataset
	case "rechistory":
		//rechistory function include dataset
	case "logfile":
		//logfile function include dataset
	case "daemonlog":
		//daemonlog function include dataset
	default:
		err = errors.New(fmt.Sprintf("-readtype cannot empty or get wrong format"))
	}

	container.Set("ERROR", err.Error())
	container.Set("DATA", dataset)

	outputstring := toolkit.JsonString(container)

	fmt.Printf("%s", outputstring)
}
