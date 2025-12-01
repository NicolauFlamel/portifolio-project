#!/bin/bash

# Government Spending Blockchain - Test Scenarios

set -e

# Backend instance endpoints
UNION_API="http://localhost:3000/api"
STATE_API="http://localhost:3001/api"
REGION_API="http://localhost:3002/api"

# Channel names
UNION_CHANNEL="union"
STATE_CHANNEL="state"
REGION_CHANNEL="region"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_step() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}[STEP]${NC} $1"
    echo -e "${BLUE}========================================${NC}"
}

log_response() {
    echo -e "${YELLOW}Response:${NC}"
    echo "$1" | jq '.' 2>/dev/null || echo "$1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# =============================================================================
# Health Check
# =============================================================================

check_api() {
    log_step "Checking all backend instances health..."

    log_info "Checking Union backend (port 3000)..."
    union_response=$(curl -s "http://localhost:3000/health" || echo '{"error": "connection failed"}')
    log_response "$union_response"

    log_info "Checking State backend (port 3001)..."
    state_response=$(curl -s "http://localhost:3001/health" || echo '{"error": "connection failed"}')
    log_response "$state_response"

    log_info "Checking Region backend (port 3002)..."
    region_response=$(curl -s "http://localhost:3002/health" || echo '{"error": "connection failed"}')
    log_response "$region_response"

    if echo "$union_response" | grep -q "healthy" && \
       echo "$state_response" | grep -q "healthy" && \
       echo "$region_response" | grep -q "healthy"; then
        log_success "All backend instances are running"

        log_info "Checking configuration details..."
        echo -e "${CYAN}Union backend:${NC}"
        curl -s "http://localhost:3000/config" | jq '.'
        echo -e "${CYAN}State backend:${NC}"
        curl -s "http://localhost:3001/config" | jq '.'
        echo -e "${CYAN}Region backend:${NC}"
        curl -s "http://localhost:3002/config" | jq '.'
    else
        log_error "Not all backends are running. Please start all backends first."
        echo "Expected: Union (3000), State (3001), Region (3002)"
        exit 1
    fi
}

# =============================================================================
# Scenario 1: Setup Document Types
# =============================================================================

setup_document_types() {
    log_step "Setting up document types for each organization..."

    echo -e "${CYAN}Each backend instance creates types on its own channel (write access)${NC}\n"

    log_info "Union backend creates document types on union channel..."
    log_info "Creating Union document type: Federal Transfer"
    curl -s -X POST "$UNION_API/$UNION_CHANNEL/document-types" \
        -H "Content-Type: application/json" \
        -d '{
            "id": "federal-transfer",
            "name": "Federal Transfer",
            "description": "Transfer of funds from federal to state level",
            "requiredFields": ["destinationState", "program", "legalBasis"],
            "optionalFields": ["observations", "attachments"]
        }' | jq '.'

    log_info "Creating Union document type: Federal Expense"
    curl -s -X POST "$UNION_API/$UNION_CHANNEL/document-types" \
        -H "Content-Type: application/json" \
        -d '{
            "id": "federal-expense",
            "name": "Federal Expense",
            "description": "Direct federal spending",
            "requiredFields": ["category", "vendor", "contractNumber"],
            "optionalFields": ["invoiceNumber", "deliveryDate"]
        }' | jq '.'

    log_info "State backend creates document types on state channel..."
    log_info "Creating State document type: State Receipt"
    curl -s -X POST "$STATE_API/$STATE_CHANNEL/document-types" \
        -H "Content-Type: application/json" \
        -d '{
            "id": "state-receipt",
            "name": "State Receipt",
            "description": "Receipt of funds from federal level",
            "requiredFields": ["sourceOrg", "program", "receiptDate"],
            "optionalFields": ["observations"]
        }' | jq '.'

    log_info "Creating State document type: State Transfer"
    curl -s -X POST "$STATE_API/$STATE_CHANNEL/document-types" \
        -H "Content-Type: application/json" \
        -d '{
            "id": "state-transfer",
            "name": "State Transfer",
            "description": "Transfer of funds from state to municipalities",
            "requiredFields": ["destinationMunicipality", "program"],
            "optionalFields": ["observations"]
        }' | jq '.'

    log_info "Region backend creates document types on region channel..."
    log_info "Creating Region document type: Municipal Receipt"
    curl -s -X POST "$REGION_API/$REGION_CHANNEL/document-types" \
        -H "Content-Type: application/json" \
        -d '{
            "id": "municipal-receipt",
            "name": "Municipal Receipt",
            "description": "Receipt of funds from state level",
            "requiredFields": ["sourceOrg", "program", "receiptDate"],
            "optionalFields": ["observations"]
        }' | jq '.'

    log_info "Creating Region document type: Municipal Expense"
    curl -s -X POST "$REGION_API/$REGION_CHANNEL/document-types" \
        -H "Content-Type: application/json" \
        -d '{
            "id": "municipal-expense",
            "name": "Municipal Expense",
            "description": "Municipal spending",
            "requiredFields": ["category", "vendor", "purpose"],
            "optionalFields": ["invoiceNumber"]
        }' | jq '.'

    log_success "Document types created on all channels"
}

