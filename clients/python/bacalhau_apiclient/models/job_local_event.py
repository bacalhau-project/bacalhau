# coding: utf-8

"""
    Bacalhau API

    This page is the reference of the Bacalhau REST API. Project docs are available at https://docs.bacalhau.org/. Find more information about Bacalhau at https://github.com/filecoin-project/bacalhau.  # noqa: E501

    OpenAPI spec version: 0.3.18.post4
    Contact: team@bacalhau.org
    Generated by: https://github.com/swagger-api/swagger-codegen.git
"""


import pprint
import re  # noqa: F401

import six

from bacalhau_apiclient.configuration import Configuration


class JobLocalEvent(object):
    """NOTE: This class is auto generated by the swagger code generator program.

    Do not edit the class manually.
    """

    """
    Attributes:
      swagger_types (dict): The key is attribute name
                            and the value is attribute type.
      attribute_map (dict): The key is attribute name
                            and the value is json key in definition.
    """
    swagger_types = {
        'event_name': 'JobLocalEventType',
        'job_id': 'str',
        'shard_index': 'int',
        'target_node_id': 'str'
    }

    attribute_map = {
        'event_name': 'EventName',
        'job_id': 'JobID',
        'shard_index': 'ShardIndex',
        'target_node_id': 'TargetNodeID'
    }

    def __init__(self, event_name=None, job_id=None, shard_index=None, target_node_id=None, _configuration=None):  # noqa: E501
        """JobLocalEvent - a model defined in Swagger"""  # noqa: E501
        if _configuration is None:
            _configuration = Configuration()
        self._configuration = _configuration

        self._event_name = None
        self._job_id = None
        self._shard_index = None
        self._target_node_id = None
        self.discriminator = None

        if event_name is not None:
            self.event_name = event_name
        if job_id is not None:
            self.job_id = job_id
        if shard_index is not None:
            self.shard_index = shard_index
        if target_node_id is not None:
            self.target_node_id = target_node_id

    @property
    def event_name(self):
        """Gets the event_name of this JobLocalEvent.  # noqa: E501


        :return: The event_name of this JobLocalEvent.  # noqa: E501
        :rtype: JobLocalEventType
        """
        return self._event_name

    @event_name.setter
    def event_name(self, event_name):
        """Sets the event_name of this JobLocalEvent.


        :param event_name: The event_name of this JobLocalEvent.  # noqa: E501
        :type: JobLocalEventType
        """

        self._event_name = event_name

    @property
    def job_id(self):
        """Gets the job_id of this JobLocalEvent.  # noqa: E501


        :return: The job_id of this JobLocalEvent.  # noqa: E501
        :rtype: str
        """
        return self._job_id

    @job_id.setter
    def job_id(self, job_id):
        """Sets the job_id of this JobLocalEvent.


        :param job_id: The job_id of this JobLocalEvent.  # noqa: E501
        :type: str
        """

        self._job_id = job_id

    @property
    def shard_index(self):
        """Gets the shard_index of this JobLocalEvent.  # noqa: E501


        :return: The shard_index of this JobLocalEvent.  # noqa: E501
        :rtype: int
        """
        return self._shard_index

    @shard_index.setter
    def shard_index(self, shard_index):
        """Sets the shard_index of this JobLocalEvent.


        :param shard_index: The shard_index of this JobLocalEvent.  # noqa: E501
        :type: int
        """

        self._shard_index = shard_index

    @property
    def target_node_id(self):
        """Gets the target_node_id of this JobLocalEvent.  # noqa: E501


        :return: The target_node_id of this JobLocalEvent.  # noqa: E501
        :rtype: str
        """
        return self._target_node_id

    @target_node_id.setter
    def target_node_id(self, target_node_id):
        """Sets the target_node_id of this JobLocalEvent.


        :param target_node_id: The target_node_id of this JobLocalEvent.  # noqa: E501
        :type: str
        """

        self._target_node_id = target_node_id

    def to_dict(self):
        """Returns the model properties as a dict"""
        result = {}

        for attr, _ in six.iteritems(self.swagger_types):
            value = getattr(self, attr)
            if isinstance(value, list):
                result[attr] = list(map(
                    lambda x: x.to_dict() if hasattr(x, "to_dict") else x,
                    value
                ))
            elif hasattr(value, "to_dict"):
                result[attr] = value.to_dict()
            elif isinstance(value, dict):
                result[attr] = dict(map(
                    lambda item: (item[0], item[1].to_dict())
                    if hasattr(item[1], "to_dict") else item,
                    value.items()
                ))
            else:
                result[attr] = value
        if issubclass(JobLocalEvent, dict):
            for key, value in self.items():
                result[key] = value

        return result

    def to_str(self):
        """Returns the string representation of the model"""
        return pprint.pformat(self.to_dict())

    def __repr__(self):
        """For `print` and `pprint`"""
        return self.to_str()

    def __eq__(self, other):
        """Returns true if both objects are equal"""
        if not isinstance(other, JobLocalEvent):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, JobLocalEvent):
            return True

        return self.to_dict() != other.to_dict()
