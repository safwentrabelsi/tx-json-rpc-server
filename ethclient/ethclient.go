package ethclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/safwentrabelsi/tx-json-rpc-server/config"
	"github.com/safwentrabelsi/tx-json-rpc-server/types"

	log "github.com/sirupsen/logrus"
)

// HTTPDoer interface defines a single method Do that takes an http.Request and returns an http.Response.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// EthClient is a struct that represents the Ethereum client which interacts with the Ethereum network.
type EthClient struct {
	URL    string
	Client HTTPDoer
	storedTransactions map[string]types.Transaction
	transactionsMutex  *sync.Mutex
	gasMonitoringFrequence time.Duration
}

var (

	// Client is the instance of the Ethereum client.
	Client *EthClient


	// Define allowed state transition for a transaction
	allowedTransitions = map[types.TransactionStatus][]types.TransactionStatus{
		types.STORED:    {types.CANCELED, types.SPEDUP, types.FAILED, types.BROADCASTED},
		types.CANCELED:  {types.SPEDUP},
		types.SPEDUP:    {},
		types.FAILED:    {},
		types.BROADCASTED: {},
	}

)

const txHashField = "tx_hash"

// Init function initializes the global Ethereum client with the configured URL and an HTTP client.
func Init()  {
	cfg := config.GetConfig()
	Client = &EthClient{
		URL:        cfg.URL(),
		Client:    &http.Client{
			Timeout: time.Second * 10, 
		},
		storedTransactions: make(map[string]types.Transaction),
		transactionsMutex:  &sync.Mutex{},
		gasMonitoringFrequence: 5 * time.Second,
	}
}

// doRequest is a helper function that sends an HTTP request to the Ethereum network and returns the response.
func (ec *EthClient) doRequest(ctx context.Context,  reqBody []byte) (*types.JSONRPCResponse, error) {
	var respBody types.JSONRPCResponse
	// Prepare headers.
	headers := http.Header{}
	headers.Add("Content-Type", "application/json")

	resp, err := ec.SendRequest(ctx, bytes.NewBuffer(reqBody), headers)
	if err != nil {
		log.Error("failed to make request: ", err)
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected http status code: %v", resp.StatusCode)
		log.Error(err)
		return nil, err
	}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		log.Error("failed to decode response body: ", err)
		return nil, err
	}

	return &respBody, nil
}

// SendRequest sends an HTTP request to the Ethereum network.
func (ec *EthClient) SendRequest(ctx context.Context, body io.Reader, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,  ec.URL, body)
	if err != nil {
		return nil, err
	}
	req.Header = headers
	return ec.Client.Do(req)
}


// sendTransaction sends a raw transaction to the Ethereum network.
func (ec *EthClient) sendTransaction(ctx context.Context, hex string)( rpcError bool,err error) {
	reqBody, err := json.Marshal(types.JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  "eth_sendRawTransaction",
		Params:  []interface{}{hex},
		ID:      1,
	})

	if err != nil {
		return false, err
	}
	

	resp, err := ec.doRequest(ctx,reqBody)
	if err != nil {
		return false,err
	}

	if resp.Error != nil  {
		return true,errors.New(resp.Error.Message)
	}

	log.WithField(txHashField,resp.Result).Info("Transaction sent successfully")

	return false,nil
}

// getGasPrice fetches the current gas price from the Ethereum network.
func (ec *EthClient) getGasPrice(ctx context.Context) (float64, error) {
	reqBody, err := json.Marshal(types.JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  "eth_gasPrice",
		Params:  []interface{}{},
		ID:      1,
	})

	if err != nil {
		return 0, err
	}

	resp, err := ec.doRequest(ctx, reqBody)
	if err != nil {
		return 0, err
	}

	if resp.Error != nil  {
		return 0, errors.New(resp.Error.Message)
	}

	gasPrice, err := strconv.ParseInt(resp.Result.(string)[2:], 16, 64)
	if err != nil {
		return 0, err
	}

	return float64(gasPrice), nil
}


