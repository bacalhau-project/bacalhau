# coding: utf-8

"""
    Bacalhau API

    This page is the reference of the Bacalhau REST API. Project docs are available at https://docs.bacalhau.org/. Find more information about Bacalhau at https://github.com/bacalhau-project/bacalhau.  # noqa: E501

    OpenAPI spec version: ${VERSION}
    Contact: team@bacalhau.org
    Generated by: https://github.com/swagger-api/swagger-codegen.git
"""

from __future__ import absolute_import

import unittest

import bacalhau_apiclient
from bacalhau_apiclient.api.job_api import JobApi  # noqa: E501
from bacalhau_apiclient.rest import ApiException


class TestJobApi(unittest.TestCase):
    """JobApi unit test stubs"""

    def setUp(self):
        self.api = JobApi()  # noqa: E501

    def tearDown(self):
        pass

    def test_cancel(self):
        """Test case for cancel

        Cancels the job with the job-id specified in the body payload.  # noqa: E501
        """
        pass

    def test_events(self):
        """Test case for events

        Returns the events related to the job-id passed in the body payload. Useful for troubleshooting.  # noqa: E501
        """
        pass

    def test_list(self):
        """Test case for list

        Simply lists jobs.  # noqa: E501
        """
        pass

    def test_logs(self):
        """Test case for logs

        Displays the logs for a current job/execution  # noqa: E501
        """
        pass

    def test_results(self):
        """Test case for results

        Returns the results of the job-id specified in the body payload.  # noqa: E501
        """
        pass

    def test_states(self):
        """Test case for states

        Returns the state of the job-id specified in the body payload.  # noqa: E501
        """
        pass

    def test_submit(self):
        """Test case for submit

        Submits a new job to the network.  # noqa: E501
        """
        pass


if __name__ == '__main__':
    unittest.main()
