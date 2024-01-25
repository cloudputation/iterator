package server

import (
	"context"
	"encoding/json"
	"fmt"
	l "log"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	pm "github.com/prometheus/client_model/go"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudputation/iterator/packages/chanmap"
	"github.com/cloudputation/iterator/packages/command"
	"github.com/cloudputation/iterator/packages/config"
  "github.com/cloudputation/iterator/packages/consul"
	"github.com/cloudputation/iterator/packages/countermap"
	log "github.com/cloudputation/iterator/packages/logger"
	"github.com/cloudputation/iterator/packages/terraform"
)

type CommandDetails struct {
    Cmd    string
    Args   []string
		TerraformScheduling string
}

const (
	// How long we are willing to wait for the HTTP server to shut down gracefully
	serverShutdownTime = time.Second * 4
)

const (
	// Enum for reasons for why a command could or couldn't run
	CmdRunNoLabelMatch CmdRunReason = iota
	CmdRunNoMax
	CmdRunNoFinger
	CmdRunFingerUnder
	CmdRunFingerOver
)

const (
	// Namespace for prometheus metrics produced by this program
	metricNamespace    = "iterator"

	ErrLabelRead       = "read"
	ErrLabelUnmarshall = "unmarshal"
	ErrLabelStart      = "start"
	SigLabelOk         = "ok"
	SigLabelFail       = "fail"
)

var (
	CmdRunDesc = map[CmdRunReason]string{
		CmdRunNoLabelMatch: "No match for alert labels",
		CmdRunNoMax:        "No maximum simultaneous command limit defined",
		CmdRunNoFinger:     "No fingerprint found for command",
		CmdRunFingerUnder:  "Command count for fingerprint is under limit",
		CmdRunFingerOver:   "Command count for fingerprint is over limit",
	}

	// These labels are meant to be applied to prometheus metrics
	CmdRunLabel = map[CmdRunReason]string{
		CmdRunNoLabelMatch: "nomatch",
		CmdRunNoMax:        "nomax",
		CmdRunNoFinger:     "nofinger",
		CmdRunFingerUnder:  "fingerunder",
		CmdRunFingerOver:   "fingerover",
	}

	procDurationOpts = prometheus.HistogramOpts{
		Namespace: metricNamespace,
		Subsystem: "process",
		Name:      "duration_seconds",
		Help:      "Time the processes handling alerts ran.",
		Buckets:   []float64{1, 10, 60, 600, 900, 1800},
	}

	procCurrentOpts = prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Subsystem: "processes",
		Name:      "current",
		Help:      "Current number of processes running.",
	}

	errCountOpts = prometheus.CounterOpts{
		Namespace: metricNamespace,
		Subsystem: "errors",
		Name:      "total",
		Help:      "Total number of errors while processing alerts.",
	}

	sigCountOpts = prometheus.CounterOpts{
		Namespace: metricNamespace,
		Subsystem: "signalled",
		Name:      "total",
		Help:      "Total number of active processes signalled due to alarm resolving.",
	}

	skipCountOpts = prometheus.CounterOpts{
		Namespace: metricNamespace,
		Subsystem: "skipped",
		Name:      "total",
		Help:      "Total number of commands that were skipped instead of run for matching alerts.",
	}

	errCountLabels  = []string{"stage"}
	sigCountLabels  = []string{"result"}
	skipCountLabels = []string{"reason"}
)

type CmdRunReason int

type Server struct {
	// initConfig refers to the preloaded server configuration
	initConfig *config.InitConfig
	// config refers to the executor's configuration
	config *config.Config
	// A mapping of an alarm fingerprint to a channel that can be used to
	// trigger action on all executing commands matching that fingerprint.
	// In our case, we want the ability to signal a running process if the matching channel is closed.
	// Alarms without a fingerprint aren't tracked by the map.
	tellFingers *chanmap.ChannelMap
	// A mapping of an alarm fingerprint to the number of commands being executed for it.
	// This is compared to the Command.Max value to determine if a command should execute.
	fingerCount *countermap.Counter
	// An instance of metrics registry.
	// We use this instead of the default, because the default only allows one instance of metrics to be registered.
	registry        *prometheus.Registry
	processDuration prometheus.Histogram
	processCurrent  prometheus.Gauge
	errCounter      *prometheus.CounterVec
	// Track number of active processes signalled due to a 'resolved' message being received from alertmanager.
	sigCounter *prometheus.CounterVec
	// Track number of commands skipped instead of run.
	skipCounter *prometheus.CounterVec
	// Map to store the command details indexed by fingerprint
	commandDetails map[string]CommandDetails
  commandDetailsMutex sync.Mutex
}

