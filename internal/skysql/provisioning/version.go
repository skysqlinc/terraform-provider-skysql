package provisioning

import "time"

type Version struct {
	Id              string    `json:"id"`
	Name            string    `json:"name"`
	Version         string    `json:"version"`
	Topology        string    `json:"topology"`
	Product         string    `json:"product"`
	DisplayName     string    `json:"display_name"`
	IsMajor         bool      `json:"is_major"`
	ReleaseDate     time.Time `json:"release_date"`
	ReleaseNotesUrl string    `json:"release_notes_url"`
}
