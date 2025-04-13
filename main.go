package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
)

type DeployRequest struct {
	ApplicationName      string            `json:"application_name"`
	EnvironmentVars      map[string]string `json:"environment_vars"`
	ExtraFilesToDownload []string          `json:"extra_files_to_download"`
}

// Deploy token which requests are authenticated against.
var deployToken string

// GitHub token used to fetch the docker compose file and pull the image.
var gitHubToken string

// Server port to listen on.
var portNumber string

func main() {
	getAndSetEnvVars()

	http.HandleFunc("POST /deploy", deployHandler)

	fmt.Printf("Server listening on port %s.\n", portNumber)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+portNumber, nil))
}

func getAndSetEnvVars() {
	if deployToken = os.Getenv("DEPLOY_TOKEN"); deployToken == "" {
		log.Fatal("DEPLOY_TOKEN environment variable not set.")
	}

	if gitHubToken = os.Getenv("GITHUB_TOKEN"); gitHubToken == "" {
		log.Fatal("GITHUB_TOKEN environment variable not set.")
	}

	if portNumber = os.Getenv("PORT"); portNumber == "" {
		portNumber = "9090"
		log.Println("PORT environment variable not set. Using default port 9090.")
	}
}

func deployHandler(w http.ResponseWriter, r *http.Request) {
	var deployRequest DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&deployRequest); err != nil {
		http.Error(w, "Invalid JSON payload.", http.StatusBadRequest)
		return
	}

	suppliedToken := r.Header.Get("Authorization")
	if suppliedToken != deployToken {
		http.Error(w, "Invalid token.", http.StatusUnauthorized)
		return
	}

	switch {
	case deployRequest.ApplicationName != "":
		err := handleDeployment(deployRequest)
		if err != nil {
			http.Error(w, "Failed to update Docker container.", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(fmt.Appendf(nil, "Application '%s' updated successfully.\n", deployRequest.ApplicationName))
	default:
		http.Error(w, "No application name provided.", http.StatusBadRequest)
	}
}

func handleDeployment(deployRequest DeployRequest) error {
	if err := downloadFile(deployRequest.ApplicationName, "docker-compose.yml"); err != nil {
		return fmt.Errorf("Failed to download docker-compose.yml: %w", err)
	}

	for _, fileName := range deployRequest.ExtraFilesToDownload {
		if err := downloadFile(deployRequest.ApplicationName, fileName); err != nil {
			return fmt.Errorf("Failed to download %s: %w", fileName, err)
		}
	}

	if err := loginToGitHubContainerRegistry(); err != nil {
		return err
	}

	if err := deployDockerCompose(deployRequest.EnvironmentVars); err != nil {
		return err
	}

	removeDanglingImages()

	log.Printf("Application '%s' updated successfully.\n", deployRequest.ApplicationName)
	return nil
}

func removeDanglingImages() {
	cmd := exec.Command("docker", "image", "prune", "-f")
	if err := cmd.Run(); err != nil {
		log.Println("Failed to remove dangling images:", err)
		return
	}
	log.Println("Removed dangling images.")
}

// Pulls latest images and (re)starts the Docker containers.
func deployDockerCompose(environmentVars map[string]string) error {
	envVars := ""
	for key, value := range environmentVars {
		envVars += fmt.Sprintf("%s=%s ", key, value)
	}
	command := "docker compose pull && " + envVars + " docker compose up -d --remove-orphans"
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error updating Docker: %s\nOutput: %s\n", err.Error(), string(output))
		return err
	}
	return nil
}

// Login to the GitHub Container Registry (ghcr.io).
func loginToGitHubContainerRegistry() error {
	cmdLogin := exec.Command("docker", "login", "ghcr.io", "-u", "nielshoek", "-p", gitHubToken)

	if err := cmdLogin.Run(); err != nil {
		log.Println("Failed to login to GitHub Container Registry:", err)
		return err
	}

	log.Println("Logged in to GitHub Container Registry successfully.")

	return nil
}

// Download the file from the GitHub repository.
func downloadFile(applicationName string, fileName string) error {
	filePath := fmt.Sprintf(
		"https://raw.githubusercontent.com/nielshoek/%s/main/%s",
		applicationName,
		fileName)
	outputPath := fmt.Sprintf("./%s", fileName)
	authHeader := fmt.Sprintf("Authorization: token %s", gitHubToken)
	cmdDownloadFile := exec.Command(
		"curl",
		"-H", authHeader,
		"-o", outputPath,
		"-w", "%{http_code}",
		filePath)
	curlOutput, err := cmdDownloadFile.CombinedOutput()

	if err != nil {
		log.Printf("Failed to download %s: %v\n", fileName, err)
		return err
	}
	if string(curlOutput[len(curlOutput)-3:]) != "200" {
		log.Printf(
			"Failed to download %s: HTTP status code %s\n",
			fileName,
			string(curlOutput[len(curlOutput)-3:]))
		return fmt.Errorf(
			"Failed to download %s: HTTP status code %s",
			fileName,
			string(curlOutput[len(curlOutput)-3:]))
	}

	log.Printf("Downloaded %s successfully.\n", fileName)

	return nil
}
