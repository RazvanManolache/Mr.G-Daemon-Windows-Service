package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func notifySubApplicationsStatusChange() {
	statuses := getStatusOnlyArray(subApplications)
	defer broadcastToSocket("statuses", statuses)
}

func getStatusOnlyArray(subApps []*SubApplication) []*SubApplicationStatus {
	var statusArray []*SubApplicationStatus
	for _, subApp := range subApps {
		statusArray = append(statusArray, subApp.getStatusOnly())
	}
	return statusArray
}

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
func stopAllSubApplications() {
	for _, subApp := range subApplications {
		subApp.stop()
	}
}

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
