package run

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/raulk/clock"

	"github.com/testground/sdk-go"
	"github.com/testground/sdk-go/runtime"
)

var (
	// _clk can be overridden with a mock clock for test purposes.
	_clk = clock.New()
)

const (
	// These ports are the HTTP ports we'll attempt to bind to. If this instance
	// is running in a Docker container, binding to 6060 is safe. If it's a
	// local:exec run, these ports belong to the host, so starting more than one
	// instance will lead to a collision. Therefore we fallback to 0.
	HTTPPort         = 6060
	HTTPPortFallback = 0
)

// HTTPListenAddr will be set to the listener address _before_ the test case is
// invoked. If we were unable to start the listener, this value will be "".
var HTTPListenAddr string

type TestCaseFn = func(env *runtime.RunEnv) error

// InitializedTestCaseFn allows users to indicate they want a basic
// initialization routine to be run before yielding control to the test case
// function itself.
//
// The initialization routine is common scaffolding that gets repeated across
// the test plans we've seen. We package it here in an attempt to keep your
// code DRY.
//
// It consists of:
//
//  1. Initializing a sync client, bound to the runenv.
//  2. Initializing a net client.
//  3. Waiting for the network to initialize.
//  4. Claiming a global sequence number.
//  5. Claiming a group-scoped sequence number.
//
// The injected InitContext is a bundle containing the result, and you can use
// its objects in your test logic. In fact, you don't need to close them
// (sync client, net client), as the SDK manages that for you.
type InitializedTestCaseFn = func(env *runtime.RunEnv, initCtx *InitContext) error

// InvokeMap takes a map of test case names and their functions, and calls the
// matched test case, or panics if the name is unrecognised.
//
// Supported function signatures are TestCaseFn and InitializedTestCaseFn.
// Refer to their respective godocs for more info.
func InvokeMap(cases map[string]interface{}) {
	runenv := runtime.CurrentRunEnv()
	defer runenv.Close()

	if fn, ok := cases[runenv.TestCase]; ok {
		invoke(runenv, fn)
	} else {
		msg := fmt.Sprintf("unrecognized test case: %s", runenv.TestCase)
		panic(msg)
	}
}

// Invoke runs the passed test-case and reports the result.
//
// Supported function signatures are TestCaseFn and InitializedTestCaseFn.
// Refer to their respective godocs for more info.
func Invoke(fn interface{}) {
	runenv := runtime.CurrentRunEnv()
	defer runenv.Close()

	invoke(runenv, fn)
}

func invoke(runenv *runtime.RunEnv, fn interface{}) {
	maybeSetupHTTPListener(runenv)

	runenv.RecordStart()

	var closer func()
	defer func() {
		if closer != nil {
			closer()
		}
	}()

	var err error
	errfile, err := runenv.CreateRawAsset("run.err")
	if err != nil {
		runenv.RecordCrash(err)
		return
	}

	rd, wr, err := os.Pipe()
	if err != nil {
		runenv.RecordCrash(err)
		return
	}

	w := io.MultiWriter(errfile, os.Stderr)
	os.Stderr = wr

	// handle the copying of stderr into run.err.
	go func() {
		defer func() {
			_ = rd.Close()
			if sdk.Verbose {
				runenv.RecordMessage("io closed")
			}
		}()

		_, err := io.Copy(w, rd)
		if err != nil && !strings.Contains(err.Error(), "file already closed") {
			runenv.RecordCrash(fmt.Errorf("stderr copy failed: %w", err))
			return
		}

		if err = errfile.Sync(); err != nil {
			runenv.RecordCrash(fmt.Errorf("stderr file tee sync failed failed: %w", err))
		}
	}()

	// Prepare the event.
	defer func() {
		if err := recover(); err != nil {
			// Handle panics by recording them in the runenv output.
			runenv.RecordCrash(err)

			// Developers expect panics to be recorded in run.err too.
			_, _ = fmt.Fprintln(os.Stderr, err)
			debug.PrintStack()
		}
	}()

	closeProfiles, err := captureProfiles(runenv)
	if err != nil {
		runenv.SLogger().Warnw("some or all profile captures failed to initialize", "error", err)
	}
	defer closeProfiles()

	errCh := make(chan error)
	go func() {
		defer close(errCh)
		defer HandlePanics()

		switch f := fn.(type) {
		case TestCaseFn:
			errCh <- f(runenv)
		case InitializedTestCaseFn:
			ic := new(InitContext)
			ic.init(runenv)
			closer = ic.close // we want to close the InitContext after having calld RecordSuccess or RecordFailure
			errCh <- f(runenv, ic)
		default:
			msg := fmt.Sprintf("unexpected function passed to Invoke*; expected types: TestCaseFn, InitializedTestCaseFn; was: %T", f)
			panic(msg)
		}
	}()

	select {
	case err := <-errCh:
		switch err {
		case nil:
			runenv.RecordSuccess()
		default:
			runenv.RecordFailure(err)
		}
	case p := <-panicHandler:
		// propagate the panic.
		runenv.RecordCrash(p.DebugStacktrace)
		panic(p.RecoverObj)
	}
}

