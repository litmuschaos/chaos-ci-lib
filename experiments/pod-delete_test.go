package experiments

import (
	"testing"

	"github.com/litmuschaos/chaos-ci-lib/pkg"
	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	engine "github.com/litmuschaos/chaos-ci-lib/pkg/generic/pod-delete/lib"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
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

	Context("Check for pod-delete experiment", func() { 
		// Define variables accessible to It and AfterEach
		var (
			experimentsDetails types.ExperimentDetails
			clients            environment.ClientSets
			chaosEngine        v1alpha1.ChaosEngine
			err                error 
		)

		BeforeEach(func() { 
			experimentsDetails = types.ExperimentDetails{}
			clients = environment.ClientSets{}
			chaosEngine = v1alpha1.ChaosEngine{}
			err = nil

			//Getting kubeConfig and Generate ClientSets
			By("[PreChaos]: Getting kubeconfig and generate clientset") 
			err = clients.GenerateClientSetFromKubeConfig()
			Expect(err).To(BeNil(), "Unable to Get the kubeconfig, due to {%v}", err) // Use BeNil directly

			//Fetching all the default ENV
			By("[PreChaos]: Fetching all default ENVs")
			klog.Infof("[PreReq]: Getting the ENVs for the %v experiment", experimentsDetails.ExperimentName)
			environment.GetENV(&experimentsDetails, "pod-delete", "pod-delete-engine")

			// Connect to ChaosCenter Infrastructure if flag is set
			By("[PreChaos]: Conditionally connecting Infra via SDK")
			if experimentsDetails.ConnectInfraFlag {
				klog.Infof("CONNECT_INFRA flag is set, attempting to connect infrastructure: %s", experimentsDetails.InfraName)
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
				if errSdk == nil { // Only proceed if connection was successful
					klog.Infof("Successfully initiated infrastructure connection via SDK for: %v", experimentsDetails.InfraName)
					// Attempt to extract the InfraID
					if infraData != nil {
						registerResponse, ok := infraData.(*models.RegisterInfraResponse)
						if ok && registerResponse != nil {
							experimentsDetails.ConnectedInfraID = registerResponse.InfraID
							klog.Infof("Stored connected infra ID: %s", experimentsDetails.ConnectedInfraID)
						} else {
							klog.Warningf("Could not assert type '%T' to *models.RegisterInfraResponse or extract InfraID from SDK response for infra '%s'", infraData, experimentsDetails.InfraName)
						}
					} else {
						klog.Warningf("Infrastructure Create call returned nil data for infra '%s'", experimentsDetails.InfraName)
					}
				}
			} else {
				klog.Info("CONNECT_INFRA flag not set, skipping SDK infrastructure connection.")
			}
		})

		It("Should check for the pod delete experiment", func() { // Use It directly

			// Ensure pre-checks passed from BeforeEach
			Expect(err).To(BeNil(), "Error during BeforeEach setup: %v", err)

			// Install RBAC for experiment Execution
			By("[Prepare]: Prepare and install RBAC")
			err = pkg.InstallRbac(&experimentsDetails, experimentsDetails.ChaosNamespace)
			Expect(err).To(BeNil(), "fail to install rbac for the experiment, due to {%v}", err)

			// Install ChaosEngine for experiment Execution
			By("[Prepare]: Prepare and install ChaosEngine")
			err = engine.InstallPodDeleteEngine(&experimentsDetails, &chaosEngine, clients)
			Expect(err).To(BeNil(), "fail to install chaosengine, due to {%v}", err)

			//Checking runner pod running state
			By("[Status]: Runner pod running status check")
			err = pkg.RunnerPodStatus(&experimentsDetails, chaosEngine.Namespace, clients)
			if err != nil && chaosEngine.Namespace != experimentsDetails.AppNS {
				err = pkg.RunnerPodStatus(&experimentsDetails, experimentsDetails.AppNS, clients)
			}
			Expect(err).To(BeNil(), "Runner pod status check failed, due to {%v}", err)

			//Chaos pod running status check
			By("[Status]: Chaos pod running status check")
			err = pkg.ChaosPodStatus(&experimentsDetails, clients)
			Expect(err).To(BeNil(), "Chaos pod status check failed, due to {%v}", err)

			//Waiting for chaos pod to get completed
			//And Print the logs of the chaos pod
			By("[Status]: Wait for chaos pod completion and then print logs")
			err = pkg.ChaosPodLogs(&experimentsDetails, clients)
			Expect(err).To(BeNil(), "Fail to get the experiment chaos pod logs, due to {%v}", err)

			//Checking the chaosresult verdict
			By("[Verdict]: Checking the chaosresult verdict")
			err = pkg.ChaosResultVerdict(&experimentsDetails, clients)
			Expect(err).To(BeNil(), "ChasoResult Verdict check failed, due to {%v}", err)

			//Checking chaosengine verdict
			By("Checking the Verdict of Chaos Engine")
			err = pkg.ChaosEngineVerdict(&experimentsDetails, clients)
			Expect(err).To(BeNil(), "ChaosEngine Verdict check failed, due to {%v}", err)
		})

		// Cleanup using AfterEach
		AfterEach(func() { // Use AfterEach directly
			if experimentsDetails.ConnectInfraFlag && experimentsDetails.ConnectedInfraID != "" {
				// Only attempt disconnect if infra was potentially connected and ID was stored
				By("[CleanUp]: Disconnecting Infra via SDK")
				klog.Infof("Attempting to disconnect infrastructure with ID: %s", experimentsDetails.ConnectedInfraID)
				// Need SDK client - ensure 'clients' is accessible or re-initialize if needed
				if clients.SDKClient == nil {
					// Re-initialize SDK client if not available (e.g., if BeforeEach failed partially)
					klog.Warning("SDK client not initialized in AfterEach, attempting re-initialization for cleanup...")
					errSdkInit := clients.GenerateClientSetFromSDK()
					if errSdkInit != nil {
						klog.Errorf("Failed to re-initialize SDK client for cleanup: %v", errSdkInit)
						// Skip disconnect if client cannot be initialized
						return 
					}
				}
				errDisconnect := clients.SDKClient.Infrastructure().Disconnect(experimentsDetails.ConnectedInfraID)
				// Use Expect directly due to dot import for Gomega
				Expect(errDisconnect).To(BeNil(), "Failed to disconnect infra %s via SDK, due to {%v}", experimentsDetails.ConnectedInfraID, errDisconnect)
				if errDisconnect == nil {
					klog.Infof("Successfully disconnected infrastructure: %s", experimentsDetails.ConnectedInfraID)
				}
			} else if experimentsDetails.ConnectInfraFlag {
				klog.Info("[CleanUp]: ConnectInfraFlag was set but no connected infra ID found, skipping SDK disconnection.")
			}
		})
	})
})
