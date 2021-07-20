package lib

import (
	"strconv"

	common "github.com/litmuschaos/chaos-ci-lib/pkg"
	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
)

//InstallDiskFillEngine installs the given chaosengine for the experiment
func InstallDiskFillEngine(experimentsDetails *types.ExperimentDetails, chaosEngine *v1alpha1.ChaosEngine, clients environment.ClientSets) error {

	experimentENV := setDiskFillExperimentENV(experimentsDetails)
	if err := common.InstallChaosEngine(experimentsDetails, chaosEngine, experimentENV, clients); err != nil {
		return err
	}
	return nil
}

// setDiskFillExperimentENV will set the ENVs for disk fill experiment
func setDiskFillExperimentENV(experimentsDetails *types.ExperimentDetails) *common.ENVDetails {
	// contains all the envs
	envDetails := common.ENVDetails{
		ENV: map[string]string{},
	}
	// Add Experiment ENV's
	envDetails.SetEnv("FILL_PERCENTAGE", strconv.Itoa(experimentsDetails.DiskFillPercentage)).
		SetEnv("TARGET_CONTAINER", experimentsDetails.TargetContainer).
		SetEnv("TARGET_PODS", experimentsDetails.TargetPods).
		SetEnv("CONTAINER_PATH", experimentsDetails.ContainerPath).
		SetEnv("PODS_AFFECTED_PERC", strconv.Itoa(experimentsDetails.PodsAffectedPerc))

	return &envDetails
}
