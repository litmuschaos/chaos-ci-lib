package litmus

import (
	"bytes"
	"fmt"
	"os/exec"
	"testing"

	"github.com/litmuschaos/chaos-ci-lib/pkg"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
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

			var (
				err         error
				out, stderr bytes.Buffer
			)
			experimentsDetails := types.ExperimentDetails{}

			//Deleting all chaosengines
			By("Deleting all chaosengine")
			err = exec.Command("kubectl", "delete", "chaosengine", "--all", "-A").Run()
			Expect(err).To(BeNil(), "Failed to delete chaosengine")
			klog.Info("All chaosengine deleted successfully")

			//Deleting all chaosexperiment
			klog.Info("Deleting all chaos experiment from helm uninstall")
			helmUninstall := exec.Command("helm", "uninstall", "k8s", "--namespace", experimentsDetails.ChaosNamespace)
			helmUninstall.Stdout = &out
			helmUninstall.Stderr = &stderr
			err = helmUninstall.Run()
			if err != nil {
				fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
				fmt.Println(err)
				Fail("Fail to uninstall litmus chaosexperiments through helm charts")
			}
			fmt.Println("Result: " + out.String())

			//Deleting all chaosengines
			By("Deleting all chaosengine")
			command := []string{"delete", "chaosengine,chaosexperiment,chaosresult", "--all", "--all-namespaces"}
			err = pkg.Kubectl(command...)
			Expect(err).To(BeNil(), "failed to delete CRs")
			klog.Info("All CRs deleted successfully")

			//Deleting crds
			By("Delete chaosengine crd")
			command = []string{"delete", "-f", "https://raw.githubusercontent.com/litmuschaos/chaos-operator/master/deploy/chaos_crds.yaml"}
			err = pkg.Kubectl(command...)
			Expect(err).To(BeNil(), "failed to delete crds")
			klog.Info("crds deleted successfully")

			//Deleting rbac
			By("Delete chaosengine rbac")
			command = []string{"delete", "-f", "https://raw.githubusercontent.com/litmuschaos/chaos-operator/master/deploy/rbac.yaml"}
			err = pkg.Kubectl(command...)
			Expect(err).To(BeNil(), "failed to create rbac")
			klog.Info("rbac deleted sucessfully")
		})
	})
})
