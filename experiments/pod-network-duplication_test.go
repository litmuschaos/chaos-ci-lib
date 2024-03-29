package experiments

import (
	"testing"

	"github.com/litmuschaos/chaos-ci-lib/pkg"
	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	engine "github.com/litmuschaos/chaos-ci-lib/pkg/generic/pod-network-duplication/lib"
	"github.com/litmuschaos/chaos-ci-lib/pkg/log"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func TestPodNetworkDuplication(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BDD test")
}

//BDD for running pod-network-duplication experiment
var _ = Describe("BDD of running pod-network-duplication experiment", func() {

	Context("Check for pod-network-duplication experiment", func() {

		It("Should check for the pod delete experiment", func() {

			experimentsDetails := types.ExperimentDetails{}
			clients := environment.ClientSets{}
			chaosEngine := v1alpha1.ChaosEngine{}

			//Getting kubeConfig and Generate ClientSets
			By("[PreChaos]: Getting kubeconfig and generate clientset")
			err := clients.GenerateClientSetFromKubeConfig()
			Expect(err).To(BeNil(), "Unable to Get the kubeconfig, due to {%v}", err)

			//Fetching all the default ENV
			By("[PreChaos]: Fetching all default ENVs")
			log.Infof("[PreReq]: Getting the ENVs for the %v experiment", experimentsDetails.ExperimentName)
			environment.GetENV(&experimentsDetails, "pod-network-duplication", "pod-network-duplication-engine")

			// Install RBAC for experiment Execution
			By("[Prepare]: Prepare and install RBAC")
			err = pkg.InstallRbac(&experimentsDetails, experimentsDetails.ChaosNamespace)
			Expect(err).To(BeNil(), "fail to install rbac for the experiment, due to {%v}", err)

			// Install ChaosEngine for experiment Execution
			By("[Prepare]: Prepare and install ChaosEngine")
			err = engine.InstallPodNetworkDuplicationEngine(&experimentsDetails, &chaosEngine, clients)
			Expect(err).To(BeNil(), "fail to install chaosengine, due to {%v}", err)

			//Checking runner pod running state
			By("[Status]: Runner pod running status check")
			err = pkg.RunnerPodStatus(&experimentsDetails, chaosEngine.Namespace, clients)
			if err != nil && chaosEngine.Namespace != experimentsDetails.AppNS {
				err = pkg.RunnerPodStatus(&experimentsDetails, experimentsDetails.AppNS, clients)
			}
			Expect(err).To(BeNil(), "Runner pod status check failed, due to {%v}", err)

			//Chaos pod running status check
			err = pkg.ChaosPodStatus(&experimentsDetails, clients)
			Expect(err).To(BeNil(), "Chaos pod status check failed, due to {%v}", err)

			//Waiting for chaos pod to get completed
			//And Print the logs of the chaos pod
			By("[Status]: Wait for chaos pod completion and then print logs")
			err = pkg.ChaosPodLogs(&experimentsDetails, clients)
			Expect(err).To(BeNil(), "Fail to get the experiment chaos pod logs, due to {%v}", err)

			//Checking the chaosresult verdict
			By("[Verdict]: Checking the chaosresult verdict")
			err = pkg.ChaosResultVerdict(&experimentsDetails, clients)
			Expect(err).To(BeNil(), "ChasoResult Verdict check failed, due to {%v}", err)

			//Checking chaosengine verdict
			By("Checking the Verdict of Chaos Engine")
			err = pkg.ChaosEngineVerdict(&experimentsDetails, clients)
			Expect(err).To(BeNil(), "ChaosEngine Verdict check failed, due to {%v}", err)

		})
	})
})
