package experiments

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/log"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	experiment "github.com/litmuschaos/litmus-go-sdk/pkg/apis/experiment"
	models "github.com/litmuschaos/litmus/chaoscenter/graphql/server/graph/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"
	yamlChe "sigs.k8s.io/yaml"
)

func TestPodNetworkLatency(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BDD test")
}

//BDD for running pod-network-latency experiment
var _ = Describe("BDD of running pod-network-latency experiment", func() {

	Context("Check for pod-network-latency experiment via SDK", func() {
		// Define variables accessible to It and AfterEach
		var (
			experimentsDetails types.ExperimentDetails
			clients            environment.ClientSets
			err                error
		)

		BeforeEach(func() {
			experimentsDetails = types.ExperimentDetails{}
			clients = environment.ClientSets{}
			err = nil

			//Getting kubeConfig and Generate ClientSets
			By("[PreChaos]: Getting kubeconfig and generate clientset")
			err = clients.GenerateClientSetFromKubeConfig()
			Expect(err).To(BeNil(), "Unable to Get the kubeconfig, due to {%v}", err)

			//Fetching all the default ENV
			By("[PreChaos]: Fetching all default ENVs")
			log.Infof("[PreReq]: Getting the ENVs for the %v experiment", experimentsDetails.ExperimentName)
			environment.GetENV(&experimentsDetails, "pod-network-latency", "pod-network-latency-engine")

			// Connect to ChaosCenter Infrastructure via SDK (Mandatory)
			By("[PreChaos]: Connecting Infra via SDK")
			klog.Infof("Attempting to connect infrastructure: %s", experimentsDetails.InfraName)
			err = clients.GenerateClientSetFromSDK()
			Expect(err).To(BeNil(), "Unable to generate Litmus SDK client, due to {%v}", err)

			sdkConfig := map[string]interface{}{
				"namespace":      experimentsDetails.InfraNamespace,
				"serviceAccount": experimentsDetails.InfraSA,
				"mode":           experimentsDetails.InfraScope,
				"description":    experimentsDetails.InfraDescription,
				"platformName":   experimentsDetails.InfraPlatformName,
				"environmentID":  experimentsDetails.InfraEnvironmentID,
				"nsExists":       experimentsDetails.InfraNsExists,
				"saExists":       experimentsDetails.InfraSaExists,
				"skipSSL":        experimentsDetails.InfraSkipSSL,
				"nodeSelector":   experimentsDetails.InfraNodeSelector,
				"tolerations":    experimentsDetails.InfraTolerations,
			}

			infraData, errSdk := clients.SDKClient.Infrastructure().Create(experimentsDetails.InfraName, sdkConfig)
			Expect(errSdk).To(BeNil(), "Failed to create infrastructure via SDK, due to {%v}", errSdk)
			
			Expect(infraData).NotTo(BeNil(), "Infrastructure Create call returned nil data for infra '%s'", experimentsDetails.InfraName)
			registerResponse, ok := infraData.(*models.RegisterInfraResponse)
			Expect(ok).To(BeTrue(), "Could not assert type '%T' to *models.RegisterInfraResponse", infraData)
			Expect(registerResponse).NotTo(BeNil(), "RegisterInfraResponse is nil after type assertion")
			Expect(registerResponse.InfraID).NotTo(BeEmpty(), "Extracted InfraID is empty")
			
			experimentsDetails.ConnectedInfraID = registerResponse.InfraID
			klog.Infof("Successfully connected infrastructure via SDK. Stored ID: %s", experimentsDetails.ConnectedInfraID)

			// Fail setup explicitly if ID is empty after checks
			Expect(experimentsDetails.ConnectedInfraID).NotTo(BeEmpty(), "Setup failed: ConnectedInfraID is empty after connection attempt.")
		})

		It("Should run the pod-network-latency experiment via SDK", func() {

			// Ensure pre-checks passed from BeforeEach
			Expect(err).To(BeNil(), "Error during BeforeEach setup: %v", err)

			// V3 SDK PATH (Now the only path)
			klog.Info("Executing V3 SDK Path for Experiment")

			// 1. Construct Experiment Request
			By("[SDK Prepare]: Constructing Chaos Experiment Request")
			experimentName := experimentsDetails.EngineName
			experimentID := experimentName + "-" + uuid.New().String()[:8]
			experimentRequest, errConstruct := ConstructPodNetworkLatencyExperimentRequest(&experimentsDetails, experimentID)
			Expect(errConstruct).To(BeNil(), "Failed to construct experiment request: %v", errConstruct)

			// 2. Create and Run Experiment via SDK
			By("[SDK Prepare]: Creating and Running Chaos Experiment")
			creds := clients.GetSDKCredentials()
			runResponse, errRun := experiment.CreateExperiment(clients.LitmusProjectID, *experimentRequest, creds)
			Expect(errRun).To(BeNil(), "Failed to create/run experiment via SDK: %v", errRun)
			Expect(runResponse.Data.RunExperimentDetails.NotifyID).NotTo(BeEmpty(), "Experiment Run ID (NotifyID) should not be empty")
			experimentsDetails.ExperimentRunID = runResponse.Data.RunExperimentDetails.NotifyID
			klog.Infof("Experiment Run successfully triggered via SDK. Run ID: %s", experimentsDetails.ExperimentRunID)

			// 3. Poll for Experiment Run Status
			By("[SDK Status]: Polling for Experiment Run Status")
			var finalStatus *models.ExperimentRun
			var pollError error
			timeout := time.After(8 * time.Minute)
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()

			pollLoop:
			for {
				select {
				case <-timeout:
					pollError = fmt.Errorf("timed out waiting for experiment run %s to complete after 8 minutes", experimentsDetails.ExperimentRunID)
					klog.Error(pollError)
					break pollLoop
				case <-ticker.C:
					runStatus, errStatus := experiment.GetExperimentRun(clients.LitmusProjectID, experimentsDetails.ExperimentRunID, creds)
					if errStatus != nil {
						klog.Errorf("Error fetching experiment run status for %s: %v", experimentsDetails.ExperimentRunID, errStatus)
						continue
					}
					currentPhase := runStatus.Data.ExperimentRun.Phase
					klog.Infof("Experiment Run %s current phase: %s", experimentsDetails.ExperimentRunID, currentPhase)
					isFinalPhase := false
					finalPhases := []string{"Completed", "Failed", "Error", "Stopped", "Skipped", "Aborted", "Timeout"}
					for _, phase := range finalPhases {
						if currentPhase == phase {
							isFinalPhase = true
							break
						}
					}
					if isFinalPhase {
						finalStatus = &runStatus.Data.ExperimentRun
						klog.Infof("Experiment Run %s reached final phase: %s", experimentsDetails.ExperimentRunID, currentPhase)
						break pollLoop
					}
				}
			}

			// 4. Post Validation / Verdict Check
			By("[SDK Verdict]: Checking Experiment Run Verdict")
			Expect(pollError).To(BeNil())
			Expect(finalStatus).NotTo(BeNil(), "Final status should not be nil after polling")
			Expect(finalStatus.Phase).To(Equal("Completed"), fmt.Sprintf("Experiment Run phase should be Completed, but got %s", finalStatus.Phase))
			
		})

		// Cleanup using AfterEach
		AfterEach(func() {
			// Disconnect only if Infra ID was successfully stored
			if experimentsDetails.ConnectedInfraID != "" {
				By("[CleanUp]: Disconnecting Infra via SDK")
				klog.Infof("Attempting to disconnect infrastructure with ID: %s", experimentsDetails.ConnectedInfraID)
				if clients.SDKClient == nil {
					klog.Warning("SDK client not initialized in AfterEach, attempting re-initialization for cleanup...")
					errSdkInit := clients.GenerateClientSetFromSDK()
					if errSdkInit != nil {
						klog.Errorf("Failed to re-initialize SDK client for cleanup: %v", errSdkInit)
						return
					}
				}
				errDisconnect := clients.SDKClient.Infrastructure().Disconnect(experimentsDetails.ConnectedInfraID)
				Expect(errDisconnect).To(BeNil(), "Failed to disconnect infra %s via SDK, due to {%v}", experimentsDetails.ConnectedInfraID, errDisconnect)
				if errDisconnect == nil {
					klog.Infof("Successfully disconnected infrastructure: %s", experimentsDetails.ConnectedInfraID)
				}
			} else {
				klog.Info("[CleanUp]: No connected infra ID found, skipping SDK disconnection.")
			}
		})
	})
})

