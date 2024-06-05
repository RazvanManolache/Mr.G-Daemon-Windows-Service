package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

var subApplicationFile string = "subapplications.json"

// SubApplication represents a subprocess configuration
type SubApplication struct {
	Id                     string             `json:"id"`                     // Unique identifier for the subprocess
	Name                   string             `json:"name"`                   // Name of the subprocess
	CommandExec            string             `json:"commandExec"`            // Command to start the subprocess
	Command                string             `json:"command"`                // Command to start the subprocess
	RestartOnCriticalError bool               `json:"restartOnCriticalError"` // Indicates if the subprocess should be restarted on critical error
	CriticalErrorMessages  []string           `json:"criticalErrorMessages"`  // Messages that warrant restart
	AutoStart              bool               `json:"autoStart"`              // Indicates if the subprocess should be started automatically
	RepoURL                string             `json:"repoURL"`                // URL of the repository
	Branch                 string             `json:"branch"`                 // Branch to checkout
	Path                   string             `json:"path"`                   // Path to the repository
	AutoUpdate             bool               `json:"autoUpdate"`             // Indicates if the repository should be updated automatically
	Flags                  []string           `json:"flags"`                  // Flags to pass to the subprocess
	AppType                string             `json:"appType"`                // Type of the application
	FirstRun               bool               `json:"firstRun"`               // Indicates if the application is running for the first time
	Installed              bool               `json:"installed"`              // Indicates if the application is installed
	HasUpdates             bool               `json:"hasUpdates"`             // Indicates if the application has updates
	LogLocation            string             `json:"logLocation"`            // Location of the log files
	SetupCommand           string             `json:"setupCommand"`           // Command to run after installation
	LogFile                *os.File           `json:"-"`                      // Log file for the subprocess, don't serialize
	Context                context.Context    `json:"-"`                      // Process object for the subprocess
	Cmd                    *exec.Cmd          `json:"-"`                      // Process object for the subprocess
	CancelContext          context.CancelFunc `json:"-"`                      // Cancel function for the subprocess
	Running                bool               `json:"running"`                // Indicates if the subprocess is running
	Status                 string             `json:"status"`                 // Status of the subprocess
	SymLinks               map[string]string  `json:"symLinks"`               // Symlinks to create
}

type SubApplicationStatus struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}

func updateStatusSubApplication(subApp *SubApplication, status string) {
	subApp.Status = status
	notifySubApplicationsStatusChange()
}

func notifySubApplicationsStatusChange() {
	statuses := getStatusOnlyArray(subApplications)
	defer broadcastToSocket("statuses", statuses)
}

func getStatusOnly(subApp *SubApplication) *SubApplicationStatus {
	return &SubApplicationStatus{
		Id:     subApp.Id,
		Status: subApp.Status,
	}
}

var getStatusOnlyArray = func(subApps []*SubApplication) []*SubApplicationStatus {
	var statusArray []*SubApplicationStatus
	for _, subApp := range subApps {
		statusArray = append(statusArray, getStatusOnly(subApp))
	}
	return statusArray
}

func addSubApplication(subApp *SubApplication) *SubApplication {
	defer listApplicationsInternal()
	if subApp.Id == "" {
		id, err := generateId()
		if err != nil {
			logToMainFile(fmt.Sprintf("Error generating id: %v", err))
			return nil
		}
		subApp.Id = id
	}

	//find if exists
	for _, s := range subApplications {
		if s.Id == subApp.Id {
			return nil
		}
	}

	subApplications = append(subApplications, subApp)
	saveSubApplications()
	if subApp.AutoStart {
		startSubApplication(subApp)
	}
	return subApp
}

func removeSubApplication(appId string) {
	for i, s := range subApplications {
		if s.Id == appId {
			subApplications = append(subApplications[:i], subApplications[i+1:]...)
			saveSubApplications()
			return
		}
	}
}

func modifySubApplication(subApp *SubApplication) *SubApplication {
	for i, s := range subApplications {
		if s.Id == subApp.Id {
			var subApp = subApplications[i]
			var running = subApp.Running
			if running {
				stopSubApplication(subApp)
			}
			subApplications[i] = subApp
			saveSubApplications()
			if running {
				startSubApplication(subApp)
			}
			return subApp
		}
	}
	return nil
}

func readSubApplications() []*SubApplication {
	var subApplications []*SubApplication
	configFile, err := os.Open(subApplicationFile)

	if err != nil {
		logToMainFile(fmt.Sprintf("Error opening config file: %v", err))

	} else {
		err = json.NewDecoder(configFile).Decode(&subApplications)
		if err != nil {
			logToMainFile(fmt.Sprintf("Error decoding config file: %v", err))
		}
	}

	defer configFile.Close()
	return subApplications
}

