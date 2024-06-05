package main

import (
	"errors"
	"os"
	"strings"
	"sync"
)

var mu sync.Mutex

func makeError(msg string, err error) error {
	return errors.New(msg + ": " + err.Error())
}

func apiStatusInternal() (*DeamonStatus, error) {
	defer broadcastToSocket("config", CurrentConfig)
	defer broadcastToSocket("subapplications", subApplications)
	go getAllKits()
	state := DeamonStatus{Name: appName, Config: CurrentConfig, SubApplications: subApplications}

	return &state, nil
}

func stopService() {
	status_app = "Stopping"
	stopAllSubApplications()
	service.quit <- struct{}{}
	os.Exit(0)
}

func restartService() {
	stopService()
	status_app = "Restarting"
	runService(appName, false)
}
func listFlagsInternal(appName string) (*FlagsAndGroups, error) {

	// find application
	var app *SubApplication = nil
	for _, subApp := range subApplications {
		if subApp.Name == appName {
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

func listDiskSpaceInternal() ([]DiskSpace, error) {
	mu.Lock()
	defer mu.Unlock()

	// Get disk space
	diskSpace, err := GetTotalDiskSpaceForAllDrives()
	if err != nil {
		return nil, makeError("error getting disk space", err)
	}
	defer broadcastToSocket("diskinfo", diskSpace)
	return diskSpace, nil
}

func listApplicationsInternal() ApplicationStatus {

	state := ApplicationStatus{Status: status_app, SubApplications: subApplications}
	defer broadcastToSocket("subapplications", subApplications)
	return state

}

func appRequestHandlerInternal(changes SubApplication, operation string) (*ApplicationStatus, error) {
	mu.Lock()
	defer mu.Unlock()

	operation = strings.ToLower(operation)

	// find application
	var app *SubApplication = nil
	for _, subApp := range subApplications {
		if subApp.Id == changes.Id {
			app = subApp
		}
	}
	if app == nil {
		if operation != "post" {
			return nil, makeError("invalid app", nil)
		}
	} else {
		if operation == "post" {
			return nil, makeError("app already exists", nil)
		}
	}

	switch operation {
	case "post":
		addSubApplication(&changes)
	case "put":
		modifySubApplication(&changes)
	case "delete":
		removeSubApplication(changes.Id)
	default:
		return nil, makeError("invalid operation", nil)
	}

	appStatus := listApplicationsInternal()

	return &appStatus, nil
}
