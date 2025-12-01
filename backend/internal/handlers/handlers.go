package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/gov-spending/backend/internal/config"
	apperrors "github.com/gov-spending/backend/internal/errors"
	"github.com/gov-spending/backend/internal/models"
	"github.com/gov-spending/backend/internal/services"
)

type Handler struct {
	fabricService *services.FabricService
	validChannels map[string]bool
	config        *config.Config
}

func NewHandler(fabricService *services.FabricService, cfg *config.Config) *Handler {
	validChannels := cfg.ValidChannels()
	channelMap := make(map[string]bool)
	for _, ch := range validChannels {
		channelMap[ch] = true
	}
	return &Handler{
		fabricService: fabricService,
		validChannels: channelMap,
		config:        cfg,
	}
}

func (h *Handler) validateChannel(c *gin.Context) (string, bool) {
	channel := c.Param("channel")
	if !h.validChannels[channel] {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error:   "Invalid channel: " + channel,
			Code:    "INVALID_CHANNEL",
		})
		return "", false
	}
	return channel, true
}

func (h *Handler) validateWriteAccess(c *gin.Context, channel string) bool {
	if !h.config.IsAdminChannel(channel) {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Success: false,
			Error:   "Write access denied: this instance does not have admin privileges on channel " + channel,
			Code:    "WRITE_ACCESS_DENIED",
		})
		return false
	}
	return true
}

func (h *Handler) handleError(c *gin.Context, err error) {
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		log.Error().
			Err(err).
			Str("path", c.Request.URL.Path).
			Msg("Unstructured error occurred")

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error:   "An internal error occurred",
			Code:    string(apperrors.ErrCodeInternalError),
		})
		return
	}

	logEvent := log.Error().
		Str("code", string(appErr.Code)).
		Str("message", appErr.Message).
		Str("path", c.Request.URL.Path).
		Bool("retriable", appErr.Retriable).
		Int("httpStatus", appErr.HTTPStatus)

	if appErr.Err != nil {
		logEvent = logEvent.Err(appErr.Err)
	}

	for key, value := range appErr.Context {
		logEvent = logEvent.Interface(key, value)
	}

	logEvent.Msg("Request failed")

	requestID, _ := c.Get("request_id")

	response := models.ErrorResponse{
		Success: false,
		Error:   appErr.Message,
		Code:    string(appErr.Code),
	}

	if appErr.Details != "" {
		response.Details = appErr.Details
	}

	if reqID, ok := requestID.(string); ok {
		response.RequestID = reqID
	}

	if len(appErr.Context) > 0 {
		response.Context = make(map[string]any)
		safeFields := []string{"channel", "operation", "step", "documentTypeId", "typeId"}
		for _, field := range safeFields {
			if val, exists := appErr.Context[field]; exists {
				response.Context[field] = val
			}
		}
		response.Context["retriable"] = appErr.Retriable
	}


	c.JSON(appErr.HTTPStatus, response)
}

// HealthCheck godoc
// @Summary      Health check
// @Description  Check if the API is running
// @Tags         Health
// @Produce      json
// @Success      200  {object}  map[string]string  "status: healthy"
// @Router       /health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "gov-spending-api",
	})
}

// ConfigInfo godoc
// @Summary      Configuration info
// @Description  Show writable channels for this backend instance
// @Tags         Health
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /config [get]
func (h *Handler) ConfigInfo(c *gin.Context) {
	writableChannels := h.config.GetWritableChannels()
	allChannels := h.config.ValidChannels()

	channelDetails := make(map[string]interface{})
	for _, ch := range allChannels {
		cfg, _ := h.config.GetChannelConfig(ch)
		channelDetails[ch] = gin.H{
			"userName":   cfg.UserName,
			"isAdmin":    h.config.IsAdminChannel(ch),
			"channelName": cfg.Name,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"writableChannels": writableChannels,
		"allChannels":      allChannels,
		"channelDetails":   channelDetails,
	})
}

// =============================================================================
// Document Types
// =============================================================================

