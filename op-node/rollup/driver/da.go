package driver

import (
	memo "github.com/ethereum-optimism/optimism/op-memo"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

func SetDAClient(cfg memo.Config) error {
	client, err := memo.NewDAClient(cfg.DaRpc)
	if err != nil {
		return err
	}
	return derive.SetDAClient(client)
}
