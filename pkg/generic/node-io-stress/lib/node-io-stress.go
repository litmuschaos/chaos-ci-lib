package lib

import (
	"strconv"

	common "github.com/litmuschaos/chaos-ci-lib/pkg"
	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
)

//InstallNodeIOStressEngine installs the given chaosengine for the experiment
func InstallNodeIOStressEngine(experimentsDetails *types.ExperimentDetails, chaosEngine *v1alpha1.ChaosEngine, clients environment.ClientSets) error {

	experimentENV := setNodeIOStressExperimentENV(experimentsDetails)
	if err := common.InstallChaosEngine(experimentsDetails, chaosEngine, experimentENV, clients); err != nil {
		return err
	}
	return nil
}

// setNodeIOStressExperimentENV will set the ENVs for disk fill experiment
func setNodeIOStressExperimentENV(experimentsDetails *types.ExperimentDetails) *common.ENVDetails {
	// contains all the envs
	envDetails := common.ENVDetails{
		ENV: map[string]string{},
	}
	// Add Experiment ENV's
	envDetails.SetEnv("FILESYSTEM_UTILIZATION_PERCENTAGE", strconv.Itoa(experimentsDetails.NodeCPUCore)).
		SetEnv("FILESYSTEM_UTILIZATION_BYTES", strconv.Itoa(experimentsDetails.FilesystemUtilizationBytes))

	return &envDetails
}
