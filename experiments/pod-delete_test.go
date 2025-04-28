package experiments

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/infrastructure"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
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
			By("[PreChaos]: Getting kubeconfig and generate clientset")
			err = clients.GenerateClientSetFromKubeConfig()
			Expect(err).To(BeNil(), "Unable to Get the kubeconfig, due to {%v}", err)

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
			Expect(err).To(BeNil(), "Failed to setup infrastructure, due to {%v}", err)
			
			// Validate that infrastructure ID is properly set
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

// Create Argo Workflow manifest for Litmus 3.0
//This replaces the approach of calling the litmus-workflows-api to get the workflow manifest
// Not sure if this is the best way to do it, need confirmation from mentors
func ConstructPodDeleteExperimentRequest(details *types.ExperimentDetails, experimentID string) (*models.SaveChaosExperimentRequest, error) {
    klog.Infof("Constructing experiment request for %s with ID %s", details.ExperimentName, experimentID)

    // Create Argo Workflow manifest for Litmus 3.0
    workflowManifest := fmt.Sprintf(`{
        "apiVersion": "argoproj.io/v1alpha1",
        "kind": "Workflow",
        "metadata": {
            "name": "%s",
            "namespace": "%s",
            "labels": {
                "infra_id": "%s",
                "subject": "{{workflow.parameters.appNamespace}}_pod-delete",
                "workflow_id": "%s",
                "workflows.argoproj.io/controller-instanceid": "%s"
            }
        },
        "spec": {
            "entrypoint": "argowf-chaos",
            "serviceAccountName": "argo-chaos",
            "securityContext": {
                "runAsUser": 1000,
                "runAsNonRoot": true
            },
            "arguments": {
                "parameters": [
                    {
                        "name": "adminModeNamespace",
                        "value": "%s"
                    },
                    {
                        "name": "appNamespace",
                        "value": "%s"
                    }
                ]
            },
            "templates": [
                {
                    "name": "argowf-chaos",
                    "steps": [
                        [
                            {
                                "name": "install-chaos-faults",
                                "template": "install-chaos-faults",
                                "arguments": {}
                            }
                        ],
                        [
                            {
                                "name": "run-chaos",
                                "template": "run-chaos",
                                "arguments": {}
                            }
                        ],
                        [
                            {
                                "name": "cleanup-chaos-resources",
                                "template": "cleanup-chaos-resources",
                                "arguments": {}
                            }
                        ]
                    ]
                },
                {
                    "name": "install-chaos-faults",
                    "inputs": {
                        "artifacts": [
                            {
                                "name": "install-chaos-faults",
                                "path": "/tmp/pod-delete.yaml",
                                "raw": {
                                    "data": "apiVersion: litmuschaos.io/v1alpha1\\ndescription:\\n  message: |\\n    Deletes a pod belonging to a deployment/statefulset/daemonset\\nkind: ChaosExperiment\\nmetadata:\\n  name: pod-delete\\nspec:\\n  definition:\\n    scope: Namespaced\\n    permissions:\\n      - apiGroups:\\n          - \\\"\\\"\\n          - \\\"apps\\\"\\n          - \\\"batch\\\"\\n          - \\\"litmuschaos.io\\\"\\n        resources:\\n          - \\\"deployments\\\"\\n          - \\\"jobs\\\"\\n          - \\\"pods\\\"\\n          - \\\"pods/log\\\"\\n          - \\\"events\\\"\\n          - \\\"configmaps\\\"\\n          - \\\"chaosengines\\\"\\n          - \\\"chaosexperiments\\\"\\n          - \\\"chaosresults\\\"\\n        verbs:\\n          - \\\"create\\\"\\n          - \\\"list\\\"\\n          - \\\"get\\\"\\n          - \\\"patch\\\"\\n          - \\\"update\\\"\\n          - \\\"delete\\\"\\n      - apiGroups:\\n          - \\\"\\\"\\n        resources:\\n          - \\\"nodes\\\"\\n        verbs:\\n          - \\\"get\\\"\\n          - \\\"list\\\"\\n    image: \\\"litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0\\\"\\n    imagePullPolicy: Always\\n    args:\\n    - -c\\n    - ./experiments -name pod-delete\\n    command:\\n    - /bin/bash\\n    env:\\n\\n    - name: TOTAL_CHAOS_DURATION\\n      value: '%d'\\n\\n    # Period to wait before and after injection of chaos in sec\\n    - name: RAMP_TIME\\n      value: ''\\n\\n    # provide the kill count\\n    - name: KILL_COUNT\\n      value: '%d'\\n\\n    - name: FORCE\\n      value: 'true'\\n\\n    - name: CHAOS_INTERVAL\\n      value: '%d'\\n\\n    labels:\\n      name: pod-delete\\n"
                                }
                            }
                        ]
                    },
                    "container": {
                        "image": "litmuschaos/k8s:latest",
                        "command": [
                            "sh",
                            "-c"
                        ],
                        "args": [
                            "kubectl apply -f /tmp/pod-delete.yaml -n {{workflow.parameters.adminModeNamespace}}"
                        ],
                        "resources": {}
                    }
                },
                {
                    "name": "run-chaos",
                    "inputs": {
                        "artifacts": [
                            {
                                "name": "run-chaos",
                                "path": "/tmp/chaosengine-run-chaos.yaml",
                                "raw": {
                                    "data": "apiVersion: litmuschaos.io/v1alpha1\\nkind: ChaosEngine\\nmetadata:\\n  namespace: \\\"{{workflow.parameters.adminModeNamespace}}\\\"\\n  labels:\\n    context: \\\"{{workflow.parameters.appNamespace}}_pod-delete\\\"\\n    workflow_run_id: \\\"{{ workflow.uid }}\\\"\\n    workflow_name: %s\\n  annotations:\\n    probeRef: '[{\\\"name\\\":\\\"ping-google\\\",\\\"mode\\\":\\\"SOT\\\"}]'\\n  generateName: run-chaos\\nspec:\\n  appinfo:\\n    appns: %s\\n    applabel: %s\\n    appkind: deployment\\n  jobCleanUpPolicy: retain\\n  engineState: active\\n  chaosServiceAccount: litmus-admin\\n  experiments:\\n    - name: pod-delete\\n      spec:\\n        components:\\n          env:\\n            - name: TOTAL_CHAOS_DURATION\\n              value: \\\"%d\\\"\\n            - name: CHAOS_INTERVAL\\n              value: \\\"%d\\\"\\n            - name: FORCE\\n              value: \\\"false\\\"\\n"
                                }
                            }
                        ]
                    },
                    "metadata": {
                        "labels": {
                            "weight": "10"
                        }
                    },
                    "container": {
                        "name": "",
                        "image": "docker.io/litmuschaos/litmus-checker:2.11.0",
                        "args": [
                            "-file=/tmp/chaosengine-run-chaos.yaml",
                            "-saveName=/tmp/engine-name"
                        ],
                        "resources": {}
                    }
                },
                {
                    "name": "cleanup-chaos-resources",
                    "container": {
                        "image": "litmuschaos/k8s:latest",
                        "command": [
                            "sh",
                            "-c"
                        ],
                        "args": [
                            "kubectl delete chaosengine -l workflow_run_id={{workflow.uid}} -n {{workflow.parameters.adminModeNamespace}}"
                        ],
                        "resources": {}
                    }
                }
            ]
        },
        "status": {}
    }`, 
    details.ExperimentName, // metadata.name
    details.ChaosNamespace, // metadata.namespace
    details.ConnectedInfraID, // metadata.labels.infra_id
    experimentID, // metadata.labels.workflow_id
    details.ConnectedInfraID, // metadata.labels.workflows.argoproj.io/controller-instanceid
    details.ChaosNamespace, // adminModeNamespace
    details.AppNS, // appNamespace
    details.ChaosDuration, // TOTAL_CHAOS_DURATION in install-chaos-faults
    details.ChaosInterval, // CHAOS_INTERVAL in install-chaos-faults
    details.ExperimentName, // workflow_name in run-chaos
    details.AppNS, // appns in run-chaos
    details.AppLabel, // applabel in run-chaos
    details.ChaosDuration, // TOTAL_CHAOS_DURATION in run-chaos
    details.ChaosInterval) // CHAOS_INTERVAL in run-chaos

    // Construct the experiment request
    experimentRequest := &models.SaveChaosExperimentRequest{
        ID:          experimentID,
        Name:        details.ExperimentName,
        InfraID:     details.ConnectedInfraID,
        Description: "Test execution via SDK client",
        Tags:        []string{"ci", "test", "sdk"},
        Manifest:    workflowManifest,
    }

    return experimentRequest, nil
}

