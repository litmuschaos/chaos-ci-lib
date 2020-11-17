package types

import (
	"os"

	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
	chaosClient "github.com/litmuschaos/chaos-operator/pkg/client/clientset/versioned/typed/litmuschaos/v1alpha1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	restclient "k8s.io/client-go/rest"
)

var (

	//ChaosNamespace : where the chaos will be performed
	ChaosNamespace = os.Getenv("APP_NS")
	//ApplicationLabel : Label of application pod
	ApplicationLabel = os.Getenv("APP_LABEL")
	//TotalChaosDuration : Time Duration of Chaos
	TotalChaosDuration = os.Getenv("TOTAL_CHAOS_DURATION")
	//ChaosInterval : Time Interval for Rerunning Chaos
	ChaosInterval = os.Getenv("CHAOS_INTERVAL")
	//TargetContainer : Name of target container
	TargetContainer = os.Getenv("TARGET_CONTAINER")
	//NodeCPUCore : Name of chores of CPU
	NodeCPUCore = os.Getenv("NODE_CPU_CORE")

	Kubeconfig string
	Config     *restclient.Config
	Client     *kubernetes.Clientset
	ClientSet  *chaosClient.LitmuschaosV1alpha1Client

	//rbacPath of different chaos experiments
	PodDeleteRbacPath             = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/pod-delete/rbac.yaml"
	ContainerKillRbacPath         = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/container-kill/rbac.yaml"
	DiskFillRbacPath              = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/disk-fill/rbac.yaml"
	NodeCPUHogRbacPath            = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/node-cpu-hog/rbac.yaml"
	NodeDrainRbacPath             = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/node-drain/rbac.yaml"
	NodeMemoryHogRbacPath         = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/node-memory-hog/rbac.yaml"
	PodCPUHogRbacPath             = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/pod-cpu-hog/rbac.yaml"
	PodMemoryHogRbacPath          = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/pod-memory-hog/rbac.yaml"
	PodNetworkCorruptionRbacPath  = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/pod-network-corruption/rbac.yaml"
	PodNetworkLatencyRbacPath     = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/pod-network-latency/rbac.yaml"
	PodNetworkLossRbacPath        = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/pod-network-loss/rbac.yaml"
	PodNetworkDuplicationRbacPath = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/pod-network-duplication/rbac.yaml"
	PodAutoscalerRbacPath         = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/pod-autoscaler/rbac.yaml"
	NodeIOStressRbacPath          = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/node-io-stress/rbac.yaml"

	//experimentPath of different chaosexperiments
	PodDeleteExperimentPath            = "https://hub.litmuschaos.io/api/chaos/master?file=charts/generic/pod-delete/experiment.yaml"
	ContainerKillExperimentPath        = "https://hub.litmuschaos.io/api/chaos/master?file=charts/generic/container-kill/experiment.yaml"
	DiskFillExperimentPath             = "https://hub.litmuschaos.io/api/chaos/master?file=charts/generic/disk-fill/experiment.yaml"
	NodeCPUHogExperimentPath           = "https://hub.litmuschaos.io/api/chaos/master?file=charts/generic/node-cpu-hog/experiment.yaml"
	NodeDrainExperimentPath            = "https://hub.litmuschaos.io/api/chaos/master?file=charts/generic/node-drain/experiment.yaml"
	NodeMemoryHogExperimentPath        = "https://hub.litmuschaos.io/api/chaos/master?file=charts/generic/node-memory-hog/experiment.yaml"
	PodCPUHogExperimentPath            = "https://hub.litmuschaos.io/api/chaos/master?file=charts/generic/pod-cpu-hog/experiment.yaml"
	PodMemoryHogExperimentPath         = "https://hub.litmuschaos.io/api/chaos/master?file=charts/generic/pod-memory-hog/experiment.yaml"
	PodNetworkCorruptionExperimentPath = "https://hub.litmuschaos.io/api/chaos/master?file=charts/generic/pod-network-corruption/experiment.yaml"
	PodNetworkLatencyExperimentPath    = "https://hub.litmuschaos.io/api/chaos/master?file=charts/generic/pod-network-latency/experiment.yaml"
	PodNetworkLossExperimentPath       = "https://hub.litmuschaos.io/api/chaos/master?file=charts/generic/pod-network-loss/experiment.yaml"

	//enginePath of different chaosengines
	PodDeleteEnginePath             = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/pod-delete/engine.yaml"
	ContainerKillEnginePath         = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/container-kill/engine.yaml"
	DiskFillEnginePath              = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/disk-fill/engine.yaml"
	NodeCPUHogEnginePath            = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/node-cpu-hog/engine.yaml"
	NodeDrainEnginePath             = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/node-drain/engine.yaml"
	NodeMemoryHogEnginePath         = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/node-memory-hog/engine.yaml"
	PodCPUHogEnginePath             = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/pod-cpu-hog/engine.yaml"
	PodMemoryHogEnginePath          = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/pod-memory-hog/engine.yaml"
	PodNetworkCorruptionEnginePath  = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/pod-network-corruption/engine.yaml"
	PodNetworkLatencyEnginePath     = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/pod-network-latency/engine.yaml"
	PodNetworkLossEnginePath        = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/pod-network-loss/engine.yaml"
	PodNetworkDuplicationEnginePath = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/pod-network-duplication/engine.yaml"
	PodAutoscalerEnginePath         = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/pod-autoscaler/engine.yaml"
	NodeIOStressEnginePath          = "https://raw.githubusercontent.com/litmuschaos/chaos-charts/master/charts/generic/node-io-stress/engine.yaml"

	//InstallLitmus : Path to create operator
	InstallLitmus = "https://litmuschaos.github.io/litmus/litmus-operator-latest.yaml"
	//LitmusCrd : Path to litmus crds
	LitmusCrd = "https://raw.githubusercontent.com/litmuschaos/chaos-operator/master/deploy/chaos_crds.yaml"
)

// EngineDetails struct is for collecting all the engine-related details
type EngineDetails struct {
	Name             string
	Experiments      []string
	AppLabel         string
	SvcAccount       string
	AppKind          string
	AppNamespace     string
	ClientUUID       string
	AuxiliaryAppInfo string
	UID              string
}

// ExperimentDetails is for collecting all the experiment-related details
type ExperimentDetails struct {
	Name       string
	Env        map[string]string
	ExpLabels  map[string]string
	ExpImage   string
	ExpArgs    []string
	JobName    string
	Namespace  string
	ConfigMaps []v1alpha1.ConfigMap
	Secrets    []v1alpha1.Secret
	SvcAccount string
}

// PodDetails struct is for collecting all pod details
type PodDetails struct {
	PodName      string
	PodNamespace string
	PodLabel     string
	PodKind      string
}
