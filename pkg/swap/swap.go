package swap

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/ethersphere/bee/pkg/logging"
	"github.com/ethersphere/bee/pkg/settlement/swap/transaction"
	statestore "github.com/ethersphere/bee/pkg/statestore/mock"
	"github.com/ethersphere/sw3-bindings/v2/simpleswapfactory"
	"github.com/sirupsen/logrus"
)

type Service struct {
	sender               common.Address
	transactionService   transaction.Service
	backend              *ethclient.Client
	tokenAddress         common.Address
	erc20ABI             abi.ABI
	simpleSwapFactoryABI abi.ABI
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

	erc20ABI, err := abi.JSON(strings.NewReader(simpleswapfactory.ERC20ABI))
	if err != nil {
		return nil, err
	}

	simpleSwapFactoryABI, err := abi.JSON(strings.NewReader(simpleswapfactory.SimpleSwapFactoryABI))
	if err != nil {
		return nil, err
	}

	return &Service{
		backend:              client,
		transactionService:   transactionService,
		tokenAddress:         common.HexToAddress(tokenAddress),
		erc20ABI:             erc20ABI,
		simpleSwapFactoryABI: simpleSwapFactoryABI,
		sender:               sender,
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
