package infrastructure

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/litmuschaos/chaos-ci-lib/pkg"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/litmuschaos/litmus-go-sdk/pkg/sdk"
	"github.com/litmuschaos/litmus/chaoscenter/graphql/server/graph/model"
	"k8s.io/klog"
)

// SetupInfrastructure handles the creation or connection to infrastructure
// It checks if infrastructure should be installed and if it's already connected
func SetupInfrastructure(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) error {
	// Check if infrastructure operations should be performed
	installInfra, _ := strconv.ParseBool(os.Getenv("INSTALL_INFRA"))
	if !installInfra {
		klog.Info("INSTALL_INFRA is set to false, skipping infrastructure setup")
		// Handle case where we're using existing infrastructure but not installing
		if experimentsDetails.ConnectedInfraID == "" && experimentsDetails.UseExistingInfra && experimentsDetails.ExistingInfraID != "" {
			experimentsDetails.ConnectedInfraID = experimentsDetails.ExistingInfraID
			klog.Infof("Manually set ConnectedInfraID to %s from ExistingInfraID", experimentsDetails.ConnectedInfraID)
		}
		return nil
	}

	// Check if we should use existing infrastructure
	useExistingInfra, _ := strconv.ParseBool(os.Getenv("USE_EXISTING_INFRA"))
	if useExistingInfra {
		infraID := os.Getenv("EXISTING_INFRA_ID")
		if infraID == "" {
			return errors.New("USE_EXISTING_INFRA is true but EXISTING_INFRA_ID is not provided")
		}
		experimentsDetails.ConnectedInfraID = infraID
		klog.Infof("Using existing infrastructure with ID: %s", infraID)
		return nil
	}

	// If not using existing infrastructure, connect to new one
	err := ConnectInfrastructure(experimentsDetails, sdkClient)
	if err != nil {
		return err
	}

	// Activate the infrastructure by deploying the manifest
	err = ActivateInfrastructure(experimentsDetails, sdkClient)
	if err != nil {
		return fmt.Errorf("failed to activate infrastructure: %v", err)
	}

	return nil
}

// SetupEnvironment checks if we should use an existing environment or create a new one
// It returns the environmentID to be used for infrastructure creation
func SetupEnvironment(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) (string, error) {
	// Check if we should use an existing environment
	useExistingEnv, _ := strconv.ParseBool(os.Getenv("USE_EXISTING_ENV"))
	if useExistingEnv {
		envID := os.Getenv("EXISTING_ENV_ID")
		if envID == "" {
			return "", errors.New("USE_EXISTING_ENV is true but EXISTING_ENV_ID is not provided")
		}
		klog.Infof("Using existing environment with ID: %s", envID)
		return envID, nil
	}

	// Create a new environment
	envName := os.Getenv("ENV_NAME")
	if envName == "" {
		envName = "chaos-ci-env" // Default environment name
	}

	// Configure environment properties
	// Valid values for environment type are "PROD" and "NON_PROD"
	envType := os.Getenv("ENV_TYPE")
	if envType == "" || (envType != "PROD" && envType != "NON_PROD") {
		envType = "NON_PROD" // Default environment type
	}

	envDescription := os.Getenv("ENV_DESCRIPTION")
	if envDescription == "" {
		envDescription = "CI Test Environment"
	}

	environmentID := pkg.GenerateEnvironmentID()

	// Create the environment request with the correct environment type
	createEnvironmentRequest := model.CreateEnvironmentRequest{
		Name:          envName,
		Type:          model.EnvironmentType(envType),
		Description:   &envDescription,
		EnvironmentID: environmentID,
	}

	// Create the environment using SDK
	klog.Infof("Creating new environment: %s with type: %s", envName, envType)
	_, err := sdkClient.Environments().Create(envName, createEnvironmentRequest)
	if err != nil {
		return "", err
	}

	klog.Infof("Successfully created environment with ID: %s", environmentID)
	return environmentID, nil
}

// ConnectInfrastructure connects to a new infrastructure via registerInfra GraphQL mutation
func ConnectInfrastructure(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) error {
	klog.Infof("Attempting to connect infrastructure: %s", experimentsDetails.InfraName)

	// Setup environment (create new or use existing)
	environmentID, err := SetupEnvironment(experimentsDetails, sdkClient)
	if err != nil {
		return err
	}

	// Use the obtained environmentID
	experimentsDetails.InfraEnvironmentID = environmentID

	// Use registerInfra GraphQL mutation to create infrastructure and get manifest
	infraID, err := createInfrastructureViaRegisterInfra(experimentsDetails, sdkClient)
	if err != nil {
		return err
	}

	experimentsDetails.ConnectedInfraID = infraID
	klog.Infof("Successfully connected infrastructure via registerInfra. Stored ID: %s", experimentsDetails.ConnectedInfraID)

	return nil
}

