package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/safwentrabelsi/tx-json-rpc-server/types"
	"github.com/stretchr/testify/require"
)

// check the message and make a better order

const (
	validTransactionRawHex = "0x02f87683aa36a78230198459682f008459682f10825208948d7526216e3c4294345ecf45ad57f9aebacfb0c487038d7ea4c6800080c080a0d93d292d7076aebac2f6eb373bc41807efdaea264472101667f978c564321b39a0226cd31db40298041f86b26a007b6f93ee563e9763c07544a9b7cafa4643624a"
	existingTransactionRaw = "0x02f8b483aa36a7824c2884391ed39884391ed39882c1eb944370841dbd5d8dbcc7028109f580eaaf65b90b4080b8446eb5441e636c646269717a726a303030643337366c6d6c6c7a6235316d0000000000000000000000000000000000000000000000000000000000000000000000000003e8c080a0f006568cd70fca2772ea6f92a4a09e9bd4df0783e85e8c4de5613207e225cfb0a06e37c03b4645e75b15cab0d15b54e433c6c29e92d6b5e9c73dc000838597b3c6"
	invalidTransactionRawHex = "0x3e3598fb8aabc3733686dd0a7a84ea35e25a34d959a68b9aeb1f5c5f7ab5877a"
	validTransactionHash = "0x3e3598fb8aabc3733686dd0a7a84ea35e25a34d959a68b9aeb1f5c5f7ab5877a"
	notFoundTransactionHash = "0xae2f861e03fc34b5a7960c43bfc57ff2d847328ac9bd2422ee27bfdbe73c8719"
)

// Mock for the EthTransactionService interface
type mockEthService struct{}



func (m *mockEthService) StoreTransaction(tx types.Transaction) error {
	if tx.RawHex == existingTransactionRaw {
		return errors.New("already STORED")
	}
	return nil
}

func (m *mockEthService) CancelTransaction(hash string) error {
	if hash == notFoundTransactionHash {
		return errors.New("transaction not found")
	}
	return nil
}

func (m *mockEthService) SendRequest(ctx context.Context, body io.Reader, headers http.Header) (*http.Response, error) {
	// Emulte the response of eth_chainId which isn't handled by this proxy
	return &http.Response{
        StatusCode: http.StatusOK,
        Body: io.NopCloser(bytes.NewBufferString(`{"jsonrpc": "2.0","id": 1,"result": "0x1"}`)),
	},nil
}

