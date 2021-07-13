package lib

import (
	"strconv"

	common "github.com/litmuschaos/chaos-ci-lib/pkg"
	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
)

//InstallNodeMemoryHogEngine installs the given chaosengine for the experiment
func InstallNodeMemoryHogEngine(experimentsDetails *types.ExperimentDetails, chaosEngine *v1alpha1.ChaosEngine, clients environment.ClientSets) error {

	experimentENV := setNodeMemoryHogExperimentENV(experimentsDetails)
	if err := common.InstallChaosEngine(experimentsDetails, chaosEngine, experimentENV, clients); err != nil {
		return err
	}
	return nil
}

// setDiskFillExperimentENV will set the ENVs for disk fill experiment
func setNodeMemoryHogExperimentENV(experimentsDetails *types.ExperimentDetails) *common.ENVDetails {
	// contains all the envs
	envDetails := common.ENVDetails{
		ENV: map[string]string{},
	}
	// Add Experiment ENV's
	envDetails.SetEnv("MEMORY_CONSUMPTION_PERCENTAGE", strconv.Itoa(experimentsDetails.MemoryConsumptionPercentage)).
		SetEnv("NODES_AFFECTED_PERC", strconv.Itoa(experimentsDetails.PodsAffectedPerc))

	return &envDetails
}
