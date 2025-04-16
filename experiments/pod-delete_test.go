package experiments

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	experiment "github.com/litmuschaos/litmus-go-sdk/pkg/apis/experiment"
	models "github.com/litmuschaos/litmus/chaoscenter/graphql/server/graph/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"
	"sigs.k8s.io/yaml"
)

func TestPodDelete(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BDD test")
}

//BDD for running pod-delete experiment
var _ = Describe("BDD of running pod-delete experiment", func() {

	Context("Check for pod-delete experiment via SDK", func() {
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
			klog.Infof("[PreReq]: Getting the ENVs for the %v experiment", experimentsDetails.ExperimentName)
			environment.GetENV(&experimentsDetails, "pod-delete", "pod-delete-engine")

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

		It("Should run the pod delete experiment via SDK", func() {

			// Ensure pre-checks passed from BeforeEach
			Expect(err).To(BeNil(), "Error during BeforeEach setup: %v", err)

			// V3 SDK PATH (Now the only path)
			klog.Info("Executing V3 SDK Path for Experiment")

			// 1. Construct Experiment Request
			By("[SDK Prepare]: Constructing Chaos Experiment Request")
			experimentName := experimentsDetails.EngineName
			experimentID := experimentName + "-" + uuid.New().String()[:8]
			experimentRequest, errConstruct := ConstructPodDeleteExperimentRequest(&experimentsDetails, experimentID)
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
			var finalStatus *experiment.ExperimentRunResponse
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

// Helper function to construct the experiment request
// FIXME: This needs a proper implementation to generate the correct manifest YAML
//        based on experimentDetails for pod-delete.
func ConstructPodDeleteExperimentRequest(details *types.ExperimentDetails, experimentID string) (*models.SaveChaosExperimentRequest, error) {
	klog.Infof("Constructing experiment request for %s with ID %s", details.ExperimentName, experimentID)

	// Placeholder manifest definition - Attempting single line definition for linter
	const manifestYAML = `apiVersion: litmuschaos.io/v1alpha1\nkind: ChaosExperiment\nmetadata:\n  name: %s\n  namespace: %s\nspec:\n  steps:\n  - name: delete-pods\n    definition:\n      chaos:\n        fault: pod-delete\n        mode: "" # FIXME: Determine mode\n        selector:\n          namespaces:\n            - "%s"\n          labelSelectors:\n            "%s"\n        # FIXME: Inject other pod-delete fields\n      probes: [] # FIXME: Add probes\n`

	// Basic formatting - NEEDS PROPER POPULATION FROM 'details'
	formattedManifest := fmt.Sprintf(manifestYAML,
		details.ExperimentName, // metadata.name
		details.ChaosNamespace, // metadata.namespace
		details.AppNS,
		details.AppLabel,
	)

	// Validate and clean up the YAML structure
	var manifestInterface interface{}
	errYaml := yaml.Unmarshal([]byte(formattedManifest), &manifestInterface)
	if errYaml != nil {
		klog.Errorf("Error unmarshalling constructed manifest: %v", errYaml)
		return nil, fmt.Errorf("failed to unmarshal constructed manifest: %w", errYaml)
	}
	finalManifestBytes, errYamlMarshal := yaml.Marshal(manifestInterface)
	if errYamlMarshal != nil {
		klog.Errorf("Error marshalling final manifest: %v", errYamlMarshal)
		return nil, fmt.Errorf("failed to marshal final manifest: %w", errYamlMarshal)
	}
	finalManifestString := string(finalManifestBytes)
	klog.Infof("Constructed Manifest: %s", finalManifestString)

	request := &models.SaveChaosExperimentRequest{
		ID:             experimentID, 
		Name:           details.ExperimentName,
		Description:    fmt.Sprintf("CI/CD Triggered Chaos Experiment: %s", details.ExperimentName), 
		Tags:           []string{"chaos-ci-lib", details.ExperimentName},
		InfraID:        details.ConnectedInfraID,
		Manifest:     finalManifestString, 
	}
	return request, nil
}

