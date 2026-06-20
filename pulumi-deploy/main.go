package main

import (
	"encoding/json"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/alb"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lb"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/servicediscovery"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type PortMapping struct {
	Name          string `json:"name"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol"`
	AppProtocol   string `json:"appProtocol"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type LogConfiguration struct {
	LogDriver string            `json:"logDriver"`
	Options   map[string]string `json:"options"`
}

type ContainerDef struct {
	Name             string            `json:"name"`
	Image            string            `json:"image"`
	PortMappings     []PortMapping     `json:"portMappings"`
	Essential        bool              `json:"essential"`
	Environment      []EnvVar          `json:"environment"`
	LogConfiguration *LogConfiguration `json:"logConfiguration"`
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		// ==============================
		vpcID := "<這裡用預設的 vpc>"      // vpc-034121b108f7b40f5
		subnetA := "<subnet 用自己的A>"  // subnet-aaaaaaaaa
		subnetB := "<subnet 用自己的B>"  // subnet-bbbbbbbbb
		subnetC := "<subnet 用自己的C>"  // // subnet-cccccccc
		ecrURI := "<用自己的 ECRURI>"    // <account_id>.dkr.ecr.ap-east-2.amazonaws.com/<repo_namespace>:latest
		regionName := "用自己習慣的region" // ap-east-2
		// ==============================
		cpuArchitecture := "ARM64"
		blueServiceName := "api-blue-service-9ea7cd8ca8e752b2"   // 這個名稱我隨便給的不要在意
		greenServiceName := "api-green-service-2f43e8d51a80e99b" // 這個名稱我隨便給的不要在意
		// ==============================
		sg, err := ec2.NewSecurityGroup(ctx, "demo-sg", &ec2.SecurityGroupArgs{
			Name:        pulumi.String("demo-sg"),
			Description: pulumi.String("demo-sg description"),
			VpcId:       pulumi.String(vpcID),
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					FromPort:   pulumi.Int(80),
					ToPort:     pulumi.Int(80),
					Protocol:   pulumi.String("tcp"),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				&ec2.SecurityGroupIngressArgs{
					FromPort: pulumi.Int(8080),
					ToPort:   pulumi.Int(8080),
					Protocol: pulumi.String("tcp"),
					Self:     pulumi.Bool(true),
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					FromPort:   pulumi.Int(0),
					ToPort:     pulumi.Int(0),
					Protocol:   pulumi.String("-1"),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
		})
		if err != nil {
			return err
		}

		tg, err := alb.NewTargetGroup(ctx, "demo-tg", &alb.TargetGroupArgs{
			Name:       pulumi.String("demo-tg"),
			TargetType: pulumi.String("ip"),
			Protocol:   pulumi.String("HTTP"),
			Port:       pulumi.Int(8080),
			VpcId:      pulumi.String(vpcID),
			HealthCheck: &alb.TargetGroupHealthCheckArgs{
				Path:               pulumi.String("/health"),
				Port:               pulumi.String("8080"),
				HealthyThreshold:   pulumi.Int(2),
				UnhealthyThreshold: pulumi.Int(3),
				Interval:           pulumi.Int(10),
			},
		})
		if err != nil {
			return err
		}

		alb, err := alb.NewLoadBalancer(ctx, "demo-alb", &alb.LoadBalancerArgs{
			Name:             pulumi.String("demo-alb"),
			LoadBalancerType: alb.LoadBalancerTypeApplication,
			Internal:         pulumi.Bool(false),
			SecurityGroups:   pulumi.StringArray{sg.ID()},
			Subnets: pulumi.StringArray{
				pulumi.String(subnetA),
				pulumi.String(subnetB),
				pulumi.String(subnetC),
			},
		})
		if err != nil {
			return err
		}
		_, err = lb.NewListener(ctx, "alb-listener", &lb.ListenerArgs{
			LoadBalancerArn: alb.Arn,
			Port:            pulumi.Int(80),
			Protocol:        pulumi.String("HTTP"),
			DefaultActions: lb.ListenerDefaultActionArray{
				&lb.ListenerDefaultActionArgs{
					Type:           pulumi.String("forward"),
					TargetGroupArn: tg.Arn,
				},
			},
		})
		if err != nil {
			return err
		}

		ns, err := servicediscovery.NewHttpNamespace(ctx, "workshop.local", &servicediscovery.HttpNamespaceArgs{
			Name: pulumi.String("workshop.local"),
		})
		if err != nil {
			return err
		}

		cluster, err := ecs.NewCluster(ctx, "demo-cluster", &ecs.ClusterArgs{
			// Name: pulumi.String("demo-cluster"), // 如果不想要隨機名稱就解開註解 -> 個人覺得隨機名稱比較好一點
			Settings: ecs.ClusterSettingArray{
				&ecs.ClusterSettingArgs{
					Name:  pulumi.String("containerInsights"),
					Value: pulumi.String("enabled"),
				},
			},
		})
		if err != nil {
			return err
		}

		existingRole, err := iam.LookupRole(ctx, &iam.LookupRoleArgs{
			Name: "ecsTaskExecutionRole",
		})
		var executionRoleArn pulumi.StringInput

		if err != nil {
			trustPolicy := `
			{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Effect": "Allow",
						"Principal": {
							"Service": "ecs-tasks.amazonaws.com"
						},
						"Action": "sts:AssumeRole"
					}
				]
			}
			`
			newRole, err := iam.NewRole(ctx, "ecsTaskExecutionRole", &iam.RoleArgs{
				Name:             pulumi.String("ecsTaskExecutionRole"),
				AssumeRolePolicy: pulumi.String(trustPolicy),
			})
			if err != nil {
				return err
			}
			_, err = iam.NewRolePolicyAttachment(ctx, "ecsTaskExecutionRolePolicy", &iam.RolePolicyAttachmentArgs{
				Role:      newRole.Name,
				PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"),
			})
			if err != nil {
				return err
			}
			_, err = iam.NewRolePolicyAttachment(ctx, "ecsCloudWatchFullAccessPolicy", &iam.RolePolicyAttachmentArgs{
				Role:      newRole.Name,
				PolicyArn: pulumi.String("arn:aws:iam::aws:policy/CloudWatchFullAccess"),
			})
			if err != nil {
				return err
			}
			executionRoleArn = newRole.Arn
		} else {
			executionRoleArn = pulumi.String(existingRole.Arn)
		}

		blueDefs := []ContainerDef{
			{
				Name:  "api",
				Image: ecrURI,
				PortMappings: []PortMapping{
					{
						Name:          "api-port",
						ContainerPort: 8080,
						Protocol:      "tcp",
						AppProtocol:   "http",
					},
				},
				Essential: true,
				Environment: []EnvVar{
					{Name: "APP_VERSION", Value: "blue"},
					{Name: "FAIL_RATE", Value: "0"},
				},
				LogConfiguration: &LogConfiguration{
					LogDriver: "awslogs",
					Options: map[string]string{
						"awslogs-group":         "/ecs/sample-log",
						"awslogs-region":        regionName,
						"awslogs-stream-prefix": "blue",
						"awslogs-create-group":  "true",
					},
				},
			},
		}

		blueServiceContainerDefinition, err := json.Marshal(blueDefs)
		if err != nil {
			return err
		}

		blueTaskDef, err := ecs.NewTaskDefinition(ctx, "api-blue", &ecs.TaskDefinitionArgs{
			Family:                  pulumi.String("api-blue"),
			Cpu:                     pulumi.String("512"),
			Memory:                  pulumi.String("1024"),
			NetworkMode:             pulumi.String("awsvpc"), // for fargate
			RequiresCompatibilities: pulumi.StringArray{pulumi.String("FARGATE")},
			ExecutionRoleArn:        executionRoleArn,
			ContainerDefinitions:    pulumi.String(string(blueServiceContainerDefinition)),
			// 這裡要設定唷
			RuntimePlatform: &ecs.TaskDefinitionRuntimePlatformArgs{
				OperatingSystemFamily: pulumi.String("LINUX"),
				CpuArchitecture:       pulumi.String(cpuArchitecture),
			},
		})
		if err != nil {
			return err
		}
		greenDefs := []ContainerDef{
			{
				Name:  "api",
				Image: ecrURI,
				PortMappings: []PortMapping{
					{
						Name:          "api-port",
						ContainerPort: 8080,
						Protocol:      "tcp",
						AppProtocol:   "http",
					},
				},
				Essential: true,
				Environment: []EnvVar{
					{Name: "APP_VERSION", Value: "green"},
					{Name: "FAIL_RATE", Value: "0"},
				},
				LogConfiguration: &LogConfiguration{
					LogDriver: "awslogs",
					Options: map[string]string{
						"awslogs-group":         "/ecs/sample-logs",
						"awslogs-region":        regionName,
						"awslogs-stream-prefix": "green",
						"awslogs-create-group":  "true",
					},
				},
			},
		}

		greenServiceContainerDefinition, err := json.Marshal(greenDefs)
		if err != nil {
			return err
		}

		greenTaskDef, err := ecs.NewTaskDefinition(ctx, "api-green", &ecs.TaskDefinitionArgs{
			Family:                  pulumi.String("api-green"),
			Cpu:                     pulumi.String("512"),
			Memory:                  pulumi.String("1024"),
			NetworkMode:             pulumi.String("awsvpc"), // for fargate
			RequiresCompatibilities: pulumi.StringArray{pulumi.String("FARGATE")},
			ExecutionRoleArn:        executionRoleArn,
			ContainerDefinitions:    pulumi.String(string(greenServiceContainerDefinition)),
			// 這裡要設定唷
			RuntimePlatform: &ecs.TaskDefinitionRuntimePlatformArgs{
				OperatingSystemFamily: pulumi.String("LINUX"),
				CpuArchitecture:       pulumi.String(cpuArchitecture),
			},
		})
		if err != nil {
			return err
		}

		_, err = ecs.NewService(ctx, "api-blue-service", &ecs.ServiceArgs{
			Name:           pulumi.String(blueServiceName),
			Cluster:        cluster.Arn,
			TaskDefinition: blueTaskDef.Arn,
			DesiredCount:   pulumi.Int(2),
			LaunchType:     pulumi.String("FARGATE"),
			NetworkConfiguration: &ecs.ServiceNetworkConfigurationArgs{
				AssignPublicIp: pulumi.Bool(true),
				SecurityGroups: pulumi.StringArray{sg.ID()},
				Subnets: pulumi.StringArray{
					pulumi.String(subnetA),
					pulumi.String(subnetB),
					pulumi.String(subnetC),
				},
			},
			LoadBalancers: ecs.ServiceLoadBalancerArray{
				&ecs.ServiceLoadBalancerArgs{
					TargetGroupArn: tg.Arn,
					ContainerName:  pulumi.String("api"),
					ContainerPort:  pulumi.Int(8080),
				},
			},
			ServiceConnectConfiguration: ecs.ServiceServiceConnectConfigurationArgs{
				Enabled:   pulumi.Bool(true),
				Namespace: ns.Name,
				Services: ecs.ServiceServiceConnectConfigurationServiceArray{
					&ecs.ServiceServiceConnectConfigurationServiceArgs{
						PortName:      pulumi.String("api-port"),
						DiscoveryName: pulumi.String("api-blue"),
						ClientAlias: ecs.ServiceServiceConnectConfigurationServiceClientAliasArray{
							&ecs.ServiceServiceConnectConfigurationServiceClientAliasArgs{
								Port:    pulumi.Int(8080),
								DnsName: pulumi.String("api-blue"),
							},
						},
					},
				},
			},
		})

		if err != nil {
			return err
		}

		_, err = ecs.NewService(ctx, "api-green-service", &ecs.ServiceArgs{
			Name:           pulumi.String(greenServiceName),
			Cluster:        cluster.Arn,
			TaskDefinition: greenTaskDef.Arn,
			DesiredCount:   pulumi.Int(1),
			LaunchType:     pulumi.String("FARGATE"),
			NetworkConfiguration: &ecs.ServiceNetworkConfigurationArgs{
				AssignPublicIp: pulumi.Bool(true),
				SecurityGroups: pulumi.StringArray{sg.ID()},
				Subnets: pulumi.StringArray{
					pulumi.String(subnetA),
					pulumi.String(subnetB),
					pulumi.String(subnetC),
				},
			},
			LoadBalancers: ecs.ServiceLoadBalancerArray{
				&ecs.ServiceLoadBalancerArgs{
					TargetGroupArn: tg.Arn,
					ContainerName:  pulumi.String("api"),
					ContainerPort:  pulumi.Int(8080),
				},
			},
			ServiceConnectConfiguration: ecs.ServiceServiceConnectConfigurationArgs{
				Enabled:   pulumi.Bool(true),
				Namespace: ns.Name,
				Services: ecs.ServiceServiceConnectConfigurationServiceArray{
					&ecs.ServiceServiceConnectConfigurationServiceArgs{
						PortName:      pulumi.String("api-port"),
						DiscoveryName: pulumi.String("api-green"),
						ClientAlias: ecs.ServiceServiceConnectConfigurationServiceClientAliasArray{
							&ecs.ServiceServiceConnectConfigurationServiceClientAliasArgs{
								Port:    pulumi.Int(8080),
								DnsName: pulumi.String("api-green"),
							},
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("albDnsName", alb.DnsName)
		ctx.Export("clusterArn", cluster.Arn)
		ctx.Export("clusterName", cluster.Name)
		ctx.Export("blueServiceName", pulumi.String(blueServiceName))
		ctx.Export("greenServiceName", pulumi.String(greenServiceName))
		ctx.Export("targetGroupArn", tg.Arn)
		ctx.Export("namespaceId", ns.ID())
		ctx.Export("securityGroupId", sg.ID())

		return nil
	})
}
