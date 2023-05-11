import { HardhatUserConfig, task } from 'hardhat/config';
import '@nomicfoundation/hardhat-toolbox';
import '@openzeppelin/hardhat-upgrades';
import 'hardhat-deploy';
import { config as dotenvConfig } from 'dotenv';
import { resolve } from 'path';

const dotenvConfigPath: string = process.env.DOTENV_CONFIG_PATH || './.env';
dotenvConfig({ path: resolve(__dirname, dotenvConfigPath) });

const config: HardhatUserConfig = {
  solidity: '0.8.17',
  defaultNetwork: 'hardhat',
  namedAccounts: {
    admin: 0,
  },
  networks: {
    hardhat: {},
    localhost: {},
  },
};

export default config;
