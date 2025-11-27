package services

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

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
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}

	requiredFieldsJSON, _ := json.Marshal(req.RequiredFields)
	optionalFieldsJSON, _ := json.Marshal(req.OptionalFields)

	_, err = contract.SubmitTransaction(
		"RegisterDocumentType",
		req.ID,
		req.Name,
		req.Description,
		string(requiredFieldsJSON),
		string(optionalFieldsJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register document type: %w", err)
	}

	log.Info().Str("typeId", req.ID).Str("channel", channelKey).Msg("Document type registered")
	return &models.IDResponse{Success: true, ID: req.ID}, nil
}

func (s *FabricService) GetDocumentType(channelKey, typeID string) (*models.DocumentType, error) {
	contract, err := s.gateway.GetContract(channelKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}

	result, err := contract.EvaluateTransaction("GetDocumentType", typeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document type: %w", err)
	}

	var docType models.DocumentType
	if err := json.Unmarshal(result, &docType); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document type: %w", err)
	}

	return &docType, nil
}

func (s *FabricService) ListDocumentTypes(channelKey, orgID string) ([]*models.DocumentType, error) {
	contract, err := s.gateway.GetContract(channelKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}

	result, err := contract.EvaluateTransaction("ListDocumentTypes", orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list document types: %w", err)
	}

	var types []*models.DocumentType
	if err := json.Unmarshal(result, &types); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document types: %w", err)
	}

	if types == nil {
		types = []*models.DocumentType{}
	}

	return types, nil
}

func (s *FabricService) DeactivateDocumentType(channelKey, typeID string) error {
	contract, err := s.gateway.GetContract(channelKey)
	if err != nil {
		return fmt.Errorf("failed to get contract: %w", err)
	}

	_, err = contract.SubmitTransaction("DeactivateDocumentType", typeID)
	if err != nil {
		return fmt.Errorf("failed to deactivate document type: %w", err)
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
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}

	docID := req.ID
	if docID == "" {
		docID = uuid.New().String()
	}

	currency := req.Currency
	if currency == "" {
		currency = "BRL"
	}

	dataJSON, _ := json.Marshal(req.Data)

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
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	log.Info().Str("docId", docID).Str("channel", channelKey).Msg("Document created")
	return &models.IDResponse{Success: true, ID: docID}, nil
}

