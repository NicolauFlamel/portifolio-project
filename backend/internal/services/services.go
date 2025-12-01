package services

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/gov-spending/backend/internal/errors"
	"github.com/gov-spending/backend/internal/models"
	"github.com/gov-spending/backend/pkg/fabric"
)

type FabricService struct {
	gateway *fabric.GatewayManager
}

func NewFabricService(gateway *fabric.GatewayManager) *FabricService {
	return &FabricService{
		gateway: gateway,
	}
}

// =============================================================================
// Document Type Operations
// =============================================================================

func (s *FabricService) RegisterDocumentType(channelKey string, req *models.CreateDocumentTypeRequest) (*models.IDResponse, error) {
	contract, err := s.gateway.GetContract(channelKey)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "get contract").
			WithContext("channel", channelKey).
			WithContext("operation", "RegisterDocumentType")
	}

	requiredFieldsJSON, err := json.Marshal(req.RequiredFields)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrCodeMarshalingFailed, "Failed to marshal required fields", err).
			WithContext("typeId", req.ID)
	}

	optionalFieldsJSON, err := json.Marshal(req.OptionalFields)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrCodeMarshalingFailed, "Failed to marshal optional fields", err).
			WithContext("typeId", req.ID)
	}

	_, err = contract.SubmitTransaction(
		"RegisterDocumentType",
		req.ID,
		req.Name,
		req.Description,
		string(requiredFieldsJSON),
		string(optionalFieldsJSON),
	)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "register document type").
			WithContext("typeId", req.ID).
			WithContext("channel", channelKey)
	}

	log.Info().Str("typeId", req.ID).Str("channel", channelKey).Msg("Document type registered")
	return &models.IDResponse{Success: true, ID: req.ID}, nil
}

func (s *FabricService) GetDocumentType(channelKey, typeID string) (*models.DocumentType, error) {
	contract, err := s.gateway.GetContract(channelKey)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "get contract").
			WithContext("channel", channelKey).
			WithContext("operation", "GetDocumentType")
	}

	result, err := contract.EvaluateTransaction("GetDocumentType", typeID)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "get document type").
			WithContext("typeId", typeID).
			WithContext("channel", channelKey)
	}

	var docType models.DocumentType
	if err := json.Unmarshal(result, &docType); err != nil {
		return nil, errors.NewAppError(errors.ErrCodeUnmarshalingFailed, "Failed to parse document type data", err).
			WithContext("typeId", typeID).
			WithContext("channel", channelKey)
	}

	return &docType, nil
}

func (s *FabricService) ListDocumentTypes(channelKey, orgID string) ([]*models.DocumentType, error) {
	contract, err := s.gateway.GetContract(channelKey)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "get contract").
			WithContext("channel", channelKey).
			WithContext("operation", "ListDocumentTypes")
	}

	result, err := contract.EvaluateTransaction("ListDocumentTypes", orgID)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "list document types").
			WithContext("orgId", orgID).
			WithContext("channel", channelKey)
	}

	var types []*models.DocumentType
	if err := json.Unmarshal(result, &types); err != nil {
		return nil, errors.NewAppError(errors.ErrCodeUnmarshalingFailed, "Failed to parse document types data", err).
			WithContext("orgId", orgID).
			WithContext("channel", channelKey)
	}

	if types == nil {
		types = []*models.DocumentType{}
	}

	return types, nil
}

func (s *FabricService) DeactivateDocumentType(channelKey, typeID string) error {
	contract, err := s.gateway.GetContract(channelKey)
	if err != nil {
		return errors.ParseBlockchainError(err, "get contract").
			WithContext("channel", channelKey).
			WithContext("operation", "DeactivateDocumentType")
	}

	_, err = contract.SubmitTransaction("DeactivateDocumentType", typeID)
	if err != nil {
		return errors.ParseBlockchainError(err, "deactivate document type").
			WithContext("typeId", typeID).
			WithContext("channel", channelKey)
	}

	log.Info().Str("typeId", typeID).Str("channel", channelKey).Msg("Document type deactivated")
	return nil
}

// =============================================================================
// Document Operations
// =============================================================================

