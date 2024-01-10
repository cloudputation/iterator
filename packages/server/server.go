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
	"github.com/cloudputation/iterator/packages/countermap"
	log "github.com/cloudputation/iterator/packages/logger"
	"github.com/cloudputation/iterator/packages/storage/consul"
	"github.com/cloudputation/iterator/packages/terraform"
)

type CommandDetails struct {
    Cmd    string
    Args   []string
}

const (
	// How long we are willing to wait for the HTTP server to shut down gracefully
	serverShutdownTime = time.Second * 4
)

const (
	// Enum for reasons of why a command could or couldn't run
	CmdRunNoLabelMatch CmdRunReason = iota
	CmdRunNoMax
	CmdRunNoFinger
	CmdRunFingerUnder
	CmdRunFingerOver
)

const (
	// Namespace for prometheus metrics produced by this program
	metricNamespace    = "am_executor"

	ErrLabelRead       = "read"
	ErrLabelUnmarshall = "unmarshal"
	ErrLabelStart      = "start"
	SigLabelOk         = "ok"
	SigLabelFail       = "fail"
)

var dataDir string
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
}

var (
	
	consulDataDir = config.ConsulFactoryDataDir
	consulExecutorDir = consulDataDir + "/executor/map/fingerprints"
)
// amDataToEnv converts prometheus alert manager template data into key=value strings,
// which are meant to be set as environment variables of commands called by this program..
func amDataToEnv(td *template.Data) []string {
  env := []string{
      "TF_VAR_AMX_RECEIVER=" + td.Receiver,
      "TF_VAR_AMX_STATUS=" + td.Status,
      "TF_VAR_AMX_EXTERNAL_URL=" + td.ExternalURL,
      "TF_VAR_AMX_ALERT_LEN=" + strconv.Itoa(len(td.Alerts)),
  }

  for prefix, mappings := range map[string]map[string]string{
      "AMX_LABEL":      td.CommonLabels,
      "AMX_GLABEL":     td.GroupLabels,
      "AMX_ANNOTATION": td.CommonAnnotations,
  } {
      for k, v := range mappings {
          envVarName := "TF_VAR_" + prefix + "_" + k
          env = append(env, envVarName+"="+v)
      }
  }

  for i, alert := range td.Alerts {
      keyPrefix := "TF_VAR_AMX_ALERT_" + strconv.Itoa(i+1)
      env = append(env,
          keyPrefix+"_STATUS"+"="+alert.Status,
          keyPrefix+"_START"+"="+timeToStr(alert.StartsAt),
          keyPrefix+"_END"+"="+timeToStr(alert.EndsAt),
          keyPrefix+"_URL"+"="+alert.GeneratorURL,
          keyPrefix+"_FINGERPRINT"+"="+alert.Fingerprint,
      )
      for p, m := range map[string]map[string]string{
          "LABEL":      alert.Labels,
          "ANNOTATION": alert.Annotations,
      } {
          for k, v := range m {
              envVarName := keyPrefix + "_" + p + "_" + k
              env = append(env, envVarName+"="+v)
          }
      }
  }

  return env
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
func (s *Server) amFiring(amMsg *template.Data) []error {
  var wg sync.WaitGroup
  var env = amDataToEnv(amMsg)

  type future struct {
      cmd *command.Command
      out chan command.CommandResult
  }

  var errors = make(chan error)
  var allErrors = make([]error, 0)

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
      ok, reason := s.CanRun(cmd, amMsg)
      if !ok {
          if s.config.Verbose {
              log.Info("Skipping command due to '%s': %s", reason, cmd)
          }
          s.skipCounter.WithLabelValues(reason.Label()).Inc()
          continue
      }

      if s.config.Verbose {
          log.Info("Executing: %s", cmd)
      }

      fingerprint, _ := cmd.Fingerprint(amMsg)
      out := make(chan command.CommandResult)
      wg.Add(1)
      go collect(future{cmd: cmd, out: out})
      go s.instrument(fingerprint, cmd, env, out)

      // Store command details in a map
      s.commandDetails[fingerprint] = CommandDetails{
          Cmd:    cmd.Cmd,
          Args:   cmd.Args,
      }
  }

  wg.Wait()
  close(errors)

  for err := range errors {
      allErrors = append(allErrors, err)
  }

  return allErrors
}

