package protocol

// DeviceType is the kind of device, as shown (with an icon) in peer lists.
type DeviceType string

const (
	DeviceMobile   DeviceType = "mobile"
	DeviceDesktop  DeviceType = "desktop"
	DeviceWeb      DeviceType = "web"
	DeviceHeadless DeviceType = "headless"
	DeviceServer   DeviceType = "server"
)

// DeviceInfo is the announcement / handshake payload. It is sent over multicast
// (with Announce set), returned by GET /info, and exchanged on POST /register.
//
// Announce is a pointer so we can distinguish three states on the wire:
//   - nil   → field omitted (e.g. /info responses)
//   - true  → a discovery probe expecting replies
//   - false → a reply to someone else's probe
type DeviceInfo struct {
	Alias       string     `json:"alias"`
	Version     string     `json:"version"`
	DeviceModel string     `json:"deviceModel,omitempty"`
	DeviceType  DeviceType `json:"deviceType,omitempty"`
	Fingerprint string     `json:"fingerprint"`
	Port        int        `json:"port"`
	Protocol    string     `json:"protocol"`
	Download    bool       `json:"download,omitempty"`
	Announce    *bool      `json:"announce,omitempty"`
}

// WithAnnounce returns a copy of d with the Announce flag set to v.
func (d DeviceInfo) WithAnnounce(v bool) DeviceInfo {
	d.Announce = &v
	return d
}
