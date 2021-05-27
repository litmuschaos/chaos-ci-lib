package pkg

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/log"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ChaosPodLogs is used to wait for the chaosPod to get completed and then prints the logs of it.
func ChaosPodLogs(experimentsDetails *types.ExperimentDetails, clients environment.ClientSets) error {

	chaosEngine, err := clients.LitmusClient.ChaosEngines(experimentsDetails.AppNS).Get(experimentsDetails.EngineName, metav1.GetOptions{})
	if err != nil {
		return errors.Errorf("fail to get the chaosengine %v err: %v", experimentsDetails.EngineName, err)
	}
	if len(chaosEngine.Status.Experiments) == 0 {
		return errors.Errorf("fail to get the chaos pod for the test")
	}
	for count := 0; count < 3000; count++ {

		chaosPod, err := clients.KubeClient.CoreV1().Pods(experimentsDetails.AppNS).Get(chaosEngine.Status.Experiments[0].ExpPod, metav1.GetOptions{})
		if err != nil {
			return errors.Errorf("fail to get the chaos pod err: %v", err)
		}
		if chaosPod.Status.Phase != "Succeeded" {
			if chaosPod.Status.Phase != "Running" && chaosPod.Status.Phase != "Pending" {
				return errors.Errorf("chaos pod is in %v state", chaosPod.Status.Phase)
			}
			time.Sleep(10 * time.Second)
			log.Infof("[Status]: Currently, the Chaos Pod is in %v State, Please Wait for its completion", chaosPod.Status.Phase)
		} else {
			break
		}
	}

	//Getting the jobList after the job gets completed
	jobPodName := (chaosEngine.Status.Experiments[0].ExpPod)
	log.Infof("JobPodName : %v \n\n\n", jobPodName)
	// After the Job gets completed further commands will print the logs.
	req := clients.KubeClient.CoreV1().Pods(experimentsDetails.AppNS).GetLogs(jobPodName, &v1.PodLogOptions{})
	readCloser, err := req.Stream()
	if err != nil {
		fmt.Println("Error2: ", err)
	} else {
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, readCloser)
		fmt.Println("Experiment logs : \n\n", buf.String())
	}

	return nil
}
