package models

import "time"

// =============================================================================
// Document Types
// =============================================================================

type DocumentType struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organizationId"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	RequiredFields []string `json:"requiredFields"`
	OptionalFields []string `json:"optionalFields"`
	CreatedAt      string   `json:"createdAt"`
	CreatedBy      string   `json:"createdBy"`
	IsActive       bool     `json:"isActive"`
}

type CreateDocumentTypeRequest struct {
	ID             string   `json:"id" binding:"required"`
	Name           string   `json:"name" binding:"required"`
	Description    string   `json:"description"`
	RequiredFields []string `json:"requiredFields"`
	OptionalFields []string `json:"optionalFields"`
}

// =============================================================================
// Documents
// =============================================================================

type DocumentStatus string

const (
	StatusActive      DocumentStatus = "ACTIVE"
	StatusInvalidated DocumentStatus = "INVALIDATED"
)

// Document represents a spending document (simplified - no complex Anchor struct)
// All fields are always present (Fabric contract schema requirement)
type Document struct {
	ID             string                 `json:"id"`
	DocumentTypeID string                 `json:"documentTypeId"`
	OrganizationID string                 `json:"organizationId"`
	ChannelID      string                 `json:"channelId"`
	Status         DocumentStatus         `json:"status"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Amount         float64                `json:"amount"`
	Currency       string                 `json:"currency"`
	Data           map[string]interface{} `json:"data"`
	ContentHash    string                 `json:"contentHash"`

	LinkedDocID     string `json:"linkedDocId"`
	LinkedChannel   string `json:"linkedChannel"`
	LinkedDocHash   string `json:"linkedDocHash"`
	LinkedDirection string `json:"linkedDirection"`

	InvalidatedBy  string `json:"invalidatedBy"`
	InvalidatedAt  string `json:"invalidatedAt"`
	InvalidReason  string `json:"invalidReason"`
	CorrectedByDoc string `json:"correctedByDoc"`

	CreatedAt string   `json:"createdAt"`
	CreatedBy string   `json:"createdBy"`
	UpdatedAt string   `json:"updatedAt"`
	UpdatedBy string   `json:"updatedBy"`
	History   []string `json:"history"`
}

type CreateDocumentRequest struct {
	ID             string                 `json:"id"`
	DocumentTypeID string                 `json:"documentTypeId" binding:"required"`
	Title          string                 `json:"title" binding:"required"`
	Description    string                 `json:"description"`
	Amount         float64                `json:"amount"`
	Currency       string                 `json:"currency"`
	Data           map[string]interface{} `json:"data"`
}

type InvalidateDocumentRequest struct {
	Reason          string `json:"reason" binding:"required"`
	CorrectionDocID string `json:"correctionDocId"`
}

// =============================================================================
// Query and Filtering
// =============================================================================

type QueryFilter struct {
	OrganizationID  string         `json:"organizationId,omitempty" form:"organizationId"`
	DocumentTypeID  string         `json:"documentTypeId,omitempty" form:"documentTypeId"`
	Status          DocumentStatus `json:"status,omitempty" form:"status"`
	FromDate        string         `json:"fromDate,omitempty" form:"fromDate"`
	ToDate          string         `json:"toDate,omitempty" form:"toDate"`
	MinAmount       float64        `json:"minAmount,omitempty" form:"minAmount"`
	MaxAmount       float64        `json:"maxAmount,omitempty" form:"maxAmount"`
	HasLinkedDoc    *bool          `json:"hasLinkedDoc,omitempty" form:"hasLinkedDoc"`
	LinkedDirection string         `json:"linkedDirection,omitempty" form:"linkedDirection"`
	PageSize        int            `json:"pageSize,omitempty" form:"pageSize"`
	Bookmark        string         `json:"bookmark,omitempty" form:"bookmark"`
}

type QueryResult struct {
	Documents []*Document `json:"documents"`
	Bookmark  string      `json:"bookmark,omitempty"`
	Total     int         `json:"total"`
}

// =============================================================================
// Cross-Channel Transfers
// =============================================================================

// InitiateTransferRequest for starting a cross-channel transfer
type InitiateTransferRequest struct {
	FromChannel    string                 `json:"fromChannel" binding:"required"`
	ToChannel      string                 `json:"toChannel" binding:"required"`
	ToOrg          string                 `json:"toOrg" binding:"required"`
	DocumentTypeID string                 `json:"documentTypeId" binding:"required"`
	Title          string                 `json:"title" binding:"required"`
	Description    string                 `json:"description"`
	Amount         float64                `json:"amount" binding:"required"`
	Currency       string                 `json:"currency"`
	Data           map[string]interface{} `json:"data"`
}

// AcknowledgeTransferRequest for acknowledging a received transfer
type AcknowledgeTransferRequest struct {
	SourceDocID    string                 `json:"sourceDocId" binding:"required"`
	SourceChannel  string                 `json:"sourceChannel" binding:"required"`
	DocumentTypeID string                 `json:"documentTypeId" binding:"required"`
	Title          string                 `json:"title" binding:"required"`
	Description    string                 `json:"description"`
	Data           map[string]interface{} `json:"data"`
}

// TransferResult is returned after transfer operations
type TransferResult struct {
	Success          bool   `json:"success"`
	ID               string `json:"id"`
	ContentHash      string `json:"contentHash"`
	Channel          string `json:"channel"`
	LinkedDocID      string `json:"linkedDocId,omitempty"`
	LinkedDocHash    string `json:"linkedDocHash,omitempty"`
	LinkedDocChannel string `json:"linkedDocChannel,omitempty"`
}

// =============================================================================
// Anchor Verification
// =============================================================================

// AnchorVerification result of verifying cross-channel link
type AnchorVerification struct {
	SourceDocID    string  `json:"sourceDocId"`
	SourceChannel  string  `json:"sourceChannel"`
	SourceHash     string  `json:"sourceHash"`
	SourceAmount   float64 `json:"sourceAmount"`
	SourceCurrency string  `json:"sourceCurrency"`

	TargetDocID    string  `json:"targetDocId"`
	TargetChannel  string  `json:"targetChannel"`
	TargetHash     string  `json:"targetHash"`
	TargetAmount   float64 `json:"targetAmount"`
	TargetCurrency string  `json:"targetCurrency"`

	HashMatch      bool     `json:"hashMatch"`
	IDMatch        bool     `json:"idMatch"`
	ChannelMatch   bool     `json:"channelMatch"`
	AmountMatch    bool     `json:"amountMatch"`
	IsValid        bool     `json:"isValid"`
	Status         string   `json:"status"` // "VERIFIED" or "MISMATCH"
	MismatchReason []string `json:"mismatchReason,omitempty"`
}

// LinkedDocuments returns both sides of a cross-channel link
type LinkedDocuments struct {
	Document       *Document `json:"document"`
	LinkedDocument *Document `json:"linkedDocument,omitempty"`
	LinkVerified   bool      `json:"linkVerified"`
}

// VerifyAnchorRequest for verifying cross-channel links
type VerifyAnchorRequest struct {
	SourceChannel string `json:"sourceChannel" binding:"required"`
	SourceDocID   string `json:"sourceDocId" binding:"required"`
	TargetChannel string `json:"targetChannel" binding:"required"`
	TargetDocID   string `json:"targetDocId" binding:"required"`
}

// =============================================================================
// Document History
// =============================================================================

type HistoryEntry struct {
	TxID      string    `json:"txId"`
	Timestamp time.Time `json:"timestamp"`
	IsDelete  bool      `json:"isDelete"`
	Document  *Document `json:"document"`
}

// =============================================================================
// API Responses
// =============================================================================

type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
    Success   bool                   `json:"success"`
    Error     string                 `json:"error"`
    Details   string                 `json:"details,omitempty"`
    Code      string                 `json:"code,omitempty"`
    RequestID string                 `json:"request_id,omitempty"`
    Context   map[string]interface{} `json:"context,omitempty"`
}

type IDResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
}