var (
	dataDir string
	terraformScheduling string
	consulDataDir = config.ConsulFactoryDataDir
	consulExecutorDir = consulDataDir + "/process/alerts"
)

// amDataToEnv converts prometheus alert manager template data into key=value strings,
// which are meant to be set as environment variables of commands called by this program..
func amDataToEnvForAlert(alert *template.Alert) []string {
  env := []string{
      "TF_VAR_ITERATOR_ALERT_STATUS=" + alert.Status,
      "TF_VAR_ITERATOR_ALERT_START=" + timeToStr(alert.StartsAt),
      "TF_VAR_ITERATOR_ALERT_END=" + timeToStr(alert.EndsAt),
      "TF_VAR_ITERATOR_ALERT_URL=" + alert.GeneratorURL,
      "TF_VAR_ITERATOR_ALERT_FINGERPRINT=" + alert.Fingerprint,
  }

  for prefix, mappings := range map[string]map[string]string{
      "ITERATOR_ALERT_LABEL":      alert.Labels,
      "ITERATOR_ALERT_ANNOTATION": alert.Annotations,
  } {
      for k, v := range mappings {
          envVarName := "TF_VAR_" + prefix + "_" + k
          env = append(env, envVarName+"="+v)
      }
  }

  return env
}

// Give default to CommandDetails
func DefaultCommandDetails(cmd string, args []string, terraformScheduling string) CommandDetails {
    if terraformScheduling == "" {
        terraformScheduling = "default" // Replace 'default_value' with the actual default value you want
    }
    return CommandDetails{
        Cmd:                 cmd,
        Args:                args,
        TerraformScheduling: terraformScheduling,
    }
}

// concatErrors returns an error representing all of the errors' strings
func concatErrors(errors ...error) error {
	var s = make([]string, 0)
	for _, err := range errors {
		if err != nil {
			s = append(s, err.Error())
		}
	}

	return fmt.Errorf(strings.Join(s, "\n"))
}

// timeToStr converts the Time struct into a string representing its Unix epoch.
func timeToStr(t time.Time) string {
	if t.IsZero() {
		return "0"
	}
	return strconv.Itoa(int(t.Unix()))
}

// handleError responds to an HTTP request with an error message and logs it
func handleError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
	log.Error("%w", err)
}

// handleHealth is meant to respond to health checks for this program
func handleHealth(w http.ResponseWriter, req *http.Request) {
	_, err := fmt.Fprint(w, "All systems are functioning within normal specifications.\n")
	if err != nil {
		handleError(w, err)
	}
}

// Label returns a prometheus-compatible label for a reason why a command could or couldn't run
func (r CmdRunReason) Label() string {
	return CmdRunLabel[r]
}

// String returns a string representation of the reason why a command could or couldn't run
func (r CmdRunReason) String() string {
	return CmdRunDesc[r]
}

// amFiring handles a triggered alert message from alertmanager
func (s *Server) amFiring(alert *template.Alert) []error {
  var wg sync.WaitGroup
  var allErrors = make([]error, 0)
  env := amDataToEnvForAlert(alert)

  type future struct {
      cmd *command.Command
      out chan command.CommandResult
  }

  var errors = make(chan error)

  var collect = func(f future) {
      defer wg.Done()
      var resultState command.Result
      for result := range f.out {
          resultState |= result.Kind
          if result.Kind.Has(command.CmdFail) && result.Err != nil && f.cmd.ShouldNotify() {
              errors <- result.Err
          }
      }
      if s.config.Verbose {
          log.Info("Command: %s, result: %s", f.cmd.String(), resultState)
      }
  }

  for _, cmd := range s.config.Commands {
      ok, reason := s.CanRun(cmd, alert)
      if !ok {
          log.Info("Skipping command due to '%s': %s", reason, cmd)
          s.skipCounter.WithLabelValues(reason.Label()).Inc()
          continue
      }

      log.Info("Executing Terraform module: %s for alert: %s", cmd.Args, alert.Labels["alertname"])

      fingerprint, ok := cmd.FingerprintForAlert(alert)
      if !ok {
					log.Info("Failed to get fingertprint for alert: %s. Skipping..", alert.Labels["alertname"])
          continue
      }

			log.Info("Setting commandDetails for fingerprint: %s, command: %s, args: %v", fingerprint, cmd.Cmd, cmd.Args)
			commandDetails := DefaultCommandDetails(cmd.Cmd, cmd.Args, cmd.TerraformScheduling)

      s.commandDetailsMutex.Lock()
      s.commandDetails[fingerprint] = commandDetails
      s.commandDetailsMutex.Unlock()

      out := make(chan command.CommandResult)
      wg.Add(1)
      go collect(future{cmd: cmd, out: out})

      err := s.instrument(fingerprint, cmd, env, out, *alert)
      if err != nil {
          allErrors = append(allErrors, err)
          continue
      }


  }

  go func() {
      wg.Wait()
      close(errors)
  }()

  for err := range errors {
      allErrors = append(allErrors, err)
  }

  return allErrors
}


