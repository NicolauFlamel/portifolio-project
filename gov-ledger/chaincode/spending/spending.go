package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SpendingContract struct {
	contractapi.Contract
}

const (
	DocPrefix  = "DOC"
	TypePrefix = "TYPE"
)

type DocumentStatus string

const (
	StatusActive      DocumentStatus = "ACTIVE"
	StatusInvalidated DocumentStatus = "INVALIDATED"
)

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

type QueryFilter struct {
	OrganizationID  string         `json:"organizationId,omitempty"`
	DocumentTypeID  string         `json:"documentTypeId,omitempty"`
	Status          DocumentStatus `json:"status,omitempty"`
	FromDate        string         `json:"fromDate,omitempty"`
	ToDate          string         `json:"toDate,omitempty"`
	MinAmount       float64        `json:"minAmount,omitempty"`
	MaxAmount       float64        `json:"maxAmount,omitempty"`
	HasLinkedDoc    *bool          `json:"hasLinkedDoc,omitempty"`
	LinkedDirection string         `json:"linkedDirection,omitempty"`
	PageSize        int            `json:"pageSize,omitempty"`
	Bookmark        string         `json:"bookmark,omitempty"`
}

type QueryResult struct {
	Documents []*Document `json:"documents"`
	Bookmark  string      `json:"bookmark,omitempty"`
	Total     int         `json:"total"`
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// =============================================================================
// Document Type Management
// =============================================================================

func (s *SpendingContract) RegisterDocumentType(ctx contractapi.TransactionContextInterface,
	id string, name string, description string, requiredFieldsJSON string, optionalFieldsJSON string) error {

	exists, err := s.documentTypeExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("document type %s already exists", id)
	}

	clientID, err := s.getClientIdentity(ctx)
	if err != nil {
		return err
	}

	orgID, err := s.getClientOrg(ctx)
	if err != nil {
		return err
	}

	var requiredFields, optionalFields []string
	if err := json.Unmarshal([]byte(requiredFieldsJSON), &requiredFields); err != nil {
		return fmt.Errorf("invalid required fields JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(optionalFieldsJSON), &optionalFields); err != nil {
		return fmt.Errorf("invalid optional fields JSON: %v", err)
	}

	docType := &DocumentType{
		ID:             id,
		OrganizationID: orgID,
		Name:           name,
		Description:    description,
		RequiredFields: requiredFields,
		OptionalFields: optionalFields,
		CreatedAt:      now(),
		CreatedBy:      clientID,
		IsActive:       true,
	}

	return s.putDocumentType(ctx, docType)
}

func (s *SpendingContract) GetDocumentType(ctx contractapi.TransactionContextInterface, id string) (*DocumentType, error) {
	key, err := ctx.GetStub().CreateCompositeKey(TypePrefix, []string{id})
	if err != nil {
		return nil, fmt.Errorf("failed to create composite key: %v", err)
	}

	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %v", err)
	}
	if data == nil {
		return nil, fmt.Errorf("document type %s not found", id)
	}

	var docType DocumentType
	if err := json.Unmarshal(data, &docType); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document type: %v", err)
	}

	return &docType, nil
}

func (s *SpendingContract) ListDocumentTypes(ctx contractapi.TransactionContextInterface, orgID string) ([]*DocumentType, error) {
	iterator, err := ctx.GetStub().GetStateByPartialCompositeKey(TypePrefix, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %v", err)
	}
	defer iterator.Close()

	var types []*DocumentType
	for iterator.HasNext() {
		result, err := iterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate: %v", err)
		}

		var docType DocumentType
		if err := json.Unmarshal(result.Value, &docType); err != nil {
			return nil, fmt.Errorf("failed to unmarshal: %v", err)
		}

		if orgID == "" || docType.OrganizationID == orgID {
			types = append(types, &docType)
		}
	}

	return types, nil
}

func (s *SpendingContract) DeactivateDocumentType(ctx contractapi.TransactionContextInterface, id string) error {
	docType, err := s.GetDocumentType(ctx, id)
	if err != nil {
		return err
	}

	orgID, err := s.getClientOrg(ctx)
	if err != nil {
		return err
	}
	if docType.OrganizationID != orgID {
		return fmt.Errorf("only the owning organization can deactivate a document type")
	}

	docType.IsActive = false
	return s.putDocumentType(ctx, docType)
}

// =============================================================================
// Document Management
// =============================================================================

