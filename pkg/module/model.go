package module

// Model defines model for Model.
type Model struct {
	// Etag the oss etag of the model
	Etag string `json:"etag"`

	// LastModificationTime the last modification time of the model
	LastModificationTime *string `json:"lastModificationTime,omitempty"`

	// Name model name
	Name string `json:"name"`

	// OssPath the oss path of the model
	OssPath string `json:"ossPath"`

	// RegisteredTime the registered time of the model
	RegisteredTime *string `json:"registeredTime,omitempty"`

	// Status the model status, registering, loading, loaded or unloaded
	Status string `json:"status"`

	// Type model type
	Type string `json:"type"`
}

// ModelBase defines model for ModelBase.
type ModelBase struct {
	// Name model name
	Name string `json:"name"`

	// OssPath the oss path of the model
	OssPath string `json:"ossPath"`

	// Type model type
	Type string `json:"type"`
}

// UploadModelJSONRequestBody defines body for UploadModel for application/json ContentType.
type UploadModelJSONRequestBody = ModelBase

// UpdateModelJSONRequestBody defines body for UpdateModel for application/json ContentType.
type UpdateModelJSONRequestBody = Model
