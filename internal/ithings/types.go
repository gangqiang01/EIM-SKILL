package ithings

type setCollectionStatusRequest struct {
	Status bool `json:"status"`
}

// DataForwardSource selects which devices' data feeds a DataForward rule.
// See common/types/v1/data_forward_types.go.
type DataForwardSource struct {
	Type     string `json:"type"` // "model" (by device model) is the only fully-wired type server-side
	EdgeId   string `json:"edgeId,omitempty"`
	ModelId  *int64 `json:"modelId,omitempty"`
	DeviceId string `json:"deviceId,omitempty"`
}

// DataForwardDestination is the forwarding target. Fields relevant to the
// destination Type ("amqp"/"kafka"/"mqtt") apply; others are ignored.
type DataForwardDestination struct {
	Type     string `json:"type"`
	Host     string `json:"host,omitempty"`
	UserName string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Topic    string `json:"topic,omitempty"` // kafka topic or mqtt topic

	// amqp
	QueueName string `json:"queueName,omitempty"`

	// mqtt
	ClientId string `json:"clientId,omitempty"`
	QOS      int    `json:"qos,omitempty"`
	Retain   bool   `json:"retain,omitempty"`
}

type createDataForwardRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateDataForwardRequest struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Source      []DataForwardSource `json:"source,omitempty"`
	Destination string              `json:"destination"`
}

type updateDataForwardStatusRequest struct {
	Status string `json:"status"` // "enable" or "disable"
}
