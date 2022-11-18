'use strict';


/**
 * Returns debug information on what the current node is doing.
 *
 * returns publicapi.debugResponse
 **/
exports.apiServer/debug = function() {
  return new Promise(function(resolve, reject) {
    var examples = {};
    examples['application/json'] = {
  "ComputeJobs" : [ {
    "ShardID" : "ShardID",
    "State" : "State"
  }, {
    "ShardID" : "ShardID",
    "State" : "State"
  } ],
  "AvailableComputeCapacity" : {
    "Memory" : 27487790694,
    "CPU" : 9.600000000000001,
    "Disk" : 212663867801,
    "GPU" : 1
  },
  "RequesterJobs" : [ {
    "ShardID" : "ShardID",
    "State" : "State",
    "CompletedNodesCount" : 6,
    "BiddingNodesCount" : 0
  }, {
    "ShardID" : "ShardID",
    "State" : "State",
    "CompletedNodesCount" : 6,
    "BiddingNodesCount" : 0
  } ]
};
    if (Object.keys(examples).length > 0) {
      resolve(examples[Object.keys(examples)[0]]);
    } else {
      resolve();
    }
  });
}


/**
 *
 * returns types.HealthInfo
 **/
exports.apiServer/healthz = function() {
  return new Promise(function(resolve, reject) {
    var examples = {};
    examples['application/json'] = {
  "FreeSpace" : {
    "IPFSMount" : {
      "All" : 0,
      "Used" : 1,
      "Free" : 6
    }
  }
};
    if (Object.keys(examples).length > 0) {
      resolve(examples[Object.keys(examples)[0]]);
    } else {
      resolve();
    }
  });
}


/**
 *
 * returns String
 **/
exports.apiServer/livez = function() {
  return new Promise(function(resolve, reject) {
    var examples = {};
    examples['application/json'] = "";
    if (Object.keys(examples).length > 0) {
      resolve(examples[Object.keys(examples)[0]]);
    } else {
      resolve();
    }
  });
}


/**
 *
 * returns String
 **/
exports.apiServer/logz = function() {
  return new Promise(function(resolve, reject) {
    var examples = {};
    examples['application/json'] = "";
    if (Object.keys(examples).length > 0) {
      resolve(examples[Object.keys(examples)[0]]);
    } else {
      resolve();
    }
  });
}


/**
 *
 * returns String
 **/
exports.apiServer/readyz = function() {
  return new Promise(function(resolve, reject) {
    var examples = {};
    examples['application/json'] = "";
    if (Object.keys(examples).length > 0) {
      resolve(examples[Object.keys(examples)[0]]);
    } else {
      resolve();
    }
  });
}


/**
 *
 * returns List
 **/
exports.apiServer/varz = function() {
  return new Promise(function(resolve, reject) {
    var examples = {};
    examples['application/json'] = [ 0, 0 ];
    if (Object.keys(examples).length > 0) {
      resolve(examples[Object.keys(examples)[0]]);
    } else {
      resolve();
    }
  });
}