// amResolved handles a resolved alert message from alertmanager
func (s *Server) amResolved(amMsg *template.Data) {
	var fingerprintFilePath string

	dataDir := s.initConfig.Server.DataDir
  executorDir := fmt.Sprintf("%s/executor/map/fingerprints", dataDir)

  for _, cmd := range s.config.Commands {
      var fingerprintData struct {
          Fingerprint string `json:"fingerprint"`
          Module      string `json:"module"`
      }
      fingerprint, ok := cmd.Fingerprint(amMsg)
      if !ok || fingerprint == "" {
          continue
      }

      // Fetching fingerprint data based on config.ConsulStorageEnabled
      switch {
      case config.ConsulStorageEnabled:
          kvPath := fmt.Sprintf("%s/%s", consulExecutorDir, fingerprint)
          data, err := consul.ConsulStoreGet(kvPath)
          if err != nil {
              log.Error("Failed to get Fingerprint data from Consul: %w", err)
              continue
          }

          err = json.Unmarshal(data, &fingerprintData)
          if err != nil {
              log.Error("Failed to unmarshal fingerprint data: %w", err)
              continue
          }

      default:
          // Construct the file path for the fingerprint JSON file
          fingerprintFilePath := fmt.Sprintf("%s/%s.json", executorDir, fingerprint)

          // Check if the fingerprint file exists
          if _, err := os.Stat(fingerprintFilePath); os.IsNotExist(err) {
              log.Error("Fingerprint file not found: %s", fingerprintFilePath)
              continue
          }

          // Read and parse the fingerprint JSON file
          fileContent, err := os.ReadFile(fingerprintFilePath)
          if err != nil {
              log.Error("Failed to read fingerprint file: %w", err)
              continue
          }

          err = json.Unmarshal(fileContent, &fingerprintData)
          if err != nil {
              log.Error("Failed to unmarshal fingerprint data: %w", err)
              continue
          }
      }

      // Process modulePath and destroy logic
      modulePath := fingerprintData.Module
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

      // Delete file from disk or remove key from Consul
      switch {
      case config.ConsulStorageEnabled:
          kvPath := fmt.Sprintf("%s/%s", consulExecutorDir, fingerprint)
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

      // Close the channel associated with the fingerprint
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

	log.Info("Alert body:")
	log.PrintJSONLog(string(data))

	var amMsg = &template.Data{}
	if err := json.Unmarshal(data, amMsg); err != nil {
		handleError(w, err)
		s.errCounter.WithLabelValues(ErrLabelUnmarshall).Inc()
		return
	}

	log.Debug("Alert template: %#v", amMsg)

	var errors []error
	switch amMsg.Status {
	case "firing":
		errors = s.amFiring(amMsg)
	case "resolved":
		// When an alert is resolved, we will attempt to signal any active commands
		// that were dispatched on behalf of it, by matching commands against fingerprints
		// used to run them.
		s.amResolved(amMsg)
	default:
		errors = append(errors, fmt.Errorf("Unknown alertmanager message status: %s", amMsg.Status))
	}

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
func (s *Server) instrument(fingerprint string, cmd *command.Command, env []string, out chan<- command.CommandResult) error {
	if s.initConfig != nil && s.initConfig.Server != nil && s.initConfig.Server.DataDir != "" {
		dataDir = s.initConfig.Server.DataDir
   	log.Debug("Data dir is: %s", dataDir)
	} else {
    dataDir = "./data"
	}
	executorDir := fmt.Sprintf("%s/executor/map/fingerprints", dataDir)

	verbose := s.config.Verbose
	s.processCurrent.Inc()
	defer s.processCurrent.Dec()
	var quit chan struct{}
	if len(fingerprint) > 0 {
		// The goroutine running the command will listen to this channel
		// to determine if it should exit early.
		quit = s.tellFingers.Add(fingerprint)
		// This value is used to determine if new commands matching this fingerprint should start.
		s.fingerCount.Inc(fingerprint)
		defer s.fingerCount.Dec(fingerprint)
	} else if s.config.Verbose {
		log.Info("Command has no fingerprint, so it won't quit early if alert is resolved first:", cmd)
	}

	done := make(chan struct{})
	cmdOut := make(chan command.CommandResult)
	// Intercept responses from commands, so that we can update metrics we're interested in
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
	cmd.Run(cmdOut, quit, done, verbose, env...)
	<-done
	s.processDuration.Observe(time.Since(start).Seconds())

	commandDetails, exists := s.commandDetails[fingerprint]
	if !exists || len(commandDetails.Args) == 0 {
			return fmt.Errorf("Command details not found or empty for fingerprint: %s", fingerprint)
	}

	chdirArg := commandDetails.Args[0]
	modulePath := chdirArg[len("-chdir="):]
	modulePath, err := filepath.Abs(modulePath)
	if err != nil {
			log.Fatal("Failed to get absolute path: %w", err)
	}

	fingerprintData := map[string]string{
		"fingerprint": fingerprint,
		"module":      modulePath,
	}

	data, err := json.MarshalIndent(fingerprintData, "", "	")
	if err != nil {
		return fmt.Errorf("Error marshaling fingerprint data: %w", err)
	}

	switch {
	case config.ConsulStorageEnabled:
		kvPath := fmt.Sprintf("%s/%s", consulExecutorDir, fingerprint)
		err = consul.ConsulStorePut(kvPath, string(data))
		if err != nil {
				return fmt.Errorf("Failed to register fingerprint on Consul: %w", err)
		}

	default:
		filePath := fmt.Sprintf("%s/%s.json", executorDir, fingerprint)
		err = os.WriteFile(filePath, data, 0644)
		if err != nil {
			return fmt.Errorf("Error writing fingerprint data to file: %w", err)
		}
	}

	return nil
}

// CanRun returns true if the Command is allowed to run based on its fingerprint and settings
func (s *Server) CanRun(cmd *command.Command, amMsg *template.Data) (bool, CmdRunReason) {
	if !cmd.Matches(amMsg) {
		return false, CmdRunNoLabelMatch
	}

	if cmd.Max <= 0 {
		return true, CmdRunNoMax
	}

	fingerprint, ok := cmd.Fingerprint(amMsg)
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