func (s *FabricService) CreateDocument(channelKey string, req *models.CreateDocumentRequest) (*models.IDResponse, error) {
	contract, err := s.gateway.GetContract(channelKey)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "get contract").
			WithContext("channel", channelKey).
			WithContext("operation", "CreateDocument")
	}

	docID := req.ID
	if docID == "" {
		docID = uuid.New().String()
	}

	currency := req.Currency
	if currency == "" {
		currency = "BRL"
	}

	dataJSON, err := json.Marshal(req.Data)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrCodeMarshalingFailed, "Failed to marshal document data", err).
			WithContext("docId", docID).
			WithContext("channel", channelKey)
	}

	_, err = contract.SubmitTransaction(
		"CreateSimpleDocument",
		docID,
		req.DocumentTypeID,
		req.Title,
		req.Description,
		fmt.Sprintf("%f", req.Amount),
		currency,
		string(dataJSON),
	)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "create document").
			WithContext("docId", docID).
			WithContext("documentTypeId", req.DocumentTypeID).
			WithContext("channel", channelKey)
	}

	log.Info().Str("docId", docID).Str("channel", channelKey).Msg("Document created")
	return &models.IDResponse{Success: true, ID: docID}, nil
}

func (s *FabricService) GetDocument(channelKey, docID string) (*models.Document, error) {
	contract, err := s.gateway.GetContract(channelKey)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "get contract").
			WithContext("channel", channelKey).
			WithContext("operation", "GetDocument")
	}

	result, err := contract.EvaluateTransaction("GetDocument", docID)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "get document").
			WithContext("docId", docID).
			WithContext("channel", channelKey)
	}

	var doc models.Document
	if err := json.Unmarshal(result, &doc); err != nil {
		return nil, errors.NewAppError(errors.ErrCodeUnmarshalingFailed, "Failed to parse document data", err).
			WithContext("docId", docID).
			WithContext("channel", channelKey)
	}

	return &doc, nil
}

func (s *FabricService) QueryDocuments(channelKey string, filter *models.QueryFilter) (*models.QueryResult, error) {
	contract, err := s.gateway.GetContract(channelKey)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "get contract").
			WithContext("channel", channelKey).
			WithContext("operation", "QueryDocuments")
	}

	filterJSON, err := json.Marshal(filter)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrCodeMarshalingFailed, "Failed to marshal query filter", err).
			WithContext("channel", channelKey)
	}

	result, err := contract.EvaluateTransaction("QueryDocuments", string(filterJSON))
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "query documents").
			WithContext("channel", channelKey).
			WithContext("hasOrgFilter", filter.OrganizationID != "").
			WithContext("hasTypeFilter", filter.DocumentTypeID != "")
	}

	var queryResult models.QueryResult
	if err := json.Unmarshal(result, &queryResult); err != nil {
		return nil, errors.NewAppError(errors.ErrCodeUnmarshalingFailed, "Failed to parse query results", err).
			WithContext("channel", channelKey)
	}

	if queryResult.Documents == nil {
		queryResult.Documents = []*models.Document{}
	}

	return &queryResult, nil
}

func (s *FabricService) InvalidateDocument(channelKey, docID string, req *models.InvalidateDocumentRequest) error {
	contract, err := s.gateway.GetContract(channelKey)
	if err != nil {
		return errors.ParseBlockchainError(err, "get contract").
			WithContext("channel", channelKey).
			WithContext("operation", "InvalidateDocument")
	}

	_, err = contract.SubmitTransaction(
		"InvalidateDocument",
		docID,
		req.Reason,
		req.CorrectionDocID,
	)
	if err != nil {
		return errors.ParseBlockchainError(err, "invalidate document").
			WithContext("docId", docID).
			WithContext("channel", channelKey).
			WithContext("reason", req.Reason)
	}

	log.Info().
		Str("docId", docID).
		Str("channel", channelKey).
		Str("reason", req.Reason).
		Msg("Document invalidated")

	return nil
}

func (s *FabricService) GetDocumentHistory(channelKey, docID string) ([]map[string]any, error) {
	contract, err := s.gateway.GetContract(channelKey)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "get contract").
			WithContext("channel", channelKey).
			WithContext("operation", "GetDocumentHistory")
	}

	result, err := contract.EvaluateTransaction("GetDocumentHistory", docID)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "get document history").
			WithContext("docId", docID).
			WithContext("channel", channelKey)
	}

	var history []map[string]any
	if err := json.Unmarshal(result, &history); err != nil {
		return nil, errors.NewAppError(errors.ErrCodeUnmarshalingFailed, "Failed to parse document history", err).
			WithContext("docId", docID).
			WithContext("channel", channelKey)
	}

	if history == nil {
		history = []map[string]any{}
	}

	return history, nil
}

