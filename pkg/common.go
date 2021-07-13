package pkg

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/litmuschaos/chaos-ci-lib/pkg/log"
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
