package environment

import (
	"os"
	"strconv"

	types "github.com/litmuschaos/chaos-ci-lib/pkg/types"
)

//GetENV fetches all the env variables from the runner pod
func GetENV(experimentDetails *types.ExperimentDetails, expName, engineName string) {
	experimentDetails.ExperimentName = expName
	experimentDetails.EngineName = engineName
	experimentDetails.OperatorName = Getenv("OPERATOR_NAME", "chaos-operator-ce")
	experimentDetails.ChaosNamespace = Getenv("CHAOS_NAMESPACE", "default")
	experimentDetails.AppNS = Getenv("APP_NS", "default")
	experimentDetails.AppLabel = Getenv("APP_LABEL", "app=nginx")
	experimentDetails.AppKind = Getenv("APP_KIND", "deployment")
	experimentDetails.JobCleanUpPolicy = Getenv("JOB_CLEANUP_POLICY", "'retain'")
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
	experimentDetails.FillPercentage, _ = strconv.Atoi(Getenv("FILL_PERCENTAGE", "80"))
	experimentDetails.CpuInjectCommand = Getenv("CPU_KILL_COMMAND", "md5sum /dev/zero")
	experimentDetails.MemoryConsumption, _ = strconv.Atoi(Getenv("MEMORY_CONSUMPTION", "500"))
	experimentDetails.FillPercentage, _ = strconv.Atoi(Getenv("MEMORY_PERCENTAGE", "80"))
	experimentDetails.ContainerRuntime = Getenv("CONTAINER_RUNTIME", "containerd")
	experimentDetails.SocketPath = Getenv("SOCKET_PATH", "/run/containerd/containerd.sock")
	experimentDetails.NetworkInterface = Getenv("NETWORK_INTERFACE", "eth0")
	experimentDetails.NetworkPacketDuplicationPercentage, _ = strconv.Atoi(Getenv("NETWORK_PACKET_DUPLICATION_PERCENTAGE", "100"))
	experimentDetails.FileSystemUtilizationPercentage, _ = strconv.Atoi(Getenv("FILESYSTEM_UTILIZATION_PERCENTAGE", "10"))
	experimentDetails.NetworkPacketLossPercentage, _ = strconv.Atoi(Getenv("NETWORK_PACKET_LOSS_PERCENTAGE", "100"))

	//All Images for running chaos test
	experimentDetails.GoExperimentImage = Getenv("EXPERIMENT_IMAGE", "litmuschaos/go-runner:ci")
	experimentDetails.OperatorImage = Getenv("OPERATOR_IMAGE", "litmuschaos/chaos-operator:ci")
	experimentDetails.RunnerImage = Getenv("RUNNER_IMAGE", "litmuschaos/chaos-runner:ci")

	// All Links for running chaos testing
	experimentDetails.RbacPath = Getenv("RBAC_PATH", "https://hub.litmuschaos.io/api/chaos/master?file=charts/generic/"+expName+"/rbac.yaml")
	experimentDetails.EnginePath = Getenv("ENGINE_PATH", "https://hub.litmuschaos.io/api/chaos/master?file=charts/generic/"+expName+"/engine.yaml")
	experimentDetails.InstallLitmus = Getenv("INSTALL_LITMUS", "https://litmuschaos.github.io/litmus/litmus-operator-latest.yaml")

}

// Getenv fetch the env and set the default value, if any
func Getenv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return value
}
