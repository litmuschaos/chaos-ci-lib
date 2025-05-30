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
	sdkTypes "github.com/litmuschaos/litmus-go-sdk/pkg/types"
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
		Name: envName,
		Type: model.EnvironmentType(envType),
		Description: &envDescription,
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

// ConnectInfrastructure connects to a new infrastructure via the SDK
func ConnectInfrastructure(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) error {
	klog.Infof("Attempting to connect infrastructure: %s", experimentsDetails.InfraName)

	// Setup environment (create new or use existing)
	environmentID, err := SetupEnvironment(experimentsDetails, sdkClient)
	if err != nil {
		return err
	}
	
	// Use the obtained environmentID
	experimentsDetails.InfraEnvironmentID = environmentID

	// Prepare infrastructure configuration
	sdkConfig := sdkTypes.Infra{
		Namespace:      experimentsDetails.InfraNamespace,
		ServiceAccount: experimentsDetails.InfraSA,
		Mode:           experimentsDetails.InfraScope,
		Description:    experimentsDetails.InfraDescription,
		PlatformName:   experimentsDetails.InfraPlatformName,
		EnvironmentID:  experimentsDetails.InfraEnvironmentID,
		NsExists:       experimentsDetails.InfraNsExists,
		SAExists:       experimentsDetails.InfraSaExists,
		SkipSSL:        experimentsDetails.InfraSkipSSL,
		NodeSelector:   experimentsDetails.InfraNodeSelector,
		Tolerations:    experimentsDetails.InfraTolerations,
	}

	// Create infrastructure via SDK
	infraID, errSdk := sdkClient.Infrastructure().Create(experimentsDetails.InfraName, sdkConfig)
	if errSdk != nil {
		return errSdk
	}

	// Process response and extract infra ID
	if infraID == "" {
		return errors.New("infrastructure create call returned nil data")
	}

	experimentsDetails.ConnectedInfraID = infraID
	klog.Infof("Successfully connected infrastructure via SDK. Stored ID: %s", experimentsDetails.ConnectedInfraID)

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

	// Get the infrastructure manifest using the SDK
	manifestContent, err := getInfrastructureManifest(experimentsDetails, sdkClient)
	if err != nil {
		return fmt.Errorf("failed to get infrastructure manifest: %v", err)
	}

	// Apply the infrastructure manifest to the cluster
	err = applyInfrastructureManifest(manifestContent, experimentsDetails)
	if err != nil {
		return fmt.Errorf("failed to apply infrastructure manifest: %v", err)
	}

	// Wait for infrastructure to become active
	err = waitForInfrastructureActivation(experimentsDetails, sdkClient)
	if err != nil {
		return fmt.Errorf("infrastructure activation timeout: %v", err)
	}

	klog.Infof("Successfully activated infrastructure: %s", experimentsDetails.ConnectedInfraID)
	return nil
}

// getInfrastructureManifest gets the infrastructure manifest using the SDK
func getInfrastructureManifest(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) ([]byte, error) {
	klog.Info("Getting infrastructure manifest via GraphQL...")
	
	// Use the GraphQL approach directly as shown by the user
	manifestContent, err := getInfrastructureManifestViaGraphQL(experimentsDetails, sdkClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest via GraphQL: %v", err)
	}

	klog.Info("Successfully retrieved infrastructure manifest")
	return manifestContent, nil
}

// getInfrastructureManifestViaURL gets the infrastructure manifest using the URL-based approach
func getInfrastructureManifestViaURL(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) ([]byte, error) {
	// Get authentication token from SDK client
	token := sdkClient.Auth().GetToken()
	if token == "" {
		return nil, fmt.Errorf("failed to get authentication token from SDK client")
	}
	
	// The manifest download URL should use the server endpoint, not frontend
	serverEndpoint := experimentsDetails.LitmusEndpoint
	if strings.Contains(serverEndpoint, ":9091") {
		serverEndpoint = strings.Replace(serverEndpoint, ":9091", ":9002", 1)
	}
	
	// Construct the manifest download URL based on the Litmus endpoint and infrastructure ID
	manifestURL := fmt.Sprintf("%s/api/file/%s.yaml", serverEndpoint, experimentsDetails.ConnectedInfraID)
	klog.Infof("Infrastructure manifest URL: %s", manifestURL)
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create the HTTP request
	req, err := http.NewRequest("GET", manifestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}
	
	// Set authentication header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Make the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download manifest: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response is successful
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download manifest: HTTP %d", resp.StatusCode)
	}

	// Read the response body
	manifestContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest content: %v", err)
	}

	klog.Info("Successfully downloaded infrastructure manifest via URL")
	return manifestContent, nil
}

