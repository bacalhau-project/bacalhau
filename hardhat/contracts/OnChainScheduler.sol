// SPDX-License-Identifier: MIT
pragma solidity >=0.8.4;
import "hardhat/console.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/Counters.sol";
import "@openzeppelin/contracts/utils/Strings.sol";

/**
    @notice An experimental contract for POC work to call Bacalhau jobs from FVM smart contracts
*/
contract OnChainScheduler is Ownable {
    struct Job {
        string id;
    }

    event EventJobCreated(Job job);
    event EventJobComplete(Job job);
    // event EventJobCancelled(Image image); ?

    constructor(
    ) {
        console.log("Greetings from the train");
    }

}
