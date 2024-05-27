package memo

import (
	"time"

	daclient "github.com/memoio/go-da-client"
)

var (
	getTimeout = time.Minute
)

// middleware client
type DAClient struct {
	Client     *daclient.MemoDAClient
	GetTimeout time.Duration
}

func NewDAClient(rpc string) (*DAClient, error) {
	// ban DA
	if rpc == defaultDaRpc {
		return nil, nil
	}
	client := daclient.NewMemoDAClient(rpc)
	err := client.Start()
	if err != nil {
		return nil, err
	}
	return &DAClient{
		Client:     client,
		GetTimeout: getTimeout,
	}, nil
}
