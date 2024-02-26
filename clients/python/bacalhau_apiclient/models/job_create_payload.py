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

class JobCreatePayload(object):
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
        'api_version': 'str',
        'client_id': 'str',
        'spec': 'AllOfJobCreatePayloadSpec'
    }

    attribute_map = {
        'api_version': 'APIVersion',
        'client_id': 'ClientID',
        'spec': 'Spec'
    }

    def __init__(self, api_version=None, client_id=None, spec=None):  # noqa: E501
        """JobCreatePayload - a model defined in Swagger"""  # noqa: E501
        self._api_version = None
        self._client_id = None
        self._spec = None
        self.discriminator = None
        self.api_version = api_version
        self.client_id = client_id
        self.spec = spec

    @property
    def api_version(self):
        """Gets the api_version of this JobCreatePayload.  # noqa: E501


        :return: The api_version of this JobCreatePayload.  # noqa: E501
        :rtype: str
        """
        return self._api_version

    @api_version.setter
    def api_version(self, api_version):
        """Sets the api_version of this JobCreatePayload.


        :param api_version: The api_version of this JobCreatePayload.  # noqa: E501
        :type: str
        """
        if api_version is None:
            raise ValueError("Invalid value for `api_version`, must not be `None`")  # noqa: E501

        self._api_version = api_version

    @property
    def client_id(self):
        """Gets the client_id of this JobCreatePayload.  # noqa: E501

        the id of the client that is submitting the job  # noqa: E501

        :return: The client_id of this JobCreatePayload.  # noqa: E501
        :rtype: str
        """
        return self._client_id

    @client_id.setter
    def client_id(self, client_id):
        """Sets the client_id of this JobCreatePayload.

        the id of the client that is submitting the job  # noqa: E501

        :param client_id: The client_id of this JobCreatePayload.  # noqa: E501
        :type: str
        """
        if client_id is None:
            raise ValueError("Invalid value for `client_id`, must not be `None`")  # noqa: E501

        self._client_id = client_id

    @property
    def spec(self):
        """Gets the spec of this JobCreatePayload.  # noqa: E501

        The specification of this job.  # noqa: E501

        :return: The spec of this JobCreatePayload.  # noqa: E501
        :rtype: AllOfJobCreatePayloadSpec
        """
        return self._spec

    @spec.setter
    def spec(self, spec):
        """Sets the spec of this JobCreatePayload.

        The specification of this job.  # noqa: E501

        :param spec: The spec of this JobCreatePayload.  # noqa: E501
        :type: AllOfJobCreatePayloadSpec
        """
        if spec is None:
            raise ValueError("Invalid value for `spec`, must not be `None`")  # noqa: E501

        self._spec = spec

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
        if issubclass(JobCreatePayload, dict):
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
        if not isinstance(other, JobCreatePayload):
            return False

        return self.__dict__ == other.__dict__

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        return not self == other
