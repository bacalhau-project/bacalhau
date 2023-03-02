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


class ResourceUsageData(object):
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
    swagger_types = {"cpu": "float", "disk": "int", "gpu": "int", "memory": "int"}

    attribute_map = {"cpu": "CPU", "disk": "Disk", "gpu": "GPU", "memory": "Memory"}

    def __init__(self, cpu=None, disk=None, gpu=None, memory=None):  # noqa: E501
        """ResourceUsageData - a model defined in Swagger"""  # noqa: E501
        self._cpu = None
        self._disk = None
        self._gpu = None
        self._memory = None
        self.discriminator = None
        if cpu is not None:
            self.cpu = cpu
        if disk is not None:
            self.disk = disk
        if gpu is not None:
            self.gpu = gpu
        if memory is not None:
            self.memory = memory

    @property
    def cpu(self):
        """Gets the cpu of this ResourceUsageData.  # noqa: E501

        cpu units  # noqa: E501

        :return: The cpu of this ResourceUsageData.  # noqa: E501
        :rtype: float
        """
        return self._cpu

    @cpu.setter
    def cpu(self, cpu):
        """Sets the cpu of this ResourceUsageData.

        cpu units  # noqa: E501

        :param cpu: The cpu of this ResourceUsageData.  # noqa: E501
        :type: float
        """

        self._cpu = cpu

    @property
    def disk(self):
        """Gets the disk of this ResourceUsageData.  # noqa: E501

        bytes  # noqa: E501

        :return: The disk of this ResourceUsageData.  # noqa: E501
        :rtype: int
        """
        return self._disk

    @disk.setter
    def disk(self, disk):
        """Sets the disk of this ResourceUsageData.

        bytes  # noqa: E501

        :param disk: The disk of this ResourceUsageData.  # noqa: E501
        :type: int
        """

        self._disk = disk

    @property
    def gpu(self):
        """Gets the gpu of this ResourceUsageData.  # noqa: E501


        :return: The gpu of this ResourceUsageData.  # noqa: E501
        :rtype: int
        """
        return self._gpu

    @gpu.setter
    def gpu(self, gpu):
        """Sets the gpu of this ResourceUsageData.


        :param gpu: The gpu of this ResourceUsageData.  # noqa: E501
        :type: int
        """

        self._gpu = gpu

    @property
    def memory(self):
        """Gets the memory of this ResourceUsageData.  # noqa: E501

        bytes  # noqa: E501

        :return: The memory of this ResourceUsageData.  # noqa: E501
        :rtype: int
        """
        return self._memory

    @memory.setter
    def memory(self, memory):
        """Sets the memory of this ResourceUsageData.

        bytes  # noqa: E501

        :param memory: The memory of this ResourceUsageData.  # noqa: E501
        :type: int
        """

        self._memory = memory

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
        if issubclass(ResourceUsageData, dict):
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
        if not isinstance(other, ResourceUsageData):
            return False

        return self.__dict__ == other.__dict__

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        return not self == other
