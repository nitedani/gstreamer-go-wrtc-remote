const MiniCssExtractPlugin = require('mini-css-extract-plugin');
const CopyWebpackPlugin = require('copy-webpack-plugin');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const { CleanWebpackPlugin } = require('clean-webpack-plugin');
const TsconfigPathsPlugin = require('tsconfig-paths-webpack-plugin');
const WebpackMessages = require('webpack-messages');
const { join } = require('path');
const { cwd } = require('process');
const { ProvidePlugin } = require('webpack');

const fileExtensions = [
  'jpg',
  'jpeg',
  'png',
  'gif',
  'eot',
  'otf',
  'svg',
  'ttf',
  'woff',
  'woff2',
  'wav',
];

module.exports = {
  resolve: {
    fallback: {
      util: false,
      http: false,
    },
    extensions: ['.js', '.jsx', '.ts', '.tsx'],
    plugins: [
      new TsconfigPathsPlugin({
        configFile: join(cwd(), 'apps', 'gui', 'tsconfig.json'),
      }),
    ],
  },
  mode: 'production',
  entry: [join(cwd(), 'apps', 'gui', 'src', 'index.tsx')],
  output: {
    path: join(cwd(), 'dist', 'gui'),
    filename: '[name].[fullhash].js',
    publicPath: '/',
  },
  stats: 'errors-only',
  performance: {
    hints: false,
  },
  optimization: {
    splitChunks: {
      cacheGroups: {
        vendors: {
          test: /[\\/]node_modules[\\/]/,
          name: 'vendors',
          chunks: 'all',
        },
      },
    },
  },
  watchOptions: {
    ignored: '/node_modules/',
  },
  module: {
    rules: [
      {
        test: new RegExp('.(' + fileExtensions.join('|') + ')$'),
        loader: 'file-loader',
        options: {
          name: '[name].[ext]',
        },
        exclude: /node_modules/,
      },
      {
        test: /\.html$/,
        loader: 'html-loader',
        exclude: /node_modules/,
      },
      {
        exclude: /node_modules/,
        test: /\.(css|scss)$/,
        use: [
          MiniCssExtractPlugin.loader,
          {
            loader: 'css-loader',
          },
          'postcss-loader',
          {
            loader: 'sass-loader',
            options: {
              sourceMap: true,
            },
          },
        ],
      },
      {
        exclude: /node_modules/,
        test: /\.tsx?$/,
        loader: 'esbuild-loader',
        options: {
          loader: 'tsx',
          target: 'esnext',
        },
      },
    ],
  },
  plugins: [
    new WebpackMessages({
      name: 'gui',
      logger: (str) => console.log(`>> ${str}`),
    }),
    new CleanWebpackPlugin(),
    new MiniCssExtractPlugin(),
    new CopyWebpackPlugin({
      patterns: [
        {
          from: join(cwd(), 'apps', 'gui', 'public'),
          to: './',
          globOptions: {
            ignore: ['**/index.html'],
          },
        },
      ],
    }),
    new HtmlWebpackPlugin({
      template: join(cwd(), 'apps', 'gui', 'public', 'index.html'),
    }),
    new ProvidePlugin({
      React: 'react',
    }),
  ],
};
