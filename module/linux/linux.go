package linux

import (
	"errors"
	"github.com/march1993/gohive/api"
	"github.com/march1993/gohive/config"
	"github.com/march1993/gohive/module"
	"os"
	"os/exec"
	"strings"
)

type linux struct{}

const (
	Prefix = "gohive_app_"
	Group  = "gohive_app"
	Suffix = ".data"
)

func init() {
	module.RegisterModule("linux", &linux{})

	cmd := exec.Command("groupadd", "-f", Group)
	if stdout, err := cmd.CombinedOutput(); err != nil {
		panic(string(stdout) + err.Error())
	}

}

func getHomeDir(name string) string {
	return config.APP_DIR + "/" + Prefix + name
}

func getDataDir(name string) string {
	return config.APP_DIR + "/" + Prefix + name + Suffix
}

func (l *linux) Create(name string) error {

	if l.Status(name).Status == api.APP_NON_EXIST {
		unixname := Prefix + name

		cmd := exec.Command("useradd",
			"-b", config.APP_DIR, // home directory
			"-m",                   // create home
			"-s", config.SSH_SHELL, // shell
			"-g", Group, // group
			"-K", "UMASK=0077",
			unixname)
		stdout, err := cmd.CombinedOutput()

		if err != nil {
			panic(string(stdout) + err.Error())
		}

		os.MkdirAll(getDataDir(name), 0700)

		return nil

	} else {
		return errors.New(api.APP_ALREADY_EXISTING)
	}
}

func (l *linux) Rename(oldName string, newName string) error {
	return nil
}

func (l *linux) Remove(name string) error {
	if l.Status(name).Status == api.APP_NON_EXIST {
		return errors.New(api.APP_NON_EXIST)
	} else {
		unixname := Prefix + name
		cmd := exec.Command("userdel", unixname)
		cmd.CombinedOutput()

		os.RemoveAll(getHomeDir(name))
		os.RemoveAll(getDataDir(name))

		return nil
	}
}

func (l *linux) Status(name string) api.Status {
	// TODO
	return api.Status{
		Status: api.APP_NON_EXIST,
	}
}

func (l *linux) Repair(name string) error {
	// TODO
	return nil
}

func (l *linux) ListRemoved() []string {
	cmd := exec.Command("members", Group)
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}

	members := strings.Split(strings.Trim(string(stdout), "\n"), " ")

	ret := []string{}

	for _, member := range members {
		if l.Status(member).Status != api.STATUS_SUCCESS {
			ret = append(ret, member)
		}
	}

	return ret
}
