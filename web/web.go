package web

const (

	// ContentTypeMetaMessage is the Content-Type for MetaMessage binary format.
	ContentTypeMetaMessage = "application/metamessage"

	// ContentTypeJSONC is the Content-Type for JSONC format.
	ContentTypeJSONC = "application/jsonc"
)

// MMError represents an error response in MetaMessage format.
type MMError struct {
	Error string `mm:"desc=Error info"`
}

// MMResp wraps response data with an optional mm tag for MetaMessage encoding.
type MMResp struct {
	Data any
	Tag  string
}
