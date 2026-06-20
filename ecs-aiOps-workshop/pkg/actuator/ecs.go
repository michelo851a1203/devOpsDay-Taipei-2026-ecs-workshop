// Package actuator for building promote or rollback ecs
package actuator

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type Action string

var (
	Proceed  Action = "proceed"
	Rollback Action = "rollback"
	Holding  Action = "hold"
)

type Actuator struct {
	Client           *ecs.Client
	Cluster          string
	BlueServiceName  string
	GreenServiceName string
}

func NewActuator(cfg aws.Config, cluster, blueServiceName, greenServiceName string) *Actuator {
	return &Actuator{
		Client:           ecs.NewFromConfig(cfg),
		Cluster:          cluster,
		BlueServiceName:  blueServiceName,
		GreenServiceName: greenServiceName,
	}
}

func (a *Actuator) Execute(ctx context.Context, action Action) error {
	switch action {
	case Proceed:
		return a.proceed(ctx)
	case Rollback:
		return a.rollback(ctx)
	case Holding:
		log.Println("holding - no action here")
		return nil
	default:
		return fmt.Errorf("unknown action")
	}
}

func (a *Actuator) proceed(ctx context.Context) error {
	_, err := a.Client.UpdateService(ctx, &ecs.UpdateServiceInput{
		Cluster:      aws.String(a.Cluster),
		Service:      aws.String(a.GreenServiceName),
		DesiredCount: aws.Int32(3),
	})
	if err != nil {
		return fmt.Errorf("[promote] update green-service failed : %w", err)
	}
	_, err = a.Client.UpdateService(ctx, &ecs.UpdateServiceInput{
		Cluster:      aws.String(a.Cluster),
		Service:      aws.String(a.BlueServiceName),
		DesiredCount: aws.Int32(0),
	})
	if err != nil {
		return fmt.Errorf("[promte] update blue-service failed : %w", err)
	}
	return nil
}

func (a *Actuator) rollback(ctx context.Context) error {
	_, err := a.Client.UpdateService(ctx, &ecs.UpdateServiceInput{
		Cluster:      aws.String(a.Cluster),
		Service:      aws.String(a.BlueServiceName),
		DesiredCount: aws.Int32(2),
	})
	if err != nil {
		return fmt.Errorf("[Rollback] update blue-service failed %w", err)
	}
	_, err = a.Client.UpdateService(ctx, &ecs.UpdateServiceInput{
		Cluster:      aws.String(a.Cluster),
		Service:      aws.String(a.GreenServiceName),
		DesiredCount: aws.Int32(0),
	})
	if err != nil {
		return fmt.Errorf("[Rollback] update green-service failed : %w", err)
	}
	return nil
}
