package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// notifySubApplicationsStatusChange notifies all connected clients of the status of all subapplications
func notifySubApplicationsStatusChange() {
	statuses := getStatusOnlyArray(subApplications)
	defer broadcastToSocket("statuses", statuses)
}

// getStatusOnlyArray returns an array of SubApplicationStatus objects from an array of SubApplication objects
func getStatusOnlyArray(subApps []*SubApplication) []*SubApplicationStatus {
	var statusArray []*SubApplicationStatus
	for _, subApp := range subApps {
		statusArray = append(statusArray, subApp.getStatusOnly())
	}
	return statusArray
}

// readSubApplications reads the subapplications from the subApplicationFile
func readSubApplications() ([]*SubApplication, error) {
	var subApplications []*SubApplication

	runningPath, err := getCurrentPath()
	if err != nil {
		logToMainFile(fmt.Sprintf("Error getting running path: %v", err))
		return nil, err
	}
	fullPath := filepath.Join(runningPath, subApplicationFile)

	configFile, err := os.Open(fullPath)

	if err != nil {
		logToMainFile(fmt.Sprintf("Error opening config file: %v", err))
		return nil, err

	} else {
		err = json.NewDecoder(configFile).Decode(&subApplications)
		if err != nil {
			logToMainFile(fmt.Sprintf("Error decoding config file: %v", err))
			return nil, err
		}
	}

	defer configFile.Close()
	return subApplications, nil
}

func autoStart() {
	for _, subApp := range subApplications {
		if subApp.AutoStart {
			subApp.start()
		}
	}
}

// checkSubApplicationUpdatesInternal checks for updates to all subapplications
func checkSubApplicationUpdatesInternal() {
	var hasUpdates bool = false
	for app := range subApplications {
		subapp := subApplications[app]
		subapp.checkUpdates()
		if subapp.HasUpdates {
			hasUpdates = true
		}
	}
	if hasUpdates {
		broadcastToSocket("subapplications", subApplications)
	}
}

// startAllSubApplications starts all subapplications that are set to autostart
func stopAllSubApplications() {
	for _, subApp := range subApplications {
		subApp.stop()
	}
}

// saveSubApplications saves the subapplications to the subApplicationFile
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
	broadcastToSocket("subapplications", subApplications)

}
