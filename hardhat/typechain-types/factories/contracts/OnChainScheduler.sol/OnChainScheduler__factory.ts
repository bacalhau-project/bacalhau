/* Autogenerated file. Do not edit manually. */
/* tslint:disable */
/* eslint-disable */
import { Signer, utils, Contract, ContractFactory, Overrides } from "ethers";
import type { Provider, TransactionRequest } from "@ethersproject/providers";
import type { PromiseOrValue } from "../../../common";
import type {
  OnChainScheduler,
  OnChainSchedulerInterface,
} from "../../../contracts/OnChainScheduler.sol/OnChainScheduler";

const _abi = [
  {
    inputs: [],
    stateMutability: "nonpayable",
    type: "constructor",
  },
  {
    anonymous: false,
    inputs: [
      {
        components: [
          {
            internalType: "string",
            name: "id",
            type: "string",
          },
        ],
        indexed: false,
        internalType: "struct OnChainScheduler.Job",
        name: "job",
        type: "tuple",
      },
    ],
    name: "EventJobComplete",
    type: "event",
  },
  {
    anonymous: false,
    inputs: [
      {
        components: [
          {
            internalType: "string",
            name: "id",
            type: "string",
          },
        ],
        indexed: false,
        internalType: "struct OnChainScheduler.Job",
        name: "job",
        type: "tuple",
      },
    ],
    name: "EventJobCreated",
    type: "event",
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        internalType: "address",
        name: "previousOwner",
        type: "address",
      },
      {
        indexed: true,
        internalType: "address",
        name: "newOwner",
        type: "address",
      },
    ],
    name: "OwnershipTransferred",
    type: "event",
  },
  {
    inputs: [],
    name: "owner",
    outputs: [
      {
        internalType: "address",
        name: "",
        type: "address",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
  {
    inputs: [],
    name: "renounceOwnership",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "address",
        name: "newOwner",
        type: "address",
      },
    ],
    name: "transferOwnership",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function",
  },
] as const;

const _bytecode =
  "0x608060405234801561001057600080fd5b5061002d61002261007a60201b60201c565b61008260201b60201c565b6100756040518060400160405280601881526020017f4772656574696e67732066726f6d2074686520747261696e000000000000000081525061014660201b61014a1760201c565b6102c0565b600033905090565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff169050816000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508173ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a35050565b6101e28160405160240161015a919061029e565b6040516020818303038152906040527f41304fac000000000000000000000000000000000000000000000000000000007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff83818316178352505050506101e560201b60201c565b50565b60008151905060006a636f6e736f6c652e6c6f679050602083016000808483855afa5050505050565b600081519050919050565b600082825260208201905092915050565b60005b8381101561024857808201518184015260208101905061022d565b60008484015250505050565b6000601f19601f8301169050919050565b60006102708261020e565b61027a8185610219565b935061028a81856020860161022a565b61029381610254565b840191505092915050565b600060208201905081810360008301526102b88184610265565b905092915050565b6105f6806102cf6000396000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c8063715018a6146100465780638da5cb5b14610050578063f2fde38b1461006e575b600080fd5b61004e61008a565b005b61005861009e565b6040516100659190610397565b60405180910390f35b610088600480360381019061008391906103e3565b6100c7565b005b6100926101e3565b61009c6000610261565b565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b6100cf6101e3565b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff160361013e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161013590610493565b60405180910390fd5b61014781610261565b50565b6101e08160405160240161015e9190610532565b6040516020818303038152906040527f41304fac000000000000000000000000000000000000000000000000000000007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff8381831617835250505050610325565b50565b6101eb61034e565b73ffffffffffffffffffffffffffffffffffffffff1661020961009e565b73ffffffffffffffffffffffffffffffffffffffff161461025f576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610256906105a0565b60405180910390fd5b565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff169050816000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508173ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a35050565b60008151905060006a636f6e736f6c652e6c6f679050602083016000808483855afa5050505050565b600033905090565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b600061038182610356565b9050919050565b61039181610376565b82525050565b60006020820190506103ac6000830184610388565b92915050565b600080fd5b6103c081610376565b81146103cb57600080fd5b50565b6000813590506103dd816103b7565b92915050565b6000602082840312156103f9576103f86103b2565b5b6000610407848285016103ce565b91505092915050565b600082825260208201905092915050565b7f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160008201527f6464726573730000000000000000000000000000000000000000000000000000602082015250565b600061047d602683610410565b915061048882610421565b604082019050919050565b600060208201905081810360008301526104ac81610470565b9050919050565b600081519050919050565b60005b838110156104dc5780820151818401526020810190506104c1565b60008484015250505050565b6000601f19601f8301169050919050565b6000610504826104b3565b61050e8185610410565b935061051e8185602086016104be565b610527816104e8565b840191505092915050565b6000602082019050818103600083015261054c81846104f9565b905092915050565b7f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572600082015250565b600061058a602083610410565b915061059582610554565b602082019050919050565b600060208201905081810360008301526105b98161057d565b905091905056fea2646970667358221220fb2b56b03f2c0f166aebd6a5b1c2848d5bd7f5ad0cff4cd3c23b8a4ed54ba68864736f6c63430008110033";

type OnChainSchedulerConstructorParams =
  | [signer?: Signer]
  | ConstructorParameters<typeof ContractFactory>;

const isSuperArgs = (
  xs: OnChainSchedulerConstructorParams
): xs is ConstructorParameters<typeof ContractFactory> => xs.length > 1;

export class OnChainScheduler__factory extends ContractFactory {
  constructor(...args: OnChainSchedulerConstructorParams) {
    if (isSuperArgs(args)) {
      super(...args);
    } else {
      super(_abi, _bytecode, args[0]);
    }
  }

  override deploy(
    overrides?: Overrides & { from?: PromiseOrValue<string> }
  ): Promise<OnChainScheduler> {
    return super.deploy(overrides || {}) as Promise<OnChainScheduler>;
  }
  override getDeployTransaction(
    overrides?: Overrides & { from?: PromiseOrValue<string> }
  ): TransactionRequest {
    return super.getDeployTransaction(overrides || {});
  }
  override attach(address: string): OnChainScheduler {
    return super.attach(address) as OnChainScheduler;
  }
  override connect(signer: Signer): OnChainScheduler__factory {
    return super.connect(signer) as OnChainScheduler__factory;
  }

  static readonly bytecode = _bytecode;
  static readonly abi = _abi;
  static createInterface(): OnChainSchedulerInterface {
    return new utils.Interface(_abi) as OnChainSchedulerInterface;
  }
  static connect(
    address: string,
    signerOrProvider: Signer | Provider
  ): OnChainScheduler {
    return new Contract(address, _abi, signerOrProvider) as OnChainScheduler;
  }
}
