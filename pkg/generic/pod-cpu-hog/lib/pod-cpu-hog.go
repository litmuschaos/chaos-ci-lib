package lib

import (
	"strconv"

	common "github.com/litmuschaos/chaos-ci-lib/pkg"
	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
)

//InstallPodCPUHogEngine installs the given chaosengine for the experiment
func InstallPodCPUHogEngine(experimentsDetails *types.ExperimentDetails, chaosEngine *v1alpha1.ChaosEngine, clients environment.ClientSets) error {

	experimentENV := setPodCPUHogExperimentENV(experimentsDetails)
	if err := common.InstallChaosEngine(experimentsDetails, chaosEngine, experimentENV, clients); err != nil {
		return err
	}
	return nil
}

// setPodCPUHogExperimentENV will set the ENVs for disk fill experiment
func setPodCPUHogExperimentENV(experimentsDetails *types.ExperimentDetails) *common.ENVDetails {
	// contains all the envs
	envDetails := common.ENVDetails{
		ENV: map[string]string{},
	}
	// Add Experiment ENV's
	envDetails.SetEnv("CONTAINER_RUNTIME", experimentsDetails.ContainerRuntime).
		SetEnv("SOCKET_PATH", experimentsDetails.SocketPath).
		SetEnv("TARGET_PODS", experimentsDetails.TargetPods).
		SetEnv("PODS_AFFECTED_PERC", strconv.Itoa(experimentsDetails.PodsAffectedPerc)).
		SetEnv("CPU_CORES", strconv.Itoa(experimentsDetails.CPU))

	return &envDetails
}
