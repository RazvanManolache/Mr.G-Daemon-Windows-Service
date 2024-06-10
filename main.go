package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var serviceName = "MrG.Daemon"
var niceServiceName = "Mr.G Daemon"

func main() {

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
						subApp.install()
					}
				}
			case "appupdate":
				for _, subApp := range subApplications {
					if subApp.Name == commands[1] {
						subApp.update()
					}
				}
			case "appstart":
				if commands[1] == "all" {
					for _, subApp := range subApplications {
						subApp.start()
					}
				} else {
					for _, subApp := range subApplications {
						if subApp.Name == commands[1] {
							subApp.start()
						}
					}
				}
			case "appstop":
				if commands[1] == "all" {
					stopAllSubApplications()
				} else {
					for _, subApp := range subApplications {
						if subApp.Name == commands[1] {
							subApp.stop()
						}
					}
				}
			case "apprestart":
				if commands[1] == "all" {
					for _, subApp := range subApplications {
						subApp.restart()
					}
				} else {
					for _, subApp := range subApplications {
						if subApp.Name == commands[1] {
							subApp.restart()
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
		subApp.stop()
	}
	close(done)
}
