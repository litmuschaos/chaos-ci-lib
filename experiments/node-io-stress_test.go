package experiments

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/infrastructure"
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

func TestNodeIOStress(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BDD test")
}

//BDD for running node-io-stress experiment
var _ = Describe("BDD of running node-io-stress experiment", func() {

	Context("Check for node-io-stress experiment via SDK", func() {
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
			environment.GetENV(&experimentsDetails, "node-io-stress", "node-io-stress-engine")

			// Initialize SDK client
			By("[PreChaos]: Initializing SDK client")
			err = clients.GenerateClientSetFromSDK()
			Expect(err).To(BeNil(), "Unable to generate Litmus SDK client, due to {%v}", err)

			// Setup infrastructure using the new module
			By("[PreChaos]: Setting up infrastructure")
			err = infrastructure.SetupInfrastructure(&experimentsDetails, &clients)
			Expect(err).To(BeNil(), "Failed to setup infrastructure, due to {%v}", err)

			// Validate that infrastructure ID is properly set
			Expect(experimentsDetails.ConnectedInfraID).NotTo(BeEmpty(), "Setup failed: ConnectedInfraID is empty after connection attempt.")
		})

		It("Should run the node-io-stress experiment via SDK", func() {

			// Ensure pre-checks passed from BeforeEach
			Expect(err).To(BeNil(), "Error during BeforeEach setup: %v", err)

			// V3 SDK PATH (Now the only path)
			klog.Info("Executing V3 SDK Path for Experiment")

			// 1. Construct Experiment Request
			By("[SDK Prepare]: Constructing Chaos Experiment Request")
			experimentName := experimentsDetails.EngineName
			experimentID := experimentName + "-" + uuid.New().String()[:8]
			experimentRequest, errConstruct := ConstructNodeIOStressExperimentRequest(&experimentsDetails, experimentID)
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
			var finalPhase string
			var pollError error
			timeout := time.After(time.Duration(experimentsDetails.ExperimentTimeout) * time.Minute)
			ticker := time.NewTicker(time.Duration(experimentsDetails.ExperimentPollingInterval) * time.Second)
			defer ticker.Stop()

			pollLoop:
			for {
				select {
				case <-timeout:
					pollError = fmt.Errorf("timed out waiting for experiment run %s to complete after %d minutes", experimentsDetails.ExperimentRunID, experimentsDetails.ExperimentTimeout)
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
					finalPhases := []string{"Completed", "Completed_With_Error", "Failed", "Error", "Stopped", "Skipped", "Aborted", "Timeout" , "Terminated"}
					for _, phase := range finalPhases {
						if currentPhase == phase {
							isFinalPhase = true
							break
						}
					}
					if isFinalPhase {
						finalPhase = currentPhase
						klog.Infof("Experiment Run %s reached final phase: %s", experimentsDetails.ExperimentRunID, currentPhase)
						break pollLoop
					}
				}
			}

			// 4. Post Validation / Verdict Check
			By("[SDK Verdict]: Checking Experiment Run Verdict")
			Expect(pollError).To(BeNil())
			Expect(finalPhase).NotTo(BeEmpty(), "Final phase should not be empty after polling")
			Expect(finalPhase).To(Equal("Completed"), fmt.Sprintf("Experiment Run phase should be Completed, but got %s", finalPhase))
			
		})

		// Cleanup using AfterEach
		AfterEach(func() {
			// Disconnect infrastructure using the new module
			By("[CleanUp]: Cleaning up infrastructure")
			errDisconnect := infrastructure.DisconnectInfrastructure(&experimentsDetails, &clients)
			Expect(errDisconnect).To(BeNil(), "Failed to clean up infrastructure, due to {%v}", errDisconnect)
		})
	})
})

// ConstructNodeIOStressExperimentRequest constructs the experiment request by fetching template from external source
func ConstructNodeIOStressExperimentRequest(details *types.ExperimentDetails, experimentID string) (*models.SaveChaosExperimentRequest, error) {
	klog.Infof("Constructing experiment request for %s with ID %s", details.ExperimentName, experimentID)

	// Fetch Engine template from external source
	var finalManifestString string
	enginePath := "https://hub.litmuschaos.io/api/chaos/master?file=charts/generic/node-io-stress/engine.yaml"

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
								// Set node-io-stress specific parameters
								for _, envVar := range env {
									if envMap, isMap := envVar.(map[string]interface{}); isMap {
										if envMap["name"] == "TOTAL_CHAOS_DURATION" {
											envMap["value"] = fmt.Sprintf("%d", details.ChaosDuration)
										} else if envMap["name"] == "FILESYSTEM_UTILIZATION_PERCENTAGE" {
											envMap["value"] = fmt.Sprintf("%d", details.FileSystemUtilizationPercentage)
										} else if envMap["name"] == "FILESYSTEM_UTILIZATION_BYTES" {
											envMap["value"] = fmt.Sprintf("%d", details.FilesystemUtilizationBytes)
										} else if envMap["name"] == "NODES_AFFECTED_PERC" {
											envMap["value"] = fmt.Sprintf("%d", details.NodesAffectedPerc)
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
