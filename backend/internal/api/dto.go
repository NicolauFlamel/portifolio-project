package api

type CreateDocumentDTO struct {
    Channel   string                 `json:"channel,omitempty"`    // optional (default config)
    Org       string                 `json:"org,omitempty"`        // "org1"|"org2", optional
    DocID     string                 `json:"docID"`                // required
    DocType   string                 `json:"docType"`              // e.g. "licitacao" | "verba" | "contrato"
    SourceGov string                 `json:"sourceGov"`            // e.g. "Union"|"State"|"Municipal"
    Payload   map[string]interface{} `json:"payload"`              // free-form content
}

type GetDocumentQuery struct {
    Channel string `form:"channel"`
    Org     string `form:"org"`
}

type ListDocumentsQuery struct {
    Channel string `form:"channel"`
    Org     string `form:"org"`
    Limit   int    `form:"limit"`
}