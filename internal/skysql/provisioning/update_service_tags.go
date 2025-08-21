package provisioning

type UpdateServiceTagsRequest struct {
	// Tags to be set for the service
	Tags map[string]string `json:"tags" example:"{\"name\": \"new-service-name\"}"`
}