func (s *SpendingContract) CreateDocument(ctx contractapi.TransactionContextInterface,
	id string, documentTypeID string, title string, description string,
	amount float64, currency string, dataJSON string,
	linkedDocID string, linkedChannel string, linkedDocHash string, linkedDirection string) error {

	exists, err := s.documentExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("document %s already exists", id)
	}

	docType, err := s.GetDocumentType(ctx, documentTypeID)
	if err != nil {
		return fmt.Errorf("document type not found: %v", err)
	}
	if !docType.IsActive {
		return fmt.Errorf("document type %s is not active", documentTypeID)
	}

	clientID, err := s.getClientIdentity(ctx)
	if err != nil {
		return err
	}
	orgID, err := s.getClientOrg(ctx)
	if err != nil {
		return err
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return fmt.Errorf("invalid data JSON: %v", err)
	}

	if err := s.validateRequiredFields(docType.RequiredFields, data); err != nil {
		return err
	}

	channelID := ctx.GetStub().GetChannelID()
	contentHash := s.calculateHash(data)
	txID := ctx.GetStub().GetTxID()

	doc := &Document{
		ID:              id,
		DocumentTypeID:  documentTypeID,
		OrganizationID:  orgID,
		ChannelID:       channelID,
		Status:          StatusActive,
		Title:           title,
		Description:     description,
		Amount:          amount,
		Currency:        currency,
		Data:            data,
		ContentHash:     contentHash,
		// Cross-channel linking fields - initialize to provided values or empty strings
		LinkedDocID:     linkedDocID,
		LinkedChannel:   linkedChannel,
		LinkedDocHash:   linkedDocHash,
		LinkedDirection: linkedDirection,
		// Invalidation fields - initialize to empty strings
		InvalidatedBy:  "",
		InvalidatedAt:  "",
		InvalidReason:  "",
		CorrectedByDoc: "",
		// Audit trail
		CreatedAt:       now(),
		CreatedBy:       clientID,
		UpdatedAt:       now(),
		UpdatedBy:       clientID,
		History:         []string{txID},
	}

	return s.putDocument(ctx, doc)
}

func (s *SpendingContract) CreateSimpleDocument(ctx contractapi.TransactionContextInterface,
	id string, documentTypeID string, title string, description string,
	amount float64, currency string, dataJSON string) error {
	return s.CreateDocument(ctx, id, documentTypeID, title, description, amount, currency, dataJSON, "", "", "", "")
}

func (s *SpendingContract) GetDocument(ctx contractapi.TransactionContextInterface, id string) (*Document, error) {
	key, err := ctx.GetStub().CreateCompositeKey(DocPrefix, []string{id})
	if err != nil {
		return nil, fmt.Errorf("failed to create composite key: %v", err)
	}

	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %v", err)
	}
	if data == nil {
		return nil, fmt.Errorf("document %s not found", id)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %v", err)
	}

	if doc.Data == nil {
		doc.Data = map[string]interface{}{}
	}
	if doc.History == nil {
		doc.History = []string{}
	}

	return &doc, nil
}

func (s *SpendingContract) QueryDocuments(ctx contractapi.TransactionContextInterface, filterJSON string) (*QueryResult, error) {
	var filter QueryFilter
	if err := json.Unmarshal([]byte(filterJSON), &filter); err != nil {
		return nil, fmt.Errorf("invalid filter JSON: %v", err)
	}

	queryString := s.buildQuery(filter)

	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	resultsIterator, meta, err := ctx.GetStub().GetQueryResultWithPagination(queryString, int32(pageSize), filter.Bookmark)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	defer resultsIterator.Close()

	var documents []*Document
	for resultsIterator.HasNext() {
		result, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate: %v", err)
		}

		var doc Document
		if err := json.Unmarshal(result.Value, &doc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal: %v", err)
		}
		if doc.Data == nil {
			doc.Data = map[string]interface{}{}
		}
		if doc.History == nil {
			doc.History = []string{}
		}
		documents = append(documents, &doc)
	}

	return &QueryResult{
		Documents: documents,
		Bookmark:  meta.GetBookmark(),
		Total:     len(documents),
	}, nil
}

func (s *SpendingContract) InvalidateDocument(ctx contractapi.TransactionContextInterface,
	id string, reason string, correctionDocID string) error {

	doc, err := s.GetDocument(ctx, id)
	if err != nil {
		return err
	}

	if doc.Status == StatusInvalidated {
		return fmt.Errorf("document %s is already invalidated", id)
	}

	orgID, err := s.getClientOrg(ctx)
	if err != nil {
		return err
	}
	if doc.OrganizationID != orgID {
		return fmt.Errorf("only the creating organization can invalidate a document")
	}

	if correctionDocID != "" {
		exists, err := s.documentExists(ctx, correctionDocID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("correction document %s not found", correctionDocID)
		}
	}

	clientID, err := s.getClientIdentity(ctx)
	if err != nil {
		return err
	}
	txID := ctx.GetStub().GetTxID()

	doc.Status = StatusInvalidated
	doc.InvalidatedBy = clientID
	doc.InvalidatedAt = now()
	doc.InvalidReason = reason
	doc.CorrectedByDoc = correctionDocID
	doc.UpdatedAt = now()
	doc.UpdatedBy = clientID
	doc.History = append(doc.History, txID)

	return s.putDocument(ctx, doc)
}

func (s *SpendingContract) UpdateDocumentLink(ctx contractapi.TransactionContextInterface,
	id string, linkedDocID string, linkedChannel string, linkedDocHash string) error {

	doc, err := s.GetDocument(ctx, id)
	if err != nil {
		return err
	}

	orgID, err := s.getClientOrg(ctx)
	if err != nil {
		return err
	}
	if doc.OrganizationID != orgID {
		return fmt.Errorf("only the creating organization can update document links")
	}

	clientID, err := s.getClientIdentity(ctx)
	if err != nil {
		return err
	}
	txID := ctx.GetStub().GetTxID()

	doc.LinkedDocID = linkedDocID
	doc.LinkedChannel = linkedChannel
	doc.LinkedDocHash = linkedDocHash
	doc.UpdatedAt = now()
	doc.UpdatedBy = clientID
	doc.History = append(doc.History, txID)

	return s.putDocument(ctx, doc)
}

