package provisioning

// ServiceEndpoint is service endpoint dto
type ServiceEndpoint struct {
	Mechanism       string   `json:"mechanism,omitempty"`
	AllowedAccounts []string `json:"allowed_accounts,omitempty"`
	Visibility      string   `json:"visibility"`
	EndpointService string   `json:"endpoint_service,omitempty"`
}

// PatchServiceEndpointsRequest godoc
type PatchServiceEndpointsRequest []ServiceEndpoint

// PatchServiceEndpointsResponse godoc
type PatchServiceEndpointsResponse []ServiceEndpoint
