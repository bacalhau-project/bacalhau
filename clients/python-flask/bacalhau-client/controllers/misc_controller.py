import connexion
import six

from bacalhau-client.models.publicapi_version_request import PublicapiVersionRequest  # noqa: E501
from bacalhau-client.models.publicapi_version_response import PublicapiVersionResponse  # noqa: E501
from bacalhau-client import util


def api_serverid():  # noqa: E501
    """Returns the id of the host node.

     # noqa: E501


    :rtype: str
    """
    return 'do some magic!'


def api_serverpeers():  # noqa: E501
    """Returns the peers connected to the host via the transport layer.

    As described in the [architecture docs](https://docs.bacalhau.org/about-bacalhau/architecture), each node is connected to a number of peer nodes.  Example response: &#x60;&#x60;&#x60;json {   \&quot;bacalhau-job-event\&quot;: [     \&quot;QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\&quot;,     \&quot;QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\&quot;,     \&quot;QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG\&quot;,     \&quot;QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT\&quot;,     \&quot;QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\&quot;   ] } &#x60;&#x60;&#x60; # noqa: E501


    :rtype: Dict[str, List[str]]
    """
    return 'do some magic!'


def api_serverversion(body):  # noqa: E501
    """Returns the build version running on the server.

    See https://github.com/filecoin-project/bacalhau/releases for a complete list of &#x60;gitversion&#x60; tags. # noqa: E501

    :param body: Request must specify a &#x60;client_id&#x60;. To retrieve your &#x60;client_id&#x60;, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run &#x60;bacalhau describe &lt;job-id&gt;&#x60; and fetch the &#x60;ClientID&#x60; field.
    :type body: dict | bytes

    :rtype: PublicapiVersionResponse
    """
    if connexion.request.is_json:
        body = PublicapiVersionRequest.from_dict(connexion.request.get_json())  # noqa: E501
    return 'do some magic!'
