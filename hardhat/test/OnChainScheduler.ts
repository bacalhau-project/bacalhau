import { loadFixture } from '@nomicfoundation/hardhat-network-helpers'
import { expect } from 'chai'
import { ethers } from 'hardhat'
import { BigNumber } from 'ethers'
import type { OnChainScheduler } from '../typechain-types/contracts/OnChainScheduler';

describe("DoWeSayHello", function () {
  
  describe("Hello", function () {
    it("Should deploy", async function () {
      const OnChainScheduler = await ethers.getContractFactory("OnChainScheduler")
      await expect(OnChainScheduler.deploy()).to.not.be.reverted
    })
  })

})
