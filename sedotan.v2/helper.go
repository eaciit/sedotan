package sedotan

import (
	"fmt"
	"os"
)

var (
	debugMode bool
)

func CheckError(err error) {
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
