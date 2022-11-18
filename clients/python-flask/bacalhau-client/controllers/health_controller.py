import connexion
import six

from bacalhau-client.models.publicapi_debug_response import PublicapiDebugResponse  # noqa: E501
from bacalhau-client.models.types_health_info import TypesHealthInfo  # noqa: E501
from bacalhau-client import util


def api_serverdebug():  # noqa: E501
    """Returns debug information on what the current node is doing.

     # noqa: E501


    :rtype: PublicapiDebugResponse
    """
    return 'do some magic!'


def api_serverhealthz():  # noqa: E501
    """api_serverhealthz

     # noqa: E501


    :rtype: TypesHealthInfo
    """
    return 'do some magic!'


def api_serverlivez():  # noqa: E501
    """api_serverlivez

     # noqa: E501


    :rtype: str
    """
    return 'do some magic!'


def api_serverlogz():  # noqa: E501
    """api_serverlogz

     # noqa: E501


    :rtype: str
    """
    return 'do some magic!'


def api_serverreadyz():  # noqa: E501
    """api_serverreadyz

     # noqa: E501


    :rtype: str
    """
    return 'do some magic!'


def api_servervarz():  # noqa: E501
    """api_servervarz

     # noqa: E501


    :rtype: List[int]
    """
    return 'do some magic!'
