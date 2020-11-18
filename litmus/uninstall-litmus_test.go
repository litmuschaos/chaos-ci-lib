package litmus

import (
	"bytes"
	"fmt"
	"os/exec"
	"testing"

	"github.com/mayadata-io/chaos-ci-lib/pkg"
	chaosTypes "github.com/mayadata-io/chaos-ci-lib/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"
)

func TestUninstallLitmus(t *testing.T) {

	RegisterFailHandler(Fail)
	RunSpecs(t, "BDD test")
}

//BDD Tests to delete litmus
var _ = Describe("BDD of Litmus cleanup", func() {

	// BDD TEST CASE 1
	Context("Check for the Litmus components", func() {

		It("Should check for deletion of Litmus", func() {

			var err error
			var out bytes.Buffer
			var stderr bytes.Buffer
			//Deleting all chaosengines
			By("Deleting all chaosengine")
			err = exec.Command("kubectl", "delete", "chaosengine", "-n", pkg.GetEnv("APP_NS", "default"), "--all").Run()
			Expect(err).To(BeNil(), "Failed to delete chaosengine")
			klog.Info("All chaosengine deleted successfully")

			//Deleting all chaosexperiment
			klog.Info("Deleting all chaos experiment from helm uninstall")
			helmUninstall := exec.Command("helm", "uninstall", "k8s", "--namespace", pkg.GetEnv("APP_NS", "default"))
			helmUninstall.Stdout = &out
			helmUninstall.Stderr = &stderr
			err = helmUninstall.Run()
			if err != nil {
				fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
				fmt.Println(err)
				Fail("Fail to uninstall litmus chaosexperiments through helm charts")
			}
			fmt.Println("Result: " + out.String())

			//Deleting all chaosresults
			By("Deleting all chaosresults")
			err = exec.Command("kubectl", "delete", "chaosresult", "-n", pkg.GetEnv("APP_NS", "default"), "--all").Run()
			Expect(err).To(BeNil(), "Failed to delete chaosresult")
			klog.Info("All chaosresult deleted successfully")

			//Deleting crds
			By("Delete chaosengine crd")
			err = exec.Command("kubectl", "delete", "-f", chaosTypes.LitmusCrd).Run()
			Expect(err).To(BeNil(), "Failed to delete crds")
			klog.Info("Litmus crds deleted successfully")

			//Deleting litmus service account
			By("Delete Litmus service account")
			err = exec.Command("kubectl", "delete", "sa", "litmus", "-n", "litmus").Run()
			Expect(err).To(BeNil(), "Failed to delete litmus service account")
			klog.Info("Litmus service account deleted sucessfully")

			//Deleting litmus role
			By("Delete Litmus role")
			err = exec.Command("kubectl", "delete", "clusterrole", "litmus").Run()
			Expect(err).To(BeNil(), "Failed to delete litmus clusterrole")
			klog.Info("Litmus clusterrole deleted sucessfully")

			//Deleting litmus operator
			By("Delete Litmus operator")
			err = exec.Command("kubectl", "delete", "deploy", "chaos-operator-ce", "-n", "litmus").Run()
			Expect(err).To(BeNil(), "Failed to delete chaos operator")
			klog.Info("Litmus chaos operator deleted sucessfully")

			//Deleting litmus namespace
			By("Delete Litmus namespace")
			err = exec.Command("kubectl", "delete", "ns", "litmus").Run()
			Expect(err).To(BeNil(), "Failed to delete litmus namespace")
			klog.Info("Litmus namespace deleted sucessfully")

		})
	})
})
