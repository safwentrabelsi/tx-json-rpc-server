package ethclient

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/safwentrabelsi/tx-json-rpc-server/types"
	"github.com/stretchr/testify/require"
)

const (
	validTransactionRawHex = "0x02f8b705728419f9d5908419f9d5908303e8b7941b696ea9f880ff3d57212cdc0c5542d56ccc36c2872386f26fc10000b84483f818b400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000012a2f34dc080a04e7d2a7780cb0e8f32a1fe2b2e9e21a66045d4f8d2c2cb0a7888aefd19e333d2a058752571b1305b410a1a06afd0c8044a9d6cfe604fe3e760fdd7c84d1f48c6bf"
	existingTransactionRaw = "0x02f8680518808082520894ef803a51bc4bcc28edf32713713b6135edbb9d7d865af3107a400080c001a06559a1bc72373a7bb8610472fb56dcc3949c2c489c000138313a4ebf35b0688ba04e7f520a9d669019aa08d9a1f67aeff90e4ef88aff3611848ab05a4ec6e5ecab"
	validTransactionHash = "0x3e3598fb8aabc3733686dd0a7a84ea35e25a34d959a68b9aeb1f5c5f7ab5877a"
	tx1SpeedUpRaw = "0x02f8700518843b9aca0084b1c5b8a882520894ef803a51bc4bcc28edf32713713b6135edbb9d7d865af3107a400080c080a0f24d3eec94e624666e2ed4326be36e60b2cf16fae9f27c3acbe40744ddafbb69a046cc9d34e94c9712548e38f5ebb4bee7987b4b9797c4288332e6799411018d69"
	tx1CancelRaw = "0x02f86a0518843b9aca00849ac5650e825208943ac6b727d731c171b84ad65622922222ddcf03c78080c001a045f0f6cb7352d12be07779d67812b2f5630b9f9ff748cf4c81d76ab99ae5b5f4a00719c023746364fca6f77f79266849abb9db926876a39001df3fcd956ebbc5df" 
)

type MockDoer struct {
	Response *http.Response
	Err      error
}

func (m *MockDoer) Do(req *http.Request) (*http.Response, error) {
	return m.Response, m.Err
}

// test doRequest function.
func TestDoRequest(t *testing.T) {
	t.Run("it decodes the response body", func(t *testing.T) {
		client := &EthClient{
			Client: &MockDoer{
				Response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"jsonrpc": "2.0", "result": "0x5f5e100", "id":1}`)),
				},
			},
		}

		reqBody, _ := json.Marshal(types.JSONRPCRequest{
			Jsonrpc: "2.0",
			Method:  "eth_gasPrice",
			Params:  []interface{}{},
			ID:      1,
		})

		resp, err := client.doRequest(context.Background(), reqBody)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}


		require.Equal(t, "2.0", resp.Jsonrpc)
		require.Equal(t, float64(1), resp.ID)
		require.Nil(t, resp.Error)
		require.Equal(t, "0x5f5e100", resp.Result)
	})
}

// Tests sendTransaction function.
func TestSendTransaction(t *testing.T) {
	t.Run("it sends a transaction successfully", func(t *testing.T) {
		client := &EthClient{
			Client: &MockDoer{
				Response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(fmt.Sprintf(`{"jsonrpc": "2.0", "result": "%s", "id":1}`,validTransactionHash))),
				},
			},
		}


		rpcError, err := client.sendTransaction(context.Background(), validTransactionRawHex)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		require.False(t, rpcError)
		
	})
	t.Run("it handles server timeout", func(t *testing.T) {
		client := &EthClient{
			Client: &MockDoer{
					Err: errors.New("net/http: request canceled (Client.Timeout exceeded while awaiting headers)"),
			},
		}


		isRPCError, err := client.sendTransaction(context.Background(), validTransactionRawHex)
		if err == nil {
			t.Fatalf("expected error, got none")
		}

		require.False(t, isRPCError)
		require.EqualError(t, err, "net/http: request canceled (Client.Timeout exceeded while awaiting headers)")
	})

	t.Run("it handles JSONRPC errors", func(t *testing.T) {
		client := &EthClient{
			Client: &MockDoer{
				Response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"jsonrpc": "2.0", "error": {"code": -32000,"message":"nonce too low"}, "id":1}`)),
				},
			},
		}


		rpcError, err := client.sendTransaction(context.Background(), validTransactionRawHex)


		require.True(t, rpcError)
		require.Contains(t,err.Error(),"nonce too low")
		
	})
}

