package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	models "github.com/litmuschaos/litmus/chaoscenter/graphql/server/graph/model"
)

// ExperimentType defines the available chaos experiment types
type ExperimentType string

const (
	PodDelete    ExperimentType = "pod-delete"
	PodCPUHog    ExperimentType = "pod-cpu-hog"
	PodMemoryHog ExperimentType = "pod-memory-hog"
	// Add more experiment types as needed
)

// ExperimentConfig holds configuration for an experiment
type ExperimentConfig struct {
	AppNamespace      string
	AppLabel          string
	AppKind           string
	ChaosDuration     string
	ChaosInterval     string
	Description       string
	Tags              []string
	
	// Common parameters
	TargetContainer   string
	PodsAffectedPerc  string
	RampTime          string
	
	// CPU hog specific parameters
	CPUCores          string
	
	// Memory hog specific parameters
	MemoryConsumption string

	// Probe configuration
	UseExistingProbe  bool
	ProbeName         string
	ProbeMode         string
}

// GetDefaultExperimentConfig returns default configuration for a given experiment type
func GetDefaultExperimentConfig(experimentType ExperimentType) ExperimentConfig {
	config := ExperimentConfig{
		AppNamespace:     "litmus-2",
		AppLabel:         "app=nginx",
		AppKind:          "deployment",
		PodsAffectedPerc: "",
		RampTime:         "",
		TargetContainer:  "",
		UseExistingProbe: true,  // Always use a probe by default
		ProbeName:        "myprobe",
		ProbeMode:        "SOT",
	}
	
	// Apply experiment-specific defaults
	switch experimentType {
	case PodDelete:
		config.ChaosDuration = "15"
		config.ChaosInterval = "5"
		config.Description = "Pod delete chaos experiment execution"
		config.Tags = []string{"pod-delete", "chaos", "litmus"}
	
	case PodCPUHog:
		config.ChaosDuration = "30"
		config.ChaosInterval = "10"
		config.CPUCores = "1"
		config.Description = "Pod CPU hog chaos experiment execution"
		config.Tags = []string{"pod-cpu-hog", "chaos", "litmus"}
	
	case PodMemoryHog:
		config.ChaosDuration = "30"
		config.ChaosInterval = "10"
		config.MemoryConsumption = "500"
		config.Description = "Pod memory hog chaos experiment execution"
		config.Tags = []string{"pod-memory-hog", "chaos", "litmus"}
	}
	
	return config
}

// ConstructExperimentRequest creates an Argo Workflow manifest for LitmusChaos
func ConstructExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string, experimentType ExperimentType, config ExperimentConfig) (*models.SaveChaosExperimentRequest, error) {
	// Get base workflow manifest for the experiment type
	manifest, err := GetExperimentManifest(experimentType, experimentName, config)
	if err != nil {
		return nil, fmt.Errorf("failed to get experiment manifest: %v", err)
	}

	// Construct the experiment request
	experimentRequest := &models.SaveChaosExperimentRequest{
		ID:          experimentID,
		Name:        experimentName,
		InfraID:     details.ConnectedInfraID,
		Description: config.Description,
		Tags:        config.Tags,
		Manifest:    manifest,
	}

	return experimentRequest, nil
}

