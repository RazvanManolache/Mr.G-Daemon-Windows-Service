package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Flag struct {
	Help     string      `json:"help"`
	Default  interface{} `json:"default"`
	Nargs    interface{} `json:"nargs"`
	Const    interface{} `json:"const"`
	Type     string      `json:"type"`
	Group    interface{} `json:"group"`
	Argument string      `json:"argument"`
	Metavar  interface{} `json:"metavar"`
}

type Group struct {
	Description string `json:"description"`
}

type FlagsAndGroups struct {
	Flags  map[string]Flag  `json:"flags"`
	Groups map[string]Group `json:"groups"`
}

// read json file and return FlagsAndGroups struct
func readFlagsAndGroups(appType string) FlagsAndGroups {
	var flagsAndGroups FlagsAndGroups

	runningPath, err := getCurrentPath()
	if err != nil {
		logToMainFile(fmt.Sprintf("Error getting running path: %v", err))
	}
	flagFile := appType + "Flags.json"
	fullPath := filepath.Join(runningPath, flagFile)

	configFile, err := os.Open(fullPath)

	if err != nil {
		fmt.Printf("Error opening flags file: %v", err)
	} else {
		err = json.NewDecoder(configFile).Decode(&flagsAndGroups)
		if err != nil {
			fmt.Printf("Error decoding flags file: %v", err)
		}
	}

	defer configFile.Close()
	return flagsAndGroups
}