# =============================================================================
# Scenario 2: Federal to State Transfer with Cross-Channel Anchoring
# =============================================================================

federal_to_state_transfer() {
    log_step "Scenario: Federal to State Transfer (Cross-Channel Anchoring)"

    echo -e "${CYAN}This scenario demonstrates cross-channel anchoring between Union and State${NC}"
    echo -e "${CYAN}Union backend initiates (port 3000), State backend acknowledges (port 3001)${NC}\n"

    log_info "Step 1: Union backend initiates transfer to State (R$ 10,000,000)"
    TRANSFER_RESPONSE=$(curl -s -X POST "$UNION_API/transfers/initiate" \
        -H "Content-Type: application/json" \
        -d '{
            "fromChannel": "union",
            "toChannel": "state",
            "toOrg": "StateMSP",
            "documentTypeId": "federal-transfer",
            "title": "Transfer to São Paulo - Education Program",
            "description": "Annual transfer for state education program",
            "amount": 10000000,
            "currency": "BRL",
            "data": {
                "destinationState": "São Paulo",
                "program": "FUNDEB",
                "legalBasis": "Lei 14.113/2020"
            }
        }')
    log_response "$TRANSFER_RESPONSE"

    TRANSFER_ID=$(echo "$TRANSFER_RESPONSE" | jq -r '.id')
    TRANSFER_HASH=$(echo "$TRANSFER_RESPONSE" | jq -r '.contentHash')

    if [ "$TRANSFER_ID" = "null" ] || [ -z "$TRANSFER_ID" ]; then
        log_error "Failed to create transfer"
        return 1
    fi

    log_success "Transfer document created on union channel"
    echo -e "${CYAN}  Transfer ID: $TRANSFER_ID${NC}"
    echo -e "${CYAN}  Content Hash: $TRANSFER_HASH${NC}"

    log_info "Step 2: Verify transfer document on union channel (readable by any backend)"
    echo -e "${YELLOW}Reading from Union backend:${NC}"
    curl -s "$UNION_API/$UNION_CHANNEL/documents/$TRANSFER_ID" | jq '.'

    log_info "Step 3: State backend acknowledges receipt (creates document with hash anchor)"
    ACK_RESPONSE=$(curl -s -X POST "$STATE_API/$STATE_CHANNEL/transfers/acknowledge" \
        -H "Content-Type: application/json" \
        -d "{
            \"sourceDocId\": \"$TRANSFER_ID\",
            \"sourceChannel\": \"union\",
            \"documentTypeId\": \"state-receipt\",
            \"title\": \"Receipt from Union - Education Program\",
            \"description\": \"Acknowledgment of FUNDEB transfer receipt\",
            \"data\": {
                \"sourceOrg\": \"UnionMSP\",
                \"program\": \"FUNDEB\",
                \"receiptDate\": \"$(date -I)\"
            }
        }")
    log_response "$ACK_RESPONSE"

    ACK_ID=$(echo "$ACK_RESPONSE" | jq -r '.id')
    LINKED_HASH=$(echo "$ACK_RESPONSE" | jq -r '.linkedDocHash')

    if [ "$ACK_ID" = "null" ] || [ -z "$ACK_ID" ]; then
        log_error "Failed to create acknowledgment"
        return 1
    fi

    log_success "Acknowledgment document created on state channel"
    echo -e "${CYAN}  Acknowledgment ID: $ACK_ID${NC}"
    echo -e "${CYAN}  Linked to Hash: $LINKED_HASH${NC}"

    log_info "Step 4: Verify acknowledgment document on state channel"
    echo -e "${YELLOW}Reading from State backend:${NC}"
    curl -s "$STATE_API/$STATE_CHANNEL/documents/$ACK_ID" | jq '.'

    log_info "Step 5: Verify anchor between documents (can use any backend)"
    echo -e "${YELLOW}Verifying from Union backend:${NC}"
    VERIFY_RESPONSE=$(curl -s -X POST "$UNION_API/anchors/verify" \
        -H "Content-Type: application/json" \
        -d "{
            \"sourceChannel\": \"union\",
            \"sourceDocId\": \"$TRANSFER_ID\",
            \"targetChannel\": \"state\",
            \"targetDocId\": \"$ACK_ID\"
        }")
    log_response "$VERIFY_RESPONSE"

    IS_VALID=$(echo "$VERIFY_RESPONSE" | jq -r '.isValid')
    if [ "$IS_VALID" = "true" ]; then
        log_success "✓ Anchor verification PASSED - documents are cryptographically linked!"
    else
        log_error "✗ Anchor verification FAILED"
        echo "Mismatch reasons: $(echo "$VERIFY_RESPONSE" | jq -r '.mismatchReason')"
    fi

    log_info "Step 6: Get linked documents view from state channel"
    curl -s "$STATE_API/$STATE_CHANNEL/documents/$ACK_ID/linked" | jq '.'

    export LAST_TRANSFER_ID=$TRANSFER_ID
    export LAST_ACK_ID=$ACK_ID
}