// DisconnectInfrastructure disconnects from infrastructure if it was created during the test
func DisconnectInfrastructure(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) error {
	// Don't disconnect if we're using an existing infrastructure
	useExistingInfra, _ := strconv.ParseBool(os.Getenv("USE_EXISTING_INFRA"))
	if useExistingInfra {
		klog.Info("Using existing infrastructure, skipping disconnection")
		return nil
	}

	// Check if we have an infrastructure to disconnect
	if experimentsDetails.ConnectedInfraID == "" {
		klog.Info("No connected infrastructure ID found, skipping disconnection")
		return nil
	}

	// Disconnect the infrastructure
	klog.Infof("Attempting to disconnect infrastructure with ID: %s", experimentsDetails.ConnectedInfraID)
	err := sdkClient.Infrastructure().Disconnect(experimentsDetails.ConnectedInfraID)
	if err != nil {
		return err
	}

	klog.Infof("Successfully disconnected infrastructure: %s", experimentsDetails.ConnectedInfraID)
	return nil
}

// ActivateInfrastructure downloads and applies the infrastructure manifest to activate the infrastructure
func ActivateInfrastructure(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) error {
	klog.Infof("Activating infrastructure: %s", experimentsDetails.ConnectedInfraID)

	// Check if infrastructure activation should be performed
	activateInfra, _ := strconv.ParseBool(os.Getenv("ACTIVATE_INFRA"))
	if !activateInfra {
		klog.Info("ACTIVATE_INFRA is set to false, skipping infrastructure activation")
		return nil
	}

	// Step 1: Ensure namespace exists (usually already exists)
	err := ensureNamespaceExists(experimentsDetails.InfraNamespace)
	if err != nil {
		return fmt.Errorf("failed to ensure namespace exists: %v", err)
	}

	// Step 2: Apply Litmus CRDs (required for infrastructure components)
	err = applyLitmusCRDs()
	if err != nil {
		return fmt.Errorf("failed to apply Litmus CRDs: %v", err)
	}

	// Step 3: Get the infrastructure manifest (already stored from registerInfra)
	manifestContent := []byte(experimentsDetails.InfraManifest)
	if len(manifestContent) == 0 {
		return fmt.Errorf("no infrastructure manifest available")
	}

	// Step 4: Apply the infrastructure manifest to the cluster
	err = applyInfrastructureManifest(manifestContent, experimentsDetails)
	if err != nil {
		return fmt.Errorf("failed to apply infrastructure manifest: %v", err)
	}

	// Step 5: Wait for infrastructure to become active
	err = waitForInfrastructureActivation(experimentsDetails, sdkClient)
	if err != nil {
		return fmt.Errorf("infrastructure activation timeout: %v", err)
	}

	klog.Infof("Successfully activated infrastructure: %s", experimentsDetails.ConnectedInfraID)
	return nil
}

