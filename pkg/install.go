package pkg

import (
	"github.com/litmuschaos/chaos-ci-lib/pkg/log"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/pkg/errors"
)

var err error

//InstallRbac installs and configure rbac for running go based chaos
func InstallRbac(experimentsDetails *types.ExperimentDetails, rbacNamespace string) error {

	//Fetch RBAC file
	err = DownloadFile("/tmp/"+experimentsDetails.ExperimentName+"-sa.yaml", experimentsDetails.RbacPath)
	if err != nil {
		return errors.Errorf("Fail to fetch the rbac file, due to %v", err)
	}
	//Modify Namespace field of the RBAC
	if rbacNamespace != "" {
		err = EditFile("/tmp/"+experimentsDetails.ExperimentName+"-sa.yaml", "namespace: default", "namespace: "+rbacNamespace)
		if err != nil {
			return errors.Errorf("Fail to Modify rbac file, due to %v", err)
		}
	}
	log.Info("[RBAC]: Installing RABC...")
	//Creating rbac
	command := []string{"apply", "-f", "/tmp/" + experimentsDetails.ExperimentName + "-sa.yaml", "-n", rbacNamespace}
	err := Kubectl(command...)
	if err != nil {
		return errors.Errorf("fail to apply rbac file, err: %v", err)
	}
	log.Info("[RBAC]: Rbac installed successfully !!!")

	return nil
}

//InstallLitmus installs the latest version of litmus
func InstallLitmus(testsDetails *types.ExperimentDetails) error {

	log.Info("Installing Litmus ...")
	if err := DownloadFile("/tmp/install-litmus.yaml", testsDetails.InstallLitmus); err != nil {
		return errors.Errorf("Fail to fetch litmus operator file, due to %v", err)
	}
	log.Info("Updating ChaosOperator Image ...")
	if err := EditFile("/tmp/install-litmus.yaml", "image: litmuschaos/chaos-operator:latest", "image: "+testsDetails.OperatorImage); err != nil {
		return errors.Errorf("Unable to update operator image, due to %v", err)

	}
	if err = EditKeyValue("/tmp/install-litmus.yaml", "  - chaos-operator", "imagePullPolicy: Always", "imagePullPolicy: "+testsDetails.ImagePullPolicy); err != nil {
		return errors.Errorf("Unable to update image pull policy, due to %v", err)
	}
	log.Info("Updating Chaos Runner Image ...")
	if err := EditKeyValue("/tmp/install-litmus.yaml", "CHAOS_RUNNER_IMAGE", "value: \"litmuschaos/chaos-runner:latest\"", "value: '"+testsDetails.RunnerImage+"'"); err != nil {
		return errors.Errorf("Unable to update runner image, due to %v", err)
	}
	//Creating engine
	command := []string{"apply", "-f", "/tmp/install-litmus.yaml"}
	err := Kubectl(command...)
	if err != nil {
		return errors.Errorf("fail to apply litmus installation file, err: %v", err)
	}
	log.Info("[Info]: Litmus installed successfully !!!")

	return nil
}