# =============================================================================
# Scenario 3: State to Region Transfer
# =============================================================================

state_to_region_transfer() {
    log_step "Scenario: State to Region Transfer"

    echo -e "${CYAN}State backend initiates (port 3001), Region backend acknowledges (port 3002)${NC}\n"

    log_info "Step 1: State backend initiates transfer to Region (R$ 1,000,000)"
    TRANSFER_RESPONSE=$(curl -s -X POST "$STATE_API/transfers/initiate" \
        -H "Content-Type: application/json" \
        -d '{
            "fromChannel": "state",
            "toChannel": "region",
            "toOrg": "RegionMSP",
            "documentTypeId": "state-transfer",
            "title": "Transfer to Campinas Region - Health Program",
            "description": "Quarterly transfer for regional health services",
            "amount": 1000000,
            "currency": "BRL",
            "data": {
                "destinationMunicipality": "Campinas",
                "program": "SUS Regional"
            }
        }')
    log_response "$TRANSFER_RESPONSE"

    TRANSFER_ID=$(echo "$TRANSFER_RESPONSE" | jq -r '.id')

    if [ "$TRANSFER_ID" = "null" ] || [ -z "$TRANSFER_ID" ]; then
        log_error "Failed to create transfer"
        return 1
    fi

    log_success "Transfer initiated on state channel: $TRANSFER_ID"

    log_info "Step 2: Region backend acknowledges receipt"
    ACK_RESPONSE=$(curl -s -X POST "$REGION_API/$REGION_CHANNEL/transfers/acknowledge" \
        -H "Content-Type: application/json" \
        -d "{
            \"sourceDocId\": \"$TRANSFER_ID\",
            \"sourceChannel\": \"state\",
            \"documentTypeId\": \"municipal-receipt\",
            \"title\": \"Receipt from State - Health Program\",
            \"description\": \"Acknowledgment of SUS transfer\",
            \"data\": {
                \"sourceOrg\": \"StateMSP\",
                \"program\": \"SUS Regional\",
                \"receiptDate\": \"$(date -I)\"
            }
        }")
    log_response "$ACK_RESPONSE"

    ACK_ID=$(echo "$ACK_RESPONSE" | jq -r '.id')

    log_info "Step 3: Verify anchor (from any backend)"
    VERIFY_RESPONSE=$(curl -s -X POST "$UNION_API/anchors/verify" \
        -H "Content-Type: application/json" \
        -d "{
            \"sourceChannel\": \"state\",
            \"sourceDocId\": \"$TRANSFER_ID\",
            \"targetChannel\": \"region\",
            \"targetDocId\": \"$ACK_ID\"
        }")
    log_response "$VERIFY_RESPONSE"

    IS_VALID=$(echo "$VERIFY_RESPONSE" | jq -r '.isValid')
    if [ "$IS_VALID" = "true" ]; then
        log_success "✓ State → Region transfer verified!"
    else
        log_error "✗ Verification failed"
    fi
}

# =============================================================================
# Scenario 4: Region to State Transfer (Reverse Flow)
# =============================================================================