// GetExperimentManifest returns the complete workflow manifest string for a given experiment type
func GetExperimentManifest(experimentType ExperimentType, experimentName string, config ExperimentConfig) (string, error) {
	// Base workflow structure that's common for all experiments
	baseManifest := map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Workflow",
		"metadata": map[string]interface{}{
			"name":      experimentName,
			"namespace": "litmus-2",
		},
		"spec": map[string]interface{}{
			"entrypoint":         string(experimentType) + "-engine",
			"serviceAccountName": "argo-chaos",
			"podGC": map[string]string{
				"strategy": "OnWorkflowCompletion",
			},
			"securityContext": map[string]interface{}{
				"runAsUser":    1000,
				"runAsNonRoot": true,
			},
			"arguments": map[string]interface{}{
				"parameters": []map[string]string{
					{
						"name":  "adminModeNamespace",
						"value": "litmus-2",
					},
				},
			},
			"templates": []map[string]interface{}{
				// Main workflow template
				{
					"name": string(experimentType) + "-engine",
					"steps": [][]map[string]interface{}{
						{
							{
								"name":     "install-chaos-faults",
								"template": "install-chaos-faults",
								"arguments": map[string]interface{}{},
							},
						},
						{
							{
								"name":     string(experimentType) + "-ce5",
								"template": string(experimentType) + "-ce5",
								"arguments": map[string]interface{}{},
							},
						},
						{
							{
								"name":     "cleanup-chaos-resources",
								"template": "cleanup-chaos-resources",
								"arguments": map[string]interface{}{},
							},
						},
					},
				},
			},
		},
		"status": map[string]interface{}{},
	}

	// Add experiment-specific templates
	templates, err := getExperimentTemplates(experimentType, config)
	if err != nil {
		return "", err
	}

	// Append templates to the base manifest
	baseManifest["spec"].(map[string]interface{})["templates"] = append(
		baseManifest["spec"].(map[string]interface{})["templates"].([]map[string]interface{}),
		templates...,
	)

	// Convert to JSON and then to pretty-printed string
	jsonBytes, err := json.MarshalIndent(baseManifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal workflow manifest: %v", err)
	}

	// Convert JSON to string
	manifestStr := string(jsonBytes)
	
	// Replace with specific placeholders
	manifestStr = strings.ReplaceAll(manifestStr, "__EXPERIMENT_NAME__", experimentName)
	manifestStr = strings.ReplaceAll(manifestStr, "__EXPERIMENT_TYPE__", string(experimentType))
	manifestStr = strings.ReplaceAll(manifestStr, "__APP_NAMESPACE__", config.AppNamespace)
	manifestStr = strings.ReplaceAll(manifestStr, "__APP_LABEL__", config.AppLabel)
	manifestStr = strings.ReplaceAll(manifestStr, "__APP_KIND__", config.AppKind)
	manifestStr = strings.ReplaceAll(manifestStr, "__CHAOS_DURATION_VALUE__", config.ChaosDuration)
	manifestStr = strings.ReplaceAll(manifestStr, "__CHAOS_INTERVAL_VALUE__", config.ChaosInterval)
	manifestStr = strings.ReplaceAll(manifestStr, "__TARGET_CONTAINER_VALUE__", config.TargetContainer)
	manifestStr = strings.ReplaceAll(manifestStr, "__PODS_AFFECTED_PERC_VALUE__", config.PodsAffectedPerc)
	manifestStr = strings.ReplaceAll(manifestStr, "__RAMP_TIME_VALUE__", config.RampTime)

	// Replace experiment-specific variables
	switch experimentType {
	case PodCPUHog:
		manifestStr = strings.ReplaceAll(manifestStr, "__CPU_CORES_VALUE__", config.CPUCores)
	case PodMemoryHog:
		manifestStr = strings.ReplaceAll(manifestStr, "__MEMORY_CONSUMPTION_VALUE__", config.MemoryConsumption)
	}
	
	return manifestStr, nil
}

func getExperimentTemplates(experimentType ExperimentType, config ExperimentConfig) ([]map[string]interface{}, error) {
    var templates []map[string]interface{}

    // Artifact name and path matching the working YAML
    installTemplate := map[string]interface{}{
        "name": "install-chaos-faults",
        "inputs": map[string]interface{}{
            "artifacts": []map[string]interface{}{
                {
                    "name": string(experimentType) + "-ce5",
                    "path": "/tmp/" + string(experimentType) + "-ce5.yaml",
                    "raw": map[string]interface{}{
                        "data": getChaosExperimentData(experimentType),
                    },
                },
            },
        },
        "container": map[string]interface{}{
            "name":    "",
            "image":   "litmuschaos/k8s:2.11.0",
            "command": []string{"sh", "-c"},
            "args": []string{
                "kubectl apply -f /tmp/ -n {{workflow.parameters.adminModeNamespace}} && sleep 30",
            },
            "resources": map[string]interface{}{},
        },
    }

    // Create the raw data string for the chaos engine
    engineData := getChaosEngineData(experimentType)
    
    // Create a raw string version of the probe annotation to directly insert into the YAML
    probeAnnotation := fmt.Sprintf("probeRef: '[{\"name\":\"%s\",\"mode\":\"%s\"}]'", 
        config.ProbeName, config.ProbeMode)
    
    // Add the annotation directly into the YAML string
    engineData = strings.Replace(
        engineData,
        "metadata:",
        "metadata:\n  annotations:\n    " + probeAnnotation,
        1)

    runTemplate := map[string]interface{}{
        "name": string(experimentType) + "-ce5",
        "inputs": map[string]interface{}{
            "artifacts": []map[string]interface{}{
                {
                    "name": string(experimentType) + "-ce5",
                    "path": "/tmp/" + string(experimentType) + "-ce5.yaml",
                    "raw": map[string]interface{}{
                        "data": engineData,
                    },
                },
            },
        },
        "outputs": map[string]interface{}{},
        "metadata": map[string]interface{}{
            "labels": map[string]string{
                "weight": "10",
            },
        },
        "container": map[string]interface{}{
            "name":  "",
            "image": "docker.io/litmuschaos/litmus-checker:2.11.0",
            "args": []string{
                "-file=/tmp/" + string(experimentType) + "-ce5.yaml",
                "-saveName=/tmp/engine-name",
            },
            "resources": map[string]interface{}{},
        },
    }

	cleanupTemplate := map[string]interface{}{
		"name":    "cleanup-chaos-resources",
		"inputs":  map[string]interface{}{},
		"outputs": map[string]interface{}{},
		"metadata": map[string]interface{}{},
		"container": map[string]interface{}{
			"name":    "",
			"image":   "litmuschaos/k8s:2.11.0",
			"command": []string{"sh", "-c"},
			"args": []string{
				"kubectl delete chaosengine -l workflow_run_id={{ workflow.uid }} -n {{workflow.parameters.adminModeNamespace}}",
			},
			"resources": map[string]interface{}{},
		},
	}

	templates = append(templates, installTemplate, runTemplate, cleanupTemplate)
	return templates, nil
}

