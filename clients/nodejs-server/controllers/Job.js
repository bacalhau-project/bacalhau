'use strict';

var utils = require('../utils/writer.js');
var Job = require('../service/JobService');

module.exports.pkg/apiServer.submit = function pkg/apiServer.submit (req, res, next, body) {
  Job.pkg/apiServer.submit(body)
    .then(function (response) {
      utils.writeJson(res, response);
    })
    .catch(function (response) {
      utils.writeJson(res, response);
    });
};

module.exports.pkg/publicapi.list = function pkg/publicapi.list (req, res, next, body) {
  Job.pkg/publicapi.list(body)
    .then(function (response) {
      utils.writeJson(res, response);
    })
    .catch(function (response) {
      utils.writeJson(res, response);
    });
};

module.exports.pkg/publicapi/events = function pkg/publicapi/events (req, res, next, body) {
  Job.pkg/publicapi/events(body)
    .then(function (response) {
      utils.writeJson(res, response);
    })
    .catch(function (response) {
      utils.writeJson(res, response);
    });
};

module.exports.pkg/publicapi/localEvents = function pkg/publicapi/localEvents (req, res, next, body) {
  Job.pkg/publicapi/localEvents(body)
    .then(function (response) {
      utils.writeJson(res, response);
    })
    .catch(function (response) {
      utils.writeJson(res, response);
    });
};

module.exports.pkg/publicapi/results = function pkg/publicapi/results (req, res, next, body) {
  Job.pkg/publicapi/results(body)
    .then(function (response) {
      utils.writeJson(res, response);
    })
    .catch(function (response) {
      utils.writeJson(res, response);
    });
};

module.exports.pkg/publicapi/states = function pkg/publicapi/states (req, res, next, body) {
  Job.pkg/publicapi/states(body)
    .then(function (response) {
      utils.writeJson(res, response);
    })
    .catch(function (response) {
      utils.writeJson(res, response);
    });
};
