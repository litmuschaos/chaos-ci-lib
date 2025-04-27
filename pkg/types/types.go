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
	ExperimentTimeout                  int // Duration in minutes for experiment timeout
	ExperimentPollingInterval          int // Duration in seconds for polling interval

	// V3 SDK Related Fields
	InstallLitmusFlag          bool
	ConnectInfraFlag           bool
	LitmusEndpoint             string
	LitmusUsername             string
	LitmusPassword             string
	LitmusProjectID            string 
	InfraName                  string 
	InfraNamespace             string 
	InfraScope                 string 
	InfraSA                    string 
	InfraDescription           string 
	InfraPlatformName          string 
	InfraEnvironmentID         string 
	InfraNsExists              bool   
	InfraSaExists              bool   
	InfraSkipSSL               bool   
	InfraNodeSelector          string 
	InfraTolerations           string 
	ConnectedInfraID           string // Stores the ID of the infra connected via SDK
	ExperimentRunID            string // Stores the ID of the experiment run started via SDK
	
	// New infrastructure control variables
	InstallInfra              bool   // Flag to determine if infrastructure should be installed
	UseExistingInfra          bool   // Flag to determine if existing infrastructure should be used
	ExistingInfraID           string // ID of existing infrastructure if UseExistingInfra is true
}