// =============================================================================
// Cross-Channel Transfer Operations (Backend-Coordinated)
// =============================================================================


func (s *FabricService) InitiateTransfer(req *models.InitiateTransferRequest) (*models.TransferResult, error) {
	// Step 1: Get source channel contract
	sourceContract, err := s.gateway.GetContract(req.FromChannel)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "get source channel contract").
			WithContext("sourceChannel", req.FromChannel).
			WithContext("targetChannel", req.ToChannel).
			WithContext("operation", "InitiateTransfer").
			WithContext("step", "get_source_contract")
	}

	transferID := uuid.New().String()

	currency := req.Currency
	if currency == "" {
		currency = "BRL"
	}

	data := req.Data
	if data == nil {
		data = make(map[string]any)
	}
	data["transferType"] = "OUTGOING"
	data["targetOrg"] = req.ToOrg
	data["targetChannel"] = req.ToChannel

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrCodeMarshalingFailed, "Failed to marshal transfer data", err).
			WithContext("transferId", transferID).
			WithContext("sourceChannel", req.FromChannel)
	}

	// Step 2: Create transfer document on source channel
	_, err = sourceContract.SubmitTransaction(
		"CreateDocument",
		transferID,
		req.DocumentTypeID,
		req.Title,
		req.Description,
		fmt.Sprintf("%f", req.Amount),
		currency,
		string(dataJSON),
		"",
		req.ToChannel,
		"",
		"OUTGOING",
	)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "create transfer document").
			WithContext("transferId", transferID).
			WithContext("sourceChannel", req.FromChannel).
			WithContext("targetChannel", req.ToChannel).
			WithContext("documentTypeId", req.DocumentTypeID).
			WithContext("step", "create_source_document").
			WithDetails("Failed to create the outgoing transfer document on the source channel")
	}

	// Step 3: Read back the created document to get content hash
	result, err := sourceContract.EvaluateTransaction("GetDocument", transferID)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "read created transfer document").
			WithContext("transferId", transferID).
			WithContext("sourceChannel", req.FromChannel).
			WithContext("step", "verify_source_document").
			WithDetails("Transfer document was created but could not be verified")
	}

	var sourceDoc models.Document
	if err := json.Unmarshal(result, &sourceDoc); err != nil {
		return nil, errors.NewAppError(errors.ErrCodeUnmarshalingFailed, "Failed to parse created transfer document", err).
			WithContext("transferId", transferID).
			WithContext("sourceChannel", req.FromChannel).
			WithContext("step", "parse_source_document")
	}

	log.Info().
		Str("transferId", transferID).
		Str("fromChannel", req.FromChannel).
		Str("toChannel", req.ToChannel).
		Str("contentHash", sourceDoc.ContentHash).
		Float64("amount", req.Amount).
		Msg("Transfer initiated")

	return &models.TransferResult{
		Success:     true,
		ID:          transferID,
		ContentHash: sourceDoc.ContentHash,
		Channel:     req.FromChannel,
	}, nil
}

