package web

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/blend/go-sdk/async"
	"github.com/blend/go-sdk/exception"
	"github.com/blend/go-sdk/logger"
)

// Verify interfaces.
var (
	_ Shutdowner = (*Healthz)(nil)
)

const (
	// VarzStarted is a common variable.
	VarzStarted = "startedUTC"
	// VarzRequests is a common variable.
	VarzRequests = "http_requests"
	// VarzRequests2xx is a common variable.
	VarzRequests2xx = "http_requests2xx"
	// VarzRequests3xx is a common variable.
	VarzRequests3xx = "http_requests3xx"
	// VarzRequests4xx is a common variable.
	VarzRequests4xx = "http_requests4xx"
	// VarzRequests5xx is a common variable.
	VarzRequests5xx = "http_requests5xx"
	// VarzErrors is a common variable.
	VarzErrors = "errors_total"
	// VarzFatals is a common variable.
	VarzFatals = "fatals_total"

	// ListenerHealthz is the uid of the healthz logger listeners.
	ListenerHealthz = "healthz"

	// ErrHealthzAppUnset is a common error.
	ErrHealthzAppUnset exception.Class = "healthz app unset"
)

// NewHealthz returns a new healthz.
func NewHealthz(hosted Shutdowner) *Healthz {
	return &Healthz{
		hosted:         hosted,
		bindAddr:       DefaultHealthzBindAddr,
		gracePeriod:    DefaultShutdownGracePeriod,
		latch:          &async.Latch{},
		defaultHeaders: map[string]string{},
	}
}

// Healthz is a sentinel / healthcheck sidecar that can run on a different
// port to the main app.
/*
It typically implements the following routes:

	/healthz - overall health endpoint, 200 on healthy, 5xx on not.
				should be used as a kubernetes readiness probe.
	/debug/vars - `pkg/expvar` output.
*/
type Healthz struct {
	self           *App
	hosted         Shutdowner
	cfg            *HealthzConfig
	bindAddr       string
	log            *logger.Logger
	latch          *async.Latch
	defaultHeaders map[string]string
	gracePeriod    time.Duration
	recoverPanics  bool
}

// WithConfig sets the healthz config and relevant properties.
func (hz *Healthz) WithConfig(cfg *HealthzConfig) *Healthz {
	hz.cfg = cfg
	hz.WithBindAddr(cfg.GetBindAddr())
	hz.WithGracePeriodSeconds(cfg.GetGracePeriod())
	hz.WithRecoverPanics(cfg.GetRecoverPanics())
	return hz
}

// Config returns the healthz config.
func (hz *Healthz) Config() *HealthzConfig {
	return hz.cfg
}

// WithBindAddr sets the bind address.
func (hz *Healthz) WithBindAddr(bindAddr string) *Healthz {
	hz.bindAddr = bindAddr
	return hz
}

// BindAddr returns the bind address.
func (hz *Healthz) BindAddr() string {
	return hz.bindAddr
}

// WithGracePeriodSeconds sets the grace period seconds
func (hz *Healthz) WithGracePeriodSeconds(gracePeriod time.Duration) *Healthz {
	hz.gracePeriod = gracePeriod
	return hz
}

// GracePeriod returns the grace period in seconds
func (hz *Healthz) GracePeriod() time.Duration {
	return hz.gracePeriod
}

// Hosted returns the underlying app.
func (hz *Healthz) Hosted() Shutdowner {
	return hz.hosted
}

// RecoverPanics returns if the app recovers panics.
func (hz *Healthz) RecoverPanics() bool {
	return hz.recoverPanics
}

// WithRecoverPanics sets if the app should recover panics.
func (hz *Healthz) WithRecoverPanics(value bool) *Healthz {
	hz.recoverPanics = value
	return hz
}

// Logger returns the diagnostics agent for the app.
func (hz *Healthz) Logger() *logger.Logger {
	return hz.log
}

// WithLogger sets the app logger agent and returns a reference to the app.
// It also sets underlying loggers in any child resources like providers and the auth manager.
func (hz *Healthz) WithLogger(log *logger.Logger) *Healthz {
	hz.log = log
	return hz
}

// WithDefaultHeader adds a default header.
func (hz *Healthz) WithDefaultHeader(key, value string) *Healthz {
	hz.defaultHeaders[key] = value
	return hz
}

