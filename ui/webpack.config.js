var webpack = require('webpack');
var path = require('path');

var APP_DIR = path.resolve(__dirname, 'src');
var BUILD_DIR = path.resolve(__dirname, 'dist');

var config = {
  devtool: 'cheap-module-source-map',
  entry: APP_DIR + '/index.jsx',
  output: {
    path: BUILD_DIR,
    filename: 'bundle.js',
    publicPath: '/dist/'
  },
  module : {
    loaders : [
      {test: /\.(js|jsx)$/,include : APP_DIR, loader : 'babel'},
      {test: /\.json$/, loader: "json"},
      {test: /\.css$/, loader: "style-loader!css-loader"},
      {test: /\.less$/, loader: 'style-loader!css-loader!less-loader'},
      {test: /\.(png|jpg|ico)$/, loader: 'url-loader?limit=100000'},
      {test: /\.woff(2)?(\?v=\d+\.\d+\.\d+)?$/, loader: "url?limit=10000&mimetype=application/font-woff&name=[name].[ext]" },
      {test: /\.ttf(\?v=\d+\.\d+\.\d+)?$/, loader: "url?limit=10000&mimetype=application/octet-stream&name=[name].[ext]" },
      {test: /\.eot(\?v=\d+\.\d+\.\d+)?$/, loader: "file" },
      {test: /\.svg(\?v=\d+\.\d+\.\d+)?$/, loader: "url?limit=10000&mimetype=image/svg+xml&name=[name].[ext]" }
    ]
  },
  resolve: {
    extensions: ['', '.js', '.jsx']
  },
  plugins:[
    {
      name: 'nodeenv',
      plugin: new webpack.DefinePlugin({
        'process.env':{
          'NODE_ENV': JSON.stringify('production')
        }
      })
    },
    {
      name: 'uglify',
      plugin: new webpack.optimize.UglifyJsPlugin({
        compress:{
          warnings: false
        }
      })
    }
  ]
};

module.exports = config;

module.exports.plugins.forEach( function(p,i) {
  if ( process.argv.indexOf( '--disable-' + p.name + '-plugin' ) === -1 ) {
    module.exports.plugins[i] = p.plugin;
  } else {
    module.exports.plugins[i] = function() {}
  }
});