// amResolved handles a resolved alert message from alertmanager
func (s *Server) amResolved(alert template.Alert) {
  dataDir := s.initConfig.Server.DataDir
  executorDir := fmt.Sprintf("%s/process/alerts", dataDir)
  alertname := alert.Labels["alertname"]

  for _, cmd := range s.config.Commands {
      var alertParameters struct {
          Fingerprint        string `json:"fingerprint"`
          Module             string `json:"module"`
          TerraformScheduling string `json:"terraform_scheduling"`
      }

      fingerprint, ok := cmd.FingerprintForAlert(&alert)
      if !ok || fingerprint == "" {
          continue
      }

			fingerprintFilePath := fmt.Sprintf("%s/%s.json", executorDir, fingerprint)

      switch {
      case config.ConsulStorageEnabled:
          kvPath := fmt.Sprintf("%s/%s", consulExecutorDir, alertname)
          data, err := consul.ConsulStoreGet(kvPath)
          if err != nil {
              log.Error("Failed to get Fingerprint data from Consul: %w", err)
              continue
          }
          err = json.Unmarshal(data, &alertParameters)
          if err != nil {
              log.Error("Failed to unmarshal fingerprint data: %w", err)
              continue
          }

      default:
          if _, err := os.Stat(fingerprintFilePath); os.IsNotExist(err) {
              log.Error("Fingerprint file not found: %s", fingerprintFilePath)
              continue
          }

          fileContent, err := os.ReadFile(fingerprintFilePath)
          if err != nil {
              log.Error("Failed to read fingerprint file: %w", err)
              continue
          }
          err = json.Unmarshal(fileContent, &alertParameters)
          if err != nil {
              log.Error("Failed to unmarshal fingerprint data: %w", err)
              continue
          }
      }

      if alertParameters.TerraformScheduling == "sawtooth" {
          log.Info("Terraform scheduling mode is set to sawtooth for this alert. Skipping destroy for alert: %s", fingerprint)
          continue
      }

      modulePath := alertParameters.Module
      if modulePath == "" {
          log.Error("Module path is empty in fingerprint file")
          continue
      }

      destroy := true
      if cmd.DestroyOnResolved != nil {
          destroy = *cmd.DestroyOnResolved
      }

      if destroy {
          go terraform.RunTerraform(modulePath, "destroy")
      }

      switch {
      case config.ConsulStorageEnabled:
          kvPath := fmt.Sprintf("%s/%s", consulExecutorDir, alertname)
          err := consul.ConsulStoreDelete(kvPath)
          if err != nil {
              log.Error("Failed to delete key from Consul: %w", err)
              continue
          }

      default:
          err := os.Remove(fingerprintFilePath)
          if err != nil {
              log.Error("Failed to delete fingerprint file: %w", err)
              continue
          }
      }

      s.tellFingers.Close(fingerprint)
		}
}

// handleWebhook is meant to respond to webhook requests from prometheus alertmanager.
// It unpacks the alert, and dispatches it to the matching programs through environment variables.
//
// If a command fails, an HTTP 500 response is returned to alertmanager.
// Note that alertmanager may treat non HTTP 200 responses as 'failure to notify', and may re-dispatch the alert to us.
func (s *Server) handleWebhook(w http.ResponseWriter, req *http.Request) {
    log.Info("Webhook triggered from remote address: %s", req.RemoteAddr)

    data, err := ioutil.ReadAll(req.Body)
    if err != nil {
        handleError(w, err)
        s.errCounter.WithLabelValues(ErrLabelRead).Inc()
        return
    }

    log.Debug("Alert body:")
    log.PrintJSONLog(string(data))

    var amMsg = &template.Data{}
    if err := json.Unmarshal(data, amMsg); err != nil {
        handleError(w, err)
        s.errCounter.WithLabelValues(ErrLabelUnmarshall).Inc()
        return
    }

    var wg sync.WaitGroup
    var errors []error
    var mu sync.Mutex

    for _, alert := range amMsg.Alerts {
        log.Debug("Processing alert: %v", alert.Labels["alertname"])
        wg.Add(1)
				alertCopy := alert

		    go func(alert template.Alert) {
		        defer wg.Done()
		        switch alert.Status {
		        case "firing":
		            if errs := s.amFiring(&alert); len(errs) > 0 {
		                mu.Lock()
		                errors = append(errors, errs...)
		                mu.Unlock()
		            }
		        case "resolved":
		            s.amResolved(alert)
		        default:
		            mu.Lock()
		            errors = append(errors, fmt.Errorf("Unknown alert status: %s", alert.Status))
		            mu.Unlock()
		        }
		    }(alertCopy)
    }

    wg.Wait()

    if len(errors) > 0 {
        handleError(w, concatErrors(errors...))
    }
}

