# coding: utf-8

from __future__ import absolute_import

from flask import json
from six import BytesIO

from bacalhau-client.models.publicapi_events_request import PublicapiEventsRequest  # noqa: E501
from bacalhau-client.models.publicapi_events_response import PublicapiEventsResponse  # noqa: E501
from bacalhau-client.models.publicapi_list_request import PublicapiListRequest  # noqa: E501
from bacalhau-client.models.publicapi_list_response import PublicapiListResponse  # noqa: E501
from bacalhau-client.models.publicapi_local_events_request import PublicapiLocalEventsRequest  # noqa: E501
from bacalhau-client.models.publicapi_local_events_response import PublicapiLocalEventsResponse  # noqa: E501
from bacalhau-client.models.publicapi_results_response import PublicapiResultsResponse  # noqa: E501
from bacalhau-client.models.publicapi_state_request import PublicapiStateRequest  # noqa: E501
from bacalhau-client.models.publicapi_state_response import PublicapiStateResponse  # noqa: E501
from bacalhau-client.models.publicapi_submit_request import PublicapiSubmitRequest  # noqa: E501
from bacalhau-client.models.publicapi_submit_response import PublicapiSubmitResponse  # noqa: E501
from bacalhau-client.test import BaseTestCase


class TestJobController(BaseTestCase):
    """JobController integration test stubs"""

    def test_pkgapi_server_submit(self):
        """Test case for pkgapi_server_submit

        Submits a new job to the network.
        """
        body = PublicapiSubmitRequest()
        response = self.client.open(
            '//submit',
            method='POST',
            data=json.dumps(body),
            content_type='application/json')
        self.assert200(response,
                       'Response body is : ' + response.data.decode('utf-8'))

    def test_pkgpublicapi_list(self):
        """Test case for pkgpublicapi_list

        Simply lists jobs.
        """
        body = PublicapiListRequest()
        response = self.client.open(
            '//list',
            method='POST',
            data=json.dumps(body),
            content_type='application/json')
        self.assert200(response,
                       'Response body is : ' + response.data.decode('utf-8'))

    def test_pkgpublicapievents(self):
        """Test case for pkgpublicapievents

        Returns the events related to the job-id passed in the body payload. Useful for troubleshooting.
        """
        body = PublicapiEventsRequest()
        response = self.client.open(
            '//events',
            method='POST',
            data=json.dumps(body),
            content_type='application/json')
        self.assert200(response,
                       'Response body is : ' + response.data.decode('utf-8'))

    def test_pkgpublicapilocal_events(self):
        """Test case for pkgpublicapilocal_events

        Returns the node's local events related to the job-id passed in the body payload. Useful for troubleshooting.
        """
        body = PublicapiLocalEventsRequest()
        response = self.client.open(
            '//local_events',
            method='POST',
            data=json.dumps(body),
            content_type='application/json')
        self.assert200(response,
                       'Response body is : ' + response.data.decode('utf-8'))

    def test_pkgpublicapiresults(self):
        """Test case for pkgpublicapiresults

        Returns the results of the job-id specified in the body payload.
        """
        body = PublicapiStateRequest()
        response = self.client.open(
            '//results',
            method='POST',
            data=json.dumps(body),
            content_type='application/json')
        self.assert200(response,
                       'Response body is : ' + response.data.decode('utf-8'))

    def test_pkgpublicapistates(self):
        """Test case for pkgpublicapistates

        Returns the state of the job-id specified in the body payload.
        """
        body = PublicapiStateRequest()
        response = self.client.open(
            '//states',
            method='POST',
            data=json.dumps(body),
            content_type='application/json')
        self.assert200(response,
                       'Response body is : ' + response.data.decode('utf-8'))


if __name__ == '__main__':
    import unittest
    unittest.main()
