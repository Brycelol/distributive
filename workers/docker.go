package workers

import (
	"github.com/CiscoCloud/distributive/tabular"
	"github.com/CiscoCloud/distributive/wrkutils"
	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"os/exec"
	"strings"
)

// RegisterDocker registers these checks so they can be used.
func RegisterDocker() {
	wrkutils.RegisterCheck("dockerimage", dockerImage, 1)
	wrkutils.RegisterCheck("dockerrunning", dockerRunning, 1)
	wrkutils.RegisterCheck("dockerrunningapi", dockerRunningAPI, 2)
	wrkutils.RegisterCheck("dockerimageregexp", dockerImageRegexp, 1)
	wrkutils.RegisterCheck("dockerrunningregexp", dockerRunningRegexp, 1)
}

// getDockerImages returns a list of all downloaded Docker images
func getDockerImages() (images []string) {
	cmd := exec.Command("docker", "images")
	return wrkutils.CommandColumnNoHeader(0, cmd)
}

// dockerImage checks to see that the specified Docker image (e.g. "user/image",
// "ubuntu", etc.) is downloaded (pulled) on the host
func dockerImage(parameters []string) (exitCode int, exitMessage string) {
	name := parameters[0]
	images := getDockerImages()
	if tabular.StrIn(name, images) {
		return 0, ""
	}
	return wrkutils.GenericError("Docker image was not found", name, images)
}

// dockerImageRegexp is like dockerImage, but with a regexp match instead
func dockerImageRegexp(parameters []string) (exitCode int, exitMessage string) {
	re := wrkutils.ParseUserRegex(parameters[0])
	images := getDockerImages()
	if tabular.ReIn(re, images) {
		return 0, ""
	}
	return wrkutils.GenericError("Docker image was not found", re.String(), images)
}

// getRunningContainers returns a list of names of running docker containers
func getRunningContainers() (containers []string) {
	out, err := exec.Command("docker", "ps", "-a").CombinedOutput()
	outstr := string(out)
	// `docker images` requires root permissions
	if err != nil && strings.Contains(outstr, "permission denied") {
		log.Fatal("Permission denied when running: docker ps -a")
	}
	if err != nil {
		log.Fatal("Error while running `docker ps -a`" + "\n\t" + err.Error())
	}
	// the output of `docker ps -a` has spaces in columns, but each column
	// is separated by 2 or more spaces
	//lines := tabular.ProbabalisticSplit(outstr)
	lines := tabular.ProbabalisticSplit(outstr)
	if len(lines) < 1 {
		return []string{}
	}
	names := tabular.GetColumnByHeader("image", lines)
	statuses := tabular.GetColumnByHeader("status", lines)
	for i, status := range statuses {
		if strings.Contains(status, "Up") && len(names) > i {
			containers = append(containers, names[i])
		}
	}
	return containers
}

// getRunningContainersAPI is like getRunningContainers, but uses an external
// library in order to access the Docker API
func getRunningContainersAPI(path string) (containers []string) {
	endpoint := path
	client, err := docker.NewClient(endpoint)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Fatal("Couldn't create Docker API client")
	}
	ctrs, err := client.ListContainers(docker.ListContainersOptions{All: false})
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Fatal("Couldn't list Docker containers")
	}
	for _, ctr := range ctrs {
		if strings.Contains(ctr.Status, "Up") {
			containers = append(containers, ctr.Image)
		}
	}
	return containers
}

// dockerRunning checks to see if a specified docker container is running
// (e.g. "user/container")
func dockerRunning(parameters []string) (exitCode int, exitMessage string) {
	name := parameters[0]
	running := getRunningContainers()
	if tabular.StrContainedIn(name, running) {
		return 0, ""
	}
	return wrkutils.GenericError("Docker container not runnning", name, running)
}

// dockerRunningAPI is like dockerRunning, but fetches its information from
// getRunningContainersAPI.
func dockerRunningAPI(parameters []string) (exitCode int, exitMessage string) {
	name := parameters[1]
	running := getRunningContainersAPI(parameters[0])
	if tabular.StrContainedIn(name, running) {
		return 0, ""
	}
	return wrkutils.GenericError("Docker container not runnning", name, running)
}

// dockerRunningRegexp is like dockerRunning, but with a regexp match instead
func dockerRunningRegexp(parameters []string) (exitCode int, exitMessage string) {
	re := wrkutils.ParseUserRegex(parameters[0])
	running := getRunningContainers()
	if tabular.ReIn(re, running) {
		return 0, ""
	}
	msg := "Docker container not runnning"
	return wrkutils.GenericError(msg, re.String(), running)
}