// This test suite is designed to test the handleRequest function, as well as the functions it depends on: isValidHexRawTx, isValidTxHash and proxyToRPCNode.
// To test more realistic scenarios and error propagation.
func TestHandleRequest(t *testing.T) {
	// Initialize a mock EthService
	service := &EthService{EthClient: &mockEthService{}}

	t.Run("when receiving a malformed JSON request, return an error", func(t *testing.T) {
		handler := http.HandlerFunc(service.handleRequest)
		rr := makeRequest(t, handler, "POST", "/", strings.NewReader("invalid json"))

		resp := parseAndCheckResponse(t, rr, http.StatusOK, nil, "2.0")
		require.Contains(t, resp.Error.Message, "invalid json request")
	})
	
	t.Run("when receiving a well formed JSON request, process it correctly", func(t *testing.T) {
		validRequest := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"eth_sendRawTransaction","params":["%s"]}`,validTransactionRawHex)

		handler := http.HandlerFunc(service.handleRequest)
		rr := makeRequest(t, handler, "POST", "/", strings.NewReader(validRequest))

		resp := parseAndCheckResponse(t, rr, http.StatusOK, float64(1), "2.0")
		require.Nil(t, resp.Error)

	})
	t.Run("when receiving a JSON request with empty params, return an error", func(t *testing.T) {
		invalidRequest := `{"jsonrpc":"2.0","id":1,"method":"eth_sendRawTransaction","params":[]}`

		handler := http.HandlerFunc(service.handleRequest)
		rr := makeRequest(t, handler, "POST", "/", strings.NewReader(invalidRequest))

		resp := parseAndCheckResponse(t, rr, http.StatusOK, float64(1), "2.0")
		require.Contains(t, resp.Error.Message, "invalid parameters: not enough params to decode")
		require.Equal(t,resp.Error.Code, -32602 )

	})
	t.Run("when receiving a JSON request with an int as param, return an error", func(t *testing.T) {
		invalidRequest := `{"jsonrpc":"2.0","id":1,"method":"eth_sendRawTransaction","params":[1]}`

		handler := http.HandlerFunc(service.handleRequest)
		rr := makeRequest(t, handler, "POST", "/", strings.NewReader(invalidRequest))

		resp := parseAndCheckResponse(t, rr, http.StatusOK, float64(1), "2.0")
		require.Contains(t, resp.Error.Message, "invalid params")
		require.Equal(t,resp.Error.Code, -32602 )
	})
	t.Run("when receiving a valid request but the transaction an invalid hex string, return an error", func(t *testing.T) {
		invalidRequest := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"eth_sendRawTransaction","params":["%s"]}`,invalidTransactionRawHex)

		handler := http.HandlerFunc(service.handleRequest)
		rr := makeRequest(t, handler, "POST", "/", strings.NewReader(invalidRequest))

		resp := parseAndCheckResponse(t, rr, http.StatusOK, float64(1), "2.0")
		require.Contains(t, resp.Error.Message, "invalid params")
		require.Equal(t,resp.Error.Code, -32602 )

	})

	t.Run("when receiving a valid request but the transaction is invalid, return an error", func(t *testing.T) {
		invalidRequest := `{"jsonrpc":"2.0","id":1,"method":"eth_sendRawTransaction","params":["0xInvalid"]}`

		handler := http.HandlerFunc(service.handleRequest)
		rr := makeRequest(t, handler, "POST", "/", strings.NewReader(invalidRequest))

		resp := parseAndCheckResponse(t, rr, http.StatusOK, float64(1), "2.0")
		require.Contains(t, resp.Error.Message, "invalid params")
		require.Equal(t,resp.Error.Code, -32602 )

	})

	t.Run("when receiving a valid request but the StoreTransaction returns an error of already Stored transaction, return an error", func(t *testing.T) {
		invalidRequest := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"eth_sendRawTransaction","params":["%s"]}`,existingTransactionRaw)

		handler := http.HandlerFunc(service.handleRequest)
		rr := makeRequest(t, handler, "POST", "/", strings.NewReader(invalidRequest))

		resp := parseAndCheckResponse(t, rr, http.StatusOK, float64(1), "2.0")
		require.Contains(t, resp.Error.Message, "already STORED")
		require.Equal(t,resp.Error.Code, -32000 )
	})
	t.Run("when receiving a cancel_transaction request with a valid transaction hash, process it correctly", func(t *testing.T) {
		validRequest := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"cancel_transaction","params":["%s"]}`,validTransactionHash)

		handler := http.HandlerFunc(service.handleRequest)
		rr := makeRequest(t, handler, "POST", "/", strings.NewReader(validRequest))

		resp := parseAndCheckResponse(t, rr, http.StatusOK, float64(1), "2.0")
		require.Nil(t, resp.Error)
		require.Equal(t, "Transaction canceled", resp.Result)
	})

	t.Run("when receiving a cancel_transaction JSON request with empty params, return an error", func(t *testing.T) {
		invalidRequest := `{"jsonrpc":"2.0","id":1,"method":"cancel_transaction","params":[]}`

		handler := http.HandlerFunc(service.handleRequest)
		rr := makeRequest(t, handler, "POST", "/", strings.NewReader(invalidRequest))

		resp := parseAndCheckResponse(t, rr, http.StatusOK, float64(1), "2.0")
		require.Contains(t, resp.Error.Message, "invalid parameters: not enough params to decode")
		require.Equal(t,resp.Error.Code, -32602 )
	})

	t.Run("when receiving a cancel_transaction JSON request with not a string param, return an error", func(t *testing.T) {
		invalidRequest := `{"jsonrpc":"2.0","id":1,"method":"cancel_transaction","params":[1]}`

		handler := http.HandlerFunc(service.handleRequest)
		rr := makeRequest(t, handler, "POST", "/", strings.NewReader(invalidRequest))

		resp := parseAndCheckResponse(t, rr, http.StatusOK, float64(1), "2.0")
		require.Contains(t, resp.Error.Message, "invalid params")
		require.Equal(t,resp.Error.Code, -32602 )
	})
	t.Run("when receiving a cancel_transaction request with an invalid transaction hash, return an error", func(t *testing.T) {
		invalidRequest := `{"jsonrpc":"2.0","id":1,"method":"cancel_transaction","params":["0xInvalid"]}`

		handler := http.HandlerFunc(service.handleRequest)
		rr := makeRequest(t, handler, "POST", "/", strings.NewReader(invalidRequest))

		resp := parseAndCheckResponse(t, rr, http.StatusOK, float64(1), "2.0")
		require.Contains(t, resp.Error.Message, "invalid params")
		require.Equal(t,resp.Error.Code, -32602 )

	})
	t.Run("when receiving a cancel_transaction request with a valid transaction hash but it was not found, return an error", func(t *testing.T) {
		invalidRequest := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"cancel_transaction","params":["%s"]}`,notFoundTransactionHash)

		handler := http.HandlerFunc(service.handleRequest)
		rr := makeRequest(t, handler, "POST", "/", strings.NewReader(invalidRequest))

		resp := parseAndCheckResponse(t, rr, http.StatusOK, float64(1), "2.0")
		require.Contains(t, resp.Error.Message, "transaction not found")
		require.Equal(t,resp.Error.Code, -32000 )
	})

	// Tests the default case and the proxyToRPCNode at once.
	t.Run("when receiving a method that is not handled by the server, process it correctly", func(t *testing.T) {
		unhandledMethodRequest := `{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}`

		handler := http.HandlerFunc(service.handleRequest)
		rr := makeRequest(t, handler, "POST", "/", strings.NewReader(unhandledMethodRequest))

		resp := parseAndCheckResponse(t, rr, http.StatusOK, float64(1), "2.0")
		require.Nil(t, resp.Error)
		require.Equal(t, "0x1", resp.Result)
	})

}

// Test raw hex transaction validation.
func TestIsValidHexRawTx(t *testing.T) {
	tests := []struct {
		name    string
		rawTx   interface{}
		wantErr bool
	}{
		{
			name:    "Valid hex string",
			rawTx:   validTransactionRawHex,
			wantErr: false,
		},
		{
			name:    "Invalid hex string (missing 0x prefix)",
			rawTx:   "abcdef",
			wantErr: true,
		},
		{
			name:    "Invalid hex string (not a string)",
			rawTx:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := isValidHexRawTx(tt.rawTx); (err != nil) != tt.wantErr {
				t.Errorf("isValidHexRawTx() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test transaction hash validation.
func TestIsValidTxHash(t *testing.T) {
	tests := []struct {
		name    string
		param   interface{}
		wantErr bool
	}{
		{
			name:    "Valid transaction hash",
			param:   validTransactionHash,
			wantErr: false,
		},
		{
			name:    "Invalid transaction hash (missing 0x prefix)",
			param:   "a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1",
			wantErr: true,
		},
		{
			name:    "Invalid transaction hash (not a string)",
			param:   123,
			wantErr: true,
		},
		{
			name:    "Invalid transaction hash (incorrect length)",
			param:   validTransactionRawHex+"1",
			wantErr: true,
		},
		{
			name:    "Invalid transaction hash (non-hex content)",
			param:   "0xg0h1i2j3k4l5m6n7o8p9q0r1s2t3u4v5w6x7y8z9a0b1c2d3e4f5g6h7i8j9",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := isValidTxHash(tt.param); (err != nil) != tt.wantErr {
				t.Errorf("isValidTxHash() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test panic recovery middleware.
func TestRecoverPanic(t *testing.T) {

	// Create a handler that panics
	handler := func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}

	handler = recoverPanic(handler)

	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	require.Equal(t, http.StatusOK, w.Code) 
	require.Contains(t, w.Body.String(), "server error") 
}


// Test helpers.

// makeRequest is a helper function for making HTTP requests.
func makeRequest(t *testing.T, handler http.HandlerFunc, method, url string, body io.Reader) *httptest.ResponseRecorder {
	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)
	
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	return rr
}

// parseAndCheckResponse is a helper function to parse and check the HTTP response.
func parseAndCheckResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatusCode int, expectedID interface{}, expectedJsonrpc string) types.JSONRPCResponse {
	require.Equal(t, expectedStatusCode, rr.Code)

	var resp types.JSONRPCResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	require.Equal(t, expectedJsonrpc, resp.Jsonrpc)
	require.Equal(t, expectedID, resp.ID)

	return resp
}