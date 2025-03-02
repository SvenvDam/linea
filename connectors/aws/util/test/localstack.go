package test

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
)

// SetupLocalstack creates a localstack container and returns an AWS config pointing to it
func SetupLocalstack(ctx context.Context) (*aws.Config, testcontainers.Container, error) {
	container, err := localstack.Run(
		ctx,
		"localstack/localstack:3.7",
		testcontainers.WithEnv(map[string]string{}),
	)
	if err != nil {
		return nil, nil, err
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port("4566/tcp"))
	if err != nil {
		return nil, nil, err
	}

	provider, err := testcontainers.NewDockerProvider()
	if err != nil {
		return nil, nil, err
	}
	defer provider.Close()

	host, err := provider.DaemonHost(ctx)
	if err != nil {
		return nil, nil, err
	}

	endpointURL := fmt.Sprintf("http://%s:%d", host, mappedPort.Int())

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithBaseEndpoint(endpointURL),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("112233445566", "112233445566", ""),
		),
		awsconfig.WithRegion("us-east-1"),
	)
	if err != nil {
		return nil, nil, err
	}

	return &awsCfg, container, nil
}
