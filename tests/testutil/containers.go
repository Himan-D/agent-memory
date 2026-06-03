package testutil

import (
	"context"
	"fmt"
	
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/neo4j"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	tcwait "github.com/testcontainers/testcontainers-go/wait"
)

type TestInfrastructure struct {
	Neo4jContainer testcontainers.Container
	QdrantContainer testcontainers.Container
	RedisContainer  *redis.RedisContainer
	Neo4jURI       string
	Neo4jUser      string
	Neo4jPassword  string
	QdrantURL      string
	RedisURL       string
}

func SetupInfrastructure(ctx context.Context) (*TestInfrastructure, error) {
	infra := &TestInfrastructure{}

	neo4jC, err := neo4j.RunContainer(ctx,
		testcontainers.WithImage("neo4j:5"),
		neo4j.WithAdminPassword("testpassword"),
		neo4j.WithLabsPlugin(neo4j.Apoc),
	)
	if err != nil {
		return nil, fmt.Errorf("neo4j container: %w", err)
	}
	infra.Neo4jContainer = neo4jC
	infra.Neo4jURI, err = neo4jC.BoltUrl(ctx)
	if err != nil {
		infra.Teardown(ctx)
		return nil, fmt.Errorf("neo4j bolt url: %w", err)
	}
	infra.Neo4jUser = "neo4j"
	infra.Neo4jPassword = "testpassword"

	qdrantReq := testcontainers.ContainerRequest{
		Image:        "qdrant/qdrant:v1.7.4",
		ExposedPorts: []string{"6333/tcp"},
		WaitingFor:   tcwait.ForLog("application started").WithStartupTimeout(30 * time.Second),
	}
	qdrantC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: qdrantReq,
		Started:         true,
	})
	if err != nil {
		infra.Teardown(ctx)
		return nil, fmt.Errorf("qdrant container: %w", err)
	}
	infra.QdrantContainer = qdrantC
	qdrantHost, err := qdrantC.Host(ctx)
	if err != nil {
		infra.Teardown(ctx)
		return nil, fmt.Errorf("qdrant host: %w", err)
	}
	qdrantPort, err := qdrantC.MappedPort(ctx, "6333")
	if err != nil {
		infra.Teardown(ctx)
		return nil, fmt.Errorf("qdrant port: %w", err)
	}
	infra.QdrantURL = fmt.Sprintf("http://%s:%s", qdrantHost, qdrantPort)

	redisC, err := redis.RunContainer(ctx)
	if err != nil {
		infra.Teardown(ctx)
		return nil, fmt.Errorf("redis container: %w", err)
	}
	infra.RedisContainer = redisC
	infra.RedisURL, err = redisC.ConnectionString(ctx)
	if err != nil {
		infra.Teardown(ctx)
		return nil, fmt.Errorf("redis connection: %w", err)
	}

	return infra, nil
}

func (ti *TestInfrastructure) Teardown(ctx context.Context) error {
	var errs []error
	if ti.Neo4jContainer != nil {
		if err := ti.Neo4jContainer.Terminate(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if ti.QdrantContainer != nil {
		if err := ti.QdrantContainer.Terminate(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if ti.RedisContainer != nil {
		if err := ti.RedisContainer.Terminate(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("teardown errors: %v", errs)
	}
	return nil
}