// RegisterDocumentType godoc
// @Summary      Register document type
// @Description  Create a new document type template with required/optional fields
// @Tags         Document Types
// @Accept       json
// @Produce      json
// @Param        channel  path      string                                true  "Channel (union, state, region)"
// @Param        request  body      models.CreateDocumentTypeRequest      true  "Document type data"
// @Success      201      {object}  models.IDResponse
// @Failure      400      {object}  models.ErrorResponse
// @Failure      500      {object}  models.ErrorResponse
// @Router       /api/{channel}/document-types [post]
func (h *Handler) RegisterDocumentType(c *gin.Context) {
	channel, ok := h.validateChannel(c)
	if !ok {
		return
	}

	// Check write access
	if !h.validateWriteAccess(c, channel) {
		return
	}

	var req models.CreateDocumentTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErr := apperrors.NewValidationError("Invalid request body: " + err.Error())
		h.handleError(c, validationErr)
		return
	}

	result, err := h.fabricService.RegisterDocumentType(channel, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}


// GetDocumentType godoc
// @Summary      Get document type
// @Description  Get specific document type by ID
// @Tags         Document Types
// @Produce      json
// @Param        channel  path      string  true  "Channel (union, state, region)"
// @Param        typeId   path      string  true  "Document type ID"
// @Success      200      {object}  models.DocumentType
// @Failure      404      {object}  models.ErrorResponse
// @Router       /api/{channel}/document-types/{typeId} [get]
func (h *Handler) GetDocumentType(c *gin.Context) {
	channel, ok := h.validateChannel(c)
	if !ok {
		return
	}

	typeID := c.Param("typeId")

	result, err := h.fabricService.GetDocumentType(channel, typeID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListDocumentTypes godoc
// @Summary      List document types
// @Description  Get all document types for a channel
// @Tags         Document Types
// @Produce      json
// @Param        channel  path      string  true  "Channel (union, state, region)"
// @Success      200      {array}   models.DocumentType
// @Router       /api/{channel}/document-types [get]
func (h *Handler) ListDocumentTypes(c *gin.Context) {
	channel, ok := h.validateChannel(c)
	if !ok {
		return
	}

	orgID := c.Query("organizationId")

	result, err := h.fabricService.ListDocumentTypes(channel, orgID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"documentTypes": result,
		"total":         len(result),
	})
}

// DeactivateDocumentType godoc
// @Summary      Deactivate document type
// @Description  Mark document type as inactive (no new documents allowed)
// @Tags         Document Types
// @Produce      json
// @Param        channel  path      string  true  "Channel (union, state, region)"
// @Param        typeId   path      string  true  "Document type ID"
// @Success      200      {object}  models.SuccessResponse
// @Failure      500      {object}  models.ErrorResponse
// @Router       /api/{channel}/document-types/{typeId} [delete]
func (h *Handler) DeactivateDocumentType(c *gin.Context) {
	channel, ok := h.validateChannel(c)
	if !ok {
		return
	}

	// Check write access
	if !h.validateWriteAccess(c, channel) {
		return
	}

	typeID := c.Param("typeId")

	if err := h.fabricService.DeactivateDocumentType(channel, typeID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "Document type deactivated",
	})
}

