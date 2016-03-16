package main

import (
	"github.com/eaciit/toolkit"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"github.com/eaciit/dbox"
	_ "github.com/eaciit/dbox/dbc/csv"
	_ "github.com/eaciit/dbox/dbc/json"
	_ "github.com/eaciit/dbox/dbc/mongo"
	_ "github.com/eaciit/dbox/dbc/xlsx"
	"github.com/eaciit/sedotan/sedotan.v2"
)

type SourceTypeEnum int

const (
	SourceType_DocExcel SourceTypeEnum = iota
	SourceType_DocMongo
)

type DestInfo struct {
	dbox.IConnection
	collection string
	desttype   string
}

var (
	basePath string = func(dir string, err error) string { return dir }(os.Getwd())
	cmd      *exec.Cmd
	histpath	string
	recpath		string
	filename	string
	filepattern	string
	config       toolkit.M
	SourceType      SourceTypeEnum
	sGrabber        *sedotan.GetDatabase
	destDboxs       map[string]*DestInfo
	histConf        toolkit.M
	extCommand      toolkit.M
	configpath   string
	_id          string
	pid          int
	snapshotdata Snapshot
	thistime     time.Time
	snapshot     string
	EC_APP_PATH  string = os.Getenv("EC_APP_PATH")
	EC_DATA_PATH string = os.Getenv("EC_DATA_PATH")
	tbasepath = strings.Replace(basePath, " ", toolkit.PathSeparator+" ", -1)
)

type Snapshot struct {
	Id             string
	Starttime      string
	Laststartgrab  string
	Lastupdate     string
	Grabcount      int
	Rowgrabbed     int
	Errorfound     int
	Lastgrabstatus string //[success|failed]
	Grabstatus     string //[running|done]
	Cgtotal        int
	Cgprocess      int
	Note           string
	Pid            int
}

func TestBuild(t *testing.T) {
	t.Skip("Skip : Comment this line to do test")
	cmd = exec.Command("cmd", "/C", "go", "build", "../daemon")
	e := cmd.Run()
	// cmd.Wait()

	if e != nil {
		t.Errorf("Error, %s \n", e)
	} else {
		t.Logf("RUN, %s \n", "success")
	}
	// cmd = exec.Command("rm", "-rf", "cli")
	// cmd.Run()
	// cmd.Wait()
}

func TestRunDaemon(t *testing.T) {
	t.Skip("Skip : Comment this line to do test")

	// tbasepath := strings.Replace(basePath, " ", toolkit.PathSeparator+" ", -1)

	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", "..\\daemon\\sedotandaemon.exe", `-config="`+tbasepath+`\config-daemon.json"`, `-logpath="`+tbasepath+`\log"`)
	} else {
		cmd = exec.Command("sudo", "../daemon/sedotandaemon", `-config="`+tbasepath+`\config-daemon.json"`, `-logpath="`+tbasepath+`\log"`)
	}

	err := cmd.Start()
	if err != nil {
		t.Errorf("Error, %s \n", err)
	}

	go func(cmd *exec.Cmd) {
		daemoninterval := 60 * time.Second
		<-time.After(daemoninterval)

		err := cmd.Process.Signal(os.Kill)
		if err != nil {
			toolkit.Println("Error, %s \n", err.Error())
		}

		if runtime.GOOS == "windows" {
			err = exec.Command("cmd", "/C", "taskkill", "/F", "/IM", "sedotandaemon.exe", "/T").Run()
		} else {
			err = exec.Command("sudo", "pkill", "sedotandaemon").Run()
		}

		if err != nil {
			toolkit.Println("Error, %s \n", err.Error())
		}

	}(cmd)

	err = cmd.Wait()
	if err != nil {
		t.Errorf("Error, %s \n", err.Error())
	}
}

func TestRunSedotanw(t *testing.T) {
	t.Skip("Skip : Comment this line to do test")

	tbasepath := strings.Replace(basePath, " ", toolkit.PathSeparator+" ", -1)
	snapshotpath := filepath.Join(tbasepath, "log", "daemonsnapshot.csv")

	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", "..\\sedotanw\\sedotanw.exe", `-config="`+tbasepath+`\config-daemon.json"`, `-snapshot="`+snapshotpath+`"`, `-id="irondcecomcn"`)
	} else {
		cmd = exec.Command("sudo", "../sedotanw/sedotanw", `-config="`+tbasepath+`/config-daemon.json"`, `-snapshot="`+snapshotpath+`"`, `-id="irondcecomcn"`)
	}

	err := cmd.Start()
	if err != nil {
		t.Errorf("Error, %s \n", err)
	}

	// go func(cmd *exec.Cmd) {
	// 	daemoninterval := 60 * time.Second
	// 	<-time.After(daemoninterval)

	// 	err := cmd.Process.Signal(os.Kill)
	// 	if err != nil {
	// 		toolkit.Println("Error, %s \n", err.Error())
	// 	}

	// 	if runtime.GOOS == "windows" {
	// 		err = exec.Command("cmd", "/C", "taskkill", "/F", "/IM", "sedotanw.exe", "/T").Run()
	// 	} else {
	// 		err = exec.Command("sudo", "pkill", "sedotanw").Run()
	// 	}

	// 	if err != nil {
	// 		toolkit.Println("Error, %s \n", err.Error())
	// 	}

	// }(cmd)

	err = cmd.Wait()
	if err != nil {
		t.Errorf("Error, %s \n", err.Error())
	}
}

func TestRunTesting(t *testing.T) {
 	// fullfilename := filepath.Join(toolkit.ToString(histConf["histpath"]), toolkit.ToString(histConf["filename"]))

	if runtime.GOOS == "windows" {
		// cmd = exec.Command("cmd", "/C", "E:\\EACIIT\\src\\github.com\\eaciit\\sedotan\\sedotan.v2\\sedotanread\\main.exe", `-config="`+tbasepath+`\config-daemon.json"`, `-logpath="`+tbasepath+`\log"`, `-readtype="history"`)
		cmd = exec.Command("cmd", "/C", "E:\\EACIIT\\src\\github.com\\eaciit\\sedotan\\sedotan.v2\\sedotanread\\main.exe", `-config="`+tbasepath+`\config-daemon.json"`, `-readtype="history"`, `-pathfile="E:\EACIIT\src\github.com\eaciit\sedotan\sedotan.v2\test\hist\HIST-GRABDCE-20160225.csv"`)
	} else {
		cmd = exec.Command("sudo", "../daemon/sedotandaemon", `-config="`+tbasepath+`\config-daemon.json"`, `-logpath="`+tbasepath+`\log"`)
	}
	// cmd = exec.Command("go run main.go -readtype=history -pathfile=E:/EACIIT\\src\\github.com\\eaciit\\sedotan\\sedotan.v2\\test\\hist\\HIST-GRABDCE-20160225.csv")
	err := cmd.Start()
	if err != nil {
		t.Errorf("Error, %s \n", err)
	}
}

