package controllers

import (
	"fmt"
	"github.com/eaciit/dbox"
	"github.com/eaciit/knot/knot.v1"
	tk "github.com/eaciit/toolkit"
	"strings"
)

type ResultController struct {
}

func (a *ResultController) Default(k *knot.WebContext) interface{} {
	k.Config.OutputType = knot.OutputTemplate
	return ""
}

func (a *ResultController) P(k *knot.WebContext) interface{} {
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

func (a *ResultController) GetData(k *knot.WebContext) interface{} {
	k.Config.OutputType = knot.OutputJson

	filename := wd + "data\\Config\\config.json"
	ci := &dbox.ConnectionInfo{filename, "", "", "", nil}
	c, e := dbox.NewConnection("json", ci)
	defer c.Close()
	e = c.Connect()
	csr, e := c.NewQuery().Select("*").Cursor(nil)
	defer csr.Close()
	data := []tk.M{}
	e = csr.Fetch(&data, 0, false)
	if e != nil {
		fmt.Println("Found : ", e)
	}
	if e != nil {
		return e.Error()
	} else {
		return data
	}
}

func (a *ResultController) GetDataFromMongo(k *knot.WebContext) interface{} {
	k.Config.OutputType = knot.OutputJson
	d := struct {
		Host       string
		Database   string
		Collection string
	}{}
	e := k.GetPayload(&d)
	ci := &dbox.ConnectionInfo{d.Host, d.Database, "", "", nil}
	c, e := dbox.NewConnection("mongo", ci)
	defer c.Close()
	e = c.Connect()
	csr, e := c.NewQuery().Select().From(d.Collection).
		Cursor(nil)
	defer csr.Close()
	data := []tk.M{}
	e = csr.Fetch(&data, 0, false)
	if e != nil {
		return e.Error()
	} else {
		return data
	}
}

func (a *ResultController) GetDataFromCsv(k *knot.WebContext) interface{} {
	k.Config.OutputType = knot.OutputJson
	d := struct {
		Host      string
		Delimiter string
		Useheader bool
	}{}
	e := k.GetPayload(&d)

	var config = map[string]interface{}{"useheader": d.Useheader, "delimiter": d.Delimiter, "dateformat": "MM-dd-YYYY"}
	ci := &dbox.ConnectionInfo{d.Host, "", "", "", config}
	c, e := dbox.NewConnection("csv", ci)
	defer c.Close()
	e = c.Connect()
	csr, e := c.NewQuery().Select("*").Cursor(nil)
	defer csr.Close()
	data := []tk.M{}
	e = csr.Fetch(&data, 0, false)

	if e != nil {
		return e.Error()
	} else {
		return data
	}
}