region_to_state_transfer() {
    log_step "Scenario: Region to State Transfer (Reverse - Municipal Reporting)"

    echo -e "${CYAN}Region backend initiates (port 3002), State backend acknowledges (port 3001)${NC}\n"

    log_info "Step 1: Region backend reports spending back to State (R$ 250,000)"
    TRANSFER_RESPONSE=$(curl -s -X POST "$REGION_API/transfers/initiate" \
        -H "Content-Type: application/json" \
        -d '{
            "fromChannel": "region",
            "toChannel": "state",
            "toOrg": "StateMSP",
            "documentTypeId": "municipal-expense",
            "title": "Municipal Health Spending Report - Q1",
            "description": "Report of health program expenditures to State",
            "amount": 250000,
            "currency": "BRL",
            "data": {
                "category": "Health Services",
                "vendor": "Multiple Vendors",
                "purpose": "Primary care services and equipment",
                "reportingPeriod": "Q1 2025"
            }
        }')
    log_response "$TRANSFER_RESPONSE"

    TRANSFER_ID=$(echo "$TRANSFER_RESPONSE" | jq -r '.id')
    TRANSFER_HASH=$(echo "$TRANSFER_RESPONSE" | jq -r '.contentHash')

    if [ "$TRANSFER_ID" = "null" ] || [ -z "$TRANSFER_ID" ]; then
        log_error "Failed to create transfer"
        return 1
    fi

    log_success "Region → State transfer initiated"
    echo -e "${CYAN}  Transfer ID: $TRANSFER_ID${NC}"
    echo -e "${CYAN}  Content Hash: $TRANSFER_HASH${NC}"

    log_info "Step 2: State backend acknowledges municipal spending report"
    ACK_RESPONSE=$(curl -s -X POST "$STATE_API/$STATE_CHANNEL/transfers/acknowledge" \
        -H "Content-Type: application/json" \
        -d "{
            \"sourceDocId\": \"$TRANSFER_ID\",
            \"sourceChannel\": \"region\",
            \"documentTypeId\": \"state-receipt\",
            \"title\": \"Acknowledgment - Municipal Health Spending\",
            \"description\": \"State acknowledgment of municipal spending report\",
            \"data\": {
                \"sourceOrg\": \"RegionMSP\",
                \"program\": \"Municipal Health Report\",
                \"receiptDate\": \"$(date -I)\",
                \"reportType\": \"Expenditure Report\"
            }
        }")
    log_response "$ACK_RESPONSE"

    ACK_ID=$(echo "$ACK_RESPONSE" | jq -r '.id')
    LINKED_HASH=$(echo "$ACK_RESPONSE" | jq -r '.linkedDocHash')

    log_success "Acknowledgment created"
    echo -e "${CYAN}  Ack ID: $ACK_ID${NC}"
    echo -e "${CYAN}  Linked Hash: $LINKED_HASH${NC}"

    log_info "Step 3: Verify Region → State anchor"
    VERIFY_RESPONSE=$(curl -s -X POST "$UNION_API/anchors/verify" \
        -H "Content-Type: application/json" \
        -d "{
            \"sourceChannel\": \"region\",
            \"sourceDocId\": \"$TRANSFER_ID\",
            \"targetChannel\": \"state\",
            \"targetDocId\": \"$ACK_ID\"
        }")
    log_response "$VERIFY_RESPONSE"

    IS_VALID=$(echo "$VERIFY_RESPONSE" | jq -r '.isValid')
    if [ "$IS_VALID" = "true" ]; then
        log_success "✓ Region → State anchor verification PASSED!"
    else
        log_error "✗ Verification failed"
    fi

    log_info "Step 4: View linked documents from state channel perspective"
    curl -s "$STATE_API/$STATE_CHANNEL/documents/$ACK_ID/linked" | jq '.'
}

# =============================================================================
# Scenario 5: State to Federal Transfer (Reverse - State Reporting)
# =============================================================================