// getChaosExperimentData returns the ChaosExperiment definition for the specified experiment type
func getChaosExperimentData(experimentType ExperimentType) string {
	switch experimentType {
	case PodDelete:
		return `apiVersion: litmuschaos.io/v1alpha1
description:
  message: |
    Deletes a pod belonging to a deployment/statefulset/daemonset
kind: ChaosExperiment
metadata:
  name: __EXPERIMENT_TYPE__
spec:
  definition:
    scope: Namespaced
    permissions:
      - apiGroups:
          - ""
        resources:
          - pods
        verbs:
          - create
          - delete
          - get
          - list
          - patch
          - update
          - deletecollection
      - apiGroups:
          - ""
        resources:
          - events
        verbs:
          - create
          - get
          - list
          - patch
          - update
      - apiGroups:
          - ""
        resources:
          - configmaps
        verbs:
          - get
          - list
      - apiGroups:
          - ""
        resources:
          - pods/log
        verbs:
          - get
          - list
          - watch
      - apiGroups:
          - ""
        resources:
          - pods/exec
        verbs:
          - get
          - list
          - create
      - apiGroups:
          - apps
        resources:
          - deployments
          - statefulsets
          - replicasets
          - daemonsets
        verbs:
          - list
          - get
      - apiGroups:
          - apps.openshift.io
        resources:
          - deploymentconfigs
        verbs:
          - list
          - get
      - apiGroups:
          - ""
        resources:
          - replicationcontrollers
        verbs:
          - get
          - list
      - apiGroups:
          - argoproj.io
        resources:
          - rollouts
        verbs:
          - list
          - get
      - apiGroups:
          - batch
        resources:
          - jobs
        verbs:
          - create
          - list
          - get
          - delete
          - deletecollection
      - apiGroups:
          - litmuschaos.io
        resources:
          - chaosengines
          - chaosexperiments
          - chaosresults
        verbs:
          - create
          - list
          - get
          - patch
          - update
          - delete
    image: "litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0"
    imagePullPolicy: Always
    args:
    - -c
    - ./experiments -name pod-delete
    command:
    - /bin/bash
    env:
    - name: TOTAL_CHAOS_DURATION
      value: '__CHAOS_DURATION_VALUE__'
    - name: RAMP_TIME
      value: '__RAMP_TIME_VALUE__'
    - name: KILL_COUNT
      value: ''
    - name: FORCE
      value: 'true'
    - name: CHAOS_INTERVAL
      value: '__CHAOS_INTERVAL_VALUE__'
    labels:
      name: __EXPERIMENT_TYPE__`
	case PodCPUHog:
		return `apiVersion: litmuschaos.io/v1alpha1
description:
  message: |
    Injects cpu consumption on pods belonging to an app deployment
kind: ChaosExperiment
metadata:
  name: __EXPERIMENT_TYPE__
spec:
  definition:
    scope: Namespaced
    permissions:
      - apiGroups:
          - ""
          - "batch"
          - "litmuschaos.io"
        resources:
          - "jobs"
          - "pods"
          - "pods/log"
          - "events"
          - "chaosengines"
          - "chaosexperiments"
          - "chaosresults"
        verbs:
          - "create"
          - "list"
          - "get"
          - "patch"
          - "update"
          - "delete"
    image: "litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0"
    args:
    - -c
    - ./experiments -name __EXPERIMENT_TYPE__
    command:
    - /bin/bash
    env:
    - name: TOTAL_CHAOS_DURATION
      value: '__CHAOS_DURATION_VALUE__'
    - name: CHAOS_INTERVAL
      value: '__CHAOS_INTERVAL_VALUE__'
    - name: CPU_CORES
      value: '__CPU_CORES_VALUE__'
    - name: PODS_AFFECTED_PERC
      value: '__PODS_AFFECTED_PERC_VALUE__'
    - name: RAMP_TIME
      value: '__RAMP_TIME_VALUE__'
    labels:
      name: __EXPERIMENT_TYPE__`

	case PodMemoryHog:
		return `apiVersion: litmuschaos.io/v1alpha1
description:
  message: |
    Injects memory consumption on pods belonging to an app deployment
kind: ChaosExperiment
metadata:
  name: __EXPERIMENT_TYPE__
spec:
  definition:
    scope: Namespaced
    permissions:
      - apiGroups:
          - ""
          - "batch"
          - "litmuschaos.io"
        resources:
          - "jobs"
          - "pods"
          - "pods/log"
          - "events"
          - "chaosengines"
          - "chaosexperiments"
          - "chaosresults"
        verbs:
          - "create"
          - "list"
          - "get"
          - "patch"
          - "update"
          - "delete"
    image: "litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0"
    args:
    - -c
    - ./experiments -name __EXPERIMENT_TYPE__
    command:
    - /bin/bash
    env:
    - name: TOTAL_CHAOS_DURATION
      value: '__CHAOS_DURATION_VALUE__'
    - name: CHAOS_INTERVAL
      value: '__CHAOS_INTERVAL_VALUE__'
    - name: MEMORY_CONSUMPTION
      value: '__MEMORY_CONSUMPTION_VALUE__'
    - name: PODS_AFFECTED_PERC
      value: '__PODS_AFFECTED_PERC_VALUE__'
    - name: RAMP_TIME
      value: '__RAMP_TIME_VALUE__'
    labels:
      name: __EXPERIMENT_TYPE__`

	default:
		return ""
	}
}