// createInfrastructureViaRegisterInfra creates infrastructure using registerInfra GraphQL mutation
func createInfrastructureViaRegisterInfra(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) (string, error) {
	// Get authentication token from SDK client
	token := sdkClient.Auth().GetToken()
	if token == "" {
		return "", fmt.Errorf("failed to get authentication token from SDK client")
	}

	// Construct the GraphQL mutation
	mutation := `
		mutation registerInfra($projectID: ID!, $request: RegisterInfraRequest!) {
			registerInfra(projectID: $projectID, request: $request) {
				infraID
				manifest
				__typename
			}
		}
	`

	// Prepare the variables for the mutation with all required fields
	variables := map[string]interface{}{
		"projectID": experimentsDetails.LitmusProjectID,
		"request": map[string]interface{}{
			"infraScope":         experimentsDetails.InfraScope,
			"name":               experimentsDetails.InfraName,
			"environmentID":      experimentsDetails.InfraEnvironmentID,
			"description":        experimentsDetails.InfraDescription,
			"platformName":       "Kubernetes", // Fixed to Kubernetes as per UI
			"infraNamespace":     experimentsDetails.InfraNamespace,
			"serviceAccount":     experimentsDetails.InfraSA,
			"infraNsExists":      experimentsDetails.InfraNsExists,
			"infraSaExists":      experimentsDetails.InfraSaExists,
			"skipSsl":            experimentsDetails.InfraSkipSSL,
			"infrastructureType": "Kubernetes", // Fixed to Kubernetes as per UI
		},
	}

	// Prepare the GraphQL request
	requestBody := map[string]interface{}{
		"operationName": "registerInfra",
		"variables":     variables,
		"query":         mutation,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal GraphQL request: %v", err)
	}

	// Make the HTTP request to the GraphQL endpoint
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	graphqlURL := fmt.Sprintf("%s/api/query", experimentsDetails.LitmusEndpoint)
	klog.Infof("Making registerInfra GraphQL request to: %s", graphqlURL)
	req, err := http.NewRequest("POST", graphqlURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Referer", experimentsDetails.LitmusEndpoint)
	req.Header.Set("Origin", experimentsDetails.LitmusEndpoint)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "chaos-ci-lib/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make GraphQL request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GraphQL request failed with status: %d", resp.StatusCode)
	}

	// Read the response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse the GraphQL response
	var graphqlResponse struct {
		Data struct {
			RegisterInfra struct {
				InfraID  string `json:"infraID"`
				Manifest string `json:"manifest"`
			} `json:"registerInfra"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	err = json.Unmarshal(responseBody, &graphqlResponse)
	if err != nil {
		return "", fmt.Errorf("failed to parse GraphQL response: %v", err)
	}

	// Check for GraphQL errors
	if len(graphqlResponse.Errors) > 0 {
		return "", fmt.Errorf("GraphQL error: %s", graphqlResponse.Errors[0].Message)
	}

	// Extract the infraID and store the manifest for later use
	infraID := graphqlResponse.Data.RegisterInfra.InfraID
	manifest := graphqlResponse.Data.RegisterInfra.Manifest

	if infraID == "" {
		return "", fmt.Errorf("empty infraID received from registerInfra response")
	}

	if manifest == "" {
		return "", fmt.Errorf("empty manifest received from registerInfra response")
	}

	// Store the manifest in experimentsDetails for later use
	experimentsDetails.InfraManifest = manifest

	klog.Infof("Successfully created infrastructure via registerInfra: %s", infraID)
	return infraID, nil
}

// ensureNamespaceExists ensures the specified namespace exists
func ensureNamespaceExists(namespace string) error {
	klog.Infof("Ensuring namespace '%s' exists...", namespace)

	// Check if namespace already exists
	command := []string{"get", "namespace", namespace}
	err := pkg.Kubectl(command...)
	if err == nil {
		klog.Infof("Namespace '%s' already exists", namespace)
		return nil
	}

	// Create namespace if it doesn't exist
	klog.Infof("Creating namespace '%s'...", namespace)
	command = []string{"create", "namespace", namespace}
	err = pkg.Kubectl(command...)
	if err != nil {
		return fmt.Errorf("failed to create namespace %s: %v", namespace, err)
	}

	klog.Infof("Successfully created namespace '%s'", namespace)
	return nil
}

// applyLitmusCRDs applies the Litmus CRDs required for infrastructure components
func applyLitmusCRDs() error {
	klog.Info("Applying Litmus CRDs...")

	// Use the CRD URL from the UI instructions
	crdURL := "https://raw.githubusercontent.com/litmuschaos/litmus/master/mkdocs/docs/3.6.1/litmus-portal-crds-3.6.1.yml"

	// Apply CRDs directly from URL
	command := []string{"apply", "-f", crdURL}
	err := pkg.Kubectl(command...)
	if err != nil {
		return fmt.Errorf("failed to apply Litmus CRDs from %s: %v", crdURL, err)
	}

	klog.Info("Successfully applied Litmus CRDs")

	// Wait a moment for CRDs to be registered
	klog.Info("Waiting for CRDs to be registered...")
	time.Sleep(5 * time.Second)

	return nil
}

// applyInfrastructureManifest applies the infrastructure manifest to the Kubernetes cluster
func applyInfrastructureManifest(manifestContent []byte, experimentsDetails *types.ExperimentDetails) error {
	klog.Info("Applying infrastructure manifest to cluster...")

	// Log the manifest content to check for ID mismatches
	manifestStr := string(manifestContent)
	klog.Infof("Expected infrastructure ID: %s", experimentsDetails.ConnectedInfraID)

	// Check if the manifest contains the correct infrastructure ID
	if strings.Contains(manifestStr, experimentsDetails.ConnectedInfraID) {
		klog.Info("✅ Manifest contains the correct infrastructure ID")
	} else {
		klog.Warning("⚠️  Manifest does NOT contain the expected infrastructure ID")
	}

	// Fix the server address if it's pointing to localhost
	if strings.Contains(manifestStr, "http://localhost:9091") {
		klog.Warning("⚠️  Manifest contains localhost server address")
		// Replace localhost with the internal Kubernetes service name
		internalServerAddr := "http://chaos-litmus-frontend-service.litmus.svc.cluster.local:9091"
		klog.Infof("Replacing localhost server address with internal service: %s", internalServerAddr)
		manifestStr = strings.ReplaceAll(manifestStr, "http://localhost:9091", internalServerAddr)
		klog.Info("✅ Successfully replaced server address in manifest")
	}

	// Convert back to bytes after potential modification
	manifestContent = []byte(manifestStr)

	// Save manifest to temporary file
	manifestFile := fmt.Sprintf("/tmp/%s-infra-manifest.yaml", experimentsDetails.ConnectedInfraID)
	err := os.WriteFile(manifestFile, manifestContent, 0644)
	if err != nil {
		return fmt.Errorf("failed to write manifest file: %v", err)
	}
	defer os.Remove(manifestFile) // Clean up temporary file

	// Apply the manifest using kubectl
	command := []string{"apply", "-f", manifestFile, "--validate=false"}
	err = pkg.Kubectl(command...)
	if err != nil {
		return fmt.Errorf("failed to apply infrastructure manifest: %v", err)
	}

	klog.Info("Successfully applied infrastructure manifest")
	return nil
}

// waitForInfrastructureActivation waits for the infrastructure to become active
func waitForInfrastructureActivation(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) error {
	klog.Info("Waiting for infrastructure to become active...")

	// Get timeout from environment variable or use default
	timeoutMinutes := 5 // Default timeout
	if timeoutStr := os.Getenv("INFRA_ACTIVATION_TIMEOUT"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil {
			timeoutMinutes = timeout
		}
	}

	timeout := time.After(time.Duration(timeoutMinutes) * time.Minute)
	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("infrastructure activation timed out after %d minutes", timeoutMinutes)
		case <-ticker.C:
			// Check infrastructure status using GraphQL
			isActive, err := checkInfrastructureStatusViaGraphQL(experimentsDetails, sdkClient)
			if err != nil {
				klog.Warningf("Error checking infrastructure status: %v", err)
				continue
			}

			if isActive {
				klog.Infof("Infrastructure %s is now active!", experimentsDetails.ConnectedInfraID)
				return nil
			}

			klog.Infof("Infrastructure %s is still not active, waiting...", experimentsDetails.ConnectedInfraID)
		}
	}
}

// checkInfrastructureStatusViaGraphQL checks if the infrastructure is active using a direct GraphQL query
func checkInfrastructureStatusViaGraphQL(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) (bool, error) {
	// Get authentication token from SDK client
	token := sdkClient.Auth().GetToken()
	if token == "" {
		return false, fmt.Errorf("failed to get authentication token from SDK client")
	}

	// Construct the GraphQL query
	query := `
		query listInfras($projectID: ID!) {
			listInfras(projectID: $projectID) {
				infras {
					infraID
					name
					isActive
					isInfraConfirmed
				}
			}
		}
	`

	// Prepare the variables for the query
	variables := map[string]interface{}{
		"projectID": experimentsDetails.LitmusProjectID,
	}

	// Prepare the GraphQL request
	requestBody := map[string]interface{}{
		"operationName": "listInfras",
		"variables":     variables,
		"query":         query,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return false, fmt.Errorf("failed to marshal GraphQL request: %v", err)
	}

	// Make the HTTP request to the GraphQL endpoint
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	graphqlURL := fmt.Sprintf("%s/api/query", experimentsDetails.LitmusEndpoint)
	req, err := http.NewRequest("POST", graphqlURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Referer", experimentsDetails.LitmusEndpoint)
	req.Header.Set("Origin", experimentsDetails.LitmusEndpoint)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "chaos-ci-lib/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to make GraphQL request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("GraphQL request failed with status: %d", resp.StatusCode)
	}

	// Read the response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse the GraphQL response
	var graphqlResponse struct {
		Data struct {
			ListInfras struct {
				Infras []struct {
					InfraID          string `json:"infraID"`
					Name             string `json:"name"`
					IsActive         bool   `json:"isActive"`
					IsInfraConfirmed bool   `json:"isInfraConfirmed"`
				} `json:"infras"`
			} `json:"listInfras"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	err = json.Unmarshal(responseBody, &graphqlResponse)
	if err != nil {
		return false, fmt.Errorf("failed to parse GraphQL response: %v", err)
	}

	// Check for GraphQL errors
	if len(graphqlResponse.Errors) > 0 {
		return false, fmt.Errorf("GraphQL error: %s", graphqlResponse.Errors[0].Message)
	}

	// Find our infrastructure in the list
	for _, infra := range graphqlResponse.Data.ListInfras.Infras {
		if infra.InfraID == experimentsDetails.ConnectedInfraID {
			klog.Infof("GraphQL: Found matching infrastructure %s: isActive=%v, isConfirmed=%v",
				infra.InfraID, infra.IsActive, infra.IsInfraConfirmed)
			return infra.IsActive, nil
		}
	}

	klog.Errorf("Infrastructure %s not found in list of %d infrastructures",
		experimentsDetails.ConnectedInfraID, len(graphqlResponse.Data.ListInfras.Infras))
	return false, fmt.Errorf("infrastructure %s not found in list", experimentsDetails.ConnectedInfraID)
}