// DefaultHeaders returns the default headers.
func (hz *Healthz) DefaultHeaders() map[string]string {
	return hz.defaultHeaders
}

// Start implements shutdowner.
func (hz *Healthz) Start() error {
	hz.latch.Starting()
	hz.self = New().
		WithHandler(hz).
		WithBindAddr(hz.bindAddr).
		WithLogger(hz.log)

	hz.latch.Started()
	return async.RunToError(hz.self.Start, hz.hosted.Start)
}

// Shutdown implements shutdowner.
func (hz *Healthz) Shutdown() error {
	context, cancel := context.WithTimeout(context.Background(), hz.GracePeriod())
	defer cancel()

	if hz.log != nil {
		hz.log.Infof("healthz is shutting down with (%s) grace period", hz.GracePeriod())
	}
	// set the next call to `/healtz` to
	// finish the shutdown
	hz.latch.Stopping()

	select {
	// if the hosted app crashes
	case <-hz.hosted.NotifyShutdown():
		return hz.self.Shutdown()
	// if the shutdown grace period expires
	case <-context.Done():
		if hz.log != nil {
			hz.log.Warningf("healthz shutdown grace period has expired")
		}
		return hz.shutdownServers()
	// if we've received a final /healthz request
	case <-hz.latch.NotifyStopped():
		return hz.shutdownServers()
	}
}

// IsRunning returns if the healthz server is running.
func (hz *Healthz) IsRunning() bool {
	return hz.self.IsRunning()
}

// NotifyStarted returns the notify started signal.
func (hz *Healthz) NotifyStarted() <-chan struct{} {
	return hz.self.NotifyStarted()
}

// NotifyShutdown returns the notify shutdown signal.
func (hz *Healthz) NotifyShutdown() <-chan struct{} {
	return hz.self.NotifyShutdown()
}

func (hz *Healthz) shutdownServers() error {
	return async.RunToError(hz.hosted.Shutdown, hz.self.Shutdown)
}

// ServeHTTP makes the router implement the http.Handler interface.
func (hz *Healthz) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if hz.recoverPanics {
		defer hz.recover(w, r)
	}

	start := time.Now()
	route := strings.ToLower(r.URL.Path)

	res := NewRawResponseWriter(w)
	res.Header().Set(HeaderContentEncoding, ContentEncodingIdentity)

	if hz.log != nil {
		hz.log.Trigger(logger.NewHTTPRequestEvent(r).WithRoute(route))

		defer func() {
			hz.log.Trigger(logger.NewHTTPResponseEvent(r).
				WithStatusCode(res.StatusCode()).
				WithElapsed(time.Since(start)).
				WithContentLength(res.ContentLength()),
			)
		}()
	}

	if len(hz.defaultHeaders) > 0 {
		for key, value := range hz.defaultHeaders {
			res.Header().Set(key, value)
		}
	}

	switch route {
	case "/healthz":
		hz.healthzHandler(res, r)
	default:
		http.NotFound(res, r)
	}

	if err := res.Close(); err != nil && err != http.ErrBodyNotAllowed && hz.log != nil {
		hz.log.Error(err)
	}
}

func (hz *Healthz) recover(w http.ResponseWriter, req *http.Request) {
	if rcv := recover(); rcv != nil {
		if hz.log != nil {
			hz.log.Fatalf("%v", rcv)
		}

		http.Error(w, fmt.Sprintf("%v", rcv), http.StatusInternalServerError)
		return
	}
}

func (hz *Healthz) healthzHandler(w ResponseWriter, r *http.Request) {
	if hz.latch.IsStopping() {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set(HeaderContentType, ContentTypeText)
		fmt.Fprintf(w, "Shutting down.\n")
		if hz.log != nil {
			hz.log.Debugf("healthz received probe while in process of shutdown")
		}
		hz.latch.Stopped()
	} else if hz.hosted.IsRunning() {
		w.WriteHeader(http.StatusOK)
		w.Header().Set(HeaderContentType, ContentTypeText)
		fmt.Fprintf(w, "OK!\n")
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set(HeaderContentType, ContentTypeText)
		fmt.Fprintf(w, "Failure!\n")
	}
	return
}
