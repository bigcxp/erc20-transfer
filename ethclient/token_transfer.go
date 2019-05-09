package ethclient

import (
	"context"
	"math/big"
	"time"

	"github.com/858chain/erc20-transfer/utils"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/crypto/sha3"
)

type TransferRequest struct {
	contractAddress string
	fromAddress     string
	toAddress       string
	amount          float64
}

func (c *Client) TokenTranser(contractName, contractAddress, fromAddress, toAddress string, amount float64) (string, error) {
	hexContractAddress := common.HexToAddress(contractAddress)
	hexFromAddress := common.HexToAddress(fromAddress)
	hexToAddress := common.HexToAddress(toAddress)

	nonce, err := c.PendingNonceAt(context.Background(), hexFromAddress)
	if err != nil {
		return "", err
	}

	value := big.NewInt(0) // in wei (0 eth)
	gasPrice, err := c.SuggestGasPrice(context.Background())
	if err != nil {
		return "", err
	}

	transferFnSignature := []byte("transfer(address,uint256)")
	hash := sha3.NewLegacyKeccak256()
	hash.Write(transferFnSignature)
	methodID := hash.Sum(nil)[:4]

	paddedAddress := common.LeftPadBytes(hexToAddress.Bytes(), 32)

	utils.L.Info(amount)
	// TODO
	amountBig := new(big.Int)
	amountBig.SetString("1000000000000000000000", 10) // 1000 tokens
	paddedAmount := common.LeftPadBytes(amountBig.Bytes(), 32)

	var data []byte
	data = append(data, methodID...)
	data = append(data, paddedAddress...)
	data = append(data, paddedAmount...)

	gasLimit, err := c.EstimateGas(context.Background(), ethereum.CallMsg{
		To:   &hexToAddress,
		Data: data,
	})
	if err != nil {
		return "", err
	}

	chainID, err := c.NetworkID(context.Background())
	if err != nil {
		return "", err
	}

	unloadedAccount := accounts.Account{Address: hexFromAddress}
	err = c.store.TimedUnlock(unloadedAccount, c.config.EthPassword, time.Duration(time.Second*10))
	if err != nil {
		return "", err
	}

	tx := types.NewTransaction(nonce, hexContractAddress, value, gasLimit, gasPrice, data)
	signedTx, err := c.store.SignTx(unloadedAccount, tx, chainID)
	if err != nil {
		return "", err
	}

	err = c.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", err
	}

	utils.L.Infof("ERC20TokenTranser contractAddress: %s, fromAddress: %s, toAddress: %s with amount %f",
		contractAddress, fromAddress, toAddress, amount)
	utils.L.Infof("txid: %s", signedTx.Hash().Hex())

	return signedTx.Hash().Hex(), nil
}