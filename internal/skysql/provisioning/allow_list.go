package provisioning

type AllowListItem struct {
	IPAddress string `json:"ip"`
	Comment   string `json:"comment"`
}

type ReadAllowListResponse []struct {
	AllowList []AllowListItem `json:"allow_list"`
}
