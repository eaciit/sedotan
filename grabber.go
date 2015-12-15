package sedotan

import (
	"bytes"
	"fmt"
	gq "github.com/PuerkitoBio/goquery"
	"github.com/eaciit/toolkit"
	"net/http"
	"strings"
	"time"
)

type GrabColumn struct {
	Alias     string
	Selector  string
	ValueType string //-- Text, Attr, InnerHtml, OuterHtml
	AttrName  string
}

type Config struct {
	Data         toolkit.M
	URL          string
	CallType     string
	DataSettings map[string]*DataSetting

	AuthType     string
	AuthUserId   string
	AuthPassword string
}

type DataSetting struct {
	RowSelector    string
	ColumnSettings []*GrabColumn
}

type Grabber struct {
	Config

	LastExecuted time.Time

	bodyByte []byte
	Response *http.Response
}

func NewGrabber(url string, calltype string, config *Config) *Grabber {
	g := new(Grabber)
	if config != nil {
		g.Config = *config
	}

	if url != "" {
		g.URL = url
	}

	if calltype != "" {
		g.CallType = calltype
	}

	return g
}

func (c *Config) setData(parm toolkit.M) {
	c.Data = parm
}

func (ds *DataSetting) Column(i int, column *GrabColumn) *GrabColumn {
	if i == 0 {
		ds.ColumnSettings = append(ds.ColumnSettings, column)
	} else if i <= len(ds.ColumnSettings) {
		ds.ColumnSettings[i-1] = column
	} else {
		return nil
	}
	return column
}

func (g *Grabber) Data() interface{} {
	return g.Config.Data
}

func (g *Grabber) DataByte() []byte {
	d := g.Data()
	if toolkit.IsValid(d) {
		return toolkit.Jsonify(d)
	}
	return []byte{}
}

func (g *Grabber) Grab(parm toolkit.M) error {
	if parm != nil {
		g.Config.setData(parm)
	}

	r, e := toolkit.HttpCall(g.URL, g.CallType, g.DataByte(), nil)
	errorTxt := ""
	if e != nil {
		errorTxt = e.Error()
	} else if r.StatusCode != 200 {
		errorTxt = r.Status
	}
	if errorTxt != "" {
		return fmt.Errorf("Unable to grab %s. %s", g.URL, errorTxt)
	}

	g.Response = r
	g.bodyByte = toolkit.HttpContent(r)
	return nil
}

func (g *Grabber) ResultString() string {
	if g.Response == nil {
		return ""
	}

	return string(g.bodyByte)
}

func (g *Grabber) ResultFromHtml(dataSettingId string, out interface{}) error {

	reader := bytes.NewReader(g.bodyByte)
	doc, e := gq.NewDocumentFromReader(reader)
	if e != nil {
		return e
	}

	ms := []toolkit.M{}
	records := doc.Find(g.Config.DataSettings[dataSettingId].RowSelector)
	recordCount := records.Length()

	for i := 0; i < recordCount; i++ {
		record := records.Eq(i)
		m := toolkit.M{}
		for cindex, c := range g.Config.DataSettings[dataSettingId].ColumnSettings {
			columnId := fmt.Sprintf("%s", cindex)
			if c.Alias != "" {
				columnId = c.Alias
			}
			sel := record.Find(c.Selector)
			var value interface{}
			valuetype := strings.ToLower(c.ValueType)
			if valuetype == "attr" {
				value, _ = sel.Attr(c.AttrName)
			} else if valuetype == "html" {
				value, _ = sel.Html()
			} else {
				value = sel.Text()
			}
			m.Set(columnId, value)
		}
		ms = append(ms, m)
	}
	if edecode := toolkit.Unjson(toolkit.Jsonify(ms), out); edecode != nil {
		return edecode
	}
	return nil
}

// func (g *Grabber) ResultFromHtml(out interface{}) error {
// 	//s := g.ResultString()
// 	//-- read using jquery

// 	reader := bytes.NewReader(g.bodyByte)
// 	doc, e := gq.NewDocumentFromReader(reader)
// 	if e != nil {
// 		return e
// 	}

// 	ms := []toolkit.M{}
// 	records := doc.Find(g.Config.RowSelector)
// 	recordCount := records.Length()
// 	//fmt.Printf("Find: %d nodes\n", recordCount)
// 	for i := 0; i < recordCount; i++ {
// 		record := records.Eq(i)
// 		m := toolkit.M{}
// 		for cindex, c := range g.Config.ColumnSettings {
// 			columnId := fmt.Sprintf("%s", cindex)
// 			if c.Alias != "" {
// 				columnId = c.Alias
// 			}
// 			sel := record.Find(c.Selector)
// 			var value interface{}
// 			valuetype := strings.ToLower(c.ValueType)
// 			if valuetype == "attr" {
// 				value, _ = sel.Attr(c.AttrName)
// 			} else if valuetype == "html" {
// 				value, _ = sel.Html()
// 			} else {
// 				value = sel.Text()
// 			}
// 			m.Set(columnId, value)
// 		}
// 		ms = append(ms, m)
// 	}
// 	if edecode := toolkit.Unjson(toolkit.Jsonify(ms), out); edecode != nil {
// 		return edecode
// 	}
// 	return nil
// }