state_to_federal_transfer() {
    log_step "Scenario: State to Federal Transfer (Reverse - State Reporting)"

    echo -e "${CYAN}CROSS-CHANNEL ANCHORING: State → Union${NC}"
    echo -e "${CYAN}State backend initiates (port 3001), Union backend acknowledges (port 3000)${NC}\n"

    log_info "Step 1: State backend reports consolidated spending to Federal (R$ 2,500,000)"
    TRANSFER_RESPONSE=$(curl -s -X POST "$STATE_API/transfers/initiate" \
        -H "Content-Type: application/json" \
        -d '{
            "fromChannel": "state",
            "toChannel": "union",
            "toOrg": "UnionMSP",
            "documentTypeId": "state-transfer",
            "title": "State Education Spending Report - FUNDEB",
            "description": "Consolidated report of FUNDEB program expenditures",
            "amount": 2500000,
            "currency": "BRL",
            "data": {
                "destinationMunicipality": "Federal Government",
                "program": "FUNDEB Accountability Report",
                "reportingPeriod": "Semester 1 2025",
                "municipalities": 50,
                "studentsServed": 125000
            }
        }')
    log_response "$TRANSFER_RESPONSE"

    TRANSFER_ID=$(echo "$TRANSFER_RESPONSE" | jq -r '.id')
    TRANSFER_HASH=$(echo "$TRANSFER_RESPONSE" | jq -r '.contentHash')

    if [ "$TRANSFER_ID" = "null" ] || [ -z "$TRANSFER_ID" ]; then
        log_error "Failed to create transfer"
        return 1
    fi

    log_success "State → Federal transfer initiated"
    echo -e "${CYAN}  Transfer ID: $TRANSFER_ID${NC}"
    echo -e "${CYAN}  Content Hash: $TRANSFER_HASH${NC}"

    log_info "Step 2: Union backend acknowledges state spending report"
    ACK_RESPONSE=$(curl -s -X POST "$UNION_API/$UNION_CHANNEL/transfers/acknowledge" \
        -H "Content-Type: application/json" \
        -d "{
            \"sourceDocId\": \"$TRANSFER_ID\",
            \"sourceChannel\": \"state\",
            \"documentTypeId\": \"federal-expense\",
            \"title\": \"Acknowledgment - State FUNDEB Report\",
            \"description\": \"Federal acknowledgment of state education spending\",
            \"data\": {
                \"category\": \"Education Accountability\",
                \"vendor\": \"State of São Paulo\",
                \"contractNumber\": \"FUNDEB-2025\",
                \"reportingState\": \"São Paulo\"
            }
        }")
    log_response "$ACK_RESPONSE"

    ACK_ID=$(echo "$ACK_RESPONSE" | jq -r '.id')
    LINKED_HASH=$(echo "$ACK_RESPONSE" | jq -r '.linkedDocHash')

    log_success "Federal acknowledgment created"
    echo -e "${CYAN}  Ack ID: $ACK_ID${NC}"
    echo -e "${CYAN}  Linked Hash: $LINKED_HASH${NC}"

    log_info "Step 3: Verify State → Federal anchor"
    VERIFY_RESPONSE=$(curl -s -X POST "$UNION_API/anchors/verify" \
        -H "Content-Type: application/json" \
        -d "{
            \"sourceChannel\": \"state\",
            \"sourceDocId\": \"$TRANSFER_ID\",
            \"targetChannel\": \"union\",
            \"targetDocId\": \"$ACK_ID\"
        }")
    log_response "$VERIFY_RESPONSE"

    IS_VALID=$(echo "$VERIFY_RESPONSE" | jq -r '.isValid')
    if [ "$IS_VALID" = "true" ]; then
        log_success "✓ State → Federal anchor verification PASSED!"
    else
        log_error "✗ Verification failed"
    fi

    log_info "Step 4: View linked documents from federal channel"
    curl -s "$UNION_API/$UNION_CHANNEL/documents/$ACK_ID/linked" | jq '.'

}

# =============================================================================
# Scenario 6: Document Invalidation and Correction
# =============================================================================

document_invalidation() {
    log_step "Scenario: Document Invalidation and Correction"

    echo -e "${CYAN}Union API endpoint (non-cross-channel operation)${NC}\n"

    log_info "Step 1: Union backend creates a document with incorrect amount"
    DOC_RESPONSE=$(curl -s -X POST "$UNION_API/$UNION_CHANNEL/documents" \
        -H "Content-Type: application/json" \
        -d '{
            "documentTypeId": "federal-expense",
            "title": "Equipment Purchase - Ministry of Health",
            "description": "Purchase of medical equipment",
            "amount": 500000,
            "currency": "BRL",
            "data": {
                "category": "Medical Equipment",
                "vendor": "MedTech Solutions",
                "contractNumber": "CT-2024-001"
            }
        }')
    log_response "$DOC_RESPONSE"

    WRONG_DOC_ID=$(echo "$DOC_RESPONSE" | jq -r '.id')
    log_info "Document with error created: $WRONG_DOC_ID"

    log_info "Step 2: Create correction document with correct amount"
    CORRECTION_RESPONSE=$(curl -s -X POST "$UNION_API/$UNION_CHANNEL/documents" \
        -H "Content-Type: application/json" \
        -d "{
            \"documentTypeId\": \"federal-expense\",
            \"title\": \"Equipment Purchase - Ministry of Health (CORRECTED)\",
            \"description\": \"Purchase of medical equipment - Corrected amount\",
            \"amount\": 550000,
            \"currency\": \"BRL\",
            \"data\": {
                \"category\": \"Medical Equipment\",
                \"vendor\": \"MedTech Solutions\",
                \"contractNumber\": \"CT-2024-001\",
                \"correction\": true,
                \"corrects\": \"$WRONG_DOC_ID\"
            }
        }")
    log_response "$CORRECTION_RESPONSE"

    CORRECTION_DOC_ID=$(echo "$CORRECTION_RESPONSE" | jq -r '.id')
    log_info "Correction document created: $CORRECTION_DOC_ID"

    log_info "Step 3: Invalidate the incorrect document"
    curl -s -X POST "$UNION_API/$UNION_CHANNEL/documents/$WRONG_DOC_ID/invalidate" \
        -H "Content-Type: application/json" \
        -d "{
            \"reason\": \"Incorrect amount. Correct amount is R\$ 550,000.00\",
            \"correctionDocId\": \"$CORRECTION_DOC_ID\"
        }" | jq '.'

    log_info "Step 4: Verify invalidated document"
    curl -s "$UNION_API/$UNION_CHANNEL/documents/$WRONG_DOC_ID" | jq '.'

    log_info "Step 5: View document history"
    curl -s "$UNION_API/$UNION_CHANNEL/documents/$WRONG_DOC_ID/history" | jq '.'

    log_success "Document invalidation completed"
}

