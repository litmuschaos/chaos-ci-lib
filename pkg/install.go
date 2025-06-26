package pkg

import (
	"io"

	yamlChe "github.com/ghodss/yaml"
	"github.com/litmuschaos/chaos-ci-lib/pkg/environment"
	"github.com/litmuschaos/chaos-ci-lib/pkg/log"
	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"bytes"
	"encoding/json"
	"net/http"
	"strconv"

	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

var err error

// CreateChaosResource creates litmus components with given inputs
func CreateChaosResource(fileData []byte, namespace string, clients environment.ClientSets) error {

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(fileData), 100)

	// for loop to install all the resouces
	for {
		//runtime defines conversions between generic types and structs to map query strings to struct objects.
		var rawObj runtime.RawExtension
		if err = decoder.Decode(&rawObj); err != nil {
			// if the object is null, successfully installed all manifest
			if rawObj.Raw == nil {
				return nil
			}
			return err
		}

		// NewDecodingSerializer adds YAML decoding support to a serializer that supports JSON.
		obj, gvk, _ := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return err
		}
		unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

		// GetAPIGroupResources uses the provided discovery client to gather
		// discovery information and populate a slice of APIGroupResources.
		gr, err := restmapper.GetAPIGroupResources(clients.KubeClient.DiscoveryClient)
		if err != nil {
			return err
		}

		mapper := restmapper.NewDiscoveryRESTMapper(gr)

		// RESTMapping returns a struct representing the resource path and conversion interfaces a
		// RESTClient should use to operate on the provided group/kind in order of versions.
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		//ResourceInterface is an API interface to a specific resource under a dynamic client
		var dri dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			unstructuredObj.SetNamespace(namespace)
			dri = clients.DynamicClient.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dri = clients.DynamicClient.Resource(mapping.Resource)
		}

		// Create Chaos Resource using dynamic resource interface
		if _, err := dri.Create(unstructuredObj, v1.CreateOptions{}); err != nil {
			if !k8serrors.IsAlreadyExists(err) {
				return err
			} else {
				// Updating present resource
				log.Infof("[Status]: Updating %v", unstructuredObj.GetKind())
				_, _ = dri.Update(unstructuredObj, v1.UpdateOptions{})
			}
		}
	}
}

// InstallGoRbac installs and configure rbac for running go based chaos
func InstallRbac(experimentsDetails *types.ExperimentDetails, rbacNamespace string) error {

	//Fetch RBAC file
	err = DownloadFile("/tmp/"+experimentsDetails.ExperimentName+"-sa.yaml", experimentsDetails.RbacPath)
	if err != nil {
		return errors.Errorf("Fail to fetch the rbac file, due to %v", err)
	}
	//Modify Namespace field of the RBAC
	if rbacNamespace != "" {
		err = EditFile("/tmp/"+experimentsDetails.ExperimentName+"-sa.yaml", "namespace: default", "namespace: "+rbacNamespace)
		if err != nil {
			return errors.Errorf("Fail to Modify rbac file, due to %v", err)
		}
	}
	log.Info("[RBAC]: Installing RABC...")
	//Creating rbac
	command := []string{"apply", "-f", "/tmp/" + experimentsDetails.ExperimentName + "-sa.yaml", "-n", rbacNamespace}
	err := Kubectl(command...)
	if err != nil {
		return errors.Errorf("fail to apply rbac file, err: %v", err)
	}
	log.Info("[RBAC]: Rbac installed successfully !!!")

	return nil
}