func (s *FabricService) AcknowledgeTransfer(targetChannelKey string, req *models.AcknowledgeTransferRequest) (*models.TransferResult, error) {
	// Step 1: Get and verify source document from source channel
	sourceContract, err := s.gateway.GetContract(req.SourceChannel)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "get source channel contract").
			WithContext("sourceChannel", req.SourceChannel).
			WithContext("targetChannel", targetChannelKey).
			WithContext("operation", "AcknowledgeTransfer").
			WithContext("step", "get_source_contract")
	}

	sourceResult, err := sourceContract.EvaluateTransaction("GetDocument", req.SourceDocID)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "get source document").
			WithContext("sourceDocId", req.SourceDocID).
			WithContext("sourceChannel", req.SourceChannel).
			WithContext("targetChannel", targetChannelKey).
			WithContext("step", "fetch_source_document").
			WithDetails("Cannot acknowledge transfer because the source document is not accessible")
	}

	var sourceDoc models.Document
	if err := json.Unmarshal(sourceResult, &sourceDoc); err != nil {
		return nil, errors.NewAppError(errors.ErrCodeUnmarshalingFailed, "Failed to parse source document", err).
			WithContext("sourceDocId", req.SourceDocID).
			WithContext("sourceChannel", req.SourceChannel).
			WithContext("step", "parse_source_document")
	}

	// Step 2: Create acknowledgment document on target channel
	targetContract, err := s.gateway.GetContract(targetChannelKey)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "get target channel contract").
			WithContext("sourceChannel", req.SourceChannel).
			WithContext("targetChannel", targetChannelKey).
			WithContext("operation", "AcknowledgeTransfer").
			WithContext("step", "get_target_contract")
	}

	ackID := uuid.New().String()

	data := req.Data
	if data == nil {
		data = make(map[string]any)
	}
	data["transferType"] = "INCOMING"
	data["sourceDocId"] = req.SourceDocID
	data["sourceChannel"] = req.SourceChannel
	data["sourceContentHash"] = sourceDoc.ContentHash
	data["sourceOrg"] = sourceDoc.OrganizationID

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrCodeMarshalingFailed, "Failed to marshal acknowledgment data", err).
			WithContext("ackId", ackID).
			WithContext("sourceDocId", req.SourceDocID)
	}

	_, err = targetContract.SubmitTransaction(
		"CreateDocument",
		ackID,
		req.DocumentTypeID,
		req.Title,
		req.Description,
		fmt.Sprintf("%f", sourceDoc.Amount),
		sourceDoc.Currency,
		string(dataJSON),
		req.SourceDocID,
		req.SourceChannel,
		sourceDoc.ContentHash,
		"INCOMING",
	)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "create acknowledgment document").
			WithContext("ackId", ackID).
			WithContext("sourceDocId", req.SourceDocID).
			WithContext("targetChannel", targetChannelKey).
			WithContext("documentTypeId", req.DocumentTypeID).
			WithContext("step", "create_ack_document").
			WithDetails("Failed to create the incoming acknowledgment document on the target channel")
	}

	ackResult, err := targetContract.EvaluateTransaction("GetDocument", ackID)
	if err != nil {
		return nil, errors.ParseBlockchainError(err, "read acknowledgment document").
			WithContext("ackId", ackID).
			WithContext("targetChannel", targetChannelKey).
			WithContext("step", "verify_ack_document").
			WithDetails("Acknowledgment document was created but could not be verified")
	}

	var ackDoc models.Document
	if err := json.Unmarshal(ackResult, &ackDoc); err != nil {
		return nil, errors.NewAppError(errors.ErrCodeUnmarshalingFailed, "Failed to parse acknowledgment document", err).
			WithContext("ackId", ackID).
			WithContext("targetChannel", targetChannelKey).
			WithContext("step", "parse_ack_document")
	}

	// Step 3: Update source document with link to acknowledgment
	_, err = sourceContract.SubmitTransaction(
		"UpdateDocumentLink",
		req.SourceDocID,
		ackID,
		targetChannelKey,
		ackDoc.ContentHash,
	)
	if err != nil {
		linkErr := errors.ParseBlockchainError(err, "update source document link").
			WithContext("sourceDocId", req.SourceDocID).
			WithContext("ackId", ackID).
			WithContext("sourceChannel", req.SourceChannel).
			WithContext("targetChannel", targetChannelKey).
			WithContext("step", "update_link")

		log.Warn().
			Err(linkErr).
			Str("sourceDocId", req.SourceDocID).
			Str("ackId", ackID).
			Str("errorCode", string(linkErr.Code)).
			Msg("Failed to update source document with acknowledgment link - transfer completed but bidirectional link not established")
	}

	log.Info().
		Str("ackId", ackID).
		Str("sourceDocId", req.SourceDocID).
		Str("sourceChannel", req.SourceChannel).
		Str("targetChannel", targetChannelKey).
		Str("linkedHash", sourceDoc.ContentHash).
		Msg("Transfer acknowledged with hash anchor")

	return &models.TransferResult{
		Success:          true,
		ID:               ackID,
		ContentHash:      ackDoc.ContentHash,
		Channel:          targetChannelKey,
		LinkedDocID:      req.SourceDocID,
		LinkedDocHash:    sourceDoc.ContentHash,
		LinkedDocChannel: req.SourceChannel,
	}, nil
}

// =============================================================================
// Anchor Verification
// =============================================================================

