package provisioning

type UpdateStorageRequest struct {
	Size       int64 `json:"size,omitempty"`
	IOPS       int64 `json:"iops,omitempty"`
	Throughput int64 `json:"throughput,omitempty"`
}