// StoreTransaction stores a transaction in memory.
func (ec *EthClient) StoreTransaction( tx types.Transaction) error {
	hash := tx.Hash().String()
	isCancelingTx := false
	for oldHash, oldTx := range ec.storedTransactions{

		if oldHash == hash  {
			// This returns an error because an Ethereum node will return an error as well with a message: "already known".
			return fmt.Errorf("already %s",oldTx.Status.String())	
		}

		// If the transaction is SPEDUP it means that there is another transaction stored that the user wanted to cancel or even speed up.
		if oldTx.Status == types.SPEDUP {
			continue
		}
		// Get the sender address from the oldtx.
		oldFromAddress, err := ethTypes.Sender(ethTypes.LatestSignerForChainID(tx.ChainId()), &oldTx.Transaction)
		if err != nil {
			log.Error("failed to get sender address from stored transaction ",err)
			continue
		}	

		// Get the sender address from the new tx.
		newFromAddress, err := ethTypes.Sender(ethTypes.LatestSignerForChainID(tx.ChainId()), &tx.Transaction)
		if err != nil {
			log.Error("failed to get sender address from new transaction ",err)
			break
		}
		// If the same wallet is sending a transaction with the same nonce usually it's to either cancel or speed up a transaction.
		if	oldFromAddress == newFromAddress && tx.Nonce() == oldTx.Nonce()  {
			// the gas caps
			gasCap := tx.GasFeeCap().Int64() + tx.GasTipCap().Int64()
			oldGasCap := oldTx.GasFeeCap().Int64() + oldTx.GasTipCap().Int64()
			// In case of a cancel transaction in a metamask way.
			if  newFromAddress == *tx.To() && tx.Value().Int64() == 0  &&  gasCap > oldGasCap && len(tx.Data())== 0  {
				isCancelingTx = true
				err = ec.changeTransactionStatus(oldHash, types.CANCELED)
				// This a way to ensure that all the transaction from the same sender are being cancelled in the scenario of a user
				// cancelling a transaction then sending another one with the same nonce then trying to cancel it again.
				if err != nil {
					continue 
				}
				log.WithField(txHashField,oldHash).Info("Canceled transaction")
			return nil
			}
			// In case of a speed up transaction in a metamask way.
			if *tx.To() == *oldTx.To() && tx.Value().Int64() == oldTx.Value().Int64() &&  gasCap > oldGasCap && bytes.Equal(tx.Data(),oldTx.Data()) {
				err = ec.changeTransactionStatus(oldHash, types.SPEDUP)
				if err != nil {
					return err
				}
				tx.Status = types.STORED
				ec.storedTransactions[hash] = tx
				log.WithField(txHashField,oldHash).Info("Sped up transaction")
				return nil
			}
			
		}
	}

	// No need to store cancelling transactions since subbmitting them will be a total loss of gas.
	if isCancelingTx {
		return nil
	}
	tx.Status = types.STORED
	ec.storedTransactions[hash] = tx
	log.WithField(txHashField,hash).Info("Stored transaction")
	return nil
}

// CancelTransaction changes the status of a transaction to canceled.
func (ec *EthClient) CancelTransaction(hash string) error {
err := ec.changeTransactionStatus(hash,types.CANCELED)
if err != nil {
	return err
}
log.WithField(txHashField,hash).Info("Canceled transaction")
return nil
}

// changeTransactionStatus is a helper function that changes the status of a transaction.
func  (ec *EthClient) changeTransactionStatus(hash string, newStatus types.TransactionStatus) error {

	ec.transactionsMutex.Lock()
	defer ec.transactionsMutex.Unlock()

	trx, ok := ec.storedTransactions[hash]
	if !ok {
		return errors.New("transaction not found") 
	}

	// Check if the new status is an allowed transition
	for _, allowedStatus := range allowedTransitions[trx.Status] {
		if newStatus == allowedStatus {
			trx.Status = newStatus
			ec.storedTransactions[hash] = trx
			return nil
		}
	}

	return fmt.Errorf("invalid status transition from %s to %s for transaction: %s", trx.Status.String(), newStatus.String(), hash)
}

// MonitorGas monitors gas prices and submits transactions when the gas price is low enough.
func (ec *EthClient) MonitorGas(ctx context.Context) {
	ticker := time.NewTicker(ec.gasMonitoringFrequence)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			gasPrice, err := ec.getGasPrice(ctx)
			if err != nil {
				log.Error("failed to get gas price: ", err)
				continue
			}
				for hash, tx := range ec.storedTransactions {
					if tx.Status != types.STORED {
						continue
					}
					if tx.GasFeeCap().Int64() +  tx.GasTipCap().Int64() >= int64(gasPrice) {
						ec.transactionsMutex.Lock()
						isRPCErr,err := ec.sendTransaction(ctx, tx.RawHex)
						if err != nil {
							log.Error("failed to send transaction: ", err)
							// If invalid transaction e.g: nonce too low, already known transaction....
							if isRPCErr {
								ec.transactionsMutex.Unlock()
								err = ec.changeTransactionStatus(hash,types.FAILED)
								if err != nil {
									// This error will never happen since only stored transaction are sent and the transaition from STORED to FAILED is allowed
									log.Error(err.Error())
								}
							}
						} else {
							ec.transactionsMutex.Unlock()
							err = ec.changeTransactionStatus(hash,types.BROADCASTED)
							if err != nil {
								// This error will never happen since only stored transaction are sent and the transaition from STORED to BROADCASTED is allowed
								log.Error(err.Error())
							}
						}
					}
			}
		case <-ctx.Done():
			return
		}
	}
}
