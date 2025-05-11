package workflow

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	probe "github.com/litmuschaos/litmus-go-sdk/pkg/apis/probe"
	"github.com/litmuschaos/litmus-go-sdk/pkg/sdk"
	models "github.com/litmuschaos/litmus/chaoscenter/graphql/server/graph/model"
)

// ExperimentType defines the available chaos experiment types
type ExperimentType string

const (
	// Pod state chaos
	PodDelete    ExperimentType = "pod-delete"
	PodCPUHog    ExperimentType = "pod-cpu-hog"
	PodMemoryHog ExperimentType = "pod-memory-hog"
	
	// Network chaos
	PodNetworkCorruption   ExperimentType = "pod-network-corruption"
	PodNetworkLatency      ExperimentType = "pod-network-latency"
	PodNetworkLoss         ExperimentType = "pod-network-loss"
	PodNetworkDuplication  ExperimentType = "pod-network-duplication"
	
	// Autoscaling
	PodAutoscaler  ExperimentType = "pod-autoscaler"
	
	// Container chaos
	ContainerKill  ExperimentType = "container-kill"
	
	// Disk chaos
	DiskFill  ExperimentType = "disk-fill"
	
	// Node chaos
	NodeCPUHog    ExperimentType = "node-cpu-hog"
	NodeMemoryHog ExperimentType = "node-memory-hog"
	NodeIOStress  ExperimentType = "node-io-stress"
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
	TargetPods        string
	DefaultHealthCheck string
	
	// CPU hog specific parameters
	CPUCores          string
	
	// Memory hog specific parameters
	MemoryConsumption string
	
	// Node memory hog specific parameters
	MemoryConsumptionPercentage  string
	MemoryConsumptionMebibytes   string
	NumberOfWorkers              string
	
	// Network chaos common parameters
	NetworkInterface  string
	TCImage           string
	LibImage          string
	ContainerRuntime  string
	SocketPath        string
	DestinationIPs    string
	DestinationHosts  string
	NodeLabel         string
	Sequence          string
	
	// Network corruption specific
	NetworkPacketCorruptionPercentage string
	
	// Network latency specific
	NetworkLatency  string
	Jitter          string
	
	// Network loss specific
	NetworkPacketLossPercentage string
	
	// Network duplication specific
	NetworkPacketDuplicationPercentage string
	
	// Pod Autoscaler specific
	ReplicaCount      string
	
	// Container kill specific parameters
	Signal            string
	
	// Disk fill specific parameters
	FillPercentage            string
	DataBlockSize             string
	EphemeralStorageMebibytes string

	// Probe configuration
	UseExistingProbe  bool
	ProbeName         string
	ProbeMode         string
}

// GetDefaultExperimentConfig returns default configuration for a given experiment type
func GetDefaultExperimentConfig(experimentType ExperimentType) ExperimentConfig {
	// Base config with common defaults
	config := ExperimentConfig{
		AppNamespace:       "litmus-2",
		AppLabel:           "app=nginx",
		AppKind:            "deployment",
		PodsAffectedPerc:   "",
		RampTime:           "",
		TargetContainer:    "",
		DefaultHealthCheck: "false",
		UseExistingProbe:   true,
		ProbeName:          "myprobe",
		ProbeMode:          "SOT",
		Sequence:           "parallel",
	}
	
	// Set network experiment common defaults
	if isNetworkExperiment(experimentType) {
		config.NetworkInterface = "eth0"
		config.TCImage = "gaiadocker/iproute2"
		config.LibImage = "litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0"
		config.ContainerRuntime = "containerd"
		config.SocketPath = "/run/containerd/containerd.sock"
		config.DestinationIPs = ""
		config.DestinationHosts = ""
		config.NodeLabel = ""
		config.TargetPods = ""
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
	
	case PodNetworkCorruption:
		config.ChaosDuration = "60"
		config.NetworkPacketCorruptionPercentage = "100"
		config.Description = "Pod network corruption chaos experiment execution"
		config.Tags = []string{"pod-network-corruption", "network-chaos", "litmus"}
	
	case PodNetworkLatency:
		config.ChaosDuration = "60"
		config.NetworkLatency = "2000"
		config.Jitter = "0"
		config.Description = "Pod network latency chaos experiment execution"
		config.Tags = []string{"pod-network-latency", "network-chaos", "litmus"}
	
	case PodNetworkLoss:
		config.ChaosDuration = "60"
		config.NetworkPacketLossPercentage = "100"
		config.Description = "Pod network loss chaos experiment execution"
		config.Tags = []string{"pod-network-loss", "network-chaos", "litmus"}
	
	case PodNetworkDuplication:
		config.ChaosDuration = "60"
		config.NetworkPacketDuplicationPercentage = "100"
		config.Description = "Pod network duplication chaos experiment execution"
		config.Tags = []string{"pod-network-duplication", "network-chaos", "litmus"}
		
	case PodAutoscaler:
		config.ChaosDuration = "60"
		config.ReplicaCount = "5"
		config.Description = "Pod autoscaler chaos experiment execution"
		config.Tags = []string{"pod-autoscaler", "autoscaling", "litmus"}
		
	case ContainerKill:
		config.ChaosDuration = "20"
		config.ChaosInterval = "10"
		config.Signal = "SIGKILL"
		config.Description = "Container kill chaos experiment execution"
		config.Tags = []string{"container-kill", "chaos", "litmus"}

	case DiskFill:
		config.ChaosDuration = "60"
		config.FillPercentage = "80"
		config.DataBlockSize = "256"
		config.EphemeralStorageMebibytes = ""
		config.Description = "Disk fill chaos experiment execution"
		config.Tags = []string{"disk-fill", "chaos", "litmus"}
		
	case NodeCPUHog:
		config.ChaosDuration = "60"
		config.CPUCores = "2"
		config.Description = "Node CPU hog chaos experiment execution"
		config.Tags = []string{"node-cpu-hog", "chaos", "litmus"}
		
	case NodeMemoryHog:
		config.ChaosDuration = "60"
		config.MemoryConsumptionPercentage = ""
		config.MemoryConsumptionMebibytes = "500"  // Set a default value
		config.NumberOfWorkers = "1"
		config.NodeLabel = ""  // Explicitly set to empty
		config.Description = "Node memory hog chaos experiment execution" 
		config.Tags = []string{"node-memory-hog", "chaos", "litmus"}
		
	case NodeIOStress:
		config.ChaosDuration = "60"
		config.Description = "Node IO stress chaos experiment execution"
		config.Tags = []string{"node-io-stress", "chaos", "litmus"}
	}
	
	return config
}

