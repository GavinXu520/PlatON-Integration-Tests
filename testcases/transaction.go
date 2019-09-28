package testcases

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	ethereum "github.com/PlatONnetwork/PlatON-Go"
	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/common/hexutil"
	"github.com/PlatONnetwork/PlatON-Go/core/types"
	"github.com/PlatONnetwork/PlatON-Go/ethclient"
)

type TxParams struct {
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	Gas      string         `json:"gas"`
	GasPrice string         `json:"gasPrice"`
	Value    string         `json:"value"`
	Data     string         `json:"data"`
}

type Response struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  string `json:"result"`
	Id      int    `json:"id"`
	Error   struct {
		Code    int32  `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type UnlockResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  bool   `json:"result"`
	Id      int    `json:"id"`
	Error   struct {
		Code    int32  `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func SendRawTransaction(ctx context.Context, client *ethclient.Client, from *PriAccount, to common.Address, value string, data []byte) (common.Hash, error) {
	v := new(big.Int)
	var err error
	if strings.HasPrefix(value, "0x") {
		bigValue, err := hexutil.DecodeBig(value)
		if err != nil {
			panic(err)
		}
		v = bigValue
	} else {
		tmp, ok := new(big.Int).SetString(value, 10)
		if !ok {
			panic(fmt.Sprintf("transfer value to int error.%s", err))
		}
		v = tmp
	}
	var msg ethereum.CallMsg
	msg.Data = data
	msg.Value = v
	msg.To = &to
	msg.From = from.Address
	gas, err := client.EstimateGas(ctx, msg)
	if err != nil {
		return common.ZeroHash, fmt.Errorf("EstimateGas fail:%v", err)
	}
	gasPrise, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return common.ZeroHash, fmt.Errorf("SuggestGasPrice fail:%v", err)
	}
	nonce, err := client.NonceAt(ctx, from.Address, nil)
	if err != nil {
		return common.ZeroHash, fmt.Errorf("get nonce fail:%v", err)
	}

	if from.Nonce < nonce {
		from.Nonce = nonce
	}

	newTx := getSignedTransaction(from, to, v, gas, gasPrise, data)

	if err := client.SendTransaction(ctx, newTx); err != nil {
		panic(err)
	}
	from.Nonce = from.Nonce + 1
	return newTx.Hash(), nil
}

func getSignedTransaction(from *PriAccount, to common.Address, value *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *types.Transaction {
	newTx, err := types.SignTx(types.NewTransaction(from.Nonce, to, value, gasLimit, gasPrice, data), types.NewEIP155Signer(new(big.Int).SetInt64(102)), from.Priv)
	if err != nil {
		panic(fmt.Errorf("sign error,%s", err.Error()))
	}
	return newTx
}
