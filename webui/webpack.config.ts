// Generated using webpack-cli https://github.com/webpack/webpack-cli

import CopyPlugin from 'copy-webpack-plugin';
import HtmlWebpackPlugin from 'html-webpack-plugin';
import MiniCssExtractPlugin from 'mini-css-extract-plugin';
import path from 'path';
import { Configuration, DefinePlugin } from "webpack";
import 'webpack-dev-server';
import { merge } from "webpack-merge";
import WorkboxWebpackPlugin from 'workbox-webpack-plugin';
const base = require("./config/webpack.config.js");

const isProduction = process.env.NODE_ENV == 'production';
const stylesHandler = MiniCssExtractPlugin.loader;

const customConfig: Configuration = {
  entry: './src/index.tsx',
  output: {
    filename: 'bundle.js',
    path: path.resolve(__dirname, 'build'),
  },
  devServer: {
    open: true,
    host: 'localhost',
    static: path.join(__dirname, "public"),
    compress: true,
    port: 8080
  },
  mode: 'development',
  plugins: [
    new HtmlWebpackPlugin({
      template: 'public/index.html',
    }),

    new MiniCssExtractPlugin(),
    new DefinePlugin({
      'process.env': {
        PUBLIC_URL: JSON.stringify(''),
      },
    }),
    new CopyPlugin({
      patterns: [
        {
          from: "public",
          to: "static",
          globOptions: {
            dot: true,
            gitignore: true,
            ignore: ["**/index.html", "**/mockServiceWorker.js"],
          },
        }
      ],
    }),
  ],
  module: {
    rules: [
      {
        test: /\.(ts|tsx)$/,
        exclude: /node_modules/,
        use: 'ts-loader',
      },
      {
        test: /\.s[ac]ss$/i,
        use: [stylesHandler, 'css-loader', 'postcss-loader', 'sass-loader'],
      },
      {
        test: /\.css$/i,
        use: [stylesHandler, 'css-loader', 'postcss-loader'],
      },
      {
        test: /\.(eot|svg|ttf|woff|woff2|png|jpg|gif)$/i,
        type: 'asset',
      },
    ],
  },
  resolve: {
    extensions: [".ts", ".tsx", ".js", ".json"], // Updated array with valid extensions
    alias: {
      '@': path.resolve(__dirname, 'src'), // Use a specific alias for your source directory
    },
    fallback: {
      "crypto": require.resolve("crypto-browserify"),
      "stream": require.resolve("stream-browserify")
    }
  },
};

const c = () => {
  if (isProduction) {
    customConfig.mode = 'production';

    customConfig.plugins = customConfig.plugins || []; // Initialize customConfig.plugins if it's undefined
    customConfig.plugins.push(new WorkboxWebpackPlugin.GenerateSW());
  }
  return customConfig;
}

module.exports = merge(base, c());

