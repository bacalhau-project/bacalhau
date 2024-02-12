const { resolve } = require('path');
const path = require('path');

module.exports = async ({ config }) => {
    config.resolve.alias['./cryptoFunctions'] = path.resolve(__dirname, '../tests/mocks/__mocks__/cryptoFunctions.ts')

    config.resolve.alias['react-router-dom'] = require.resolve('react-router-dom');
    
    config.module.rules.push({
        test: /\.scss$/,
        use: [
            'style-loader',
            'css-loader',
            'sass-loader',
        ],
        include: path.resolve(__dirname, '../src'),
    });

    return config;
};