# =============================================================================
# Scenario 7: Internal Spending (Contractors & Equipment - No Anchoring)
# =============================================================================

internal_spending() {
    log_step "Scenario: Internal Spending (Contractors & Equipment)"

    echo -e "${CYAN}This scenario demonstrates standalone documents WITHOUT cross-channel anchoring.${NC}"
    echo -e "${CYAN}These are internal expenses: contractors, equipment, utilities, etc.${NC}"
    echo -e "${CYAN}Each backend writes to its own channel - NO cross-channel anchoring.${NC}"
    echo ""

    log_info "Step 1: Union backend records contractor payment (R$ 250,000)"
    CONTRACTOR_RESPONSE=$(curl -s -X POST "$UNION_API/$UNION_CHANNEL/documents" \
        -H "Content-Type: application/json" \
        -d '{
            "documentTypeId": "federal-expense",
            "title": "Software Development - Tech Solutions Ltda",
            "description": "Payment for citizen portal development - Phase 1",
            "amount": 250000,
            "currency": "BRL",
            "data": {
                "category": "IT Services",
                "vendor": "Tech Solutions Ltda",
                "contractNumber": "CT-2025-042",
                "serviceDescription": "Custom citizen portal development",
                "invoiceNumber": "INV-2025-001",
                "taxId": "12.345.678/0001-90",
                "deliveryDate": "2025-01-15"
            }
        }')
    log_response "$CONTRACTOR_RESPONSE"

    CONTRACTOR_DOC_ID=$(echo "$CONTRACTOR_RESPONSE" | jq -r '.id')

    if [ "$CONTRACTOR_DOC_ID" = "null" ] || [ -z "$CONTRACTOR_DOC_ID" ]; then
        log_error "Failed to create contractor payment"
        return 1
    fi

    log_success "Contractor payment recorded on union channel"
    echo -e "${CYAN}  Document ID: $CONTRACTOR_DOC_ID${NC}"
    echo -e "${CYAN}  Note: linkedDocId is empty (no anchoring)${NC}"

    log_info "Step 2: View contractor payment details"
    CONTRACTOR_DOC=$(curl -s "$UNION_API/$UNION_CHANNEL/documents/$CONTRACTOR_DOC_ID")
    echo "$CONTRACTOR_DOC" | jq '{
        id,
        title,
        amount,
        linkedDocId,
        linkedChannel,
        "vendor": .data.vendor,
        "contractNumber": .data.contractNumber
    }'

    log_info "Step 3: State backend records equipment purchase (R$ 850,000)"
    EQUIPMENT_RESPONSE=$(curl -s -X POST "$STATE_API/$STATE_CHANNEL/documents" \
        -H "Content-Type: application/json" \
        -d '{
            "documentTypeId": "state-transfer",
            "title": "Medical Equipment - MRI Scanner",
            "description": "MRI Scanner purchase for Hospital Regional",
            "amount": 850000,
            "currency": "BRL",
            "data": {
                "destinationMunicipality": "N/A - Equipment Purchase",
                "program": "Health Infrastructure",
                "vendor": "MedEquip Brazil",
                "equipmentType": "MRI Scanner",
                "model": "Siemens Magnetom Vida",
                "quantity": 1,
                "warrantyYears": 5,
                "installationDate": "2025-02-01"
            }
        }')
    log_response "$EQUIPMENT_RESPONSE"

    EQUIPMENT_DOC_ID=$(echo "$EQUIPMENT_RESPONSE" | jq -r '.id')
    log_success "Equipment purchase recorded on state channel"
    echo -e "${CYAN}  Document ID: $EQUIPMENT_DOC_ID${NC}"

    log_info "Step 4: Region backend records consulting service (R$ 75,000)"
    CONSULTING_RESPONSE=$(curl -s -X POST "$REGION_API/$REGION_CHANNEL/documents" \
        -H "Content-Type: application/json" \
        -d '{
            "documentTypeId": "municipal-expense",
            "title": "IT Consulting - Cloud Migration",
            "description": "Consulting services for municipal systems cloud migration",
            "amount": 75000,
            "currency": "BRL",
            "data": {
                "category": "Consulting Services",
                "vendor": "CloudTech Consultoria",
                "purpose": "Cloud migration planning and execution",
                "projectName": "Municipal Cloud Migration",
                "duration": "3 months",
                "deliverables": "Architecture design, Migration plan, Staff training"
            }
        }')
    log_response "$CONSULTING_RESPONSE"

    CONSULTING_DOC_ID=$(echo "$CONSULTING_RESPONSE" | jq -r '.id')
    log_success "Consulting service recorded on region channel"
    echo -e "${CYAN}  Document ID: $CONSULTING_DOC_ID${NC}"

    log_info "Step 5: Union backend records utility expense (R$ 125,000)"
    UTILITY_RESPONSE=$(curl -s -X POST "$UNION_API/$UNION_CHANNEL/documents" \
        -H "Content-Type: application/json" \
        -d '{
            "documentTypeId": "federal-expense",
            "title": "Electricity - Government Complex",
            "description": "Monthly electricity bill for federal government complex",
            "amount": 125000,
            "currency": "BRL",
            "data": {
                "category": "Utilities",
                "vendor": "Eletrobras",
                "contractNumber": "UTIL-2025-E001",
                "utilityType": "Electricity",
                "accountNumber": "ACC-12345",
                "billingPeriod": "2025-01",
                "consumption": "150000 kWh"
            }
        }')
    log_response "$UTILITY_RESPONSE"

    UTILITY_DOC_ID=$(echo "$UTILITY_RESPONSE" | jq -r '.id')
    log_success "Utility expense recorded on union channel"

    log_info "Step 6: Query all expenses on union channel (includes contractors)"
    curl -s "$UNION_API/$UNION_CHANNEL/documents?documentTypeId=federal-expense" | \
        jq '{total, documents: [.documents[] | {id, title, amount, vendor: .data.vendor}]}'

    log_info "Step 7: Query high-value expenses across all channels (> R$ 200K)"
    echo "Union channel (> R$ 200K):"
    curl -s "$UNION_API/$UNION_CHANNEL/documents?minAmount=200000" | \
        jq '{count: .total, documents: [.documents[] | {title, amount}]}'

    echo ""
    echo "State channel (> R$ 200K):"
    curl -s "$STATE_API/$STATE_CHANNEL/documents?minAmount=200000" | \
        jq '{count: .total, documents: [.documents[] | {title, amount}]}'

    log_info "Step 8: Compare anchored vs non-anchored documents"

    echo -e "${GREEN}Non-Anchored Document (Contractor):${NC}"
    curl -s "$UNION_API/$UNION_CHANNEL/documents/$CONTRACTOR_DOC_ID" | \
        jq '{
            id,
            title,
            amount,
            linkedDocId: .linkedDocId,
            linkedChannel: .linkedChannel,
            linkedDirection: .linkedDirection,
            note: (if .linkedDocId == "" then "No anchoring - standalone document" else "Anchored to another channel" end)
        }'

    echo ""
    echo -e "${GREEN}Anchored Document (Transfer - if exists from previous scenarios):${NC}"

    TRANSFER_DOC=$(curl -s "$UNION_API/$UNION_CHANNEL/documents?linkedDirection=OUTGOING" | jq -r '.documents[0].id // empty')

    if [ -n "$TRANSFER_DOC" ]; then
        curl -s "$UNION_API/$UNION_CHANNEL/documents/$TRANSFER_DOC" | \
            jq '{
                id,
                title,
                amount,
                linkedDocId: .linkedDocId,
                linkedChannel: .linkedChannel,
                linkedDirection: .linkedDirection,
                note: (if .linkedDocId != "" then "Anchored to " + .linkedChannel + " channel" else "Not anchored" end)
            }'
    else
        echo "No anchored transfers found (run federal-state scenario first)"
    fi
}