// getChaosEngineData returns the ChaosEngine definition for the specified experiment type
func getChaosEngineData(experimentType ExperimentType) string {
    switch experimentType {
    case PodDelete:
        return `apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  namespace: "{{workflow.parameters.adminModeNamespace}}"
  labels:
    workflow_run_id: "{{ workflow.uid }}"
    workflow_name: __EXPERIMENT_NAME__
  generateName: __EXPERIMENT_TYPE__-ce5
spec:
  appinfo:
    appns: __APP_NAMESPACE__
    applabel: __APP_LABEL__
    appkind: __APP_KIND__
  engineState: active
  chaosServiceAccount: litmus-admin
  experiments:
    - name: __EXPERIMENT_TYPE__
      spec:
        components:
          env:
            - name: TOTAL_CHAOS_DURATION
              value: "__CHAOS_DURATION_VALUE__"
            - name: RAMP_TIME
              value: "__RAMP_TIME_VALUE__"
            - name: FORCE
              value: "true"
            - name: CHAOS_INTERVAL
              value: "__CHAOS_INTERVAL_VALUE__"
            - name: PODS_AFFECTED_PERC
              value: "__PODS_AFFECTED_PERC_VALUE__"
            - name: TARGET_CONTAINER
              value: "__TARGET_CONTAINER_VALUE__"`

    case PodCPUHog:
        return `apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  namespace: "{{workflow.parameters.adminModeNamespace}}"
  labels:
    workflow_run_id: "{{ workflow.uid }}"
    workflow_name: __EXPERIMENT_NAME__
  generateName: __EXPERIMENT_TYPE__-ce5
spec:
  appinfo:
    appns: __APP_NAMESPACE__
    applabel: __APP_LABEL__
    appkind: __APP_KIND__
  engineState: active
  chaosServiceAccount: litmus-admin
  experiments:
    - name: __EXPERIMENT_TYPE__
      spec:
        components:
          env:
            - name: TOTAL_CHAOS_DURATION
              value: "__CHAOS_DURATION_VALUE__"
            - name: CPU_CORES
              value: "__CPU_CORES_VALUE__"
            - name: TARGET_CONTAINER
              value: "__TARGET_CONTAINER_VALUE__"
            - name: PODS_AFFECTED_PERC
              value: "__PODS_AFFECTED_PERC_VALUE__"
            - name: RAMP_TIME
              value: "__RAMP_TIME_VALUE__"`

    case PodMemoryHog:
        return `apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  namespace: "{{workflow.parameters.adminModeNamespace}}"
  labels:
    workflow_run_id: "{{ workflow.uid }}"
    workflow_name: __EXPERIMENT_NAME__
  generateName: __EXPERIMENT_TYPE__-ce5
spec:
  appinfo:
    appns: __APP_NAMESPACE__
    applabel: __APP_LABEL__
    appkind: __APP_KIND__
  engineState: active
  chaosServiceAccount: litmus-admin
  experiments:
    - name: __EXPERIMENT_TYPE__
      spec:
        components:
          env:
            - name: TOTAL_CHAOS_DURATION
              value: "__CHAOS_DURATION_VALUE__"
            - name: MEMORY_CONSUMPTION
              value: "__MEMORY_CONSUMPTION_VALUE__"
            - name: TARGET_CONTAINER
              value: "__TARGET_CONTAINER_VALUE__"
            - name: PODS_AFFECTED_PERC
              value: "__PODS_AFFECTED_PERC_VALUE__"
            - name: RAMP_TIME
              value: "__RAMP_TIME_VALUE__"`

    default:
        return ""
    }
}

