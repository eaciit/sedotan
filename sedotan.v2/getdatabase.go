package sedotan

import (
	// "bytes"
	// "fmt"
	// "github.com/eaciit/cast"
	"github.com/eaciit/dbox"
	_ "github.com/eaciit/dbox/dbc/csv"
	_ "github.com/eaciit/dbox/dbc/json"
	_ "github.com/eaciit/dbox/dbc/mongo"
	_ "github.com/eaciit/dbox/dbc/xlsx"
	"github.com/eaciit/toolkit"
	// "reflect"
	// "regexp"
	"errors"
	// "strings"
	"strings"
	"time"
)

// type ViewColumn struct {
// 	Alias    string
// 	Selector string
// 	// ValueType string //-- Text, Attr, InnerHtml, OuterHtml
// 	// AttrName  string
// }
type MapColumn struct {
	Source      string
	SType       string
	Destination string //-- Text, Attr, InnerHtml, OuterHtml
	DType       string
}

type CollectionSetting struct {
	Collection  string
	MapsColumns []*MapColumn
	FilterCond  toolkit.M
	filterDbox  *dbox.Filter
}

type GetDatabase struct {
	dbox.ConnectionInfo
	conn               dbox.IConnection
	desttype           string
	CollectionSettings map[string]*CollectionSetting

	LastExecuted time.Time

	Response []toolkit.M
}

func NewGetDatabase(host string, desttype string, connInfo *dbox.ConnectionInfo) (*GetDatabase, error) {
	g := new(GetDatabase)
	if connInfo != nil {
		g.ConnectionInfo = *connInfo
	}

	if desttype != "" {
		g.desttype = desttype
	}

	if host != "" {
		g.Host = host
	}

	if g.Host == "" || g.desttype == "" {
		return nil, errors.New("Host or Type cannot blank")
	}

	return g, nil
}

func (ds *CollectionSetting) Column(i int, column *MapColumn) *MapColumn {
	if i == 0 {
		ds.MapsColumns = append(ds.MapsColumns, column)
	} else if i <= len(ds.MapsColumns) {
		ds.MapsColumns[i-1] = column
	} else {
		return nil
	}
	return column
}

// func (ds *CollectionSetting) SetFilterCond(filter toolkit.M) {
// 	ds.FilterCond = filter
// 	ds.filterDbox =
// }

func (g *GetDatabase) CloseConn() {
	g.conn.Close()
}

func (g *GetDatabase) GetQuery(dataSettingId string) (iQ dbox.IQuery, err error) {

	g.conn, err = dbox.NewConnection(g.desttype, &g.ConnectionInfo)
	if err != nil {
		return
	}

	err = g.conn.Connect()
	if err != nil {
		return
	}

	// defer c.Close()

	iQ = g.conn.NewQuery()
	if g.CollectionSettings[dataSettingId].Collection != "" {
		iQ.From(g.CollectionSettings[dataSettingId].Collection)
	}

	aSelect := make([]string, 0, 0)
	for _, val := range g.CollectionSettings[dataSettingId].MapsColumns {
		tstring := val.Source
		if strings.Contains(val.Source, "|") {
			splitstring := strings.Split(val.Source, "|")
			tstring = splitstring[0]
		}

		if tstring != "" && toolkit.HasMember(aSelect, tstring) {
			aSelect = append(aSelect, tstring)
		}

	}

	if len(aSelect) > 0 {
		iQ.Select(aSelect...)
	}

	if len(g.CollectionSettings[dataSettingId].FilterCond) > 0 {
		iQ.Where(g.CollectionSettings[dataSettingId].filterDbox)
	}

	return
	// csr, e := iQ.Cursor(nil)

	// if e != nil {
	// 	return e
	// }
	// if csr == nil {
	// 	return e
	// }
	// defer csr.Close()

	// results := make([]toolkit.M, 0)
	// e = csr.Fetch(&results, 0, false)
	// if e != nil {
	// 	return e
	// }

	// ms := []toolkit.M{}
	// for _, val := range results {
	// 	m := toolkit.M{}
	// 	for _, column := range g.CollectionSettings[dataSettingId].MapsColumns {
	// 		m.Set(column.Source, "")
	// 		if val.Has(column.Destination) {
	// 			m.Set(column.Source, val[column.Destination])
	// 		}
	// 	}
	// 	ms = append(ms, m)
	// }

	// if edecode := toolkit.Unjson(toolkit.Jsonify(ms), out); edecode != nil {
	// 	return edecode
	// }
	// return nil
}

