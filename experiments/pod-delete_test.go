package experiments

import (
	"bytes"
	"fmt"
	"testing"
	"text/template"
	"time"

	"github.com/litmuschaos/chaos-ci-lib/pkg"
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
            experimentRequest, errConstruct := ConstructPodDeleteExperimentRequest(&experimentsDetails, experimentID, experimentName)
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

// Create Argo Workflow manifest for Litmus 3.0
func ConstructPodDeleteExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
    klog.Infof("Constructing experiment request for %s with ID %s", details.ExperimentName, experimentID)
    
    // Define the Argo Workflow manifest template
    const workflowTemplate = `{
        "apiVersion": "argoproj.io/v1alpha1",
        "kind": "Workflow",
        "metadata": {
            "name": "{{.ExperimentName}}",
            "namespace": "litmus-2"
        },
        "spec": {
            "entrypoint": "pod-delete-engine",
            "serviceAccountName": "argo-chaos",
            "podGC":{
                "strategy": "OnWorkflowCompletion"
            },
            "securityContext": {
                "runAsUser": 1000,
                "runAsNonRoot": true
            },
            "arguments": {
                "parameters": [
                    {
                        "name": "adminModeNamespace",
                        "value": "litmus-2"
                    }
                ]
            },
            "templates": [
                {
                    "name": "pod-delete-engine",
                    "steps": [
                        [
                            {
                                "name": "install-chaos-faults",
                                "template": "install-chaos-faults"
                            }
                        ],
                        [
                            {
                                "name": "pod-delete-ce5",
                                "template": "pod-delete-ce5"
                            }
                        ],
                        [
                            {
                                "name": "cleanup-chaos-resources",
                                "template": "cleanup-chaos-resources"
                            }
                        ]
                    ]
                },
                {
                    "name": "install-chaos-faults",
                    "inputs": {
                        "artifacts": [
                            {
                                "name": "pod-delete-ce5",
                                "path": "/tmp/pod-delete-ce5.yaml",
                                "raw": {
                                    "data": "apiVersion: litmuschaos.io/v1alpha1\ndescription:\n  message: |\n    Deletes a pod belonging to a deployment/statefulset/daemonset\nkind: ChaosExperiment\nmetadata:\n  name: pod-delete\nspec:\n  definition:\n    scope: Namespaced\n    permissions:\n      - apiGroups:\n          - \"\"\n        resources:\n          - pods\n        verbs:\n          - create\n          - delete\n          - get\n          - list\n          - patch\n          - update\n          - deletecollection\n      - apiGroups:\n          - \"\"\n        resources:\n          - events\n        verbs:\n          - create\n          - get\n          - list\n          - patch\n          - update\n      - apiGroups:\n          - \"\"\n        resources:\n          - configmaps\n        verbs:\n          - get\n          - list\n      - apiGroups:\n          - \"\"\n        resources:\n          - pods/log\n        verbs:\n          - get\n          - list\n          - watch\n      - apiGroups:\n          - \"\"\n        resources:\n          - pods/exec\n        verbs:\n          - get\n          - list\n          - create\n      - apiGroups:\n          - apps\n        resources:\n          - deployments\n          - statefulsets\n          - replicasets\n          - daemonsets\n        verbs:\n          - list\n          - get\n      - apiGroups:\n          - apps.openshift.io\n        resources:\n          - deploymentconfigs\n        verbs:\n          - list\n          - get\n      - apiGroups:\n          - \"\"\n        resources:\n          - replicationcontrollers\n        verbs:\n          - get\n          - list\n      - apiGroups:\n          - argoproj.io\n        resources:\n          - rollouts\n        verbs:\n          - list\n          - get\n      - apiGroups:\n          - batch\n        resources:\n          - jobs\n        verbs:\n          - create\n          - list\n          - get\n          - delete\n          - deletecollection\n      - apiGroups:\n          - litmuschaos.io\n        resources:\n          - chaosengines\n          - chaosexperiments\n          - chaosresults\n        verbs:\n          - create\n          - list\n          - get\n          - patch\n          - update\n          - delete\n    image: \"litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0\"\n    imagePullPolicy: Always\n    args:\n    - -c\n    - ./experiments -name pod-delete\n    command:\n    - /bin/bash\n    env:\n    - name: TOTAL_CHAOS_DURATION\n      value: '15'\n    - name: RAMP_TIME\n      value: ''\n    - name: KILL_COUNT\n      value: ''\n    - name: FORCE\n      value: 'true'\n    - name: CHAOS_INTERVAL\n      value: '5'\n    labels:\n      name: pod-delete\n"
                                }
                            }
                        ]
                    },
                    "container": {
                        "name": "",
                        "image": "litmuschaos/k8s:2.11.0",
                        "command": [
                            "sh",
                            "-c"
                        ],
                        "args": [
                            "kubectl apply -f /tmp/ -n {{ .WorkflowAdminNamespace }} && sleep 30"
                        ],
                        "resources": {}
                    }
                },
                {
                    "name": "pod-delete-ce5",
                    "inputs": {
                        "artifacts": [
                            {
                                "name": "pod-delete-ce5",
                                "path": "/tmp/pod-delete-ce5.yaml",
                                "raw": {
                                    "data": "apiVersion: litmuschaos.io/v1alpha1\nkind: ChaosEngine\nmetadata:\n  namespace: \"{{ .WorkflowAdminNamespace }}\"\n  labels:\n    workflow_run_id: \"{{ .WorkflowUID }}\"\n    workflow_name: {{.ExperimentName}}\n  annotations:\n    probeRef: '[{\"name\":\"myprobe\",\"mode\":\"SOT\"}]'\n  generateName: pod-delete-ce5\nspec:\n  appinfo:\n    appns: {{.AppNamespace}}\n    applabel: {{.AppLabel}}\n    appkind: {{.AppKind}}\n  engineState: active\n  chaosServiceAccount: litmus-admin\n  experiments:\n    - name: pod-delete\n      spec:\n        components:\n          env:\n            - name: TOTAL_CHAOS_DURATION\n              value: \"{{.ChaosDuration}}\"\n            - name: RAMP_TIME\n              value: \"\"\n            - name: FORCE\n              value: \"true\"\n            - name: CHAOS_INTERVAL\n              value: \"{{.ChaosInterval}}\"\n            - name: PODS_AFFECTED_PERC\n              value: \"\"\n            - name: TARGET_CONTAINER\n              value: \"\"\n            - name: TARGET_PODS\n              value: \"\"\n            - name: DEFAULT_HEALTH_CHECK\n              value: \"false\"\n            - name: NODE_LABEL\n              value: \"\"\n            - name: SEQUENCE\n              value: parallel\n"
                                }
                            }
                        ]
                    },
                    "outputs": {},
                    "metadata": {
                        "labels": {
                            "weight": "10"
                        }
                    },
                    "container": {
                        "name": "",
                        "image": "docker.io/litmuschaos/litmus-checker:2.11.0",
                        "args": [
                            "-file=/tmp/pod-delete-ce5.yaml",
                            "-saveName=/tmp/engine-name"
                        ],
                        "resources": {}
                    }
                },
                {
                    "name": "cleanup-chaos-resources",
                    "inputs": {},
                    "outputs": {},
                    "metadata": {},
                    "container": {
                        "name": "",
                        "image": "litmuschaos/k8s:2.11.0",
                        "command": [
                            "sh",
                            "-c"
                        ],
                        "args": [
                            "kubectl delete chaosengine -l workflow_run_id={{ .WorkflowUID }} -n {{ .WorkflowAdminNamespace }}"
                        ],
                        "resources": {}
                    }
                }
            ]
        },
        "status": {}
    }`
    
    // Define template data structure
    type WorkflowTemplateData struct {
        ExperimentName         string
        AppNamespace           string
        AppLabel               string
        AppKind                string
        ChaosDuration          string
        ChaosInterval          string
        WorkflowUID            string
        WorkflowAdminNamespace string
    }
    
    // Populate template data
    data := WorkflowTemplateData{
        ExperimentName:         experimentName,
        AppNamespace:           "litmus-2", // Default or get from details if available
        AppLabel:               "app=nginx", // Default or get from details if available
        AppKind:                "deployment", // Default or get from details if available
        ChaosDuration:          "15", // Default or get from details
        ChaosInterval:          "5", // Default or get from details
        WorkflowUID:            "{{ workflow.uid }}", // Pass Argo variables through template
        WorkflowAdminNamespace: "{{workflow.parameters.adminModeNamespace}}", // Pass Argo variables through template
    }
    
    // Parse the template
    tmpl, err := template.New("workflow").Parse(workflowTemplate)
    if err != nil {
        return nil, fmt.Errorf("failed to parse workflow template: %v", err)
    }
    
    // Execute the template with our data
    var manifestBuffer bytes.Buffer
    if err := tmpl.Execute(&manifestBuffer, data); err != nil {
        return nil, fmt.Errorf("failed to execute workflow template: %v", err)
    }
    
    // Construct the experiment request
    experimentRequest := &models.SaveChaosExperimentRequest{
        ID:          experimentID,
        Name:        experimentName,  
        InfraID:     details.ConnectedInfraID,
        Description: "Pod delete chaos experiment execution",
        Tags:        []string{"pod", "chaos", "litmus"},
        Manifest:    manifestBuffer.String(),
    }

    return experimentRequest, nil
}

