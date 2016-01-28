package controllers

import (
	"fmt"
	"github.com/eaciit/dbox"
	"github.com/eaciit/knot/knot.v1"
	// "io/ioutil"
	"encoding/json"
	_ "github.com/eaciit/dbox/dbc/json"
	tk "github.com/eaciit/toolkit"
	"os"
	"strings"
	"time"
)

var (
	wd = func() string {
		d, _ := os.Getwd()
		return d + "/../"
	}()
)

type ConfigurationController struct {
}

func (a *ConfigurationController) Default(k *knot.WebContext) interface{} {
	k.Config.OutputType = knot.OutputTemplate
	data := tk.M{}
	d, _ := os.Getwd()
	d = strings.Replace(d, "/cli", "", -1)
	data.Set("data_dir", d+"/data/Output/")
	data.Set("log_dir", d+"/data/Log/")
	return data
}

func (a *ConfigurationController) P(k *knot.WebContext) interface{} {
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

func (a *ConfigurationController) Save(k *knot.WebContext) interface{} {
	var (
		filename string
	)

	d := struct {
		Data   string
		IsEdit bool
		NameID string
	}{}
	e := k.GetPayload(&d)
	k.Config.OutputType = knot.OutputJson

	current_data := tk.M{}
	e = json.Unmarshal([]byte(d.Data), &current_data)
	if e != nil {
		fmt.Println("Found : ", e)
	}

	filename = wd + "data/Config/config.json"
	ci := &dbox.ConnectionInfo{filename, "", "", "", nil}
	c, e := dbox.NewConnection("json", ci)
	defer c.Close()
	e = c.Connect()
	if d.IsEdit == true {
		e = c.NewQuery().Where(dbox.Eq("nameid", d.NameID)).Delete().Exec(nil)
	}
	time.Sleep(1200 * time.Millisecond)
	e = c.NewQuery().Insert().Exec(tk.M{"data": current_data})
	if e != nil {
		fmt.Println("Found : ", e)
	}
	if e != nil {
		return e.Error()
	} else {
		return d.Data
	}
}

func (a *ConfigurationController) Delete(k *knot.WebContext) interface{} {
	var (
		filename string
	)

	d := struct {
		NameID string
	}{}
	e := k.GetPayload(&d)
	k.Config.OutputType = knot.OutputJson

	filename = wd + "data/Config/config.json"
	ci := &dbox.ConnectionInfo{filename, "", "", "", nil}
	c, e := dbox.NewConnection("json", ci)
	defer c.Close()
	e = c.Connect()
	e = c.NewQuery().Where(dbox.Eq("nameid", d.NameID)).Delete().Exec(nil)
	if e != nil {
		fmt.Println("Found : ", e)
	}
	if e != nil {
		return e.Error()
	} else {
		return "OK"
	}
}

func (a *ConfigurationController) TestingDBOX(k *knot.WebContext) interface{} {
	d := struct {
		Data string
	}{}

	d.Data = "test"

	k.Config.OutputType = knot.OutputJson

	dataurl := tk.M{}
	dataurl["Pu00231_Input.trade_date"] = "20151214"
	dataurl["Pu00231_Input.variety"] = "i"
	dataurl["Pu00231_Input.trade_type"] = "0"
	dataurl["Submit"] = "Go"
	dataurl["action"] = "Pu00231_result"

	filename := wd + "data/temp.json"
	// filename = "C:\\Gopath\\src\\github.com\\eaciit\\sedotan\\sedotan.v1\\webapps\\cli/../data\\temp.json"
	// filename = filename[0:len(filename)-3] +
	// filename = "C:\\Gopath\\src\\github.com\\eaciit\\sedotan\\sedotan.v1\\webapps\\data\\temp.json"
	fmt.Println(filename)
	ci := &dbox.ConnectionInfo{filename, "", "", "", nil}
	// ci := &dbox.ConnectionInfo{"C:\\Gopath\\src\\tempjson.json", "", "", "", nil}
	c, e := dbox.NewConnection("json", ci)
	if e != nil {
		fmt.Println("Found : ", e)
	}
	defer c.Close()
	e = c.Connect()
	e = c.NewQuery().Insert().Exec(tk.M{"data": dataurl})
	if e != nil {
		fmt.Println("Found : ", e)
	}

	if e != nil {
		return e.Error()
	} else {
		return d.Data
	}
}

func (a *ConfigurationController) GetData(k *knot.WebContext) interface{} {
	k.Config.OutputType = knot.OutputJson

	filename := wd + "data/Config/config.json"
	ci := &dbox.ConnectionInfo{filename, "", "", "", nil}
	c, e := dbox.NewConnection("json", ci)
	defer c.Close()
	e = c.Connect()
	csr, e := c.NewQuery().Select("*").Cursor(nil)
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

func (a *ConfigurationController) GetURL(k *knot.WebContext) interface{} {

	d := struct {
		URL       string
		Method    string
		Parameter tk.M
	}{}
	e := k.GetPayload(&d)
	if e != nil {
		fmt.Println(e)
	}
	k.Config.OutputType = knot.OutputJson
	fmt.Println(d)
	r, e := tk.HttpCall(d.URL, d.Method, nil, tk.M{}.Set("formvalues", d.Parameter))
	fmt.Println(e)
	if e != nil {
		return e.Error()
	} else {
		return tk.HttpContentString(r)
	}
}