// isNetworkExperiment returns true if the experiment type is a network experiment
func isNetworkExperiment(experimentType ExperimentType) bool {
	return experimentType == PodNetworkCorruption ||
		experimentType == PodNetworkLatency ||
		experimentType == PodNetworkLoss ||
		experimentType == PodNetworkDuplication
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
	manifestStr = strings.ReplaceAll(manifestStr, "__TARGET_PODS_VALUE__", config.TargetPods)
	manifestStr = strings.ReplaceAll(manifestStr, "__DEFAULT_HEALTH_CHECK_VALUE__", config.DefaultHealthCheck)
	
	// Replace network experiment specific placeholders
	if isNetworkExperiment(experimentType) {
		manifestStr = strings.ReplaceAll(manifestStr, "__NETWORK_INTERFACE_VALUE__", config.NetworkInterface)
		manifestStr = strings.ReplaceAll(manifestStr, "__TC_IMAGE_VALUE__", config.TCImage)
		manifestStr = strings.ReplaceAll(manifestStr, "__LIB_IMAGE_VALUE__", config.LibImage)
		manifestStr = strings.ReplaceAll(manifestStr, "__CONTAINER_RUNTIME_VALUE__", config.ContainerRuntime)
		manifestStr = strings.ReplaceAll(manifestStr, "__SOCKET_PATH_VALUE__", config.SocketPath)
		manifestStr = strings.ReplaceAll(manifestStr, "__DESTINATION_IPS_VALUE__", config.DestinationIPs)
		manifestStr = strings.ReplaceAll(manifestStr, "__DESTINATION_HOSTS_VALUE__", config.DestinationHosts)
		manifestStr = strings.ReplaceAll(manifestStr, "__SEQUENCE_VALUE__", config.Sequence)
		
		// Handle NODE_LABEL specially - if it's empty, remove the entire env var entry
		if config.NodeLabel == "" {
			// More aggressive pattern match for disk-fill
			nodeLabelRegex1 := regexp.MustCompile(`\s*-\s+name:\s+NODE_LABEL\s+value:\s+["']?__NODE_LABEL_VALUE__["']?\s*`)
			nodeLabelRegex2 := regexp.MustCompile(`\s*-\s+name:\s+["']?NODE_LABEL["']?\s+value:\s+.*\s*`)
			nodeLabelRegex3 := regexp.MustCompile(`\s*name:\s+["']?NODE_LABEL["']?\s+value:\s+.*\s*`)
			
			manifestStr = nodeLabelRegex1.ReplaceAllString(manifestStr, "")
			manifestStr = nodeLabelRegex2.ReplaceAllString(manifestStr, "")
			manifestStr = nodeLabelRegex3.ReplaceAllString(manifestStr, "")
		} else {
			manifestStr = strings.ReplaceAll(manifestStr, "__NODE_LABEL_VALUE__", config.NodeLabel)
		}
		
		// Replace experiment-specific network parameters
		switch experimentType {
		case PodNetworkCorruption:
			manifestStr = strings.ReplaceAll(manifestStr, "__NETWORK_PACKET_CORRUPTION_PERCENTAGE_VALUE__", config.NetworkPacketCorruptionPercentage)
		case PodNetworkLatency:
			manifestStr = strings.ReplaceAll(manifestStr, "__NETWORK_LATENCY_VALUE__", config.NetworkLatency)
			manifestStr = strings.ReplaceAll(manifestStr, "__JITTER_VALUE__", config.Jitter)
		case PodNetworkLoss:
			manifestStr = strings.ReplaceAll(manifestStr, "__NETWORK_PACKET_LOSS_PERCENTAGE_VALUE__", config.NetworkPacketLossPercentage)
		case PodNetworkDuplication:
			manifestStr = strings.ReplaceAll(manifestStr, "__NETWORK_PACKET_DUPLICATION_PERCENTAGE_VALUE__", config.NetworkPacketDuplicationPercentage)
		}
	} else {
		// Replace non-network specific placeholders
		switch experimentType {
		case PodCPUHog:
			manifestStr = strings.ReplaceAll(manifestStr, "__CPU_CORES_VALUE__", config.CPUCores)
		case PodMemoryHog:
			manifestStr = strings.ReplaceAll(manifestStr, "__MEMORY_CONSUMPTION_VALUE__", config.MemoryConsumption)
		case PodAutoscaler:
			manifestStr = strings.ReplaceAll(manifestStr, "__REPLICA_COUNT_VALUE__", config.ReplicaCount)
		case ContainerKill:
			manifestStr = strings.ReplaceAll(manifestStr, "__SIGNAL_VALUE__", config.Signal)
		case DiskFill:
			manifestStr = strings.ReplaceAll(manifestStr, "__FILL_PERCENTAGE_VALUE__", config.FillPercentage)
			manifestStr = strings.ReplaceAll(manifestStr, "__DATA_BLOCK_SIZE_VALUE__", config.DataBlockSize)
			manifestStr = strings.ReplaceAll(manifestStr, "__EPHEMERAL_STORAGE_MEBIBYTES_VALUE__", config.EphemeralStorageMebibytes)
			
			// Handle NODE_LABEL specially for DiskFill experiment - if it's empty, remove the entire env var entry
			if config.NodeLabel == "" {
				// More aggressive pattern match for disk-fill
				nodeLabelRegex1 := regexp.MustCompile(`\s*-\s+name:\s+NODE_LABEL\s+value:\s+["']?__NODE_LABEL_VALUE__["']?\s*`)
				nodeLabelRegex2 := regexp.MustCompile(`\s*-\s+name:\s+["']?NODE_LABEL["']?\s+value:\s+.*\s*`)
				nodeLabelRegex3 := regexp.MustCompile(`\s*name:\s+["']?NODE_LABEL["']?\s+value:\s+.*\s*`)
				
				manifestStr = nodeLabelRegex1.ReplaceAllString(manifestStr, "")
				manifestStr = nodeLabelRegex2.ReplaceAllString(manifestStr, "")
				manifestStr = nodeLabelRegex3.ReplaceAllString(manifestStr, "")
			} else {
				manifestStr = strings.ReplaceAll(manifestStr, "__NODE_LABEL_VALUE__", config.NodeLabel)
			}
		case NodeCPUHog:
			// NODE_LABEL specific handling (similar to disk-fill)
			if config.NodeLabel == "" {
				// Remove NODE_LABEL environment variable if empty
				nodeLabelRegex1 := regexp.MustCompile(`\s*-\s+name:\s+NODE_LABEL\s+value:\s+["']?__NODE_LABEL_VALUE__["']?\s*`)
				nodeLabelRegex2 := regexp.MustCompile(`\s*-\s+name:\s+["']?NODE_LABEL["']?\s+value:\s+.*\s*`)
				nodeLabelRegex3 := regexp.MustCompile(`\s*name:\s+["']?NODE_LABEL["']?\s+value:\s+.*\s*`)
				
				manifestStr = nodeLabelRegex1.ReplaceAllString(manifestStr, "")
				manifestStr = nodeLabelRegex2.ReplaceAllString(manifestStr, "")
				manifestStr = nodeLabelRegex3.ReplaceAllString(manifestStr, "")
			} else {
				manifestStr = strings.ReplaceAll(manifestStr, "__NODE_LABEL_VALUE__", config.NodeLabel)
			}
		case NodeMemoryHog:
			// Replace memory consumption values
			manifestStr = strings.ReplaceAll(manifestStr, "__MEMORY_CONSUMPTION_PERCENTAGE_VALUE__", config.MemoryConsumptionPercentage)
			manifestStr = strings.ReplaceAll(manifestStr, "__MEMORY_CONSUMPTION_MEBIBYTES_VALUE__", config.MemoryConsumptionMebibytes)
			manifestStr = strings.ReplaceAll(manifestStr, "__NUMBER_OF_WORKERS_VALUE__", config.NumberOfWorkers)
			manifestStr = strings.ReplaceAll(manifestStr, "__TARGET_NODES_VALUE__", config.TargetPods)
			manifestStr = strings.ReplaceAll(manifestStr, "__NODES_AFFECTED_PERC_VALUE__", config.PodsAffectedPerc)
			
			// Handle NODE_LABEL specially - if it's empty, replace with empty string
			if config.NodeLabel == "" {
				manifestStr = strings.ReplaceAll(manifestStr, "__NODE_LABEL_VALUE__", "")
			} else {
				manifestStr = strings.ReplaceAll(manifestStr, "__NODE_LABEL_VALUE__", config.NodeLabel)
			}
		}
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
  name: pod-delete
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
      # Additional permissions omitted for brevity
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
      name: pod-delete`
	case PodCPUHog:
		return `apiVersion: litmuschaos.io/v1alpha1
description:
  message: |
    Injects cpu consumption on pods belonging to an app deployment
kind: ChaosExperiment
metadata:
  name: pod-cpu-hog
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
    - ./experiments -name pod-cpu-hog
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
      name: pod-cpu-hog`

	case PodMemoryHog:
		return `apiVersion: litmuschaos.io/v1alpha1
description:
  message: |
    Injects memory consumption on pods belonging to an app deployment
kind: ChaosExperiment
metadata:
  name: pod-memory-hog
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
    - ./experiments -name pod-memory-hog
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
      name: pod-memory-hog`

    case PodNetworkCorruption:
        return `apiVersion: litmuschaos.io/v1alpha1
description:
  message: |
    Inject network packet corruption into application pod
kind: ChaosExperiment
metadata:
  name: pod-network-corruption
  labels:
    name: pod-network-corruption
    app.kubernetes.io/part-of: litmus
    app.kubernetes.io/component: chaosexperiment
    app.kubernetes.io/version: 3.16.0
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
    image: litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0
    imagePullPolicy: Always
    args:
    - -c
    - ./experiments -name pod-network-corruption
    command:
    - /bin/bash
    env:
    - name: TARGET_CONTAINER
      value: "__TARGET_CONTAINER_VALUE__"
    - name: LIB_IMAGE
      value: "__LIB_IMAGE_VALUE__"
    - name: NETWORK_INTERFACE
      value: "__NETWORK_INTERFACE_VALUE__"
    - name: TC_IMAGE
      value: "__TC_IMAGE_VALUE__"
    - name: NETWORK_PACKET_CORRUPTION_PERCENTAGE
      value: "__NETWORK_PACKET_CORRUPTION_PERCENTAGE_VALUE__"
    - name: TOTAL_CHAOS_DURATION
      value: "__CHAOS_DURATION_VALUE__"
    - name: RAMP_TIME
      value: "__RAMP_TIME_VALUE__"
    - name: PODS_AFFECTED_PERC
      value: "__PODS_AFFECTED_PERC_VALUE__"
    - name: TARGET_PODS
      value: "__TARGET_PODS_VALUE__"
    - name: NODE_LABEL
      value: "__NODE_LABEL_VALUE__"
    - name: CONTAINER_RUNTIME
      value: "__CONTAINER_RUNTIME_VALUE__"
    - name: DESTINATION_IPS
      value: "__DESTINATION_IPS_VALUE__"
    - name: DESTINATION_HOSTS
      value: "__DESTINATION_HOSTS_VALUE__"
    - name: SOCKET_PATH
      value: "__SOCKET_PATH_VALUE__"
    - name: DEFAULT_HEALTH_CHECK
      value: "__DEFAULT_HEALTH_CHECK_VALUE__"
    - name: SEQUENCE
      value: "__SEQUENCE_VALUE__"
    labels:
      name: pod-network-corruption`

    case PodNetworkLatency:
        return `apiVersion: litmuschaos.io/v1alpha1
description:
  message: |
    Injects network latency on pods belonging to an app deployment
kind: ChaosExperiment
metadata:
  name: pod-network-latency
  labels:
    name: pod-network-latency
    app.kubernetes.io/part-of: litmus
    app.kubernetes.io/component: chaosexperiment
    app.kubernetes.io/version: 3.16.0
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
    image: litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0
    imagePullPolicy: Always
    args:
    - -c
    - ./experiments -name pod-network-latency
    command:
    - /bin/bash
    env:
    - name: TARGET_CONTAINER
      value: "__TARGET_CONTAINER_VALUE__"
    - name: NETWORK_INTERFACE
      value: "__NETWORK_INTERFACE_VALUE__"
    - name: LIB_IMAGE
      value: "__LIB_IMAGE_VALUE__"
    - name: TC_IMAGE
      value: "__TC_IMAGE_VALUE__"
    - name: NETWORK_LATENCY
      value: "__NETWORK_LATENCY_VALUE__"
    - name: TOTAL_CHAOS_DURATION
      value: "__CHAOS_DURATION_VALUE__"
    - name: RAMP_TIME
      value: "__RAMP_TIME_VALUE__"
    - name: JITTER
      value: "__JITTER_VALUE__"
    - name: PODS_AFFECTED_PERC
      value: "__PODS_AFFECTED_PERC_VALUE__"
    - name: TARGET_PODS
      value: "__TARGET_PODS_VALUE__"
    - name: CONTAINER_RUNTIME
      value: "__CONTAINER_RUNTIME_VALUE__"
    - name: DEFAULT_HEALTH_CHECK
      value: "__DEFAULT_HEALTH_CHECK_VALUE__"
    - name: DESTINATION_IPS
      value: "__DESTINATION_IPS_VALUE__"
    - name: DESTINATION_HOSTS
      value: "__DESTINATION_HOSTS_VALUE__"
    - name: SOCKET_PATH
      value: "__SOCKET_PATH_VALUE__"
    - name: NODE_LABEL
      value: "__NODE_LABEL_VALUE__"
    - name: SEQUENCE
      value: "__SEQUENCE_VALUE__"
    labels:
      name: pod-network-latency`

    case PodNetworkLoss:
        return `apiVersion: litmuschaos.io/v1alpha1
description:
  message: |
    Injects network packet loss on pods belonging to an app deployment
kind: ChaosExperiment
metadata:
  name: pod-network-loss
  labels:
    name: pod-network-loss
    app.kubernetes.io/part-of: litmus
    app.kubernetes.io/component: chaosexperiment
    app.kubernetes.io/version: 3.16.0
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
    image: litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0
    imagePullPolicy: Always
    args:
    - -c
    - ./experiments -name pod-network-loss
    command:
    - /bin/bash
    env:
    - name: TARGET_CONTAINER
      value: "__TARGET_CONTAINER_VALUE__"
    - name: LIB_IMAGE
      value: "__LIB_IMAGE_VALUE__"
    - name: NETWORK_INTERFACE
      value: "__NETWORK_INTERFACE_VALUE__"
    - name: TC_IMAGE
      value: "__TC_IMAGE_VALUE__"
    - name: NETWORK_PACKET_LOSS_PERCENTAGE
      value: "__NETWORK_PACKET_LOSS_PERCENTAGE_VALUE__"
    - name: TOTAL_CHAOS_DURATION
      value: "__CHAOS_DURATION_VALUE__"
    - name: RAMP_TIME
      value: "__RAMP_TIME_VALUE__"
    - name: PODS_AFFECTED_PERC
      value: "__PODS_AFFECTED_PERC_VALUE__"
    - name: DEFAULT_HEALTH_CHECK
      value: "__DEFAULT_HEALTH_CHECK_VALUE__"
    - name: TARGET_PODS
      value: "__TARGET_PODS_VALUE__"
    - name: NODE_LABEL
      value: "__NODE_LABEL_VALUE__"
    - name: CONTAINER_RUNTIME
      value: "__CONTAINER_RUNTIME_VALUE__"
    - name: DESTINATION_IPS
      value: "__DESTINATION_IPS_VALUE__"
    - name: DESTINATION_HOSTS
      value: "__DESTINATION_HOSTS_VALUE__"
    - name: SOCKET_PATH
      value: "__SOCKET_PATH_VALUE__"
    - name: SEQUENCE
      value: "__SEQUENCE_VALUE__"
    labels:
      name: pod-network-loss`

    case PodNetworkDuplication:
        return `apiVersion: litmuschaos.io/v1alpha1
description:
  message: |
    Injects network packet duplication on pods belonging to an app deployment
kind: ChaosExperiment
metadata:
  name: pod-network-duplication
  labels:
    name: pod-network-duplication
    app.kubernetes.io/part-of: litmus
    app.kubernetes.io/component: chaosexperiment
    app.kubernetes.io/version: 3.16.0
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
    image: litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0
    imagePullPolicy: Always
    args:
    - -c
    - ./experiments -name pod-network-duplication
    command:
    - /bin/bash
    env:
    - name: TOTAL_CHAOS_DURATION
      value: "__CHAOS_DURATION_VALUE__"
    - name: RAMP_TIME
      value: "__RAMP_TIME_VALUE__"
    - name: TARGET_CONTAINER
      value: "__TARGET_CONTAINER_VALUE__"
    - name: TC_IMAGE
      value: "__TC_IMAGE_VALUE__"
    - name: NETWORK_INTERFACE
      value: "__NETWORK_INTERFACE_VALUE__"
    - name: NETWORK_PACKET_DUPLICATION_PERCENTAGE
      value: "__NETWORK_PACKET_DUPLICATION_PERCENTAGE_VALUE__"
    - name: TARGET_PODS
      value: "__TARGET_PODS_VALUE__"
    - name: NODE_LABEL
      value: "__NODE_LABEL_VALUE__"
    - name: PODS_AFFECTED_PERC
      value: "__PODS_AFFECTED_PERC_VALUE__"
    - name: LIB_IMAGE
      value: "__LIB_IMAGE_VALUE__"
    - name: CONTAINER_RUNTIME
      value: "__CONTAINER_RUNTIME_VALUE__"
    - name: DEFAULT_HEALTH_CHECK
      value: "__DEFAULT_HEALTH_CHECK_VALUE__"
    - name: DESTINATION_IPS
      value: "__DESTINATION_IPS_VALUE__"
    - name: DESTINATION_HOSTS
      value: "__DESTINATION_HOSTS_VALUE__"
    - name: SOCKET_PATH
      value: "__SOCKET_PATH_VALUE__"
    - name: SEQUENCE
      value: "__SEQUENCE_VALUE__"
    labels:
      name: pod-network-duplication`
      
    case PodAutoscaler:
        return `apiVersion: litmuschaos.io/v1alpha1
description:
  message: |
    Scale the application replicas and test the node autoscaling on cluster
kind: ChaosExperiment
metadata:
  name: pod-autoscaler
  labels:
    name: pod-autoscaler
    app.kubernetes.io/part-of: litmus
    app.kubernetes.io/component: chaosexperiment
    app.kubernetes.io/version: 3.16.0
spec:
  definition:
    scope: Cluster
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
        verbs:
          - list
          - get
          - patch
          - update
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
    image: litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0
    imagePullPolicy: Always
    args:
      - -c
      - ./experiments -name pod-autoscaler
    command:
      - /bin/bash
    env:
      - name: TOTAL_CHAOS_DURATION
        value: "__CHAOS_DURATION_VALUE__"
      - name: RAMP_TIME
        value: "__RAMP_TIME_VALUE__"
      - name: REPLICA_COUNT
        value: "__REPLICA_COUNT_VALUE__"
      - name: DEFAULT_HEALTH_CHECK
        value: "__DEFAULT_HEALTH_CHECK_VALUE__"
    labels:
      name: pod-autoscaler
      app.kubernetes.io/part-of: litmus
      app.kubernetes.io/component: experiment-job
      app.kubernetes.io/version: 3.16.0`

	case ContainerKill:
		return `apiVersion: litmuschaos.io/v1alpha1
description:
  message: |
    Kills a container belonging to an application pod 
kind: ChaosExperiment
metadata:
  name: container-kill
  labels:
    name: container-kill
    app.kubernetes.io/part-of: litmus
    app.kubernetes.io/component: chaosexperiment
    app.kubernetes.io/version: 3.16.0
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
    image: "litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0"
    imagePullPolicy: Always
    args:
    - -c
    - ./experiments -name container-kill
    command:
    - /bin/bash
    env:
    - name: TARGET_CONTAINER
      value: '__TARGET_CONTAINER_VALUE__'
    - name: RAMP_TIME
      value: '__RAMP_TIME_VALUE__'
    - name: TARGET_PODS
      value: '__TARGET_PODS_VALUE__'
    - name: CHAOS_INTERVAL
      value: '__CHAOS_INTERVAL_VALUE__'
    - name: SIGNAL
      value: '__SIGNAL_VALUE__'
    - name: SOCKET_PATH
      value: '/run/containerd/containerd.sock'
    - name: CONTAINER_RUNTIME
      value: 'containerd'
    - name: TOTAL_CHAOS_DURATION
      value: '__CHAOS_DURATION_VALUE__'
    - name: PODS_AFFECTED_PERC
      value: '__PODS_AFFECTED_PERC_VALUE__'
    - name: DEFAULT_HEALTH_CHECK
      value: '__DEFAULT_HEALTH_CHECK_VALUE__'
    - name: LIB_IMAGE
      value: 'litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0'
    - name: SEQUENCE
      value: 'parallel'
    labels:
      name: container-kill`

	case DiskFill:
		return `apiVersion: litmuschaos.io/v1alpha1
description:
  message: |
    Fillup Ephemeral Storage of a Resource
kind: ChaosExperiment
metadata:
  name: disk-fill
  labels:
    name: disk-fill
    app.kubernetes.io/part-of: litmus
    app.kubernetes.io/component: chaosexperiment
    app.kubernetes.io/version: 3.16.0
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
    image: "litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0"
    imagePullPolicy: Always
    args:
    - -c
    - ./experiments -name disk-fill
    command:
    - /bin/bash
    env:
    - name: TARGET_CONTAINER
      value: '__TARGET_CONTAINER_VALUE__'
    - name: FILL_PERCENTAGE
      value: '__FILL_PERCENTAGE_VALUE__'
    - name: TOTAL_CHAOS_DURATION
      value: '__CHAOS_DURATION_VALUE__'
    - name: RAMP_TIME
      value: '__RAMP_TIME_VALUE__'
    - name: DATA_BLOCK_SIZE
      value: '__DATA_BLOCK_SIZE_VALUE__'
    - name: TARGET_PODS
      value: '__TARGET_PODS_VALUE__'
    - name: EPHEMERAL_STORAGE_MEBIBYTES
      value: '__EPHEMERAL_STORAGE_MEBIBYTES_VALUE__'
    - name: PODS_AFFECTED_PERC
      value: '__PODS_AFFECTED_PERC_VALUE__'
    - name: DEFAULT_HEALTH_CHECK
      value: '__DEFAULT_HEALTH_CHECK_VALUE__'
    - name: LIB_IMAGE
      value: 'litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0'
    - name: SOCKET_PATH
      value: '/run/containerd/containerd.sock'
    - name: CONTAINER_RUNTIME
      value: 'containerd'
    - name: SEQUENCE
      value: 'parallel'
    labels:
      name: disk-fill`

	case NodeCPUHog:
		return `apiVersion: litmuschaos.io/v1alpha1
description:
  message: |
    Injects cpu consumption on node
kind: ChaosExperiment
metadata:
  name: node-cpu-hog
  labels:
    name: node-cpu-hog
    app.kubernetes.io/part-of: litmus
    app.kubernetes.io/component: chaosexperiment
    app.kubernetes.io/version: 3.16.0
spec:
  definition:
    scope: Cluster
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
    image: "litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0"
    imagePullPolicy: Always
    args:
    - -c
    - ./experiments -name node-cpu-hog
    command:
    - /bin/bash
    env:
    - name: TOTAL_CHAOS_DURATION
      value: '__CHAOS_DURATION_VALUE__'
    - name: RAMP_TIME
      value: '__RAMP_TIME_VALUE__'
    - name: NODE_CPU_CORE
      value: ''
    - name: CPU_LOAD
      value: '100'
    - name: NODES_AFFECTED_PERC
      value: '__PODS_AFFECTED_PERC_VALUE__'
    - name: TARGET_NODES
      value: '__TARGET_PODS_VALUE__'
    - name: DEFAULT_HEALTH_CHECK
      value: '__DEFAULT_HEALTH_CHECK_VALUE__'
    - name: LIB_IMAGE
      value: 'litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0'
    - name: SEQUENCE
      value: 'parallel'
    labels:
      name: node-cpu-hog`
      
	case NodeMemoryHog:
		return `apiVersion: litmuschaos.io/v1alpha1
description:
  message: |
    Give a memory hog on a node belonging to a deployment
kind: ChaosExperiment
metadata:
  name: node-memory-hog
  labels:
    name: node-memory-hog
    app.kubernetes.io/part-of: litmus
    app.kubernetes.io/component: chaosexperiment
    app.kubernetes.io/version: 3.16.0
spec:
  definition:
    scope: Cluster
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
    image: "litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0"
    imagePullPolicy: Always
    args:
    - -c
    - ./experiments -name node-memory-hog
    command:
    - /bin/bash
    env:
    - name: TOTAL_CHAOS_DURATION
      value: '__CHAOS_DURATION_VALUE__'
    - name: RAMP_TIME
      value: '__RAMP_TIME_VALUE__'
    - name: MEMORY_CONSUMPTION_PERCENTAGE
      value: '__MEMORY_CONSUMPTION_PERCENTAGE_VALUE__'
    - name: MEMORY_CONSUMPTION_MEBIBYTES
      value: '__MEMORY_CONSUMPTION_MEBIBYTES_VALUE__'
    - name: NUMBER_OF_WORKERS
      value: '__NUMBER_OF_WORKERS_VALUE__'
    - name: TARGET_NODES
      value: '__TARGET_NODES_VALUE__'
    - name: NODE_LABEL
      value: '__NODE_LABEL_VALUE__'
    - name: NODES_AFFECTED_PERC
      value: '__NODES_AFFECTED_PERC_VALUE__'
    - name: DEFAULT_HEALTH_CHECK
      value: '__DEFAULT_HEALTH_CHECK_VALUE__'
    - name: LIB_IMAGE
      value: "litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0"
    - name: SEQUENCE
      value: "parallel"`
      
	case NodeIOStress:
		return `apiVersion: litmuschaos.io/v1alpha1
description:
  message: |
    Injects IO stress on node
kind: ChaosExperiment
metadata:
  name: node-io-stress
  labels:
    name: node-io-stress
    app.kubernetes.io/part-of: litmus
    app.kubernetes.io/component: chaosexperiment
    app.kubernetes.io/version: 3.16.0
spec:
  definition:
    scope: Cluster
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
    image: "litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0"
    imagePullPolicy: Always
    args:
    - -c
    - ./experiments -name node-io-stress
    command:
    - /bin/bash
    env:
    - name: TOTAL_CHAOS_DURATION
      value: '__CHAOS_DURATION_VALUE__'
    - name: RAMP_TIME
      value: '__RAMP_TIME_VALUE__'
    - name: FILESYSTEM_UTILIZATION_PERCENTAGE
      value: '10'
    - name: FILESYSTEM_UTILIZATION_BYTES
      value: ''
    - name: NODES_AFFECTED_PERC
      value: '__PODS_AFFECTED_PERC_VALUE__'
    - name: TARGET_NODES
      value: '__TARGET_PODS_VALUE__'
    - name: DEFAULT_HEALTH_CHECK
      value: '__DEFAULT_HEALTH_CHECK_VALUE__'
    - name: LIB_IMAGE
      value: 'litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0'
    - name: SEQUENCE
      value: 'parallel'
    labels:
      name: node-io-stress`

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
  generateName: pod-delete-ce5
spec:
  appinfo:
    appns: __APP_NAMESPACE__
    applabel: __APP_LABEL__
    appkind: __APP_KIND__
  engineState: active
  chaosServiceAccount: litmus-admin
  experiments:
    - name: pod-delete
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
  generateName: pod-cpu-hog-ce5
spec:
  appinfo:
    appns: __APP_NAMESPACE__
    applabel: __APP_LABEL__
    appkind: __APP_KIND__
  engineState: active
  chaosServiceAccount: litmus-admin
  experiments:
    - name: pod-cpu-hog
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
  generateName: pod-memory-hog-ce5
spec:
  appinfo:
    appns: __APP_NAMESPACE__
    applabel: __APP_LABEL__
    appkind: __APP_KIND__
  engineState: active
  chaosServiceAccount: litmus-admin
  experiments:
    - name: pod-memory-hog
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

    case PodNetworkCorruption:
        return `apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  namespace: "{{workflow.parameters.adminModeNamespace}}"
  labels:
    workflow_run_id: "{{ workflow.uid }}"
    workflow_name: __EXPERIMENT_NAME__
  generateName: pod-network-corruption-ce5
spec:
  engineState: active
  appinfo:
    appns: __APP_NAMESPACE__
    applabel: __APP_LABEL__
    appkind: __APP_KIND__
  chaosServiceAccount: litmus-admin
  experiments:
    - name: pod-network-corruption
      spec:
        components:
          env:
            - name: TARGET_CONTAINER
              value: "__TARGET_CONTAINER_VALUE__"
            - name: LIB_IMAGE
              value: "__LIB_IMAGE_VALUE__"
            - name: NETWORK_INTERFACE
              value: "__NETWORK_INTERFACE_VALUE__"
            - name: TC_IMAGE
              value: "__TC_IMAGE_VALUE__"
            - name: NETWORK_PACKET_CORRUPTION_PERCENTAGE
              value: "__NETWORK_PACKET_CORRUPTION_PERCENTAGE_VALUE__"
            - name: TOTAL_CHAOS_DURATION
              value: "__CHAOS_DURATION_VALUE__"
            - name: RAMP_TIME
              value: "__RAMP_TIME_VALUE__"
            - name: PODS_AFFECTED_PERC
              value: "__PODS_AFFECTED_PERC_VALUE__"
            - name: TARGET_PODS
              value: "__TARGET_PODS_VALUE__"
            - name: NODE_LABEL
              value: "__NODE_LABEL_VALUE__"
            - name: CONTAINER_RUNTIME
              value: "__CONTAINER_RUNTIME_VALUE__"
            - name: DESTINATION_IPS
              value: "__DESTINATION_IPS_VALUE__"
            - name: DESTINATION_HOSTS
              value: "__DESTINATION_HOSTS_VALUE__"
            - name: SOCKET_PATH
              value: "__SOCKET_PATH_VALUE__"
            - name: DEFAULT_HEALTH_CHECK
              value: "__DEFAULT_HEALTH_CHECK_VALUE__"
            - name: SEQUENCE
              value: "__SEQUENCE_VALUE__"`

    case PodNetworkLatency:
        return `apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  namespace: "{{workflow.parameters.adminModeNamespace}}"
  labels:
    workflow_run_id: "{{ workflow.uid }}"
    workflow_name: __EXPERIMENT_NAME__
  generateName: pod-network-latency-ce5
spec:
  engineState: active
  appinfo:
    appns: __APP_NAMESPACE__
    applabel: __APP_LABEL__
    appkind: __APP_KIND__
  chaosServiceAccount: litmus-admin
  experiments:
    - name: pod-network-latency
      spec:
        components:
          env:
            - name: TARGET_CONTAINER
              value: "__TARGET_CONTAINER_VALUE__"
            - name: NETWORK_INTERFACE
              value: "__NETWORK_INTERFACE_VALUE__"
            - name: LIB_IMAGE
              value: "__LIB_IMAGE_VALUE__"
            - name: TC_IMAGE
              value: "__TC_IMAGE_VALUE__"
            - name: NETWORK_LATENCY
              value: "__NETWORK_LATENCY_VALUE__"
            - name: TOTAL_CHAOS_DURATION
              value: "__CHAOS_DURATION_VALUE__"
            - name: RAMP_TIME
              value: "__RAMP_TIME_VALUE__"
            - name: JITTER
              value: "__JITTER_VALUE__"
            - name: PODS_AFFECTED_PERC
              value: "__PODS_AFFECTED_PERC_VALUE__"
            - name: TARGET_PODS
              value: "__TARGET_PODS_VALUE__"
            - name: CONTAINER_RUNTIME
              value: "__CONTAINER_RUNTIME_VALUE__"
            - name: DEFAULT_HEALTH_CHECK
              value: "__DEFAULT_HEALTH_CHECK_VALUE__"
            - name: DESTINATION_IPS
              value: "__DESTINATION_IPS_VALUE__"
            - name: DESTINATION_HOSTS
              value: "__DESTINATION_HOSTS_VALUE__"
            - name: SOCKET_PATH
              value: "__SOCKET_PATH_VALUE__"
            - name: NODE_LABEL
              value: "__NODE_LABEL_VALUE__"
            - name: SEQUENCE
              value: "__SEQUENCE_VALUE__"`

    case PodNetworkLoss:
        return `apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  namespace: "{{workflow.parameters.adminModeNamespace}}"
  labels:
    workflow_run_id: "{{ workflow.uid }}"
    workflow_name: __EXPERIMENT_NAME__
  generateName: pod-network-loss-ce5
spec:
  engineState: active
  appinfo:
    appns: __APP_NAMESPACE__
    applabel: __APP_LABEL__
    appkind: __APP_KIND__
  chaosServiceAccount: litmus-admin
  experiments:
    - name: pod-network-loss
      spec:
        components:
          env:
            - name: TARGET_CONTAINER
              value: "__TARGET_CONTAINER_VALUE__"
            - name: LIB_IMAGE
              value: "__LIB_IMAGE_VALUE__"
            - name: NETWORK_INTERFACE
              value: "__NETWORK_INTERFACE_VALUE__"
            - name: TC_IMAGE
              value: "__TC_IMAGE_VALUE__"
            - name: NETWORK_PACKET_LOSS_PERCENTAGE
              value: "__NETWORK_PACKET_LOSS_PERCENTAGE_VALUE__"
            - name: TOTAL_CHAOS_DURATION
              value: "__CHAOS_DURATION_VALUE__"
            - name: RAMP_TIME
              value: "__RAMP_TIME_VALUE__"
            - name: PODS_AFFECTED_PERC
              value: "__PODS_AFFECTED_PERC_VALUE__"
            - name: DEFAULT_HEALTH_CHECK
              value: "__DEFAULT_HEALTH_CHECK_VALUE__"
            - name: TARGET_PODS
              value: "__TARGET_PODS_VALUE__"
            - name: NODE_LABEL
              value: "__NODE_LABEL_VALUE__"
            - name: CONTAINER_RUNTIME
              value: "__CONTAINER_RUNTIME_VALUE__"
            - name: DESTINATION_IPS
              value: "__DESTINATION_IPS_VALUE__"
            - name: DESTINATION_HOSTS
              value: "__DESTINATION_HOSTS_VALUE__"
            - name: SOCKET_PATH
              value: "__SOCKET_PATH_VALUE__"
            - name: SEQUENCE
              value: "__SEQUENCE_VALUE__"`

    case PodNetworkDuplication:
        return `apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  namespace: "{{workflow.parameters.adminModeNamespace}}"
  labels:
    workflow_run_id: "{{ workflow.uid }}"
    workflow_name: __EXPERIMENT_NAME__
  generateName: pod-network-duplication-ce5
spec:
  engineState: active
  appinfo:
    appns: __APP_NAMESPACE__
    applabel: __APP_LABEL__
    appkind: __APP_KIND__
  chaosServiceAccount: litmus-admin
  experiments:
    - name: pod-network-duplication
      spec:
        components:
          env:
            - name: TOTAL_CHAOS_DURATION
              value: "__CHAOS_DURATION_VALUE__"
            - name: RAMP_TIME
              value: "__RAMP_TIME_VALUE__"
            - name: TARGET_CONTAINER
              value: "__TARGET_CONTAINER_VALUE__"
            - name: TC_IMAGE
              value: "__TC_IMAGE_VALUE__"
            - name: NETWORK_INTERFACE
              value: "__NETWORK_INTERFACE_VALUE__"
            - name: NETWORK_PACKET_DUPLICATION_PERCENTAGE
              value: "__NETWORK_PACKET_DUPLICATION_PERCENTAGE_VALUE__"
            - name: TARGET_PODS
              value: "__TARGET_PODS_VALUE__"
            - name: NODE_LABEL
              value: "__NODE_LABEL_VALUE__"
            - name: PODS_AFFECTED_PERC
              value: "__PODS_AFFECTED_PERC_VALUE__"
            - name: LIB_IMAGE
              value: "__LIB_IMAGE_VALUE__"
            - name: CONTAINER_RUNTIME
              value: "__CONTAINER_RUNTIME_VALUE__"
            - name: DEFAULT_HEALTH_CHECK
              value: "__DEFAULT_HEALTH_CHECK_VALUE__"
            - name: DESTINATION_IPS
              value: "__DESTINATION_IPS_VALUE__"
            - name: DESTINATION_HOSTS
              value: "__DESTINATION_HOSTS_VALUE__"
            - name: SOCKET_PATH
              value: "__SOCKET_PATH_VALUE__"
            - name: SEQUENCE
              value: "__SEQUENCE_VALUE__"`
              
    case PodAutoscaler:
        return `apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  name: nginx-chaos
  namespace: "{{workflow.parameters.adminModeNamespace}}"
  labels:
    workflow_run_id: "{{ workflow.uid }}"
    workflow_name: __EXPERIMENT_NAME__
  generateName: pod-autoscaler-ce5
spec:
  engineState: active
  auxiliaryAppInfo: ""
  appinfo:
    appns: __APP_NAMESPACE__
    applabel: __APP_LABEL__
    appkind: __APP_KIND__
  chaosServiceAccount: litmus-admin
  experiments:
    - name: pod-autoscaler
      spec:
        components:
          env:
            - name: TOTAL_CHAOS_DURATION
              value: "__CHAOS_DURATION_VALUE__"
            - name: RAMP_TIME
              value: "__RAMP_TIME_VALUE__"
            - name: REPLICA_COUNT
              value: "__REPLICA_COUNT_VALUE__"
            - name: DEFAULT_HEALTH_CHECK
              value: "__DEFAULT_HEALTH_CHECK_VALUE__"`

    case ContainerKill:
	    return `apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  namespace: "{{workflow.parameters.adminModeNamespace}}"
  labels:
    workflow_run_id: "{{ workflow.uid }}"
    workflow_name: __EXPERIMENT_NAME__
  generateName: container-kill-ce5
spec:
  engineState: active
  appinfo:
    appns: __APP_NAMESPACE__
    applabel: __APP_LABEL__
    appkind: __APP_KIND__
  chaosServiceAccount: litmus-admin
  experiments:
    - name: container-kill
      spec:
        components:
          env:
            - name: TARGET_CONTAINER
              value: "__TARGET_CONTAINER_VALUE__"
            - name: RAMP_TIME
              value: "__RAMP_TIME_VALUE__"
            - name: TARGET_PODS
              value: "__TARGET_PODS_VALUE__"
            - name: CHAOS_INTERVAL
              value: "__CHAOS_INTERVAL_VALUE__"
            - name: SIGNAL
              value: "__SIGNAL_VALUE__"
            - name: SOCKET_PATH
              value: "/run/containerd/containerd.sock"
            - name: CONTAINER_RUNTIME
              value: "containerd"
            - name: TOTAL_CHAOS_DURATION
              value: "__CHAOS_DURATION_VALUE__"
            - name: PODS_AFFECTED_PERC
              value: "__PODS_AFFECTED_PERC_VALUE__"
            - name: DEFAULT_HEALTH_CHECK
              value: "__DEFAULT_HEALTH_CHECK_VALUE__"
            - name: LIB_IMAGE
              value: "litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0"
            - name: SEQUENCE
              value: "parallel"`

    case DiskFill:
	    return `apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  namespace: "{{workflow.parameters.adminModeNamespace}}"
  labels:
    workflow_run_id: "{{ workflow.uid }}"
    workflow_name: __EXPERIMENT_NAME__
  generateName: disk-fill-ce5
spec:
  engineState: active
  appinfo:
    appns: __APP_NAMESPACE__
    applabel: __APP_LABEL__
    appkind: __APP_KIND__
  chaosServiceAccount: litmus-admin
  experiments:
    - name: disk-fill
      spec:
        components:
          env:
            - name: TARGET_CONTAINER
              value: "__TARGET_CONTAINER_VALUE__"
            - name: FILL_PERCENTAGE
              value: "__FILL_PERCENTAGE_VALUE__"
            - name: TOTAL_CHAOS_DURATION
              value: "__CHAOS_DURATION_VALUE__"
            - name: RAMP_TIME
              value: "__RAMP_TIME_VALUE__"
            - name: DATA_BLOCK_SIZE
              value: "__DATA_BLOCK_SIZE_VALUE__"
            - name: TARGET_PODS
              value: "__TARGET_PODS_VALUE__"
            - name: EPHEMERAL_STORAGE_MEBIBYTES
              value: "__EPHEMERAL_STORAGE_MEBIBYTES_VALUE__"
            - name: PODS_AFFECTED_PERC
              value: "__PODS_AFFECTED_PERC_VALUE__"
            - name: DEFAULT_HEALTH_CHECK
              value: "__DEFAULT_HEALTH_CHECK_VALUE__"
            - name: LIB_IMAGE
              value: "litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0"
            - name: SOCKET_PATH
              value: "/run/containerd/containerd.sock"
            - name: CONTAINER_RUNTIME
              value: "containerd"
            - name: SEQUENCE
              value: "parallel"`
			  
	case NodeCPUHog:
		return `apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  namespace: "{{workflow.parameters.adminModeNamespace}}"
  labels:
    workflow_run_id: "{{ workflow.uid }}"
    workflow_name: __EXPERIMENT_NAME__
  generateName: node-cpu-hog-ce5
spec:
  engineState: active
  annotationCheck: "false"
  chaosServiceAccount: litmus-admin
  experiments:
    - name: node-cpu-hog
      spec:
        components:
          env:
            - name: TOTAL_CHAOS_DURATION
              value: "__CHAOS_DURATION_VALUE__"
            - name: RAMP_TIME
              value: "__RAMP_TIME_VALUE__"
            - name: NODE_CPU_CORE
              value: ""
            - name: CPU_LOAD
              value: "100"
            - name: NODES_AFFECTED_PERC
              value: "__PODS_AFFECTED_PERC_VALUE__"
            - name: TARGET_NODES
              value: "__TARGET_PODS_VALUE__"
            - name: DEFAULT_HEALTH_CHECK
              value: "__DEFAULT_HEALTH_CHECK_VALUE__"
            - name: SEQUENCE
              value: "parallel"`
			  
	case NodeMemoryHog:
		return `apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  namespace: "{{workflow.parameters.adminModeNamespace}}"
  labels:
    workflow_run_id: "{{ workflow.uid }}"
    workflow_name: __EXPERIMENT_NAME__
  generateName: node-memory-hog-ce5
spec:
  engineState: active
  annotationCheck: "false"
  chaosServiceAccount: litmus-admin
  experiments:
    - name: node-memory-hog
      spec:
        components:
          env:
            - name: TOTAL_CHAOS_DURATION
              value: "__CHAOS_DURATION_VALUE__"
            - name: RAMP_TIME
              value: "__RAMP_TIME_VALUE__"
            - name: MEMORY_CONSUMPTION_PERCENTAGE
              value: "__MEMORY_CONSUMPTION_PERCENTAGE_VALUE__"
            - name: MEMORY_CONSUMPTION_MEBIBYTES
              value: "__MEMORY_CONSUMPTION_MEBIBYTES_VALUE__"
            - name: NUMBER_OF_WORKERS
              value: "__NUMBER_OF_WORKERS_VALUE__"
            - name: TARGET_NODES
              value: "__TARGET_NODES_VALUE__"
            - name: NODE_LABEL
              value: "__NODE_LABEL_VALUE__"
            - name: NODES_AFFECTED_PERC
              value: "__NODES_AFFECTED_PERC_VALUE__"
            - name: DEFAULT_HEALTH_CHECK
              value: "__DEFAULT_HEALTH_CHECK_VALUE__"
            - name: LIB_IMAGE
              value: "litmuschaos.docker.scarf.sh/litmuschaos/go-runner:3.16.0"
            - name: SEQUENCE
              value: "parallel"`
			  
	case NodeIOStress:
		return `apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  namespace: "{{workflow.parameters.adminModeNamespace}}"
  labels:
    workflow_run_id: "{{ workflow.uid }}"
    workflow_name: __EXPERIMENT_NAME__
  generateName: node-io-stress-ce5
spec:
  engineState: active
  annotationCheck: "false"
  chaosServiceAccount: litmus-admin
  experiments:
    - name: node-io-stress
      spec:
        components:
          env:
            - name: TOTAL_CHAOS_DURATION
              value: "__CHAOS_DURATION_VALUE__"
            - name: RAMP_TIME
              value: "__RAMP_TIME_VALUE__"
            - name: FILESYSTEM_UTILIZATION_PERCENTAGE
              value: "10"
            - name: FILESYSTEM_UTILIZATION_BYTES
              value: ""
            - name: NODES_AFFECTED_PERC
              value: "__PODS_AFFECTED_PERC_VALUE__"
            - name: TARGET_NODES
              value: "__TARGET_PODS_VALUE__"
            - name: DEFAULT_HEALTH_CHECK
              value: "__DEFAULT_HEALTH_CHECK_VALUE__"
            - name: SEQUENCE
              value: "parallel"`

    default:
        return ""
    }
}

// Helper functions for constructing experiment requests with default configuration
func ConstructPodDeleteExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(PodDelete)
	applyProbeConfigFromEnv(&config)
	return ConstructExperimentRequest(details, experimentID, experimentName, PodDelete, config)
}

func ConstructPodCPUHogExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(PodCPUHog)
	applyProbeConfigFromEnv(&config)
	return ConstructExperimentRequest(details, experimentID, experimentName, PodCPUHog, config)
}

func ConstructPodMemoryHogExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(PodMemoryHog)
	applyProbeConfigFromEnv(&config)
	return ConstructExperimentRequest(details, experimentID, experimentName, PodMemoryHog, config)
}

func ConstructPodNetworkCorruptionExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(PodNetworkCorruption)
	applyProbeConfigFromEnv(&config)
	return ConstructExperimentRequest(details, experimentID, experimentName, PodNetworkCorruption, config)
}

func ConstructPodNetworkLatencyExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(PodNetworkLatency)
	applyProbeConfigFromEnv(&config)
	return ConstructExperimentRequest(details, experimentID, experimentName, PodNetworkLatency, config)
}

func ConstructPodNetworkLossExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(PodNetworkLoss)
	applyProbeConfigFromEnv(&config)
	return ConstructExperimentRequest(details, experimentID, experimentName, PodNetworkLoss, config)
}

func ConstructPodNetworkDuplicationExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(PodNetworkDuplication)
	applyProbeConfigFromEnv(&config)
	return ConstructExperimentRequest(details, experimentID, experimentName, PodNetworkDuplication, config)
}

func ConstructPodAutoscalerExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(PodAutoscaler)
	applyProbeConfigFromEnv(&config)
	return ConstructExperimentRequest(details, experimentID, experimentName, PodAutoscaler, config)
}

func ConstructContainerKillExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(ContainerKill)
	applyProbeConfigFromEnv(&config)
	return ConstructExperimentRequest(details, experimentID, experimentName, ContainerKill, config)
}

func ConstructDiskFillExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(DiskFill)
	applyProbeConfigFromEnv(&config)
	return ConstructExperimentRequest(details, experimentID, experimentName, DiskFill, config)
}

func ConstructNodeCPUHogExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(NodeCPUHog)
	applyProbeConfigFromEnv(&config)
	return ConstructExperimentRequest(details, experimentID, experimentName, NodeCPUHog, config)
}

func ConstructNodeMemoryHogExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(NodeMemoryHog)
	applyProbeConfigFromEnv(&config)
	return ConstructExperimentRequest(details, experimentID, experimentName, NodeMemoryHog, config)
}

func ConstructNodeIOStressExperimentRequest(details *types.ExperimentDetails, experimentID string, experimentName string) (*models.SaveChaosExperimentRequest, error) {
	config := GetDefaultExperimentConfig(NodeIOStress)
	applyProbeConfigFromEnv(&config)
	return ConstructExperimentRequest(details, experimentID, experimentName, NodeIOStress, config)
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
				// We now support creating custom probes through the SDK.
				// This is handled in the CreateProbe function which should be called before experiment creation.
				// Environment variables LITMUS_CREATE_PROBE, LITMUS_PROBE_TYPE, etc. control probe creation.
				log.Printf("Note: To create a custom probe, set LITMUS_CREATE_PROBE=true along with other probe parameters.")
				log.Printf("For now, using specified probe details: %s with mode: %s\n", config.ProbeName, config.ProbeMode)
			} else {
				log.Printf("Using existing probe: %s with mode: %s\n", config.ProbeName, config.ProbeMode)
			}
		} else {
			log.Printf("Warning: Failed to parse LITMUS_USE_EXISTING_PROBE environment variable: %v\n", err)
		}
	} else {
		log.Printf("No probe configuration provided. Using default probe: %s with mode: %s\n", config.ProbeName, config.ProbeMode)
	}
}

// CreateProbe creates a probe using the experiment details
func CreateProbe(details *types.ExperimentDetails, sdkClient sdk.Client, litmusProjectID string) error {
	if !details.CreateProbe {
		log.Println("Skipping probe creation as LITMUS_CREATE_PROBE is not set to true")
		return nil
	}

	log.Printf("Creating a new probe with name: %s", details.ProbeName)

	// Setup defaults for HTTP probe
	trueBool := true
	desc := fmt.Sprintf("HTTP probe for %s", details.ProbeName)
	
	// Prepare probe request based on probe type
	var probeReq probe.ProbeRequest
	
	switch details.ProbeType {
	case "httpProbe":
		probeReq = probe.ProbeRequest{
			Name:               details.ProbeName,
			Description:        &desc,
			Type:               probe.ProbeTypeHTTPProbe,
			InfrastructureType: probe.InfrastructureTypeKubernetes,
			Tags:               []string{"http", "probe", "chaos"},
			KubernetesHTTPProperties: &probe.KubernetesHTTPProbeRequest{
				ProbeTimeout: details.ProbeTimeout,
				Interval:     details.ProbeInterval,
				Attempt:      &details.ProbeAttempts,
				URL:          details.ProbeURL,
				Method: &probe.Method{
					Get: &probe.GetMethod{
						ResponseCode: details.ProbeResponseCode,
						Criteria:     "==",
					},
				},
				InsecureSkipVerify: &trueBool,
			},
		}
	case "cmdProbe":
		probeReq = probe.ProbeRequest{
			Name:               details.ProbeName,
			Description:        &desc,
			Type:               probe.ProbeTypeCMDProbe,
			InfrastructureType: probe.InfrastructureTypeKubernetes,
			Tags:               []string{"cmd", "probe", "chaos"},
			KubernetesCMDProperties: &probe.KubernetesCMDProbeRequest{
				Command:      "ls -l",
				ProbeTimeout: details.ProbeTimeout,
				Interval:     details.ProbeInterval,
				Attempt:      &details.ProbeAttempts,
				Comparator: &probe.ComparatorInput{
					Type:     "string",
					Criteria: "contains",
					Value:    "total",
				},
			},
		}
	default:
		return fmt.Errorf("unsupported probe type: %s", details.ProbeType)
	}
	
	// Create probe
	createdProbe, err := sdkClient.Probes().Create(probeReq, litmusProjectID)
	if err != nil {
		log.Printf("Failed to create probe: %v", err)
		return err
	}

	log.Printf("Successfully created probe: %s", createdProbe.Name)
	
  details.CreatedProbeID = createdProbe.Name
	return nil
}