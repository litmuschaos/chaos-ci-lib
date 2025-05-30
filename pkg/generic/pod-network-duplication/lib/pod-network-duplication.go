package lib

import (
	"strconv"

	common "github.com/litmuschaos/chaos-ci-lib/pkg"
	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
)

// InstallPodNetworkDuplicationEngine installs the given chaosengine for the experiment
func InstallPodNetworkDuplicationEngine(experimentsDetails *types.ExperimentDetails, chaosEngine *v1alpha1.ChaosEngine, clients environment.ClientSets) error {

	experimentENV := setPodNetworkDuplicationExperimentENV(experimentsDetails)
	if err := common.InstallChaosEngine(experimentsDetails, chaosEngine, experimentENV, clients); err != nil {
		return err
	}
	return nil
}

// setPodNetworkDuplicationExperimentENV will set the ENVs for pod-network-duplication experiment
func setPodNetworkDuplicationExperimentENV(experimentsDetails *types.ExperimentDetails) *common.ENVDetails {
	// contains all the envs
	envDetails := common.ENVDetails{
		ENV: map[string]string{},
	}
	// Add Experiment ENV's
	envDetails.SetEnv("CONTAINER_RUNTIME", experimentsDetails.ContainerRuntime).
		SetEnv("SOCKET_PATH", experimentsDetails.SocketPath).
		SetEnv("TARGET_PODS", experimentsDetails.TargetPods).
		SetEnv("PODS_AFFECTED_PERC", strconv.Itoa(experimentsDetails.PodsAffectedPerc)).
		SetEnv("NETWORK_INTERFACE", experimentsDetails.NetworkInterface).
		SetEnv("NETWORK_PACKET_DUPLICATION_PERCENTAGE", strconv.Itoa(experimentsDetails.NetworkPacketDuplicationPercentage))

	return &envDetails
}
