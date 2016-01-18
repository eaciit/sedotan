package controllers

import (
	// "fmt"
	"github.com/eaciit/dbox"
	_ "github.com/eaciit/dbox/dbc/json"
	"github.com/eaciit/knot/knot.v1"
	"github.com/eaciit/sedotan/sedotan.v1/webapps/modules"
	"github.com/eaciit/toolkit"
	"reflect"
	"strings"
)

var (
	filename = wd + "data\\Config\\config_backup.json"
)

type DashboardController struct {
}

func (a *DashboardController) Default(k *knot.WebContext) interface{} {
	k.Config.OutputType = knot.OutputTemplate
	return ""
}

func (a *DashboardController) P(k *knot.WebContext) interface{} {
	k.Config.OutputType = knot.OutputTemplate
	vn := ""
	qs := k.Request.RequestURI
	if qs != "" {
		qss := strings.Split(qs, "?")
		if len(qss) > 1 {
			vn = strings.Split(qss[1], "&")[0]
			if strings.HasSuffix(vn, ".html") == false {
				vn += ".html"
			}
		}
		k.Config.ViewName = vn
	}
	return ""
}

func (a *DashboardController) Griddashboard(k *knot.WebContext) interface{} {
	k.Config.OutputType = knot.OutputJson

	ci := &dbox.ConnectionInfo{filename, "", "", "", nil}
	c, e := dbox.NewConnection("json", ci)
	if e != nil {
		return e
	}

	e = c.Connect()
	if e != nil {
		return e
	}
	defer c.Close()
	csr, e := c.NewQuery().Select("nameid", "url", "grabinterval", "intervaltype", "datasettings").Cursor(nil)
	defer csr.Close()

	result := make([]toolkit.M, 0)
	e = csr.Fetch(&result, 0, false)
	if e != nil {
		return e
	}

	return result
}

func (a *DashboardController) Startservice(k *knot.WebContext) interface{} {
	k.Config.OutputType = knot.OutputJson

	t := struct {
		NameId string
	}{}
	e := k.GetPayload(&t)
	if e != nil {
		return e.Error()
	}

	ds, _ := Getquery(t.NameId)
	er, isRun := modules.Process(ds)
	if er != nil {
		return er.Error()
	}

	return isRun
}

func (a *DashboardController) Stopservice(k *knot.WebContext) interface{} {
	k.Config.OutputType = knot.OutputJson

	t := struct {
		NameId string
	}{}
	e := k.GetPayload(&t)
	if e != nil {
		return e.Error()
	}

	ds, _ := Getquery(t.NameId)
	er, isRun := modules.StopProcess(ds)
	if er != nil {
		return er.Error()
	}

	return isRun
}

func (a *DashboardController) Stat(k *knot.WebContext) interface{} {
	k.Config.OutputType = knot.OutputJson
	t := struct {
		NameId   string
		BtnClick string
	}{}
	e := k.GetPayload(&t)
	if e != nil {
		return e.Error()
	}

	ds, _ := Getquery(t.NameId)
	gs := modules.NewGrabService()
	grabStatus := gs.CheckStat(ds)

	return grabStatus
}

func Getquery(nameid string) ([]interface{}, error) {
	ci := &dbox.ConnectionInfo{filename, "", "", "", nil}
	c, e := dbox.NewConnection("json", ci)
	if e != nil {
		return nil, e
	}

	e = c.Connect()
	if e != nil {
		return nil, e
	}
	defer c.Close()

	csr, e := c.NewQuery().Where(dbox.Eq("nameid", nameid)).Cursor(nil)
	if e != nil {
		return nil, e
	}

	result := make([]interface{}, 0)
	e = csr.Fetch(&result, 0, false)
	if e != nil {
		return nil, e
	}
	return result, nil
}

func (a *DashboardController) Gethistory(k *knot.WebContext) interface{} {
	k.Config.OutputType = knot.OutputJson
	t := struct {
		NameId string
	}{}
	e := k.GetPayload(&t)
	if e != nil {
		return e.Error()
	}

	hm := modules.NewHistory(t.NameId)
	hs := hm.OpenHistory()

	if reflect.ValueOf(hs).Kind() == reflect.String {
		if strings.Contains(hs.(string), "Cannot Open File") {
			return nil
		}
	}

	return hs
}

func (a *DashboardController) Getlog(k *knot.WebContext) interface{} {
	k.Config.OutputType = knot.OutputJson
	t := struct {
		Date   string
		NameId string
	}{}
	e := k.GetPayload(&t)
	if e != nil {
		return e.Error()
	}

	ds, _ := Getquery(t.NameId)

	hs := modules.NewHistory(t.NameId)
	logs := hs.GetLog(ds, t.Date)

	return logs
}
