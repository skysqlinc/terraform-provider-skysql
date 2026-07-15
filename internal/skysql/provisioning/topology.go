package provisioning

type Topology struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DisplayName   string `json:"display_name,omitempty"`
	ServiceType   string `json:"service_type"`
	StorageEngine string `json:"storage_engine,omitempty"`
	Order         int    `json:"order,omitempty"`
	IsDefault     bool   `json:"is_default"`
	Database      string `json:"database,omitempty"`
}
