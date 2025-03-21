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
	ApplicationName string            `json:"application_name"`
	EnvironmentVars map[string]string `json:"environment_vars"`
}

var deployToken string
var gitHubToken string

func main() {
	deployToken = os.Getenv("DEPLOY_TOKEN")
	gitHubToken = os.Getenv("GITHUB_TOKEN")
	if deployToken == "" {
		log.Fatal("DEPLOY_TOKEN environment variable not set.")
	}
	http.HandleFunc("POST /deploy", deployHandler)
	fmt.Println("Server listening on port 9090.")
	log.Fatal(http.ListenAndServe(":9090", nil))
}

func deployHandler(w http.ResponseWriter, r *http.Request) {
	var deployRequest DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&deployRequest); err != nil {
		http.Error(w, "Invalid JSON payload.", http.StatusBadRequest)
		return
	}
	fmt.Println(deployRequest)

	suppliedToken := r.Header.Get("Authorization")
	if suppliedToken != deployToken {
		http.Error(w, "Invalid token.", http.StatusUnauthorized)
		return
	}

	switch {
	case deployRequest.ApplicationName != "":
		err := handleDeployment(deployRequest.ApplicationName, deployRequest.EnvironmentVars)
		if err != nil {
			http.Error(w, "Failed to update Docker container.", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(fmt.Appendf(nil, "Application '%s' updated successfully.\n", deployRequest.ApplicationName))
	default:
		http.Error(w, "Unknown event.", http.StatusBadRequest)
	}
}

func handleDeployment(applicationName string, environmentVars map[string]string) error {
	if err := downloadDockerComposeFile(applicationName); err != nil {
		return err
	}

	if err := loginToGitHubContainerRegistry(); err != nil {
		return err
	}

	if err := deployDockerCompose(environmentVars); err != nil {
		return err
	}

	log.Printf("Application '%s' updated successfully.\n", applicationName)
	return nil
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

// Download the docker-compose.yml file from the GitHub repository.
func downloadDockerComposeFile(applicationName string) error {
	dockerComposePath := fmt.Sprintf(
		"https://raw.githubusercontent.com/nielshoek/%s/main/docker-compose.yml",
		applicationName)
	outputPath := "./docker-compose.yml"
	authHeader := fmt.Sprintf("Authorization: token %s", gitHubToken)
	cmdDownloadDockerCompose := exec.Command(
		"curl",
		"-H", authHeader,
		"-o", outputPath,
		"-w", "%{http_code}",
		dockerComposePath)
	curlOutput, err := cmdDownloadDockerCompose.CombinedOutput()

	if err != nil {
		log.Println("Failed to download docker-compose.yml:", err)
		return err
	}
	if string(curlOutput[len(curlOutput)-3:]) != "200" {
		log.Printf(
			"Failed to download docker-compose.yml: HTTP status code %s\n",
			string(curlOutput[len(curlOutput)-3:]))
		return fmt.Errorf(
			"Failed to download docker-compose.yml: HTTP status code %s",
			string(curlOutput[len(curlOutput)-3:]))
	}

	log.Println("Downloaded docker-compose.yml successfully.")

	return nil
}
