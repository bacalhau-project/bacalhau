const { resolve } = require('path');
const path = require('path');
import MiniCssExtractPlugin from 'mini-css-extract-plugin';

const stylesHandler = MiniCssExtractPlugin.loader;

module.exports = async ({ config }) => {
    config.resolve.alias['./cryptoFunctions'] = path.resolve(__dirname, '../tests/mocks/__mocks__/cryptoFunctions.ts')

    config.resolve.alias['react-router-dom'] = require.resolve('react-router-dom');
    
    config.
    
    config.module.rules.push(
        {
            test: /\.css$/i,
            use: [stylesHandler, 'css-loader', 'postcss-loader'],
        },
    ),
    
    config.module.rules.push({
        include: path.resolve(__dirname, '../src'),
    });

    return config;
};