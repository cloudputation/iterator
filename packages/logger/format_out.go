package logger

import (
  "encoding/json"
  "gopkg.in/yaml.v2"
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
    Debug(line)
  }

  return nil
}

func PrintYAMLLog(yamlStr string) error {
  var yamlObj interface{}
  err := yaml.Unmarshal([]byte(yamlStr), &yamlObj)
  if err != nil {
    return err
  }
  prettyYAML, err := yaml.Marshal(yamlObj)
  if err != nil {
    return err
  }
  for _, line := range strings.Split(string(prettyYAML), "\n") {
    Debug(line)
  }

  return nil
}
