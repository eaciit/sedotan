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
	"os"
	"strings"
)

var (
	configPath string
	config     toolkit.M
	debugMode  bool
)

func checkError(err error) {
	if err == nil {
		return
	}

	if debugMode {
		panic(err.Error())
	} else {
		fmt.Printf("ERROR! %s\n", err.Error())
	}

	os.Exit(0)
}

func validateArguments() {
	if configPath == "" {
		checkError(errors.New("-config cannot be empty"))
	}
}

func fetchConfig() {
	if strings.Contains(configPath, "http") {
		res, err := http.Get(configPath)
		checkError(err)
		defer res.Body.Close()

		decoder := json.NewDecoder(res.Body)
		err = decoder.Decode(&config)
		checkError(err)
	} else {
		bytes, err := ioutil.ReadFile(configPath)
		checkError(err)

		err = json.Unmarshal(bytes, &config)
		checkError(err)
	}
}

func main() {
	flagConfigPath := flag.String("config", "", "config file")
	flagDebugMode := flag.Bool("debug", false, "debug mode")

	flag.Parse()
	configPath = *flagConfigPath
	debugMode = *flagDebugMode

	validateArguments()
	fetchConfig()
	_, err := sedotan.Process(config)
	checkError(err)
}
