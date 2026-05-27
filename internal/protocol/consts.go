// Package protocol holds the LocalSend v2.1 wire types and constants. It
// performs no I/O so it can be unit-tested against captured payloads.
package protocol

// Network defaults from the LocalSend v2.1 spec.
const (
	MulticastAddr   = "224.0.0.167"
	MulticastPort   = 53317
	DefaultPort     = 53317
	ProtocolVersion = "2.1"
)

// HTTP API paths, all served under the device's port.
const (
	PathRegister      = "/api/localsend/v2/register"
	PathInfo          = "/api/localsend/v2/info"
	PathPrepareUpload = "/api/localsend/v2/prepare-upload"
	PathUpload        = "/api/localsend/v2/upload"
	PathCancel        = "/api/localsend/v2/cancel"
)
