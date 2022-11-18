'use strict';

var utils = require('../utils/writer.js');
var Misc = require('../service/MiscService');

module.exports.apiServer/id = function apiServer/id (req, res, next) {
  Misc.apiServer/id()
    .then(function (response) {
      utils.writeJson(res, response);
    })
    .catch(function (response) {
      utils.writeJson(res, response);
    });
};

module.exports.apiServer/peers = function apiServer/peers (req, res, next) {
  Misc.apiServer/peers()
    .then(function (response) {
      utils.writeJson(res, response);
    })
    .catch(function (response) {
      utils.writeJson(res, response);
    });
};

module.exports.apiServer/version = function apiServer/version (req, res, next, body) {
  Misc.apiServer/version(body)
    .then(function (response) {
      utils.writeJson(res, response);
    })
    .catch(function (response) {
      utils.writeJson(res, response);
    });
};
