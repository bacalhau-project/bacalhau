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

class LegacyCancelRequest(object):
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
        'client_public_key': 'str',
        'payload': 'AllOflegacyCancelRequestPayload',
        'signature': 'str'
    }

    attribute_map = {
        'client_public_key': 'client_public_key',
        'payload': 'payload',
        'signature': 'signature'
    }

    def __init__(self, client_public_key=None, payload=None, signature=None):  # noqa: E501
        """LegacyCancelRequest - a model defined in Swagger"""  # noqa: E501
        self._client_public_key = None
        self._payload = None
        self._signature = None
        self.discriminator = None
        self.client_public_key = client_public_key
        self.payload = payload
        self.signature = signature

    @property
    def client_public_key(self):
        """Gets the client_public_key of this LegacyCancelRequest.  # noqa: E501

        The base64-encoded public key of the client:  # noqa: E501

        :return: The client_public_key of this LegacyCancelRequest.  # noqa: E501
        :rtype: str
        """
        return self._client_public_key

    @client_public_key.setter
    def client_public_key(self, client_public_key):
        """Sets the client_public_key of this LegacyCancelRequest.

        The base64-encoded public key of the client:  # noqa: E501

        :param client_public_key: The client_public_key of this LegacyCancelRequest.  # noqa: E501
        :type: str
        """
        if client_public_key is None:
            raise ValueError("Invalid value for `client_public_key`, must not be `None`")  # noqa: E501

        self._client_public_key = client_public_key

    @property
    def payload(self):
        """Gets the payload of this LegacyCancelRequest.  # noqa: E501

        The data needed to cancel a running job on the network  # noqa: E501

        :return: The payload of this LegacyCancelRequest.  # noqa: E501
        :rtype: AllOflegacyCancelRequestPayload
        """
        return self._payload

    @payload.setter
    def payload(self, payload):
        """Sets the payload of this LegacyCancelRequest.

        The data needed to cancel a running job on the network  # noqa: E501

        :param payload: The payload of this LegacyCancelRequest.  # noqa: E501
        :type: AllOflegacyCancelRequestPayload
        """
        if payload is None:
            raise ValueError("Invalid value for `payload`, must not be `None`")  # noqa: E501

        self._payload = payload

    @property
    def signature(self):
        """Gets the signature of this LegacyCancelRequest.  # noqa: E501

        A base64-encoded signature of the data, signed by the client:  # noqa: E501

        :return: The signature of this LegacyCancelRequest.  # noqa: E501
        :rtype: str
        """
        return self._signature

    @signature.setter
    def signature(self, signature):
        """Sets the signature of this LegacyCancelRequest.

        A base64-encoded signature of the data, signed by the client:  # noqa: E501

        :param signature: The signature of this LegacyCancelRequest.  # noqa: E501
        :type: str
        """
        if signature is None:
            raise ValueError("Invalid value for `signature`, must not be `None`")  # noqa: E501

        self._signature = signature

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
        if issubclass(LegacyCancelRequest, dict):
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
        if not isinstance(other, LegacyCancelRequest):
            return False

        return self.__dict__ == other.__dict__

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        return not self == other