type ProfilesCloseFn = func() error

func captureProfiles(runenv *runtime.RunEnv) (ProfilesCloseFn, error) {
	outDir := runenv.TestOutputsPath

	var (
		merr        *multierror.Error
		wg          sync.WaitGroup
		ctx, cancel = context.WithCancel(context.Background())
	)

	ret := func() error {
		// cancel all other profiles, and wait until they have yielded.
		cancel()
		wg.Wait()
		return nil
	}

	for kind, value := range runenv.TestCaptureProfiles {
		switch kind {
		case "cpu":
			runenv.SLogger().Infof("writing cpu profile")

			path := filepath.Join(outDir, "cpu.prof")
			f, err := os.Create(path)
			if err != nil {
				err = fmt.Errorf("failed to create CPU profile output file: %w", err)
				merr = multierror.Append(merr, err)
				continue
			}
			if err = pprof.StartCPUProfile(f); err != nil {
				err = fmt.Errorf("failed to start capturing CPU profile: %w", err)
				merr = multierror.Append(merr, err)
				continue
			}

			wg.Add(1)
			go func() {
				defer wg.Done()

				<-ctx.Done()
				// stop the CPU profile.
				pprof.StopCPUProfile()
				_ = f.Close()
			}()

		default:
			prof := pprof.Lookup(kind)
			if prof == nil {
				merr = multierror.Append(merr, fmt.Errorf("profile of kind %s not recognized; skipped", kind))
				continue
			}
			freq, err := time.ParseDuration(value)
			if err != nil {
				merr = multierror.Append(merr, fmt.Errorf("unparseable duration for profile of kind %s: %s", kind, value))
				continue
			}

			runenv.SLogger().Infof("writing %s profile every %s", kind, freq)

			kind := kind
			wg.Add(1)
			go func() {
				defer wg.Done()

				ticker := _clk.Ticker(freq)
				for {
					select {
					case <-ticker.C:
						path := filepath.Join(outDir, fmt.Sprintf("%s.%s.prof", kind, _clk.Now().Format(time.RFC3339)))
						f, err := os.Create(path)
						if err != nil {
							runenv.SLogger().Warnw("failed to create output file for profile", "kind", kind, "path", path, "error", err)
							continue
						}
						runenv.SLogger().Debugf("writing profile: %s", path)
						if err = prof.WriteTo(f, 0); err != nil {
							runenv.SLogger().Warnw("failed to write profile", "kind", kind, "path", path, "error", err)
							continue
						}
						_ = f.Close()
					case <-ctx.Done():
						return // exiting
					}
				}
			}()

		}
	}

	return ret, merr.ErrorOrNil()
}

func maybeSetupHTTPListener(runenv *runtime.RunEnv) {
	if HTTPListenAddr != "" {
		// already set up.
		return
	}

	addr := fmt.Sprintf("0.0.0.0:%d", HTTPPort)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		addr = fmt.Sprintf("0.0.0.0:%d", HTTPPortFallback)
		if l, err = net.Listen("tcp", addr); err != nil {
			runenv.RecordMessage("error registering default http handler at: %s: %s", addr, err)
			return
		}
	}

	// DefaultServeMux already includes the pprof handler, add the
	// Prometheus handler.
	http.DefaultServeMux.Handle("/metrics", promhttp.Handler())

	HTTPListenAddr = l.Addr().String()

	runenv.RecordMessage("registering default http handler at: http://%s/ (pprof: http://%s/debug/pprof/)", HTTPListenAddr, HTTPListenAddr)

	go func() {
		_ = http.Serve(l, nil)
	}()
}
