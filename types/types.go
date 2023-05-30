package types

import "github.com/ethereum/go-ethereum/core/types"

// JSONRPCRequest defines the structure of an incoming JSON-RPC request.
type JSONRPCRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  []interface{} `json:"params"`
	ID      interface{}    `json:"id"`
}


// JSONRPCResponse defines the structure of a JSON-RPC response.
type JSONRPCResponse struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError defines the structure of an error in a JSON-RPC response.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// TransactionStatus represents the current status of a transaction.
type TransactionStatus int

// Enum values for TransactionStatus.
const (
	STORED TransactionStatus = iota
	CANCELED
	SPEDUP
	FAILED
	BROADCASTED
)

// String method provides a string representation for the TransactionStatus enum.
func (s TransactionStatus) String() string {
	return [...]string{"STORED", "CANCELED", "SPEDUP","FAILED","BROADCASTED"}[s]
}

// Transaction struct extends the go-ethereum core Transaction type with application-specific fields.
type Transaction struct {
	types.Transaction
	Status TransactionStatus
	RawHex string
}

