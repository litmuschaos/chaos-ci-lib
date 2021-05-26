package pkg

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/litmuschaos/chaos-ci-lib/pkg/log"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/pkg/errors"
	"k8s.io/klog"
)

func Kubectl(command ...string) error {

	var out, stderr bytes.Buffer

	cmd := exec.Command("kubectl", command...)
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		log.Infof(fmt.Sprint(err) + ": " + stderr.String())
		log.Infof("Error: %v", err)
		return err
	}
	klog.Infof("%v", out.String())
	return nil
}

func ModifyEngineSpec(experimentsDetails *types.ExperimentDetails, appinfo bool) error {

	// Fetch Chaos Engine
	if err = DownloadFile("/tmp/"+experimentsDetails.ExperimentName+"-ce.yaml", experimentsDetails.EnginePath); err != nil {
		return errors.Errorf("Fail to fetch the engine file, due to %v", err)
	}
	// Add imagePullPolicy of chaos-runner to Always
	if err = AddAfterMatch("/tmp/"+experimentsDetails.ExperimentName+"-ce.yaml", "jobCleanUpPolicy", "  components:\n    runner:\n      imagePullPolicy: "+experimentsDetails.ImagePullPolicy); err != nil {
		log.Warnf("Fail to add a new line due to %v", err)
	}
	// Add imagePullPolicy of chaos-runner to Always
	if err = AddAfterMatch("/tmp/"+experimentsDetails.ExperimentName+"-ce.yaml", "jobCleanUpPolicy", "  annotationCheck: '"+experimentsDetails.AnnotationCheck+"'"); err != nil {
		log.Warnf("Fail to add a new line due to %v", err)
	}
	// Modify the spec of engine file
	if err = EditFile("/tmp/"+experimentsDetails.ExperimentName+"-ce.yaml", "name: nginx-chaos", "name: "+experimentsDetails.EngineName+""); err != nil {
		if err = EditFile("/tmp/"+experimentsDetails.ExperimentName+"-ce.yaml", "name: nginx-network-chaos", "name: "+experimentsDetails.EngineName+""); err != nil {
			log.Warnf("Fail to Update the engine file, due to %v", err)
		}
	}
	if err = EditFile("/tmp/"+experimentsDetails.ExperimentName+"-ce.yaml", "namespace: default", "namespace: "+experimentsDetails.ChaosNamespace+""); err != nil {
		return errors.Errorf("Fail to Update the engine file, due to %v", err)
	}
	if err = EditFile("/tmp/"+experimentsDetails.ExperimentName+"-ce.yaml", "jobCleanUpPolicy: 'delete'", "jobCleanUpPolicy: "+experimentsDetails.JobCleanUpPolicy+""); err != nil {
		log.Warnf("Fail to Update the engine file, due to %v", err)
	}
	// Modify appinfo
	if appinfo {
		if err = EditFile("/tmp/"+experimentsDetails.ExperimentName+"-ce.yaml", "appns: 'default'", "appns: "+experimentsDetails.AppNS+""); err != nil {
			return errors.Errorf("Fail to Update the engine file, due to %v", err)
		}
		if err = EditFile("/tmp/"+experimentsDetails.ExperimentName+"-ce.yaml", "applabel: 'app=nginx'", "applabel: "+experimentsDetails.AppLabel+""); err != nil {
			return errors.Errorf("Fail to Update the engine file, due to %v", err)
		}
		if err = EditFile("/tmp/"+experimentsDetails.ExperimentName+"-ce.yaml", "appkind: 'deployment'", "appkind: "+experimentsDetails.AppKind+""); err != nil {
			return errors.Errorf("Fail to Update the engine file, due to %v", err)
		}
	}
	return nil
}
