package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type Action string

const (
	Promote  Action = "promote"
	Rollback Action = "rollback"
	Hold     Action = "hold"
)

type ECSActuator struct {
	Client       *ecs.Client
	Cluster      string
	BlueService  string
	GreenService string
}

func NewECSActuator(cfg aws.Config, cluster, blueService, greenService string) *ECSActuator {
	return &ECSActuator{
		Client:       ecs.NewFromConfig(cfg),
		Cluster:      cluster,
		BlueService:  blueService,
		GreenService: greenService,
	}
}

func (a *ECSActuator) Execute(ctx context.Context, action Action) error {
	switch action {
	case Promote:
		log.Printf("\033[032mcurrent action : %s\033[0m\n", action)
		return a.promote(ctx)
	case Rollback:
		log.Printf("\033[031mcurrent action : %s\033[0m\n", action)
		return a.rollback(ctx)
	case Hold:
		log.Println("hold - no action")
		return nil

	default:
		return fmt.Errorf("unknown error")
	}
}

func (a *ECSActuator) promote(ctx context.Context) error {
	_, err := a.Client.UpdateService(ctx, &ecs.UpdateServiceInput{
		Cluster:      aws.String(a.Cluster),
		Service:      aws.String(a.GreenService),
		DesiredCount: aws.Int32(3),
	})
	if err != nil {
		return err
	}
	_, err = a.Client.UpdateService(ctx, &ecs.UpdateServiceInput{
		Cluster:      aws.String(a.Cluster),
		Service:      aws.String(a.BlueService),
		DesiredCount: aws.Int32(0),
	})

	if err != nil {
		return err
	}
	return nil
}

func (a *ECSActuator) rollback(ctx context.Context) error {
	_, err := a.Client.UpdateService(ctx, &ecs.UpdateServiceInput{
		Cluster:      aws.String(a.Cluster),
		Service:      aws.String(a.GreenService),
		DesiredCount: aws.Int32(0),
	})
	if err != nil {
		return err
	}
	return nil
}

func main() {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("ap-east-2"), // 這個可以換成自己喜歡的 region
		config.WithSharedConfigProfile("<這個用自己的 sso profile>"),
	)

	if err != nil {
		log.Fatalf("Load aws config error : %v\n", err)
	}

	actuator := NewECSActuator(
		cfg,
		"<cluster 名稱>",         // cluster
		"<blue service name>",  // blue-service name
		"<green service name>", // green-service name
	)

	currentAction := Rollback
	err = actuator.Execute(ctx, currentAction)
	if err != nil {
		log.Fatalf("%s error : %v\n", currentAction, err)
	}
}