func calculateFlags(subApp *SubApplication) {
	if subApp.AppType == "comfy" {
		subApp.Flags = append(subApp.Flags, "comfy")
	}
}

func runSetupCommand(subApp *SubApplication) error {
	if subApp.SetupCommand == "" {
		return nil
	}
	fullPath, err := getInstallLocation(subApp)
	if err != nil {
		logToFile("log", fmt.Sprintf("Failed to get install location for subapplication %s: %v", subApp.Name, err), subApp)
		return err
	}
	command := subApp.SetupCommand
	if strings.Contains(command, "$dir") {
		installLoc, err := getInstallLocation(subApp)
		if err != nil {
			logToMainFile(fmt.Sprintf("Failed to get install location for subapplication %s: %v", subApp.Name, err))
			return err
		}
		command = strings.Replace(command, "$dir", installLoc, -1)
	}
	commandParams := strings.Split(command, " ")
	command = commandParams[0]

	joinedRest := strings.Join(commandParams[1:], " ")

	ctx, _ := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, command)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CmdLine: joinedRest}
	cmd.Dir = fullPath
	err = cmd.Run()
	if err != nil {
		logToMainFile(fmt.Sprintf("Failed to run setup command for subapplication %s: %v", subApp.Name, err))
		return err
	}
	return nil
}