# =============================================================================
# Scenario 8: Query Documents
# =============================================================================

query_documents() {
    log_step "Scenario: Query Documents"

    echo -e "${CYAN}All Union API endpoints (read operations - no cross-channel anchoring needed)${NC}\n"

    log_info "Query all documents on union channel"
    curl -s "$UNION_API/$UNION_CHANNEL/documents" | jq '.'

    log_info "Query documents with amount > 100000"
    curl -s "$UNION_API/$UNION_CHANNEL/documents?minAmount=100000" | jq '.'

    log_info "Query active documents only"
    curl -s "$UNION_API/$UNION_CHANNEL/documents?status=ACTIVE" | jq '.'

    log_info "Query outgoing transfers (documents with cross-channel links)"
    curl -s "$UNION_API/$UNION_CHANNEL/documents?linkedDirection=OUTGOING" | jq '.'

    log_info "Query incoming documents on state channel"
    curl -s "$STATE_API/$STATE_CHANNEL/documents?linkedDirection=INCOMING" | jq '.'
}

# =============================================================================
# Main
# =============================================================================

run_all() {
    check_api
    setup_document_types
    sleep 2
    federal_to_state_transfer
    sleep 2
    state_to_region_transfer
    sleep 2
    region_to_state_transfer
    sleep 2
    state_to_federal_transfer
    sleep 2
    document_invalidation
    sleep 2
    internal_spending
    sleep 2
    query_documents
    
    log_step "All test scenarios completed!"
}

