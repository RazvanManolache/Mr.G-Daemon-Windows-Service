package main

import (
	"log"
	"os"
)

type myService struct {
	quit chan struct{}
	done chan struct{}
}

func run() {
	var err error = nil
	subApplications, err = readSubApplications()
	if err != nil {
		logToMainFile("Could not read configuration file for applications.")
	}

	go readConfigFile()
	go startServer()

	scheduler()
	//detectGPU()
	detectGPU_Windows()
	autoStart()
}

func runInteractive() {
	log.Print("Running in interactive mode")
	service := newMyService()
	run()
	baseLoop(service.quit, service.done)
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
