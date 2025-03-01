module github.com/svenvdam/linea/connectors/aws

go 1.19

require (
	github.com/aws/aws-sdk-go-v2/service/sqs v1.29.7
	github.com/stretchr/testify v1.10.0
	github.com/svenvdam/linea v0.2.0
)

require (
	github.com/aws/aws-sdk-go-v2 v1.25.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.3 // indirect
	github.com/aws/smithy-go v1.20.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/svenvdam/linea => ../../
