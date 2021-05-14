package swap

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/ethersphere/bee/pkg/logging"
	"github.com/ethersphere/bee/pkg/settlement/swap/transaction"
	statestore "github.com/ethersphere/bee/pkg/statestore/mock"
	"github.com/ethersphere/go-sw3-abi/sw3abi"
	"github.com/sirupsen/logrus"
)

type Service struct {
	backend            *ethclient.Client
	erc20ABI           abi.ABI
	sender             common.Address
	tokenAddress       common.Address
	transactionService transaction.Service
}

func NewService(backend string, privateKeyHex string, tokenAddress string) (s *Service, err error) {
	client, err := ethclient.Dial(backend)
	if err != nil {
		return nil, err
	}

	data, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, err
	}

	privKey, err := crypto.DecodeSecp256k1PrivateKey(data)
	if err != nil {
		return nil, err
	}

	signer := crypto.NewDefaultSigner(privKey)
	store := statestore.NewStateStore()

	sender, err := signer.EthereumAddress()
	if err != nil {
		return nil, err
	}
	logger := logging.New(os.Stdout, logrus.TraceLevel) // TODO: check this
	chainID := big.NewInt(0)                            // TODO: check this
	transactionService, err := transaction.NewService(logger, client, signer, store, chainID)
	if err != nil {
		return nil, err
	}

	erc20ABI := transaction.ParseABIUnchecked(sw3abi.ERC20ABIv0_3_1)

	return &Service{
		backend:            client,
		erc20ABI:           erc20ABI,
		sender:             sender,
		tokenAddress:       common.HexToAddress(tokenAddress),
		transactionService: transactionService,
	}, nil
}

func (s *Service) Fund(ctx context.Context, ethDeposit *big.Int, tokenDeposit *big.Int, address string) (err error) {
	ethAddress := common.HexToAddress(address)

	if ethDeposit.Cmp(big.NewInt(0)) != 0 {
		txHash, err := s.transactionService.Send(ctx, &transaction.TxRequest{
			To:    &ethAddress,
			Value: ethDeposit,
		})
		if err != nil {
			return err
		}
		fmt.Printf("sent ETH deposit %s address %s txHash %s\n", ethDeposit, address, txHash.String())
	}

	if tokenDeposit.Cmp(big.NewInt(0)) != 0 {
		data, err := s.erc20ABI.Pack("transfer", ethAddress, tokenDeposit)
		if err != nil {
			return err
		}

		txHash, err := s.transactionService.Send(ctx, &transaction.TxRequest{
			To:   &s.tokenAddress,
			Data: data,
		})
		if err != nil {
			return err
		}
		fmt.Printf("sent BZZ deposit %s address %s txHash %s\n", tokenDeposit, address, txHash.String())
	}

	return nil
}
