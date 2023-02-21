package publicapi

import (
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
)

var LINESOFLOGTOPRINT = 100

func GenerateHealthData() types.HealthInfo {
	var healthInfo types.HealthInfo

	// Generating all, free, used amounts for each - in case these are different mounts, they'll have different
	// All and Free values, if they're all on the same machine, then those values should be the same
	// If "All" is 0, that means the directory does not exist
	healthInfo.DiskFreeSpace.IPFSMount = MountUsage("/data/ipfs")
	healthInfo.DiskFreeSpace.ROOT = MountUsage("/")
	healthInfo.DiskFreeSpace.TMP = MountUsage("/tmp")

	return healthInfo
}

// livez godoc
//
//	@ID			livez
//	@Tags		Utils
//	@Produce	text/plain
//	@Success	200	{object}	string	"TODO"
//	@Router		/livez [get]
func (apiServer *APIServer) livez(res http.ResponseWriter, req *http.Request) {
	// Extremely simple liveness check (should be fine to be public / no-auth)
	log.Ctx(req.Context()).Debug().Msg("Received OK request")
	res.Header().Add("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)
	_, err := res.Write([]byte("OK"))
	if err != nil {
		log.Ctx(req.Context()).Warn().Err(err).Msg("Error writing body for OK request.")
	}
}

// logz godoc
//
//	@ID			logz
//	@Tags		Utils
//	@Produce	text/plain
//	@Success	200	{object}	string	"TODO"
//	@Router		/logz [get]
func (apiServer *APIServer) logz(res http.ResponseWriter, req *http.Request) {
	log.Ctx(req.Context()).Debug().Msg("Received logz request")
	res.Header().Add("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)
	fileOutput, err := TailFile(LINESOFLOGTOPRINT, "/tmp/ipfs.log")
	if err != nil {
		missingLogFileMsg := "File not found at /tmp/ipfs.log"
		log.Ctx(req.Context()).Warn().Msgf(missingLogFileMsg)
		_, err = res.Write([]byte("File not found at /tmp/ipfs.log"))
		if err != nil {
			log.Ctx(req.Context()).Warn().Msgf("Could not write response body for missing log file to response.")
		}
		return
	}
	_, err = res.Write(fileOutput)
	if err != nil {
		log.Ctx(req.Context()).Warn().Msg("Error writing body for logz request.")
	}
}

// readyz godoc
//
//	@ID			readyz
//	@Tags		Utils
//	@Produce	text/plain
//	@Success	200	{object}	string
//	@Router		/readyz [get]
func (apiServer *APIServer) readyz(res http.ResponseWriter, req *http.Request) {
	log.Ctx(req.Context()).Debug().Msg("Received readyz request.")
	// TODO: Add checker for queue that this node can accept submissions
	isReady := true

	res.Header().Add("Content-Type", "text/plain")
	if !isReady {
		res.WriteHeader(http.StatusServiceUnavailable)
	}
	res.WriteHeader(http.StatusOK)
	_, err := res.Write([]byte("READY"))
	if err != nil {
		log.Ctx(req.Context()).Warn().Msg("Error writing body for readyz request.")
	}
}

// healthz godoc
//
//	@ID			healthz
//	@Tags		Utils
//	@Produce	json
//	@Success	200	{object}	types.HealthInfo
//	@Router		/healthz [get]
func (apiServer *APIServer) healthz(res http.ResponseWriter, req *http.Request) {
	// TODO: A list of health information. Should require authing (of some kind)
	log.Ctx(req.Context()).Debug().Msg("Received healthz request.")
	res.Header().Add("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

	// Ideas:
	// CPU usage

	healthInfo := GenerateHealthData()
	healthJSONBlob, _ := model.JSONMarshalWithMax(healthInfo)

	_, err := res.Write(healthJSONBlob)
	if err != nil {
		log.Ctx(req.Context()).Warn().Msg("Error writing body for healthz request.")
	}
}

// varz godoc
//
//	@ID			varz
//	@Tags		Utils
//	@Produce	json
//	@Success	200	{object}	json.RawMessage
//	@Router		/varz [get]
func (apiServer *APIServer) varz(res http.ResponseWriter, req *http.Request) {
	// TODO: Fill in with the configuration settings for this node
	res.WriteHeader(http.StatusOK)
	res.Header().Add("Content-Type", "application/json")

	_, err := res.Write([]byte("{}"))
	if err != nil {
		log.Ctx(req.Context()).Warn().Msg("Error writing body for varz request.")
	}
}
