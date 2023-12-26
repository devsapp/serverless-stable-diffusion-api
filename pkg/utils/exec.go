package utils

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os/exec"
	"syscall"
)

type ExecItem struct {
	Pid    int
	Status int
	Args   []string
	Output string
	Stdout io.ReadCloser
}

func DoExec(shell, dir string, env []string) *ExecItem {
	execItem := &ExecItem{
		Status: 0,
	}
	cmd := exec.Command("/bin/bash", "-c", shell)
	stdout, _ := cmd.StdoutPipe()
	defer stdout.Close()
	if dir != "" {
		cmd.Dir = dir
	}
	if env != nil {
		cmd.Env = env
	}
	if err := cmd.Start(); err != nil {
		fmt.Println("cmd.Start err: ", err.Error())
	}

	execItem.Args = cmd.Args
	execItem.Pid = cmd.Process.Pid
	result, _ := ioutil.ReadAll(stdout)
	execItem.Output = string(result)
	if err := cmd.Wait(); err != nil {
		if ex, ok := err.(*exec.ExitError); ok {
			execItem.Status = ex.Sys().(syscall.WaitStatus).ExitStatus()
		}
	}
	return execItem
}

func DoExecAsync(shell, dir string, env []string) (*ExecItem, error) {
	execItem := &ExecItem{
		Status: 0,
	}
	cmd := exec.Command("/bin/bash", "-c", shell)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout
	if env != nil {
		cmd.Env = env
	}
	//cmd.Stderr = os.Stderr
	//cmd.Stdout = os.Stdout
	if dir != "" {
		cmd.Dir = dir
	}
	if err := cmd.Start(); err != nil {
		logrus.Errorf("cmd.Start err: %s", err.Error())
		return nil, errors.New("sd start error")
	}

	execItem.Args = cmd.Args
	execItem.Pid = cmd.Process.Pid
	execItem.Stdout = stdout
	return execItem, nil
}
