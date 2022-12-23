"""Sign a message with the user's private ID key."""

# def stringify_dict(d: dict) -> str:
#     """Stringify a dict."""
#     s = ""
#     od = collections.OrderedDict(sorted(d.items()))
#     for key, value in od:
#         s += str(value)
#     return s


# def hashable_value(payload) -> bytes:
#     """Return a hashable value for the payload."""
#     print("payload")
#     print(payload)
#     s = str()
#     s += payload['ClientID']
#     s += payload['APIVersion']

#     # Spec
#     s += payload['Spec']['Engine']
#     s += payload['Spec']['Verifier']
#     s += payload['Spec']['Publisher']

# # Spec.Docker
# if 'Docker' in payload['Spec'].keys():
#     if 'Image' in payload['Spec']['Docker'].keys():
#         s += payload['Spec']['Docker']['Image']
#     if 'Entrypoint' in payload['Spec']['Docker'].keys():
#         s += "".join(payload['Spec']['Docker']['Entrypoint'])
#     if 'EntrypointVariables' in payload['Spec']['Docker'].keys():
#         s += "".join(payload['Spec']['Docker']['EntrypointVariables'])
#     if 'WorkingDirectory' in payload['Spec']['Docker'].keys():
#         s += payload['Spec']['Docker']['WorkingDirectory']

# # Spec.Language - not implemented
# # Spec.Wasm - not implemented

# # Spec.Resources
# if 'Resources' in payload['Spec'].keys():
#     if 'GPU' in payload['Spec']['Resources'].keys():
#         s += payload['Spec']['Resources']['GPU']
#     if 'CPU' in payload['Spec']['Resources'].keys():
#         s += payload['Spec']['Resources']['CPU']
#     if 'Memory' in payload['Spec']['Resources'].keys():
#         s += payload['Spec']['Resources']['Memory']
#     if 'Disk' in payload['Spec']['Resources'].keys():
#         s += payload['Spec']['Resources']['Disk']

# # Spec.Timeout
# if 'Timeout' in payload['Spec'].keys():
#     s += "{:f}".format(payload['Spec']['Timeout'])

# # Spec.Inputs
# if 'Inputs' in payload['Spec'].keys():
#     for i in payload['Spec']['Inputs']:
#         s += i['StorageSource']
#         s += i['Name']
#         s += i['path']
#         # TODO @enricorotundo CID, URL, Metadata

# # Spec.Contexts
# # Spec.Outputs
# if 'outputs' in payload['Spec'].keys():
#     for i in payload['Spec']['outputs']:
#         s += i['StorageSource']
#         s += i['Name']
#         # s += i['CID']
#         # s += i['URL']
#         s += i['path']
#         # s += i['metadata']

# # Spec.Annotations
# if 'Annotations' in payload['Spec'].keys():
#     for value in payload['Spec']['Annotations'].items():
#         s += value

# # Spec.Sharding
# if 'Sharding' in payload['Spec'].keys():
#     if 'GlobPattern' in payload['Spec']['Sharding'].keys():
#         s += payload['Spec']['Sharding']['GlobPattern']
#     if 'BatchSize' in payload['Spec']['Sharding'].keys():
#         s += str(payload['Spec']['Sharding']['BatchSize'])
#     if 'GlobPatternBasePath' in payload['Spec']['Sharding'].keys():
#         s += payload['Spec']['Sharding']['GlobPatternBasePath']

# # Spec.DoNotTrack
# if 'DoNotTrack' in payload['Spec'].keys():
#     s += str(payload['Spec']['DoNotTrack']).lower() # small case true/false

# # Spec.ExecutionPlan
# if 'ExecutionPlan' in payload['Spec'].keys():
#     if 'ShardsTotal' in payload['Spec']['ExecutionPlan'].keys():
#         s += str(payload['Spec']['ExecutionPlan']['ShardsTotal'])
#         print("ShardsTotal")
#         print(str(payload['Spec']['ExecutionPlan']['ShardsTotal']))

# # Spec.Deal
# if 'Deal' in payload['Spec'].keys():
#     if 'Concurrency' in payload['Spec']['Deal'].keys():
#         s += str(payload['Spec']['Deal']['Concurrency'])
#     if 'Confidence' in payload['Spec']['Deal'].keys():
#         s += str(payload['Spec']['Deal']['Confidence'])
#     if 'MinBids' in payload['Spec']['Deal'].keys():
#         s += str(payload['Spec']['Deal']['MinBids'])

# # Spec.Context
# if 'Context' in payload['Spec'].keys():
#     s += payload['Spec']['Context']

# print("hashed payload")
# print(s)
# return s.encode()
