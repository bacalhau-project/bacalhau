# coding: utf-8

"""
    Bacalhau API

    This page is the reference of the Bacalhau REST API. Project docs are available at https://docs.bacalhau.org/. Find more information about Bacalhau at https://github.com/filecoin-project/bacalhau.  # noqa: E501

    OpenAPI spec version: 1.0.0
    Contact: team@bacalhau.org
    Generated by: https://github.com/swagger-api/swagger-codegen.git
"""


import pprint
import re  # noqa: F401

import six

from bacalhau_client.configuration import Configuration


class FreeSpace(object):
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
        'ipfs_mount': 'MountStatus',
        'root': 'MountStatus',
        'tmp': 'MountStatus'
    }

    attribute_map = {
        'ipfs_mount': 'IPFSMount',
        'root': 'root',
        'tmp': 'tmp'
    }

    def __init__(self, ipfs_mount=None, root=None, tmp=None, _configuration=None):  # noqa: E501
        """FreeSpace - a model defined in Swagger"""  # noqa: E501
        if _configuration is None:
            _configuration = Configuration()
        self._configuration = _configuration

        self._ipfs_mount = None
        self._root = None
        self._tmp = None
        self.discriminator = None

        if ipfs_mount is not None:
            self.ipfs_mount = ipfs_mount
        if root is not None:
            self.root = root
        if tmp is not None:
            self.tmp = tmp

    @property
    def ipfs_mount(self):
        """Gets the ipfs_mount of this FreeSpace.  # noqa: E501


        :return: The ipfs_mount of this FreeSpace.  # noqa: E501
        :rtype: MountStatus
        """
        return self._ipfs_mount

    @ipfs_mount.setter
    def ipfs_mount(self, ipfs_mount):
        """Sets the ipfs_mount of this FreeSpace.


        :param ipfs_mount: The ipfs_mount of this FreeSpace.  # noqa: E501
        :type: MountStatus
        """

        self._ipfs_mount = ipfs_mount

    @property
    def root(self):
        """Gets the root of this FreeSpace.  # noqa: E501


        :return: The root of this FreeSpace.  # noqa: E501
        :rtype: MountStatus
        """
        return self._root

    @root.setter
    def root(self, root):
        """Sets the root of this FreeSpace.


        :param root: The root of this FreeSpace.  # noqa: E501
        :type: MountStatus
        """

        self._root = root

    @property
    def tmp(self):
        """Gets the tmp of this FreeSpace.  # noqa: E501


        :return: The tmp of this FreeSpace.  # noqa: E501
        :rtype: MountStatus
        """
        return self._tmp

    @tmp.setter
    def tmp(self, tmp):
        """Sets the tmp of this FreeSpace.


        :param tmp: The tmp of this FreeSpace.  # noqa: E501
        :type: MountStatus
        """

        self._tmp = tmp

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
        if issubclass(FreeSpace, dict):
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
        if not isinstance(other, FreeSpace):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, FreeSpace):
            return True

        return self.to_dict() != other.to_dict()
