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

class S3StorageSpec(object):
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
        'bucket': 'str',
        'checksum': 'str',
        'endpoint': 'str',
        'key': 'str',
        'region': 'str',
        'version_id': 'str'
    }

    attribute_map = {
        'bucket': 'Bucket',
        'checksum': 'Checksum',
        'endpoint': 'Endpoint',
        'key': 'Key',
        'region': 'Region',
        'version_id': 'VersionID'
    }

    def __init__(self, bucket=None, checksum=None, endpoint=None, key=None, region=None, version_id=None):  # noqa: E501
        """S3StorageSpec - a model defined in Swagger"""  # noqa: E501
        self._bucket = None
        self._checksum = None
        self._endpoint = None
        self._key = None
        self._region = None
        self._version_id = None
        self.discriminator = None
        if bucket is not None:
            self.bucket = bucket
        if checksum is not None:
            self.checksum = checksum
        if endpoint is not None:
            self.endpoint = endpoint
        if key is not None:
            self.key = key
        if region is not None:
            self.region = region
        if version_id is not None:
            self.version_id = version_id

    @property
    def bucket(self):
        """Gets the bucket of this S3StorageSpec.  # noqa: E501


        :return: The bucket of this S3StorageSpec.  # noqa: E501
        :rtype: str
        """
        return self._bucket

    @bucket.setter
    def bucket(self, bucket):
        """Sets the bucket of this S3StorageSpec.


        :param bucket: The bucket of this S3StorageSpec.  # noqa: E501
        :type: str
        """

        self._bucket = bucket

    @property
    def checksum(self):
        """Gets the checksum of this S3StorageSpec.  # noqa: E501


        :return: The checksum of this S3StorageSpec.  # noqa: E501
        :rtype: str
        """
        return self._checksum

    @checksum.setter
    def checksum(self, checksum):
        """Sets the checksum of this S3StorageSpec.


        :param checksum: The checksum of this S3StorageSpec.  # noqa: E501
        :type: str
        """

        self._checksum = checksum

    @property
    def endpoint(self):
        """Gets the endpoint of this S3StorageSpec.  # noqa: E501


        :return: The endpoint of this S3StorageSpec.  # noqa: E501
        :rtype: str
        """
        return self._endpoint

    @endpoint.setter
    def endpoint(self, endpoint):
        """Sets the endpoint of this S3StorageSpec.


        :param endpoint: The endpoint of this S3StorageSpec.  # noqa: E501
        :type: str
        """

        self._endpoint = endpoint

    @property
    def key(self):
        """Gets the key of this S3StorageSpec.  # noqa: E501


        :return: The key of this S3StorageSpec.  # noqa: E501
        :rtype: str
        """
        return self._key

    @key.setter
    def key(self, key):
        """Sets the key of this S3StorageSpec.


        :param key: The key of this S3StorageSpec.  # noqa: E501
        :type: str
        """

        self._key = key

    @property
    def region(self):
        """Gets the region of this S3StorageSpec.  # noqa: E501


        :return: The region of this S3StorageSpec.  # noqa: E501
        :rtype: str
        """
        return self._region

    @region.setter
    def region(self, region):
        """Sets the region of this S3StorageSpec.


        :param region: The region of this S3StorageSpec.  # noqa: E501
        :type: str
        """

        self._region = region

    @property
    def version_id(self):
        """Gets the version_id of this S3StorageSpec.  # noqa: E501


        :return: The version_id of this S3StorageSpec.  # noqa: E501
        :rtype: str
        """
        return self._version_id

    @version_id.setter
    def version_id(self, version_id):
        """Sets the version_id of this S3StorageSpec.


        :param version_id: The version_id of this S3StorageSpec.  # noqa: E501
        :type: str
        """

        self._version_id = version_id

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
        if issubclass(S3StorageSpec, dict):
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
        if not isinstance(other, S3StorageSpec):
            return False

        return self.__dict__ == other.__dict__

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        return not self == other
