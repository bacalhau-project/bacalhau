"""Main module."""


from bacalhau_apiclient.api import job_api
from bacalhau_apiclient.api import utils_api
from bacalhau_apiclient.models.version_request import VersionRequest


client = job_api.ApiClient()
utils_instance = utils_api.UtilsApi(client)
print(utils_instance.version(VersionRequest(client_id="test")))



