package main

import (
	"log"
	"os"

	"golang.org/x/sys/windows/svc"
)

type myService struct {
	quit chan struct{}
	done chan struct{}
}

func runApplication() {
	IsWindowsService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}
	if !IsWindowsService {
		runInteractive()
	} else {
		runService(serviceName, false)
	}
}

func run() {
	go readConfigFile()
	go startServer()

	scheduler()
	detectGPU()
	detectGPU_Windows()

	subApplications = readSubApplications()

	for _, subApp := range subApplications {
		if subApp.AutoStart {
			subApp.start()
		}
	}
}

func runInteractive() {
	log.Print("Running in interactive mode")
	service := newMyService()
	go run()
	go baseLoop(service.quit, service.done)
}

var status_app string = "starting"

func restartService() {
	softStopService()
	status_app = "Restarting"
	runService(serviceName, false)
}

func softStopService() {
	status_app = "Stopping"
	stopAllSubApplications()
	service.quit <- struct{}{}
	os.Exit(0)
}
