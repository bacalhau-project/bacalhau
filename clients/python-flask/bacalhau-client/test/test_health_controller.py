# coding: utf-8

from __future__ import absolute_import

from flask import json
from six import BytesIO

from bacalhau-client.models.publicapi_debug_response import PublicapiDebugResponse  # noqa: E501
from bacalhau-client.models.types_health_info import TypesHealthInfo  # noqa: E501
from bacalhau-client.test import BaseTestCase


class TestHealthController(BaseTestCase):
    """HealthController integration test stubs"""

    def test_api_serverdebug(self):
        """Test case for api_serverdebug

        Returns debug information on what the current node is doing.
        """
        response = self.client.open(
            '//debug',
            method='GET')
        self.assert200(response,
                       'Response body is : ' + response.data.decode('utf-8'))

    def test_api_serverhealthz(self):
        """Test case for api_serverhealthz

        
        """
        response = self.client.open(
            '//healthz',
            method='GET')
        self.assert200(response,
                       'Response body is : ' + response.data.decode('utf-8'))

    def test_api_serverlivez(self):
        """Test case for api_serverlivez

        
        """
        response = self.client.open(
            '//livez',
            method='GET')
        self.assert200(response,
                       'Response body is : ' + response.data.decode('utf-8'))

    def test_api_serverlogz(self):
        """Test case for api_serverlogz

        
        """
        response = self.client.open(
            '//logz',
            method='GET')
        self.assert200(response,
                       'Response body is : ' + response.data.decode('utf-8'))

    def test_api_serverreadyz(self):
        """Test case for api_serverreadyz

        
        """
        response = self.client.open(
            '//readyz',
            method='GET')
        self.assert200(response,
                       'Response body is : ' + response.data.decode('utf-8'))

    def test_api_servervarz(self):
        """Test case for api_servervarz

        
        """
        response = self.client.open(
            '//varz',
            method='GET')
        self.assert200(response,
                       'Response body is : ' + response.data.decode('utf-8'))


if __name__ == '__main__':
    import unittest
    unittest.main()
