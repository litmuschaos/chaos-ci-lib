package types

// ExperimentDetails is for collecting all the test-related details
type ExperimentDetails struct {
	ExperimentName                     string
	EngineName                         string
	OperatorName                       string
	ChaosNamespace                     string
	ChaosInterval                      int
	RbacPath                           string
	EnginePath                         string
	AppNS                              string
	AppLabel                           string
	AppKind                            string
	JobCleanUpPolicy                   string
	AnnotationCheck                    string
	ApplicationNodeName                string
	GoExperimentImage                  string
	ImagePullPolicy                    string
	ChaosDuration                      int
	ChaosServiceAccount                string
	Force                              string
	CPU                                int
	CpuInjectCommand                   string
	NodeSelectorName                   string
	Delay                              int
	Duration                           int
	TargetContainer                    string
	DiskFillPercentage                 int
	FillPercentage                     int
	MemoryConsumption                  int
	NodeCPUCore                        int
	NetworkLatency                     string
	NetworkInterface                   string
	ContainerRuntime                   string
	ContainerPath                      string
	SocketPath                         string
	NetworkPacketDuplicationPercentage int
	NetworkPacketCorruptionPercentage  int
	FileSystemUtilizationPercentage    int
	FilesystemUtilizationBytes         int
	NetworkPacketLossPercentage        int
	OperatorImage                      string
	RunnerImage                        string
	InstallLitmus                      string
	MemoryConsumptionPercentage        int
	TargetPods                         string
	PodsAffectedPerc                   int
	NodesAffectedPerc                  int
	Replicas                           int
	ExperimentTimeout                  int
	ExperimentPollingInterval          int

	// V3 SDK Related Fields
	InstallLitmusFlag  bool
	ConnectInfraFlag   bool
	LitmusEndpoint     string
	LitmusUsername     string
	LitmusPassword     string
	LitmusProjectID    string
	InfraName          string
	InfraNamespace     string
	InfraScope         string
	InfraSA            string
	InfraDescription   string
	InfraPlatformName  string
	InfraEnvironmentID string
	InfraNsExists      bool
	InfraSaExists      bool
	InfraSkipSSL       bool
	InfraNodeSelector  string
	InfraTolerations   string
	ConnectedInfraID   string // Stores the ID of the infra connected via SDK
	InfraManifest      string // Stores the manifest returned by registerInfra
	ExperimentRunID    string // Stores the ID of the experiment run started via SDK

	// New infrastructure control variables
	InstallInfra     bool   // Flag to determine if infrastructure should be installed
	UseExistingInfra bool   // Flag to determine if existing infrastructure should be used
	ExistingInfraID  string // ID of existing infrastructure if UseExistingInfra is true

	// Infrastructure activation control
	ActivateInfra          bool // Flag to determine if infrastructure should be activated
	InfraActivationTimeout int  // Timeout in minutes for infrastructure activation

	// Probe configuration
	CreateProbe       bool   // Flag to determine if a new probe should be created
	ProbeType         string // Type of probe (http, cmd, etc.)
	ProbeName         string // Name of the probe
	ProbeMode         string // Mode of the probe (SOT, EOT, Edge, Continuous, etc.)
	ProbeURL          string // URL for HTTP probe
	ProbeTimeout      string // Timeout for probe
	ProbeInterval     string // Interval for probe
	ProbeAttempts     int    // Number of attempts for probe
	ProbeResponseCode string // Expected HTTP response code for HTTP probe
	CreatedProbeID    string // ID of the created probe
}
