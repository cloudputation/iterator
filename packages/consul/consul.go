package consul

import (
		"fmt"
		"encoding/json"
		"strings"

		"github.com/hashicorp/consul/api"
		"github.com/cloudputation/iterator/packages/config"
		log "github.com/cloudputation/iterator/packages/logger"
)

type IteratorStatus struct {
	Status string `json:"iterator_status"`
}

var ConsulClient *api.Client
var err error
var statusDir = config.ConsulFactoryDataDir + "/status"


func InitConsul(consulAddress string) error {
	consulConfig := api.DefaultConfig()
	consulConfig.Address = consulAddress

	ConsulClient, err = api.NewClient(consulConfig)
	if err != nil {
			return fmt.Errorf("Failed to initialize Consul client: %w", err)
	}
	log.Info("Consul client initialized successfully.")


	return nil
}

func BootstrapConsul() error {
	kv := ConsulClient.KV()

	// Try to read a key-value pair from Consul
	log.Info("Checking if data store is initialized.")
	pair, _, err := kv.Get(statusDir, nil)
	if err != nil {
		return fmt.Errorf("Failed to initiate Consul connection: %w", err)
	}

	if pair == nil {
		log.Info("Consul data store is not initialized. Initializing..")
		// Create an instance of FactoryStatus with the status set to "initialized"
		status := IteratorStatus{Status: "initialized"}

		statusJSON, err := json.Marshal(status)
		if err != nil {
			return fmt.Errorf("Failed to marshal JSON: %w", err)
		}

		writeOptions := &api.WriteOptions{}
		p := &api.KVPair{Key: statusDir, Value: statusJSON}
		_, err = kv.Put(p, writeOptions)
		if err != nil {
			return fmt.Errorf("Failed to initialize data store on Consul: %w", err)
		}
		log.Info("Data store initialized successfully.")
	} else {
		log.Info("Data store is already initialized.")
	}


	return nil
}

func ConsulStoreGet(key string) ([]byte, error) {
    kv := ConsulClient.KV()

    kvPair, _, err := kv.Get(key, nil)
    if err != nil {
        return nil, fmt.Errorf("Failed to query key on Consul: %v", err)
    }
    if kvPair == nil {
        return nil, fmt.Errorf("Key not found: %s", key)
    }

    return kvPair.Value, nil
}

func ConsulStorePut(keyPath, jsonData string) error {
	kv := ConsulClient.KV()

	writeOptions := &api.WriteOptions{}
	p := &api.KVPair{Key: keyPath, Value: []byte(jsonData)}
	_, err = kv.Put(p, writeOptions)
	if err != nil {
			return fmt.Errorf("Failed to upload key: %s, error: %w", keyPath, err)
	}


	return nil
}

func ConsulStoreDelete(keyPath string) error {
	kv := ConsulClient.KV()

	_, err := kv.Delete(keyPath, nil)
	if err != nil {
			return fmt.Errorf("Failed to delete key: %s, error: %w", keyPath, err)
	}


	return nil
}

func ConsulStoreListKeys(path string, recursive bool) ([]string, error) {
	kv := ConsulClient.KV()

	var keys []string
	var separator string

	// Ensure path ends with a slash for correct prefix behavior
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	// If not recursive, set the separator to filter out sub-directory keys
	if !recursive {
		separator = "/"
	}

	// Get the list of keys
	kvPairs, _, err := kv.Keys(path, separator, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to list keys at path: %s, error: %w", path, err)
	}

	for _, key := range kvPairs {
		// Skip the directory path itself
		if key == path {
			continue
		}

		// Remove the path prefix from the key
		trimmedKey := strings.TrimPrefix(key, path)

		// Remove any trailing slash, indicating a sub-directory
		trimmedKey = strings.TrimSuffix(trimmedKey, "/")

		if trimmedKey != "" {
			keys = append(keys, trimmedKey)
		}
	}

	return keys, nil
}
