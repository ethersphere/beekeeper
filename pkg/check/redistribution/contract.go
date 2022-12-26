// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package redistribution

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// RedistributionReveal is an auto generated low-level Go binding around an user-defined struct.
type RedistributionReveal struct {
	Owner        common.Address
	Overlay      [32]byte
	Stake        *big.Int
	StakeDensity *big.Int
	Hash         [32]byte
	Depth        uint8
}

// RedistributionMetaData contains all meta data concerning the Redistribution contract.
var RedistributionMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"staking\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"postageContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"oracleContract\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"roundNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"overlay\",\"type\":\"bytes32\"}],\"name\":\"Committed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_count\",\"type\":\"uint256\"}],\"name\":\"CountCommits\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_count\",\"type\":\"uint256\"}],\"name\":\"CountReveals\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"roundNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"overlay\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"stakeDensity\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"reserveCommitment\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"depth\",\"type\":\"uint8\"}],\"name\":\"Revealed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"previousAdminRole\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"newAdminRole\",\"type\":\"bytes32\"}],\"name\":\"RoleAdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleGranted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleRevoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"depth\",\"type\":\"uint8\"}],\"name\":\"TruthSelected\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"overlay\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"stakeDensity\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"depth\",\"type\":\"uint8\"}],\"indexed\":false,\"internalType\":\"structRedistribution.Reveal\",\"name\":\"winner\",\"type\":\"tuple\"}],\"name\":\"WinnerSelected\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DEFAULT_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OracleContract\",\"outputs\":[{\"internalType\":\"contractPriceOracle\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"PAUSER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"PostageContract\",\"outputs\":[{\"internalType\":\"contractPostageStamp\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"Stakes\",\"outputs\":[{\"internalType\":\"contractStakeRegistry\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"claim\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_obfuscatedHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"_overlay\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_roundNumber\",\"type\":\"uint256\"}],\"name\":\"commit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentClaimRound\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentCommitRound\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"currentCommits\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"overlay\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"obfuscatedHash\",\"type\":\"bytes32\"},{\"internalType\":\"bool\",\"name\":\"revealed\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"revealIndex\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentPhaseClaim\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentPhaseCommit\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentPhaseReveal\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentRevealRound\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"currentReveals\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"overlay\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"stakeDensity\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"depth\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentRound\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentRoundAnchor\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"returnVal\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentRoundReveals\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"overlay\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"stakeDensity\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"depth\",\"type\":\"uint8\"}],\"internalType\":\"structRedistribution.Reveal[]\",\"name\":\"\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentSeed\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleAdmin\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"grantRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"hasRole\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"A\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"B\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"minimum\",\"type\":\"uint8\"}],\"name\":\"inProximity\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"overlay\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"depth\",\"type\":\"uint8\"}],\"name\":\"isParticipatingInUpcomingRound\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_overlay\",\"type\":\"bytes32\"}],\"name\":\"isWinner\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minimumStake\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nextSeed\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"penaltyMultiplierDisagreement\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"penaltyMultiplierNonRevealed\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"renounceRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_overlay\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"_depth\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"_hash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"_revealNonce\",\"type\":\"bytes32\"}],\"name\":\"reveal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"revokeRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"roundLength\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unPause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"winner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"overlay\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"stakeDensity\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"depth\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_overlay\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"_depth\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"_hash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"revealNonce\",\"type\":\"bytes32\"}],\"name\":\"wrapCommit\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
}

// RedistributionABI is the input ABI used to generate the binding from.
// Deprecated: Use RedistributionMetaData.ABI instead.
var RedistributionABI = RedistributionMetaData.ABI

// Redistribution is an auto generated Go binding around an Ethereum contract.
type Redistribution struct {
	RedistributionCaller     // Read-only binding to the contract
	RedistributionTransactor // Write-only binding to the contract
	RedistributionFilterer   // Log filterer for contract events
}

