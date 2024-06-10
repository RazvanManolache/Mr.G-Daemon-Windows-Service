package main

import (
	"bufio"
	"context"
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

var subApplications []*SubApplication

// updateStatus updates the status of the subprocess

func (subApp *SubApplication) updateStatus(status string) {
	subApp.Status = status
	notifySubApplicationsStatusChange()
}

// getStatusOnly returns a SubApplicationStatus object with only the id and status
func (subApp *SubApplication) getStatusOnly() *SubApplicationStatus {
	return &SubApplicationStatus{
		Id:     subApp.Id,
		Status: subApp.Status,
	}
}

// calculateFlags calculates the flags for the subprocess
func (subApp *SubApplication) calculateFlags() {
	if subApp.AppType == "comfy" {
		subApp.Flags = append(subApp.Flags, "comfy")
	}
}

// runSetupCommand runs the setup command for the subprocess
func (subApp *SubApplication) runSetupCommand() error {
	if subApp.SetupCommand == "" {
		return nil
	}
	fullPath, err := getInstallLocation(subApp)
	if err != nil {
		logToFile("log", fmt.Sprintf("Failed to get install location for subapplication %s: %v", subApp.Name, err), subApp)
		return err
	}
	command := subApp.SetupCommand
	command = strings.Replace(command, "$dir", fullPath, -1)
	commandParams := strings.Split(command, " ")
	command = commandParams[0]

	joinedRest := strings.Join(commandParams[1:], " ")

	cmd, err := subApp.createCommand(command)
	subApp.CancelContext = nil
	subApp.Context = nil
	subApp.Cmd = nil
	if err != nil {
		logToFile("log", fmt.Sprintf("Error creating command for subapplication %s: %v", subApp.Name, err), subApp)
		subApp.updateStatus("Failed")
		return err
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CmdLine: joinedRest}
	cmd.Dir = fullPath
	logToFile("log", fmt.Sprintf("Running setup command for subapplication %s: %s", subApp.Name, command), subApp)
	logToFile("log", fmt.Sprintf("	with params: %s", joinedRest), subApp)

	err = cmd.Start()
	if err != nil {
		logToMainFile(fmt.Sprintf("Failed to run setup command for subapplication %s: %v", subApp.Name, err))
		return err
	}
	cmd.Wait()
	return nil
}

// createCommand creates a command for the subprocess
func (subApp *SubApplication) createCommand(command string) (*exec.Cmd, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, command)
	subApp.CancelContext = cancel
	subApp.Context = ctx
	subApp.Cmd = cmd
	stdoutReader, err := cmd.StdoutPipe()
	if err != nil {
		logToFile("log", fmt.Sprintf("Error creating stdout pipe: %v", err), subApp)
		cancel()
		subApp.updateStatus("Failed")
		return nil, err
	}

	stderrReader, err := cmd.StderrPipe()
	if err != nil {
		logToFile("log", fmt.Sprintf("Error creating stderr pipe: %v", err), subApp)
		cancel()
		subApp.updateStatus("Failed")
		return nil, err
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
			subApp.updateStatus("Running")
			logToFile("console", scanner.Text(), subApp)

			if subApp.RestartOnCriticalError {
				for _, message := range subApp.CriticalErrorMessages {
					if strings.Contains(scanner.Text(), message) {
						logToFile("log", fmt.Sprintf("Detected critical message, restarting: %s", message), subApp, true)
						subApp.restart()
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

	return cmd, nil
}

func (subAppDef *SubApplication) start() {
	subApp := subAppDef.getCurrent()

	if subApp.AutoUpdate {
		subApp.update()
	}
	subApp.updateStatus("Starting")
	if subApp.FirstRun {
		subApp.calculateFlags()
		subApp.FirstRun = false
		saveSubApplications()
	}
	var err error
	subApp.LogFile, err = os.OpenFile(fmt.Sprintf("%s.log", subApp.Name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logToMainFile(fmt.Sprintf("Error opening log file for %s: %v", subApp.Name, err))
		subApp.updateStatus("Not started")
		return
	}
	defer subApp.LogFile.Close()

	if subApp.Context != nil && subApp.CancelContext != nil {
		logToFile("log", "Subprocess is already running", subApp)
		subApp.updateStatus("Running")
		return
	}

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

	cmd, err := subApp.createCommand(subApp.CommandExec)
	if err != nil {
		logToFile("log", fmt.Sprintf("Error creating command for subapplication %s: %v", subApp.Name, err), subApp)
		subApp.updateStatus("Failed")
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CmdLine: command}
	// pid := os.Getpid()
	// handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP} //, HideWindow: true, ParentProcess: handle}
	//get the current working directory

	cmd.Dir = fullPath
	err = cmd.Start()

	if err != nil {
		logToFile("log", fmt.Sprintf("Error starting %s: %v", subApp.Name, err), subApp, true)

		subApp.CancelContext()
		subApp.updateStatus("Failed")
		return
	}
	subApp.updateStatus("Running")
	subApp.Running = true
	logToFile("log", "Subprocess started", subApp, true)

}

func (subAppDef *SubApplication) stop() {
	subApp := subAppDef.getCurrent()

	if subApp.Context == nil || subApp.CancelContext == nil {
		logToFile("log", "Subprocess is not running", subApp)
		subApp.updateStatus("Stopped")
		return
	}
	subApp.updateStatus("Stopping")
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
	subApp.Cmd = nil
	subApp.CancelContext = nil
	subApp.Running = false

	logToFile("log", "Subprocess stopped", subApp)
	subApp.updateStatus("Stopped")
}

// restart restarts the subprocess
func (subAppDef *SubApplication) restart() {
	subApp := subAppDef.getCurrent()
	subApp.updateStatus("Restarting")
	subApp.stop()
	time.Sleep(1 * time.Second)
	subApp.start()
}

// getCurrent returns the current subprocess based on the id
func (subAppDef *SubApplication) getCurrent() *SubApplication {
	for _, subApp := range subApplications {
		if subApp.Id == subAppDef.Id {
			return subApp
		}
	}
	return nil

}

// modify modifies the subprocess, restarting it if necessary
func (subApp *SubApplication) modify() *SubApplication {
	for i, s := range subApplications {
		if s.Id == subApp.Id {
			var subApp = subApplications[i]
			var running = subApp.Running
			if running {
				subApp.stop()
			}
			subApplications[i] = subApp
			saveSubApplications()
			if running {
				subApp.start()
			}
			return subApp
		}
	}
	return nil
}

// add adds a subprocess to the list
func (subApp *SubApplication) add() *SubApplication {
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
		subApp.start()
	}
	return subApp
}

// remove removes a subprocess from the list

func (subApp *SubApplication) remove() {
	for i, s := range subApplications {
		if s.Id == subApp.Id {
			if subApp.Running {
				subApp.stop()
			}
			subApplications = append(subApplications[:i], subApplications[i+1:]...)
			saveSubApplications()
			return
		}
	}
}

// listFlags lists the flags for the subprocess
func (sub *SubApplication) listFlags() (*FlagsAndGroups, error) {

	// find application
	var app *SubApplication = nil
	for _, subApp := range subApplications {
		if subApp.Name == sub.Name || subApp.Id == sub.Id {
			app = subApp
		}
	}
	if app == nil {
		return nil, makeError("invalid app", nil)
	}

	// get type of app and get flags
	flagsAndGroups := readFlagsAndGroups(app.AppType)
	defer broadcastToSocket("flags", flagsAndGroups)

	return &flagsAndGroups, nil
}
