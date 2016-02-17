package main

import (
	"os"
	"os/exec"
	"testing"
)

var (
	basePath string = func(dir string, err error) string { return dir }(os.Getwd())
	cmd      *exec.Cmd
)

func deleteBuildedApp() {
	cmd = exec.Command("rm", "-rf", "cli")
	cmd.Run()
	cmd.Wait()
}

func TestBuild(t *testing.T) {
	deleteBuildedApp()

	cmd = exec.Command("go", "build", "../cli")
	cmd.Run()
	cmd.Wait()
}

func TestRunCommand(t *testing.T) {
	cmd = exec.Command("go", "run", "../cli/main.go", `-config="config.json"`, "-debug=true")
	cmd.Run()
	cmd.Wait()

	deleteBuildedApp()
}
