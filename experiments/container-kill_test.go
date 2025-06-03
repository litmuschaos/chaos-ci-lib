package experiments

import (
	"fmt"
	"testing"
	"time"

	"github.com/litmuschaos/chaos-ci-lib/pkg"
	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/infrastructure"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/litmuschaos/chaos-ci-lib/pkg/workflow"
	"github.com/litmuschaos/litmus-go-sdk/pkg/sdk"
	models "github.com/litmuschaos/litmus/chaoscenter/graphql/server/graph/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"
)

func TestContainerKill(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BDD test")
}

// BDD for running container-kill experiment
var _ = Describe("BDD of running container-kill experiment", func() {

	Context("Check for container-kill experiment via SDK", func() {
		// Define variables accessible to It and AfterEach
		var (
			experimentsDetails types.ExperimentDetails
			sdkClient          sdk.Client
			err                error
		)

		BeforeEach(func() {
			experimentsDetails = types.ExperimentDetails{}
			err = nil

			//Fetching all the default ENV
			By("[PreChaos]: Fetching all default ENVs")
			klog.Infof("[PreReq]: Getting the ENVs for the %v experiment", experimentsDetails.ExperimentName)
			environment.GetENV(&experimentsDetails, "container-kill", "container-kill-engine")

			// Initialize SDK client
			By("[PreChaos]: Initializing SDK client")
			sdkClient, err = environment.GenerateClientSetFromSDK()
			Expect(err).To(BeNil(), "Unable to generate Litmus SDK client, due to {%v}", err)

			// Setup infrastructure
			By("[PreChaos]: Setting up infrastructure")
			err = infrastructure.SetupInfrastructure(&experimentsDetails, sdkClient)
			Expect(err).To(BeNil(), "Failed to setup infrastructure, due to {%v}", err)

			// Validate that infrastructure ID is properly set
			Expect(experimentsDetails.ConnectedInfraID).NotTo(BeEmpty(), "Setup failed: ConnectedInfraID is empty after connection attempt.")

			// Setup probe if configured to do so
			if experimentsDetails.CreateProbe {
				By("[PreChaos]: Setting up probe")
				err = workflow.CreateProbe(&experimentsDetails, sdkClient, experimentsDetails.LitmusProjectID)
				Expect(err).To(BeNil(), "Failed to create probe, due to {%v}", err)
				// Validate that probe was created successfully
				Expect(experimentsDetails.CreatedProbeID).NotTo(BeEmpty(), "Probe creation failed: CreatedProbeID is empty")
			}
		})

		It("Should run the container kill experiment via SDK", func() {

			// Ensure pre-checks passed from BeforeEach
			Expect(err).To(BeNil(), "Error during BeforeEach setup: %v", err)
			klog.Info("Executing V3 SDK Path for Experiment")

			// 1. Construct Experiment Request
			By("[SDK Prepare]: Constructing Chaos Experiment Request")
			experimentName := pkg.GenerateUniqueExperimentName("container-kill")
			experimentsDetails.ExperimentName = experimentName
			experimentID := pkg.GenerateExperimentID()
			experimentRequest, errConstruct := workflow.ConstructContainerKillExperimentRequest(&experimentsDetails, experimentID, experimentName)
			Expect(errConstruct).To(BeNil(), "Failed to construct experiment request: %v", errConstruct)

			// 2. Create and Run Experiment via SDK
			By("[SDK Prepare]: Creating and Running Chaos Experiment")
			klog.Infof("About to create experiment with ID: %s, Name: %s, InfraID: %s",
				experimentID, experimentName, experimentsDetails.ConnectedInfraID)
			klog.Infof("Experiment request details: %+v", experimentRequest)

			createResponse, err := sdkClient.Experiments().Create(experimentsDetails.LitmusProjectID, *experimentRequest)
			Expect(err).To(BeNil(), "Failed to create experiment via SDK: %v", err)
			klog.Infof("Created experiment successfully. Response: %s", createResponse)
			klog.Infof("Experiment creation completed for ID: %s", experimentID)

			// 3. Get the experiment run ID
			By("[SDK Query]: Polling for experiment run to become available")
			klog.Infof("Starting to poll for experiment runs for experiment ID: %s", experimentID)
			klog.Infof("Connected Infrastructure ID: %s", experimentsDetails.ConnectedInfraID)
			klog.Infof("Project ID: %s", experimentsDetails.LitmusProjectID)

			var experimentRunID string
			maxRetries := 20
			found := false

			for i := 0; i < maxRetries; i++ {
				klog.Infof("Polling attempt %d/%d for experiment runs...", i+1, maxRetries)
				time.Sleep(5 * time.Second)

				listExperimentRunsReq := models.ListExperimentRunRequest{
					ExperimentIDs: []*string{&experimentID},
				}

				klog.Infof("Making ListRuns request with ExperimentIDs: [%s]", experimentID)
				runsList, err := sdkClient.Experiments().ListRuns(listExperimentRunsReq)
				if err != nil {
					klog.Warningf("Error fetching experiment runs on attempt %d: %v", i+1, err)
					continue
				}

				klog.Infof("Attempt %d: Found %d experiment runs for experiment %s on infra %s", i+1,
					len(runsList.ExperimentRuns), experimentID, experimentsDetails.ConnectedInfraID)

				// Log details of found experiment runs for debugging
				if len(runsList.ExperimentRuns) == 0 {
					klog.Infof("  No experiment runs found in response")
				} else {
					for j, run := range runsList.ExperimentRuns {
						klog.Infof("  Run %d: ID=%s, ExperimentID=%s, Phase=%s",
							j+1, run.ExperimentRunID, run.ExperimentID, run.Phase)
					}
				}

				if len(runsList.ExperimentRuns) > 0 {
					experimentRunID = runsList.ExperimentRuns[0].ExperimentRunID
					klog.Infof("Found experiment run ID: %s", experimentRunID)
					found = true
					break
				}

				klog.Infof("No experiment runs found yet, retrying after delay...")
			}

			Expect(found).To(BeTrue(), "No experiment runs found for experiment after %d retries", maxRetries)

			// 4. Poll for Experiment Run Status
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
					pollError = fmt.Errorf("timed out waiting for experiment run %s to complete after %d minutes", experimentRunID, experimentsDetails.ExperimentTimeout)
					klog.Error(pollError)
					break pollLoop
				case <-ticker.C:
					phase, errStatus := sdkClient.Experiments().GetRunPhase(experimentRunID)
					if errStatus != nil {
						klog.Errorf("Error fetching experiment run status for %s: %v", experimentRunID, errStatus)
						continue
					}
					klog.Infof("Experiment Run %s current phase: %s", experimentRunID, phase)

					finalPhases := []string{"Completed", "Completed_With_Error", "Failed", "Error", "Stopped", "Skipped", "Aborted", "Timeout", "Terminated"}
					if pkg.ContainsString(finalPhases, phase) {
						finalPhase = phase
						klog.Infof("Experiment Run %s reached final phase: %s", experimentRunID, phase)
						break pollLoop
					}
				}
			}

			// 5. Post Validation / Verdict Check
			By("[SDK Verdict]: Checking Experiment Run Verdict")
			Expect(pollError).To(BeNil())
			Expect(finalPhase).NotTo(BeEmpty(), "Final phase should not be empty after polling")
			Expect(finalPhase).To(Equal("Completed"), fmt.Sprintf("Experiment Run phase should be Completed, but got %s", finalPhase))
		})
		// Cleanup using AfterEach
		AfterEach(func() {
			// Disconnect infrastructure using the new module
			By("[CleanUp]: Cleaning up infrastructure")
			errDisconnect := infrastructure.DisconnectInfrastructure(&experimentsDetails, sdkClient)
			Expect(errDisconnect).To(BeNil(), "Failed to clean up infrastructure, due to {%v}", errDisconnect)
		})
	})
})
