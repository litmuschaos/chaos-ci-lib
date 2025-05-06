package workflow

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	models "github.com/litmuschaos/litmus/chaoscenter/graphql/server/graph/model"
)

// ExperimentConfig holds configuration for an experiment
type ExperimentConfig struct {
	AppNamespace  string
	AppLabel      string
	AppKind       string
	ChaosDuration string
	ChaosInterval string
	Description   string
	Tags          []string
	// Additional fields can be added as needed for different experiments
}

// GetDefaultExperimentConfig returns default configuration for a given experiment type
func GetDefaultExperimentConfig(experimentType string) ExperimentConfig {
	config := ExperimentConfig{
		AppNamespace:  "litmus-2",
		AppLabel:      "app=nginx",
		AppKind:       "deployment",
		ChaosDuration: "15",
		ChaosInterval: "5",
		Description:   experimentType + " chaos experiment execution",
		Tags:          []string{experimentType, "chaos", "litmus"},
	}
	
	return config
}

// ConstructExperimentRequest creates an Argo Workflow manifest for Litmus 3.0
func ConstructExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string, experimentType string, experimentConfig ExperimentConfig) (*models.SaveChaosExperimentRequest, error) {
	// Get workflow template based on experiment type and configuration
	workflowTemplate, err := GetWorkflowTemplate(experimentType, experimentConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow template: %v", err)
	}
	
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
		ExperimentType         string
	}
	
	// Populate template data with defaults or values from details
	data := WorkflowTemplateData{
		ExperimentName:         experimentName,
		ExperimentType:         experimentType,
		AppNamespace:           experimentConfig.AppNamespace,
		AppLabel:               experimentConfig.AppLabel,
		AppKind:                experimentConfig.AppKind,
		ChaosDuration:          experimentConfig.ChaosDuration,
		ChaosInterval:          experimentConfig.ChaosInterval,
		WorkflowUID:            "{{ workflow.uid }}", 
		WorkflowAdminNamespace: "{{workflow.parameters.adminModeNamespace}}", 
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
		Description: experimentConfig.Description,
		Tags:        experimentConfig.Tags,
		Manifest:    manifestBuffer.String(),
	}

	return experimentRequest, nil
}

