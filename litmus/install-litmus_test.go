package litmus

import (
	"bytes"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/litmuschaos/chaos-ci-lib/pkg"
	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/log"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"
)

var (
	out    bytes.Buffer
	stderr bytes.Buffer
)

func TestInstallLitmus(t *testing.T) {

	RegisterFailHandler(Fail)
	RunSpecs(t, "BDD test")
}

//BDD Tests to Install Litmus
var _ = Describe("BDD of Litmus installation", func() {

	// BDD TEST CASE 1
	Context("Check for the Litmus components", func() {

		It("Should check for creation of Litmus", func() {

			experimentsDetails := types.ExperimentDetails{}
			clients := environment.ClientSets{}
			//Getting kubeConfig and Generate ClientSets
			By("[PreChaos]: Getting kubeconfig and generate clientset")
			err := clients.GenerateClientSetFromKubeConfig()
			Expect(err).To(BeNil(), "Unable to Get the kubeconfig, due to {%v}", err)

			//Fetching all the default ENV
			//Note: please don't provide custom experiment name here
			By("[PreChaos]: Fetching all default ENVs")
			klog.Infof("[PreReq]: Getting the ENVs for the %v test", experimentsDetails.ExperimentName)
			environment.GetENV(&experimentsDetails, "install-litmus", "")

			//Installing Litmus
			By("Installing Litmus")
			err = pkg.InstallLitmus(&experimentsDetails)
			Expect(err).To(BeNil(), "Litmus installation failed, due to {%v}", err)

			//Checking the status of operator
			operator, _ := clients.KubeClient.AppsV1().Deployments("litmus").Get("chaos-operator-ce", metav1.GetOptions{})
			count := 0
			for operator.Status.UnavailableReplicas != 0 {
				if count < 50 {
					fmt.Printf("Unavaliable Count: %v \n", operator.Status.UnavailableReplicas)
					operator, _ = clients.KubeClient.AppsV1().Deployments("litmus").Get("chaos-operator-ce", metav1.GetOptions{})
					time.Sleep(5 * time.Second)
					count++
				} else {
					Fail("Operator is not in Ready state Time Out")
				}
			}
			log.Info("[Info]: Chaos Operator created successfully")
			log.Info("[Info]: Installing all chaos experiment from helm install")
			helmInstall := exec.Command("bash", "helm-install.sh")
			helmInstall.Stdout = &out
			helmInstall.Stderr = &stderr
			err = helmInstall.Run()
			if err != nil {
				fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
				fmt.Println(err)
				Fail("Fail to install litmus chaosexperiments through helm charts")
			}
			fmt.Println("Result: " + out.String())

		})
	})

})