func startSubApplication(subAppDef *SubApplication) {
	subApp := getCurrentSubApplication(subAppDef)

	if subApp.AutoUpdate {
		updateSubApplication(subApp)
	}
	updateStatusSubApplication(subApp, "Starting")
	runSetupCommand(subApp)
	if subApp.FirstRun {
		calculateFlags(subApp)
		subApp.FirstRun = false
		saveSubApplications()
	}
	var err error
	subApp.LogFile, err = os.OpenFile(fmt.Sprintf("%s.log", subApp.Name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logToMainFile(fmt.Sprintf("Error opening log file for %s: %v", subApp.Name, err))
		updateStatusSubApplication(subApp, "Not started")
		return
	}
	defer subApp.LogFile.Close()

	if subApp.Context != nil && subApp.CancelContext != nil {
		logToFile("log", "Subprocess is already running", subApp)
		updateStatusSubApplication(subApp, "Running")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	subApp.Context = ctx
	subApp.CancelContext = cancel

	var command = subApp.Command
	var commandExec = subApp.CommandExec
	if len(subApp.Flags) > 0 {
		command = fmt.Sprintf("%s %s", command, strings.Join(subApp.Flags, " "))
	}

	fullPath, err := getInstallLocation(subApp)
	if err != nil {
		logToFile("log", fmt.Sprintf("Failed to get install location for subapplication %s: %v", subApp.Name, err), subApp)
		return
	}
	command = strings.Replace(command, "$dir", fullPath, -1)
	commandExec = strings.Replace(commandExec, "$dir", fullPath, -1)
	logToMainFile(fmt.Sprintf("Starting subprocess: %s", commandExec))
	logToMainFile(fmt.Sprintf("	with params: %s", command))
	logToMainFile(fmt.Sprintf("	in directory: %s", fullPath))
	cmd := exec.CommandContext(ctx, commandExec)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CmdLine: command}
	// pid := os.Getpid()
	// handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP} //, HideWindow: true, ParentProcess: handle}
	//get the current working directory

	cmd.Dir = fullPath

	stdoutReader, err := cmd.StdoutPipe()
	if err != nil {
		logToFile("log", fmt.Sprintf("Error creating stdout pipe: %v", err), subApp)
		cancel()
		updateStatusSubApplication(subApp, "Failed")
		return
	}

	stderrReader, err := cmd.StderrPipe()
	if err != nil {
		logToFile("log", fmt.Sprintf("Error creating stderr pipe: %v", err), subApp)
		cancel()
		updateStatusSubApplication(subApp, "Failed")
		return
	}

	// output := make(chan string)
	// errors := make(chan string)

	// stdoutReader, stdoutWriter := io.Pipe()
	// stderrReader, stderrWriter := io.Pipe()

	// cmd.Stdout = stdoutWriter
	// cmd.Stderr = stderrWriter

	// readPipe := func(reader *io.PipeReader, ch chan<- string) {
	// 	scanner := bufio.NewScanner(reader)
	// 	for scanner.Scan() {
	// 		line := scanner.Text()
	// 		ch <- line
	// 	}
	// 	if err := scanner.Err(); err != nil {
	// 		fmt.Printf("Error reading from pipe: %v\n", err)
	// 	}
	// 	close(ch)
	// }

	// // Start reading from the pipes
	// go readPipe(stdoutReader, output)
	// go readPipe(stderrReader, errors)

	go func() {
		// for {
		// 	msg := <-output
		// 	updateStatusSubApplication(subApp, "Running")
		// 	logToFile("console", msg, subApp)

		// 	if subApp.RestartOnCriticalError {
		// 		for _, message := range subApp.CriticalErrorMessages {
		// 			if strings.Contains(msg, message) {
		// 				logToFile("log", fmt.Sprintf("Detected critical message, restarting: %s", message), subApp, true)
		// 				restartSubApplication(subApp)
		// 			}
		// 		}
		// 	}
		// }
		scanner := bufio.NewScanner(stdoutReader)
		for scanner.Scan() {
			updateStatusSubApplication(subApp, "Running")
			logToFile("console", scanner.Text(), subApp)

			if subApp.RestartOnCriticalError {
				for _, message := range subApp.CriticalErrorMessages {
					if strings.Contains(scanner.Text(), message) {
						logToFile("log", fmt.Sprintf("Detected critical message, restarting: %s", message), subApp, true)
						restartSubApplication(subApp)
					}
				}
			}
		}
	}()

	go func() {
		// for {
		// 	msg := <-errors
		// 	logToFile("console", msg, subApp)
		// }
		scanner := bufio.NewScanner(stderrReader)
		for scanner.Scan() {
			logToFile("console", scanner.Text(), subApp)
		}
	}()

	err = cmd.Start()

	subApp.Cmd = cmd
	if err != nil {
		logToFile("log", fmt.Sprintf("Error starting %s: %v", subApp.Name, err), subApp, true)

		cancel()
		updateStatusSubApplication(subApp, "Failed")
		return
	}
	updateStatusSubApplication(subApp, "Running")
	subApp.Running = true
	logToFile("log", "Subprocess started", subApp, true)

}

func getCurrentSubApplication(subAppDef *SubApplication) *SubApplication {
	for _, subApp := range subApplications {
		if subApp.Id == subAppDef.Id {
			return subApp
		}
	}
	return nil

}

func stopSubApplication(subAppDef *SubApplication) {
	subApp := getCurrentSubApplication(subAppDef)

	if subApp.Context == nil || subApp.CancelContext == nil {
		logToFile("log", "Subprocess is not running", subApp)
		updateStatusSubApplication(subApp, "Stopped")
		return
	}
	updateStatusSubApplication(subApp, "Stopping")
	err := subApp.Cmd.Process.Kill()
	if err != nil {
		logToFile("log", fmt.Sprintf("Error stopping %s: %v", subApp.Name, err), subApp)
	}
	subApp.CancelContext()
	// d, e := syscall.LoadDLL("kernel32.dll")
	// if e != nil {

	// }
	// p, e := d.FindProc("GenerateConsoleCtrlEvent")
	// if e != nil {
	// }

	// r, _, e := p.Call(syscall.CTRL_BREAK_EVENT, uintptr(subApp.Cmd.Process.Pid))
	// if r == 0 {
	// 	//t.Fatalf("GenerateConsoleCtrlEvent: %v\n", e)
	// }
	subApp.Context = nil
	subApp.CancelContext = nil
	subApp.Running = false

	logToFile("log", "Subprocess stopped", subApp)
	updateStatusSubApplication(subApp, "Stopped")
}

func restartSubApplication(subAppDef *SubApplication) {
	subApp := getCurrentSubApplication(subAppDef)
	updateStatusSubApplication(subApp, "Restarting")
	stopSubApplication(subApp)
	time.Sleep(1 * time.Second)
	startSubApplication(subApp)
}

func checkSubApplicationUpdatesInternal() {
	var hasUpdates bool = false
	for app := range subApplications {
		subapp := subApplications[app]
		checkIfSubApplicationHasUpdates(subapp)
		if subapp.HasUpdates {
			hasUpdates = true
		}
	}
	if hasUpdates {
		broadcastToSocket("subapplications", subApplications)
	}
}

func stopAllSubApplications() {
	for _, subApp := range subApplications {
		stopSubApplication(subApp)
	}
}

func saveSubApplications() {
	configFile, err := os.Create(subApplicationFile)
	if err != nil {
		logToMainFile(fmt.Sprintf("Error creating config file: %v", err))
		return
	}
	defer configFile.Close()

	encoder := json.NewEncoder(configFile)
	encoder.SetIndent("", "    ")
	err = encoder.Encode(subApplications)
	if err != nil {
		logToMainFile(fmt.Sprintf("Error encoding config file: %v", err))
	}

}
