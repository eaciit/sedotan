package main

import (
	"github.com/eaciit/toolkit"
	"os"
	"os/exec"
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

func TestRunCommand(t *testing.T) {
	// t.Skip("Skip : Comment this line to do test")

	tbasepath := strings.Replace(basePath, " ", toolkit.PathSeparator+" ", -1)

	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", "go", "run", "../daemon/main.go", `-config="`+tbasepath+`\config-daemon.json"`, `-logpath="`+tbasepath+`\log"`)
	} else {
		cmd = exec.Command("go", "run", "../daemon/main.go", `-config="`+tbasepath+`\config-daemon.json"`, `-logpath="`+tbasepath+`\log"`)
	}

	err := cmd.Start()
	if err != nil {
		t.Errorf("Error, %s \n", err)
	}

	go func(cmd *exec.Cmd) {
		daemoninterval := 10 * time.Second
		<-time.After(daemoninterval)

		err := cmd.Process.Signal(os.Kill)
		if err != nil {
			toolkit.Println("Error, %s \n", err.Error())
		}

		err = exec.Command("cmd", "/C", "taskkill", "/F", "/IM", "main.exe", "/T").Run()
		if err != nil {
			toolkit.Println("Error, %s \n", err.Error())
		}

	}(cmd)

	err = cmd.Wait()
	if err != nil {
		t.Errorf("Error, %s \n", err.Error())
	}
}
