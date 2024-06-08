package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/uuid"
)

func getTodayAsString() string {
	return time.Now().Format("2006-01-02")
}

func generateId() (string, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return uuid.String(), nil
}

func getCurrentPath() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	if _, err := os.Stat(exPath); os.IsNotExist(err) {
		os.Mkdir(exPath, os.ModePerm)
	}
	return exPath, nil
}

func getFolderWithCreate(folder string, subfolders ...string) (string, int, error) {
	_, err := os.Stat(folder)
	if os.IsNotExist(err) {
		err = os.MkdirAll(folder, os.ModePerm)
		if err != nil {
			return "", -1, err
		}
	}
	var i int = 0
	for _, subfolder := range subfolders {
		folder = filepath.Join(folder, subfolder)
		_, err := os.Stat(folder)
		if os.IsNotExist(err) {
			err = os.MkdirAll(folder, os.ModePerm)
			if err != nil {
				return folder, i, err
			}
		}
		i++
	}
	return folder, i, nil
}

func createSymLink(installLoc string, source string, destination string) {
	dataloc, err := getDataLocation()
	if err != nil {
		logToFile("log", fmt.Sprintf("Failed to get data location for subapplication %s: %v", source, err), nil)
		return
	}
	sourcePath := source
	if !filepath.IsAbs(sourcePath) {
		sourcePath, _, err = getFolderWithCreate(installLoc, source)
		if err != nil {
			logToFile("log", fmt.Sprintf("Failed to get folder for source %s: %v", source, err), nil)
			return
		}
	}
	destinationPath := destination
	if !filepath.IsAbs(destinationPath) {
		destinationPath, _, err = getFolderWithCreate(dataloc, destination)
		if err != nil {
			logToFile("log", fmt.Sprintf("Failed to create destination folder for subapplication %s: %v", source, err), nil)
			return
		}
	}
	isSymLink, linkedFolder, err := isSymlink(sourcePath)
	if err != nil {
		logToFile("log", fmt.Sprintf("Failed to check symlink for subapplication %s: %v", source, err), nil)
		return
	}
	if !isSymLink {
		doSymLink(sourcePath, destinationPath)
		return
	}
	same, err := comparePaths(linkedFolder, destinationPath)
	if err != nil {
		logToFile("log", fmt.Sprintf("Failed to compare symlink for subapplication %s: %v", source, err), nil)
		return
	}
	if !same {
		doSymLink(sourcePath, destinationPath)
		return
	}

}

func doSymLink(source string, destination string) {

	err := moveAll(source, destination)
	if err != nil {
		logToFile("log", fmt.Sprintf("Failed to copy contents for subapplication %s: %v", source, err), nil)
	}

	//
	_, err = os.Stat(source)
	if err == nil || !os.IsNotExist(err) {
		err = os.Remove(source)
		if err != nil {
			logToFile("log", fmt.Sprintf("Failed to remove symlink for subapplication %s: %v", source, err), nil)
			return
		}
	}

	err = os.Symlink(destination, source)
	if err != nil {
		logToFile("log", fmt.Sprintf("Failed to create symlink for subapplication %s: %v", source, err), nil)
		return
	}
	logToFile("log", fmt.Sprintf("symlinked %s to %s", source, destination), nil)
}

func moveAll(srcDir, dstDir string) error {
	// Ensure source directory exists
	srcInfo, err := os.Stat(srcDir)
	if err != nil {
		return fmt.Errorf("source directory does not exist: %w", err)
	}

	if !srcInfo.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	// Ensure destination directory exists
	dstInfo, err := os.Stat(dstDir)
	if os.IsNotExist(err) {
		// Create destination directory if it does not exist
		err = os.MkdirAll(dstDir, srcInfo.Mode())
		if err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}
	} else if !dstInfo.IsDir() {
		return fmt.Errorf("destination is not a directory")
	}

	// Walk through the source directory
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the source directory itself
		if path == srcDir {
			return nil
		}

		// Determine the new path in the destination directory
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		newPath := filepath.Join(dstDir, relPath)

		// Move files and directories
		if info.IsDir() {
			err = os.MkdirAll(newPath, info.Mode())
			if err != nil {
				return err
			}
		} else {
			err = os.Rename(path, newPath)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Remove the source directory after moving its contents
	return os.RemoveAll(srcDir)
}

func isSymlink(path string) (bool, string, error) {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		return false, "", err
	}

	// Check if the file mode is a symbolic link
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		// Readlink returns the destination of the symbolic link
		linkTarget, err := os.Readlink(path)
		if err != nil {
			return true, "", err
		}
		return true, linkTarget, nil
	}

	return false, "", nil
}

