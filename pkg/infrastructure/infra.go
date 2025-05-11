package infrastructure

import (
	"errors"
	"os"
	"strconv"

	"github.com/litmuschaos/chaos-ci-lib/pkg/types"
	"github.com/litmuschaos/litmus-go-sdk/pkg/sdk"
	sdkTypes "github.com/litmuschaos/litmus-go-sdk/pkg/types"
	"k8s.io/klog"
)

// SetupInfrastructure handles the creation or connection to infrastructure
// It checks if infrastructure should be installed and if it's already connected
func SetupInfrastructure(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) error {
	// Check if infrastructure operations should be performed
	installInfra, _ := strconv.ParseBool(os.Getenv("INSTALL_INFRA"))
	if !installInfra {
		klog.Info("INSTALL_INFRA is set to false, skipping infrastructure setup")
		return nil
	}

	// Check if we should use existing infrastructure
	useExistingInfra, _ := strconv.ParseBool(os.Getenv("USE_EXISTING_INFRA"))
	if useExistingInfra {
		infraID := os.Getenv("EXISTING_INFRA_ID")
		if infraID == "" {
			return errors.New("USE_EXISTING_INFRA is true but EXISTING_INFRA_ID is not provided")
		}
		experimentsDetails.ConnectedInfraID = infraID
		klog.Infof("Using existing infrastructure with ID: %s", infraID)
		return nil
	}

	// If not using existing infrastructure, connect to new one
	return ConnectInfrastructure(experimentsDetails, sdkClient)
}

// ConnectInfrastructure connects to a new infrastructure via the SDK
func ConnectInfrastructure(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) error {
	klog.Infof("Attempting to connect infrastructure: %s", experimentsDetails.InfraName)
	

	// Prepare infrastructure configuration
	sdkConfig := sdkTypes.Infra{
		Namespace:      experimentsDetails.InfraNamespace,
		ServiceAccount: experimentsDetails.InfraSA,
		Mode:           experimentsDetails.InfraScope,
		Description:    experimentsDetails.InfraDescription,
		PlatformName:   experimentsDetails.InfraPlatformName,
		EnvironmentID:  experimentsDetails.InfraEnvironmentID,
		NsExists:       experimentsDetails.InfraNsExists,
		SAExists:       experimentsDetails.InfraSaExists,
		SkipSSL:        experimentsDetails.InfraSkipSSL,
		NodeSelector:   experimentsDetails.InfraNodeSelector,
		Tolerations:    experimentsDetails.InfraTolerations,
	}

	// Create infrastructure via SDK
	infraID, errSdk := sdkClient.Infrastructure().Create(experimentsDetails.InfraName, sdkConfig)
	if errSdk != nil {
		return errSdk
	}

	// Process response and extract infra ID
	if infraID == "" {
		return errors.New("infrastructure create call returned nil data")
	}

	experimentsDetails.ConnectedInfraID = infraID
	klog.Infof("Successfully connected infrastructure via SDK. Stored ID: %s", experimentsDetails.ConnectedInfraID)


	experimentsDetails.ConnectedInfraID = infraID
	klog.Infof("Successfully connected infrastructure via SDK. Stored ID: %s", experimentsDetails.ConnectedInfraID)
	
	return nil
}

// DisconnectInfrastructure disconnects from infrastructure if it was created during the test
func DisconnectInfrastructure(experimentsDetails *types.ExperimentDetails, sdkClient sdk.Client) error {
	// Don't disconnect if we're using an existing infrastructure
	useExistingInfra, _ := strconv.ParseBool(os.Getenv("USE_EXISTING_INFRA"))
	if useExistingInfra {
		klog.Info("Using existing infrastructure, skipping disconnection")
		return nil
	}

	// Check if we have an infrastructure to disconnect
	if experimentsDetails.ConnectedInfraID == "" {
		klog.Info("No connected infrastructure ID found, skipping disconnection")
		return nil
	}

	// Disconnect the infrastructure
	klog.Infof("Attempting to disconnect infrastructure with ID: %s", experimentsDetails.ConnectedInfraID)
	err := sdkClient.Infrastructure().Disconnect(experimentsDetails.ConnectedInfraID)
	if err != nil {
		return err
	}

	klog.Infof("Successfully disconnected infrastructure: %s", experimentsDetails.ConnectedInfraID)
	return nil
} 