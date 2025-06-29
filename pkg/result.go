package pkg

import (
	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/log"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ChaosResultVerdict checks the chaos result verdict
func ChaosResultVerdict(experimentsDetails *types.ExperimentDetails, clients environment.ClientSets) error {

	if err = WaitForChaosResultCompletion(experimentsDetails, clients); err != nil {
		return errors.Errorf("engine state check failed, err %v", err)
	}

	chaosResult, err := clients.LitmusClient.ChaosResults(experimentsDetails.ChaosNamespace).Get(experimentsDetails.EngineName+"-"+experimentsDetails.ExperimentName, metav1.GetOptions{})
	if err != nil {
		return errors.Errorf("Fail to get the chaosresult, due to %v", err)
	}
	log.Infof("Chaos Result Verdict is: %v", chaosResult.Status.ExperimentStatus.Verdict)
	if string(chaosResult.Status.ExperimentStatus.Verdict) != "Pass" {
		return errors.Errorf("Fail to get the chaosresult verdict as \"Pass\"")
	}

	return nil
}

// ChaosEngineVerdict checks the chaosengine verdict
func ChaosEngineVerdict(experimentsDetails *types.ExperimentDetails, clients environment.ClientSets) error {

	if err = WaitForEngineCompletion(experimentsDetails, clients); err != nil {
		return errors.Errorf("engine state check failed, err %v", err)
	}

	chaosEngine, err := clients.LitmusClient.ChaosEngines(experimentsDetails.ChaosNamespace).Get(experimentsDetails.EngineName, metav1.GetOptions{})
	if err != nil {
		return errors.Errorf("Fail to get the chaosengine, due to %v", err)
	}
	log.Infof("Chaos Engine Verdict is: %v", chaosEngine.Status.Experiments[0].Verdict)

	if string(chaosEngine.Status.Experiments[0].Verdict) != "Pass" {
		return errors.Errorf("Fail to get the chaosengine verdict as \"Pass\"")
	}

	return nil
}
