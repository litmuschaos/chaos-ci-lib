package litmus

import (
	"bytes"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
	"github.com/mayadata-io/chaos-ci-lib/pkg"
	chaosTypes "github.com/mayadata-io/chaos-ci-lib/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	scheme "k8s.io/client-go/kubernetes/scheme"
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

			//Installing Litmus
			By("Installing Litmus")
			var err error
			//Prerequisite of the test
			config, err := pkg.GetKubeConfig()
			Expect(err).To(BeNil(), "Failed to get kubeconfig client")
			client, err := kubernetes.NewForConfig(config)
			Expect(err).To(BeNil(), "failed to get client")
			err = v1alpha1.AddToScheme(scheme.Scheme)
			if err != nil {
				fmt.Println(err)
			}
			klog.Info("Installing Litmus")
			err = pkg.DownloadFile("install-litmus.yaml", chaosTypes.InstallLitmus)
			Expect(err).To(BeNil(), "fail to fetch operator yaml file to install litmus")
			klog.Info("Updating Operator Image")
			err = pkg.EditFile("install-litmus.yaml", "image: litmuschaos/chaos-operator:latest", "image: "+pkg.GetEnv("OPERATOR_IMAGE", "litmuschaos/chaos-operator:latest"))
			Expect(err).To(BeNil(), "Failed to update the operator image")
			klog.Info("Updating Runner Image")
			err = pkg.EditKeyValue("install-litmus.yaml", "CHAOS_RUNNER_IMAGE", "value: \"litmuschaos/chaos-runner:latest\"", "value: '"+pkg.GetEnv("RUNNER_IMAGE", "litmuschaos/chaos-runner:latest")+"'")
			Expect(err).To(BeNil(), "Failed to update chaos interval")
			cmd := exec.Command("kubectl", "apply", "-f", "install-litmus.yaml")
			cmd.Stdout = &out
			cmd.Stderr = &stderr
			err = cmd.Run()
			if err != nil {
				fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
				fmt.Println(err)
				Fail("Fail to install litmus")
			}
			fmt.Println("Result: " + out.String())

			//Checking the status of operator
			operator, _ := client.AppsV1().Deployments(pkg.GetEnv("APP_NS", "default")).Get("chaos-operator-ce", metav1.GetOptions{})
			count := 0
			for operator.Status.UnavailableReplicas != 0 {
				if count < 50 {
					fmt.Printf("Unavaliable Count: %v \n", operator.Status.UnavailableReplicas)
					operator, _ = client.AppsV1().Deployments(pkg.GetEnv("APP_NS", "default")).Get("chaos-operator-ce", metav1.GetOptions{})
					time.Sleep(5 * time.Second)
					count++
				} else {
					Fail("Operator is not in Ready state Time Out")
				}
			}
			klog.Info("Chaos Operator created successfully")
			klog.Info("Litmus installed successfully")
			klog.Info("Installing all chaos experiment from helm install")
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
