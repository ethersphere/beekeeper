// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package stake

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

// StakeMetaData contains all meta data concerning the Stake contract.
var StakeMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_bzzToken\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"_NetworkId\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"previousAdminRole\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"newAdminRole\",\"type\":\"bytes32\"}],\"name\":\"RoleAdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleGranted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleRevoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"slashed\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"time\",\"type\":\"uint256\"}],\"name\":\"StakeFrozen\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"slashed\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"StakeSlashed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"overlay\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"stakeAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"lastUpdatedBlock\",\"type\":\"uint256\"}],\"name\":\"StakeUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DEFAULT_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"PAUSER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"REDISTRIBUTOR_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bzzToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"nonce\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"depositStake\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"overlay\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"time\",\"type\":\"uint256\"}],\"name\":\"freezeDeposit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleAdmin\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"grantRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"hasRole\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"overlay\",\"type\":\"bytes32\"}],\"name\":\"lastUpdatedBlockNumberOfOverlay\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"overlay\",\"type\":\"bytes32\"}],\"name\":\"ownerOfOverlay\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"renounceRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"revokeRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"overlay\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"slashDeposit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"overlay\",\"type\":\"bytes32\"}],\"name\":\"stakeOfOverlay\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"stakes\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"overlay\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"stakeAmount\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"lastUpdatedBlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isValue\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unPause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"overlay\",\"type\":\"bytes32\"}],\"name\":\"usableStakeOfOverlay\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"overlay\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawFromStake\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// StakeABI is the input ABI used to generate the binding from.
// Deprecated: Use StakeMetaData.ABI instead.
var StakeABI = StakeMetaData.ABI

// Stake is an auto generated Go binding around an Ethereum contract.
type Stake struct {
	StakeCaller     // Read-only binding to the contract
	StakeTransactor // Write-only binding to the contract
	StakeFilterer   // Log filterer for contract events
}

