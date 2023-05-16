// SPDX-License-Identifier: MIT
pragma solidity >=0.8.4;
import "hardhat/console.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/Strings.sol";

/**
    @notice An experimental contract for POC work to call Bacalhau jobs from FVM smart contracts
*/
contract OnChainScheduler is Ownable {

    // the job spec will live on ipfs or filecoin
    // the same storage drivers as bacalhau will be used by the compute node to download
    // the spec based on it's cid
    struct Job {
      string cid;
      // where do we download the job spec from?
      string storageDriver;
      address owner;
      // how many copies of the job should be run?
      int replicas;
      // how many offers should be considered before selecting some nodes?
      int minBids;
      string solver;
      string[] computeNodes;
    }

    struct ComputeNode {
      // this id can by anything as long as it's unique amoungst other nodes
      string id;
      // who get's paid for running this node?
      address owner;
      int cpu;
      int ram;
      int gpu;
    }

    struct Solver {
      // this id can by anything as long as it's unique amoungst other nodes
      string id;
      // who get's paid for running this node?
      address owner;
      int cpu;
      int ram;
      int gpu;
    }

    event EventJobCreated(Job job);
    event EventComputeNodeCreated(ComputeNode node);
    event EventSolverCreated(ComputeNode node);
    event EventJobMatched(Job job, ComputeNode node, Solver solver);
    event EventExecutionComplete(Job job, ComputeNode node);
    event EventExecutionError(Job job, ComputeNode node);
    event EventVerificationSuccess(Job job, ComputeNode node);
    event EventVerificationError(Job job, ComputeNode node);

    mapping(string => Job) jobs;
    mapping(string => ComputeNode) nodes;
    mapping(string => Solver) solvers;

    constructor(
    ) {
        console.log("Greetings from the train");
    }

}
