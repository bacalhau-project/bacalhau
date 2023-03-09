# coding: utf-8

"""
    Bacalhau API

    This page is the reference of the Bacalhau REST API. Project docs are available at https://docs.bacalhau.org/. Find more information about Bacalhau at https://github.com/bacalhau-project/bacalhau.  # noqa: E501

    OpenAPI spec version: 0.3.23.post8
    Contact: team@bacalhau.org
    Generated by: https://github.com/swagger-api/swagger-codegen.git
"""

import pprint
import re  # noqa: F401

import six


class JobHistory(object):
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
        "comment": "str",
        "compute_reference": "str",
        "execution_state": "StateChangeExecutionStateType",
        "job_id": "str",
        "job_state": "StateChangeJobStateType",
        "new_version": "int",
        "node_id": "str",
        "shard_index": "int",
        "shard_state": "StateChangeShardStateType",
        "time": "str",
        "type": "JobHistoryType",
    }

    attribute_map = {
        "comment": "Comment",
        "compute_reference": "ComputeReference",
        "execution_state": "ExecutionState",
        "job_id": "JobID",
        "job_state": "JobState",
        "new_version": "NewVersion",
        "node_id": "NodeID",
        "shard_index": "ShardIndex",
        "shard_state": "ShardState",
        "time": "Time",
        "type": "Type",
    }

    def __init__(
        self,
        comment=None,
        compute_reference=None,
        execution_state=None,
        job_id=None,
        job_state=None,
        new_version=None,
        node_id=None,
        shard_index=None,
        shard_state=None,
        time=None,
        type=None,
    ):  # noqa: E501
        """JobHistory - a model defined in Swagger"""  # noqa: E501
        self._comment = None
        self._compute_reference = None
        self._execution_state = None
        self._job_id = None
        self._job_state = None
        self._new_version = None
        self._node_id = None
        self._shard_index = None
        self._shard_state = None
        self._time = None
        self._type = None
        self.discriminator = None
        if comment is not None:
            self.comment = comment
        if compute_reference is not None:
            self.compute_reference = compute_reference
        if execution_state is not None:
            self.execution_state = execution_state
        if job_id is not None:
            self.job_id = job_id
        if job_state is not None:
            self.job_state = job_state
        if new_version is not None:
            self.new_version = new_version
        if node_id is not None:
            self.node_id = node_id
        if shard_index is not None:
            self.shard_index = shard_index
        if shard_state is not None:
            self.shard_state = shard_state
        if time is not None:
            self.time = time
        if type is not None:
            self.type = type

    @property
    def comment(self):
        """Gets the comment of this JobHistory.  # noqa: E501


        :return: The comment of this JobHistory.  # noqa: E501
        :rtype: str
        """
        return self._comment

    @comment.setter
    def comment(self, comment):
        """Sets the comment of this JobHistory.


        :param comment: The comment of this JobHistory.  # noqa: E501
        :type: str
        """

        self._comment = comment

    @property
    def compute_reference(self):
        """Gets the compute_reference of this JobHistory.  # noqa: E501


        :return: The compute_reference of this JobHistory.  # noqa: E501
        :rtype: str
        """
        return self._compute_reference

    @compute_reference.setter
    def compute_reference(self, compute_reference):
        """Sets the compute_reference of this JobHistory.


        :param compute_reference: The compute_reference of this JobHistory.  # noqa: E501
        :type: str
        """

        self._compute_reference = compute_reference

    @property
    def execution_state(self):
        """Gets the execution_state of this JobHistory.  # noqa: E501


        :return: The execution_state of this JobHistory.  # noqa: E501
        :rtype: StateChangeExecutionStateType
        """
        return self._execution_state

    @execution_state.setter
    def execution_state(self, execution_state):
        """Sets the execution_state of this JobHistory.


        :param execution_state: The execution_state of this JobHistory.  # noqa: E501
        :type: StateChangeExecutionStateType
        """

        self._execution_state = execution_state

    @property
    def job_id(self):
        """Gets the job_id of this JobHistory.  # noqa: E501


        :return: The job_id of this JobHistory.  # noqa: E501
        :rtype: str
        """
        return self._job_id

    @job_id.setter
    def job_id(self, job_id):
        """Sets the job_id of this JobHistory.


        :param job_id: The job_id of this JobHistory.  # noqa: E501
        :type: str
        """

        self._job_id = job_id

    @property
    def job_state(self):
        """Gets the job_state of this JobHistory.  # noqa: E501


        :return: The job_state of this JobHistory.  # noqa: E501
        :rtype: StateChangeJobStateType
        """
        return self._job_state

    @job_state.setter
    def job_state(self, job_state):
        """Sets the job_state of this JobHistory.


        :param job_state: The job_state of this JobHistory.  # noqa: E501
        :type: StateChangeJobStateType
        """

        self._job_state = job_state

    @property
    def new_version(self):
        """Gets the new_version of this JobHistory.  # noqa: E501


        :return: The new_version of this JobHistory.  # noqa: E501
        :rtype: int
        """
        return self._new_version

    @new_version.setter
    def new_version(self, new_version):
        """Sets the new_version of this JobHistory.


        :param new_version: The new_version of this JobHistory.  # noqa: E501
        :type: int
        """

        self._new_version = new_version

    @property
    def node_id(self):
        """Gets the node_id of this JobHistory.  # noqa: E501


        :return: The node_id of this JobHistory.  # noqa: E501
        :rtype: str
        """
        return self._node_id

    @node_id.setter
    def node_id(self, node_id):
        """Sets the node_id of this JobHistory.


        :param node_id: The node_id of this JobHistory.  # noqa: E501
        :type: str
        """

        self._node_id = node_id

    @property
    def shard_index(self):
        """Gets the shard_index of this JobHistory.  # noqa: E501


        :return: The shard_index of this JobHistory.  # noqa: E501
        :rtype: int
        """
        return self._shard_index

    @shard_index.setter
    def shard_index(self, shard_index):
        """Sets the shard_index of this JobHistory.


        :param shard_index: The shard_index of this JobHistory.  # noqa: E501
        :type: int
        """

        self._shard_index = shard_index

    @property
    def shard_state(self):
        """Gets the shard_state of this JobHistory.  # noqa: E501


        :return: The shard_state of this JobHistory.  # noqa: E501
        :rtype: StateChangeShardStateType
        """
        return self._shard_state

    @shard_state.setter
    def shard_state(self, shard_state):
        """Sets the shard_state of this JobHistory.


        :param shard_state: The shard_state of this JobHistory.  # noqa: E501
        :type: StateChangeShardStateType
        """

        self._shard_state = shard_state

    @property
    def time(self):
        """Gets the time of this JobHistory.  # noqa: E501


        :return: The time of this JobHistory.  # noqa: E501
        :rtype: str
        """
        return self._time

    @time.setter
    def time(self, time):
        """Sets the time of this JobHistory.


        :param time: The time of this JobHistory.  # noqa: E501
        :type: str
        """

        self._time = time

    @property
    def type(self):
        """Gets the type of this JobHistory.  # noqa: E501


        :return: The type of this JobHistory.  # noqa: E501
        :rtype: JobHistoryType
        """
        return self._type

    @type.setter
    def type(self, type):
        """Sets the type of this JobHistory.


        :param type: The type of this JobHistory.  # noqa: E501
        :type: JobHistoryType
        """

        self._type = type

    def to_dict(self):
        """Returns the model properties as a dict"""
        result = {}

        for attr, _ in six.iteritems(self.swagger_types):
            value = getattr(self, attr)
            if isinstance(value, list):
                result[attr] = list(
                    map(lambda x: x.to_dict() if hasattr(x, "to_dict") else x, value)
                )
            elif hasattr(value, "to_dict"):
                result[attr] = value.to_dict()
            elif isinstance(value, dict):
                result[attr] = dict(
                    map(
                        lambda item: (item[0], item[1].to_dict())
                        if hasattr(item[1], "to_dict")
                        else item,
                        value.items(),
                    )
                )
            else:
                result[attr] = value
        if issubclass(JobHistory, dict):
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
        if not isinstance(other, JobHistory):
            return False

        return self.__dict__ == other.__dict__

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        return not self == other
