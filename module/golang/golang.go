package golang

import (
	"github.com/march1993/gohive/api"
	"github.com/march1993/gohive/config"
	"github.com/march1993/gohive/module"
	_ "github.com/march1993/gohive/module/git"
	"github.com/march1993/gohive/util"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type golang struct{}

func init() {
	module.RegisterModule("golang", &golang{})
	module.RegisterRuncom(runcomHandler)
	if err := os.MkdirAll(config.GOLANG_DIR, config.GOLANG_DIR_PERM); err != nil {
		panic(err.Error())
	}
	if err := os.Chmod(config.GOLANG_DIR, config.GOLANG_DIR_PERM); err != nil {
		panic(err.Error())
	}
	if err := os.Chown(config.GOLANG_DIR, config.GOLANG_DIR_O_USER, config.GOLANG_DIR_O_GROUP); err != nil {
		panic(err.Error())
	}

}

func runcomHandler(name string) []string {

	version := config.AppConfigGet(name, "golang", "version", "")
	goroot := config.GOLANG_DIR + "/" + GO_PREFIX + version

	return []string{
		"export GOPATH=" + config.GOLANG_GOPATH,
		"export GOROOT=" + goroot,
		"export PATH=$GOROOT/bin:$PATH",
	}
}

func (g *golang) Create(name string) api.Status {
	// do nothing
	return api.Status{Status: api.STATUS_SUCCESS}
}

func (g *golang) Remove(name string) api.Status {
	// do nothing
	return api.Status{Status: api.STATUS_SUCCESS}
}

func (g *golang) Status(name string) api.Status {

	version := config.AppConfigGet(name, "golang", "version", "")

	if version == "" {
		return api.Status{
			Status: api.STATUS_FAILURE,
			Reason: api.GOLANG_VERSION_UNSET,
		}
	}

	unixname := config.APP_PREFIX + name
	stdout, err := exec.Command("runuser",
		unixname,
		"-l",
		"-c", "go version",
	).CombinedOutput()
	if err != nil {
		return api.Status{
			Status: api.STATUS_FAILURE,
			Reason: string(stdout),
		}
	} else {
		output := string(stdout)
		if strings.Contains(output, version) {
			return api.Status{
				Status: api.STATUS_SUCCESS,
				Result: output,
			}
		} else {
			return api.Status{
				Status: api.STATUS_FAILURE,
				Reason: api.GOLANG_VERSION_MISMATCHING,
				Addition: map[string]string{
					"Desired":   version,
					"Presented": output,
				},
			}
		}

	}
}

func SetGolangVersion(name string, version string) {
	config.AppConfigSet(name, "golang", "version", version)
}

func (g *golang) Repair(name string) api.Status {
	// unixname := config.APP_PREFIX + name

	version := config.AppConfigGet(name, "golang", "version", "")

	list := GetGolangList()

	// don't have certain version of golang, download it
	if !util.Includes(list, GO_PREFIX+version) {
		status := SetGolangInstallation(version)
		if status.Status != api.STATUS_SUCCESS {
			return status
		}
	}

	return api.Status{Status: api.STATUS_SUCCESS}

}

func (g *golang) ListRemoved() []string {
	return []string{}
}

const (
	GO_PREFIX        = "go"
	GO_DOWNLOAD_PATH = "https://dl.google.com/go/go{{VERSION}}.linux-amd64.tar.gz"
)

func GetGolangList() []string {

	result := []string{}

	files, err := ioutil.ReadDir(config.GOLANG_DIR)

	if err != nil {
		panic(err.Error())
	}

	for _, file := range files {
		if file.IsDir() {
			name := file.Name()

			if strings.HasPrefix(name, GO_PREFIX) {
				result = append(result, name)
			}
		}
	}

	return result
}

func SetGolangInstallation(version string) api.Status {

	tmpPath := config.GOLANG_DIR + "/go" + version + ".tmp"
	tmp, err := os.Create(tmpPath)

	if err != nil {
		return api.Status{
			Status: api.STATUS_FAILURE,
			Reason: err.Error(),
		}
	}

	defer tmp.Close()
	defer os.Remove(tmpPath)

	resp, err := http.Get(strings.Replace(GO_DOWNLOAD_PATH, "{{VERSION}}", version, -1))

	if err != nil {
		return api.Status{
			Status: api.STATUS_FAILURE,
			Reason: err.Error(),
		}
	}

	if resp.StatusCode != http.StatusOK {
		return api.Status{
			Status: api.STATUS_FAILURE,
			Reason: api.REASON_DOWNLOAD_FAILED,
		}
	}

	defer resp.Body.Close()

	_, err = io.Copy(tmp, resp.Body)

	if err != nil {

		return api.Status{
			Status: api.STATUS_FAILURE,
			Reason: err.Error(),
		}
	}

	outputPath := config.GOLANG_DIR + "/go" + version

	if t := util.Shell("mkdir", outputPath); t != "" {
		return api.Status{
			Status: api.STATUS_FAILURE,
			Reason: t,
		}
	}

	if t := util.Shell("tar",
		"xf", tmpPath,
		"-C", outputPath,
		"--strip-components", "1"); t != "" {
		return api.Status{
			Status: api.STATUS_FAILURE,
			Reason: t,
		}
	}

	return api.Status{Status: api.STATUS_SUCCESS}
}

func TriggerBuild(name string) api.Status {
	unixname := config.APP_PREFIX + name

	stdout, err := exec.Command("runuser",
		unixname,
		"-l",
		"-c", "~/repo.git/hooks/post-update",
	).CombinedOutput()

	if err != nil {
		return api.Status{
			Status: api.STATUS_FAILURE,
			Reason: string(stdout),
		}
	} else {
		return api.Status{
			Status: api.STATUS_SUCCESS,
			Reason: string(stdout),
		}
	}
}

type GetLastBuildResult struct {
	Stdout string
	Stderr string
}

func GetLastBuild(name string) api.Status {

	errs := []string{}
	stdout, err := ioutil.ReadFile(config.GetHomeDir(name) + "/lastbuild.stdout")
	if err != nil {
		errs = append(errs, err.Error())
	}

	stderr, err := ioutil.ReadFile(config.GetHomeDir(name) + "/lastbuild.stderr")
	if err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return api.Status{
			Status: api.STATUS_FAILURE,
			Reason: strings.Join(errs, "\n"),
		}
	} else {
		return api.Status{
			Status: api.STATUS_SUCCESS,
			Result: GetLastBuildResult{
				Stdout: string(stdout),
				Stderr: string(stderr),
			}}
	}
}
