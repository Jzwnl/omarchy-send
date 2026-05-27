package protocol

// FileMetadata describes one file offered in a prepare-upload request. SHA256
// and Preview are optional; the spec permits omitting the hash to avoid
// pre-hashing large files.
type FileMetadata struct {
	ID       string          `json:"id"`
	FileName string          `json:"fileName"`
	Size     int64           `json:"size"`
	FileType string          `json:"fileType"`
	SHA256   string          `json:"sha256,omitempty"`
	Preview  string          `json:"preview,omitempty"`
	Metadata *FileTimestamps `json:"metadata,omitempty"`
}

// FileTimestamps carries optional modified/accessed times (RFC 3339 strings).
type FileTimestamps struct {
	Modified string `json:"modified,omitempty"`
	Accessed string `json:"accessed,omitempty"`
}

// PrepareUploadRequest is the body POSTed to /prepare-upload by the sender.
// Files is keyed by fileId.
type PrepareUploadRequest struct {
	Info  DeviceInfo              `json:"info"`
	Files map[string]FileMetadata `json:"files"`
}

// PrepareUploadResponse is returned by the receiver: a session id plus a
// single-use token per fileId.
type PrepareUploadResponse struct {
	SessionID string            `json:"sessionId"`
	Files     map[string]string `json:"files"`
}
