package provisioning

type UpdateServiceNodesNumberRequest struct {
	Nodes         int64  `json:"nodes,omitempty"`
	MaxscaleNodes *int64 `json:"maxscale_nodes,omitempty"`
}
