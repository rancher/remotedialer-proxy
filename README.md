# remotedialer-proxy

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A proxy for the Rancher remotedialer.

## Overview
DUMMY CHANGE
This application serves as a proxy for the Rancher remotedialer, facilitating communication between the Rancher server and the managed clusters. It is configured entirely through environment variables.

## Usage

To run the proxy, you must set the following environment variables:

| Variable          | Description                                       | Required |
| ----------------- | ------------------------------------------------- | -------- |
| `TLS_NAME`        | The client name (SAN) for the certificate.        | Yes      |
| `CA_NAME`         | The name of the certificate authority secret.     | Yes      |
| `CERT_CA_NAMESPACE` | The namespace of the certificate secret.          | Yes      |
| `CERT_CA_NAME`    | The name of the certificate secret.               | Yes      |
| `SECRET`          | The remotedialer secret.                          | Yes      |
| `PROXY_PORT`      | The TCP port for the remotedialer-proxy.          | Yes      |
| `PEER_PORT`       | The cluster-external service port.                | Yes      |
| `HTTPS_PORT`      | The HTTPS port for the remotedialer-proxy.        | Yes      |
| `DEBUG`           | Set to enable debug logging.                      | No       |

Once the environment variables are set, you can run the application:

```bash
go run ./cmd/proxy
```

## Building

To build the application from source, run the following command:

```bash
go build ./cmd/proxy
```

## Contributing

Please see the `CODEOWNERS` file for information on who to contact for contributions.