// initMetrics initializes prometheus metrics
func (s *Server) initMetrics() error {
	var pd, pc pm.Metric
	err := s.processDuration.Write(&pd)
	if err != nil {
		return err
	}

	err = s.processCurrent.Write(&pc)
	if err != nil {
		return err
	}

	_ = s.errCounter.WithLabelValues(ErrLabelRead)
	_ = s.errCounter.WithLabelValues(ErrLabelUnmarshall)
	_ = s.errCounter.WithLabelValues(ErrLabelStart)
	_ = s.sigCounter.WithLabelValues(ErrLabelStart)
	_ = s.sigCounter.WithLabelValues(SigLabelOk)
	_ = s.sigCounter.WithLabelValues(SigLabelFail)
	_ = s.skipCounter.WithLabelValues(CmdRunNoLabelMatch.Label())
	_ = s.skipCounter.WithLabelValues(CmdRunFingerOver.Label())

	return nil
}

// instrument a command.
// It is meant to be called as a goroutine with context provided by handleWebhook.
//
// The prometheus structs use sync/atomic in methods like Dec and Observe,
// so they're safe to call concurrently from goroutines.
func (s *Server) instrument(fingerprint string, cmd *command.Command, env []string, out chan<- command.CommandResult, alert template.Alert) error {
    dataDir := "./data"
    if s.initConfig != nil && s.initConfig.Server != nil && s.initConfig.Server.DataDir != "" {
        dataDir = s.initConfig.Server.DataDir
    }
    executorDir := fmt.Sprintf("%s/process/alerts", dataDir)
		alertName := alert.Labels["alertname"]
		var terraform_scheduling string

    s.processCurrent.Inc()
    defer s.processCurrent.Dec()

    var quit chan struct{}
    if len(fingerprint) > 0 {
        quit = s.tellFingers.Add(fingerprint)
        s.fingerCount.Inc(fingerprint)
        defer s.fingerCount.Dec(fingerprint)
    }

    done := make(chan struct{})
    cmdOut := make(chan command.CommandResult)

    go func() {
        defer close(out)
        for r := range cmdOut {
            if r.Kind.Has(command.CmdFail) && r.Err != nil && cmd.ShouldNotify() {
                s.errCounter.WithLabelValues(ErrLabelStart).Inc()
            }
            if r.Kind.Has(command.CmdSigOk) {
                s.sigCounter.WithLabelValues(SigLabelOk).Inc()
            }
            if r.Kind.Has(command.CmdSigFail) {
                s.sigCounter.WithLabelValues(SigLabelFail).Inc()
            }
            out <- r
        }
    }()

    start := time.Now()
		log.Debug("Running command for alert fingerprint: %s", fingerprint)
		cmd.Run(out, quit, done, env...)
    <-done
    s.processDuration.Observe(time.Since(start).Seconds())


		s.commandDetailsMutex.Lock()
		commandDetails, exists := s.commandDetails[fingerprint]
		s.commandDetailsMutex.Unlock()

    if !exists || len(commandDetails.Args) == 0 {
				log.Error("Command details not found or empty for fingerprint: %s, current map: %+v", fingerprint, s.commandDetails)
        return fmt.Errorf("Command details not found or empty for fingerprint: %s", fingerprint)
    }

    chdirArg := commandDetails.Args[0]
    modulePath := chdirArg[len("-chdir="):]
    modulePath, err := filepath.Abs(modulePath)
    if err != nil {
        log.Fatal("Failed to get absolute path: %w", err)
    }

		terraform_scheduling = commandDetails.TerraformScheduling

    alertParameters := map[string]string{
        "fingerprint": fingerprint,
        "module":      modulePath,
				"terraform_scheduling": terraform_scheduling,
    }

    data, err := json.MarshalIndent(alertParameters, "", "    ")
    if err != nil {
        return fmt.Errorf("Error marshaling fingerprint data: %w", err)
    }

		switch {
		case config.ConsulStorageEnabled:
			log.Info("Using Consul as storage backend for alert: %s", alertName)
			kvPath := fmt.Sprintf("%s/%s", consulExecutorDir, alertName)
			err = consul.ConsulStorePut(kvPath, string(data))
			if err != nil {
					return fmt.Errorf("Failed to register fingerprint on Consul: %w", err)
			}
		default:
			log.Info("Using defaut storage backend for alert: %s", alertName)
			filePath := fmt.Sprintf("%s/%s.json", executorDir, fingerprint)
			err = os.WriteFile(filePath, data, 0644)
			if err != nil {
					return fmt.Errorf("Error writing fingerprint data to file: %w", err)
			}
		}


		return nil
}

