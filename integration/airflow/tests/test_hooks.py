#!/usr/bin/env python

"""Tests for `bacalhau_airflow.hooks` package."""


import unittest

from bacalhau_airflow.hooks import BacalhauHook


class TestBacalhauHook(unittest.TestCase):
    # """Tests for `bacalhau_airflow.hooks` package."""

    def setUp(self):
        """Set up test fixtures, if any."""

    def tearDown(self):
        """Tear down test fixtures, if any."""

    def test_submit_job(self):
        """Test submit_job."""
        api_version = "V1beta1"
        spec = dict(
            engine="Docker",
            verifier="Noop",
            publisher="Estuary",
            docker=dict(
                image="ubuntu",
                entrypoint=["echo", "TestBacalhauAirflowOperator"],
            ),
            deal=dict(concurrency=1, confidence=0, min_bids=0),
        )
        hook = BacalhauHook()
        job_id = hook.submit_job(api_version=api_version, job_spec=spec)
        self.assertIsNotNone(job_id)
        self.assertIsInstance(job_id, str)
        return job_id

    def test_get_results(self):
        """Test get_results."""
        job_id = self.test_submit_job()
        hook = BacalhauHook()
        results = hook.get_results(job_id)
        self.assertIsNotNone(results)
        self.assertIsInstance(results, list)
        # self.assertGreaterEqual(len(results), 1)

    def test_get_events(self):
        """Test get_events."""
        job_id = self.test_submit_job()
        hook = BacalhauHook()
        events = hook.get_events(job_id)
        self.assertIsNotNone(events)
        self.assertIsInstance(events, dict)
        # self.assertGreaterEqual(len(events), 1)
