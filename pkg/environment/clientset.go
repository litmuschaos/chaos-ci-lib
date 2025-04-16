package environment

import (
	"os"

	chaosClient "github.com/litmuschaos/chaos-operator/pkg/client/clientset/versioned/typed/litmuschaos/v1alpha1"
	litmusSDK "github.com/litmuschaos/litmus-go-sdk/pkg/sdk"
	"github.com/pkg/errors"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// // ClientSets is a collection of clientSets and kubeConfig needed
type ClientSets struct {
	KubeClient    *kubernetes.Clientset
	LitmusClient  *chaosClient.LitmuschaosV1alpha1Client
	KubeConfig    *rest.Config
	DynamicClient dynamic.Interface
	SDKClient     litmusSDK.Client
	LitmusProjectID string
}

// GenerateClientSetFromKubeConfig will generate the Litmus SDK client
func (clientSets *ClientSets) GenerateClientSetFromSDK() error {
	// Initialize Litmus SDK client
	endpoint := os.Getenv("LITMUS_ENDPOINT")
	username := os.Getenv("LITMUS_USERNAME")
	password := os.Getenv("LITMUS_PASSWORD")
	projectID := os.Getenv("LITMUS_PROJECT_ID")
	
	// Check if environment variables are set
	if endpoint == "" || username == "" || password == "" || projectID == "" {
		return errors.New("LITMUS_ENDPOINT, LITMUS_USERNAME, LITMUS_PASSWORD, and LITMUS_PROJECT_ID environment variables must be set")
	}
	
	// Initialize Litmus SDK client
	sdkClient, err := litmusSDK.NewClient(litmusSDK.ClientOptions{
		Endpoint: endpoint,
		Username: username,
		Password: password,
	})
	if err != nil {
		return errors.Wrapf(err, "Unable to create Litmus SDK client: %v", err)
	}
	
	clientSets.SDKClient = sdkClient
	clientSets.LitmusProjectID = projectID
	return nil
}


// GenerateClientSetFromKubeConfig will generation both ClientSets (k8s, and Litmus) as well as the KubeConfig
func (clientSets *ClientSets) GenerateClientSetFromKubeConfig() error {

	config, err := getKubeConfig()
	if err != nil {
		return err
	}
	k8sClientSet, err := GenerateK8sClientSet(config)
	if err != nil {
		return err
	}
	litmusClientSet, err := GenerateLitmusClientSet(config)
	if err != nil {
		return err
	}
	dynamicClientSet, err := DynamicClientSet(config)
	if err != nil {
		return err
	}

	clientSets.KubeClient = k8sClientSet
	clientSets.LitmusClient = litmusClientSet
	clientSets.KubeConfig = config
	clientSets.DynamicClient = dynamicClientSet
	return nil
}

// getKubeConfig setup the config for access cluster resource
func getKubeConfig() (*rest.Config, error) {

	KubeConfig := os.Getenv("KUBECONFIG")
	// Use in-cluster config if kubeconfig path is not specified
	if KubeConfig == "" {
		return rest.InClusterConfig()
	}
	config, err := clientcmd.BuildConfigFromFlags("", KubeConfig)
	if err != nil {
		return config, err
	}
	return config, err
}

// GenerateK8sClientSet will generation k8s client
func GenerateK8sClientSet(config *rest.Config) (*kubernetes.Clientset, error) {
	k8sClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to generate kubernetes clientSet %s: ", err)
	}
	return k8sClientSet, nil
}

// GenerateLitmusClientSet will generate a LitmusClient
func GenerateLitmusClientSet(config *rest.Config) (*chaosClient.LitmuschaosV1alpha1Client, error) {
	litmusClientSet, err := chaosClient.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to create LitmusClientSet: %v", err)
	}
	return litmusClientSet, nil
}

// DynamicClientSet will generate a DynamicClient
func DynamicClientSet(config *rest.Config) (dynamic.Interface, error) {
	dynamicClientSet, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to create DynamicClientSet: %v", err)
	}
	return dynamicClientSet, nil
}

