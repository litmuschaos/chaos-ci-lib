package lib

import (
	"strconv"

	common "github.com/litmuschaos/chaos-ci-lib/pkg"
	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
)

//InstallPodAutoscalerEngine installs the given chaosengine for the experiment
func InstallPodAutoscalerEngine(experimentsDetails *types.ExperimentDetails, chaosEngine *v1alpha1.ChaosEngine, clients environment.ClientSets) error {

	experimentENV := setPodAutoscalerExperimentENV(experimentsDetails)
	if err := common.InstallChaosEngine(experimentsDetails, chaosEngine, experimentENV, clients); err != nil {
		return err
	}
	return nil
}

// setPodAutoscalerExperimentENV will set the ENVs for disk fill experiment
func setPodAutoscalerExperimentENV(experimentsDetails *types.ExperimentDetails) *common.ENVDetails {
	// contains all the envs
	envDetails := common.ENVDetails{
		ENV: map[string]string{},
	}
	// Add Experiment ENV's
	envDetails.SetEnv("REPLICA_COUNT", strconv.Itoa(experimentsDetails.Replicas))
	return &envDetails
}