// GetWorkflowTemplate returns the workflow template for a given experiment type
func GetWorkflowTemplate(experimentType string, config ExperimentConfig) (string, error) {
	// Map of templates for different experiment types
	templates := map[string]string{
		"pod-delete": `{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind": "Workflow",
		"metadata": {
			"name": "{{.ExperimentName}}",
			"namespace": "litmus-2"
		},
		"spec": {
			"entrypoint": "{{.ExperimentType}}-engine",
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
					"name": "{{.ExperimentType}}-engine",
					"steps": [
						[
							{
								"name": "install-chaos-faults",
								"template": "install-chaos-faults"
							}
						],
						[
							{
								"name": "{{.ExperimentType}}-ce5",
								"template": "{{.ExperimentType}}-ce5"
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
								"name": "{{.ExperimentType}}-ce5",
								"path": "/tmp/{{.ExperimentType}}-ce5.yaml",
								"raw": {
									"data": "apiVersion: litmuschaos.io/v1alpha1\ndescription:\n  message: |\n    Deletes a pod belonging to a deployment/statefulset/daemonset\nkind: ChaosExperiment\nmetadata:\n  name: {{.ExperimentType}}\nspec:\n  definition:\n    scope: Namespaced\n    permissions:\n      - apiGroups:\n          - \"\"\n        resources:\n          - pods\n        verbs:\n          - create\n          - delete\n          - get\n          - list\n          - patch\n          - update\n          - deletecollection\n      - apiGroups:\n          - \"\"\n        resources:\n          - events\n        verbs:\n          - create\n          - get\n          - list\n          - patch\n          - update\n      - apiGroups:\n          - \"\"\n        resources:\n          - configmaps\n        verbs:\n          - get\n          - list\n      - apiGroups:\n          - \"\"\n        resources:\n          - pods/log\n        verbs:\n          - get\n          - list\n          - watch\n      - apiGroups:\n          - \"\"\n        resources:\n          - pods/exec\n        verbs:\n          - get\n          - list\n          - create\n      - apiGroups:\n          - apps\n        resources:\n          - deployments\n          - statefulsets\n          - replicasets\n          - daemonsets\n        verbs:\n          - list\n          - get\n      - apiGroups:\n          - apps.openshift.io\n        resources:\n          - deploymentconfigs\n        verbs:\n          - list\n          - get\n      - apiGroups:\n          - \"\"\n        resources:\n          - replicationcontrollers\n        verbs:\n          - get\n          - list\n      - apiGroups:\n          - argoproj.io\n        resources:\n          - rollouts\n        verbs:\n          - list\n          - get\n      - apiGroups:\n          - batch\n        resources:\n          - jobs\n        verbs:\n          - create\n          - list\n          - get\n          - delete\n          - deletecollection\n      - apiGroups:\n          - litmuschaos.io\n        resources:\n          - chaosengines\n          - chaosexperiments\n          - chaosresults\n        verbs:\n          - create\n          - list\n          - get\n          - patch\n          - update\n          - delete\n    image: \"litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0\"\n    imagePullPolicy: Always\n    args:\n    - -c\n    - ./experiments -name {{.ExperimentType}}\n    command:\n    - /bin/bash\n    env:\n    - name: TOTAL_CHAOS_DURATION\n      value: '15'\n    - name: RAMP_TIME\n      value: ''\n    - name: KILL_COUNT\n      value: ''\n    - name: FORCE\n      value: 'true'\n    - name: CHAOS_INTERVAL\n      value: '5'\n    labels:\n      name: {{.ExperimentType}}\n"
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
					"name": "{{.ExperimentType}}-ce5",
					"inputs": {
						"artifacts": [
							{
								"name": "{{.ExperimentType}}-ce5",
								"path": "/tmp/{{.ExperimentType}}-ce5.yaml",
								"raw": {
									"data": "apiVersion: litmuschaos.io/v1alpha1\nkind: ChaosEngine\nmetadata:\n  namespace: \"{{ .WorkflowAdminNamespace }}\"\n  labels:\n    workflow_run_id: \"{{ .WorkflowUID }}\"\n    workflow_name: {{.ExperimentName}}\n  annotations:\n    probeRef: '[{\"name\":\"myprobe\",\"mode\":\"SOT\"}]'\n  generateName: {{.ExperimentType}}-ce5\nspec:\n  appinfo:\n    appns: {{.AppNamespace}}\n    applabel: {{.AppLabel}}\n    appkind: {{.AppKind}}\n  engineState: active\n  chaosServiceAccount: litmus-admin\n  experiments:\n    - name: {{.ExperimentType}}\n      spec:\n        components:\n          env:\n            - name: TOTAL_CHAOS_DURATION\n              value: \"{{.ChaosDuration}}\"\n            - name: RAMP_TIME\n              value: \"\"\n            - name: FORCE\n              value: \"true\"\n            - name: CHAOS_INTERVAL\n              value: \"{{.ChaosInterval}}\"\n            - name: PODS_AFFECTED_PERC\n              value: \"\"\n            - name: TARGET_CONTAINER\n              value: \"\"\n            - name: TARGET_PODS\n              value: \"\"\n            - name: DEFAULT_HEALTH_CHECK\n              value: \"false\"\n            - name: NODE_LABEL\n              value: \"\"\n            - name: SEQUENCE\n              value: parallel\n"
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
							"-file=/tmp/{{.ExperimentType}}-ce5.yaml",
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
	}`,
		// Add more experiment types as needed
	}
	
	// Get the template for the specified experiment type
	tmpl, ok := templates[experimentType]
	if !ok {
		return "", fmt.Errorf("no template found for experiment type: %s", experimentType)
	}
	
	return tmpl, nil
}

// ConstructPodDeleteExperimentRequest is a helper function specifically for pod-delete experiments
func ConstructPodDeleteExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig("pod-delete")
	return ConstructExperimentRequest(details, experimentID, experimentName, "pod-delete", config)
} 