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
	"fmt"
	"github.com/eaciit/colony-core/v0"
	"github.com/eaciit/sshclient"
	"golang.org/x/crypto/ssh"
)

var s *ServerController

type ServerController struct {
	
}

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

func TestRunSedotanReadHistory(t *testing.T) {
	t.Skip("Skip : Comment this line to do test")
	var history = toolkit.M{}
	arrcmd := make([]string, 0, 0)

	arrcmd = append(arrcmd, EC_APP_PATH+`\bin\sedotanread.exe`)
	arrcmd = append(arrcmd, `-readtype=history`)
	arrcmd = append(arrcmd, `-pathfile=`+EC_DATA_PATH+`\webgrabber\history\HIST-GRABDCE-20160316.csv`)

	if runtime.GOOS == "windows" {
		historystring, _ := toolkit.RunCommand(arrcmd[0], arrcmd[1:]...)
		err := toolkit.UnjsonFromString(historystring, &history)
		if err != nil {
			t.Errorf("Error, %s \n", err)
		}
	} else {
		// cmd = exec.Command("sudo", "../daemon/sedotandaemon", `-config="`+tbasepath+`\config-daemon.json"`, `-logpath="`+tbasepath+`\log"`)
	}
	
	fmt.Println(history)
}

func TestRunSedotanReadSnapshot(t *testing.T) {
	t.Skip("Skip : Comment this line to do test")
	arrcmd := make([]string, 0, 0)
	result := toolkit.M{}

	arrcmd = append(arrcmd, EC_APP_PATH+`\bin\sedotanread.exe`)
	arrcmd = append(arrcmd, `-readtype=snapshot`)
	arrcmd = append(arrcmd, `-pathfile=`+EC_DATA_PATH+`\daemon\daemonsnapshot.csv`)
	arrcmd = append(arrcmd, `-nameid=irondcecomcn`)

	if runtime.GOOS == "windows" {
		SnapShot, err := toolkit.RunCommand(arrcmd[0], arrcmd[1:]...)
		err = toolkit.UnjsonFromString(SnapShot, &result)
		if err != nil {
			t.Errorf("Error, %s \n", err)
		}
	} else {
		// cmd = exec.Command("sudo", "../daemon/sedotandaemon", `-config="`+tbasepath+`\config-daemon.json"`, `-logpath="`+tbasepath+`\log"`)
	}
	
	fmt.Println(result)
}

func TestRunSedotanReadRecHistory(t *testing.T) {
	t.Skip("Skip : Comment this line to do test")
	arrcmd := make([]string, 0, 0)
	result := toolkit.M{}

	arrcmd = append(arrcmd, EC_APP_PATH+`\bin\sedotanread.exe`)
	arrcmd = append(arrcmd, `-readtype=rechistory`)
	arrcmd = append(arrcmd, `-recfile=E:\EACIIT\src\github.com\eaciit\colony-app\data-root\webgrabber\historyrec\irondcecomcn.Iron01-20160316022830.csv`)

	if runtime.GOOS == "windows" {
		cmd = exec.Command(arrcmd[0], arrcmd[1:]...)
		rechistory, err := toolkit.RunCommand(arrcmd[0], arrcmd[1:]...)
		err = toolkit.UnjsonFromString(rechistory, &result)
		if err != nil {
			t.Errorf("Error, %s \n", err)
		}
		byteoutput, err := cmd.CombinedOutput()
		if err != nil {
			// Log.AddLog(fmt.Sprintf("[%v] run at %v, found error : %v", eid, sedotan.DateToString(thistime), err.Error()), "ERROR")
		}
		err = toolkit.UnjsonFromString(string(byteoutput), &result)
	} else {
		// cmd = exec.Command("sudo", "../daemon/sedotandaemon", `-config="`+tbasepath+`\config-daemon.json"`, `-logpath="`+tbasepath+`\log"`)
	}
	
	fmt.Println(result)
}

func TestRunSedotanReadRecHistorySSH(t *testing.T) {
	// t.Skip("Skip : Comment this line to do test")

	data := new(colonycore.Server)
	
	sshSetting, client, err := s.SSHConnect(data)
	if err != nil {
		fmt.Println(err)
	}
	defer client.Close()

	output, err := sshSetting.GetOutputCommandSsh(`dir`)
	if err != nil {
		// do something
	}
	fmt.Println(output)
}

func (s *ServerController) SSHConnect(payload *colonycore.Server) (sshclient.SshSetting, *ssh.Client, error) {
	client := sshclient.SshSetting{}
	client.SSHHost = "192.168.56.103:22"

	client.SSHAuthType = 0
	client.SSHUser = "eaciit1"
	client.SSHPassword = "12345"

	//fmt.Println(client) {192.168.56.103:22 eaciit1 12345  0}

	theClient, err := client.Connect()

	return client, theClient, err
}