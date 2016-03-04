package sedotan

import (
	"fmt"
	"github.com/eaciit/toolkit"
	"os"
	"time"
)

const (
	dateformat string = "YYYY-MM-dd HH-mm-ss"
)

func CheckError(err error) {
	if err == nil {
		return
	}

	fmt.Printf("ERROR! %s\n", err.Error())

	os.Exit(0)
}

func DateToString(tm time.Time) string {
	if tm.IsZero() {
		tm = TimeNow()
	}
	return toolkit.Date2String(tm, dateformat)
}

func StringToDate(sdate string) time.Time {
	return toolkit.String2Date(sdate, dateformat).UTC()
}

func DateMinutePress(tm time.Time) time.Time {
	return toolkit.String2Date(toolkit.Date2String(tm, "YYYY-MM-dd HH-mm"), "YYYY-MM-dd HH-mm").UTC()
}

func DateSecondPress(tm time.Time) time.Time {
	return toolkit.String2Date(toolkit.Date2String(tm, dateformat), dateformat).UTC()
}

func TimeNow() time.Time {
	return time.Now().UTC()
}

func DefaultValue(datatype string) interface{} {

	switch datatype {
	case "int":
		return int(0)
	case "float32":
		return float32(0)
	case "float64":
		return float64(0)
	case "string":
		return ""
	}

	return nil
}