// ConstructPodNetworkLatencyExperimentRequest constructs the experiment request by fetching template from external source
func ConstructPodNetworkLatencyExperimentRequest(details *types.ExperimentDetails, experimentID string) (*models.SaveChaosExperimentRequest, error) {
	klog.Infof("Constructing experiment request for %s with ID %s", details.ExperimentName, experimentID)

	// Fetch Engine template from external source
	var finalManifestString string
	enginePath := "https://hub.litmuschaos.io/api/chaos/master?file=charts/generic/pod-network-latency/engine.yaml"

	// Fetch YAML template
	res, err := http.Get(enginePath)
	if err != nil {
		klog.Errorf("Failed to fetch the engine template, due to %v", err)
		return nil, fmt.Errorf("failed to fetch engine template: %w", err)
	}
	defer res.Body.Close()

	// Read template content
	fileInput, err := ioutil.ReadAll(res.Body)
	if err != nil {
		klog.Errorf("Failed to read data from response: %v", err)
		return nil, fmt.Errorf("failed to read template data: %w", err)
	}

	// Parse the template
	var yamlObj interface{}
	err = yamlChe.Unmarshal(fileInput, &yamlObj)
	if err != nil {
		klog.Errorf("Error unmarshalling fetched template: %v", err)
		return nil, fmt.Errorf("failed to unmarshal template: %w", err)
	}

	// Convert to map to modify
	yamlMap, ok := yamlObj.(map[string]interface{})
	if ok {
		// Update metadata fields
		if metadata, metaOk := yamlMap["metadata"].(map[string]interface{}); metaOk {
			metadata["name"] = details.ExperimentName
			metadata["namespace"] = details.ChaosNamespace
		}

		// Update spec fields if needed
		if spec, specOk := yamlMap["spec"].(map[string]interface{}); specOk {
			if experiments, expOk := spec["experiments"].([]interface{}); expOk && len(experiments) > 0 {
				if exp, expItemOk := experiments[0].(map[string]interface{}); expItemOk {
					// Set experiment name
					exp["name"] = details.ExperimentName
					
					// Update app info if present
					if appinfo, appOk := exp["spec"].(map[string]interface{}); appOk {
						if app, targetOk := appinfo["appinfo"].(map[string]interface{}); targetOk {
							app["appns"] = details.AppNS
							app["applabel"] = details.AppLabel
						}
					}
					
					// Set experiment-specific parameters
					if components, compOk := exp["spec"].(map[string]interface{}); compOk {
						if compEnv, envOk := components["components"].(map[string]interface{}); envOk {
							if env, envListOk := compEnv["env"].([]interface{}); envListOk {
								// Set pod-network-latency specific parameters
								for _, envVar := range env {
									if envMap, isMap := envVar.(map[string]interface{}); isMap {
										if envMap["name"] == "TOTAL_CHAOS_DURATION" {
											envMap["value"] = fmt.Sprintf("%d", details.ChaosDuration)
										} else if envMap["name"] == "CONTAINER_RUNTIME" {
											envMap["value"] = details.ContainerRuntime
										} else if envMap["name"] == "SOCKET_PATH" {
											envMap["value"] = details.SocketPath
										} else if envMap["name"] == "TARGET_CONTAINER" {
											envMap["value"] = details.TargetContainer
										} else if envMap["name"] == "NETWORK_LATENCY" {
											envMap["value"] = fmt.Sprintf("%d", details.NetworkLatency)
										} else if envMap["name"] == "PODS_AFFECTED_PERC" {
											envMap["value"] = fmt.Sprintf("%d", details.PodsAffectedPerc)
										}
									}
								}
							}
						}
					}
				}
			}
		}

		// Marshal back to YAML
		finalManifestBytes, errMarshal := yamlChe.Marshal(yamlMap)
		if errMarshal != nil {
			klog.Errorf("Error marshalling modified template: %v", errMarshal)
			return nil, fmt.Errorf("failed to marshal modified template: %w", errMarshal)
		}
		finalManifestString = string(finalManifestBytes)
	} else {
		return nil, fmt.Errorf("failed to parse template as map")
	}

	klog.Infof("Constructed Manifest from template: %s", finalManifestString)

	request := &models.SaveChaosExperimentRequest{
		ID:             experimentID,
		Name:           details.ExperimentName,
		Description:    fmt.Sprintf("CI/CD Triggered Chaos Experiment: %s", details.ExperimentName),
		Tags:           []string{"chaos-ci-lib", details.ExperimentName},
		InfraID:        details.ConnectedInfraID,
		Manifest:       finalManifestString,
	}
	return request, nil
}
