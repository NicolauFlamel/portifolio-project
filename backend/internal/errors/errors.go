package errors

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type ErrorCode string

const (
	// Network and connectivity errors (retriable)
	ErrCodeNetworkFailure     ErrorCode = "NETWORK_FAILURE"
	ErrCodeConnectionTimeout  ErrorCode = "CONNECTION_TIMEOUT"
	ErrCodeChannelUnavailable ErrorCode = "CHANNEL_UNAVAILABLE"
	ErrCodeGatewayFailure     ErrorCode = "GATEWAY_FAILURE"

	// Resource errors
	ErrCodeNotFound        ErrorCode = "NOT_FOUND"
	ErrCodeAlreadyExists   ErrorCode = "ALREADY_EXISTS"
	ErrCodeInvalidChannel  ErrorCode = "INVALID_CHANNEL"
	ErrCodeContractNotFound ErrorCode = "CONTRACT_NOT_FOUND"

	// Validation errors (not retriable)
	ErrCodeValidationFailed    ErrorCode = "VALIDATION_FAILED"
	ErrCodeInvalidInput        ErrorCode = "INVALID_INPUT"
	ErrCodeInvalidDocumentType ErrorCode = "INVALID_DOCUMENT_TYPE"
	ErrCodeInvalidTransfer     ErrorCode = "INVALID_TRANSFER"
	ErrCodeInvalidAnchor       ErrorCode = "INVALID_ANCHOR"

	// Transaction errors
	ErrCodeTransactionFailed  ErrorCode = "TRANSACTION_FAILED"
	ErrCodeTransactionTimeout ErrorCode = "TRANSACTION_TIMEOUT"
	ErrCodeEndorsementFailed  ErrorCode = "ENDORSEMENT_FAILED"
	ErrCodeCommitFailed       ErrorCode = "COMMIT_FAILED"

	// Authentication and authorization
	ErrCodeUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrCodePermissionDenied   ErrorCode = "PERMISSION_DENIED"
	ErrCodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	ErrCodeCertificateError   ErrorCode = "CERTIFICATE_ERROR"

	// Data errors
	ErrCodeMarshalingFailed   ErrorCode = "MARSHALING_FAILED"
	ErrCodeUnmarshalingFailed ErrorCode = "UNMARSHALING_FAILED"
	ErrCodeDataCorruption     ErrorCode = "DATA_CORRUPTION"

	// Cross-channel operation errors
	ErrCodeTransferInitFailed    ErrorCode = "TRANSFER_INIT_FAILED"
	ErrCodeTransferAckFailed     ErrorCode = "TRANSFER_ACK_FAILED"
	ErrCodeLinkUpdateFailed      ErrorCode = "LINK_UPDATE_FAILED"
	ErrCodeAnchorVerifyFailed    ErrorCode = "ANCHOR_VERIFY_FAILED"
	ErrCodeSourceDocNotFound     ErrorCode = "SOURCE_DOC_NOT_FOUND"
	ErrCodeLinkedDocUnavailable  ErrorCode = "LINKED_DOC_UNAVAILABLE"

	// Query errors
	ErrCodeQueryFailed   ErrorCode = "QUERY_FAILED"
	ErrCodeQueryTimeout  ErrorCode = "QUERY_TIMEOUT"
	ErrCodeInvalidQuery  ErrorCode = "INVALID_QUERY"

	// System errors
	ErrCodeInternalError  ErrorCode = "INTERNAL_ERROR"
	ErrCodeConfigError    ErrorCode = "CONFIG_ERROR"
	ErrCodeServiceFailure ErrorCode = "SERVICE_FAILURE"
)

