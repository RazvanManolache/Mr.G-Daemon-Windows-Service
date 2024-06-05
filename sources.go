package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

type GitHubContent struct {
	Content string `json:"content"`
}

func constructGitHubAPIURL(repo string, path string) string {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		fmt.Println("Invalid repository format. Use 'owner/repo'")
		return ""
	}
	owner, repoName := parts[0], parts[1]
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repoName, path)
}

func readGitHubFile(repo string, path string) (string, error) {
	url := constructGitHubAPIURL(repo, path)
	if url == "" {
		return "", fmt.Errorf("invalid repository format")
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error: HTTP status %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	var content GitHubContent
	err = json.Unmarshal(body, &content)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(content.Content)
	if err != nil {
		return "", fmt.Errorf("error decoding base64 content: %v", err)
	}

	return string(decoded), nil
}

func getAllKits() []SubApplication {

	var kits []SubApplication = getKitList(mainAppKitRepository)
	for _, repo := range CurrentConfig.AppKitRepositories {
		other := getKitList(repo)
		for _, k := range other {
			//probably should add more validations here
			found := false
			for _, kit := range kits {
				if kit.Id == k.Id {
					found = true
					break
				}
			}
			if !found {
				kits = append(kits, k)
			}

		}
	}
	broadcastToSocket("kits", kits)
	return kits
}

func getKitList(repoUrl string) []SubApplication {
	content, err := readGitHubFile(repoUrl, "list.json")
	if err != nil {
		logToMainFile(fmt.Sprintf("Failed to get kits list: %v", err))
		return nil
	}

	var kits []SubApplication
	err = json.Unmarshal([]byte(content), &kits)
	if err != nil {
		logToMainFile(fmt.Sprintf("Failed to unmarshal kits list: %v", err))
		return nil
	}
	return kits
}

func installSubApplication(subAppDef *SubApplication) bool {
	subApp := getCurrentSubApplication(subAppDef)
	logToMainFile(fmt.Sprintf("Installing subapplication: %s", subApp.Name))
	installLoc, err := getInstallLocation(subApp)
	if err != nil {
		logToFile("log", fmt.Sprintf("Failed to get install location for subapplication %s: %v", subApp.Name, err), nil)
		return false
	}
	repo, err := git.PlainClone(installLoc, false, &git.CloneOptions{
		URL:           subApp.RepoURL,
		ReferenceName: plumbing.NewBranchReferenceName(subApp.Branch),
		Progress:      os.Stdout,
	})
	if err != nil {
		logToMainFile(fmt.Sprintf("Failed to install subapplication %s: %v", subApp.Name, err))
		return false
	}
	// init submodules
	err = updateSubModules(repo)
	if err != nil {
		return false
	}
	//run setup comand
	err = runSetupCommand(subApp)
	if err != nil {
		return false
	}

	logToMainFile(fmt.Sprintf("Installed subapplication %s", subApp.Name))
	subApp.Installed = true
	subApp.FirstRun = true
	saveSubApplications()

	checkSymLinks(subApp)
	return true
}

func updateSubModules(repo *git.Repository) error {
	w, err := repo.Worktree()
	if err != nil {
		logToMainFile(fmt.Sprintf("Failed to get worktree for subapplication: %v", err))
		return err
	}

	// Update and initialize submodules
	submodules, err := w.Submodules()
	if err != nil {
		logToMainFile(fmt.Sprintf("Failed to get submodules for subapplication: %v", err))
		return err
	}

	for _, submodule := range submodules {
		err = submodule.Init()
		if err != nil && err != git.ErrSubmoduleAlreadyInitialized {
			logToMainFile(fmt.Sprintf("Failed to initialize submodule: %v", err))
			return err
		}

		err = submodule.Update(&git.SubmoduleUpdateOptions{
			Init:              true,
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		})
		if err != nil {
			logToMainFile(fmt.Sprintf("Failed to update submodule: %v", err))
			return err
		}
	}
	return nil
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
	err = w.Pull(&git.PullOptions{
		RemoteName:    "origin",
		ReferenceName: plumbing.NewBranchReferenceName(subApp.Branch),
		Progress:      os.Stdout,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		logToMainFile(fmt.Sprintf("Failed to update subapplication %s: %v", subApp.Name, err))
		return false
	}

	err = updateSubModules(r)
	if err != nil {
		return false
	}
	err = runSetupCommand(subApp)
	if err != nil {
		return false
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
