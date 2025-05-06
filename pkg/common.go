package pkg

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/google/uuid"
	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/log"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// GetUID will return the uid from chaosengine
func GetUID(engineName, namespace string, clients environment.ClientSets) (string, error) {

	chaosEngine, err := clients.LitmusClient.ChaosEngines(namespace).Get(engineName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Errorf("fail to get the chaosengine %v err: %v", engineName, err)
	}
	return string(chaosEngine.UID), nil
}

// ENVDetails contains the ENV details
type ENVDetails struct {
	ENV map[string]string
}

// SetEnv sets the env inside envDetails struct
func (envDetails *ENVDetails) SetEnv(key, value string) *ENVDetails {
	if value != "" {
		envDetails.ENV[key] = value
	}
	return envDetails
}

// GenerateUniqueExperimentName creates a unique experiment name by combining
// a base name with a unique identifier. This ensures there are no naming
// conflicts between experiment runs
func GenerateUniqueExperimentName(baseName string) string {
	// Generate a short UUID (first 8 characters)
	uniqueID := uuid.New().String()[:8]
	return fmt.Sprintf("%s-%s", baseName, uniqueID)
}

// GenerateExperimentIDs creates all necessary IDs for a chaos experiment
// including workflow name, experiment ID, and other related identifiers.
func GenerateExperimentID() (string) {
	// Generate a unique ID for the experiment
	experimentID := uuid.New().String()[:8]
	return experimentID
}