package pkg

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/log"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ChaosPodLogs will print the experiment and helper pod logs
func ChaosPodLogs(experimentsDetails *types.ExperimentDetails, clients environment.ClientSets) error {

	if err := printChaosPodLogs(experimentsDetails, clients); err != nil {
		return err
	}
	uid, err := GetUID(experimentsDetails.EngineName, experimentsDetails.ChaosNamespace, clients)
	if err != nil {
		return errors.Errorf("fail to get uid from experiment label,err: %v", err)
	}
	if err = printHelperPodLogs(experimentsDetails.ExperimentName, experimentsDetails.ChaosNamespace, uid, experimentsDetails.JobCleanUpPolicy, clients); err != nil {
		return errors.Errorf("fail to get the helper pod, err: %v", err)
	}
	return nil
}

// printChaosPodLogs will wait for the chaosPod to get completed and then prints the logs of it.
func printChaosPodLogs(experimentsDetails *types.ExperimentDetails, clients environment.ClientSets) error {

	chaosEngine, err := clients.LitmusClient.ChaosEngines(experimentsDetails.ChaosNamespace).Get(experimentsDetails.EngineName, metav1.GetOptions{})
	if err != nil {
		chaosEngine, err = clients.LitmusClient.ChaosEngines(experimentsDetails.AppNS).Get(experimentsDetails.EngineName, metav1.GetOptions{})
		if err != nil {
			return errors.Errorf("fail to get the chaosengine %v err: %v", experimentsDetails.EngineName, err)
		}
	}

	if len(chaosEngine.Status.Experiments) == 0 {
		return errors.Errorf("fail to get the chaos pod for the test")
	}
	for count := 0; count < 3000; count++ {

		chaosPod, err := clients.KubeClient.CoreV1().Pods(experimentsDetails.ChaosNamespace).Get(chaosEngine.Status.Experiments[0].ExpPod, metav1.GetOptions{})
		if err != nil {
			chaosPod, err = clients.KubeClient.CoreV1().Pods(experimentsDetails.AppNS).Get(chaosEngine.Status.Experiments[0].ExpPod, metav1.GetOptions{})
			if err != nil {
				return errors.Errorf("fail to get the chaos pod err: %v", err)
			}
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
	chaosPodName := (chaosEngine.Status.Experiments[0].ExpPod)
	log.Infof("[Info]: chaos pod name is: %v ", chaosPodName)
	if err = getPodLogs(chaosPodName, experimentsDetails.ChaosNamespace, clients); err != nil {
		return err
	}

	return nil
}

// printHelperPodLogs will print the helper pod logs when the experiment is not passed
func printHelperPodLogs(experimentName, namespace, UID, jobCleanupPolicy string, clients environment.ClientSets) error {

	podList, err := clients.KubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil || len(podList.Items) == 0 {
		return errors.Errorf("fail to get the pods in chaos ns, err:%v", err)
	}
	for _, pod := range podList.Items {
		if strings.Contains(pod.Name, experimentName+"-helper") && pod.Labels["chaosUID"] == UID && jobCleanupPolicy != "delete" {
			if err = getPodLogs(pod.Name, namespace, clients); err != nil {
				log.Errorf("fail to get the logs of helper pod %v, err: %v", pod.Name, err)
			}
		}
	}
	return nil
}

// getPodLogs will print the logs of the given pod
func getPodLogs(podName, namespace string, clients environment.ClientSets) error {

	req := clients.KubeClient.CoreV1().Pods(namespace).GetLogs(podName, &v1.PodLogOptions{})
	readCloser, err := req.Stream()
	if err != nil {
		return errors.Errorf("fail to print the logs of %v pod, err: %v", podName, err)
	}
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, readCloser)
	if err != nil {
		return errors.Errorf("fail to read the logs, err: %v", err)
	}
	fmt.Println("\n"+podName+" logs : \n\n", buf.String())
	return nil
}
