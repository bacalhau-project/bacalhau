# go-pinning-service-http-client


[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://protocol.ai)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](https://ipfs.io/)
[![](https://img.shields.io/badge/status-draft-yellow.svg?style=flat-square)](https://github.com/ipfs/specs/#understanding-the-meaning-of-the-spec-badges-and-their-lifecycle)

An IPFS Pinning Service HTTP Client

> This repo is contains a reference implementation of a client for the [IPFS Pinning Services API Spec](https://github.com/ipfs/pinning-services-api-spec)

## Lead Maintainer

[Adin Schmahmann](https://github.com/aschmahmann)

## Updating Pinning Service Spec

Download the openapi-generator from https://github.com/OpenAPITools/openapi-generator and generate the code using:

Current code generated with: openapi-generator 5.0.0-beta

```
openapi-generator generate -g go-experimental -i https://raw.githubusercontent.com/ipfs/pinning-services-api-spec/master/ipfs-pinning-service.yaml -o openapi
rm openapi/go.mod openapi/go.sum
```

Notes:
Due to https://github.com/OpenAPITools/openapi-generator/issues/7473 the code generator the http error codes processing
may need some manual editing.

`go-experimental` is becoming mainstream and so in later versions will be replaced with `go`

## Contributing

Contributions are welcome! This repository is part of the IPFS project and therefore governed by our [contributing guidelines](https://github.com/ipfs/community/blob/master/CONTRIBUTING.md).

## License

[SPDX-License-Identifier: Apache-2.0 OR MIT](LICENSE.md)