// StakeCaller is an auto generated read-only Go binding around an Ethereum contract.
type StakeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type StakeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StakeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StakeSession struct {
	Contract     *Stake            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StakeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StakeCallerSession struct {
	Contract *StakeCaller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// StakeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StakeTransactorSession struct {
	Contract     *StakeTransactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StakeRaw is an auto generated low-level Go binding around an Ethereum contract.
type StakeRaw struct {
	Contract *Stake // Generic contract binding to access the raw methods on
}

// StakeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StakeCallerRaw struct {
	Contract *StakeCaller // Generic read-only contract binding to access the raw methods on
}

// StakeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StakeTransactorRaw struct {
	Contract *StakeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewStake creates a new instance of Stake, bound to a specific deployed contract.
func NewStake(address common.Address, backend bind.ContractBackend) (*Stake, error) {
	contract, err := bindStake(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Stake{StakeCaller: StakeCaller{contract: contract}, StakeTransactor: StakeTransactor{contract: contract}, StakeFilterer: StakeFilterer{contract: contract}}, nil
}

// NewStakeCaller creates a new read-only instance of Stake, bound to a specific deployed contract.
func NewStakeCaller(address common.Address, caller bind.ContractCaller) (*StakeCaller, error) {
	contract, err := bindStake(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StakeCaller{contract: contract}, nil
}

// NewStakeTransactor creates a new write-only instance of Stake, bound to a specific deployed contract.
func NewStakeTransactor(address common.Address, transactor bind.ContractTransactor) (*StakeTransactor, error) {
	contract, err := bindStake(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StakeTransactor{contract: contract}, nil
}

// NewStakeFilterer creates a new log filterer instance of Stake, bound to a specific deployed contract.
func NewStakeFilterer(address common.Address, filterer bind.ContractFilterer) (*StakeFilterer, error) {
	contract, err := bindStake(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StakeFilterer{contract: contract}, nil
}

// bindStake binds a generic wrapper to an already deployed contract.
func bindStake(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(StakeABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Stake *StakeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Stake.Contract.StakeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Stake *StakeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Stake.Contract.StakeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Stake *StakeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Stake.Contract.StakeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Stake *StakeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Stake.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Stake *StakeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Stake.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Stake *StakeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Stake.Contract.contract.Transact(opts, method, params...)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Stake *StakeCaller) DEFAULTADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Stake.contract.Call(opts, &out, "DEFAULT_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Stake *StakeSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _Stake.Contract.DEFAULTADMINROLE(&_Stake.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Stake *StakeCallerSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _Stake.Contract.DEFAULTADMINROLE(&_Stake.CallOpts)
}

// PAUSERROLE is a free data retrieval call binding the contract method 0xe63ab1e9.
//
// Solidity: function PAUSER_ROLE() view returns(bytes32)
func (_Stake *StakeCaller) PAUSERROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Stake.contract.Call(opts, &out, "PAUSER_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// PAUSERROLE is a free data retrieval call binding the contract method 0xe63ab1e9.
//
// Solidity: function PAUSER_ROLE() view returns(bytes32)
func (_Stake *StakeSession) PAUSERROLE() ([32]byte, error) {
	return _Stake.Contract.PAUSERROLE(&_Stake.CallOpts)
}

// PAUSERROLE is a free data retrieval call binding the contract method 0xe63ab1e9.
//
// Solidity: function PAUSER_ROLE() view returns(bytes32)
func (_Stake *StakeCallerSession) PAUSERROLE() ([32]byte, error) {
	return _Stake.Contract.PAUSERROLE(&_Stake.CallOpts)
}

// REDISTRIBUTORROLE is a free data retrieval call binding the contract method 0xa6471a1d.
//
// Solidity: function REDISTRIBUTOR_ROLE() view returns(bytes32)
func (_Stake *StakeCaller) REDISTRIBUTORROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Stake.contract.Call(opts, &out, "REDISTRIBUTOR_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// REDISTRIBUTORROLE is a free data retrieval call binding the contract method 0xa6471a1d.
//
// Solidity: function REDISTRIBUTOR_ROLE() view returns(bytes32)
func (_Stake *StakeSession) REDISTRIBUTORROLE() ([32]byte, error) {
	return _Stake.Contract.REDISTRIBUTORROLE(&_Stake.CallOpts)
}

// REDISTRIBUTORROLE is a free data retrieval call binding the contract method 0xa6471a1d.
//
// Solidity: function REDISTRIBUTOR_ROLE() view returns(bytes32)
func (_Stake *StakeCallerSession) REDISTRIBUTORROLE() ([32]byte, error) {
	return _Stake.Contract.REDISTRIBUTORROLE(&_Stake.CallOpts)
}

// BzzToken is a free data retrieval call binding the contract method 0x420fc4db.
//
// Solidity: function bzzToken() view returns(address)
func (_Stake *StakeCaller) BzzToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Stake.contract.Call(opts, &out, "bzzToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BzzToken is a free data retrieval call binding the contract method 0x420fc4db.
//
// Solidity: function bzzToken() view returns(address)
func (_Stake *StakeSession) BzzToken() (common.Address, error) {
	return _Stake.Contract.BzzToken(&_Stake.CallOpts)
}

// BzzToken is a free data retrieval call binding the contract method 0x420fc4db.
//
// Solidity: function bzzToken() view returns(address)
func (_Stake *StakeCallerSession) BzzToken() (common.Address, error) {
	return _Stake.Contract.BzzToken(&_Stake.CallOpts)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Stake *StakeCaller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Stake.contract.Call(opts, &out, "getRoleAdmin", role)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Stake *StakeSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _Stake.Contract.GetRoleAdmin(&_Stake.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Stake *StakeCallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _Stake.Contract.GetRoleAdmin(&_Stake.CallOpts, role)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Stake *StakeCaller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []interface{}
	err := _Stake.contract.Call(opts, &out, "hasRole", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Stake *StakeSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _Stake.Contract.HasRole(&_Stake.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Stake *StakeCallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _Stake.Contract.HasRole(&_Stake.CallOpts, role, account)
}

// LastUpdatedBlockNumberOfOverlay is a free data retrieval call binding the contract method 0xede41302.
//
// Solidity: function lastUpdatedBlockNumberOfOverlay(bytes32 overlay) view returns(uint256)
func (_Stake *StakeCaller) LastUpdatedBlockNumberOfOverlay(opts *bind.CallOpts, overlay [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Stake.contract.Call(opts, &out, "lastUpdatedBlockNumberOfOverlay", overlay)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LastUpdatedBlockNumberOfOverlay is a free data retrieval call binding the contract method 0xede41302.
//
// Solidity: function lastUpdatedBlockNumberOfOverlay(bytes32 overlay) view returns(uint256)
func (_Stake *StakeSession) LastUpdatedBlockNumberOfOverlay(overlay [32]byte) (*big.Int, error) {
	return _Stake.Contract.LastUpdatedBlockNumberOfOverlay(&_Stake.CallOpts, overlay)
}

// LastUpdatedBlockNumberOfOverlay is a free data retrieval call binding the contract method 0xede41302.
//
// Solidity: function lastUpdatedBlockNumberOfOverlay(bytes32 overlay) view returns(uint256)
func (_Stake *StakeCallerSession) LastUpdatedBlockNumberOfOverlay(overlay [32]byte) (*big.Int, error) {
	return _Stake.Contract.LastUpdatedBlockNumberOfOverlay(&_Stake.CallOpts, overlay)
}

// OwnerOfOverlay is a free data retrieval call binding the contract method 0xa0d22b21.
//
// Solidity: function ownerOfOverlay(bytes32 overlay) view returns(address)
func (_Stake *StakeCaller) OwnerOfOverlay(opts *bind.CallOpts, overlay [32]byte) (common.Address, error) {
	var out []interface{}
	err := _Stake.contract.Call(opts, &out, "ownerOfOverlay", overlay)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OwnerOfOverlay is a free data retrieval call binding the contract method 0xa0d22b21.
//
// Solidity: function ownerOfOverlay(bytes32 overlay) view returns(address)
func (_Stake *StakeSession) OwnerOfOverlay(overlay [32]byte) (common.Address, error) {
	return _Stake.Contract.OwnerOfOverlay(&_Stake.CallOpts, overlay)
}

// OwnerOfOverlay is a free data retrieval call binding the contract method 0xa0d22b21.
//
// Solidity: function ownerOfOverlay(bytes32 overlay) view returns(address)
func (_Stake *StakeCallerSession) OwnerOfOverlay(overlay [32]byte) (common.Address, error) {
	return _Stake.Contract.OwnerOfOverlay(&_Stake.CallOpts, overlay)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_Stake *StakeCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Stake.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_Stake *StakeSession) Paused() (bool, error) {
	return _Stake.Contract.Paused(&_Stake.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_Stake *StakeCallerSession) Paused() (bool, error) {
	return _Stake.Contract.Paused(&_Stake.CallOpts)
}

// StakeOfOverlay is a free data retrieval call binding the contract method 0x48962b93.
//
// Solidity: function stakeOfOverlay(bytes32 overlay) view returns(uint256)
func (_Stake *StakeCaller) StakeOfOverlay(opts *bind.CallOpts, overlay [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Stake.contract.Call(opts, &out, "stakeOfOverlay", overlay)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StakeOfOverlay is a free data retrieval call binding the contract method 0x48962b93.
//
// Solidity: function stakeOfOverlay(bytes32 overlay) view returns(uint256)
func (_Stake *StakeSession) StakeOfOverlay(overlay [32]byte) (*big.Int, error) {
	return _Stake.Contract.StakeOfOverlay(&_Stake.CallOpts, overlay)
}

// StakeOfOverlay is a free data retrieval call binding the contract method 0x48962b93.
//
// Solidity: function stakeOfOverlay(bytes32 overlay) view returns(uint256)
func (_Stake *StakeCallerSession) StakeOfOverlay(overlay [32]byte) (*big.Int, error) {
	return _Stake.Contract.StakeOfOverlay(&_Stake.CallOpts, overlay)
}

// Stakes is a free data retrieval call binding the contract method 0x8fee6407.
//
// Solidity: function stakes(bytes32 ) view returns(bytes32 overlay, uint256 stakeAmount, address owner, uint256 lastUpdatedBlockNumber, bool isValue)
func (_Stake *StakeCaller) Stakes(opts *bind.CallOpts, arg0 [32]byte) (struct {
	Overlay                [32]byte
	StakeAmount            *big.Int
	Owner                  common.Address
	LastUpdatedBlockNumber *big.Int
	IsValue                bool
}, error) {
	var out []interface{}
	err := _Stake.contract.Call(opts, &out, "stakes", arg0)

	outstruct := new(struct {
		Overlay                [32]byte
		StakeAmount            *big.Int
		Owner                  common.Address
		LastUpdatedBlockNumber *big.Int
		IsValue                bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Overlay = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.StakeAmount = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Owner = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	outstruct.LastUpdatedBlockNumber = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.IsValue = *abi.ConvertType(out[4], new(bool)).(*bool)

	return *outstruct, err

}

// Stakes is a free data retrieval call binding the contract method 0x8fee6407.
//
// Solidity: function stakes(bytes32 ) view returns(bytes32 overlay, uint256 stakeAmount, address owner, uint256 lastUpdatedBlockNumber, bool isValue)
func (_Stake *StakeSession) Stakes(arg0 [32]byte) (struct {
	Overlay                [32]byte
	StakeAmount            *big.Int
	Owner                  common.Address
	LastUpdatedBlockNumber *big.Int
	IsValue                bool
}, error) {
	return _Stake.Contract.Stakes(&_Stake.CallOpts, arg0)
}

// Stakes is a free data retrieval call binding the contract method 0x8fee6407.
//
// Solidity: function stakes(bytes32 ) view returns(bytes32 overlay, uint256 stakeAmount, address owner, uint256 lastUpdatedBlockNumber, bool isValue)
func (_Stake *StakeCallerSession) Stakes(arg0 [32]byte) (struct {
	Overlay                [32]byte
	StakeAmount            *big.Int
	Owner                  common.Address
	LastUpdatedBlockNumber *big.Int
	IsValue                bool
}, error) {
	return _Stake.Contract.Stakes(&_Stake.CallOpts, arg0)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Stake *StakeCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _Stake.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Stake *StakeSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Stake.Contract.SupportsInterface(&_Stake.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Stake *StakeCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Stake.Contract.SupportsInterface(&_Stake.CallOpts, interfaceId)
}

// UsableStakeOfOverlay is a free data retrieval call binding the contract method 0xabe38543.
//
// Solidity: function usableStakeOfOverlay(bytes32 overlay) view returns(uint256)
func (_Stake *StakeCaller) UsableStakeOfOverlay(opts *bind.CallOpts, overlay [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Stake.contract.Call(opts, &out, "usableStakeOfOverlay", overlay)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// UsableStakeOfOverlay is a free data retrieval call binding the contract method 0xabe38543.
//
// Solidity: function usableStakeOfOverlay(bytes32 overlay) view returns(uint256)
func (_Stake *StakeSession) UsableStakeOfOverlay(overlay [32]byte) (*big.Int, error) {
	return _Stake.Contract.UsableStakeOfOverlay(&_Stake.CallOpts, overlay)
}

// UsableStakeOfOverlay is a free data retrieval call binding the contract method 0xabe38543.
//
// Solidity: function usableStakeOfOverlay(bytes32 overlay) view returns(uint256)
func (_Stake *StakeCallerSession) UsableStakeOfOverlay(overlay [32]byte) (*big.Int, error) {
	return _Stake.Contract.UsableStakeOfOverlay(&_Stake.CallOpts, overlay)
}

// DepositStake is a paid mutator transaction binding the contract method 0x1ed2cd40.
//
// Solidity: function depositStake(address _owner, bytes32 nonce, uint256 amount) returns()
func (_Stake *StakeTransactor) DepositStake(opts *bind.TransactOpts, _owner common.Address, nonce [32]byte, amount *big.Int) (*types.Transaction, error) {
	return _Stake.contract.Transact(opts, "depositStake", _owner, nonce, amount)
}

// DepositStake is a paid mutator transaction binding the contract method 0x1ed2cd40.
//
// Solidity: function depositStake(address _owner, bytes32 nonce, uint256 amount) returns()
func (_Stake *StakeSession) DepositStake(_owner common.Address, nonce [32]byte, amount *big.Int) (*types.Transaction, error) {
	return _Stake.Contract.DepositStake(&_Stake.TransactOpts, _owner, nonce, amount)
}

// DepositStake is a paid mutator transaction binding the contract method 0x1ed2cd40.
//
// Solidity: function depositStake(address _owner, bytes32 nonce, uint256 amount) returns()
func (_Stake *StakeTransactorSession) DepositStake(_owner common.Address, nonce [32]byte, amount *big.Int) (*types.Transaction, error) {
	return _Stake.Contract.DepositStake(&_Stake.TransactOpts, _owner, nonce, amount)
}

// FreezeDeposit is a paid mutator transaction binding the contract method 0x837fd16a.
//
// Solidity: function freezeDeposit(bytes32 overlay, uint256 time) returns()
func (_Stake *StakeTransactor) FreezeDeposit(opts *bind.TransactOpts, overlay [32]byte, time *big.Int) (*types.Transaction, error) {
	return _Stake.contract.Transact(opts, "freezeDeposit", overlay, time)
}

// FreezeDeposit is a paid mutator transaction binding the contract method 0x837fd16a.
//
// Solidity: function freezeDeposit(bytes32 overlay, uint256 time) returns()
func (_Stake *StakeSession) FreezeDeposit(overlay [32]byte, time *big.Int) (*types.Transaction, error) {
	return _Stake.Contract.FreezeDeposit(&_Stake.TransactOpts, overlay, time)
}

// FreezeDeposit is a paid mutator transaction binding the contract method 0x837fd16a.
//
// Solidity: function freezeDeposit(bytes32 overlay, uint256 time) returns()
func (_Stake *StakeTransactorSession) FreezeDeposit(overlay [32]byte, time *big.Int) (*types.Transaction, error) {
	return _Stake.Contract.FreezeDeposit(&_Stake.TransactOpts, overlay, time)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Stake *StakeTransactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Stake.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Stake *StakeSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Stake.Contract.GrantRole(&_Stake.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Stake *StakeTransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Stake.Contract.GrantRole(&_Stake.TransactOpts, role, account)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_Stake *StakeTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Stake.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_Stake *StakeSession) Pause() (*types.Transaction, error) {
	return _Stake.Contract.Pause(&_Stake.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_Stake *StakeTransactorSession) Pause() (*types.Transaction, error) {
	return _Stake.Contract.Pause(&_Stake.TransactOpts)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Stake *StakeTransactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Stake.contract.Transact(opts, "renounceRole", role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Stake *StakeSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Stake.Contract.RenounceRole(&_Stake.TransactOpts, role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Stake *StakeTransactorSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Stake.Contract.RenounceRole(&_Stake.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Stake *StakeTransactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Stake.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Stake *StakeSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Stake.Contract.RevokeRole(&_Stake.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Stake *StakeTransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Stake.Contract.RevokeRole(&_Stake.TransactOpts, role, account)
}

// SlashDeposit is a paid mutator transaction binding the contract method 0xa6ce31d4.
//
// Solidity: function slashDeposit(bytes32 overlay, uint256 amount) returns()
func (_Stake *StakeTransactor) SlashDeposit(opts *bind.TransactOpts, overlay [32]byte, amount *big.Int) (*types.Transaction, error) {
	return _Stake.contract.Transact(opts, "slashDeposit", overlay, amount)
}

// SlashDeposit is a paid mutator transaction binding the contract method 0xa6ce31d4.
//
// Solidity: function slashDeposit(bytes32 overlay, uint256 amount) returns()
func (_Stake *StakeSession) SlashDeposit(overlay [32]byte, amount *big.Int) (*types.Transaction, error) {
	return _Stake.Contract.SlashDeposit(&_Stake.TransactOpts, overlay, amount)
}

// SlashDeposit is a paid mutator transaction binding the contract method 0xa6ce31d4.
//
// Solidity: function slashDeposit(bytes32 overlay, uint256 amount) returns()
func (_Stake *StakeTransactorSession) SlashDeposit(overlay [32]byte, amount *big.Int) (*types.Transaction, error) {
	return _Stake.Contract.SlashDeposit(&_Stake.TransactOpts, overlay, amount)
}

// UnPause is a paid mutator transaction binding the contract method 0xf7b188a5.
//
// Solidity: function unPause() returns()
func (_Stake *StakeTransactor) UnPause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Stake.contract.Transact(opts, "unPause")
}

// UnPause is a paid mutator transaction binding the contract method 0xf7b188a5.
//
// Solidity: function unPause() returns()
func (_Stake *StakeSession) UnPause() (*types.Transaction, error) {
	return _Stake.Contract.UnPause(&_Stake.TransactOpts)
}

// UnPause is a paid mutator transaction binding the contract method 0xf7b188a5.
//
// Solidity: function unPause() returns()
func (_Stake *StakeTransactorSession) UnPause() (*types.Transaction, error) {
	return _Stake.Contract.UnPause(&_Stake.TransactOpts)
}

// WithdrawFromStake is a paid mutator transaction binding the contract method 0xe34c4527.
//
// Solidity: function withdrawFromStake(bytes32 overlay, uint256 amount) returns()
func (_Stake *StakeTransactor) WithdrawFromStake(opts *bind.TransactOpts, overlay [32]byte, amount *big.Int) (*types.Transaction, error) {
	return _Stake.contract.Transact(opts, "withdrawFromStake", overlay, amount)
}

// WithdrawFromStake is a paid mutator transaction binding the contract method 0xe34c4527.
//
// Solidity: function withdrawFromStake(bytes32 overlay, uint256 amount) returns()
func (_Stake *StakeSession) WithdrawFromStake(overlay [32]byte, amount *big.Int) (*types.Transaction, error) {
	return _Stake.Contract.WithdrawFromStake(&_Stake.TransactOpts, overlay, amount)
}

// WithdrawFromStake is a paid mutator transaction binding the contract method 0xe34c4527.
//
// Solidity: function withdrawFromStake(bytes32 overlay, uint256 amount) returns()
func (_Stake *StakeTransactorSession) WithdrawFromStake(overlay [32]byte, amount *big.Int) (*types.Transaction, error) {
	return _Stake.Contract.WithdrawFromStake(&_Stake.TransactOpts, overlay, amount)
}

// StakePausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the Stake contract.
type StakePausedIterator struct {
	Event *StakePaused // Event containing the contract specifics and raw log

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
func (it *StakePausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakePaused)
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
		it.Event = new(StakePaused)
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
func (it *StakePausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakePausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakePaused represents a Paused event raised by the Stake contract.
type StakePaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_Stake *StakeFilterer) FilterPaused(opts *bind.FilterOpts) (*StakePausedIterator, error) {

	logs, sub, err := _Stake.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &StakePausedIterator{contract: _Stake.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_Stake *StakeFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *StakePaused) (event.Subscription, error) {

	logs, sub, err := _Stake.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakePaused)
				if err := _Stake.contract.UnpackLog(event, "Paused", log); err != nil {
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
func (_Stake *StakeFilterer) ParsePaused(log types.Log) (*StakePaused, error) {
	event := new(StakePaused)
	if err := _Stake.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakeRoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the Stake contract.
type StakeRoleAdminChangedIterator struct {
	Event *StakeRoleAdminChanged // Event containing the contract specifics and raw log

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
func (it *StakeRoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakeRoleAdminChanged)
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
		it.Event = new(StakeRoleAdminChanged)
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
func (it *StakeRoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakeRoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakeRoleAdminChanged represents a RoleAdminChanged event raised by the Stake contract.
type StakeRoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_Stake *StakeFilterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*StakeRoleAdminChangedIterator, error) {

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

	logs, sub, err := _Stake.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &StakeRoleAdminChangedIterator{contract: _Stake.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_Stake *StakeFilterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *StakeRoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

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

	logs, sub, err := _Stake.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakeRoleAdminChanged)
				if err := _Stake.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
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
func (_Stake *StakeFilterer) ParseRoleAdminChanged(log types.Log) (*StakeRoleAdminChanged, error) {
	event := new(StakeRoleAdminChanged)
	if err := _Stake.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakeRoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the Stake contract.
type StakeRoleGrantedIterator struct {
	Event *StakeRoleGranted // Event containing the contract specifics and raw log

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
func (it *StakeRoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakeRoleGranted)
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
		it.Event = new(StakeRoleGranted)
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
func (it *StakeRoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakeRoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakeRoleGranted represents a RoleGranted event raised by the Stake contract.
type StakeRoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_Stake *StakeFilterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*StakeRoleGrantedIterator, error) {

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

	logs, sub, err := _Stake.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &StakeRoleGrantedIterator{contract: _Stake.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_Stake *StakeFilterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *StakeRoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _Stake.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakeRoleGranted)
				if err := _Stake.contract.UnpackLog(event, "RoleGranted", log); err != nil {
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
func (_Stake *StakeFilterer) ParseRoleGranted(log types.Log) (*StakeRoleGranted, error) {
	event := new(StakeRoleGranted)
	if err := _Stake.contract.UnpackLog(event, "RoleGranted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakeRoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the Stake contract.
type StakeRoleRevokedIterator struct {
	Event *StakeRoleRevoked // Event containing the contract specifics and raw log

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
func (it *StakeRoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakeRoleRevoked)
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
		it.Event = new(StakeRoleRevoked)
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
func (it *StakeRoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakeRoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakeRoleRevoked represents a RoleRevoked event raised by the Stake contract.
type StakeRoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_Stake *StakeFilterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*StakeRoleRevokedIterator, error) {

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

	logs, sub, err := _Stake.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &StakeRoleRevokedIterator{contract: _Stake.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_Stake *StakeFilterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *StakeRoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _Stake.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakeRoleRevoked)
				if err := _Stake.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
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
func (_Stake *StakeFilterer) ParseRoleRevoked(log types.Log) (*StakeRoleRevoked, error) {
	event := new(StakeRoleRevoked)
	if err := _Stake.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakeStakeFrozenIterator is returned from FilterStakeFrozen and is used to iterate over the raw logs and unpacked data for StakeFrozen events raised by the Stake contract.
type StakeStakeFrozenIterator struct {
	Event *StakeStakeFrozen // Event containing the contract specifics and raw log

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
func (it *StakeStakeFrozenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakeStakeFrozen)
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
		it.Event = new(StakeStakeFrozen)
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
func (it *StakeStakeFrozenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakeStakeFrozenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakeStakeFrozen represents a StakeFrozen event raised by the Stake contract.
type StakeStakeFrozen struct {
	Slashed [32]byte
	Time    *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterStakeFrozen is a free log retrieval operation binding the contract event 0x340439a63c1ee2404f5b7010cab559b4dcbfc28b8baab9acade354fd688ae2b9.
//
// Solidity: event StakeFrozen(bytes32 slashed, uint256 time)
func (_Stake *StakeFilterer) FilterStakeFrozen(opts *bind.FilterOpts) (*StakeStakeFrozenIterator, error) {

	logs, sub, err := _Stake.contract.FilterLogs(opts, "StakeFrozen")
	if err != nil {
		return nil, err
	}
	return &StakeStakeFrozenIterator{contract: _Stake.contract, event: "StakeFrozen", logs: logs, sub: sub}, nil
}

// WatchStakeFrozen is a free log subscription operation binding the contract event 0x340439a63c1ee2404f5b7010cab559b4dcbfc28b8baab9acade354fd688ae2b9.
//
// Solidity: event StakeFrozen(bytes32 slashed, uint256 time)
func (_Stake *StakeFilterer) WatchStakeFrozen(opts *bind.WatchOpts, sink chan<- *StakeStakeFrozen) (event.Subscription, error) {

	logs, sub, err := _Stake.contract.WatchLogs(opts, "StakeFrozen")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakeStakeFrozen)
				if err := _Stake.contract.UnpackLog(event, "StakeFrozen", log); err != nil {
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

// ParseStakeFrozen is a log parse operation binding the contract event 0x340439a63c1ee2404f5b7010cab559b4dcbfc28b8baab9acade354fd688ae2b9.
//
// Solidity: event StakeFrozen(bytes32 slashed, uint256 time)
func (_Stake *StakeFilterer) ParseStakeFrozen(log types.Log) (*StakeStakeFrozen, error) {
	event := new(StakeStakeFrozen)
	if err := _Stake.contract.UnpackLog(event, "StakeFrozen", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakeStakeSlashedIterator is returned from FilterStakeSlashed and is used to iterate over the raw logs and unpacked data for StakeSlashed events raised by the Stake contract.
type StakeStakeSlashedIterator struct {
	Event *StakeStakeSlashed // Event containing the contract specifics and raw log

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
func (it *StakeStakeSlashedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakeStakeSlashed)
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
		it.Event = new(StakeStakeSlashed)
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
func (it *StakeStakeSlashedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakeStakeSlashedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakeStakeSlashed represents a StakeSlashed event raised by the Stake contract.
type StakeStakeSlashed struct {
	Slashed [32]byte
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterStakeSlashed is a free log retrieval operation binding the contract event 0x0956b50d4d586f6b9c90800d4e713bd2b866e044decd047e6d70ea20314ab308.
//
// Solidity: event StakeSlashed(bytes32 slashed, uint256 amount)
func (_Stake *StakeFilterer) FilterStakeSlashed(opts *bind.FilterOpts) (*StakeStakeSlashedIterator, error) {

	logs, sub, err := _Stake.contract.FilterLogs(opts, "StakeSlashed")
	if err != nil {
		return nil, err
	}
	return &StakeStakeSlashedIterator{contract: _Stake.contract, event: "StakeSlashed", logs: logs, sub: sub}, nil
}

// WatchStakeSlashed is a free log subscription operation binding the contract event 0x0956b50d4d586f6b9c90800d4e713bd2b866e044decd047e6d70ea20314ab308.
//
// Solidity: event StakeSlashed(bytes32 slashed, uint256 amount)
func (_Stake *StakeFilterer) WatchStakeSlashed(opts *bind.WatchOpts, sink chan<- *StakeStakeSlashed) (event.Subscription, error) {

	logs, sub, err := _Stake.contract.WatchLogs(opts, "StakeSlashed")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakeStakeSlashed)
				if err := _Stake.contract.UnpackLog(event, "StakeSlashed", log); err != nil {
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

// ParseStakeSlashed is a log parse operation binding the contract event 0x0956b50d4d586f6b9c90800d4e713bd2b866e044decd047e6d70ea20314ab308.
//
// Solidity: event StakeSlashed(bytes32 slashed, uint256 amount)
func (_Stake *StakeFilterer) ParseStakeSlashed(log types.Log) (*StakeStakeSlashed, error) {
	event := new(StakeStakeSlashed)
	if err := _Stake.contract.UnpackLog(event, "StakeSlashed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakeStakeUpdatedIterator is returned from FilterStakeUpdated and is used to iterate over the raw logs and unpacked data for StakeUpdated events raised by the Stake contract.
type StakeStakeUpdatedIterator struct {
	Event *StakeStakeUpdated // Event containing the contract specifics and raw log

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
func (it *StakeStakeUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakeStakeUpdated)
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
		it.Event = new(StakeStakeUpdated)
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
func (it *StakeStakeUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakeStakeUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakeStakeUpdated represents a StakeUpdated event raised by the Stake contract.
type StakeStakeUpdated struct {
	Overlay          [32]byte
	StakeAmount      *big.Int
	Owner            common.Address
	LastUpdatedBlock *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterStakeUpdated is a free log retrieval operation binding the contract event 0x61e979698346a2aa868a3a9f08d30c846174841dc9b074bbf2a82d20554bc682.
//
// Solidity: event StakeUpdated(bytes32 indexed overlay, uint256 stakeAmount, address owner, uint256 lastUpdatedBlock)
func (_Stake *StakeFilterer) FilterStakeUpdated(opts *bind.FilterOpts, overlay [][32]byte) (*StakeStakeUpdatedIterator, error) {

	var overlayRule []interface{}
	for _, overlayItem := range overlay {
		overlayRule = append(overlayRule, overlayItem)
	}

	logs, sub, err := _Stake.contract.FilterLogs(opts, "StakeUpdated", overlayRule)
	if err != nil {
		return nil, err
	}
	return &StakeStakeUpdatedIterator{contract: _Stake.contract, event: "StakeUpdated", logs: logs, sub: sub}, nil
}

// WatchStakeUpdated is a free log subscription operation binding the contract event 0x61e979698346a2aa868a3a9f08d30c846174841dc9b074bbf2a82d20554bc682.
//
// Solidity: event StakeUpdated(bytes32 indexed overlay, uint256 stakeAmount, address owner, uint256 lastUpdatedBlock)
func (_Stake *StakeFilterer) WatchStakeUpdated(opts *bind.WatchOpts, sink chan<- *StakeStakeUpdated, overlay [][32]byte) (event.Subscription, error) {

	var overlayRule []interface{}
	for _, overlayItem := range overlay {
		overlayRule = append(overlayRule, overlayItem)
	}

	logs, sub, err := _Stake.contract.WatchLogs(opts, "StakeUpdated", overlayRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakeStakeUpdated)
				if err := _Stake.contract.UnpackLog(event, "StakeUpdated", log); err != nil {
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

// ParseStakeUpdated is a log parse operation binding the contract event 0x61e979698346a2aa868a3a9f08d30c846174841dc9b074bbf2a82d20554bc682.
//
// Solidity: event StakeUpdated(bytes32 indexed overlay, uint256 stakeAmount, address owner, uint256 lastUpdatedBlock)
func (_Stake *StakeFilterer) ParseStakeUpdated(log types.Log) (*StakeStakeUpdated, error) {
	event := new(StakeStakeUpdated)
	if err := _Stake.contract.UnpackLog(event, "StakeUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakeUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the Stake contract.
type StakeUnpausedIterator struct {
	Event *StakeUnpaused // Event containing the contract specifics and raw log

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
func (it *StakeUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakeUnpaused)
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
		it.Event = new(StakeUnpaused)
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
func (it *StakeUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakeUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakeUnpaused represents a Unpaused event raised by the Stake contract.
type StakeUnpaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_Stake *StakeFilterer) FilterUnpaused(opts *bind.FilterOpts) (*StakeUnpausedIterator, error) {

	logs, sub, err := _Stake.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &StakeUnpausedIterator{contract: _Stake.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_Stake *StakeFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *StakeUnpaused) (event.Subscription, error) {

	logs, sub, err := _Stake.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakeUnpaused)
				if err := _Stake.contract.UnpackLog(event, "Unpaused", log); err != nil {
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
func (_Stake *StakeFilterer) ParseUnpaused(log types.Log) (*StakeUnpaused, error) {
	event := new(StakeUnpaused)
	if err := _Stake.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