func (s *SpendingContract) GetDocumentHistory(ctx contractapi.TransactionContextInterface, id string) ([]map[string]interface{}, error) {
	key, err := ctx.GetStub().CreateCompositeKey(DocPrefix, []string{id})
	if err != nil {
		return nil, fmt.Errorf("failed to create composite key: %v", err)
	}

	iterator, err := ctx.GetStub().GetHistoryForKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %v", err)
	}
	defer iterator.Close()

	var history []map[string]interface{}
	for iterator.HasNext() {
		result, err := iterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate: %v", err)
		}

		var doc Document
		if err := json.Unmarshal(result.Value, &doc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal: %v", err)
		}

		entry := map[string]interface{}{
			"txId":      result.TxId,
			"timestamp": result.Timestamp.AsTime().Format("2006-01-02T15:04:05Z"),
			"isDelete":  result.IsDelete,
			"document":  doc,
		}
		history = append(history, entry)
	}

	return history, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

func (s *SpendingContract) documentExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	key, err := ctx.GetStub().CreateCompositeKey(DocPrefix, []string{id})
	if err != nil {
		return false, err
	}
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return false, err
	}
	return data != nil, nil
}

func (s *SpendingContract) documentTypeExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	key, err := ctx.GetStub().CreateCompositeKey(TypePrefix, []string{id})
	if err != nil {
		return false, err
	}
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return false, err
	}
	return data != nil, nil
}

func (s *SpendingContract) putDocument(ctx contractapi.TransactionContextInterface, doc *Document) error {
	key, err := ctx.GetStub().CreateCompositeKey(DocPrefix, []string{doc.ID})
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %v", err)
	}

	return ctx.GetStub().PutState(key, data)
}

func (s *SpendingContract) putDocumentType(ctx contractapi.TransactionContextInterface, docType *DocumentType) error {
	key, err := ctx.GetStub().CreateCompositeKey(TypePrefix, []string{docType.ID})
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	data, err := json.Marshal(docType)
	if err != nil {
		return fmt.Errorf("failed to marshal document type: %v", err)
	}

	return ctx.GetStub().PutState(key, data)
}

func (s *SpendingContract) getClientIdentity(ctx contractapi.TransactionContextInterface) (string, error) {
	id, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("failed to get client identity: %v", err)
	}
	return id, nil
}

func (s *SpendingContract) getClientOrg(ctx contractapi.TransactionContextInterface) (string, error) {
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", fmt.Errorf("failed to get MSP ID: %v", err)
	}
	return mspID, nil
}

func (s *SpendingContract) validateRequiredFields(required []string, data map[string]interface{}) error {
	for _, field := range required {
		if _, ok := data[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}
	return nil
}

func (s *SpendingContract) calculateHash(data map[string]interface{}) string {
	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

func (s *SpendingContract) buildQuery(filter QueryFilter) string {
	selector := make(map[string]interface{})

	if filter.OrganizationID != "" {
		selector["organizationId"] = filter.OrganizationID
	}
	if filter.DocumentTypeID != "" {
		selector["documentTypeId"] = filter.DocumentTypeID
	}
	if filter.Status != "" {
		selector["status"] = filter.Status
	}
	if filter.LinkedDirection != "" {
		selector["linkedDirection"] = filter.LinkedDirection
	}
	if filter.HasLinkedDoc != nil {
		if *filter.HasLinkedDoc {
			selector["linkedDocId"] = map[string]interface{}{"$ne": ""}
		} else {
			selector["linkedDocId"] = ""
		}
	}

	if filter.MinAmount > 0 || filter.MaxAmount > 0 {
		amountFilter := make(map[string]interface{})
		if filter.MinAmount > 0 {
			amountFilter["$gte"] = filter.MinAmount
		}
		if filter.MaxAmount > 0 {
			amountFilter["$lte"] = filter.MaxAmount
		}
		selector["amount"] = amountFilter
	}

	if filter.FromDate != "" || filter.ToDate != "" {
		dateFilter := make(map[string]interface{})
		if filter.FromDate != "" {
			dateFilter["$gte"] = filter.FromDate
		}
		if filter.ToDate != "" {
			dateFilter["$lte"] = filter.ToDate
		}
		selector["createdAt"] = dateFilter
	}

	query := map[string]interface{}{
		"selector": selector,
		"sort":     []map[string]string{{"createdAt": "desc"}},
	}

	queryJSON, _ := json.Marshal(query)
	return string(queryJSON)
}

// =============================================================================
// Main
// =============================================================================

func main() {
	chaincode, err := contractapi.NewChaincode(&SpendingContract{})
	if err != nil {
		fmt.Printf("Error creating spending chaincode: %v\n", err)
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting spending chaincode: %v\n", err)
	}
}
