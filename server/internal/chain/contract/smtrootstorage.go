// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contract

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
	_ = abi.ConvertType
)

// SMTRootStorageMetaData contains all meta data concerning the SMTRootStorage contract.
var SMTRootStorageMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_relayer\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"issuerId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"root\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"crlNumber\",\"type\":\"uint256\"}],\"name\":\"RootUpdated\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"issuerId\",\"type\":\"bytes32\"}],\"name\":\"getRoot\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"relayer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"roots\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"root\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"crlNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"issuerId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"newRoot\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"crlNumber\",\"type\":\"uint256\"}],\"name\":\"setRoot\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561000f575f5ffd5b506040516106a13803806106a1833981810160405281019061003191906100d4565b805f5f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550506100ff565b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6100a38261007a565b9050919050565b6100b381610099565b81146100bd575f5ffd5b50565b5f815190506100ce816100aa565b92915050565b5f602082840312156100e9576100e8610076565b5b5f6100f6848285016100c0565b91505092915050565b6105958061010c5f395ff3fe608060405234801561000f575f5ffd5b506004361061004a575f3560e01c80638406c0791461004e57806384f942211461006c578063ae6dead71461009c578063b7137076146100ce575b5f5ffd5b6100566100ea565b60405161006391906102fd565b60405180910390f35b6100866004803603810190610081919061034d565b61010e565b6040516100939190610390565b60405180910390f35b6100b660048036038101906100b1919061034d565b61012a565b6040516100c5939291906103a9565b60405180910390f35b6100e860048036038101906100e39190610408565b610150565b005b5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b5f60015f8381526020019081526020015f205f01549050919050565b6001602052805f5260405f205f91509050805f0154908060010154908060020154905083565b5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146101de576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016101d5906104b2565b60405180910390fd5b60015f8481526020019081526020015f20600101548111610234576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161022b9061051a565b60405180910390fd5b60405180606001604052808381526020018281526020014281525060015f8581526020019081526020015f205f820151815f01556020820151816001015560408201518160020155905050827f156798a72d63b37f86ff1ecc41eec4f30e3c7b325c8a410c2671f2e7fc0c30f383836040516102b1929190610538565b60405180910390a2505050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6102e7826102be565b9050919050565b6102f7816102dd565b82525050565b5f6020820190506103105f8301846102ee565b92915050565b5f5ffd5b5f819050919050565b61032c8161031a565b8114610336575f5ffd5b50565b5f8135905061034781610323565b92915050565b5f6020828403121561036257610361610316565b5b5f61036f84828501610339565b91505092915050565b5f819050919050565b61038a81610378565b82525050565b5f6020820190506103a35f830184610381565b92915050565b5f6060820190506103bc5f830186610381565b6103c96020830185610381565b6103d66040830184610381565b949350505050565b6103e781610378565b81146103f1575f5ffd5b50565b5f81359050610402816103de565b92915050565b5f5f5f6060848603121561041f5761041e610316565b5b5f61042c86828701610339565b935050602061043d868287016103f4565b925050604061044e868287016103f4565b9150509250925092565b5f82825260208201905092915050565b7f756e617574686f72697a656400000000000000000000000000000000000000005f82015250565b5f61049c600c83610458565b91506104a782610468565b602082019050919050565b5f6020820190508181035f8301526104c981610490565b9050919050565b7f7374616c652043524c00000000000000000000000000000000000000000000005f82015250565b5f610504600983610458565b915061050f826104d0565b602082019050919050565b5f6020820190508181035f830152610531816104f8565b9050919050565b5f60408201905061054b5f830185610381565b6105586020830184610381565b939250505056fea26469706673582212206ed6ef79bac3b6354bc69bad121ae58ac217ba95ed02e80bb6890189abae423c64736f6c634300081c0033",
}

// SMTRootStorageABI is the input ABI used to generate the binding from.
// Deprecated: Use SMTRootStorageMetaData.ABI instead.
var SMTRootStorageABI = SMTRootStorageMetaData.ABI

// SMTRootStorageBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SMTRootStorageMetaData.Bin instead.
var SMTRootStorageBin = SMTRootStorageMetaData.Bin

