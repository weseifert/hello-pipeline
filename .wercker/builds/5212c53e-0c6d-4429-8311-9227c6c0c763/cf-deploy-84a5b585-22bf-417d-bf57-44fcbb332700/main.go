package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

//TODO: accept env from wercker container
//TODO: prefer flags over env
//TODO: validate parameters and supply meaningful output
//TODO: verify cf command exists, if not, install cf

const notSupplied = "<not-supplied>"

var (
	api         string
	usr         string
	pwd         string
	org         string
	spc         string
	appname     string
	errors      []string
	dockerImage string
)

func init() {
	flag.StringVar(&api, "api", notSupplied, "Target CF API URL. Overrides API env variable")
	flag.StringVar(&usr, "user", notSupplied, "CF User. Overrides USER env variable ")
	flag.StringVar(&pwd, "password", notSupplied, "CF Password. Overrides PASSWORD env variable")
	flag.StringVar(&org, "org", notSupplied, "CF Org. Overrides ORG env variable")
	flag.StringVar(&spc, "space", notSupplied, "CF Space. Overrides SPACE env variable")
	flag.StringVar(&appname, "appname", notSupplied, "Name of application to be pushed to Cloud Foundry. Overrides APPNAME env variable")
	flag.StringVar(&dockerImage, "docker-image", notSupplied, "Optional. Path to docker image.  Overrides DOCKER_IMAGE env variable")
}

func main() {
	flag.Parse()
	errors = make([]string, 0)

	api = reconcileWithEnvironment(api, "WERCKER_CF_DEPLOY_API", true)
	usr = reconcileWithEnvironment(usr, "WERCKER_CF_DEPLOY_USER", true)
	pwd = reconcileWithEnvironment(pwd, "WERCKER_CF_DEPLOY_PASSWORD", true)
	org = reconcileWithEnvironment(org, "WERCKER_CF_DEPLOY_ORG", true)
	spc = reconcileWithEnvironment(spc, "WERCKER_CF_DEPLOY_SPACE", true)
	appname = reconcileWithEnvironment(appname, "WERCKER_CF_DEPLOY_APPNAME", true)

	if len(errors) > 0 {
		for _, v := range errors {
			fmt.Println(v)
		}
		os.Exit(1)
	}

	fmt.Println("Downloading and installing CF CLI...")
	if ok := installCF(); !ok {
		fmt.Println("Unable to install CF CLI.")
		os.Exit(1)
	}
	fmt.Println("CF CLI installed.")

	fmt.Println("Logging in to Cloud Foundry...")
	if ok := loginCF(); !ok {
		fmt.Println("Unable to log in to Cloud Foundry.")
		os.Exit(1)
	}

	fmt.Println("Generating cf push command...")
	pushCommand := determinePushCommand()
	if len(pushCommand) == 0 {
		fmt.Println("Error: Push command not created.")
		fmt.Printf("API: %s\nUSR: %s\nPWD: %s\nORG: %s\nSPC: %s\n", api, usr, pwd, org, spc)
		os.Exit(1)
	}
	cmdString := strings.Join(pushCommand, " ")
	fmt.Printf("Generated Command: cf %s\n", cmdString)

	fmt.Println("Deploying app...")
	deployCommand := exec.Command("cf", pushCommand...)
	out, err := deployCommand.StdoutPipe()
	if err != nil {
		fmt.Printf("Error writing output to STDOUT: %s", err)
	}
	if err = deployCommand.Start(); err != nil {
		fmt.Printf("Error executing command: %s", err)
	}

	quit := make(chan bool)
	takeDump(out, quit)

	err = deployCommand.Wait()
	go func() { quit <- true }()
	if err != nil {
		fmt.Printf("ERROR OCCURRED: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("SUCCESS.\n")
	os.Exit(0)
}

func determinePushCommand() (cmd []string) {
	//Docker
	dockerImage = reconcileWithEnvironment(dockerImage, "WERCKER_CF_DEPLOY_DOCKER_IMAGE", false)

	if len(dockerImage) > 0 {
		commandString := fmt.Sprintf("push %s -o %s", appname, dockerImage)
		cmd = strings.Split(commandString, " ")
	}
	return
}

func installCF() bool {
	downloadCommand := exec.Command("wget", "-O", "cf.tgz", "https://cli.run.pivotal.io/stable?release=linux64-binary")
	err := downloadCommand.Run()
	if err != nil {
		fmt.Printf("Error retrieving cf binary: %s", err)
		return false
	}

	unzipCommand := exec.Command("tar", "-zxf", "cf.tgz")
	err = unzipCommand.Run()
	if err != nil {
		fmt.Printf("Error unpacking cf binary: %s", err)
		return false
	}
	return true
}

func loginCF() bool {
	loginCommand := exec.Command("cf", "login", "-a", api, "-u", usr, "-p", pwd, "-o", org, "-s", spc)
	out, err := loginCommand.StdoutPipe()
	if err != nil {
		fmt.Printf("Error connecting command output to STDOUT: %s", err)
		return false
	}
	if err = loginCommand.Start(); err != nil {
		fmt.Printf("Error executing CF login command: %s", err)
		return false
	}

	quit := make(chan bool)
	takeDump(out, quit)

	err = loginCommand.Wait()
	go func() { quit <- true }()
	if err != nil {
		fmt.Printf("Error logging in to CF: %s\n", err)
		return false
	}
	return true
}

func reconcileWithEnvironment(orig string, envName string, required bool) (result string) {
	result = orig
	if orig == notSupplied {
		result = os.Getenv(envName)
	}
	if len(result) == 0 && required {
		errors = append(errors, fmt.Sprintf("%s not supplied via flag or environment.", envName))
	}
	return
}

func takeDump(pipe io.ReadCloser, quit chan bool) {
	ch := make(chan string)

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := pipe.Read(buf)
			if n != 0 {
				ch <- string(buf[:n])
			}
			if err != nil {
				break
			}
		}
		close(ch)
	}()

loop:
	for {
		select {
		case s, ok := <-ch:
			if !ok {
				break loop
			}
			fmt.Print(s)
		case <-quit:
			break loop
		}
	}
}
