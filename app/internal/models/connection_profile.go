package models

type ProtocolType string

const (
	ProtocolFTP    ProtocolType = "ftp"
	ProtocolSFTP   ProtocolType = "sftp"
	ProtocolS3     ProtocolType = "s3"
	ProtocolWebDAV ProtocolType = "webdav"
	ProtocolSMB    ProtocolType = "smb"
	ProtocolNFS    ProtocolType = "nfs"
)

type ConnectionProfile struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Protocol          ProtocolType      `json:"protocol"`
	Host              string            `json:"host"`
	Port              int               `json:"port"`
	AuthType          string            `json:"authType"`
	Credentials       map[string]string `json:"credentials,omitempty"`
	Path              string            `json:"path"`
	Metadata          map[string]any    `json:"metadata,omitempty"`
	Username          string            `json:"username,omitempty"`
	CredentialsMasked map[string]bool   `json:"credentialsMasked,omitempty"`
	CreatedAt         int64             `json:"createdAt"`
	UpdatedAt         int64             `json:"updatedAt"`
}