// CanRun returns true if the Command is allowed to run based on its fingerprint and settings
func (s *Server) CanRun(cmd *command.Command, alert *template.Alert) (bool, CmdRunReason) {
	if !cmd.Matches(alert) {
		return false, CmdRunNoLabelMatch
	}

	if cmd.Max <= 0 {
		return true, CmdRunNoMax
	}

	fingerprint, ok := cmd.FingerprintForAlert(alert)
	if !ok || fingerprint == "" {
		return true, CmdRunNoFinger
	}

	v, ok := s.fingerCount.Get(fingerprint)
	if !ok || v < cmd.Max {
		return true, CmdRunFingerUnder
	}

	return false, CmdRunFingerOver
}

// Start runs a golang http server with the given routes.
// Returns
// * a reference to the HTTP server (so that we can gracefully shut it down)
// a channel that will contain the error result of the ListenAndServe call
func (s *Server) Start() (*http.Server, chan error) {
	log.Info("STARTING SERVER")
	log.Info("Listening on port: %s", s.config.ListenAddr)
	serverPort := fmt.Sprintf(":%s", s.config.ListenAddr)
	s.registry.MustRegister(s.processDuration)
	s.registry.MustRegister(s.processCurrent)
	s.registry.MustRegister(s.errCounter)
	s.registry.MustRegister(s.sigCounter)
	s.registry.MustRegister(s.skipCounter)

	// Initialize metrics
	err := s.initMetrics()
	if err != nil {
		panic(err)
	}

	// We use our own instance of ServeMux instead of DefaultServeMux,
	// to keep handler registration separate between server instances.
	mux := http.NewServeMux()
	srv := &http.Server{Addr: serverPort, Handler: mux}
	mux.HandleFunc("/", s.handleWebhook)
	mux.HandleFunc("/_health", handleHealth)
	mux.Handle("/metrics", promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{
		// Prometheus can use the same logger we are, when printing errors about serving metrics
		ErrorLog: l.New(os.Stderr, "", l.LstdFlags),
		// Include metric handler errors in metrics output
		Registry: s.registry,
	}))

	// Start http server in a goroutine, so that it doesn't block other activities
	var httpSrvResult = make(chan error, 1)
	go func() {
		defer close(httpSrvResult)
		commands := make([]string, len(s.config.Commands))
		for i, e := range s.config.Commands {
			commands[i] = e.String()
		}

		if (s.config.TLSCrt != "") && (s.config.TLSKey != "") {
			if s.config.Verbose {
				log.Info("HTTPS on")
			}
			httpSrvResult <- srv.ListenAndServeTLS(s.config.TLSCrt, s.config.TLSKey)
		} else {
			if s.config.Verbose {
				log.Info("HTTPS off")
			}
			httpSrvResult <- srv.ListenAndServe()
		}
	}()

	return srv, httpSrvResult
}

func StopServer(srv *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), serverShutdownTime)
	defer cancel()
	return srv.Shutdown(ctx)
}

// NewServer returns a new server instance
func NewServer(initConfig *config.InitConfig, config *config.Config) *Server {
	s := Server{
			initConfig:      initConfig,
			config:          config,
			tellFingers:     chanmap.NewChannelMap(),
			fingerCount:     countermap.NewCounter(),
			registry:        prometheus.NewPedanticRegistry(),
			processDuration: prometheus.NewHistogram(procDurationOpts),
			processCurrent:  prometheus.NewGauge(procCurrentOpts),
			errCounter:      prometheus.NewCounterVec(errCountOpts, errCountLabels),
			sigCounter:      prometheus.NewCounterVec(sigCountOpts, sigCountLabels),
			skipCounter:     prometheus.NewCounterVec(skipCountOpts, skipCountLabels),
			commandDetails:	 make(map[string]CommandDetails),
	}

	return &s
}