func (s *FabricService) VerifyAnchor(sourceChannel, sourceDocID, targetChannel, targetDocID string) (*models.AnchorVerification, error) {
	// Fetch source document
	sourceDoc, err := s.GetDocument(sourceChannel, sourceDocID)
	if err != nil {
		// The error from GetDocument is already structured
		appErr, ok := err.(*errors.AppError)
		if ok {
			return nil, appErr.
				WithContext("operation", "VerifyAnchor").
				WithContext("document", "source").
				WithDetails("Cannot verify anchor: source document is not accessible")
		}
		return nil, errors.ParseBlockchainError(err, "get source document for anchor verification").
			WithContext("sourceChannel", sourceChannel).
			WithContext("sourceDocId", sourceDocID)
	}

	// Fetch target document
	targetDoc, err := s.GetDocument(targetChannel, targetDocID)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			return nil, appErr.
				WithContext("operation", "VerifyAnchor").
				WithContext("document", "target").
				WithDetails("Cannot verify anchor: target document is not accessible")
		}
		return nil, errors.ParseBlockchainError(err, "get target document for anchor verification").
			WithContext("targetChannel", targetChannel).
			WithContext("targetDocId", targetDocID)
	}

	verification := &models.AnchorVerification{
		SourceDocID:       sourceDocID,
		SourceChannel:     sourceChannel,
		SourceContentHash: sourceDoc.ContentHash,
		SourceAmount:      sourceDoc.Amount,
		SourceCurrency:    sourceDoc.Currency,
		TargetDocID:       targetDocID,
		TargetChannel:     targetChannel,
		TargetContentHash: targetDoc.ContentHash,
		TargetLinkedHash:  targetDoc.LinkedDocHash, 
		TargetAmount:      targetDoc.Amount,
		TargetCurrency:    targetDoc.Currency,
	}

	verification.HashMatch = (targetDoc.LinkedDocHash == sourceDoc.ContentHash)
	verification.IDMatch = (targetDoc.LinkedDocID == sourceDocID)
	verification.ChannelMatch = (targetDoc.LinkedChannel == sourceChannel)
	verification.AmountMatch = (targetDoc.Amount == sourceDoc.Amount)

	verification.IsValid = verification.HashMatch && verification.IDMatch && verification.ChannelMatch && verification.AmountMatch

	if verification.IsValid {
		verification.Status = "VERIFIED"
	} else {
		verification.Status = "MISMATCH"
		var reasons []string
		if !verification.HashMatch {
			reasons = append(reasons, "content hash mismatch")
		}
		if !verification.IDMatch {
			reasons = append(reasons, "document ID mismatch")
		}
		if !verification.ChannelMatch {
			reasons = append(reasons, "channel mismatch")
		}
		if !verification.AmountMatch {
			reasons = append(reasons, "amount mismatch")
		}
		verification.MismatchReason = reasons
	}

	return verification, nil
}

func (s *FabricService) GetLinkedDocuments(channelKey, docID string) (*models.LinkedDocuments, error) {
	// Get the primary document
	doc, err := s.GetDocument(channelKey, docID)
	if err != nil {
		return nil, err
	}

	result := &models.LinkedDocuments{
		Document: doc,
	}

	if doc.LinkedDocID != "" && doc.LinkedChannel != "" {
		linkedDoc, err := s.GetDocument(doc.LinkedChannel, doc.LinkedDocID)
		if err != nil {
			linkedErr := errors.ParseBlockchainError(err, "get linked document").
				WithContext("docId", docID).
				WithContext("linkedDocId", doc.LinkedDocID).
				WithContext("linkedChannel", doc.LinkedChannel).
				WithContext("primaryChannel", channelKey)

			log.Warn().
				Err(linkedErr).
				Str("docId", docID).
				Str("linkedDocId", doc.LinkedDocID).
				Str("linkedChannel", doc.LinkedChannel).
				Str("errorCode", string(linkedErr.Code)).
				Msg("Failed to fetch linked document")

			result.LinkedDocument = nil
			result.LinkVerified = false
		} else {
			result.LinkedDocument = linkedDoc
			result.LinkVerified = (linkedDoc.ContentHash == doc.LinkedDocHash)

			if !result.LinkVerified {
				log.Warn().
					Str("docId", docID).
					Str("linkedDocId", doc.LinkedDocID).
					Str("expectedHash", doc.LinkedDocHash).
					Str("actualHash", linkedDoc.ContentHash).
					Msg("Linked document hash mismatch detected")
			}
		}
	}

	return result, nil
}