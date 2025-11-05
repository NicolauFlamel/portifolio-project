package models

type Transaction struct {
	TxID         string `json:"tx_id"`
	FromEntity   string `json:"from_entity"`
	ToEntity     string `json:"to_entity"`
	Amount       int    `json:"amount"`
	LinkedTxHash string `json:"linked_tx_hash,omitempty"`
	Timestamp    string `json:"timestamp"`
	Description  string `json:"description,omitempty"`
}