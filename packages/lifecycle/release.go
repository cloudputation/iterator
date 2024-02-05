package lifecycle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudputation/iterator/packages/config"
	"github.com/cloudputation/iterator/packages/consul"
	log "github.com/cloudputation/iterator/packages/logger"
	"github.com/cloudputation/iterator/packages/terraform"
)

type Alert struct {
	Fingerprint        string `json:"fingerprint"`
	Module             string `json:"module"`
	TerraformScheduling string `json:"terraform_scheduling"`
}

func HandleSawtoothScheduling(cfg *config.InitConfig, alertName string) error {
	var alertData []byte
	var err error
	var alertPath = fmt.Sprintf("%s/process/alerts/%s", config.ConsulFactoryDataDir, alertName)

	switch {
	case config.ConsulStorageEnabled:
		alertData, err = consul.ConsulStoreGet(alertPath)
		if err != nil {
			return fmt.Errorf("failed to retrieve alert data for %s from Consul: %v", alertName, err)
		}
	default:
		dataDir := cfg.Server.DataDir
		executorDir := fmt.Sprintf("%s/process/alerts", dataDir)
		filePath := filepath.Join(executorDir, fmt.Sprintf("%s.json", alertName))
		alertData, err = os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to retrieve alert data for %s from file system: %v", alertName, err)
		}
	}

	var alert Alert
	err = json.Unmarshal(alertData, &alert)
	if err != nil {
		return fmt.Errorf("failed to unmarshal alert data: %v", err)
	}

	if alert.TerraformScheduling == "sawtooth" {
		terraformDriver := cfg.Server.TerraformDriver

		log.Info("Sawtooth scheduling detected for alert: %s. Triggering Terraform destroy for module: %s", alertName, alert.Module)
		err := terraform.RunTerraform(terraformDriver, alert.Module, "destroy")
		if err != nil {
			return fmt.Errorf("failed to destroy terraform resource for alert: %s on module: %s: %v", alertName, alert.Module, err)
		}
		log.Info("Terraform destroy successful for alert: %s on module: %s", alertName, alert.Module)

		err = consul.ConsulStoreDelete(alertPath)
		if err != nil {
			return fmt.Errorf("failed to delete alert data for %s from Consul: %v", alertName, err)
		}
	}

	return nil
}
