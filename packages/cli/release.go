package cli

import (
  "bytes"
  "encoding/json"
  "fmt"
  "log"
  "net/http"
  "io/ioutil"
)

// handleRelease creates a JSON object with the alert name and sends it to the specified address.
func (app *App) handleRelease(alertName string) error {
  // Create the JSON object
  alertData := map[string]string{"alert_name": alertName}
  jsonData, err := json.Marshal(alertData)
  if err != nil {
    return fmt.Errorf("error marshaling alert data: %v", err)
  }

  address := fmt.Sprintf("http://%s", app.Config.Server.Address)
  endpoint := "/release"

  // Send JSON to the /release endpoint
  resp, err := http.Post(address+endpoint, "application/json", bytes.NewBuffer(jsonData))
  if err != nil {
    return fmt.Errorf("error sending release request: %v", err)
  }
  defer func() {
    if err := resp.Body.Close(); err != nil {
      log.Printf("Error closing response body: %v", err)
    }
  }()

  body, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    return fmt.Errorf("error reading response body: %v", err)
  }

  if resp.StatusCode != http.StatusOK {
    return fmt.Errorf("server response: %s", string(body))
  }

  log.Printf("Release request sent successfully for alert: %s", alertName)

  return nil
}
