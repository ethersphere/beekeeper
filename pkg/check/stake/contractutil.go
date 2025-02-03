package stake

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func newStake(opts Options) (*Stake, *ethclient.Client, error) {
	if opts.GethURL == "" {
		panic(errors.New("geth URL not provided"))
	}

	geth, err := ethclient.Dial(opts.GethURL)
	if err != nil {
		return nil, nil, fmt.Errorf("dial: %w", err)
	}

	addr := common.HexToAddress(opts.ContractAddr)
	contract, err := NewStake(addr, geth)
	if err != nil {
		return nil, nil, fmt.Errorf("new contract instance: %w", err)
	}

	return contract, geth, nil
}

func newSession(contract *Stake, geth *ethclient.Client, opts Options) (*StakeSession, error) {
	gasPrice, err := geth.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get suggested price: %w", err)
	}

	privateKey, err := crypto.HexToECDSA(opts.CallerPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("casti public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := geth.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return nil, fmt.Errorf("get nonce: %w", err)
	}

	chainID, err := geth.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get chain ID: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("new transactor: %w", err)
	}

	session := &StakeSession{
		Contract: contract,
		CallOpts: bind.CallOpts{},
		TransactOpts: bind.TransactOpts{
			Signer:   auth.Signer,
			Nonce:    big.NewInt(int64(nonce)),
			From:     fromAddress,
			GasLimit: uint64(300_000), // in units,
			GasPrice: gasPrice,
		},
	}

	return session, nil
}