func comparePaths(path1, path2 string) (bool, error) {
	// Clean and absolute paths
	absPath1, err := filepath.Abs(filepath.Clean(path1))
	if err != nil {
		return false, err
	}

	absPath2, err := filepath.Abs(filepath.Clean(path2))
	if err != nil {
		return false, err
	}

	// If on Windows, compare case-insensitively
	if runtime.GOOS == "windows" {
		return filepath.Clean(absPath1) == filepath.Clean(absPath2), nil
	}

	// On Unix-like systems, compare case-sensitively
	return absPath1 == absPath2, nil
}

func getDataLocation() (string, error) {
	if CurrentConfig.DataFolder == "" {
		CurrentConfig.DataFolder = "data"
		writeConfigFile()
	}
	if filepath.IsAbs(CurrentConfig.DataFolder) {
		folder, _, err := getFolderWithCreate(CurrentConfig.DataFolder)
		if err == nil {
			return folder, nil
		}
	}
	exPath, err := getCurrentPath()
	if err != nil {
		return "", err
	}
	folder, _, err := getFolderWithCreate(exPath, CurrentConfig.DataFolder)
	if err == nil {
		return folder, nil
	}
	folder, _, err = getFolderWithCreate(exPath, "data")
	if err == nil {
		return folder, nil
	}
	return "", fmt.Errorf("failed to get data location for subapplication %s", CurrentConfig.DataFolder)

}

func getInstallLocation(subApp *SubApplication) (string, error) {
	//if no path is provided, use the id
	if subApp.Path == "" {
		subApp.Path = subApp.Id
	}
	//if it's an absolute path, use it
	if filepath.IsAbs(subApp.Path) {
		folder, _, err := getFolderWithCreate(subApp.Path)
		if err == nil {
			return folder, nil
		}
	}
	if filepath.IsAbs(CurrentConfig.ApplicationFolder) {
		folder, cnt, err := getFolderWithCreate(CurrentConfig.ApplicationFolder, subApp.Path)
		if err == nil {
			return folder, nil
		}
		if cnt == 0 {
			folder, _, err := getFolderWithCreate(CurrentConfig.ApplicationFolder, subApp.Id)
			if err == nil {
				return folder, nil
			}
		}
	}
	exPath, err := getCurrentPath()
	if err != nil {
		return "", err
	}
	folder, cnt, err := getFolderWithCreate(exPath, CurrentConfig.ApplicationFolder, subApp.Path)
	if err == nil {
		return folder, nil
	}
	switch cnt {
	case -1:
		return "", err
	case 0:
		appfolder, _, err := getFolderWithCreate(exPath, "applications", subApp.Path)
		if err == nil {
			return appfolder, nil
		}
		directappfolder, _, err := getFolderWithCreate(exPath, subApp.Path)
		if err == nil {
			return directappfolder, nil
		}
	case 1:
		appfolder, _, err := getFolderWithCreate(exPath, CurrentConfig.ApplicationFolder, subApp.Id)
		if err == nil {
			subApp.Path = subApp.Id
			saveSubApplications()
			return appfolder, nil
		}
	}
	return "", fmt.Errorf("failed to get install location for subapplication %s", subApp.Name)

}

// an exercise in covering all the bases, using very explicit code
func getLogLocation(name string, subApp *SubApplication) (string, error) {

	if CurrentConfig.LogFolder == "" {
		CurrentConfig.LogFolder = "logs"
		writeConfigFile()
	}

	//if path is provided in name, use it instead, it should take precedence
	if filepath.IsAbs(name) {
		folder, _, err := getFolderWithCreate(name)
		if err == nil {
			if subApp != nil && subApp.LogLocation != folder {
				subApp.LogLocation = folder
				saveSubApplications()
			}
			return folder, nil
		}
	}

	if filepath.IsAbs(CurrentConfig.LogFolder) {
		//doesn't matter what it returns, best we can do is try
		folder, _, _ := getFolderWithCreate(CurrentConfig.LogFolder, name)
		if folder != "" {
			return folder, nil
		}

	}

	// name and logfolder are relative paths
	exPath, err := getCurrentPath()
	if err != nil {
		//this is a fatal error, no need to try to recover
		return "", err
	}
	//create subdirectory for logs
	folder, cnt, err := getFolderWithCreate(exPath, CurrentConfig.LogFolder, name)
	if err == nil {
		return folder, nil
	}
	switch cnt {
	case -1:
		//this is a fatal error, no need to try to recover, no idea what happened
		return "", err
	case 0:
		//log folder was created, maybe the LogFolder name is bad, we try again, if this fails we're done
		logsfolder, _, _ := getFolderWithCreate(exPath, "logs", name)
		if logsfolder != "" {
			return logsfolder, nil
		}

	}
	return folder, nil

}
