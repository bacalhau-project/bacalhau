# coding: utf-8

"""
    Bacalhau API

    This page is the reference of the Bacalhau REST API. Project docs are available at https://docs.bacalhau.org/. Find more information about Bacalhau at https://github.com/bacalhau-project/bacalhau.  # noqa: E501

    OpenAPI spec version: ${VERSION}
    Contact: team@bacalhau.org
    Generated by: https://github.com/swagger-api/swagger-codegen.git
"""

import pprint
import re  # noqa: F401

import six

class ComputeNodeInfo(object):
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
        'available_capacity': 'Resources',
        'enqueued_executions': 'int',
        'execution_engines': 'list[str]',
        'max_capacity': 'Resources',
        'max_job_requirements': 'Resources',
        'publishers': 'list[str]',
        'queue_capacity': 'Resources',
        'running_executions': 'int',
        'storage_sources': 'list[str]'
    }

    attribute_map = {
        'available_capacity': 'AvailableCapacity',
        'enqueued_executions': 'EnqueuedExecutions',
        'execution_engines': 'ExecutionEngines',
        'max_capacity': 'MaxCapacity',
        'max_job_requirements': 'MaxJobRequirements',
        'publishers': 'Publishers',
        'queue_capacity': 'QueueCapacity',
        'running_executions': 'RunningExecutions',
        'storage_sources': 'StorageSources'
    }

    def __init__(self, available_capacity=None, enqueued_executions=None, execution_engines=None, max_capacity=None, max_job_requirements=None, publishers=None, queue_capacity=None, running_executions=None, storage_sources=None):  # noqa: E501
        """ComputeNodeInfo - a model defined in Swagger"""  # noqa: E501
        self._available_capacity = None
        self._enqueued_executions = None
        self._execution_engines = None
        self._max_capacity = None
        self._max_job_requirements = None
        self._publishers = None
        self._queue_capacity = None
        self._running_executions = None
        self._storage_sources = None
        self.discriminator = None
        if available_capacity is not None:
            self.available_capacity = available_capacity
        if enqueued_executions is not None:
            self.enqueued_executions = enqueued_executions
        if execution_engines is not None:
            self.execution_engines = execution_engines
        if max_capacity is not None:
            self.max_capacity = max_capacity
        if max_job_requirements is not None:
            self.max_job_requirements = max_job_requirements
        if publishers is not None:
            self.publishers = publishers
        if queue_capacity is not None:
            self.queue_capacity = queue_capacity
        if running_executions is not None:
            self.running_executions = running_executions
        if storage_sources is not None:
            self.storage_sources = storage_sources

    @property
    def available_capacity(self):
        """Gets the available_capacity of this ComputeNodeInfo.  # noqa: E501


        :return: The available_capacity of this ComputeNodeInfo.  # noqa: E501
        :rtype: Resources
        """
        return self._available_capacity

    @available_capacity.setter
    def available_capacity(self, available_capacity):
        """Sets the available_capacity of this ComputeNodeInfo.


        :param available_capacity: The available_capacity of this ComputeNodeInfo.  # noqa: E501
        :type: Resources
        """

        self._available_capacity = available_capacity

    @property
    def enqueued_executions(self):
        """Gets the enqueued_executions of this ComputeNodeInfo.  # noqa: E501


        :return: The enqueued_executions of this ComputeNodeInfo.  # noqa: E501
        :rtype: int
        """
        return self._enqueued_executions

    @enqueued_executions.setter
    def enqueued_executions(self, enqueued_executions):
        """Sets the enqueued_executions of this ComputeNodeInfo.


        :param enqueued_executions: The enqueued_executions of this ComputeNodeInfo.  # noqa: E501
        :type: int
        """

        self._enqueued_executions = enqueued_executions

    @property
    def execution_engines(self):
        """Gets the execution_engines of this ComputeNodeInfo.  # noqa: E501


        :return: The execution_engines of this ComputeNodeInfo.  # noqa: E501
        :rtype: list[str]
        """
        return self._execution_engines

    @execution_engines.setter
    def execution_engines(self, execution_engines):
        """Sets the execution_engines of this ComputeNodeInfo.


        :param execution_engines: The execution_engines of this ComputeNodeInfo.  # noqa: E501
        :type: list[str]
        """

        self._execution_engines = execution_engines

    @property
    def max_capacity(self):
        """Gets the max_capacity of this ComputeNodeInfo.  # noqa: E501


        :return: The max_capacity of this ComputeNodeInfo.  # noqa: E501
        :rtype: Resources
        """
        return self._max_capacity

    @max_capacity.setter
    def max_capacity(self, max_capacity):
        """Sets the max_capacity of this ComputeNodeInfo.


        :param max_capacity: The max_capacity of this ComputeNodeInfo.  # noqa: E501
        :type: Resources
        """

        self._max_capacity = max_capacity

    @property
    def max_job_requirements(self):
        """Gets the max_job_requirements of this ComputeNodeInfo.  # noqa: E501


        :return: The max_job_requirements of this ComputeNodeInfo.  # noqa: E501
        :rtype: Resources
        """
        return self._max_job_requirements

    @max_job_requirements.setter
    def max_job_requirements(self, max_job_requirements):
        """Sets the max_job_requirements of this ComputeNodeInfo.


        :param max_job_requirements: The max_job_requirements of this ComputeNodeInfo.  # noqa: E501
        :type: Resources
        """

        self._max_job_requirements = max_job_requirements

    @property
    def publishers(self):
        """Gets the publishers of this ComputeNodeInfo.  # noqa: E501


        :return: The publishers of this ComputeNodeInfo.  # noqa: E501
        :rtype: list[str]
        """
        return self._publishers

    @publishers.setter
    def publishers(self, publishers):
        """Sets the publishers of this ComputeNodeInfo.


        :param publishers: The publishers of this ComputeNodeInfo.  # noqa: E501
        :type: list[str]
        """

        self._publishers = publishers

    @property
    def queue_capacity(self):
        """Gets the queue_capacity of this ComputeNodeInfo.  # noqa: E501


        :return: The queue_capacity of this ComputeNodeInfo.  # noqa: E501
        :rtype: Resources
        """
        return self._queue_capacity

    @queue_capacity.setter
    def queue_capacity(self, queue_capacity):
        """Sets the queue_capacity of this ComputeNodeInfo.


        :param queue_capacity: The queue_capacity of this ComputeNodeInfo.  # noqa: E501
        :type: Resources
        """

        self._queue_capacity = queue_capacity

    @property
    def running_executions(self):
        """Gets the running_executions of this ComputeNodeInfo.  # noqa: E501


        :return: The running_executions of this ComputeNodeInfo.  # noqa: E501
        :rtype: int
        """
        return self._running_executions

    @running_executions.setter
    def running_executions(self, running_executions):
        """Sets the running_executions of this ComputeNodeInfo.


        :param running_executions: The running_executions of this ComputeNodeInfo.  # noqa: E501
        :type: int
        """

        self._running_executions = running_executions

    @property
    def storage_sources(self):
        """Gets the storage_sources of this ComputeNodeInfo.  # noqa: E501


        :return: The storage_sources of this ComputeNodeInfo.  # noqa: E501
        :rtype: list[str]
        """
        return self._storage_sources

    @storage_sources.setter
    def storage_sources(self, storage_sources):
        """Sets the storage_sources of this ComputeNodeInfo.


        :param storage_sources: The storage_sources of this ComputeNodeInfo.  # noqa: E501
        :type: list[str]
        """

        self._storage_sources = storage_sources

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
        if issubclass(ComputeNodeInfo, dict):
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
        if not isinstance(other, ComputeNodeInfo):
            return False

        return self.__dict__ == other.__dict__

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        return not self == other