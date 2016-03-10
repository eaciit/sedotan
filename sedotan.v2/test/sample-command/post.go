package main

import (
	"fmt"
	"github.com/eaciit/toolkit"
	"os"
)

func main() {
	jsonarg := toolkit.M{}
	
	err := toolkit.UnjsonFromString(os.Args[1], &jsonarg)
	if err != nil{
		return
	}

	jsonarg["Date"] = "000000"
	jsonarg[`MB 62% Fe`] = "POST " + toolkit.ToString(jsonarg[`MB 62% Fe`])
	jsonarg[`Platts 62% Fe IODEX`] = toolkit.ToString(toolkit.ToFloat64(jsonarg[`Platts 62% Fe IODEX`], 6, toolkit.RoundingAuto)*5)
	fmt.Println(toolkit.JsonString(jsonarg))
}