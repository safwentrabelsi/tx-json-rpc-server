package rpc

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/safwentrabelsi/tx-json-rpc-server/config"
	"github.com/safwentrabelsi/tx-json-rpc-server/types"
	log "github.com/sirupsen/logrus"
)

// EthServiceInterface defines the interface for Ethereum services.
type EthServiceInterface interface {
    StoreTransaction( tx types.Transaction) error
	CancelTransaction(hex string) error
	SendRequest(ctx context.Context,body io.Reader, headers http.Header) (*http.Response, error)
}

// EthService is a service struct that uses an implementation of the EthTransactionService interface.
type EthService struct {
	EthClient EthServiceInterface
}

// StartServer initializes and starts the server with provided EthServiceInterface implementation and listening address.
func StartServer(ec EthServiceInterface) error {
	addr := config.GetConfig().Addr()
	service := &EthService{EthClient: ec}
	http.HandleFunc("/", recoverPanic(service.handleRequest))
	log.Info("Starting server on :",addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Error("Failed to start server: ", err)
		return err
	}
	return nil
}

// handleRequest handles incoming HTTP requests by decoding the JSON RPC request and processing the request based on the specified method.
func (s *EthService) handleRequest(w http.ResponseWriter, r *http.Request) {
    var req types.JSONRPCRequest
    bodyBytes, err := io.ReadAll(r.Body)
    if err != nil {
        log.Error("Failed to read request body: ", err)
		writeJSONRPCError(w, req.ID, -32700, "parse error")
        return
    }
    bodyReader := bytes.NewReader(bodyBytes)

    err = json.NewDecoder(bytes.NewBuffer(bodyBytes)).Decode(&req)
    if err != nil {
        log.Error("Failed to decode request body: ", err)
		writeJSONRPCError(w, req.ID, -32600, "invalid json request")
        return
    }

	// For the proxy, make sure to reset the reader.
    bodyReader.Seek(0, io.SeekStart)

	switch req.Method {
	case "eth_sendRawTransaction":
		res := types.JSONRPCResponse{
			Jsonrpc: "2.0",
			ID: req.ID,
		}
		if len(req.Params) > 0 {
			// Validate the raw transaction hex.
			err = isValidHexRawTx(req.Params[0])
			if  err != nil {
				log.Error(err.Error())
				writeJSONRPCError(w, req.ID, -32602, "invalid params")
				return
			}
			rawHex := req.Params[0].(string)
			// Decode to bytes
			bytesTx, err := hex.DecodeString(rawHex[2:]) 
			if err != nil {
				log.Error("Failed to decode transaction data: ", err.Error())
				writeJSONRPCError(w, req.ID, -32602, "invalid params")
				return
			}
			// Unmarshal to tx type.
			tx := types.Transaction{}
			err = tx.UnmarshalBinary(bytesTx)
			if err != nil {
				log.Error("Failed to unmarshal transaction data: ", err.Error())
				writeJSONRPCError(w, req.ID, -32602, "invalid params")
				return
			}

			// Store transaction with its raw hex.
			tx.RawHex = rawHex
			err = s.EthClient.StoreTransaction(tx)
			if err != nil {
				log.Error(err.Error())
				writeJSONRPCError(w, req.ID, -32000, err.Error())
				return
			}
			// Return transaction hash.
			res.Result = tx.Hash().String()
			} else {
				// No params receiverd
				log.Error("Failed to retrieve raw transaction")
				writeJSONRPCError(w, req.ID, -32602, "invalid parameters: not enough params to decode")
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(res)
		break
	case "cancel_transaction":
		res := types.JSONRPCResponse{
			Jsonrpc: "2.0",
			ID: req.ID,
		}

		if len(req.Params) > 0 {
			// Valide the transaction hash
			err = isValidTxHash(req.Params[0])
			if  err != nil {
				log.Error(err.Error())
				writeJSONRPCError(w, req.ID, -32602, "invalid params")
				return
			}

			// Cancel the transaction.
			err := s.EthClient.CancelTransaction(req.Params[0].(string))
			if err != nil {
				log.Error(err.Error())
				writeJSONRPCError(w, req.ID, -32000, err.Error())
				return
			}
			// Return message as a result.
			res.Result = "Transaction canceled"
			} else {
				// No params receiverd
				log.Error("Failed to retrieve transaction hash")
				writeJSONRPCError(w, req.ID, -32602, "invalid parameters: not enough params to decode")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(res)
		break
		default:
			s.proxyToRPCNode(w, r, bodyReader)

		}
	}
	

// proxyToRPCNode is used to forward requests that are not handled by the EthService to the Ethereum RPC node.	
func (s *EthService) proxyToRPCNode(w http.ResponseWriter, r *http.Request,body io.Reader) {
	resp, err := s.EthClient.SendRequest(r.Context(), body, r.Header)
	if err != nil {
		log.Error("Failed to send request: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	io.Copy(w, resp.Body)
}

	
// writeJSONRPCError is a utility function to write JSON RPC error responses.
func writeJSONRPCError(w http.ResponseWriter, id interface{}, code int, message string) {
	res := types.JSONRPCResponse{
		Jsonrpc: "2.0",
		ID:      id,
		Error: &types.JSONRPCError{
			Code: code,
			Message: message,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// isValidHexRawTx validates if the provided raw transaction.
func isValidHexRawTx(rawTx interface{}) error {
	rawTxStr, ok := rawTx.(string)
	if !ok {
		return fmt.Errorf("the param is not a string")
	}

	if  rawTxStr[:2] != "0x" {
		return fmt.Errorf("invalid transaction hex")
	}
	// No need to decode since the hex will be decoded after this validation.
	return nil
}

// isValidTxHash validates if the provided transaction hash is valid
func isValidTxHash(param interface{}) error {
	hashStr, ok := param.(string)
	if !ok {
		return fmt.Errorf("the param is not a string")
	}
	
	if len(hashStr) != 66 ||  hashStr[:2] != "0x" {
		return fmt.Errorf("invalid transaction hash")
	}

	_, err := hex.DecodeString(hashStr[2:])
	if err != nil {
		return fmt.Errorf("invalid transaction hash: %w", err)
	}

	return nil
}


// Recover panic middleware.
func recoverPanic(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("panic: %+v", err)
				// Id should be the request.ID but to retrieve it in this middleware would harm the performance.
				writeJSONRPCError(w, nil, -32000, "server error")
			}
		}()

		next(w, r)
	}
}