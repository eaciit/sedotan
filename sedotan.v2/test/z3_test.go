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
)

var (
	basePath string = func(dir string, err error) string { return dir }(os.Getwd())
	cmd      *exec.Cmd
)

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
	// t.Skip("Skip : Comment this line to do test")

	tbasepath := strings.Replace(basePath, " ", toolkit.PathSeparator+" ", -1)

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
