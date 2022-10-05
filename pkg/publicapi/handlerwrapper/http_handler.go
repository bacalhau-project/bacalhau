package handlerwrapper

import (
	"net/http"
	"strings"

	"github.com/felixge/httpsnoop"
)

type HTTPHandlerWrapper struct {
	nodeID             string
	httpHandler        http.Handler
	requestInfoHandler RequestInfoHandler
}

func NewHTTPHandlerWrapper(
	nodeID string,
	httpHandler http.Handler,
	requestInfoHandler RequestInfoHandler) *HTTPHandlerWrapper {
	return &HTTPHandlerWrapper{
		nodeID:             nodeID,
		httpHandler:        httpHandler,
		requestInfoHandler: requestInfoHandler,
	}
}

// An HTTP handler that triggers another handler, capturs info about the request and calls request info handler.
func (wrapper *HTTPHandlerWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ri := &HTTPRequestInfo{
		Method:    r.Method,
		URI:       r.URL.String(),
		Referer:   r.Header.Get("Referer"),
		UserAgent: r.Header.Get("User-Agent"),
		NodeID:    wrapper.nodeID,
	}

	ri.Ipaddr = requestGetRemoteAddress(r)

	// this runs http handler and captures information about HTTP request
	m := httpsnoop.CaptureMetrics(wrapper.httpHandler, w, r)

	ri.StatusCode = m.Code
	ri.Size = m.Written
	ri.Duration = m.Duration.Milliseconds()
	ri.ClientID = w.Header().Get(HTTPHeaderClientID)
	ri.JobID = w.Header().Get(HTTPHeaderJobID)
	wrapper.requestInfoHandler.Handle(r.Context(), ri)
}

// Request.RemoteAddress contains port, which we want to remove i.e.:
// "[::1]:58292" => "[::1]"
func ipAddrFromRemoteAddr(s string) string {
	idx := strings.LastIndex(s, ":")
	if idx == -1 {
		return s
	}
	return s[:idx]
}

// requestGetRemoteAddress returns ip address of the client making the request,
// taking into account http proxies
func requestGetRemoteAddress(r *http.Request) string {
	hdr := r.Header
	hdrRealIP := hdr.Get("X-Real-Ip")
	hdrForwardedFor := hdr.Get("X-Forwarded-For")
	if hdrRealIP == "" && hdrForwardedFor == "" {
		return ipAddrFromRemoteAddr(r.RemoteAddr)
	}
	if hdrForwardedFor != "" {
		// X-Forwarded-For is potentially a list of addresses separated with ","
		parts := strings.Split(hdrForwardedFor, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		// TODO: should return first non-local address
		return parts[0]
	}
	return hdrRealIP
}