// Helper functions for specific experiment types
func ConstructPodDeleteExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(PodDelete)
	
	// Apply probe configuration from environment variables if set
	applyProbeConfigFromEnv(&config)
	
	return ConstructExperimentRequest(details, experimentID, experimentName, PodDelete, config)
}

func ConstructPodCPUHogExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(PodCPUHog)
	
	// Apply probe configuration from environment variables if set
	applyProbeConfigFromEnv(&config)
	
	return ConstructExperimentRequest(details, experimentID, experimentName, PodCPUHog, config)
}

func ConstructPodMemoryHogExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(PodMemoryHog)
	
	// Apply probe configuration from environment variables if set
	applyProbeConfigFromEnv(&config)
	
	return ConstructExperimentRequest(details, experimentID, experimentName, PodMemoryHog, config)
}

// applyProbeConfigFromEnv reads probe configuration from environment variables and applies them to the config
func applyProbeConfigFromEnv(config *ExperimentConfig) {
	// Check if probe configuration is specified in environment variables
	useExistingProbeStr := os.Getenv("LITMUS_USE_EXISTING_PROBE")
	if useExistingProbeStr != "" {
		useExistingProbe, err := strconv.ParseBool(useExistingProbeStr)
		if err == nil {
			config.UseExistingProbe = useExistingProbe
			
			// Get probe name and mode regardless of useExistingProbe value
			probeName := os.Getenv("LITMUS_PROBE_NAME")
			if probeName != "" {
				config.ProbeName = probeName
			}
			
			probeMode := os.Getenv("LITMUS_PROBE_MODE")
			if probeMode != "" {
				config.ProbeMode = probeMode
			}
			
			if !useExistingProbe {
				fmt.Println("Warning: Creating custom probes is not supported at this time. Using the specified probe details as fallback.")
			} else {
				fmt.Printf("Using probe: %s with mode: %s\n", config.ProbeName, config.ProbeMode)
			}
		} else {
			fmt.Printf("Warning: Failed to parse LITMUS_USE_EXISTING_PROBE environment variable: %v\n", err)
		}
	} else {
		fmt.Printf("No probe configuration provided. Using default probe: %s with mode: %s\n", config.ProbeName, config.ProbeMode)
	}
}

// Helper functions for specific experiment types with probe configuration
func ConstructPodDeleteExperimentRequestWithProbe(details *types.ExperimentDetails, experimentID string, experimentName string, useExistingProbe bool, probeName, probeMode string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(PodDelete)
	
	// Apply probe configuration
	config.UseExistingProbe = useExistingProbe
	if useExistingProbe {
		config.ProbeName = probeName
		config.ProbeMode = probeMode
	}
	
	return ConstructExperimentRequest(details, experimentID, experimentName, PodDelete, config)
}

func ConstructPodCPUHogExperimentRequestWithProbe(details *types.ExperimentDetails, experimentID string, experimentName string, useExistingProbe bool, probeName, probeMode string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(PodCPUHog)
	
	// Apply probe configuration
	config.UseExistingProbe = useExistingProbe
	if useExistingProbe {
		config.ProbeName = probeName
		config.ProbeMode = probeMode
	}
	
	return ConstructExperimentRequest(details, experimentID, experimentName, PodCPUHog, config)
}

func ConstructPodMemoryHogExperimentRequestWithProbe(details *types.ExperimentDetails, experimentID string, experimentName string, useExistingProbe bool, probeName, probeMode string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(PodMemoryHog)
	
	// Apply probe configuration
	config.UseExistingProbe = useExistingProbe
	if useExistingProbe {
		config.ProbeName = probeName
		config.ProbeMode = probeMode
	}
	
	return ConstructExperimentRequest(details, experimentID, experimentName, PodMemoryHog, config)
}