// CreateDocument godoc
// @Summary      Create document
// @Description  Create a new spending document (contractor payment, equipment, etc.)
// @Tags         Documents
// @Accept       json
// @Produce      json
// @Param        channel  path      string                        true  "Channel (union, state, region)"
// @Param        request  body      models.CreateDocumentRequest  true  "Document data"
// @Success      201      {object}  models.Document
// @Failure      400      {object}  models.ErrorResponse
// @Failure      500      {object}  models.ErrorResponse
// @Router       /api/{channel}/documents [post]
func (h *Handler) CreateDocument(c *gin.Context) {
	channel, ok := h.validateChannel(c)
	if !ok {
		return
	}

	// Check write access
	if !h.validateWriteAccess(c, channel) {
		return
	}

	var req models.CreateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErr := apperrors.NewValidationError("Invalid request body: " + err.Error())
		h.handleError(c, validationErr)
		return
	}

	result, err := h.fabricService.CreateDocument(channel, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetDocument godoc
// @Summary      Get document
// @Description  Get specific document by ID with all details
// @Tags         Documents
// @Produce      json
// @Param        channel  path      string  true  "Channel (union, state, region)"
// @Param        docId    path      string  true  "Document ID"
// @Success      200      {object}  models.Document
// @Failure      404      {object}  models.ErrorResponse
// @Router       /api/{channel}/documents/{docId} [get]
func (h *Handler) GetDocument(c *gin.Context) {
	channel, ok := h.validateChannel(c)
	if !ok {
		return
	}

	docID := c.Param("docId")

	result, err := h.fabricService.GetDocument(channel, docID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// QueryDocuments godoc
// @Summary      Query documents
// @Description  Search and filter documents with pagination
// @Tags         Documents
// @Produce      json
// @Param        channel          path      string  true   "Channel (union, state, region)"
// @Param        organizationId   query     string  false  "Filter by organization"
// @Param        documentTypeId   query     string  false  "Filter by document type"
// @Param        status           query     string  false  "Filter by status (ACTIVE, INVALIDATED)"  Enums(ACTIVE, INVALIDATED)
// @Param        fromDate         query     string  false  "From date (ISO 8601)"
// @Param        toDate           query     string  false  "To date (ISO 8601)"
// @Param        minAmount        query     number  false  "Minimum amount"
// @Param        maxAmount        query     number  false  "Maximum amount"
// @Param        hasLinkedDoc     query     bool    false  "Has linked document"
// @Param        linkedDirection  query     string  false  "Link direction"  Enums(OUTGOING, INCOMING)
// @Param        pageSize         query     int     false  "Page size"  default(20)
// @Param        bookmark         query     string  false  "Pagination bookmark"
// @Success      200              {object}  models.QueryResult
// @Router       /api/{channel}/documents [get]
func (h *Handler) QueryDocuments(c *gin.Context) {
	channel, ok := h.validateChannel(c)
	if !ok {
		return
	}

	var filter models.QueryFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		validationErr := apperrors.NewValidationError("Invalid query parameters: " + err.Error())
		h.handleError(c, validationErr)
		return
	}

	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	result, err := h.fabricService.QueryDocuments(channel, &filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}


// InvalidateDocument godoc
// @Summary      Invalidate document
// @Description  Mark document as invalid (immutable - never deleted). Used for error correction.
// @Tags         Documents
// @Accept       json
// @Produce      json
// @Param        channel  path      string                            true  "Channel (union, state, region)"
// @Param        docId    path      string                            true  "Document ID"
// @Param        request  body      models.InvalidateDocumentRequest  true  "Invalidation reason and correction doc"
// @Success      200      {object}  models.SuccessResponse
// @Failure      400      {object}  models.ErrorResponse
// @Failure      403      {object}  models.ErrorResponse  "Only creating organization can invalidate"
// @Failure      500      {object}  models.ErrorResponse
// @Router       /api/{channel}/documents/{docId}/invalidate [post]
func (h *Handler) InvalidateDocument(c *gin.Context) {
	channel, ok := h.validateChannel(c)
	if !ok {
		return
	}

	// Check write access
	if !h.validateWriteAccess(c, channel) {
		return
	}

	docID := c.Param("docId")

	var req models.InvalidateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErr := apperrors.NewValidationError("Invalid request body: " + err.Error())
		h.handleError(c, validationErr)
		return
	}

	if err := h.fabricService.InvalidateDocument(channel, docID, &req); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "Document invalidated",
	})
}

// GetDocumentHistory godoc
// @Summary      Get document history
// @Description  Get complete transaction history for a document (all versions)
// @Tags         Documents
// @Produce      json
// @Param        channel  path      string  true  "Channel (union, state, region)"
// @Param        docId    path      string  true  "Document ID"
// @Success      200      {array}   models.HistoryEntry
// @Failure      404      {object}  models.ErrorResponse
// @Router       /api/{channel}/documents/{docId}/history [get]
func (h *Handler) GetDocumentHistory(c *gin.Context) {
	channel, ok := h.validateChannel(c)
	if !ok {
		return
	}

	docID := c.Param("docId")

	result, err := h.fabricService.GetDocumentHistory(channel, docID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": result,
		"total":   len(result),
	})
}

