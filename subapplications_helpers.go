package main

import (
	"encoding/json"
	"fmt"
	"os"
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

}
