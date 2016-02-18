package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/eaciit/sedotan/sedotan.v2"
	"github.com/eaciit/toolkit"
	"io/ioutil"
	"net/http"
	"strings"
)

var (
	configPath string
	config     toolkit.M
	debugMode  bool
)

func fetchConfig(_id string) {
	var result interface{}

	if strings.Contains(configPath, "http") {
		res, err := http.Get(configPath)
		sedotan.CheckError(err)
		defer res.Body.Close()

		decoder := json.NewDecoder(res.Body)
		err = decoder.Decode(&result)
		sedotan.CheckError(err)
	} else {
		bytes, err := ioutil.ReadFile(configPath)
		sedotan.CheckError(err)

		err = json.Unmarshal(bytes, &result)
		sedotan.CheckError(err)
	}

	switch toolkit.TypeName(result) {
	case "[]interface {}":
		isFound := false
		for _, eachRaw := range result.([]interface{}) {
			each := eachRaw.(map[string]interface{})
			if each["_id"].(string) == _id {
				m, err := toolkit.ToM(each)
				sedotan.CheckError(err)

				config = m
				isFound = true
			}
		}

		if !isFound {
			sedotan.CheckError(errors.New(fmt.Sprintf("config with _id %s is not found\n%#v", _id, result)))
		}
	case "map[string]interface {}":
		m, err := toolkit.ToM(result)
		sedotan.CheckError(err)
		config = m
	default:
		sedotan.CheckError(errors.New(fmt.Sprintf("invalid config file\n%#v", result)))
	}
}

func main() {
	flagConfigPath := flag.String("config", "", "config file")
	flagDebugMode := flag.Bool("debug", false, "debug mode")
	flagID := flag.String("id", "", "_id of the config (if array)")

	flag.Parse()
	configPath = *flagConfigPath
	debugMode = *flagDebugMode

	if configPath == "" {
		sedotan.CheckError(errors.New("-config cannot be empty"))
	}

	fetchConfig(*flagID)
	_, err := sedotan.Process(config)
	sedotan.CheckError(err)
}
