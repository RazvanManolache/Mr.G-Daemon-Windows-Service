package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

var configFile = "config.json"

var mainAppKitRepository = "https://github.com/RazvanManolache/Mr.G-Daemon-Kits-List"

var CurrentConfig Config

type Config struct {
	CheckDisksInterval                 int      `json:"checkDisksInterval"`
	CheckSubApplicationsInterval       int      `json:"checkSubApplicationsInterval"`
	CheckSubApplicationsUpdateInterval int      `json:"checkSubApplicationsUpdateInterval"`
	ApplicationFolder                  string   `json:"applicationFolder"`
	LogFolder                          string   `json:"logFolder"`
	DataFolder                         string   `json:"dataFolder"`
	AppKitRepositories                 []string `json:"appKitRepositories"`
}

func getConfigFile() (string, error) {
	runningPath, err := getCurrentPath()
	if err != nil {
		logToMainFile(fmt.Sprintf("Error getting running path: %v", err))
		return "", nil
	}
	fullPath := filepath.Join(runningPath, configFile)
	return fullPath, nil
}

func readConfigFile() (*Config, error) {
	configFilePath, err := getConfigFile()
	if err != nil {
		logToMainFile(fmt.Sprintf("Error opening config file: %v", err))
		return nil, err
	}

	configFile, err := os.Open(configFilePath)

	if err != nil {
		logToMainFile(fmt.Sprintf("Error opening config file: %v", err))
		return nil, err

	} else {
		err = json.NewDecoder(configFile).Decode(&CurrentConfig)
		if err != nil {
			logToMainFile(fmt.Sprintf("Error decoding config file: %v", err))
			return nil, err
		}
	}

	defer configFile.Close()
	defer broadcastToSocket("config", CurrentConfig)
	return &CurrentConfig, nil
}

func writeConfigFile() (*Config, error) {
	configFilePath, err := getConfigFile()
	if err != nil {
		logToMainFile(fmt.Sprintf("Error opening config file: %v", err))
		return nil, err
	}
	configFile, err := os.Create(configFilePath)

	if err != nil {
		logToMainFile(fmt.Sprintf("Error creating config file: %v", err))
		return nil, err
	} else {
		err = json.NewEncoder(configFile).Encode(CurrentConfig)
		if err != nil {
			logToMainFile(fmt.Sprintf("Error encoding config file: %v", err))
			return nil, err
		}
	}

	defer configFile.Close()
	return readConfigFile()
}

func updateConfigFile(data map[string]string) (*Config, error) {

	if data != nil {

		for key, value := range data {
			switch key {
			case "checkDisksInterval":
				intValue, err := strconv.Atoi(value)
				if err == nil {
					CurrentConfig.CheckDisksInterval = intValue
				}

			case "checkSubApplicationsInterval":
				intValue, err := strconv.Atoi(value)
				if err == nil {
					CurrentConfig.CheckSubApplicationsInterval = intValue
				}

			case "checkSubApplicationsUpdateInterval":
				intValue, err := strconv.Atoi(value)
				if err == nil {
					CurrentConfig.CheckSubApplicationsUpdateInterval = intValue
				}
			case "applicationFolder":
				CurrentConfig.ApplicationFolder = value
			case "logFolder":
				CurrentConfig.LogFolder = value
			case "dataFolder":
				CurrentConfig.DataFolder = value
			}

		}

		return writeConfigFile()
	}

	return &CurrentConfig, nil
}
