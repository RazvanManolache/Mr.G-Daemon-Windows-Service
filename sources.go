package main

import (
	"fmt"
	"os"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

func installSubApplication(subAppDef *SubApplication) bool {
	subApp := getCurrentSubApplication(subAppDef)
	logToMainFile(fmt.Sprintf("Installing subapplication: %s", subApp.Name))
	installLoc, err := getInstallLocation(subApp)
	if err != nil {
		logToFile("log", fmt.Sprintf("Failed to get install location for subapplication %s: %v", subApp.Name, err), nil)
		return false
	}
	if _, err := git.PlainClone(installLoc, false, &git.CloneOptions{
		URL:           subApp.RepoURL,
		ReferenceName: plumbing.NewBranchReferenceName(subApp.Branch),
		Progress:      os.Stdout,
	}); err != nil {
		logToMainFile(fmt.Sprintf("Failed to install subapplication %s: %v", subApp.Name, err))
		return false
	} else {
		logToMainFile(fmt.Sprintf("Installed subapplication %s", subApp.Name))
		subApp.Installed = true
		subApp.FirstRun = true
		saveSubApplications()
	}
	checkSymLinks(subApp)
	return true
}

func uninstallSubApplication(subAppDef *SubApplication) {
	subApp := getCurrentSubApplication(subAppDef)
	logToMainFile(fmt.Sprintf("Uninstalling subapplication: %s", subApp.Name))
	stopSubApplication(subApp)
	installLoc, err := getInstallLocation(subApp)
	if err != nil {
		logToFile("log", fmt.Sprintf("Failed to get install location for subapplication %s: %v", subApp.Name, err), nil)
		return
	}
	os.RemoveAll(installLoc)
	//remove from list
	for i, s := range subApplications {
		if s.Id == subApp.Id {
			subApplications = append(subApplications[:i], subApplications[i+1:]...)
			break
		}
	}
	saveSubApplications()
}

func checkIfSubApplicationHasUpdates(subAppDef *SubApplication) bool {
	subApp := getCurrentSubApplication(subAppDef)
	installLoc, err := getInstallLocation(subApp)
	if err != nil {
		logToFile("log", fmt.Sprintf("Failed to get install location for subapplication %s: %v", subApp.Name, err), nil)
		return false
	}
	r, err := git.PlainOpen(installLoc)
	defer broadcastToSocket("subapplications", subApplications)
	if err != nil {
		return false
		// logToMainFile(fmt.Sprintf("Application not found %s for update, installing: %v", subApp.Name, err))
		// os.RemoveAll(getInstallLocation(subApp))
		// installed := installSubApplication(subApp)
		// if !installed {
		// 	return false
		// }
	}
	w, err := r.Worktree()
	if err != nil {
		logToMainFile(fmt.Sprintf("Failed to get worktree for subapplication %s: %v", subApp.Name, err))
		return false
	}
	//git remote update
	err = r.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		logToMainFile(fmt.Sprintf("Failed to update subapplication %s: %v", subApp.Name, err))
		return false
	}

	err = w.Pull(&git.PullOptions{
		RemoteName:    "origin",
		ReferenceName: plumbing.NewBranchReferenceName(subApp.Branch),
		Progress:      os.Stdout,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		logToMainFile(fmt.Sprintf("Application is up to date %s: %v", subApp.Name, err))
		subApp.HasUpdates = false
		return false
	}
	subApp.HasUpdates = true
	return true

}

func updateSubApplication(subAppDef *SubApplication) bool {
	subApp := getCurrentSubApplication(subAppDef)
	logToMainFile(fmt.Sprintf("Updating subapplication: %s", subApp.Name))
	installLoc, err := getInstallLocation(subApp)
	if err != nil {
		logToFile("log", fmt.Sprintf("Failed to get install location for subapplication %s: %v", subApp.Name, err), nil)
		return false
	}
	r, err := git.PlainOpen(installLoc)
	if git.ErrRepositoryNotExists == err {

		logToMainFile(fmt.Sprintf("Application not found %s for update, installing: %v", subApp.Name, err))
		os.RemoveAll(installLoc)
		installed := installSubApplication(subApp)
		return installed
	}
	w, err := r.Worktree()
	if err != nil {
		logToMainFile(fmt.Sprintf("Failed to get worktree for subapplication %s: %v", subApp.Name, err))
		return false
	}
	if err := w.Pull(&git.PullOptions{
		RemoteName:    "origin",
		ReferenceName: plumbing.NewBranchReferenceName(subApp.Branch),
		Progress:      os.Stdout,
	}); err != nil && err != git.NoErrAlreadyUpToDate {
		logToMainFile(fmt.Sprintf("Failed to update subapplication %s: %v", subApp.Name, err))
	} else {
		logToMainFile(fmt.Sprintf("Updated subapplication %s", subApp.Name))
	}
	checkSymLinks(subApp)
	return true
}

func checkSymLinks(subApp *SubApplication) {
	installLoc, err := getInstallLocation(subApp)
	if err != nil {
		logToFile("log", fmt.Sprintf("Failed to get install location for subapplication %s: %v", subApp.Name, err), nil)
		return
	}
	links := subApp.SymLinks

	for source, destination := range links {
		createSymLink(installLoc, source, destination)
	}

}
