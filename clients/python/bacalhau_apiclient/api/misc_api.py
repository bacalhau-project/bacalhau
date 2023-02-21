# coding: utf-8

"""
    Bacalhau API

    This page is the reference of the Bacalhau REST API. Project docs are available at https://docs.bacalhau.org/. Find more information about Bacalhau at https://github.com/bacalhau-project/bacalhau.  # noqa: E501

    OpenAPI spec version: 0.3.18.post4
    Contact: team@bacalhau.org
    Generated by: https://github.com/swagger-api/swagger-codegen.git
"""


from __future__ import absolute_import

import re  # noqa: F401

# python 2 and python 3 compatibility library
import six

from bacalhau_apiclient.api_client import ApiClient


class MiscApi(object):
    """NOTE: This class is auto generated by the swagger code generator program.

    Do not edit the class manually.
    Ref: https://github.com/swagger-api/swagger-codegen
    """

    def __init__(self, api_client=None):
        if api_client is None:
            api_client = ApiClient()
        self.api_client = api_client

    def api_serverversion(self, version_request, **kwargs):  # noqa: E501
        """Returns the build version running on the server.  # noqa: E501

        See https://github.com/bacalhau-project/bacalhau/releases for a complete list of `gitversion` tags.  # noqa: E501
        This method makes a synchronous HTTP request by default. To make an
        asynchronous HTTP request, please pass async_req=True
        >>> thread = api.api_serverversion(version_request, async_req=True)
        >>> result = thread.get()

        :param async_req bool
        :param VersionRequest version_request: Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field. (required)
        :return: VersionResponse
                 If the method is called asynchronously,
                 returns the request thread.
        """
        kwargs['_return_http_data_only'] = True
        if kwargs.get('async_req'):
            return self.api_serverversion_with_http_info(version_request, **kwargs)  # noqa: E501
        else:
            (data) = self.api_serverversion_with_http_info(version_request, **kwargs)  # noqa: E501
            return data

    def api_serverversion_with_http_info(self, version_request, **kwargs):  # noqa: E501
        """Returns the build version running on the server.  # noqa: E501

        See https://github.com/bacalhau-project/bacalhau/releases for a complete list of `gitversion` tags.  # noqa: E501
        This method makes a synchronous HTTP request by default. To make an
        asynchronous HTTP request, please pass async_req=True
        >>> thread = api.api_serverversion_with_http_info(version_request, async_req=True)
        >>> result = thread.get()

        :param async_req bool
        :param VersionRequest version_request: Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field. (required)
        :return: VersionResponse
                 If the method is called asynchronously,
                 returns the request thread.
        """

        all_params = ['version_request']  # noqa: E501
        all_params.append('async_req')
        all_params.append('_return_http_data_only')
        all_params.append('_preload_content')
        all_params.append('_request_timeout')

        params = locals()
        for key, val in six.iteritems(params['kwargs']):
            if key not in all_params:
                raise TypeError(
                    "Got an unexpected keyword argument '%s'"
                    " to method api_serverversion" % key
                )
            params[key] = val
        del params['kwargs']
        # verify the required parameter 'version_request' is set
        if self.api_client.client_side_validation and ('version_request' not in params or
                                                       params['version_request'] is None):  # noqa: E501
            raise ValueError("Missing the required parameter `version_request` when calling `api_serverversion`")  # noqa: E501

        collection_formats = {}

        path_params = {}

        query_params = []

        header_params = {}

        form_params = []
        local_var_files = {}

        body_params = None
        if 'version_request' in params:
            body_params = params['version_request']
        # HTTP header `Accept`
        header_params['Accept'] = self.api_client.select_header_accept(
            ['application/json'])  # noqa: E501

        # HTTP header `Content-Type`
        header_params['Content-Type'] = self.api_client.select_header_content_type(  # noqa: E501
            ['application/json'])  # noqa: E501

        # Authentication setting
        auth_settings = []  # noqa: E501

        return self.api_client.call_api(
            '/version', 'POST',
            path_params,
            query_params,
            header_params,
            body=body_params,
            post_params=form_params,
            files=local_var_files,
            response_type='VersionResponse',  # noqa: E501
            auth_settings=auth_settings,
            async_req=params.get('async_req'),
            _return_http_data_only=params.get('_return_http_data_only'),
            _preload_content=params.get('_preload_content', True),
            _request_timeout=params.get('_request_timeout'),
            collection_formats=collection_formats)
