package pkg

import (
	"time"

	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/log"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/retry"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

//RunnerPodStatus will check the runner pod running state
func RunnerPodStatus(experimentsDetails *types.ExperimentDetails, runnerNamespace string, clients environment.ClientSets) (error, error) {

	//Fetching the runner pod and Checking if it gets in Running state or not
	runner, err := clients.KubeClient.CoreV1().Pods(runnerNamespace).Get(experimentsDetails.EngineName+"-runner", metav1.GetOptions{})
	if err != nil {
		return nil, errors.Errorf("Unable to get the runner pod, due to %v", err)
	}
	log.Infof("name : %v ", runner.Name)
	//Running it for infinite time (say 3000 * 10)
	//The Gitlab job will quit if it takes more time than default time (10 min)
	for i := 0; i < 300; i++ {
		if string(runner.Status.Phase) != "Running" {
			time.Sleep(1 * time.Second)
			runner, err = clients.KubeClient.CoreV1().Pods(runnerNamespace).Get(experimentsDetails.EngineName+"-runner", metav1.GetOptions{})
			if err != nil || runner.Status.Phase == "Succeeded" || runner.Status.Phase == "" {
				return nil, errors.Errorf("Fail to get the runner pod status after sleep, due to %v", err)
			}
			log.Infof("The Runner pod is in %v State ", runner.Status.Phase)
		} else {
			break
		}
	}

	if runner.Status.Phase != "Running" {
		return nil, errors.Errorf("Runner pod fail to come in running state, due to %v", err)
	}
	log.Info("Runner pod is in Running state")

	return nil, nil
}

//PodStatusCheck checks the pod running status
func PodStatusCheck(experimentsDetails *types.ExperimentDetails, clients environment.ClientSets) error {
	PodList, err := clients.KubeClient.CoreV1().Pods(experimentsDetails.AppNS).List(metav1.ListOptions{LabelSelector: experimentsDetails.AppLabel})
	if err != nil {
		return errors.Errorf("fail to get the list of pods, due to %v", err)
	}
	var flag = false
	for _, pod := range PodList.Items {
		if string(pod.Status.Phase) != "Running" {
			for count := 0; count < 20; count++ {
				PodList, err := clients.KubeClient.CoreV1().Pods(experimentsDetails.AppNS).List(metav1.ListOptions{LabelSelector: experimentsDetails.AppLabel})
				if err != nil {
					return errors.Errorf("fail to get the list of pods, due to %v", err)
				}
				for _, pod := range PodList.Items {
					if string(pod.Status.Phase) != "Running" {
						log.Infof("Currently, the experiment job pod is in %v State, Please Wait ...", pod.Status.Phase)
						time.Sleep(5 * time.Second)
					} else {
						flag = true
						break

					}
				}
				if flag == true {
					break
				}
				if count == 19 {
					return errors.Errorf("pod fails to come in running state, due to %v", err)
				}
			}
		}
	}
	log.Info("[Status]: Pod is in Running state")

	return nil
}

// ChaosPodStatus will check the creation of chaos pod
func ChaosPodStatus(experimentsDetails *types.ExperimentDetails, clients environment.ClientSets) error {

	for count := 0; count < (experimentsDetails.Duration / experimentsDetails.Delay); count++ {

		chaosEngine, err := clients.LitmusClient.ChaosEngines(experimentsDetails.ChaosNamespace).Get(experimentsDetails.EngineName, metav1.GetOptions{})
		if err != nil {
			return errors.Errorf("fail to get the chaosengine %v err: %v", experimentsDetails.EngineName, err)
		}
		if len(chaosEngine.Status.Experiments) == 0 {
			time.Sleep(time.Duration(experimentsDetails.Delay) * time.Second)
			log.Info("[Status]: Experiment initializing")
			if count == ((experimentsDetails.Duration / experimentsDetails.Delay) - 1) {
				return errors.Errorf("Experiment pod fail to initialise, due to %v", err)
			}

		} else if len(chaosEngine.Status.Experiments[0].ExpPod) == 0 {
			time.Sleep(time.Duration(experimentsDetails.Delay) * time.Second)
			if count == ((experimentsDetails.Duration / experimentsDetails.Delay) - 1) {
				return errors.Errorf("Experiment pod fails to create, due to %v", err)
			}
		} else if chaosEngine.Status.Experiments[0].Status != "Running" {
			time.Sleep(time.Duration(experimentsDetails.Delay) * time.Second)
			log.Infof("[Status]: Currently, the Chaos Pod is in %v state, Please Wait...", chaosEngine.Status.Experiments[0].Status)
			if count == ((experimentsDetails.Duration / experimentsDetails.Delay) - 1) {
				return errors.Errorf("Experiment pod fails to get in running state, due to %v", err)
			}
		} else {
			break
		}
	}
	log.Info("[Status]: Chaos pod initiated successfully")
	return nil
}

//WaitForEngineCompletion waits for engine state to get completed
func WaitForEngineCompletion(experimentsDetails *types.ExperimentDetails, clients environment.ClientSets) error {
	err := retry.
		Times(uint(experimentsDetails.Duration / experimentsDetails.Delay)).
		Wait(time.Duration(experimentsDetails.Delay) * time.Second).
		Try(func(attempt uint) error {
			chaosEngine, err := clients.LitmusClient.ChaosEngines(experimentsDetails.ChaosNamespace).Get(experimentsDetails.EngineName, metav1.GetOptions{})
			if err != nil {
				return errors.Errorf("Fail to get the chaosengine, due to %v", err)
			}

			if string(chaosEngine.Status.EngineStatus) != "completed" {
				log.Infof("Engine status is %v", chaosEngine.Status.EngineStatus)
				return errors.Errorf("Engine is not yet completed")
			}
			log.Infof("Engine status is %v", chaosEngine.Status.EngineStatus)

			return nil
		})

	return err
}

//WaitForRunnerCompletion waits for runner pod completion
func WaitForRunnerCompletion(experimentsDetails *types.ExperimentDetails, clients environment.ClientSets) error {
	err := retry.
		Times(uint(experimentsDetails.Duration / experimentsDetails.Delay)).
		Wait(time.Duration(experimentsDetails.Delay) * time.Second).
		Try(func(attempt uint) error {
			runner, err := clients.KubeClient.CoreV1().Pods(experimentsDetails.ChaosNamespace).Get(experimentsDetails.EngineName+"-runner", metav1.GetOptions{})
			if err != nil {
				return errors.Errorf("Unable to get the runner pod, due to %v", err)
			}

			if string(runner.Status.Phase) != "Succeeded" {
				log.Infof("Runner pod status is %v", runner.Status.Phase)
				return errors.Errorf("Runner pod is not yet completed")
			}
			log.Infof("Runner pod status is %v", runner.Status.Phase)

			return nil
		})

	return err
}

//WaitForChaosResultCompletion waits for chaosresult state to get completed
func WaitForChaosResultCompletion(experimentsDetails *types.ExperimentDetails, clients environment.ClientSets) error {
	err := retry.
		Times(uint(experimentsDetails.Duration / experimentsDetails.Delay)).
		Wait(time.Duration(experimentsDetails.Delay) * time.Second).
		Try(func(attempt uint) error {
			chaosResult, err := clients.LitmusClient.ChaosResults(experimentsDetails.ChaosNamespace).Get(experimentsDetails.EngineName+"-"+experimentsDetails.ExperimentName, metav1.GetOptions{})
			if err != nil {
				return errors.Errorf("Fail to get the chaosresult, due to %v", err)
			}

			if string(chaosResult.Status.ExperimentStatus.Phase) != "Completed" {
				klog.Infof("ChaosResult status is %v", chaosResult.Status.ExperimentStatus.Phase)
				return errors.Errorf("ChaosResult is not yet completed")
			}
			klog.Infof("ChaosResult status is %v", chaosResult.Status.ExperimentStatus.Phase)

			return nil
		})

	return err
}
