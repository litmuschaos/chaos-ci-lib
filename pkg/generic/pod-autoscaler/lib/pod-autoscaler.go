package lib

import (
	"strconv"
	"time"

	"github.com/litmuschaos/chaos-ci-lib/pkg"
	"github.com/litmuschaos/chaos-ci-lib/pkg/log"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/pkg/errors"
)

var err error

//InstallPodAutoscalerEngine installs the given chaosengine for the experiment
func InstallPodAutoscalerEngine(experimentsDetails *types.ExperimentDetails) error {

	if err = pkg.ModifyEngineSpec(experimentsDetails, true); err != nil {
		return errors.Errorf("Fail to Update the engine file, due to %v", err)
	}
	//Modify ENVs
	if err = pkg.EditKeyValue("/tmp/"+experimentsDetails.ExperimentName+"-ce.yaml", "TOTAL_CHAOS_DURATION", "value: '60'", "value: '"+strconv.Itoa(experimentsDetails.ChaosDuration)+"'"); err != nil {
		return errors.Errorf("Fail to Update the engine file, due to %v", err)
	}
	log.Info("[Engine]: Installing ChaosEngine...")
	//Creating engine
	command := []string{"apply", "-f", "/tmp/" + experimentsDetails.ExperimentName + "-ce.yaml", "-n", experimentsDetails.ChaosNamespace}
	err := pkg.Kubectl(command...)
	if err != nil {
		return errors.Errorf("fail to apply engine file, err: %v", err)
	}
	log.Info("[Engine]: ChaosEngine Installed Successfully !!!")
	time.Sleep(2 * time.Second)

	return nil
}