type AppError struct {
	Code       ErrorCode              
	Message    string                 
	Details    string                 
	Err        error                  
	HTTPStatus int                    
	Retriable  bool                   
	Context    map[string]interface{} 
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(code ErrorCode, message string, err error) *AppError {
	appErr := &AppError{
		Code:    code,
		Message: message,
		Err:     err,
		Context: make(map[string]interface{}),
	}
	appErr.setDefaults()
	return appErr
}

func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

func (e *AppError) WithHTTPStatus(status int) *AppError {
	e.HTTPStatus = status
	return e
}

func (e *AppError) setDefaults() {
	switch e.Code {
	case ErrCodeNetworkFailure, ErrCodeConnectionTimeout, ErrCodeChannelUnavailable:
		e.HTTPStatus = http.StatusServiceUnavailable
		e.Retriable = true

	case ErrCodeGatewayFailure:
		e.HTTPStatus = http.StatusBadGateway
		e.Retriable = true

	case ErrCodeNotFound, ErrCodeSourceDocNotFound, ErrCodeContractNotFound:
		e.HTTPStatus = http.StatusNotFound
		e.Retriable = false

	case ErrCodeAlreadyExists:
		e.HTTPStatus = http.StatusConflict
		e.Retriable = false

	case ErrCodeValidationFailed, ErrCodeInvalidInput, ErrCodeInvalidChannel,
		ErrCodeInvalidDocumentType, ErrCodeInvalidTransfer, ErrCodeInvalidQuery,
		ErrCodeInvalidAnchor:
		e.HTTPStatus = http.StatusBadRequest
		e.Retriable = false

	case ErrCodeUnauthorized, ErrCodeInvalidCredentials:
		e.HTTPStatus = http.StatusUnauthorized
		e.Retriable = false

	case ErrCodePermissionDenied:
		e.HTTPStatus = http.StatusForbidden
		e.Retriable = false

	case ErrCodeCertificateError:
		e.HTTPStatus = http.StatusUnauthorized
		e.Retriable = false

	case ErrCodeTransactionTimeout, ErrCodeQueryTimeout:
		e.HTTPStatus = http.StatusGatewayTimeout
		e.Retriable = true

	case ErrCodeTransactionFailed, ErrCodeEndorsementFailed, ErrCodeCommitFailed:
		e.HTTPStatus = http.StatusInternalServerError
		e.Retriable = true

	case ErrCodeMarshalingFailed, ErrCodeUnmarshalingFailed, ErrCodeDataCorruption:
		e.HTTPStatus = http.StatusInternalServerError
		e.Retriable = false

	case ErrCodeTransferInitFailed, ErrCodeTransferAckFailed, ErrCodeLinkUpdateFailed:
		e.HTTPStatus = http.StatusInternalServerError
		e.Retriable = true
	case ErrCodeAnchorVerifyFailed:
		e.HTTPStatus = http.StatusBadRequest
		e.Retriable = false
	case ErrCodeLinkedDocUnavailable:
		e.HTTPStatus = http.StatusFailedDependency // 424
		e.Retriable = true

	case ErrCodeQueryFailed:
		e.HTTPStatus = http.StatusInternalServerError
		e.Retriable = true

	default:
		e.HTTPStatus = http.StatusInternalServerError
		e.Retriable = false
	}
}

func ParseBlockchainError(err error, operation string) *AppError {
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	errLower := strings.ToLower(errMsg)

	if strings.Contains(errLower, "connection refused") ||
		strings.Contains(errLower, "connection reset") ||
		strings.Contains(errLower, "no such host") ||
		strings.Contains(errLower, "network is unreachable") {
		return NewAppError(
			ErrCodeNetworkFailure,
			fmt.Sprintf("Failed to connect to blockchain network during %s", operation),
			err,
		).WithDetails("The blockchain network is currently unreachable. Please try again later.")
	}

	if strings.Contains(errLower, "timeout") || strings.Contains(errLower, "deadline exceeded") {
		if strings.Contains(operation, "query") {
			return NewAppError(
				ErrCodeQueryTimeout,
				fmt.Sprintf("Query operation timed out: %s", operation),
				err,
			).WithDetails("The query took too long to execute. Try narrowing your search criteria.")
		}
		return NewAppError(
			ErrCodeTransactionTimeout,
			fmt.Sprintf("Transaction timed out during %s", operation),
			err,
		).WithDetails("The blockchain transaction did not complete in time. Please retry.")
	}

	if strings.Contains(errLower, "not found") ||
		strings.Contains(errLower, "does not exist") ||
		strings.Contains(errLower, "no rows") {

		if strings.Contains(errLower, "document") {
			return NewAppError(
				ErrCodeNotFound,
				"Document not found",
				err,
			).WithDetails("The requested document does not exist in the blockchain.")
		}
		if strings.Contains(errLower, "type") {
			return NewAppError(
				ErrCodeInvalidDocumentType,
				"Document type not found",
				err,
			).WithDetails("The specified document type does not exist or has been deactivated.")
		}
		if strings.Contains(errLower, "contract") || strings.Contains(errLower, "chaincode") {
			return NewAppError(
				ErrCodeContractNotFound,
				"Smart contract not found",
				err,
			).WithDetails("The blockchain smart contract is not available on this channel.")
		}
		return NewAppError(
			ErrCodeNotFound,
			fmt.Sprintf("Resource not found during %s", operation),
			err,
		)
	}

	if strings.Contains(errLower, "already exists") ||
		strings.Contains(errLower, "duplicate") ||
		strings.Contains(errLower, "conflict") {
		return NewAppError(
			ErrCodeAlreadyExists,
			"Resource already exists",
			err,
		).WithDetails("A resource with this identifier already exists in the blockchain.")
	}

	if strings.Contains(errLower, "invalid") ||
		strings.Contains(errLower, "validation failed") ||
		strings.Contains(errLower, "missing required field") ||
		strings.Contains(errLower, "bad request") {
		return NewAppError(
			ErrCodeValidationFailed,
			"Blockchain validation failed",
			err,
		).WithDetails("The blockchain rejected the request due to validation errors.")
	}

	if strings.Contains(errLower, "endorsement") ||
		strings.Contains(errLower, "endorser") {
		return NewAppError(
			ErrCodeEndorsementFailed,
			fmt.Sprintf("Failed to get endorsement for %s", operation),
			err,
		).WithDetails("The blockchain peers could not endorse the transaction. This may be due to a policy violation.")
	}

	if strings.Contains(errLower, "commit") ||
		strings.Contains(errLower, "mvcc") ||
		strings.Contains(errLower, "version mismatch") {
		return NewAppError(
			ErrCodeCommitFailed,
			fmt.Sprintf("Failed to commit transaction for %s", operation),
			err,
		).WithDetails("The transaction was endorsed but could not be committed. There may have been a concurrent update.")
	}

	if strings.Contains(errLower, "permission denied") ||
		strings.Contains(errLower, "access denied") ||
		strings.Contains(errLower, "forbidden") {
		return NewAppError(
			ErrCodePermissionDenied,
			"Permission denied",
			err,
		).WithDetails("You do not have permission to perform this operation on the blockchain.")
	}

	if strings.Contains(errLower, "unauthorized") ||
		strings.Contains(errLower, "authentication failed") {
		return NewAppError(
			ErrCodeUnauthorized,
			"Authentication failed",
			err,
		).WithDetails("Could not authenticate with the blockchain network.")
	}

	if strings.Contains(errLower, "certificate") ||
		strings.Contains(errLower, "cert") ||
		strings.Contains(errLower, "tls") {
		return NewAppError(
			ErrCodeCertificateError,
			"Certificate error",
			err,
		).WithDetails("There was an error with the blockchain credentials. Please check the certificate configuration.")
	}

	if strings.Contains(errLower, "hash") ||
		strings.Contains(errLower, "anchor") ||
		strings.Contains(errLower, "mismatch") {
		return NewAppError(
			ErrCodeAnchorVerifyFailed,
			"Anchor verification failed",
			err,
		).WithDetails("The document hashes do not match between channels. Data integrity check failed.")
	}

	if strings.Contains(errLower, "channel") {
		return NewAppError(
			ErrCodeChannelUnavailable,
			fmt.Sprintf("Channel unavailable during %s", operation),
			err,
		).WithDetails("The specified blockchain channel is not available.")
	}

	return NewAppError(
		ErrCodeTransactionFailed,
		fmt.Sprintf("Blockchain transaction failed: %s", operation),
		err,
	).WithDetails("The blockchain operation failed. Please try again or contact support if the issue persists.")
}

func SanitizeError(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()

	msg = sanitizeFilePaths(msg)

	msg = sanitizeInternalAddresses(msg)

	if strings.Contains(msg, "certificate") || strings.Contains(msg, "cert") {
		msg = "Certificate error occurred"
	}

	return msg
}

func sanitizeFilePaths(msg string) string {
	patterns := []string{
		"/home/", "/var/", "/usr/", "/opt/", "/tmp/",
		"C:\\", "D:\\", "/Users/",
	}

	for _, pattern := range patterns {
		if idx := strings.Index(msg, pattern); idx != -1 {
			end := idx
			for end < len(msg) && msg[end] != ' ' && msg[end] != ':' && msg[end] != '\n' {
				end++
			}
			msg = msg[:idx] + "[path]" + msg[end:]
		}
	}

	return msg
}

func sanitizeInternalAddresses(msg string) string {
	msg = strings.ReplaceAll(msg, "127.0.0.1", "[blockchain-host]")
	msg = strings.ReplaceAll(msg, "localhost", "[blockchain-host]")

	if strings.Contains(msg, "192.168.") || strings.Contains(msg, "10.") || strings.Contains(msg, "172.") {
		msg = strings.ReplaceAll(msg, "192.168.", "[blockchain-host]:")
		msg = strings.ReplaceAll(msg, "10.", "[blockchain-host]:")
	}

	return msg
}

func NewNotFoundError(resource string, id string) *AppError {
	return NewAppError(
		ErrCodeNotFound,
		fmt.Sprintf("%s not found", resource),
		errors.New("resource not found"),
	).WithContext("resource", resource).WithContext("id", id)
}

func NewValidationError(message string) *AppError {
	return NewAppError(
		ErrCodeValidationFailed,
		message,
		errors.New("validation failed"),
	)
}

func NewInvalidChannelError(channel string) *AppError {
	return NewAppError(
		ErrCodeInvalidChannel,
		"Invalid channel",
		errors.New("invalid channel"),
	).WithContext("channel", channel).
		WithDetails("Valid channels are: union, state, region")
}
