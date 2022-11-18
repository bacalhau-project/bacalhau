# coding: utf-8

from __future__ import absolute_import

from flask import json
from six import BytesIO

from bacalhau-client.models.publicapi_version_request import PublicapiVersionRequest  # noqa: E501
from bacalhau-client.models.publicapi_version_response import PublicapiVersionResponse  # noqa: E501
from bacalhau-client.test import BaseTestCase


class TestMiscController(BaseTestCase):
    """MiscController integration test stubs"""

    def test_api_serverid(self):
        """Test case for api_serverid

        Returns the id of the host node.
        """
        response = self.client.open(
            '//id',
            method='GET')
        self.assert200(response,
                       'Response body is : ' + response.data.decode('utf-8'))

    def test_api_serverpeers(self):
        """Test case for api_serverpeers

        Returns the peers connected to the host via the transport layer.
        """
        response = self.client.open(
            '//peers',
            method='GET')
        self.assert200(response,
                       'Response body is : ' + response.data.decode('utf-8'))

    def test_api_serverversion(self):
        """Test case for api_serverversion

        Returns the build version running on the server.
        """
        body = PublicapiVersionRequest()
        response = self.client.open(
            '//version',
            method='POST',
            data=json.dumps(body),
            content_type='application/json')
        self.assert200(response,
                       'Response body is : ' + response.data.decode('utf-8'))


if __name__ == '__main__':
    import unittest
    unittest.main()