func (g *GetDatabase) ResultFromDatabase(dataSettingId string, out interface{}) error {

	c, e := dbox.NewConnection(g.desttype, &g.ConnectionInfo)
	if e != nil {
		return e
	}

	e = c.Connect()
	if e != nil {
		return e
	}

	defer c.Close()

	iQ := c.NewQuery()
	if g.CollectionSettings[dataSettingId].Collection != "" {
		iQ.From(g.CollectionSettings[dataSettingId].Collection)
	}

	for _, val := range g.CollectionSettings[dataSettingId].MapsColumns {
		iQ.Select(val.Source)
	}

	if len(g.CollectionSettings[dataSettingId].FilterCond) > 0 {
		iQ.Where(g.CollectionSettings[dataSettingId].filterDbox)
	}

	csr, e := iQ.Cursor(nil)

	if e != nil {
		return e
	}
	if csr == nil {
		return e
	}
	defer csr.Close()

	results := make([]toolkit.M, 0)
	e = csr.Fetch(&results, 0, false)
	if e != nil {
		return e
	}

	ms := []toolkit.M{}
	for _, val := range results {
		m := toolkit.M{}
		for _, column := range g.CollectionSettings[dataSettingId].MapsColumns {
			m.Set(column.Source, "")
			if val.Has(column.Destination) {
				m.Set(column.Source, val[column.Destination])
			}
		}
		ms = append(ms, m)
	}

	if edecode := toolkit.Unjson(toolkit.Jsonify(ms), out); edecode != nil {
		return edecode
	}
	return nil
}

func (ds *CollectionSetting) SetFilterCond(filter toolkit.M) {
	if filter == nil {
		ds.FilterCond = toolkit.M{}
	} else {
		ds.FilterCond = filter
	}

	if len(ds.FilterCond) > 0 {
		ds.filterDbox = filterCondition(ds.FilterCond)
	}
}

func filterCondition(cond toolkit.M) *dbox.Filter {
	fb := new(dbox.Filter)

	for key, val := range cond {
		if key == "$and" || key == "$or" {
			afb := []*dbox.Filter{}
			for _, sVal := range val.([]interface{}) {
				rVal := sVal.(map[string]interface{})
				mVal := toolkit.M{}
				for rKey, mapVal := range rVal {
					mVal.Set(rKey, mapVal)
				}

				afb = append(afb, filterCondition(mVal))
			}

			if key == "$and" {
				fb = dbox.And(afb...)
			} else {
				fb = dbox.Or(afb...)
			}

		} else {
			if toolkit.TypeName(val) == "map[string]interface {}" {
				mVal := val.(map[string]interface{})
				tomVal, _ := toolkit.ToM(mVal)
				switch {
				case tomVal.Has("$eq"):
					fb = dbox.Eq(key, tomVal["$eq"])
				case tomVal.Has("$ne"):
					fb = dbox.Ne(key, tomVal["$ne"])
				case tomVal.Has("$regex"):
					fb = dbox.Contains(key, toolkit.ToString(tomVal["$regex"]))
				case tomVal.Has("$gt"):
					fb = dbox.Gt(key, tomVal["$gt"])
				case tomVal.Has("$gte"):
					fb = dbox.Gte(key, tomVal["$gte"])
				case tomVal.Has("$lt"):
					fb = dbox.Lt(key, tomVal["$lt"])
				case tomVal.Has("$lte"):
					fb = dbox.Lte(key, tomVal["$lte"])
				case tomVal.Has("$in"):
					tval := make([]interface{}, 0, 0)
					if toolkit.TypeName(tomVal["$in"]) == "[]interface {}" {
						for _, tin := range tomVal["$in"].([]interface{}) {
							tval = append(tval, tin)
						}
					} else {
						tval = append(tval, tomVal["$in"])
					}

					fb = dbox.In(key, tval...)
				case tomVal.Has("$nin"):
					tval := make([]interface{}, 0, 0)
					if toolkit.TypeName(tomVal["$nin"]) == "[]interface {}" {
						for _, tin := range tomVal["$nin"].([]interface{}) {
							tval = append(tval, tin)
						}
					} else {
						tval = append(tval, tomVal["$nin"])
					}

					fb = dbox.Nin(key, tval...)
				}
			} else {
				fb = dbox.Eq(key, val)
			}
		}
	}

	return fb
}
