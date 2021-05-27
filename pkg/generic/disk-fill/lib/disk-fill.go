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

//InstallDiskFillEngine installs the given chaosengine for the experiment
func InstallDiskFillEngine(experimentsDetails *types.ExperimentDetails) error {

	if err = pkg.ModifyEngineSpec(experimentsDetails, true); err != nil {
		return errors.Errorf("Fail to Update the engine file, due to %v", err)
	}
	//Modify ENVs
	if err = pkg.EditKeyValue("/tmp/"+experimentsDetails.ExperimentName+"-ce.yaml", "FILL_PERCENTAGE", "value: '80'", "value: '"+strconv.Itoa(experimentsDetails.FillPercentage)+"'"); err != nil {
		return errors.Errorf("Fail to Update the engine file, due to %v", err)
	}
	if err = pkg.EditKeyValue("/tmp/"+experimentsDetails.ExperimentName+"-ce.yaml", "TARGET_CONTAINER", "value: 'nginx'", "value: '"+experimentsDetails.TargetContainer+"'"); err != nil {
		log.Warnf("Fail to Update the engine file, due to %v", err)
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
