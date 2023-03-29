package provisioning

type AvailabilityZone struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Region   string `json:"region_name"`
	Provider string `json:"provider"`
}
