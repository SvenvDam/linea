# AWS Connector for Linea

This package provides connectors for AWS services to be used with the Linea streaming library.

## Supported Services

### Amazon SQS

The SQS package currently provides:

- **Source**: Read messages from an SQS queue

Additional functionality (sinks and flows) will be added in future updates.

## Getting Started

### Prerequisites

- Go 1.19 or later
- AWS SDK for Go V2
- AWS credentials configured

### Installation

```bash
go get github.com/svenvdam/linea/connectors/aws
```

## Usage

For detailed usage examples and API documentation, please refer to the package's GoDoc
or the source code comments.

The AWS connectors follow the same patterns as the core Linea library, providing
sources, flows, and sinks that can be composed into streaming data pipelines.

## License

Same as the parent Linea project.
