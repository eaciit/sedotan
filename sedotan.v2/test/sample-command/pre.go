package main

import (
	"fmt"
	"github.com/eaciit/toolkit"
	"os"
	"time"
)

func main() {
	jsonarg := toolkit.M{}
	
	err := toolkit.UnjsonFromString(os.Args[1], &jsonarg)
	if err != nil{
		return
	}

	jsonarg["Date"] = time.Now().Format("020106")
	jsonarg[`MB 62% Fe`] = "FE " + toolkit.ToString(jsonarg[`MB 62% Fe`])
	jsonarg[`Platts 62% Fe IODEX`] = toolkit.ToString(toolkit.ToFloat64(jsonarg[`Platts 62% Fe IODEX`], 6, toolkit.RoundingAuto))
	fmt.Println(toolkit.JsonString(jsonarg))
}