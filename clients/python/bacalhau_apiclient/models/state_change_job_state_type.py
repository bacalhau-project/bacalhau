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

class StateChangeJobStateType(object):
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
        'new': 'JobStateType',
        'previous': 'JobStateType'
    }

    attribute_map = {
        'new': 'New',
        'previous': 'Previous'
    }

    def __init__(self, new=None, previous=None):  # noqa: E501
        """StateChangeJobStateType - a model defined in Swagger"""  # noqa: E501
        self._new = None
        self._previous = None
        self.discriminator = None
        if new is not None:
            self.new = new
        if previous is not None:
            self.previous = previous

    @property
    def new(self):
        """Gets the new of this StateChangeJobStateType.  # noqa: E501


        :return: The new of this StateChangeJobStateType.  # noqa: E501
        :rtype: JobStateType
        """
        return self._new

    @new.setter
    def new(self, new):
        """Sets the new of this StateChangeJobStateType.


        :param new: The new of this StateChangeJobStateType.  # noqa: E501
        :type: JobStateType
        """

        self._new = new

    @property
    def previous(self):
        """Gets the previous of this StateChangeJobStateType.  # noqa: E501


        :return: The previous of this StateChangeJobStateType.  # noqa: E501
        :rtype: JobStateType
        """
        return self._previous

    @previous.setter
    def previous(self, previous):
        """Sets the previous of this StateChangeJobStateType.


        :param previous: The previous of this StateChangeJobStateType.  # noqa: E501
        :type: JobStateType
        """

        self._previous = previous

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
        if issubclass(StateChangeJobStateType, dict):
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
        if not isinstance(other, StateChangeJobStateType):
            return False

        return self.__dict__ == other.__dict__

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        return not self == other