// DeploySMTRootStorage deploys a new Ethereum contract, binding an instance of SMTRootStorage to it.
func DeploySMTRootStorage(auth *bind.TransactOpts, backend bind.ContractBackend, _relayer common.Address) (common.Address, *types.Transaction, *SMTRootStorage, error) {
	parsed, err := SMTRootStorageMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SMTRootStorageBin), backend, _relayer)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SMTRootStorage{SMTRootStorageCaller: SMTRootStorageCaller{contract: contract}, SMTRootStorageTransactor: SMTRootStorageTransactor{contract: contract}, SMTRootStorageFilterer: SMTRootStorageFilterer{contract: contract}}, nil
}

// SMTRootStorage is an auto generated Go binding around an Ethereum contract.
type SMTRootStorage struct {
	SMTRootStorageCaller     // Read-only binding to the contract
	SMTRootStorageTransactor // Write-only binding to the contract
	SMTRootStorageFilterer   // Log filterer for contract events
}

// SMTRootStorageCaller is an auto generated read-only Go binding around an Ethereum contract.
type SMTRootStorageCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SMTRootStorageTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SMTRootStorageTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SMTRootStorageFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SMTRootStorageFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SMTRootStorageSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SMTRootStorageSession struct {
	Contract     *SMTRootStorage   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SMTRootStorageCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SMTRootStorageCallerSession struct {
	Contract *SMTRootStorageCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// SMTRootStorageTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SMTRootStorageTransactorSession struct {
	Contract     *SMTRootStorageTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// SMTRootStorageRaw is an auto generated low-level Go binding around an Ethereum contract.
type SMTRootStorageRaw struct {
	Contract *SMTRootStorage // Generic contract binding to access the raw methods on
}

// SMTRootStorageCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SMTRootStorageCallerRaw struct {
	Contract *SMTRootStorageCaller // Generic read-only contract binding to access the raw methods on
}

// SMTRootStorageTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SMTRootStorageTransactorRaw struct {
	Contract *SMTRootStorageTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSMTRootStorage creates a new instance of SMTRootStorage, bound to a specific deployed contract.
func NewSMTRootStorage(address common.Address, backend bind.ContractBackend) (*SMTRootStorage, error) {
	contract, err := bindSMTRootStorage(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SMTRootStorage{SMTRootStorageCaller: SMTRootStorageCaller{contract: contract}, SMTRootStorageTransactor: SMTRootStorageTransactor{contract: contract}, SMTRootStorageFilterer: SMTRootStorageFilterer{contract: contract}}, nil
}

// NewSMTRootStorageCaller creates a new read-only instance of SMTRootStorage, bound to a specific deployed contract.
func NewSMTRootStorageCaller(address common.Address, caller bind.ContractCaller) (*SMTRootStorageCaller, error) {
	contract, err := bindSMTRootStorage(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SMTRootStorageCaller{contract: contract}, nil
}

// NewSMTRootStorageTransactor creates a new write-only instance of SMTRootStorage, bound to a specific deployed contract.
func NewSMTRootStorageTransactor(address common.Address, transactor bind.ContractTransactor) (*SMTRootStorageTransactor, error) {
	contract, err := bindSMTRootStorage(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SMTRootStorageTransactor{contract: contract}, nil
}

// NewSMTRootStorageFilterer creates a new log filterer instance of SMTRootStorage, bound to a specific deployed contract.
func NewSMTRootStorageFilterer(address common.Address, filterer bind.ContractFilterer) (*SMTRootStorageFilterer, error) {
	contract, err := bindSMTRootStorage(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SMTRootStorageFilterer{contract: contract}, nil
}

// bindSMTRootStorage binds a generic wrapper to an already deployed contract.
func bindSMTRootStorage(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SMTRootStorageMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SMTRootStorage *SMTRootStorageRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SMTRootStorage.Contract.SMTRootStorageCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SMTRootStorage *SMTRootStorageRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SMTRootStorage.Contract.SMTRootStorageTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SMTRootStorage *SMTRootStorageRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SMTRootStorage.Contract.SMTRootStorageTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SMTRootStorage *SMTRootStorageCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SMTRootStorage.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SMTRootStorage *SMTRootStorageTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SMTRootStorage.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SMTRootStorage *SMTRootStorageTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SMTRootStorage.Contract.contract.Transact(opts, method, params...)
}

// GetRoot is a free data retrieval call binding the contract method 0x84f94221.
//
// Solidity: function getRoot(bytes32 issuerId) view returns(uint256)
func (_SMTRootStorage *SMTRootStorageCaller) GetRoot(opts *bind.CallOpts, issuerId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _SMTRootStorage.contract.Call(opts, &out, "getRoot", issuerId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetRoot is a free data retrieval call binding the contract method 0x84f94221.
//
// Solidity: function getRoot(bytes32 issuerId) view returns(uint256)
func (_SMTRootStorage *SMTRootStorageSession) GetRoot(issuerId [32]byte) (*big.Int, error) {
	return _SMTRootStorage.Contract.GetRoot(&_SMTRootStorage.CallOpts, issuerId)
}

// GetRoot is a free data retrieval call binding the contract method 0x84f94221.
//
// Solidity: function getRoot(bytes32 issuerId) view returns(uint256)
func (_SMTRootStorage *SMTRootStorageCallerSession) GetRoot(issuerId [32]byte) (*big.Int, error) {
	return _SMTRootStorage.Contract.GetRoot(&_SMTRootStorage.CallOpts, issuerId)
}

// Relayer is a free data retrieval call binding the contract method 0x8406c079.
//
// Solidity: function relayer() view returns(address)
func (_SMTRootStorage *SMTRootStorageCaller) Relayer(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SMTRootStorage.contract.Call(opts, &out, "relayer")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Relayer is a free data retrieval call binding the contract method 0x8406c079.
//
// Solidity: function relayer() view returns(address)
func (_SMTRootStorage *SMTRootStorageSession) Relayer() (common.Address, error) {
	return _SMTRootStorage.Contract.Relayer(&_SMTRootStorage.CallOpts)
}

// Relayer is a free data retrieval call binding the contract method 0x8406c079.
//
// Solidity: function relayer() view returns(address)
func (_SMTRootStorage *SMTRootStorageCallerSession) Relayer() (common.Address, error) {
	return _SMTRootStorage.Contract.Relayer(&_SMTRootStorage.CallOpts)
}

// Roots is a free data retrieval call binding the contract method 0xae6dead7.
//
// Solidity: function roots(bytes32 ) view returns(uint256 root, uint256 crlNumber, uint256 updatedAt)
func (_SMTRootStorage *SMTRootStorageCaller) Roots(opts *bind.CallOpts, arg0 [32]byte) (struct {
	Root      *big.Int
	CrlNumber *big.Int
	UpdatedAt *big.Int
}, error) {
	var out []interface{}
	err := _SMTRootStorage.contract.Call(opts, &out, "roots", arg0)

	outstruct := new(struct {
		Root      *big.Int
		CrlNumber *big.Int
		UpdatedAt *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Root = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.CrlNumber = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.UpdatedAt = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Roots is a free data retrieval call binding the contract method 0xae6dead7.
//
// Solidity: function roots(bytes32 ) view returns(uint256 root, uint256 crlNumber, uint256 updatedAt)
func (_SMTRootStorage *SMTRootStorageSession) Roots(arg0 [32]byte) (struct {
	Root      *big.Int
	CrlNumber *big.Int
	UpdatedAt *big.Int
}, error) {
	return _SMTRootStorage.Contract.Roots(&_SMTRootStorage.CallOpts, arg0)
}

// Roots is a free data retrieval call binding the contract method 0xae6dead7.
//
// Solidity: function roots(bytes32 ) view returns(uint256 root, uint256 crlNumber, uint256 updatedAt)
func (_SMTRootStorage *SMTRootStorageCallerSession) Roots(arg0 [32]byte) (struct {
	Root      *big.Int
	CrlNumber *big.Int
	UpdatedAt *big.Int
}, error) {
	return _SMTRootStorage.Contract.Roots(&_SMTRootStorage.CallOpts, arg0)
}

// SetRoot is a paid mutator transaction binding the contract method 0xb7137076.
//
// Solidity: function setRoot(bytes32 issuerId, uint256 newRoot, uint256 crlNumber) returns()
func (_SMTRootStorage *SMTRootStorageTransactor) SetRoot(opts *bind.TransactOpts, issuerId [32]byte, newRoot *big.Int, crlNumber *big.Int) (*types.Transaction, error) {
	return _SMTRootStorage.contract.Transact(opts, "setRoot", issuerId, newRoot, crlNumber)
}

// SetRoot is a paid mutator transaction binding the contract method 0xb7137076.
//
// Solidity: function setRoot(bytes32 issuerId, uint256 newRoot, uint256 crlNumber) returns()
func (_SMTRootStorage *SMTRootStorageSession) SetRoot(issuerId [32]byte, newRoot *big.Int, crlNumber *big.Int) (*types.Transaction, error) {
	return _SMTRootStorage.Contract.SetRoot(&_SMTRootStorage.TransactOpts, issuerId, newRoot, crlNumber)
}

// SetRoot is a paid mutator transaction binding the contract method 0xb7137076.
//
// Solidity: function setRoot(bytes32 issuerId, uint256 newRoot, uint256 crlNumber) returns()
func (_SMTRootStorage *SMTRootStorageTransactorSession) SetRoot(issuerId [32]byte, newRoot *big.Int, crlNumber *big.Int) (*types.Transaction, error) {
	return _SMTRootStorage.Contract.SetRoot(&_SMTRootStorage.TransactOpts, issuerId, newRoot, crlNumber)
}

// SMTRootStorageRootUpdatedIterator is returned from FilterRootUpdated and is used to iterate over the raw logs and unpacked data for RootUpdated events raised by the SMTRootStorage contract.
type SMTRootStorageRootUpdatedIterator struct {
	Event *SMTRootStorageRootUpdated // Event containing the contract specifics and raw log

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
func (it *SMTRootStorageRootUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SMTRootStorageRootUpdated)
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
		it.Event = new(SMTRootStorageRootUpdated)
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
func (it *SMTRootStorageRootUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SMTRootStorageRootUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SMTRootStorageRootUpdated represents a RootUpdated event raised by the SMTRootStorage contract.
type SMTRootStorageRootUpdated struct {
	IssuerId  [32]byte
	Root      *big.Int
	CrlNumber *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterRootUpdated is a free log retrieval operation binding the contract event 0x156798a72d63b37f86ff1ecc41eec4f30e3c7b325c8a410c2671f2e7fc0c30f3.
//
// Solidity: event RootUpdated(bytes32 indexed issuerId, uint256 root, uint256 crlNumber)
func (_SMTRootStorage *SMTRootStorageFilterer) FilterRootUpdated(opts *bind.FilterOpts, issuerId [][32]byte) (*SMTRootStorageRootUpdatedIterator, error) {

	var issuerIdRule []interface{}
	for _, issuerIdItem := range issuerId {
		issuerIdRule = append(issuerIdRule, issuerIdItem)
	}

	logs, sub, err := _SMTRootStorage.contract.FilterLogs(opts, "RootUpdated", issuerIdRule)
	if err != nil {
		return nil, err
	}
	return &SMTRootStorageRootUpdatedIterator{contract: _SMTRootStorage.contract, event: "RootUpdated", logs: logs, sub: sub}, nil
}

// WatchRootUpdated is a free log subscription operation binding the contract event 0x156798a72d63b37f86ff1ecc41eec4f30e3c7b325c8a410c2671f2e7fc0c30f3.
//
// Solidity: event RootUpdated(bytes32 indexed issuerId, uint256 root, uint256 crlNumber)
func (_SMTRootStorage *SMTRootStorageFilterer) WatchRootUpdated(opts *bind.WatchOpts, sink chan<- *SMTRootStorageRootUpdated, issuerId [][32]byte) (event.Subscription, error) {

	var issuerIdRule []interface{}
	for _, issuerIdItem := range issuerId {
		issuerIdRule = append(issuerIdRule, issuerIdItem)
	}

	logs, sub, err := _SMTRootStorage.contract.WatchLogs(opts, "RootUpdated", issuerIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SMTRootStorageRootUpdated)
				if err := _SMTRootStorage.contract.UnpackLog(event, "RootUpdated", log); err != nil {
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

// ParseRootUpdated is a log parse operation binding the contract event 0x156798a72d63b37f86ff1ecc41eec4f30e3c7b325c8a410c2671f2e7fc0c30f3.
//
// Solidity: event RootUpdated(bytes32 indexed issuerId, uint256 root, uint256 crlNumber)
func (_SMTRootStorage *SMTRootStorageFilterer) ParseRootUpdated(log types.Log) (*SMTRootStorageRootUpdated, error) {
	event := new(SMTRootStorageRootUpdated)
	if err := _SMTRootStorage.contract.UnpackLog(event, "RootUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
