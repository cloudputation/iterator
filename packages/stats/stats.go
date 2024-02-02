package stats

import (
  "fmt"
	"encoding/json"

  "github.com/cloudputation/iterator/packages/config"
  "github.com/cloudputation/iterator/packages/consul"
	log "github.com/cloudputation/iterator/packages/logger"
)

type IteratorStatus struct {
	Status       string   `json:"iterator_status"`
	ActiveAlerts []string `json:"active_alerts,omitempty"`
}

// UpdateStatusWithActiveAlerts updates the status key with active alerts.
func UpdateStatusWithActiveAlerts() error {
	// Retrieve keys from the /process/alerts path
	alertsPath := config.ConsulFactoryDataDir + "/process/alerts"
	recursive := false

	alertKeys, err := consul.ConsulStoreListKeys(alertsPath, recursive)
	if err != nil {
		return fmt.Errorf("Failed to retrieve alert keys: %v", err)
	}

	// Retrieve the current status
	statusPath := config.ConsulFactoryDataDir + "/status"
	currentStatusJSON, err := consul.ConsulStoreGet(statusPath)
	if err != nil {
		return fmt.Errorf("Failed to retrieve current status: %v", err)
	}

	// Unmarshal the current status
	var currentStatus IteratorStatus
	err = json.Unmarshal(currentStatusJSON, &currentStatus)
	if err != nil {
		return fmt.Errorf("Failed to unmarshall current status: %v", err)
	}

	// Update the status with the retrieved alert keys
	currentStatus.ActiveAlerts = alertKeys

	// Marshal the updated status back to JSON
	updatedStatusJSON, err := json.MarshalIndent(currentStatus, "", "    ")
	if err != nil {
		return fmt.Errorf("Failed to marshall updated status: %v", err)
	}

	// Update the status key in Consul
	err = consul.ConsulStorePut(statusPath, string(updatedStatusJSON))
	if err != nil {
		return fmt.Errorf("Failed to update status key: %v", err)
	}

	log.Info("Status key updated successfully with active alerts")

  return nil
}
