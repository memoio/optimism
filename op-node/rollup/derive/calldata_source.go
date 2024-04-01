package derive

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	memo "github.com/ethereum-optimism/optimism/op-memo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var daClient *memo.DAClient

func SetDAClient(c *memo.DAClient) error {
	if daClient != nil {
		return errors.New("da client already configured")
	}
	daClient = c
	return nil
}

// CalldataSource is a fault tolerant approach to fetching data.
// The constructor will never fail & it will instead re-attempt the fetcher
// at a later point.
type CalldataSource struct {
	// Internal state + data
	open bool
	data []eth.Data
	// Required to re-attempt fetching
	ref     eth.L1BlockRef
	dsCfg   DataSourceConfig
	fetcher L1TransactionFetcher
	log     log.Logger

	batcherAddr common.Address
}

// NewCalldataSource creates a new calldata source. It suppresses errors in fetching the L1 block if they occur.
// If there is an error, it will attempt to fetch the result on the next call to `Next`.
func NewCalldataSource(ctx context.Context, log log.Logger, dsCfg DataSourceConfig, fetcher L1TransactionFetcher, ref eth.L1BlockRef, batcherAddr common.Address) DataIter {
	_, txs, err := fetcher.InfoAndTxsByHash(ctx, ref.Hash)
	if err != nil {
		return &CalldataSource{
			open:        false,
			ref:         ref,
			dsCfg:       dsCfg,
			fetcher:     fetcher,
			log:         log,
			batcherAddr: batcherAddr,
		}
	}
	return &CalldataSource{
		open: true,
		data: DataFromEVMTransactions(dsCfg, batcherAddr, txs, log.New("origin", ref)),
	}
}

// Next returns the next piece of data if it has it. If the constructor failed, this
// will attempt to reinitialize itself. If it cannot find the block it returns a ResetError
// otherwise it returns a temporary error if fetching the block returns an error.
func (ds *CalldataSource) Next(ctx context.Context) (eth.Data, error) {
	if !ds.open {
		if _, txs, err := ds.fetcher.InfoAndTxsByHash(ctx, ds.ref.Hash); err == nil {
			ds.open = true
			ds.data = DataFromEVMTransactions(ds.dsCfg, ds.batcherAddr, txs, ds.log)
		} else if errors.Is(err, ethereum.NotFound) {
			return nil, NewResetError(fmt.Errorf("failed to open calldata source: %w", err))
		} else {
			return nil, NewTemporaryError(fmt.Errorf("failed to open calldata source: %w", err))
		}
	}
	if len(ds.data) == 0 {
		return nil, io.EOF
	} else {
		data := ds.data[0]
		ds.data = ds.data[1:]
		return data, nil
	}
}

// DataFromEVMTransactions filters all of the transactions and returns the calldata from transactions
// that are sent to the batch inbox address from the batch sender address.
// This will return an empty array if no valid transactions are found.
func DataFromEVMTransactions(dsCfg DataSourceConfig, batcherAddr common.Address, txs types.Transactions, log log.Logger) []eth.Data {
	out := []eth.Data{}
	for _, tx := range txs {
		if isValidBatchTx(tx, dsCfg.l1Signer, dsCfg.batchInboxAddress, batcherAddr) {
			// MeeDA: if the calldata is represented in MemoDerivation marker, then fetch it from MeeDA layer
			if daClient == nil {
				out = append(out, tx.Data())
				continue
			}
			data := tx.Data()
			switch len(data) {
			case 0:
				out = append(out, data)
			default:
				switch data[0] {
				case memo.DerivationVersionMemo:
					log.Info("MeeDA: blob request", "id", hex.EncodeToString(tx.Data()))
					ctx, cancel := context.WithTimeout(context.Background(), daClient.GetTimeout)
					blobs, err := daClient.Client.Get(ctx, [][]byte{data[1:]})
					cancel()
					if err != nil {
						log.Warn("MeeDA: failed to resolve frame", "err", err)
						log.Info("MeeDA: using eth fallback")
						out = append(out, data)
						continue
					}
					if len(blobs) != 1 {
						log.Warn("MeeDA: unexpected length for blobs", "expected", 1, "got", len(blobs))
						if len(blobs) == 0 {
							log.Warn("MeeDA: skipping empty blobs")
							continue
						}
					}
					out = append(out, blobs[0])
				default:
					out = append(out, data)
					log.Info("MeeDA: using eth fallback")
				}
			}
		}
	}
	return out
}