// GetLinkedDocuments godoc
// @Summary      Get document with linked document
// @Description  Get document and its cross-channel linked document (if any)
// @Tags         Documents
// @Produce      json
// @Param        channel  path      string  true  "Channel (union, state, region)"
// @Param        docId    path      string  true  "Document ID"
// @Success      200      {object}  models.LinkedDocuments
// @Failure      404      {object}  models.ErrorResponse
// @Router       /api/{channel}/documents/{docId}/linked [get]
func (h *Handler) GetLinkedDocuments(c *gin.Context) {
	channel, ok := h.validateChannel(c)
	if !ok {
		return
	}

	docID := c.Param("docId")

	result, err := h.fabricService.GetLinkedDocuments(channel, docID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// InitiateTransfer godoc
// @Summary      Initiate cross-channel transfer
// @Description  Start an inter-government transfer (e.g., Federal → State). Creates document with cryptographic hash.
// @Tags         Transfers
// @Accept       json
// @Produce      json
// @Param        request  body      models.InitiateTransferRequest  true  "Transfer details"
// @Success      201      {object}  models.TransferResult
// @Failure      400      {object}  models.ErrorResponse
// @Failure      500      {object}  models.ErrorResponse
// @Router       /api/transfers/initiate [post]
func (h *Handler) InitiateTransfer(c *gin.Context) {
	var req models.InitiateTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErr := apperrors.NewValidationError("Invalid request body: " + err.Error())
		h.handleError(c, validationErr)
		return
	}

	if !h.validChannels[req.FromChannel] {
		channelErr := apperrors.NewInvalidChannelError(req.FromChannel)
		h.handleError(c, channelErr)
		return
	}
	if !h.validChannels[req.ToChannel] {
		channelErr := apperrors.NewInvalidChannelError(req.ToChannel)
		h.handleError(c, channelErr)
		return
	}

	// Check write access on FromChannel (where the transfer originates)
	if !h.config.IsAdminChannel(req.FromChannel) {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Success: false,
			Error:   "Write access denied: this instance does not have admin privileges on channel " + req.FromChannel,
			Code:    "WRITE_ACCESS_DENIED",
		})
		return
	}

	result, err := h.fabricService.InitiateTransfer(&req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}


// AcknowledgeTransfer godoc
// @Summary      Acknowledge transfer
// @Description  Acknowledge a received cross-channel transfer. Creates linked document on target channel.
// @Tags         Transfers
// @Accept       json
// @Produce      json
// @Param        channel  path      string                              true  "Target channel (union, state, region)"
// @Param        request  body      models.AcknowledgeTransferRequest   true  "Acknowledgment details"
// @Success      201      {object}  models.TransferResult
// @Failure      400      {object}  models.ErrorResponse
// @Failure      500      {object}  models.ErrorResponse
// @Router       /api/{channel}/transfers/acknowledge [post]
func (h *Handler) AcknowledgeTransfer(c *gin.Context) {
	channel, ok := h.validateChannel(c)
	if !ok {
		return
	}

	// Check write access
	if !h.validateWriteAccess(c, channel) {
		return
	}

	var req models.AcknowledgeTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErr := apperrors.NewValidationError("Invalid request body: " + err.Error())
		h.handleError(c, validationErr)
		return
	}

	if !h.validChannels[req.SourceChannel] {
		channelErr := apperrors.NewInvalidChannelError(req.SourceChannel)
		h.handleError(c, channelErr)
		return
	}

	result, err := h.fabricService.AcknowledgeTransfer(channel, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

// VerifyAnchor godoc
// @Summary      Verify cross-channel link
// @Description  Verify that two documents are properly linked across channels using cryptographic hashes
// @Tags         Verification
// @Accept       json
// @Produce      json
// @Param        request  body      models.VerifyAnchorRequest  true  "Source and target document info"
// @Success      200      {object}  models.AnchorVerification
// @Failure      400      {object}  models.ErrorResponse
// @Router       /api/anchors/verify [post]
func (h *Handler) VerifyAnchor(c *gin.Context) {
	var req models.VerifyAnchorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErr := apperrors.NewValidationError("Invalid request body: " + err.Error())
		h.handleError(c, validationErr)
		return
	}

	// Validate channels
	if !h.validChannels[req.SourceChannel] {
		channelErr := apperrors.NewInvalidChannelError(req.SourceChannel)
		h.handleError(c, channelErr)
		return
	}
	if !h.validChannels[req.TargetChannel] {
		channelErr := apperrors.NewInvalidChannelError(req.TargetChannel)
		h.handleError(c, channelErr)
		return
	}

	result, err := h.fabricService.VerifyAnchor(
		req.SourceChannel,
		req.SourceDocID,
		req.TargetChannel,
		req.TargetDocID,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}