func (s *FabricService) GetDocument(channelKey, docID string) (*models.Document, error) {
	contract, err := s.gateway.GetContract(channelKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}

	result, err := contract.EvaluateTransaction("GetDocument", docID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	var doc models.Document
	if err := json.Unmarshal(result, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	return &doc, nil
}

func (s *FabricService) QueryDocuments(channelKey string, filter *models.QueryFilter) (*models.QueryResult, error) {
	contract, err := s.gateway.GetContract(channelKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}

	filterJSON, _ := json.Marshal(filter)

	result, err := contract.EvaluateTransaction("QueryDocuments", string(filterJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}

	var queryResult models.QueryResult
	if err := json.Unmarshal(result, &queryResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal query result: %w", err)
	}

	if queryResult.Documents == nil {
		queryResult.Documents = []*models.Document{}
	}

	return &queryResult, nil
}

func (s *FabricService) InvalidateDocument(channelKey, docID string, req *models.InvalidateDocumentRequest) error {
	contract, err := s.gateway.GetContract(channelKey)
	if err != nil {
		return fmt.Errorf("failed to get contract: %w", err)
	}

	_, err = contract.SubmitTransaction(
		"InvalidateDocument",
		docID,
		req.Reason,
		req.CorrectionDocID,
	)
	if err != nil {
		return fmt.Errorf("failed to invalidate document: %w", err)
	}

	log.Info().
		Str("docId", docID).
		Str("channel", channelKey).
		Str("reason", req.Reason).
		Msg("Document invalidated")

	return nil
}

func (s *FabricService) GetDocumentHistory(channelKey, docID string) ([]map[string]interface{}, error) {
	contract, err := s.gateway.GetContract(channelKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}

	result, err := contract.EvaluateTransaction("GetDocumentHistory", docID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document history: %w", err)
	}

	var history []map[string]interface{}
	if err := json.Unmarshal(result, &history); err != nil {
		return nil, fmt.Errorf("failed to unmarshal history: %w", err)
	}

	if history == nil {
		history = []map[string]interface{}{}
	}

	return history, nil
}

// =============================================================================
// Cross-Channel Transfer Operations (Backend-Coordinated)
// =============================================================================


func (s *FabricService) InitiateTransfer(req *models.InitiateTransferRequest) (*models.TransferResult, error) {
	sourceContract, err := s.gateway.GetContract(req.FromChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to get source channel contract: %w", err)
	}

	transferID := uuid.New().String()

	currency := req.Currency
	if currency == "" {
		currency = "BRL"
	}

	data := req.Data
	if data == nil {
		data = make(map[string]interface{})
	}
	data["transferType"] = "OUTGOING"
	data["targetOrg"] = req.ToOrg
	data["targetChannel"] = req.ToChannel

	dataJSON, _ := json.Marshal(data)


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
		return nil, fmt.Errorf("failed to create transfer document: %w", err)
	}

	result, err := sourceContract.EvaluateTransaction("GetDocument", transferID)
	if err != nil {
		return nil, fmt.Errorf("failed to read created document: %w", err)
	}

	var sourceDoc models.Document
	if err := json.Unmarshal(result, &sourceDoc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal source document: %w", err)
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
	sourceContract, err := s.gateway.GetContract(req.SourceChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to get source channel contract: %w", err)
	}

	sourceResult, err := sourceContract.EvaluateTransaction("GetDocument", req.SourceDocID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source document from %s: %w", req.SourceChannel, err)
	}

	var sourceDoc models.Document
	if err := json.Unmarshal(sourceResult, &sourceDoc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal source document: %w", err)
	}

	targetContract, err := s.gateway.GetContract(targetChannelKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get target channel contract: %w", err)
	}

	ackID := uuid.New().String()

	data := req.Data
	if data == nil {
		data = make(map[string]interface{})
	}
	data["transferType"] = "INCOMING"
	data["sourceDocId"] = req.SourceDocID
	data["sourceChannel"] = req.SourceChannel
	data["sourceContentHash"] = sourceDoc.ContentHash
	data["sourceOrg"] = sourceDoc.OrganizationID

	dataJSON, _ := json.Marshal(data)

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
		return nil, fmt.Errorf("failed to create acknowledgment document: %w", err)
	}

	ackResult, err := targetContract.EvaluateTransaction("GetDocument", ackID)
	if err != nil {
		return nil, fmt.Errorf("failed to read acknowledgment document: %w", err)
	}

	var ackDoc models.Document
	if err := json.Unmarshal(ackResult, &ackDoc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal acknowledgment document: %w", err)
	}

	_, err = sourceContract.SubmitTransaction(
		"UpdateDocumentLink",
		req.SourceDocID,
		ackID,            
		targetChannelKey,  
		ackDoc.ContentHash, 
	)
	if err != nil {
		log.Warn().
			Err(err).
			Str("sourceDocId", req.SourceDocID).
			Msg("Failed to update source document with acknowledgment link")
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
	sourceDoc, err := s.GetDocument(sourceChannel, sourceDocID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source document: %w", err)
	}

	targetDoc, err := s.GetDocument(targetChannel, targetDocID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target document: %w", err)
	}

	verification := &models.AnchorVerification{
		SourceDocID:      sourceDocID,
		SourceChannel:    sourceChannel,
		SourceHash:       sourceDoc.ContentHash,
		SourceAmount:     sourceDoc.Amount,
		SourceCurrency:   sourceDoc.Currency,
		TargetDocID:      targetDocID,
		TargetChannel:    targetChannel,
		TargetHash:       targetDoc.ContentHash,
		TargetAmount:     targetDoc.Amount,
		TargetCurrency:   targetDoc.Currency,
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
	doc, err := s.GetDocument(channelKey, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	result := &models.LinkedDocuments{
		Document: doc,
	}

	if doc.LinkedDocID != "" && doc.LinkedChannel != "" {
		linkedDoc, err := s.GetDocument(doc.LinkedChannel, doc.LinkedDocID)
		if err != nil {
			log.Warn().
				Err(err).
				Str("linkedDocId", doc.LinkedDocID).
				Str("linkedChannel", doc.LinkedChannel).
				Msg("Failed to get linked document")
		} else {
			result.LinkedDocument = linkedDoc
			result.LinkVerified = (linkedDoc.ContentHash == doc.LinkedDocHash)
		}
	}

	return result, nil
}