// tests the get getGasPrice function.
func TestGetGasPrice(t *testing.T) {

	t.Run("it gets the gas price successfully", func(t *testing.T) {
		client := &EthClient{
			Client: &MockDoer{
				Response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"jsonrpc": "2.0", "result": "0x5f5e100", "id":1}`)),
				},
			},
		}

		gasPrice, err := client.getGasPrice(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		require.Equal(t, gasPrice,float64(100000000))

	})
	
	t.Run("it returns error when doRequest fails", func(t *testing.T) {
		client := &EthClient{
			Client: &MockDoer{
				Err: errors.New("net/http: request canceled"),
			},
		}

		_, err := client.getGasPrice(context.Background())
		if err == nil || err.Error() != "net/http: request canceled" {
			t.Fatalf("expected net/http: request canceled error, got %v", err)
		}
	})

	t.Run("it returns error when response contains error", func(t *testing.T) {
		client := &EthClient{
			Client: &MockDoer{
				Response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"jsonrpc": "2.0", "error": {"code": -32000, "message": "Server error"}, "id":1}`)),
				},
			},
		}

		_, err := client.getGasPrice(context.Background())
		if err == nil || err.Error() != "Server error" {
			t.Fatalf("expected Method not found error, got %v", err)
		}
	})

	t.Run("it returns error when response cannot be parsed", func(t *testing.T) {
		client := &EthClient{
			Client: &MockDoer{
				Response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"jsonrpc": "2.0", "result": "invalid", "id":1}`)),
				},
			},
		}

		_, err := client.getGasPrice(context.Background())
		if err == nil || !strings.Contains(err.Error(), "invalid syntax") {
			t.Fatalf("expected invalid syntax error, got %v", err)
		}
	})
}

// tests the storeTransaction Function.
func TestStoreTransaction(t *testing.T) {

    // Test data
	// tx1
	tx1, err := getTxFromRaw(existingTransactionRaw) 
	if err != nil {
		t.Fatalf("Failed to decode transaction data: %v", err)
	}
	

	// tx2
	tx2, err := getTxFromRaw(validTransactionRawHex) 
	if err != nil {
		t.Fatalf("Failed to decode transaction data: %v", err)
	}


	// tx1 cancel transaction
	tx1Cancel, err := getTxFromRaw(tx1CancelRaw) 
	if err != nil {
		t.Fatalf("Failed to decode transaction data: %v", err)
	}

	// tx1 speed up transaction
	tx1SpeedUp, err := getTxFromRaw(tx1SpeedUpRaw) 
	if err != nil {
		t.Fatalf("Failed to decode transaction data: %v", err)
	}
	
	client := &EthClient{
		storedTransactions: map[string]types.Transaction{
			tx1.Hash().String(): *tx1,
		},
		transactionsMutex: &sync.Mutex{},

	}

    t.Run("store a new transaction", func(t *testing.T) {
        // Prepare a new transaction
        tx := tx2
        err := client.StoreTransaction(*tx)

        require.NoError(t, err)
        require.Equal(t, tx2.Status, client.storedTransactions[tx2.Hash().String()].Status)
    })

    t.Run("attempt to store a transaction with an existing hash", func(t *testing.T) {
        tx := tx1
        err := client.StoreTransaction(*tx)

        require.Error(t, err)
        require.Contains(t, err.Error(), "already STORED")
    })

    t.Run("attempt to cancel a transaction", func(t *testing.T) {
        tx := tx1Cancel
        err := client.StoreTransaction(*tx)

        require.NoError(t, err)
        require.Equal(t, types.CANCELED, client.storedTransactions[tx1.Hash().String()].Status)
    })

    t.Run("attempt to speed up a transaction", func(t *testing.T) {
        tx := tx1SpeedUp
        err := client.StoreTransaction(*tx)

        require.NoError(t, err)
        require.Equal(t, types.SPEDUP, client.storedTransactions[tx1.Hash().String()].Status)
        require.Equal(t, types.STORED, client.storedTransactions[tx.Hash().String()].Status)
    })
}

// tests the cancelTransaction function.
func TestCancelTransaction(t *testing.T) {
    // Test data
	// tx1
	tx1, err := getTxFromRaw(existingTransactionRaw) 
	if err != nil {
		t.Fatalf("Failed to decode transaction data: %v", err)
	}
	
	  // Initialize EthClient
    client := &EthClient{
		storedTransactions: map[string]types.Transaction{
			tx1.Hash().String(): *tx1,
		},
		transactionsMutex: &sync.Mutex{},

	}

	client.storedTransactions[tx1.Hash().String()] = *tx1

  

    t.Run("cancel an existing transaction", func(t *testing.T) {
        err := client.CancelTransaction(tx1.Hash().String())
        require.NoError(t, err)
        require.Equal(t, types.CANCELED, client.storedTransactions[tx1.Hash().String()].Status)
    })

    t.Run("attempt to cancel a non-existing transaction", func(t *testing.T) {
        err := client.CancelTransaction("non-existing")
        require.Error(t, err)
        require.Contains(t, err.Error(), "transaction not found")
    })
}


// tests the changeTransactionStatus function
func TestChangeTransactionStatus(t *testing.T) {
    // Test data
	
    // tx1
	tx1, err := getTxFromRaw(existingTransactionRaw) 
	if err != nil {
		t.Fatalf("Failed to decode transaction data: %v", err)
	}

    client := &EthClient{
		storedTransactions: map[string]types.Transaction{
			tx1.Hash().String(): *tx1,
		},
		transactionsMutex: &sync.Mutex{},

	}



    t.Run("valid status transition", func(t *testing.T) {
        err := client.changeTransactionStatus(tx1.Hash().String(), types.CANCELED)
        require.NoError(t, err)
        require.Equal(t, types.CANCELED, client.storedTransactions[tx1.Hash().String()].Status)
    })

    t.Run("invalid status transition", func(t *testing.T) {

		// Prepare the transaction
		tx1.Status = types.CANCELED
		client.storedTransactions[tx1.Hash().String()] = *tx1


        err := client.changeTransactionStatus(tx1.Hash().String(), types.STORED)
        require.Error(t, err)
        require.Contains(t, err.Error(), "invalid status transition")
        require.Equal(t, types.CANCELED, client.storedTransactions[tx1.Hash().String()].Status)
    })

    t.Run("non-existing transaction", func(t *testing.T) {
        err := client.changeTransactionStatus("non-existing", types.CANCELED)
        require.Error(t, err)
        require.Contains(t, err.Error(), "transaction not found")
    })
}

// For the gasMonitor test I will to mock the do function to be able to read the body twice.
type MonitorGasMockDoer struct {
	Response *http.Response
	Err      error
}
func (m *MonitorGasMockDoer) Do(req *http.Request) (*http.Response, error) {
	body := io.NopCloser(strings.NewReader(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	return &http.Response{
		StatusCode: http.StatusOK,
		Body: body,
	}, nil
}
func TestMonitorGas(t *testing.T) {

	t.Run("broadcast the transaction at the right gas price", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// prepare data
		tx, err := getTxFromRaw(tx1SpeedUpRaw) 
		if err != nil {
			t.Fatalf("Failed to decode transaction data: %v", err)
		}

		
		ec := &EthClient{
				storedTransactions: map[string]types.Transaction{
					tx.Hash().String(): *tx,
				},
				transactionsMutex: &sync.Mutex{},
				gasMonitoringFrequence: time.Millisecond * 50,
				Client: &MonitorGasMockDoer{},
			}
		
		go ec.MonitorGas(ctx)

		// Give the MonitorGas method some time to run
		time.Sleep(time.Millisecond * 60)

		require.Equal(t, types.BROADCASTED, ec.storedTransactions[tx.Hash().String()].Status)
	})

	t.Run("gas price isn't low enough to broadcast transaction.", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// prepare data
		tx, err := getTxFromRaw(existingTransactionRaw) 
		if err != nil {
			t.Fatalf("Failed to decode transaction data: %v", err)
		}

		
		ec := &EthClient{
				storedTransactions: map[string]types.Transaction{
					tx.Hash().String(): *tx,
				},
				transactionsMutex: &sync.Mutex{},
				gasMonitoringFrequence: time.Millisecond * 50,
				Client: &MonitorGasMockDoer{},
			}
		
		go ec.MonitorGas(ctx)

		// Give the MonitorGas method some time to run
		time.Sleep(time.Millisecond * 60)

		require.Equal(t, types.STORED, ec.storedTransactions[tx.Hash().String()].Status)
	})

	t.Run("try to broadcast the transaction but sendTransaction return an error but not rpcError", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// prepare data
		tx, err := getTxFromRaw(tx1SpeedUpRaw) 
		if err != nil {
			t.Fatalf("Failed to decode transaction data: %v", err)
		}

		
		ec := &EthClient{
				storedTransactions: map[string]types.Transaction{
					tx.Hash().String(): *tx,
				},
				transactionsMutex: &sync.Mutex{},
				gasMonitoringFrequence: time.Millisecond * 50,
				Client: &MockDoer{
					Response: &http.Response{
						StatusCode: http.StatusOK,
						Body:      io.NopCloser(strings.NewReader(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`)),
					},
				},
			}
		
		go ec.MonitorGas(ctx)

		// Give the MonitorGas method some time to run
		time.Sleep(time.Millisecond * 60)

		require.Equal(t, types.STORED, ec.storedTransactions[tx.Hash().String()].Status)
	})

}



// Test helpers.
func getTxFromRaw(rawHex string) (*types.Transaction,error){
	bytesTx, err := hex.DecodeString(rawHex[2:]) 
	 if err != nil {
		 return nil,err
	 }
	 tx := &types.Transaction{}
	 err = tx.UnmarshalBinary(bytesTx)
	 if err != nil {
		return nil,err
	 }
 
	 tx.Status = types.STORED
	 tx.RawHex = existingTransactionRaw

	 return tx,nil
}