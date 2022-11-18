'use strict';

var utils = require('../utils/writer.js');
var Health = require('../service/HealthService');

module.exports.apiServer/debug = function apiServer/debug (req, res, next) {
  Health.apiServer/debug()
    .then(function (response) {
      utils.writeJson(res, response);
    })
    .catch(function (response) {
      utils.writeJson(res, response);
    });
};

module.exports.apiServer/healthz = function apiServer/healthz (req, res, next) {
  Health.apiServer/healthz()
    .then(function (response) {
      utils.writeJson(res, response);
    })
    .catch(function (response) {
      utils.writeJson(res, response);
    });
};

module.exports.apiServer/livez = function apiServer/livez (req, res, next) {
  Health.apiServer/livez()
    .then(function (response) {
      utils.writeJson(res, response);
    })
    .catch(function (response) {
      utils.writeJson(res, response);
    });
};

module.exports.apiServer/logz = function apiServer/logz (req, res, next) {
  Health.apiServer/logz()
    .then(function (response) {
      utils.writeJson(res, response);
    })
    .catch(function (response) {
      utils.writeJson(res, response);
    });
};

module.exports.apiServer/readyz = function apiServer/readyz (req, res, next) {
  Health.apiServer/readyz()
    .then(function (response) {
      utils.writeJson(res, response);
    })
    .catch(function (response) {
      utils.writeJson(res, response);
    });
};

module.exports.apiServer/varz = function apiServer/varz (req, res, next) {
  Health.apiServer/varz()
    .then(function (response) {
      utils.writeJson(res, response);
    })
    .catch(function (response) {
      utils.writeJson(res, response);
    });
};
