package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// TransferContract handles money transfers
type TransferContract struct {
	contractapi.Contract
}

// Transfer represents a money transfer record
type Transfer struct {
	ID          string    `json:"id"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"` // pending, completed, rejected
}

// Balance represents an entity's balance
type Balance struct {
	Entity string  `json:"entity"`
	Amount float64 `json:"amount"`
}

// InitLedger initializes the ledger with some test data
func (t *TransferContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	// Initialize Union balance
	unionBalance := Balance{
		Entity: "union",
		Amount: 1000000.0, // Starting with 1 million
	}

	balanceJSON, err := json.Marshal(unionBalance)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState("balance_union", balanceJSON)
	if err != nil {
		return fmt.Errorf("failed to initialize union balance: %v", err)
	}

	return nil
}

// RecordTransfer records a new transfer from union to state/county
func (t *TransferContract) RecordTransfer(ctx contractapi.TransactionContextInterface, transferID, from, to string, amount float64, description string) error {
	// Check if transfer already exists
	exists, err := t.TransferExists(ctx, transferID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("transfer %s already exists", transferID)
	}

	// Create transfer record
	transfer := Transfer{
		ID:          transferID,
		From:        from,
		To:          to,
		Amount:      amount,
		Description: description,
		Timestamp:   time.Now(),
		Status:      "pending",
	}

	transferJSON, err := json.Marshal(transfer)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(transferID, transferJSON)
	if err != nil {
		return fmt.Errorf("failed to record transfer: %v", err)
	}

	return nil
}

// CompleteTransfer completes a pending transfer and updates balances
func (t *TransferContract) CompleteTransfer(ctx contractapi.TransactionContextInterface, transferID string) error {
	// Get transfer
	transfer, err := t.GetTransfer(ctx, transferID)
	if err != nil {
		return err
	}

	if transfer.Status != "pending" {
		return fmt.Errorf("transfer %s is not pending", transferID)
	}

	// Get sender balance
	senderBalance, err := t.GetBalance(ctx, transfer.From)
	if err != nil {
		return err
	}

	// Check if sender has sufficient balance
	if senderBalance.Amount < transfer.Amount {
		return fmt.Errorf("insufficient balance for %s", transfer.From)
	}

	// Update sender balance
	senderBalance.Amount -= transfer.Amount
	senderBalanceJSON, err := json.Marshal(senderBalance)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutState("balance_"+transfer.From, senderBalanceJSON)
	if err != nil {
		return err
	}

	// Get or create receiver balance
	receiverBalance, err := t.GetBalance(ctx, transfer.To)
	if err != nil {
		receiverBalance = &Balance{
			Entity: transfer.To,
			Amount: 0,
		}
	}

	// Update receiver balance
	receiverBalance.Amount += transfer.Amount
	receiverBalanceJSON, err := json.Marshal(receiverBalance)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutState("balance_"+transfer.To, receiverBalanceJSON)
	if err != nil {
		return err
	}

	// Update transfer status
	transfer.Status = "completed"
	transferJSON, err := json.Marshal(transfer)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutState(transferID, transferJSON)
	if err != nil {
		return err
	}

	return nil
}

// GetTransfer returns a transfer by ID
func (t *TransferContract) GetTransfer(ctx contractapi.TransactionContextInterface, transferID string) (*Transfer, error) {
	transferJSON, err := ctx.GetStub().GetState(transferID)
	if err != nil {
		return nil, fmt.Errorf("failed to read transfer: %v", err)
	}
	if transferJSON == nil {
		return nil, fmt.Errorf("transfer %s does not exist", transferID)
	}

	var transfer Transfer
	err = json.Unmarshal(transferJSON, &transfer)
	if err != nil {
		return nil, err
	}

	return &transfer, nil
}

// GetBalance returns the balance of an entity
func (t *TransferContract) GetBalance(ctx contractapi.TransactionContextInterface, entity string) (*Balance, error) {
	balanceJSON, err := ctx.GetStub().GetState("balance_" + entity)
	if err != nil {
		return nil, fmt.Errorf("failed to read balance: %v", err)
	}
	if balanceJSON == nil {
		return nil, fmt.Errorf("balance for %s does not exist", entity)
	}

	var balance Balance
	err = json.Unmarshal(balanceJSON, &balance)
	if err != nil {
		return nil, err
	}

	return &balance, nil
}

// TransferExists checks if a transfer exists
func (t *TransferContract) TransferExists(ctx contractapi.TransactionContextInterface, transferID string) (bool, error) {
	transferJSON, err := ctx.GetStub().GetState(transferID)
	if err != nil {
		return false, fmt.Errorf("failed to read transfer: %v", err)
	}

	return transferJSON != nil, nil
}

// GetAllTransfers returns all transfers
func (t *TransferContract) GetAllTransfers(ctx contractapi.TransactionContextInterface) ([]*Transfer, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var transfers []*Transfer
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var transfer Transfer
		err = json.Unmarshal(queryResponse.Value, &transfer)
		if err != nil {
			continue // Skip non-transfer records
		}

		// Only include if it has transfer structure
		if transfer.ID != "" {
			transfers = append(transfers, &transfer)
		}
	}

	return transfers, nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(&TransferContract{})
	if err != nil {
		fmt.Printf("Error creating transfer chaincode: %v\n", err)
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting transfer chaincode: %v\n", err)
	}
}