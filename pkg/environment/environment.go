package environment

import (
	"os"
	"strconv"

	types "github.com/litmuschaos/chaos-ci-lib/pkg/types"
)

// GetENV fetches all the env variables from the runner pod
func GetENV(experimentDetails *types.ExperimentDetails, expName, engineName string) {
	experimentDetails.ExperimentName = expName
	experimentDetails.EngineName = engineName
	experimentDetails.OperatorName = Getenv("OPERATOR_NAME", "chaos-operator-ce")
	experimentDetails.ChaosNamespace = Getenv("CHAOS_NAMESPACE", "default")
	experimentDetails.AppNS = Getenv("APP_NS", "litmus")
	experimentDetails.AppLabel = Getenv("APP_LABEL", "app=nginx")
	experimentDetails.AppKind = Getenv("APP_KIND", "deployment")
	experimentDetails.JobCleanUpPolicy = Getenv("JOB_CLEANUP_POLICY", "retain")
	experimentDetails.AnnotationCheck = Getenv("ANNOTATION_CHECK", "false")
	experimentDetails.ApplicationNodeName = Getenv("APPLICATION_NODE_NAME", "")
	experimentDetails.NodeSelectorName = Getenv("APPLICATION_NODE_NAME", "")
	experimentDetails.ImagePullPolicy = Getenv("IMAGE_PULL_POLICY", "Always")
	experimentDetails.ChaosDuration, _ = strconv.Atoi(Getenv("TOTAL_CHAOS_DURATION", "60"))
	experimentDetails.ChaosInterval, _ = strconv.Atoi(Getenv("CHAOS_INTERVAL", "30"))
	experimentDetails.TargetContainer = Getenv("TARGET_CONTAINER", "")
	experimentDetails.CPU, _ = strconv.Atoi(Getenv("CPU_CORES", "1"))
	experimentDetails.NodeCPUCore, _ = strconv.Atoi(Getenv("NODE_CPU_CORE", "2"))
	experimentDetails.Force = Getenv("FORCE", "false")
	experimentDetails.ChaosServiceAccount = Getenv("CHAOS_SERVICE_ACCOUNT", expName+"-sa")
	experimentDetails.Delay, _ = strconv.Atoi(Getenv("DELAY", "5"))
	experimentDetails.Duration, _ = strconv.Atoi(Getenv("DURATION", "90"))
	experimentDetails.DiskFillPercentage, _ = strconv.Atoi(Getenv("FILL_PERCENTAGE", "20"))
	experimentDetails.CpuInjectCommand = Getenv("CPU_KILL_COMMAND", "md5sum /dev/zero")
	experimentDetails.MemoryConsumption, _ = strconv.Atoi(Getenv("MEMORY_CONSUMPTION", "500"))
	experimentDetails.FillPercentage, _ = strconv.Atoi(Getenv("MEMORY_PERCENTAGE", "80"))
	experimentDetails.ContainerRuntime = Getenv("CONTAINER_RUNTIME", "containerd")
	experimentDetails.ContainerPath = Getenv("CONTAINER_PATH", "/var/lib/containerd/io.containerd.grpc.v1.cri/containers/")
	experimentDetails.SocketPath = Getenv("SOCKET_PATH", "/run/containerd/containerd.sock")
	experimentDetails.MemoryConsumptionPercentage, _ = strconv.Atoi(Getenv("MEMORY_CONSUMPTION_PERCENTAGE", "30"))
	experimentDetails.NetworkInterface = Getenv("NETWORK_INTERFACE", "eth0")
	experimentDetails.NetworkPacketDuplicationPercentage, _ = strconv.Atoi(Getenv("NETWORK_PACKET_DUPLICATION_PERCENTAGE", "100"))
	experimentDetails.FileSystemUtilizationPercentage, _ = strconv.Atoi(Getenv("FILESYSTEM_UTILIZATION_PERCENTAGE", "10"))
	experimentDetails.NetworkPacketLossPercentage, _ = strconv.Atoi(Getenv("NETWORK_PACKET_LOSS_PERCENTAGE", "100"))
	experimentDetails.TargetPods = Getenv("TARGET_PODS", "")
	experimentDetails.PodsAffectedPerc, _ = strconv.Atoi(Getenv("PODS_AFFECTED_PERC", "0"))
	experimentDetails.NodesAffectedPerc, _ = strconv.Atoi(Getenv("NODES_AFFECTED_PERC", "0"))
	experimentDetails.FilesystemUtilizationBytes, _ = strconv.Atoi(Getenv("FILESYSTEM_UTILIZATION_BYTES", ""))
	experimentDetails.Replicas, _ = strconv.Atoi(Getenv("REPLICA_COUNT", "0"))
	experimentDetails.ExperimentTimeout, _ = strconv.Atoi(Getenv("EXPERIMENT_TIMEOUT", "8"))
	experimentDetails.ExperimentPollingInterval, _ = strconv.Atoi(Getenv("EXPERIMENT_POLLING_INTERVAL", "15"))

	//All Images for running chaos test
	experimentDetails.GoExperimentImage = Getenv("EXPERIMENT_IMAGE", "litmuschaos/go-runner:ci")
	experimentDetails.OperatorImage = Getenv("OPERATOR_IMAGE", "litmuschaos/chaos-operator:ci")
	experimentDetails.RunnerImage = Getenv("RUNNER_IMAGE", "litmuschaos/chaos-runner:ci")

	// All Links for running chaos testing
	experimentDetails.RbacPath = Getenv("RBAC_PATH", "https://hub.litmuschaos.io/api/chaos/master?file=charts/generic/"+expName+"/rbac.yaml")
	experimentDetails.EnginePath = Getenv("ENGINE_PATH", "https://hub.litmuschaos.io/api/chaos/master?file=charts/generic/"+expName+"/engine.yaml")
	experimentDetails.InstallLitmus = Getenv("INSTALL_LITMUS_URL", "https://litmuschaos.github.io/litmus/litmus-operator-latest.yaml")

	// V3 SDK Related ENV parsing
	experimentDetails.InstallLitmusFlag, _ = strconv.ParseBool(Getenv("INSTALL_CHAOS_CENTER", "false"))
	experimentDetails.ConnectInfraFlag, _ = strconv.ParseBool(Getenv("CONNECT_INFRA", "false"))
	experimentDetails.LitmusEndpoint = Getenv("LITMUS_ENDPOINT", "")
	experimentDetails.LitmusUsername = Getenv("LITMUS_USERNAME", "")
	experimentDetails.LitmusPassword = Getenv("LITMUS_PASSWORD", "")
	experimentDetails.LitmusProjectID = Getenv("LITMUS_PROJECT_ID", "")
	experimentDetails.InfraName = Getenv("INFRA_NAME", "ci-infra-"+expName)
	experimentDetails.InfraNamespace = Getenv("INFRA_NAMESPACE", "litmus")
	experimentDetails.InfraScope = Getenv("INFRA_SCOPE", "namespace")
	experimentDetails.InfraSA = Getenv("INFRA_SERVICE_ACCOUNT", "litmus")
	experimentDetails.InfraDescription = Getenv("INFRA_DESCRIPTION", "CI Test Infrastructure")
	experimentDetails.InfraPlatformName = Getenv("INFRA_PLATFORM_NAME", "others")
	experimentDetails.InfraEnvironmentID = Getenv("INFRA_ENVIRONMENT_ID", "")
	experimentDetails.InfraNsExists, _ = strconv.ParseBool(Getenv("INFRA_NS_EXISTS", "false"))
	experimentDetails.InfraSaExists, _ = strconv.ParseBool(Getenv("INFRA_SA_EXISTS", "false"))
	experimentDetails.InfraSkipSSL, _ = strconv.ParseBool(Getenv("INFRA_SKIP_SSL", "false"))
	experimentDetails.InfraNodeSelector = Getenv("INFRA_NODE_SELECTOR", "")
	experimentDetails.InfraTolerations = Getenv("INFRA_TOLERATIONS", "")

	// New infrastructure control variables
	experimentDetails.InstallInfra, _ = strconv.ParseBool(Getenv("INSTALL_INFRA", "true"))
	experimentDetails.UseExistingInfra, _ = strconv.ParseBool(Getenv("USE_EXISTING_INFRA", "false"))
	experimentDetails.ExistingInfraID = Getenv("EXISTING_INFRA_ID", "")

	// Infrastructure activation control
	experimentDetails.ActivateInfra, _ = strconv.ParseBool(Getenv("ACTIVATE_INFRA", "true"))
	experimentDetails.InfraActivationTimeout, _ = strconv.Atoi(Getenv("INFRA_ACTIVATION_TIMEOUT", "5"))

	// Probe configuration
	experimentDetails.CreateProbe, _ = strconv.ParseBool(Getenv("LITMUS_CREATE_PROBE", "false"))
	experimentDetails.ProbeType = Getenv("LITMUS_PROBE_TYPE", "httpProbe")
	experimentDetails.ProbeName = Getenv("LITMUS_PROBE_NAME", "http-probe")
	experimentDetails.ProbeMode = Getenv("LITMUS_PROBE_MODE", "SOT")
	experimentDetails.ProbeURL = Getenv("LITMUS_PROBE_URL", "http://localhost:8080/health")
	experimentDetails.ProbeTimeout = Getenv("LITMUS_PROBE_TIMEOUT", "30s")
	experimentDetails.ProbeInterval = Getenv("LITMUS_PROBE_INTERVAL", "10s")
	experimentDetails.ProbeAttempts, _ = strconv.Atoi(Getenv("LITMUS_PROBE_ATTEMPTS", "1"))
	experimentDetails.ProbeResponseCode = Getenv("LITMUS_PROBE_RESPONSE_CODE", "200")
}

// Getenv fetch the env and set the default value, if any
func Getenv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return value
}
