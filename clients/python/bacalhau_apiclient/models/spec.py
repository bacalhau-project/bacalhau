# coding: utf-8

"""
    Bacalhau API

    This page is the reference of the Bacalhau REST API. Project docs are available at https://docs.bacalhau.org/. Find more information about Bacalhau at https://github.com/filecoin-project/bacalhau.  # noqa: E501

    OpenAPI spec version: 0.3.18.post4
    Contact: team@bacalhau.org
    Generated by: https://github.com/swagger-api/swagger-codegen.git
"""


import pprint
import re  # noqa: F401

import six

from bacalhau_apiclient.configuration import Configuration


class Spec(object):
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
        'annotations': 'list[str]',
        'contexts': 'list[StorageSpec]',
        'deal': 'SpecDeal',
        'do_not_track': 'bool',
        'docker': 'SpecDocker',
        'engine': 'SpecEngine',
        'execution_plan': 'SpecExecutionPlan',
        'language': 'JobSpecLanguage',
        'network': 'SpecNetwork',
        'node_selectors': 'list[LabelSelectorRequirement]',
        'publisher': 'SpecPublisher',
        'resources': 'SpecResources',
        'sharding': 'SpecSharding',
        'timeout': 'float',
        'verifier': 'Verifier',
        'wasm': 'JobSpecWasm',
        'inputs': 'list[StorageSpec]',
        'outputs': 'list[StorageSpec]'
    }

    attribute_map = {
        'annotations': 'Annotations',
        'contexts': 'Contexts',
        'deal': 'Deal',
        'do_not_track': 'DoNotTrack',
        'docker': 'Docker',
        'engine': 'Engine',
        'execution_plan': 'ExecutionPlan',
        'language': 'Language',
        'network': 'Network',
        'node_selectors': 'NodeSelectors',
        'publisher': 'Publisher',
        'resources': 'Resources',
        'sharding': 'Sharding',
        'timeout': 'Timeout',
        'verifier': 'Verifier',
        'wasm': 'Wasm',
        'inputs': 'inputs',
        'outputs': 'outputs'
    }

    def __init__(self, annotations=None, contexts=None, deal=None, do_not_track=None, docker=None, engine=None, execution_plan=None, language=None, network=None, node_selectors=None, publisher=None, resources=None, sharding=None, timeout=None, verifier=None, wasm=None, inputs=None, outputs=None, _configuration=None):  # noqa: E501
        """Spec - a model defined in Swagger"""  # noqa: E501
        if _configuration is None:
            _configuration = Configuration()
        self._configuration = _configuration

        self._annotations = None
        self._contexts = None
        self._deal = None
        self._do_not_track = None
        self._docker = None
        self._engine = None
        self._execution_plan = None
        self._language = None
        self._network = None
        self._node_selectors = None
        self._publisher = None
        self._resources = None
        self._sharding = None
        self._timeout = None
        self._verifier = None
        self._wasm = None
        self._inputs = None
        self._outputs = None
        self.discriminator = None

        if annotations is not None:
            self.annotations = annotations
        if contexts is not None:
            self.contexts = contexts
        if deal is not None:
            self.deal = deal
        if do_not_track is not None:
            self.do_not_track = do_not_track
        if docker is not None:
            self.docker = docker
        if engine is not None:
            self.engine = engine
        if execution_plan is not None:
            self.execution_plan = execution_plan
        if language is not None:
            self.language = language
        if network is not None:
            self.network = network
        if node_selectors is not None:
            self.node_selectors = node_selectors
        if publisher is not None:
            self.publisher = publisher
        if resources is not None:
            self.resources = resources
        if sharding is not None:
            self.sharding = sharding
        if timeout is not None:
            self.timeout = timeout
        if verifier is not None:
            self.verifier = verifier
        if wasm is not None:
            self.wasm = wasm
        if inputs is not None:
            self.inputs = inputs
        if outputs is not None:
            self.outputs = outputs

    @property
    def annotations(self):
        """Gets the annotations of this Spec.  # noqa: E501

        Annotations on the job - could be user or machine assigned  # noqa: E501

        :return: The annotations of this Spec.  # noqa: E501
        :rtype: list[str]
        """
        return self._annotations

    @annotations.setter
    def annotations(self, annotations):
        """Sets the annotations of this Spec.

        Annotations on the job - could be user or machine assigned  # noqa: E501

        :param annotations: The annotations of this Spec.  # noqa: E501
        :type: list[str]
        """

        self._annotations = annotations

    @property
    def contexts(self):
        """Gets the contexts of this Spec.  # noqa: E501

        Input volumes that will not be sharded for example to upload code into a base image every shard will get the full range of context volumes  # noqa: E501

        :return: The contexts of this Spec.  # noqa: E501
        :rtype: list[StorageSpec]
        """
        return self._contexts

    @contexts.setter
    def contexts(self, contexts):
        """Sets the contexts of this Spec.

        Input volumes that will not be sharded for example to upload code into a base image every shard will get the full range of context volumes  # noqa: E501

        :param contexts: The contexts of this Spec.  # noqa: E501
        :type: list[StorageSpec]
        """

        self._contexts = contexts

    @property
    def deal(self):
        """Gets the deal of this Spec.  # noqa: E501


        :return: The deal of this Spec.  # noqa: E501
        :rtype: SpecDeal
        """
        return self._deal

    @deal.setter
    def deal(self, deal):
        """Sets the deal of this Spec.


        :param deal: The deal of this Spec.  # noqa: E501
        :type: SpecDeal
        """

        self._deal = deal

    @property
    def do_not_track(self):
        """Gets the do_not_track of this Spec.  # noqa: E501

        Do not track specified by the client  # noqa: E501

        :return: The do_not_track of this Spec.  # noqa: E501
        :rtype: bool
        """
        return self._do_not_track

    @do_not_track.setter
    def do_not_track(self, do_not_track):
        """Sets the do_not_track of this Spec.

        Do not track specified by the client  # noqa: E501

        :param do_not_track: The do_not_track of this Spec.  # noqa: E501
        :type: bool
        """

        self._do_not_track = do_not_track

    @property
    def docker(self):
        """Gets the docker of this Spec.  # noqa: E501


        :return: The docker of this Spec.  # noqa: E501
        :rtype: SpecDocker
        """
        return self._docker

    @docker.setter
    def docker(self, docker):
        """Sets the docker of this Spec.


        :param docker: The docker of this Spec.  # noqa: E501
        :type: SpecDocker
        """

        self._docker = docker

    @property
    def engine(self):
        """Gets the engine of this Spec.  # noqa: E501


        :return: The engine of this Spec.  # noqa: E501
        :rtype: SpecEngine
        """
        return self._engine

    @engine.setter
    def engine(self, engine):
        """Sets the engine of this Spec.


        :param engine: The engine of this Spec.  # noqa: E501
        :type: SpecEngine
        """

        self._engine = engine

    @property
    def execution_plan(self):
        """Gets the execution_plan of this Spec.  # noqa: E501


        :return: The execution_plan of this Spec.  # noqa: E501
        :rtype: SpecExecutionPlan
        """
        return self._execution_plan

    @execution_plan.setter
    def execution_plan(self, execution_plan):
        """Sets the execution_plan of this Spec.


        :param execution_plan: The execution_plan of this Spec.  # noqa: E501
        :type: SpecExecutionPlan
        """

        self._execution_plan = execution_plan

    @property
    def language(self):
        """Gets the language of this Spec.  # noqa: E501


        :return: The language of this Spec.  # noqa: E501
        :rtype: JobSpecLanguage
        """
        return self._language

    @language.setter
    def language(self, language):
        """Sets the language of this Spec.


        :param language: The language of this Spec.  # noqa: E501
        :type: JobSpecLanguage
        """

        self._language = language

    @property
    def network(self):
        """Gets the network of this Spec.  # noqa: E501


        :return: The network of this Spec.  # noqa: E501
        :rtype: SpecNetwork
        """
        return self._network

    @network.setter
    def network(self, network):
        """Sets the network of this Spec.


        :param network: The network of this Spec.  # noqa: E501
        :type: SpecNetwork
        """

        self._network = network

    @property
    def node_selectors(self):
        """Gets the node_selectors of this Spec.  # noqa: E501

        NodeSelectors is a selector which must be true for the compute node to run this job.  # noqa: E501

        :return: The node_selectors of this Spec.  # noqa: E501
        :rtype: list[LabelSelectorRequirement]
        """
        return self._node_selectors

    @node_selectors.setter
    def node_selectors(self, node_selectors):
        """Sets the node_selectors of this Spec.

        NodeSelectors is a selector which must be true for the compute node to run this job.  # noqa: E501

        :param node_selectors: The node_selectors of this Spec.  # noqa: E501
        :type: list[LabelSelectorRequirement]
        """

        self._node_selectors = node_selectors

    @property
    def publisher(self):
        """Gets the publisher of this Spec.  # noqa: E501


        :return: The publisher of this Spec.  # noqa: E501
        :rtype: SpecPublisher
        """
        return self._publisher

    @publisher.setter
    def publisher(self, publisher):
        """Sets the publisher of this Spec.


        :param publisher: The publisher of this Spec.  # noqa: E501
        :type: SpecPublisher
        """

        self._publisher = publisher

    @property
    def resources(self):
        """Gets the resources of this Spec.  # noqa: E501


        :return: The resources of this Spec.  # noqa: E501
        :rtype: SpecResources
        """
        return self._resources

    @resources.setter
    def resources(self, resources):
        """Sets the resources of this Spec.


        :param resources: The resources of this Spec.  # noqa: E501
        :type: SpecResources
        """

        self._resources = resources

    @property
    def sharding(self):
        """Gets the sharding of this Spec.  # noqa: E501


        :return: The sharding of this Spec.  # noqa: E501
        :rtype: SpecSharding
        """
        return self._sharding

    @sharding.setter
    def sharding(self, sharding):
        """Sets the sharding of this Spec.


        :param sharding: The sharding of this Spec.  # noqa: E501
        :type: SpecSharding
        """

        self._sharding = sharding

    @property
    def timeout(self):
        """Gets the timeout of this Spec.  # noqa: E501

        How long a job can run in seconds before it is killed. This includes the time required to run, verify and publish results  # noqa: E501

        :return: The timeout of this Spec.  # noqa: E501
        :rtype: float
        """
        return self._timeout

    @timeout.setter
    def timeout(self, timeout):
        """Sets the timeout of this Spec.

        How long a job can run in seconds before it is killed. This includes the time required to run, verify and publish results  # noqa: E501

        :param timeout: The timeout of this Spec.  # noqa: E501
        :type: float
        """

        self._timeout = timeout

    @property
    def verifier(self):
        """Gets the verifier of this Spec.  # noqa: E501


        :return: The verifier of this Spec.  # noqa: E501
        :rtype: Verifier
        """
        return self._verifier

    @verifier.setter
    def verifier(self, verifier):
        """Sets the verifier of this Spec.


        :param verifier: The verifier of this Spec.  # noqa: E501
        :type: Verifier
        """

        self._verifier = verifier

    @property
    def wasm(self):
        """Gets the wasm of this Spec.  # noqa: E501


        :return: The wasm of this Spec.  # noqa: E501
        :rtype: JobSpecWasm
        """
        return self._wasm

    @wasm.setter
    def wasm(self, wasm):
        """Sets the wasm of this Spec.


        :param wasm: The wasm of this Spec.  # noqa: E501
        :type: JobSpecWasm
        """

        self._wasm = wasm

    @property
    def inputs(self):
        """Gets the inputs of this Spec.  # noqa: E501

        the data volumes we will read in the job for example \"read this ipfs cid\" TODO: #667 Replace with \"Inputs\", \"Outputs\" (note the caps) for yaml/json when we update the n.js file  # noqa: E501

        :return: The inputs of this Spec.  # noqa: E501
        :rtype: list[StorageSpec]
        """
        return self._inputs

    @inputs.setter
    def inputs(self, inputs):
        """Sets the inputs of this Spec.

        the data volumes we will read in the job for example \"read this ipfs cid\" TODO: #667 Replace with \"Inputs\", \"Outputs\" (note the caps) for yaml/json when we update the n.js file  # noqa: E501

        :param inputs: The inputs of this Spec.  # noqa: E501
        :type: list[StorageSpec]
        """

        self._inputs = inputs

    @property
    def outputs(self):
        """Gets the outputs of this Spec.  # noqa: E501

        the data volumes we will write in the job for example \"write the results to ipfs\"  # noqa: E501

        :return: The outputs of this Spec.  # noqa: E501
        :rtype: list[StorageSpec]
        """
        return self._outputs

    @outputs.setter
    def outputs(self, outputs):
        """Sets the outputs of this Spec.

        the data volumes we will write in the job for example \"write the results to ipfs\"  # noqa: E501

        :param outputs: The outputs of this Spec.  # noqa: E501
        :type: list[StorageSpec]
        """

        self._outputs = outputs

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
        if issubclass(Spec, dict):
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
        if not isinstance(other, Spec):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, Spec):
            return True

        return self.to_dict() != other.to_dict()
