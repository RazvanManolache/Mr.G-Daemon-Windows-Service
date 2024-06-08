package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var serviceName = "MrG.AI.Daemon"
var niceServiceName = "Mr.G AI Daemon"

var subApplications []*SubApplication

func main() {
	readConfigFile()
	err := startServer()
	if err != nil {
		logToMainFile(fmt.Sprintf("Error starting server: %v", err))
		fmt.Printf("Error starting server: %v\n", err)
		return
	}

	scheduler()
	detectGPU()
	detectGPU_Windows()
	runApplication()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		stopAllSubApplications()
		os.Exit(1)
	}()
}

// Implement mainService
func baseLoop(quit <-chan struct{}, done chan<- struct{}) {
	subApplications = readSubApplications()

	for _, subApp := range subApplications {
		if subApp.AutoStart {
			startSubApplication(subApp)
		}
	}

	go func() {
		for {
			var command string
			fmt.Print("Enter command (start, stop, restart, exit): ")
			fmt.Scanln(&command)
			commands := strings.Split(command, " ")

			if len(commands) < 2 {
				logToMainFile("Invalid command")
				continue
			}

			switch commands[0] {
			case "install":
				installService(serviceName, niceServiceName)
				return
			case "remove":
				removeService(serviceName)
				return
			case "start":
				startService(serviceName)
				return
			case "stop":
				stopService(serviceName)
				return
			case "stopservice":
				softStopService()
			case "restartservice":
				restartService()
			case "appinstall":
				for _, subApp := range subApplications {
					if subApp.Name == commands[1] {
						installSubApplication(subApp)
					}
				}
			case "appupdate":
				for _, subApp := range subApplications {
					if subApp.Name == commands[1] {
						updateSubApplication(subApp)
					}
				}
			case "appstart":
				if commands[1] == "all" {
					for _, subApp := range subApplications {
						startSubApplication(subApp)
					}
				} else {
					for _, subApp := range subApplications {
						if subApp.Name == commands[1] {
							startSubApplication(subApp)
						}
					}
				}
			case "appstop":
				if commands[1] == "all" {
					stopAllSubApplications()
				} else {
					for _, subApp := range subApplications {
						if subApp.Name == commands[1] {
							stopSubApplication(subApp)
						}
					}
				}
			case "apprestart":
				if commands[1] == "all" {
					for _, subApp := range subApplications {
						restartSubApplication(subApp)
					}
				} else {
					for _, subApp := range subApplications {
						if subApp.Name == commands[1] {
							restartSubApplication(subApp)
						}
					}
				}

			default:
				logToMainFile(fmt.Sprintf("Unknown command: %s", command))
			}
		}
	}()

	<-quit
	for _, subApp := range subApplications {
		stopSubApplication(subApp)
	}
	close(done)
}