// RedistributionCaller is an auto generated read-only Go binding around an Ethereum contract.
type RedistributionCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RedistributionTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RedistributionTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RedistributionFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RedistributionFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RedistributionSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RedistributionSession struct {
	Contract     *Redistribution   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RedistributionCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RedistributionCallerSession struct {
	Contract *RedistributionCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// RedistributionTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RedistributionTransactorSession struct {
	Contract     *RedistributionTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// RedistributionRaw is an auto generated low-level Go binding around an Ethereum contract.
type RedistributionRaw struct {
	Contract *Redistribution // Generic contract binding to access the raw methods on
}

// RedistributionCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RedistributionCallerRaw struct {
	Contract *RedistributionCaller // Generic read-only contract binding to access the raw methods on
}

// RedistributionTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RedistributionTransactorRaw struct {
	Contract *RedistributionTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRedistribution creates a new instance of Redistribution, bound to a specific deployed contract.
func NewRedistribution(address common.Address, backend bind.ContractBackend) (*Redistribution, error) {
	contract, err := bindRedistribution(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Redistribution{RedistributionCaller: RedistributionCaller{contract: contract}, RedistributionTransactor: RedistributionTransactor{contract: contract}, RedistributionFilterer: RedistributionFilterer{contract: contract}}, nil
}

// NewRedistributionCaller creates a new read-only instance of Redistribution, bound to a specific deployed contract.
func NewRedistributionCaller(address common.Address, caller bind.ContractCaller) (*RedistributionCaller, error) {
	contract, err := bindRedistribution(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RedistributionCaller{contract: contract}, nil
}

// NewRedistributionTransactor creates a new write-only instance of Redistribution, bound to a specific deployed contract.
func NewRedistributionTransactor(address common.Address, transactor bind.ContractTransactor) (*RedistributionTransactor, error) {
	contract, err := bindRedistribution(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RedistributionTransactor{contract: contract}, nil
}

// NewRedistributionFilterer creates a new log filterer instance of Redistribution, bound to a specific deployed contract.
func NewRedistributionFilterer(address common.Address, filterer bind.ContractFilterer) (*RedistributionFilterer, error) {
	contract, err := bindRedistribution(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RedistributionFilterer{contract: contract}, nil
}

// bindRedistribution binds a generic wrapper to an already deployed contract.
func bindRedistribution(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(RedistributionABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Redistribution *RedistributionRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Redistribution.Contract.RedistributionCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Redistribution *RedistributionRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Redistribution.Contract.RedistributionTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Redistribution *RedistributionRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Redistribution.Contract.RedistributionTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Redistribution *RedistributionCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Redistribution.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Redistribution *RedistributionTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Redistribution.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Redistribution *RedistributionTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Redistribution.Contract.contract.Transact(opts, method, params...)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Redistribution *RedistributionCaller) DEFAULTADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "DEFAULT_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Redistribution *RedistributionSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _Redistribution.Contract.DEFAULTADMINROLE(&_Redistribution.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Redistribution *RedistributionCallerSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _Redistribution.Contract.DEFAULTADMINROLE(&_Redistribution.CallOpts)
}

// OracleContract is a free data retrieval call binding the contract method 0x69da9114.
//
// Solidity: function OracleContract() view returns(address)
func (_Redistribution *RedistributionCaller) OracleContract(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "OracleContract")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OracleContract is a free data retrieval call binding the contract method 0x69da9114.
//
// Solidity: function OracleContract() view returns(address)
func (_Redistribution *RedistributionSession) OracleContract() (common.Address, error) {
	return _Redistribution.Contract.OracleContract(&_Redistribution.CallOpts)
}

// OracleContract is a free data retrieval call binding the contract method 0x69da9114.
//
// Solidity: function OracleContract() view returns(address)
func (_Redistribution *RedistributionCallerSession) OracleContract() (common.Address, error) {
	return _Redistribution.Contract.OracleContract(&_Redistribution.CallOpts)
}

// PAUSERROLE is a free data retrieval call binding the contract method 0xe63ab1e9.
//
// Solidity: function PAUSER_ROLE() view returns(bytes32)
func (_Redistribution *RedistributionCaller) PAUSERROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "PAUSER_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// PAUSERROLE is a free data retrieval call binding the contract method 0xe63ab1e9.
//
// Solidity: function PAUSER_ROLE() view returns(bytes32)
func (_Redistribution *RedistributionSession) PAUSERROLE() ([32]byte, error) {
	return _Redistribution.Contract.PAUSERROLE(&_Redistribution.CallOpts)
}

// PAUSERROLE is a free data retrieval call binding the contract method 0xe63ab1e9.
//
// Solidity: function PAUSER_ROLE() view returns(bytes32)
func (_Redistribution *RedistributionCallerSession) PAUSERROLE() ([32]byte, error) {
	return _Redistribution.Contract.PAUSERROLE(&_Redistribution.CallOpts)
}

// PostageContract is a free data retrieval call binding the contract method 0x18350096.
//
// Solidity: function PostageContract() view returns(address)
func (_Redistribution *RedistributionCaller) PostageContract(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "PostageContract")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PostageContract is a free data retrieval call binding the contract method 0x18350096.
//
// Solidity: function PostageContract() view returns(address)
func (_Redistribution *RedistributionSession) PostageContract() (common.Address, error) {
	return _Redistribution.Contract.PostageContract(&_Redistribution.CallOpts)
}

// PostageContract is a free data retrieval call binding the contract method 0x18350096.
//
// Solidity: function PostageContract() view returns(address)
func (_Redistribution *RedistributionCallerSession) PostageContract() (common.Address, error) {
	return _Redistribution.Contract.PostageContract(&_Redistribution.CallOpts)
}

// Stakes is a free data retrieval call binding the contract method 0x5d4844ea.
//
// Solidity: function Stakes() view returns(address)
func (_Redistribution *RedistributionCaller) Stakes(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "Stakes")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Stakes is a free data retrieval call binding the contract method 0x5d4844ea.
//
// Solidity: function Stakes() view returns(address)
func (_Redistribution *RedistributionSession) Stakes() (common.Address, error) {
	return _Redistribution.Contract.Stakes(&_Redistribution.CallOpts)
}

// Stakes is a free data retrieval call binding the contract method 0x5d4844ea.
//
// Solidity: function Stakes() view returns(address)
func (_Redistribution *RedistributionCallerSession) Stakes() (common.Address, error) {
	return _Redistribution.Contract.Stakes(&_Redistribution.CallOpts)
}

// CurrentClaimRound is a free data retrieval call binding the contract method 0x6f94aaf2.
//
// Solidity: function currentClaimRound() view returns(uint256)
func (_Redistribution *RedistributionCaller) CurrentClaimRound(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "currentClaimRound")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CurrentClaimRound is a free data retrieval call binding the contract method 0x6f94aaf2.
//
// Solidity: function currentClaimRound() view returns(uint256)
func (_Redistribution *RedistributionSession) CurrentClaimRound() (*big.Int, error) {
	return _Redistribution.Contract.CurrentClaimRound(&_Redistribution.CallOpts)
}

// CurrentClaimRound is a free data retrieval call binding the contract method 0x6f94aaf2.
//
// Solidity: function currentClaimRound() view returns(uint256)
func (_Redistribution *RedistributionCallerSession) CurrentClaimRound() (*big.Int, error) {
	return _Redistribution.Contract.CurrentClaimRound(&_Redistribution.CallOpts)
}

// CurrentCommitRound is a free data retrieval call binding the contract method 0x69bfac01.
//
// Solidity: function currentCommitRound() view returns(uint256)
func (_Redistribution *RedistributionCaller) CurrentCommitRound(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "currentCommitRound")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CurrentCommitRound is a free data retrieval call binding the contract method 0x69bfac01.
//
// Solidity: function currentCommitRound() view returns(uint256)
func (_Redistribution *RedistributionSession) CurrentCommitRound() (*big.Int, error) {
	return _Redistribution.Contract.CurrentCommitRound(&_Redistribution.CallOpts)
}

// CurrentCommitRound is a free data retrieval call binding the contract method 0x69bfac01.
//
// Solidity: function currentCommitRound() view returns(uint256)
func (_Redistribution *RedistributionCallerSession) CurrentCommitRound() (*big.Int, error) {
	return _Redistribution.Contract.CurrentCommitRound(&_Redistribution.CallOpts)
}

// CurrentCommits is a free data retrieval call binding the contract method 0x72286cba.
//
// Solidity: function currentCommits(uint256 ) view returns(bytes32 overlay, address owner, uint256 stake, bytes32 obfuscatedHash, bool revealed, uint256 revealIndex)
func (_Redistribution *RedistributionCaller) CurrentCommits(opts *bind.CallOpts, arg0 *big.Int) (struct {
	Overlay        [32]byte
	Owner          common.Address
	Stake          *big.Int
	ObfuscatedHash [32]byte
	Revealed       bool
	RevealIndex    *big.Int
}, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "currentCommits", arg0)

	outstruct := new(struct {
		Overlay        [32]byte
		Owner          common.Address
		Stake          *big.Int
		ObfuscatedHash [32]byte
		Revealed       bool
		RevealIndex    *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Overlay = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.Owner = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.Stake = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.ObfuscatedHash = *abi.ConvertType(out[3], new([32]byte)).(*[32]byte)
	outstruct.Revealed = *abi.ConvertType(out[4], new(bool)).(*bool)
	outstruct.RevealIndex = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// CurrentCommits is a free data retrieval call binding the contract method 0x72286cba.
//
// Solidity: function currentCommits(uint256 ) view returns(bytes32 overlay, address owner, uint256 stake, bytes32 obfuscatedHash, bool revealed, uint256 revealIndex)
func (_Redistribution *RedistributionSession) CurrentCommits(arg0 *big.Int) (struct {
	Overlay        [32]byte
	Owner          common.Address
	Stake          *big.Int
	ObfuscatedHash [32]byte
	Revealed       bool
	RevealIndex    *big.Int
}, error) {
	return _Redistribution.Contract.CurrentCommits(&_Redistribution.CallOpts, arg0)
}

// CurrentCommits is a free data retrieval call binding the contract method 0x72286cba.
//
// Solidity: function currentCommits(uint256 ) view returns(bytes32 overlay, address owner, uint256 stake, bytes32 obfuscatedHash, bool revealed, uint256 revealIndex)
func (_Redistribution *RedistributionCallerSession) CurrentCommits(arg0 *big.Int) (struct {
	Overlay        [32]byte
	Owner          common.Address
	Stake          *big.Int
	ObfuscatedHash [32]byte
	Revealed       bool
	RevealIndex    *big.Int
}, error) {
	return _Redistribution.Contract.CurrentCommits(&_Redistribution.CallOpts, arg0)
}

// CurrentPhaseClaim is a free data retrieval call binding the contract method 0x8d8b6428.
//
// Solidity: function currentPhaseClaim() view returns(bool)
func (_Redistribution *RedistributionCaller) CurrentPhaseClaim(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "currentPhaseClaim")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// CurrentPhaseClaim is a free data retrieval call binding the contract method 0x8d8b6428.
//
// Solidity: function currentPhaseClaim() view returns(bool)
func (_Redistribution *RedistributionSession) CurrentPhaseClaim() (bool, error) {
	return _Redistribution.Contract.CurrentPhaseClaim(&_Redistribution.CallOpts)
}

// CurrentPhaseClaim is a free data retrieval call binding the contract method 0x8d8b6428.
//
// Solidity: function currentPhaseClaim() view returns(bool)
func (_Redistribution *RedistributionCallerSession) CurrentPhaseClaim() (bool, error) {
	return _Redistribution.Contract.CurrentPhaseClaim(&_Redistribution.CallOpts)
}

// CurrentPhaseCommit is a free data retrieval call binding the contract method 0xd1e8b63d.
//
// Solidity: function currentPhaseCommit() view returns(bool)
func (_Redistribution *RedistributionCaller) CurrentPhaseCommit(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "currentPhaseCommit")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// CurrentPhaseCommit is a free data retrieval call binding the contract method 0xd1e8b63d.
//
// Solidity: function currentPhaseCommit() view returns(bool)
func (_Redistribution *RedistributionSession) CurrentPhaseCommit() (bool, error) {
	return _Redistribution.Contract.CurrentPhaseCommit(&_Redistribution.CallOpts)
}

// CurrentPhaseCommit is a free data retrieval call binding the contract method 0xd1e8b63d.
//
// Solidity: function currentPhaseCommit() view returns(bool)
func (_Redistribution *RedistributionCallerSession) CurrentPhaseCommit() (bool, error) {
	return _Redistribution.Contract.CurrentPhaseCommit(&_Redistribution.CallOpts)
}

// CurrentPhaseReveal is a free data retrieval call binding the contract method 0x2f3906da.
//
// Solidity: function currentPhaseReveal() view returns(bool)
func (_Redistribution *RedistributionCaller) CurrentPhaseReveal(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "currentPhaseReveal")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// CurrentPhaseReveal is a free data retrieval call binding the contract method 0x2f3906da.
//
// Solidity: function currentPhaseReveal() view returns(bool)
func (_Redistribution *RedistributionSession) CurrentPhaseReveal() (bool, error) {
	return _Redistribution.Contract.CurrentPhaseReveal(&_Redistribution.CallOpts)
}

// CurrentPhaseReveal is a free data retrieval call binding the contract method 0x2f3906da.
//
// Solidity: function currentPhaseReveal() view returns(bool)
func (_Redistribution *RedistributionCallerSession) CurrentPhaseReveal() (bool, error) {
	return _Redistribution.Contract.CurrentPhaseReveal(&_Redistribution.CallOpts)
}

// CurrentRevealRound is a free data retrieval call binding the contract method 0x7fe019c6.
//
// Solidity: function currentRevealRound() view returns(uint256)
func (_Redistribution *RedistributionCaller) CurrentRevealRound(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "currentRevealRound")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CurrentRevealRound is a free data retrieval call binding the contract method 0x7fe019c6.
//
// Solidity: function currentRevealRound() view returns(uint256)
func (_Redistribution *RedistributionSession) CurrentRevealRound() (*big.Int, error) {
	return _Redistribution.Contract.CurrentRevealRound(&_Redistribution.CallOpts)
}

// CurrentRevealRound is a free data retrieval call binding the contract method 0x7fe019c6.
//
// Solidity: function currentRevealRound() view returns(uint256)
func (_Redistribution *RedistributionCallerSession) CurrentRevealRound() (*big.Int, error) {
	return _Redistribution.Contract.CurrentRevealRound(&_Redistribution.CallOpts)
}

// CurrentReveals is a free data retrieval call binding the contract method 0x82b39b1b.
//
// Solidity: function currentReveals(uint256 ) view returns(address owner, bytes32 overlay, uint256 stake, uint256 stakeDensity, bytes32 hash, uint8 depth)
func (_Redistribution *RedistributionCaller) CurrentReveals(opts *bind.CallOpts, arg0 *big.Int) (struct {
	Owner        common.Address
	Overlay      [32]byte
	Stake        *big.Int
	StakeDensity *big.Int
	Hash         [32]byte
	Depth        uint8
}, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "currentReveals", arg0)

	outstruct := new(struct {
		Owner        common.Address
		Overlay      [32]byte
		Stake        *big.Int
		StakeDensity *big.Int
		Hash         [32]byte
		Depth        uint8
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Owner = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Overlay = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)
	outstruct.Stake = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.StakeDensity = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.Hash = *abi.ConvertType(out[4], new([32]byte)).(*[32]byte)
	outstruct.Depth = *abi.ConvertType(out[5], new(uint8)).(*uint8)

	return *outstruct, err

}

// CurrentReveals is a free data retrieval call binding the contract method 0x82b39b1b.
//
// Solidity: function currentReveals(uint256 ) view returns(address owner, bytes32 overlay, uint256 stake, uint256 stakeDensity, bytes32 hash, uint8 depth)
func (_Redistribution *RedistributionSession) CurrentReveals(arg0 *big.Int) (struct {
	Owner        common.Address
	Overlay      [32]byte
	Stake        *big.Int
	StakeDensity *big.Int
	Hash         [32]byte
	Depth        uint8
}, error) {
	return _Redistribution.Contract.CurrentReveals(&_Redistribution.CallOpts, arg0)
}

// CurrentReveals is a free data retrieval call binding the contract method 0x82b39b1b.
//
// Solidity: function currentReveals(uint256 ) view returns(address owner, bytes32 overlay, uint256 stake, uint256 stakeDensity, bytes32 hash, uint8 depth)
func (_Redistribution *RedistributionCallerSession) CurrentReveals(arg0 *big.Int) (struct {
	Owner        common.Address
	Overlay      [32]byte
	Stake        *big.Int
	StakeDensity *big.Int
	Hash         [32]byte
	Depth        uint8
}, error) {
	return _Redistribution.Contract.CurrentReveals(&_Redistribution.CallOpts, arg0)
}

// CurrentRound is a free data retrieval call binding the contract method 0x8a19c8bc.
//
// Solidity: function currentRound() view returns(uint256)
func (_Redistribution *RedistributionCaller) CurrentRound(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "currentRound")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CurrentRound is a free data retrieval call binding the contract method 0x8a19c8bc.
//
// Solidity: function currentRound() view returns(uint256)
func (_Redistribution *RedistributionSession) CurrentRound() (*big.Int, error) {
	return _Redistribution.Contract.CurrentRound(&_Redistribution.CallOpts)
}

// CurrentRound is a free data retrieval call binding the contract method 0x8a19c8bc.
//
// Solidity: function currentRound() view returns(uint256)
func (_Redistribution *RedistributionCallerSession) CurrentRound() (*big.Int, error) {
	return _Redistribution.Contract.CurrentRound(&_Redistribution.CallOpts)
}

// CurrentRoundAnchor is a free data retrieval call binding the contract method 0x64c34a85.
//
// Solidity: function currentRoundAnchor() view returns(bytes32 returnVal)
func (_Redistribution *RedistributionCaller) CurrentRoundAnchor(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "currentRoundAnchor")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CurrentRoundAnchor is a free data retrieval call binding the contract method 0x64c34a85.
//
// Solidity: function currentRoundAnchor() view returns(bytes32 returnVal)
func (_Redistribution *RedistributionSession) CurrentRoundAnchor() ([32]byte, error) {
	return _Redistribution.Contract.CurrentRoundAnchor(&_Redistribution.CallOpts)
}

// CurrentRoundAnchor is a free data retrieval call binding the contract method 0x64c34a85.
//
// Solidity: function currentRoundAnchor() view returns(bytes32 returnVal)
func (_Redistribution *RedistributionCallerSession) CurrentRoundAnchor() ([32]byte, error) {
	return _Redistribution.Contract.CurrentRoundAnchor(&_Redistribution.CallOpts)
}

// CurrentRoundReveals is a free data retrieval call binding the contract method 0x2a4e6249.
//
// Solidity: function currentRoundReveals() view returns((address,bytes32,uint256,uint256,bytes32,uint8)[])
func (_Redistribution *RedistributionCaller) CurrentRoundReveals(opts *bind.CallOpts) ([]RedistributionReveal, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "currentRoundReveals")

	if err != nil {
		return *new([]RedistributionReveal), err
	}

	out0 := *abi.ConvertType(out[0], new([]RedistributionReveal)).(*[]RedistributionReveal)

	return out0, err

}

// CurrentRoundReveals is a free data retrieval call binding the contract method 0x2a4e6249.
//
// Solidity: function currentRoundReveals() view returns((address,bytes32,uint256,uint256,bytes32,uint8)[])
func (_Redistribution *RedistributionSession) CurrentRoundReveals() ([]RedistributionReveal, error) {
	return _Redistribution.Contract.CurrentRoundReveals(&_Redistribution.CallOpts)
}

// CurrentRoundReveals is a free data retrieval call binding the contract method 0x2a4e6249.
//
// Solidity: function currentRoundReveals() view returns((address,bytes32,uint256,uint256,bytes32,uint8)[])
func (_Redistribution *RedistributionCallerSession) CurrentRoundReveals() ([]RedistributionReveal, error) {
	return _Redistribution.Contract.CurrentRoundReveals(&_Redistribution.CallOpts)
}

// CurrentSeed is a free data retrieval call binding the contract method 0x83220626.
//
// Solidity: function currentSeed() view returns(bytes32)
func (_Redistribution *RedistributionCaller) CurrentSeed(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "currentSeed")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CurrentSeed is a free data retrieval call binding the contract method 0x83220626.
//
// Solidity: function currentSeed() view returns(bytes32)
func (_Redistribution *RedistributionSession) CurrentSeed() ([32]byte, error) {
	return _Redistribution.Contract.CurrentSeed(&_Redistribution.CallOpts)
}

// CurrentSeed is a free data retrieval call binding the contract method 0x83220626.
//
// Solidity: function currentSeed() view returns(bytes32)
func (_Redistribution *RedistributionCallerSession) CurrentSeed() ([32]byte, error) {
	return _Redistribution.Contract.CurrentSeed(&_Redistribution.CallOpts)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Redistribution *RedistributionCaller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "getRoleAdmin", role)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Redistribution *RedistributionSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _Redistribution.Contract.GetRoleAdmin(&_Redistribution.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Redistribution *RedistributionCallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _Redistribution.Contract.GetRoleAdmin(&_Redistribution.CallOpts, role)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Redistribution *RedistributionCaller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "hasRole", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Redistribution *RedistributionSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _Redistribution.Contract.HasRole(&_Redistribution.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Redistribution *RedistributionCallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _Redistribution.Contract.HasRole(&_Redistribution.CallOpts, role, account)
}

// InProximity is a free data retrieval call binding the contract method 0xfb00f2f3.
//
// Solidity: function inProximity(bytes32 A, bytes32 B, uint8 minimum) pure returns(bool)
func (_Redistribution *RedistributionCaller) InProximity(opts *bind.CallOpts, A [32]byte, B [32]byte, minimum uint8) (bool, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "inProximity", A, B, minimum)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// InProximity is a free data retrieval call binding the contract method 0xfb00f2f3.
//
// Solidity: function inProximity(bytes32 A, bytes32 B, uint8 minimum) pure returns(bool)
func (_Redistribution *RedistributionSession) InProximity(A [32]byte, B [32]byte, minimum uint8) (bool, error) {
	return _Redistribution.Contract.InProximity(&_Redistribution.CallOpts, A, B, minimum)
}

// InProximity is a free data retrieval call binding the contract method 0xfb00f2f3.
//
// Solidity: function inProximity(bytes32 A, bytes32 B, uint8 minimum) pure returns(bool)
func (_Redistribution *RedistributionCallerSession) InProximity(A [32]byte, B [32]byte, minimum uint8) (bool, error) {
	return _Redistribution.Contract.InProximity(&_Redistribution.CallOpts, A, B, minimum)
}

// IsParticipatingInUpcomingRound is a free data retrieval call binding the contract method 0xb78a52a7.
//
// Solidity: function isParticipatingInUpcomingRound(bytes32 overlay, uint8 depth) view returns(bool)
func (_Redistribution *RedistributionCaller) IsParticipatingInUpcomingRound(opts *bind.CallOpts, overlay [32]byte, depth uint8) (bool, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "isParticipatingInUpcomingRound", overlay, depth)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsParticipatingInUpcomingRound is a free data retrieval call binding the contract method 0xb78a52a7.
//
// Solidity: function isParticipatingInUpcomingRound(bytes32 overlay, uint8 depth) view returns(bool)
func (_Redistribution *RedistributionSession) IsParticipatingInUpcomingRound(overlay [32]byte, depth uint8) (bool, error) {
	return _Redistribution.Contract.IsParticipatingInUpcomingRound(&_Redistribution.CallOpts, overlay, depth)
}

// IsParticipatingInUpcomingRound is a free data retrieval call binding the contract method 0xb78a52a7.
//
// Solidity: function isParticipatingInUpcomingRound(bytes32 overlay, uint8 depth) view returns(bool)
func (_Redistribution *RedistributionCallerSession) IsParticipatingInUpcomingRound(overlay [32]byte, depth uint8) (bool, error) {
	return _Redistribution.Contract.IsParticipatingInUpcomingRound(&_Redistribution.CallOpts, overlay, depth)
}

// IsWinner is a free data retrieval call binding the contract method 0x77c75d10.
//
// Solidity: function isWinner(bytes32 _overlay) view returns(bool)
func (_Redistribution *RedistributionCaller) IsWinner(opts *bind.CallOpts, _overlay [32]byte) (bool, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "isWinner", _overlay)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsWinner is a free data retrieval call binding the contract method 0x77c75d10.
//
// Solidity: function isWinner(bytes32 _overlay) view returns(bool)
func (_Redistribution *RedistributionSession) IsWinner(_overlay [32]byte) (bool, error) {
	return _Redistribution.Contract.IsWinner(&_Redistribution.CallOpts, _overlay)
}

// IsWinner is a free data retrieval call binding the contract method 0x77c75d10.
//
// Solidity: function isWinner(bytes32 _overlay) view returns(bool)
func (_Redistribution *RedistributionCallerSession) IsWinner(_overlay [32]byte) (bool, error) {
	return _Redistribution.Contract.IsWinner(&_Redistribution.CallOpts, _overlay)
}

// MinimumStake is a free data retrieval call binding the contract method 0xec5ffac2.
//
// Solidity: function minimumStake() view returns(uint256)
func (_Redistribution *RedistributionCaller) MinimumStake(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "minimumStake")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinimumStake is a free data retrieval call binding the contract method 0xec5ffac2.
//
// Solidity: function minimumStake() view returns(uint256)
func (_Redistribution *RedistributionSession) MinimumStake() (*big.Int, error) {
	return _Redistribution.Contract.MinimumStake(&_Redistribution.CallOpts)
}

// MinimumStake is a free data retrieval call binding the contract method 0xec5ffac2.
//
// Solidity: function minimumStake() view returns(uint256)
func (_Redistribution *RedistributionCallerSession) MinimumStake() (*big.Int, error) {
	return _Redistribution.Contract.MinimumStake(&_Redistribution.CallOpts)
}

// NextSeed is a free data retrieval call binding the contract method 0x62fd29ae.
//
// Solidity: function nextSeed() view returns(bytes32)
func (_Redistribution *RedistributionCaller) NextSeed(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "nextSeed")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// NextSeed is a free data retrieval call binding the contract method 0x62fd29ae.
//
// Solidity: function nextSeed() view returns(bytes32)
func (_Redistribution *RedistributionSession) NextSeed() ([32]byte, error) {
	return _Redistribution.Contract.NextSeed(&_Redistribution.CallOpts)
}

// NextSeed is a free data retrieval call binding the contract method 0x62fd29ae.
//
// Solidity: function nextSeed() view returns(bytes32)
func (_Redistribution *RedistributionCallerSession) NextSeed() ([32]byte, error) {
	return _Redistribution.Contract.NextSeed(&_Redistribution.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_Redistribution *RedistributionCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_Redistribution *RedistributionSession) Paused() (bool, error) {
	return _Redistribution.Contract.Paused(&_Redistribution.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_Redistribution *RedistributionCallerSession) Paused() (bool, error) {
	return _Redistribution.Contract.Paused(&_Redistribution.CallOpts)
}

// PenaltyMultiplierDisagreement is a free data retrieval call binding the contract method 0x4e3727d2.
//
// Solidity: function penaltyMultiplierDisagreement() view returns(uint256)
func (_Redistribution *RedistributionCaller) PenaltyMultiplierDisagreement(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "penaltyMultiplierDisagreement")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PenaltyMultiplierDisagreement is a free data retrieval call binding the contract method 0x4e3727d2.
//
// Solidity: function penaltyMultiplierDisagreement() view returns(uint256)
func (_Redistribution *RedistributionSession) PenaltyMultiplierDisagreement() (*big.Int, error) {
	return _Redistribution.Contract.PenaltyMultiplierDisagreement(&_Redistribution.CallOpts)
}

// PenaltyMultiplierDisagreement is a free data retrieval call binding the contract method 0x4e3727d2.
//
// Solidity: function penaltyMultiplierDisagreement() view returns(uint256)
func (_Redistribution *RedistributionCallerSession) PenaltyMultiplierDisagreement() (*big.Int, error) {
	return _Redistribution.Contract.PenaltyMultiplierDisagreement(&_Redistribution.CallOpts)
}

// PenaltyMultiplierNonRevealed is a free data retrieval call binding the contract method 0xc203ce52.
//
// Solidity: function penaltyMultiplierNonRevealed() view returns(uint256)
func (_Redistribution *RedistributionCaller) PenaltyMultiplierNonRevealed(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "penaltyMultiplierNonRevealed")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PenaltyMultiplierNonRevealed is a free data retrieval call binding the contract method 0xc203ce52.
//
// Solidity: function penaltyMultiplierNonRevealed() view returns(uint256)
func (_Redistribution *RedistributionSession) PenaltyMultiplierNonRevealed() (*big.Int, error) {
	return _Redistribution.Contract.PenaltyMultiplierNonRevealed(&_Redistribution.CallOpts)
}

// PenaltyMultiplierNonRevealed is a free data retrieval call binding the contract method 0xc203ce52.
//
// Solidity: function penaltyMultiplierNonRevealed() view returns(uint256)
func (_Redistribution *RedistributionCallerSession) PenaltyMultiplierNonRevealed() (*big.Int, error) {
	return _Redistribution.Contract.PenaltyMultiplierNonRevealed(&_Redistribution.CallOpts)
}

// RoundLength is a free data retrieval call binding the contract method 0x8b649b94.
//
// Solidity: function roundLength() view returns(uint256)
func (_Redistribution *RedistributionCaller) RoundLength(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "roundLength")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// RoundLength is a free data retrieval call binding the contract method 0x8b649b94.
//
// Solidity: function roundLength() view returns(uint256)
func (_Redistribution *RedistributionSession) RoundLength() (*big.Int, error) {
	return _Redistribution.Contract.RoundLength(&_Redistribution.CallOpts)
}

// RoundLength is a free data retrieval call binding the contract method 0x8b649b94.
//
// Solidity: function roundLength() view returns(uint256)
func (_Redistribution *RedistributionCallerSession) RoundLength() (*big.Int, error) {
	return _Redistribution.Contract.RoundLength(&_Redistribution.CallOpts)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Redistribution *RedistributionCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Redistribution *RedistributionSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Redistribution.Contract.SupportsInterface(&_Redistribution.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Redistribution *RedistributionCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Redistribution.Contract.SupportsInterface(&_Redistribution.CallOpts, interfaceId)
}

// Winner is a free data retrieval call binding the contract method 0xdfbf53ae.
//
// Solidity: function winner() view returns(address owner, bytes32 overlay, uint256 stake, uint256 stakeDensity, bytes32 hash, uint8 depth)
func (_Redistribution *RedistributionCaller) Winner(opts *bind.CallOpts) (struct {
	Owner        common.Address
	Overlay      [32]byte
	Stake        *big.Int
	StakeDensity *big.Int
	Hash         [32]byte
	Depth        uint8
}, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "winner")

	outstruct := new(struct {
		Owner        common.Address
		Overlay      [32]byte
		Stake        *big.Int
		StakeDensity *big.Int
		Hash         [32]byte
		Depth        uint8
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Owner = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Overlay = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)
	outstruct.Stake = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.StakeDensity = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.Hash = *abi.ConvertType(out[4], new([32]byte)).(*[32]byte)
	outstruct.Depth = *abi.ConvertType(out[5], new(uint8)).(*uint8)

	return *outstruct, err

}

// Winner is a free data retrieval call binding the contract method 0xdfbf53ae.
//
// Solidity: function winner() view returns(address owner, bytes32 overlay, uint256 stake, uint256 stakeDensity, bytes32 hash, uint8 depth)
func (_Redistribution *RedistributionSession) Winner() (struct {
	Owner        common.Address
	Overlay      [32]byte
	Stake        *big.Int
	StakeDensity *big.Int
	Hash         [32]byte
	Depth        uint8
}, error) {
	return _Redistribution.Contract.Winner(&_Redistribution.CallOpts)
}

// Winner is a free data retrieval call binding the contract method 0xdfbf53ae.
//
// Solidity: function winner() view returns(address owner, bytes32 overlay, uint256 stake, uint256 stakeDensity, bytes32 hash, uint8 depth)
func (_Redistribution *RedistributionCallerSession) Winner() (struct {
	Owner        common.Address
	Overlay      [32]byte
	Stake        *big.Int
	StakeDensity *big.Int
	Hash         [32]byte
	Depth        uint8
}, error) {
	return _Redistribution.Contract.Winner(&_Redistribution.CallOpts)
}

// WrapCommit is a free data retrieval call binding the contract method 0xce987745.
//
// Solidity: function wrapCommit(bytes32 _overlay, uint8 _depth, bytes32 _hash, bytes32 revealNonce) pure returns(bytes32)
func (_Redistribution *RedistributionCaller) WrapCommit(opts *bind.CallOpts, _overlay [32]byte, _depth uint8, _hash [32]byte, revealNonce [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Redistribution.contract.Call(opts, &out, "wrapCommit", _overlay, _depth, _hash, revealNonce)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// WrapCommit is a free data retrieval call binding the contract method 0xce987745.
//
// Solidity: function wrapCommit(bytes32 _overlay, uint8 _depth, bytes32 _hash, bytes32 revealNonce) pure returns(bytes32)
func (_Redistribution *RedistributionSession) WrapCommit(_overlay [32]byte, _depth uint8, _hash [32]byte, revealNonce [32]byte) ([32]byte, error) {
	return _Redistribution.Contract.WrapCommit(&_Redistribution.CallOpts, _overlay, _depth, _hash, revealNonce)
}

// WrapCommit is a free data retrieval call binding the contract method 0xce987745.
//
// Solidity: function wrapCommit(bytes32 _overlay, uint8 _depth, bytes32 _hash, bytes32 revealNonce) pure returns(bytes32)
func (_Redistribution *RedistributionCallerSession) WrapCommit(_overlay [32]byte, _depth uint8, _hash [32]byte, revealNonce [32]byte) ([32]byte, error) {
	return _Redistribution.Contract.WrapCommit(&_Redistribution.CallOpts, _overlay, _depth, _hash, revealNonce)
}

// Claim is a paid mutator transaction binding the contract method 0x4e71d92d.
//
// Solidity: function claim() returns()
func (_Redistribution *RedistributionTransactor) Claim(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Redistribution.contract.Transact(opts, "claim")
}

// Claim is a paid mutator transaction binding the contract method 0x4e71d92d.
//
// Solidity: function claim() returns()
func (_Redistribution *RedistributionSession) Claim() (*types.Transaction, error) {
	return _Redistribution.Contract.Claim(&_Redistribution.TransactOpts)
}

// Claim is a paid mutator transaction binding the contract method 0x4e71d92d.
//
// Solidity: function claim() returns()
func (_Redistribution *RedistributionTransactorSession) Claim() (*types.Transaction, error) {
	return _Redistribution.Contract.Claim(&_Redistribution.TransactOpts)
}

// Commit is a paid mutator transaction binding the contract method 0x4a2e7598.
//
// Solidity: function commit(bytes32 _obfuscatedHash, bytes32 _overlay, uint256 _roundNumber) returns()
func (_Redistribution *RedistributionTransactor) Commit(opts *bind.TransactOpts, _obfuscatedHash [32]byte, _overlay [32]byte, _roundNumber *big.Int) (*types.Transaction, error) {
	return _Redistribution.contract.Transact(opts, "commit", _obfuscatedHash, _overlay, _roundNumber)
}

// Commit is a paid mutator transaction binding the contract method 0x4a2e7598.
//
// Solidity: function commit(bytes32 _obfuscatedHash, bytes32 _overlay, uint256 _roundNumber) returns()
func (_Redistribution *RedistributionSession) Commit(_obfuscatedHash [32]byte, _overlay [32]byte, _roundNumber *big.Int) (*types.Transaction, error) {
	return _Redistribution.Contract.Commit(&_Redistribution.TransactOpts, _obfuscatedHash, _overlay, _roundNumber)
}

// Commit is a paid mutator transaction binding the contract method 0x4a2e7598.
//
// Solidity: function commit(bytes32 _obfuscatedHash, bytes32 _overlay, uint256 _roundNumber) returns()
func (_Redistribution *RedistributionTransactorSession) Commit(_obfuscatedHash [32]byte, _overlay [32]byte, _roundNumber *big.Int) (*types.Transaction, error) {
	return _Redistribution.Contract.Commit(&_Redistribution.TransactOpts, _obfuscatedHash, _overlay, _roundNumber)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Redistribution *RedistributionTransactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Redistribution.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Redistribution *RedistributionSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Redistribution.Contract.GrantRole(&_Redistribution.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Redistribution *RedistributionTransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Redistribution.Contract.GrantRole(&_Redistribution.TransactOpts, role, account)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_Redistribution *RedistributionTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Redistribution.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_Redistribution *RedistributionSession) Pause() (*types.Transaction, error) {
	return _Redistribution.Contract.Pause(&_Redistribution.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_Redistribution *RedistributionTransactorSession) Pause() (*types.Transaction, error) {
	return _Redistribution.Contract.Pause(&_Redistribution.TransactOpts)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Redistribution *RedistributionTransactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Redistribution.contract.Transact(opts, "renounceRole", role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Redistribution *RedistributionSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Redistribution.Contract.RenounceRole(&_Redistribution.TransactOpts, role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Redistribution *RedistributionTransactorSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Redistribution.Contract.RenounceRole(&_Redistribution.TransactOpts, role, account)
}

// Reveal is a paid mutator transaction binding the contract method 0xc1d810d5.
//
// Solidity: function reveal(bytes32 _overlay, uint8 _depth, bytes32 _hash, bytes32 _revealNonce) returns()
func (_Redistribution *RedistributionTransactor) Reveal(opts *bind.TransactOpts, _overlay [32]byte, _depth uint8, _hash [32]byte, _revealNonce [32]byte) (*types.Transaction, error) {
	return _Redistribution.contract.Transact(opts, "reveal", _overlay, _depth, _hash, _revealNonce)
}

// Reveal is a paid mutator transaction binding the contract method 0xc1d810d5.
//
// Solidity: function reveal(bytes32 _overlay, uint8 _depth, bytes32 _hash, bytes32 _revealNonce) returns()
func (_Redistribution *RedistributionSession) Reveal(_overlay [32]byte, _depth uint8, _hash [32]byte, _revealNonce [32]byte) (*types.Transaction, error) {
	return _Redistribution.Contract.Reveal(&_Redistribution.TransactOpts, _overlay, _depth, _hash, _revealNonce)
}

// Reveal is a paid mutator transaction binding the contract method 0xc1d810d5.
//
// Solidity: function reveal(bytes32 _overlay, uint8 _depth, bytes32 _hash, bytes32 _revealNonce) returns()
func (_Redistribution *RedistributionTransactorSession) Reveal(_overlay [32]byte, _depth uint8, _hash [32]byte, _revealNonce [32]byte) (*types.Transaction, error) {
	return _Redistribution.Contract.Reveal(&_Redistribution.TransactOpts, _overlay, _depth, _hash, _revealNonce)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Redistribution *RedistributionTransactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Redistribution.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Redistribution *RedistributionSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Redistribution.Contract.RevokeRole(&_Redistribution.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Redistribution *RedistributionTransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Redistribution.Contract.RevokeRole(&_Redistribution.TransactOpts, role, account)
}

// UnPause is a paid mutator transaction binding the contract method 0xf7b188a5.
//
// Solidity: function unPause() returns()
func (_Redistribution *RedistributionTransactor) UnPause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Redistribution.contract.Transact(opts, "unPause")
}

// UnPause is a paid mutator transaction binding the contract method 0xf7b188a5.
//
// Solidity: function unPause() returns()
func (_Redistribution *RedistributionSession) UnPause() (*types.Transaction, error) {
	return _Redistribution.Contract.UnPause(&_Redistribution.TransactOpts)
}

// UnPause is a paid mutator transaction binding the contract method 0xf7b188a5.
//
// Solidity: function unPause() returns()
func (_Redistribution *RedistributionTransactorSession) UnPause() (*types.Transaction, error) {
	return _Redistribution.Contract.UnPause(&_Redistribution.TransactOpts)
}

// RedistributionCommittedIterator is returned from FilterCommitted and is used to iterate over the raw logs and unpacked data for Committed events raised by the Redistribution contract.
type RedistributionCommittedIterator struct {
	Event *RedistributionCommitted // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RedistributionCommittedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RedistributionCommitted)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RedistributionCommitted)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RedistributionCommittedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RedistributionCommittedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RedistributionCommitted represents a Committed event raised by the Redistribution contract.
type RedistributionCommitted struct {
	RoundNumber *big.Int
	Overlay     [32]byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterCommitted is a free log retrieval operation binding the contract event 0x68e0867601a98978930107aee7f425665e61edd70ca594c68ca5da9e81f84c29.
//
// Solidity: event Committed(uint256 roundNumber, bytes32 overlay)
func (_Redistribution *RedistributionFilterer) FilterCommitted(opts *bind.FilterOpts) (*RedistributionCommittedIterator, error) {

	logs, sub, err := _Redistribution.contract.FilterLogs(opts, "Committed")
	if err != nil {
		return nil, err
	}
	return &RedistributionCommittedIterator{contract: _Redistribution.contract, event: "Committed", logs: logs, sub: sub}, nil
}

// WatchCommitted is a free log subscription operation binding the contract event 0x68e0867601a98978930107aee7f425665e61edd70ca594c68ca5da9e81f84c29.
//
// Solidity: event Committed(uint256 roundNumber, bytes32 overlay)
func (_Redistribution *RedistributionFilterer) WatchCommitted(opts *bind.WatchOpts, sink chan<- *RedistributionCommitted) (event.Subscription, error) {

	logs, sub, err := _Redistribution.contract.WatchLogs(opts, "Committed")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RedistributionCommitted)
				if err := _Redistribution.contract.UnpackLog(event, "Committed", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseCommitted is a log parse operation binding the contract event 0x68e0867601a98978930107aee7f425665e61edd70ca594c68ca5da9e81f84c29.
//
// Solidity: event Committed(uint256 roundNumber, bytes32 overlay)
func (_Redistribution *RedistributionFilterer) ParseCommitted(log types.Log) (*RedistributionCommitted, error) {
	event := new(RedistributionCommitted)
	if err := _Redistribution.contract.UnpackLog(event, "Committed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RedistributionCountCommitsIterator is returned from FilterCountCommits and is used to iterate over the raw logs and unpacked data for CountCommits events raised by the Redistribution contract.
type RedistributionCountCommitsIterator struct {
	Event *RedistributionCountCommits // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RedistributionCountCommitsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RedistributionCountCommits)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RedistributionCountCommits)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RedistributionCountCommitsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RedistributionCountCommitsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RedistributionCountCommits represents a CountCommits event raised by the Redistribution contract.
type RedistributionCountCommits struct {
	Count *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterCountCommits is a free log retrieval operation binding the contract event 0x6752c5e71c95fb93bc7137adeb115a33fa4e54e2683e33d3f90c2bb1c4b6c2a5.
//
// Solidity: event CountCommits(uint256 _count)
func (_Redistribution *RedistributionFilterer) FilterCountCommits(opts *bind.FilterOpts) (*RedistributionCountCommitsIterator, error) {

	logs, sub, err := _Redistribution.contract.FilterLogs(opts, "CountCommits")
	if err != nil {
		return nil, err
	}
	return &RedistributionCountCommitsIterator{contract: _Redistribution.contract, event: "CountCommits", logs: logs, sub: sub}, nil
}

// WatchCountCommits is a free log subscription operation binding the contract event 0x6752c5e71c95fb93bc7137adeb115a33fa4e54e2683e33d3f90c2bb1c4b6c2a5.
//
// Solidity: event CountCommits(uint256 _count)
func (_Redistribution *RedistributionFilterer) WatchCountCommits(opts *bind.WatchOpts, sink chan<- *RedistributionCountCommits) (event.Subscription, error) {

	logs, sub, err := _Redistribution.contract.WatchLogs(opts, "CountCommits")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RedistributionCountCommits)
				if err := _Redistribution.contract.UnpackLog(event, "CountCommits", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseCountCommits is a log parse operation binding the contract event 0x6752c5e71c95fb93bc7137adeb115a33fa4e54e2683e33d3f90c2bb1c4b6c2a5.
//
// Solidity: event CountCommits(uint256 _count)
func (_Redistribution *RedistributionFilterer) ParseCountCommits(log types.Log) (*RedistributionCountCommits, error) {
	event := new(RedistributionCountCommits)
	if err := _Redistribution.contract.UnpackLog(event, "CountCommits", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RedistributionCountRevealsIterator is returned from FilterCountReveals and is used to iterate over the raw logs and unpacked data for CountReveals events raised by the Redistribution contract.
type RedistributionCountRevealsIterator struct {
	Event *RedistributionCountReveals // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RedistributionCountRevealsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RedistributionCountReveals)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RedistributionCountReveals)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RedistributionCountRevealsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RedistributionCountRevealsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RedistributionCountReveals represents a CountReveals event raised by the Redistribution contract.
type RedistributionCountReveals struct {
	Count *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterCountReveals is a free log retrieval operation binding the contract event 0x4c03de6a759749c0c9387b7014634dc5c6af610e1366023d90751c783a998f8d.
//
// Solidity: event CountReveals(uint256 _count)
func (_Redistribution *RedistributionFilterer) FilterCountReveals(opts *bind.FilterOpts) (*RedistributionCountRevealsIterator, error) {

	logs, sub, err := _Redistribution.contract.FilterLogs(opts, "CountReveals")
	if err != nil {
		return nil, err
	}
	return &RedistributionCountRevealsIterator{contract: _Redistribution.contract, event: "CountReveals", logs: logs, sub: sub}, nil
}

// WatchCountReveals is a free log subscription operation binding the contract event 0x4c03de6a759749c0c9387b7014634dc5c6af610e1366023d90751c783a998f8d.
//
// Solidity: event CountReveals(uint256 _count)
func (_Redistribution *RedistributionFilterer) WatchCountReveals(opts *bind.WatchOpts, sink chan<- *RedistributionCountReveals) (event.Subscription, error) {

	logs, sub, err := _Redistribution.contract.WatchLogs(opts, "CountReveals")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RedistributionCountReveals)
				if err := _Redistribution.contract.UnpackLog(event, "CountReveals", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseCountReveals is a log parse operation binding the contract event 0x4c03de6a759749c0c9387b7014634dc5c6af610e1366023d90751c783a998f8d.
//
// Solidity: event CountReveals(uint256 _count)
func (_Redistribution *RedistributionFilterer) ParseCountReveals(log types.Log) (*RedistributionCountReveals, error) {
	event := new(RedistributionCountReveals)
	if err := _Redistribution.contract.UnpackLog(event, "CountReveals", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RedistributionPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the Redistribution contract.
type RedistributionPausedIterator struct {
	Event *RedistributionPaused // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RedistributionPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RedistributionPaused)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RedistributionPaused)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RedistributionPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RedistributionPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RedistributionPaused represents a Paused event raised by the Redistribution contract.
type RedistributionPaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_Redistribution *RedistributionFilterer) FilterPaused(opts *bind.FilterOpts) (*RedistributionPausedIterator, error) {

	logs, sub, err := _Redistribution.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &RedistributionPausedIterator{contract: _Redistribution.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_Redistribution *RedistributionFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *RedistributionPaused) (event.Subscription, error) {

	logs, sub, err := _Redistribution.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RedistributionPaused)
				if err := _Redistribution.contract.UnpackLog(event, "Paused", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePaused is a log parse operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_Redistribution *RedistributionFilterer) ParsePaused(log types.Log) (*RedistributionPaused, error) {
	event := new(RedistributionPaused)
	if err := _Redistribution.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RedistributionRevealedIterator is returned from FilterRevealed and is used to iterate over the raw logs and unpacked data for Revealed events raised by the Redistribution contract.
type RedistributionRevealedIterator struct {
	Event *RedistributionRevealed // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RedistributionRevealedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RedistributionRevealed)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RedistributionRevealed)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RedistributionRevealedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RedistributionRevealedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RedistributionRevealed represents a Revealed event raised by the Redistribution contract.
type RedistributionRevealed struct {
	RoundNumber       *big.Int
	Overlay           [32]byte
	Stake             *big.Int
	StakeDensity      *big.Int
	ReserveCommitment [32]byte
	Depth             uint8
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRevealed is a free log retrieval operation binding the contract event 0x13fc17fd71632266fe82092de6dd91a06b4fa68d8dc950492e5421cbed55a6a5.
//
// Solidity: event Revealed(uint256 roundNumber, bytes32 overlay, uint256 stake, uint256 stakeDensity, bytes32 reserveCommitment, uint8 depth)
func (_Redistribution *RedistributionFilterer) FilterRevealed(opts *bind.FilterOpts) (*RedistributionRevealedIterator, error) {

	logs, sub, err := _Redistribution.contract.FilterLogs(opts, "Revealed")
	if err != nil {
		return nil, err
	}
	return &RedistributionRevealedIterator{contract: _Redistribution.contract, event: "Revealed", logs: logs, sub: sub}, nil
}

// WatchRevealed is a free log subscription operation binding the contract event 0x13fc17fd71632266fe82092de6dd91a06b4fa68d8dc950492e5421cbed55a6a5.
//
// Solidity: event Revealed(uint256 roundNumber, bytes32 overlay, uint256 stake, uint256 stakeDensity, bytes32 reserveCommitment, uint8 depth)
func (_Redistribution *RedistributionFilterer) WatchRevealed(opts *bind.WatchOpts, sink chan<- *RedistributionRevealed) (event.Subscription, error) {

	logs, sub, err := _Redistribution.contract.WatchLogs(opts, "Revealed")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RedistributionRevealed)
				if err := _Redistribution.contract.UnpackLog(event, "Revealed", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRevealed is a log parse operation binding the contract event 0x13fc17fd71632266fe82092de6dd91a06b4fa68d8dc950492e5421cbed55a6a5.
//
// Solidity: event Revealed(uint256 roundNumber, bytes32 overlay, uint256 stake, uint256 stakeDensity, bytes32 reserveCommitment, uint8 depth)
func (_Redistribution *RedistributionFilterer) ParseRevealed(log types.Log) (*RedistributionRevealed, error) {
	event := new(RedistributionRevealed)
	if err := _Redistribution.contract.UnpackLog(event, "Revealed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RedistributionRoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the Redistribution contract.
type RedistributionRoleAdminChangedIterator struct {
	Event *RedistributionRoleAdminChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RedistributionRoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RedistributionRoleAdminChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RedistributionRoleAdminChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RedistributionRoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RedistributionRoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RedistributionRoleAdminChanged represents a RoleAdminChanged event raised by the Redistribution contract.
type RedistributionRoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_Redistribution *RedistributionFilterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*RedistributionRoleAdminChangedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []interface{}
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []interface{}
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _Redistribution.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &RedistributionRoleAdminChangedIterator{contract: _Redistribution.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_Redistribution *RedistributionFilterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *RedistributionRoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []interface{}
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []interface{}
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _Redistribution.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RedistributionRoleAdminChanged)
				if err := _Redistribution.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleAdminChanged is a log parse operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_Redistribution *RedistributionFilterer) ParseRoleAdminChanged(log types.Log) (*RedistributionRoleAdminChanged, error) {
	event := new(RedistributionRoleAdminChanged)
	if err := _Redistribution.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RedistributionRoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the Redistribution contract.
type RedistributionRoleGrantedIterator struct {
	Event *RedistributionRoleGranted // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RedistributionRoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RedistributionRoleGranted)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RedistributionRoleGranted)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RedistributionRoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RedistributionRoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RedistributionRoleGranted represents a RoleGranted event raised by the Redistribution contract.
type RedistributionRoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_Redistribution *RedistributionFilterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*RedistributionRoleGrantedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Redistribution.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &RedistributionRoleGrantedIterator{contract: _Redistribution.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_Redistribution *RedistributionFilterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *RedistributionRoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Redistribution.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RedistributionRoleGranted)
				if err := _Redistribution.contract.UnpackLog(event, "RoleGranted", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleGranted is a log parse operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_Redistribution *RedistributionFilterer) ParseRoleGranted(log types.Log) (*RedistributionRoleGranted, error) {
	event := new(RedistributionRoleGranted)
	if err := _Redistribution.contract.UnpackLog(event, "RoleGranted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RedistributionRoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the Redistribution contract.
type RedistributionRoleRevokedIterator struct {
	Event *RedistributionRoleRevoked // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RedistributionRoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RedistributionRoleRevoked)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RedistributionRoleRevoked)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RedistributionRoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RedistributionRoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RedistributionRoleRevoked represents a RoleRevoked event raised by the Redistribution contract.
type RedistributionRoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_Redistribution *RedistributionFilterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*RedistributionRoleRevokedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Redistribution.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &RedistributionRoleRevokedIterator{contract: _Redistribution.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_Redistribution *RedistributionFilterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *RedistributionRoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Redistribution.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RedistributionRoleRevoked)
				if err := _Redistribution.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleRevoked is a log parse operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_Redistribution *RedistributionFilterer) ParseRoleRevoked(log types.Log) (*RedistributionRoleRevoked, error) {
	event := new(RedistributionRoleRevoked)
	if err := _Redistribution.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RedistributionTruthSelectedIterator is returned from FilterTruthSelected and is used to iterate over the raw logs and unpacked data for TruthSelected events raised by the Redistribution contract.
type RedistributionTruthSelectedIterator struct {
	Event *RedistributionTruthSelected // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RedistributionTruthSelectedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RedistributionTruthSelected)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RedistributionTruthSelected)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RedistributionTruthSelectedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RedistributionTruthSelectedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RedistributionTruthSelected represents a TruthSelected event raised by the Redistribution contract.
type RedistributionTruthSelected struct {
	Hash  [32]byte
	Depth uint8
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTruthSelected is a free log retrieval operation binding the contract event 0x34e8eda4cd857cd2865becf58a47748f31415f4a382cbb2cc0c64b9a27c717be.
//
// Solidity: event TruthSelected(bytes32 hash, uint8 depth)
func (_Redistribution *RedistributionFilterer) FilterTruthSelected(opts *bind.FilterOpts) (*RedistributionTruthSelectedIterator, error) {

	logs, sub, err := _Redistribution.contract.FilterLogs(opts, "TruthSelected")
	if err != nil {
		return nil, err
	}
	return &RedistributionTruthSelectedIterator{contract: _Redistribution.contract, event: "TruthSelected", logs: logs, sub: sub}, nil
}

// WatchTruthSelected is a free log subscription operation binding the contract event 0x34e8eda4cd857cd2865becf58a47748f31415f4a382cbb2cc0c64b9a27c717be.
//
// Solidity: event TruthSelected(bytes32 hash, uint8 depth)
func (_Redistribution *RedistributionFilterer) WatchTruthSelected(opts *bind.WatchOpts, sink chan<- *RedistributionTruthSelected) (event.Subscription, error) {

	logs, sub, err := _Redistribution.contract.WatchLogs(opts, "TruthSelected")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RedistributionTruthSelected)
				if err := _Redistribution.contract.UnpackLog(event, "TruthSelected", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTruthSelected is a log parse operation binding the contract event 0x34e8eda4cd857cd2865becf58a47748f31415f4a382cbb2cc0c64b9a27c717be.
//
// Solidity: event TruthSelected(bytes32 hash, uint8 depth)
func (_Redistribution *RedistributionFilterer) ParseTruthSelected(log types.Log) (*RedistributionTruthSelected, error) {
	event := new(RedistributionTruthSelected)
	if err := _Redistribution.contract.UnpackLog(event, "TruthSelected", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RedistributionUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the Redistribution contract.
type RedistributionUnpausedIterator struct {
	Event *RedistributionUnpaused // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RedistributionUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RedistributionUnpaused)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RedistributionUnpaused)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RedistributionUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RedistributionUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RedistributionUnpaused represents a Unpaused event raised by the Redistribution contract.
type RedistributionUnpaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_Redistribution *RedistributionFilterer) FilterUnpaused(opts *bind.FilterOpts) (*RedistributionUnpausedIterator, error) {

	logs, sub, err := _Redistribution.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &RedistributionUnpausedIterator{contract: _Redistribution.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_Redistribution *RedistributionFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *RedistributionUnpaused) (event.Subscription, error) {

	logs, sub, err := _Redistribution.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RedistributionUnpaused)
				if err := _Redistribution.contract.UnpackLog(event, "Unpaused", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUnpaused is a log parse operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_Redistribution *RedistributionFilterer) ParseUnpaused(log types.Log) (*RedistributionUnpaused, error) {
	event := new(RedistributionUnpaused)
	if err := _Redistribution.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RedistributionWinnerSelectedIterator is returned from FilterWinnerSelected and is used to iterate over the raw logs and unpacked data for WinnerSelected events raised by the Redistribution contract.
type RedistributionWinnerSelectedIterator struct {
	Event *RedistributionWinnerSelected // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RedistributionWinnerSelectedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RedistributionWinnerSelected)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RedistributionWinnerSelected)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RedistributionWinnerSelectedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RedistributionWinnerSelectedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RedistributionWinnerSelected represents a WinnerSelected event raised by the Redistribution contract.
type RedistributionWinnerSelected struct {
	Winner RedistributionReveal
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterWinnerSelected is a free log retrieval operation binding the contract event 0x2756aa512df0e32847d196f374c5b2fa5f30705f2fe3a75b8baeac52f2af5b39.
//
// Solidity: event WinnerSelected((address,bytes32,uint256,uint256,bytes32,uint8) winner)
func (_Redistribution *RedistributionFilterer) FilterWinnerSelected(opts *bind.FilterOpts) (*RedistributionWinnerSelectedIterator, error) {

	logs, sub, err := _Redistribution.contract.FilterLogs(opts, "WinnerSelected")
	if err != nil {
		return nil, err
	}
	return &RedistributionWinnerSelectedIterator{contract: _Redistribution.contract, event: "WinnerSelected", logs: logs, sub: sub}, nil
}

// WatchWinnerSelected is a free log subscription operation binding the contract event 0x2756aa512df0e32847d196f374c5b2fa5f30705f2fe3a75b8baeac52f2af5b39.
//
// Solidity: event WinnerSelected((address,bytes32,uint256,uint256,bytes32,uint8) winner)
func (_Redistribution *RedistributionFilterer) WatchWinnerSelected(opts *bind.WatchOpts, sink chan<- *RedistributionWinnerSelected) (event.Subscription, error) {

	logs, sub, err := _Redistribution.contract.WatchLogs(opts, "WinnerSelected")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RedistributionWinnerSelected)
				if err := _Redistribution.contract.UnpackLog(event, "WinnerSelected", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseWinnerSelected is a log parse operation binding the contract event 0x2756aa512df0e32847d196f374c5b2fa5f30705f2fe3a75b8baeac52f2af5b39.
//
// Solidity: event WinnerSelected((address,bytes32,uint256,uint256,bytes32,uint8) winner)
func (_Redistribution *RedistributionFilterer) ParseWinnerSelected(log types.Log) (*RedistributionWinnerSelected, error) {
	event := new(RedistributionWinnerSelected)
	if err := _Redistribution.contract.UnpackLog(event, "WinnerSelected", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
