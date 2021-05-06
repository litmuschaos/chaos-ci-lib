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
	FillPercentage                     int
	MemoryConsumption                  int
	NodeCPUCore                        int
	NetworkLatency                     string
	NetworkInterface                   string
	ContainerRuntime                   string
	SocketPath                         string
	NetworkPacketDuplicationPercentage int
	FileSystemUtilizationPercentage    int
	NetworkPacketLossPercentage        int
	OperatorImage                      string
	RunnerImage                        string
	InstallLitmus                      string
}
