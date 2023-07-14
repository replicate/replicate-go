package replicate

type Model struct {
	URL            string        `json:"url"`
	Owner          string        `json:"owner"`
	Name           string        `json:"name"`
	Description    *string       `json:"description,omitempty"`
	Visibility     string        `json:"visibility"`
	GithubURL      *string       `json:"github_url,omitempty"`
	PaperURL       *string       `json:"paper_url,omitempty"`
	LicenseURL     *string       `json:"license_url,omitempty"`
	RunCount       int           `json:"run_count"`
	CoverImageURL  *string       `json:"cover_image_url,omitempty"`
	DefaultExample *Prediction   `json:"default_example,omitempty"`
	LatestVersion  *ModelVersion `json:"latest_version,omitempty"`
}

type ModelVersion struct {
	ID            string      `json:"id"`
	CreatedAt     string      `json:"created_at"`
	CogVersion    string      `json:"cog_version"`
	OpenAPISchema interface{} `json:"openapi_schema"`
}