// InstallChaosEngine installs the given go based chaos engine
func InstallChaosEngine(experimentsDetails *types.ExperimentDetails, chaosEngine *v1alpha1.ChaosEngine, experimentENVs *ENVDetails, clients environment.ClientSets) error {

	// contains all the envs
	envDetails := ENVDetails{
		ENV: map[string]string{},
	}

	//Fetch Engine file
	res, err := http.Get(experimentsDetails.EnginePath)
	if err != nil {
		return errors.Errorf("Fail to fetch the engine file, due to %v", err)
	}

	// ReadAll reads from response until an error or EOF and returns the data it read.
	fileInput, err := io.ReadAll(res.Body)
	if err != nil {
		log.Errorf("Fail to read data from response: %v", err)
	}

	// Unmarshal decodes the fileInput into chaosEngine
	err = yamlChe.Unmarshal([]byte(fileInput), &chaosEngine)
	if err != nil {
		log.Errorf("error when unmarshalling: %v", err)
	}

	// Add JobCleanUpPolicy of chaos-runner to retain
	chaosEngine.Spec.JobCleanUpPolicy = v1alpha1.CleanUpPolicy(experimentsDetails.JobCleanUpPolicy)

	// Add ImagePullPolicy of chaos-runner to Always
	chaosEngine.Spec.Components.Runner.ImagePullPolicy = corev1.PullPolicy(experimentsDetails.ImagePullPolicy)

	// Modify the spec of engine file
	chaosEngine.Name = experimentsDetails.EngineName
	chaosEngine.Namespace = experimentsDetails.ChaosNamespace

	// If ChaosEngine contain App Info then update it
	if chaosEngine.Spec.Appinfo.Appns != "" && chaosEngine.Spec.Appinfo.Applabel != "" {
		chaosEngine.Spec.Appinfo.Appns = experimentsDetails.AppNS
		chaosEngine.Spec.Appinfo.Applabel = experimentsDetails.AppLabel
	}
	chaosEngine.Spec.Appinfo.AppKind = experimentsDetails.AppKind
	chaosEngine.Spec.ChaosServiceAccount = experimentsDetails.ChaosServiceAccount
	chaosEngine.Spec.Experiments[0].Name = experimentsDetails.ExperimentName
	chaosEngine.Spec.AnnotationCheck = experimentsDetails.AnnotationCheck

	// Add common ENV's
	envDetails.SetEnv("TOTAL_CHAOS_DURATION", strconv.Itoa(experimentsDetails.ChaosDuration)).
		SetEnv("CHAOS_INTERVAL", strconv.Itoa(experimentsDetails.ChaosInterval))

	// Add experiment specific ENV's
	for key, value := range experimentENVs.ENV {
		envDetails.SetEnv(key, value)
	}

	// update App Node Details
	if experimentsDetails.ApplicationNodeName != "" {
		envDetails.SetEnv("TARGET_NODE", experimentsDetails.ApplicationNodeName)
		if chaosEngine.Spec.Experiments[0].Spec.Components.NodeSelector == nil {
			chaosEngine.Spec.Experiments[0].Spec.Components.NodeSelector = map[string]string{}
		}
		chaosEngine.Spec.Experiments[0].Spec.Components.NodeSelector["kubernetes.io/hostname"] = experimentsDetails.NodeSelectorName
	}

	// update all the value corresponding to keys from the ENV's in Engine
	for key, value := range chaosEngine.Spec.Experiments[0].Spec.Components.ENV {
		_, ok := envDetails.ENV[value.Name]
		if ok {
			chaosEngine.Spec.Experiments[0].Spec.Components.ENV[key].Value = envDetails.ENV[value.Name]
		}
	}

	// Marshal serializes the values provided into a YAML document.
	fileData, err := json.Marshal(chaosEngine)
	if err != nil {
		return errors.Errorf("Fail to marshal ChaosEngine %v", err)
	}

	//Creating chaos engine
	log.Info("[Engine]: Installing ChaosEngine...")
	if err = CreateChaosResource(fileData, experimentsDetails.ChaosNamespace, clients); err != nil {
		return errors.Errorf("fail to apply engine file, err: %v", err)
	}
	log.Info("[Engine]: ChaosEngine Installed Successfully !!!")
	return nil
}

// InstallLitmus installs the latest version of litmus
func InstallLitmus(testsDetails *types.ExperimentDetails) error {

	log.Info("Installing Litmus ...")
	if err := DownloadFile("/tmp/install-litmus.yaml", testsDetails.InstallLitmus); err != nil {
		return errors.Errorf("Fail to fetch litmus operator file, due to %v", err)
	}
	log.Info("Updating ChaosOperator Image ...")
	if err := EditFile("/tmp/install-litmus.yaml", "image: litmuschaos/chaos-operator:latest", "image: "+testsDetails.OperatorImage); err != nil {
		return errors.Errorf("Unable to update operator image, due to %v", err)

	}
	if err = EditKeyValue("/tmp/install-litmus.yaml", "  - chaos-operator", "imagePullPolicy: Always", "imagePullPolicy: "+testsDetails.ImagePullPolicy); err != nil {
		return errors.Errorf("Unable to update image pull policy, due to %v", err)
	}
	log.Info("Updating Chaos Runner Image ...")
	if err := EditKeyValue("/tmp/install-litmus.yaml", "CHAOS_RUNNER_IMAGE", "value: \"litmuschaos/chaos-runner:latest\"", "value: '"+testsDetails.RunnerImage+"'"); err != nil {
		return errors.Errorf("Unable to update runner image, due to %v", err)
	}
	//Creating engine
	command := []string{"apply", "-f", "/tmp/install-litmus.yaml"}
	err := Kubectl(command...)
	if err != nil {
		return errors.Errorf("fail to apply litmus installation file, err: %v", err)
	}
	log.Info("[Info]: Litmus installed successfully !!!")

	return nil
}
