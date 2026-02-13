package models

type TransferDirection string

const (
	TransferUpload   TransferDirection = "upload"
	TransferDownload TransferDirection = "download"
)

type TransferStatus string

const (
	TransferQueued    TransferStatus = "queued"
	TransferRunning   TransferStatus = "running"
	TransferCompleted TransferStatus = "completed"
	TransferFailed    TransferStatus = "failed"
	TransferCanceled  TransferStatus = "canceled"
)

type TransferItem struct {
	ID               string            `json:"id"`
	SessionID        string            `json:"sessionID"`
	Protocol         ProtocolType      `json:"protocol"`
	Direction        TransferDirection `json:"direction"`
	LocalPath        string            `json:"localPath"`
	RemotePath       string            `json:"remotePath"`
	BytesTotal       int64             `json:"bytesTotal"`
	BytesTransferred int64             `json:"bytesTransferred"`
	StartedAt        int64             `json:"startedAt"`
	FinishedAt       int64             `json:"finishedAt"`
	Status           TransferStatus    `json:"status"`
	Error            string            `json:"error"`
}

type TransfersPayload struct {
	Items []TransferItem `json:"items"`
}