case "${1:-}" in
    check)
        check_api
        ;;
    types)
        setup_document_types
        ;;
    federal-state)
        federal_to_state_transfer
        ;;
    state-region)
        state_to_region_transfer
        ;;
    region-state)
        region_to_state_transfer
        ;;
    state-federal)
        state_to_federal_transfer
        ;;
    invalidate)
        document_invalidation
        ;;
    internal)
        internal_spending
        ;;
    query)
        query_documents
        ;;
    all)
        run_all
        ;;
    *)
        echo "Government Spending Blockchain - Test Scenarios"
        echo "==============================================="
        echo ""
        echo "Usage: $0 {check|types|federal-state|state-region|region-state|state-federal|invalidate|internal|query|all}"
        echo ""
        echo "ARCHITECTURE:"
        echo "  Union Backend  → localhost:3000 (admin: union channel)"
        echo "  State Backend  → localhost:3001 (admin: state channel)"
        echo "  Region Backend → localhost:3002 (admin: region channel)"
        echo ""
        echo "COMMANDS:"
        echo "  check         - Check if all backend instances are running"
        echo "  types         - Setup document types on all channels"
        echo ""
        echo "CROSS-CHANNEL ANCHORING SCENARIOS:"
        echo "  federal-state - Union → State transfer with cryptographic anchoring"
        echo "  state-federal - State → Union transfer (reverse flow with anchoring)"
        echo "  state-region  - State → Region transfer"
        echo "  region-state  - Region → State transfer (reverse flow)"
        echo ""
        echo "NON-CROSS-CHANNEL SCENARIOS (Union API):"
        echo "  invalidate    - Document invalidation and correction"
        echo "  internal      - Internal spending (contractors, equipment, utilities)"
        echo "  query         - Document queries across channels"
        echo ""
        echo "  all           - Run all scenarios (complete test suite)"
        echo ""
        echo "CONFIGURATION:"
        echo "  Union API:  $UNION_API"
        echo "  State API:  $STATE_API"
        echo "  Region API: $REGION_API"
        echo "  Channels:   $UNION_CHANNEL, $STATE_CHANNEL, $REGION_CHANNEL"
        echo ""
        echo "EXAMPLES:"
        echo "  $0 check          # Check all backends are running"
        echo "  $0 types          # Setup document types"
        echo "  $0 federal-state  # Test Union → State cross-channel anchoring"
        echo "  $0 internal       # Test internal spending (no cross-channel)"
        echo "  $0 all            # Run complete test suite"
        echo ""
        echo "KEY CONCEPTS:"
        echo "  • Cross-channel anchoring: Documents link across channels using content hashes"
        echo "  • Union ↔ State: Full bidirectional anchoring demonstrated"
        echo "  • Each backend writes only to its own channel (admin rights)"
        echo "  • All backends can read from all channels"
        echo "  • Anchor verification works from any backend"
        exit 1
        ;;
esac