// getInfrastructureManifestViaGraphQL gets the infrastructure manifest using GraphQL registerInfra mutation
func getInfrastructureManifestViaGraphQL(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) ([]byte, error) {
	// Get authentication token from SDK client
	token := sdkClient.Auth().GetToken()
	if token == "" {
		return nil, fmt.Errorf("failed to get authentication token from SDK client")
	}
	
	// Construct the GraphQL mutation based on the UI pattern
	mutation := `
		mutation registerInfra($projectID: ID!, $request: RegisterInfraRequest!) {
			registerInfra(projectID: $projectID, request: $request) {
				manifest
				__typename
			}
		}
	`
	
	// Prepare the variables for the mutation
	variables := map[string]interface{}{
		"projectID": experimentsDetails.LitmusProjectID,
		"request": map[string]interface{}{
			"infraScope":         experimentsDetails.InfraScope,
			"name":              experimentsDetails.InfraName,
			"environmentID":     experimentsDetails.InfraEnvironmentID,
			"description":       experimentsDetails.InfraDescription,
			"platformName":      "Kubernetes", // Fixed to Kubernetes as per UI
			"infraNamespace":    experimentsDetails.InfraNamespace,
			"serviceAccount":    experimentsDetails.InfraSA,
			"infraNsExists":     experimentsDetails.InfraNsExists,
			"infraSaExists":     experimentsDetails.InfraSaExists,
			"skipSsl":           experimentsDetails.InfraSkipSSL,
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
		return nil, fmt.Errorf("failed to marshal GraphQL request: %v", err)
	}
	
	// Make the HTTP request to the GraphQL endpoint
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// The GraphQL endpoint is on the frontend port with /api/query path
	graphqlURL := fmt.Sprintf("%s/api/query", experimentsDetails.LitmusEndpoint)
	klog.Infof("Making GraphQL request to: %s", graphqlURL)
	req, err := http.NewRequest("POST", graphqlURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
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
		return nil, fmt.Errorf("failed to make GraphQL request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GraphQL request failed with status: %d", resp.StatusCode)
	}
	
	// Read the response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	
	// Parse the GraphQL response
	var graphqlResponse struct {
		Data struct {
			RegisterInfra struct {
				Manifest string `json:"manifest"`
			} `json:"registerInfra"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	
	err = json.Unmarshal(responseBody, &graphqlResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GraphQL response: %v", err)
	}
	
	// Check for GraphQL errors
	if len(graphqlResponse.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", graphqlResponse.Errors[0].Message)
	}
	
	// Extract the manifest
	manifest := graphqlResponse.Data.RegisterInfra.Manifest
	if manifest == "" {
		return nil, fmt.Errorf("empty manifest received from GraphQL response")
	}
	
	return []byte(manifest), nil
}

// getInfrastructureManifestURL constructs the URL to download the infrastructure manifest
// This method is kept for backward compatibility but is no longer used
func getInfrastructureManifestURL(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) (string, error) {
	// Construct the manifest download URL based on the Litmus endpoint and infrastructure ID
	// This follows the pattern: {LITMUS_ENDPOINT}/api/file/{infraID}.yaml
	manifestURL := fmt.Sprintf("%s/api/file/%s.yaml", experimentsDetails.LitmusEndpoint, experimentsDetails.ConnectedInfraID)
	klog.Infof("Infrastructure manifest URL: %s", manifestURL)
	return manifestURL, nil
}

// downloadInfrastructureManifest downloads the infrastructure manifest from the given URL
// This method is kept for backward compatibility but is no longer used
func downloadInfrastructureManifest(manifestURL string) ([]byte, error) {
	klog.Info("Downloading infrastructure manifest...")
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make the HTTP request
	resp, err := client.Get(manifestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download manifest: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response is successful
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download manifest: HTTP %d", resp.StatusCode)
	}

	// Read the response body
	manifestContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest content: %v", err)
	}

	klog.Info("Successfully downloaded infrastructure manifest")
	return manifestContent, nil
}

// applyInfrastructureManifest applies the infrastructure manifest to the Kubernetes cluster
func applyInfrastructureManifest(manifestContent []byte, experimentsDetails *types.ExperimentDetails) error {
	klog.Info("Applying infrastructure manifest to cluster...")

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
			// Check infrastructure status using SDK
			isActive, err := checkInfrastructureStatus(experimentsDetails, sdkClient)
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

// checkInfrastructureStatus checks if the infrastructure is active using the SDK
func checkInfrastructureStatus(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) (bool, error) {
	klog.Infof("Checking infrastructure status for ID: %s", experimentsDetails.ConnectedInfraID)
	
	// Use direct GraphQL query since SDK List method is not working correctly
	klog.Info("Using GraphQL query to check infrastructure status...")
	isActive, err := checkInfrastructureStatusViaGraphQL(experimentsDetails, sdkClient)
	if err != nil {
		klog.Errorf("GraphQL query failed: %v", err)
		
		// Fallback to SDK method (though it seems to be broken)
		klog.Info("Trying SDK List method as fallback...")
		infraList, sdkErr := sdkClient.Infrastructure().List()
		if sdkErr != nil {
			klog.Errorf("Failed to list infrastructures via SDK: %v", sdkErr)
			return false, fmt.Errorf("both GraphQL and SDK methods failed: GraphQL error: %v, SDK error: %v", err, sdkErr)
		}
		
		klog.Infof("SDK List: Raw response: %+v", infraList)
		klog.Infof("SDK List: Total infrastructures found: %d", len(infraList.Infras))
		
		// Find our infrastructure in the SDK list
		for _, infra := range infraList.Infras {
			if infra.InfraID == experimentsDetails.ConnectedInfraID {
				klog.Infof("SDK List: Found matching infrastructure %s: isActive=%v, isConfirmed=%v", 
					infra.InfraID, infra.IsActive, infra.IsInfraConfirmed)
				return infra.IsActive, nil
			}
		}
		
		return false, fmt.Errorf("infrastructure %s not found in either GraphQL or SDK results", experimentsDetails.ConnectedInfraID)
	}
	
	return isActive, nil
}

// checkInfrastructureStatusViaGraphQL checks if the infrastructure is active using a direct GraphQL query
func checkInfrastructureStatusViaGraphQL(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) (bool, error) {
	// Get authentication token from SDK client
	token := sdkClient.Auth().GetToken()
	if token == "" {
		return false, fmt.Errorf("failed to get authentication token from SDK client")
	}
	
	// Construct the GraphQL mutation based on the UI pattern
	mutation := `
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
	
	// Prepare the variables for the mutation
	variables := map[string]interface{}{
		"projectID": experimentsDetails.LitmusProjectID,
	}
	
	// Prepare the GraphQL request
	requestBody := map[string]interface{}{
		"operationName": "listInfras",
		"variables":     variables,
		"query":         mutation,
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
	
	// The GraphQL endpoint is on the frontend port with /api/query path
	graphqlURL := fmt.Sprintf("%s/api/query", experimentsDetails.LitmusEndpoint)
	klog.Infof("Making GraphQL request to: %s", graphqlURL)
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
	
	klog.Infof("GraphQL: Raw response body: %s", string(responseBody))
	
	// Parse the GraphQL response
	var graphqlResponse struct {
		Data struct {
			ListInfras struct {
				Infras []struct {
					InfraID         string `json:"infraID"`
					Name            string `json:"name"`
					IsActive        bool   `json:"isActive"`
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