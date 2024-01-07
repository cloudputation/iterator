package logger

import (
    "encoding/json"
    "strings"
)

func PrintJSONLog(jsonStr string) error {
  var jsonObj interface{}
	err := json.Unmarshal([]byte(jsonStr), &jsonObj)
	if err != nil {
		return err
	}
	prettyJSON, err := json.MarshalIndent(jsonObj, "", "    ")
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(prettyJSON), "\n") {
		Info(line)
	}

	return nil
}
