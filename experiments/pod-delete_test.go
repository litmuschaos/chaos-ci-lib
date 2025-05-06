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
	experiment "github.com/litmuschaos/litmus-go-sdk/pkg/apis/experiment"
	models "github.com/litmuschaos/litmus/chaoscenter/graphql/server/graph/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"
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
			// By("[PreChaos]: Getting kubeconfig and generate clientset")
			// err = clients.GenerateClientSetFromKubeConfig()
			// Expect(err).To(BeNil(), "Unable to Get the kubeconfig, due to {%v}", err)

			//Fetching all the default ENV
			By("[PreChaos]: Fetching all default ENVs")
			klog.Infof("[PreReq]: Getting the ENVs for the %v experiment", experimentsDetails.ExperimentName)
			environment.GetENV(&experimentsDetails, "pod-delete", "pod-delete-engine")

			// Initialize SDK client
			By("[PreChaos]: Initializing SDK client")
			err = clients.GenerateClientSetFromSDK()
			Expect(err).To(BeNil(), "Unable to generate Litmus SDK client, due to {%v}", err)

			// Setup infrastructure using the new module
			By("[PreChaos]: Setting up infrastructure")
			err = infrastructure.SetupInfrastructure(&experimentsDetails, &clients)
			if experimentsDetails.ConnectedInfraID == "" && experimentsDetails.UseExistingInfra && experimentsDetails.ExistingInfraID != "" {
				experimentsDetails.ConnectedInfraID = experimentsDetails.ExistingInfraID
				klog.Infof("Manually set ConnectedInfraID to %s from ExistingInfraID", experimentsDetails.ConnectedInfraID)
			}
			Expect(err).To(BeNil(), "Failed to setup infrastructure, due to {%v}", err)
			
			// Validate that infrastructure ID is properly set
			Expect(experimentsDetails.ConnectedInfraID).NotTo(BeEmpty(), "Setup failed: ConnectedInfraID is empty after connection attempt.")
		})

		It("Should run the pod delete experiment via SDK", func() {

			// Ensure pre-checks passed from BeforeEach
			Expect(err).To(BeNil(), "Error during BeforeEach setup: %v", err)
			klog.Info("Executing V3 SDK Path for Experiment")


            // 1. Construct Experiment Request
            By("[SDK Prepare]: Constructing Chaos Experiment Request")
            experimentName := pkg.GenerateUniqueExperimentName("pod-delete")
            experimentsDetails.ExperimentName = experimentName
            experimentID := pkg.GenerateExperimentID()
            experimentRequest, errConstruct := workflow.ConstructPodDeleteExperimentRequest(&experimentsDetails, experimentID, experimentName)
            Expect(errConstruct).To(BeNil(), "Failed to construct experiment request: %v", errConstruct)

            // 2. Create and Run Experiment via SDK
			By("[SDK Prepare]: Creating and Running Chaos Experiment")
			creds := clients.GetSDKCredentials()
            _ , err := experiment.CreateExperiment(clients.LitmusProjectID, *experimentRequest, creds)
            Expect(err).To(BeNil(), "Failed to create experiment via SDK: %v", err)
            _, errRun := experiment.RunExperiment(clients.LitmusProjectID, experimentID, creds)
            Expect(errRun).To(BeNil(), "Failed to run experiment via SDK: %v", errRun)
           
            By("[SDK Query]: Fetching latest experiment run ID")
            // Get experiment runs for this experiment
            runsList, err := experiment.GetExperimentRunsList(
                clients.LitmusProjectID, 
                models.ListExperimentRunRequest{
                    ExperimentIDs: []*string{&experimentID},
                    Pagination: &models.Pagination{
                        Page: 1,
                        Limit: 1,
                    },
                }, 
                creds,
            )
            Expect(err).To(BeNil(), "Failed to fetch experiment runs: %v", err)
    
            if len(runsList.ListExperimentRunDetails.ExperimentRuns) > 0 {
                experimentsDetails.ExperimentRunID = runsList.ListExperimentRunDetails.ExperimentRuns[0].ExperimentRunID
                klog.Infof("Latest experiment run ID: %s", experimentsDetails.ExperimentRunID)
            } else {
                Fail("No experiment runs found for experiment: " + experimentID)
            }
            
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
					currentPhase := runStatus.ExperimentRun.Phase
					klog.Infof("Experiment Run %s current phase: %s", experimentsDetails.ExperimentRunID, currentPhase)
					finalPhases := []string{"Completed", "Completed_With_Error", "Failed", "Error", "Stopped", "Skipped", "Aborted", "Timeout", "Terminated"}
					if pkg.ContainsString(finalPhases, string(currentPhase)) {
						finalPhase = string(currentPhase)
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

