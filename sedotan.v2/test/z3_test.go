package main

import (
	"github.com/eaciit/toolkit"
	"os"
	"os/exec"
	"strings"
	"testing"
)

var (
	basePath string = func(dir string, err error) string { return dir }(os.Getwd())
	cmd      *exec.Cmd
)

func TestBuild(t *testing.T) {
	t.Skip("Skip : Comment this line to do test")
	cmd = exec.Command("go", "build", "../sedotanwd")
	cmd.Run()
	cmd.Wait()

	cmd = exec.Command("rm", "-rf", "cli")
	cmd.Run()
	cmd.Wait()
}

func TestRunCommand(t *testing.T) {
	// cmd = exec.Command("go", "run", "../sedotanwd/main.go", `-config="config-web-alip.json"`, "-debug=true")
	// cmd.Run()
	// cmd.Wait()
	// res, e := toolkit.RunCommand("go", "run", "../sedotanwd/main.go", `-config="config-web-alip.json"`, "-debug=true")

	tbasepath := strings.Replace(basePath, " ", toolkit.PathSeparator+" ", -1)
	res, e := toolkit.RunCommand("cmd", "/C", "go", "run", "../daemon/main.go", `-config="`+tbasepath+`\config-daemon.json"`, `-logpath="`+tbasepath+`\log"`)
	if e != nil {
		t.Errorf("Error, %s \n", e)
	} else {
		t.Logf("RUN, %s \n", res)
	}
}
