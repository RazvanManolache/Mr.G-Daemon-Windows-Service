package main

import (
	"errors"
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
	state := DeamonStatus{Name: serviceName, Config: CurrentConfig, SubApplications: subApplications}

	return &state, nil
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
		changes.add()
	case "put":
		changes.modify()
	case "delete":
		changes.remove()
	default:
		return nil, makeError("invalid operation", nil)
	}

	appStatus := listApplicationsInternal()

	return &appStatus, nil
}
