package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var appName = "MrG.AI.Daemon"

func main() {
	go readConfigFile()
	go startServer()
	go scheduler()
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

var subApplications []*SubApplication

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
			case "stopservice":
				stopService()
			case "restartservice":
				restartService()
			case "install":
				for _, subApp := range subApplications {
					if subApp.Name == commands[1] {
						installSubApplication(subApp)
					}
				}
			case "update":
				for _, subApp := range subApplications {
					if subApp.Name == commands[1] {
						updateSubApplication(subApp)
					}
				}
			case "start":
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
			case "stop":
				if commands[1] == "all" {
					stopAllSubApplications()
				} else {
					for _, subApp := range subApplications {
						if subApp.Name == commands[1] {
							stopSubApplication(subApp)
						}
					}
				}
			case "restart":
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
