package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/fatih/color"
)

func MustLoadConfig(p, r string) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithSharedConfigProfile(p))
	if err != nil {
		return aws.Config{}, fmt.Errorf("Erro ao carregar SDK %v", err)
	}
	return cfg, nil
}

func getInstancesWSnapshots(ctx context.Context, client *ec2.Client, key, value string) ([]string, []string, error) {
	resp, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		return nil, nil, fmt.Errorf("Erro ao chamar describe instances: Erro %v\n", err)
	}
	instancesOK := make([]string, 0)
	instancesNotOK := make([]string, 0)
	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			id, status := checkingTags(instance, key, value)
			if status == true {
				instancesOK = append(instancesOK, id)
			} else {
				instancesNotOK = append(instancesNotOK, id)
			}
		}
	}
	return instancesOK, instancesNotOK, nil
}

func checkingTags(instance types.Instance, key, value string) (string, bool) {
	for _, tag := range instance.Tags {
		if *tag.Key == key && *tag.Value == value {
			return *instance.InstanceId, true
		} else {
			return *instance.InstanceId, false
		}
	}
	return "", false
}

func getInstanceNameByID(ctx context.Context, client *ec2.Client, instanceID string) (string, error) {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}

	resp, err := client.DescribeInstances(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to describe instances: %v", err)
	}

	if len(resp.Reservations) == 0 || len(resp.Reservations[0].Instances) == 0 {
		return "", fmt.Errorf("instance not found for ID: %s", instanceID)
	}

	instance := resp.Reservations[0].Instances[0]

	for _, tag := range instance.Tags {
		if *tag.Key == "Name" {
			return *tag.Value, nil
		}
	}

	return "", fmt.Errorf("instance name not found for ID: %s", instanceID)
}

func settingFlags() (profileReturn, regionReturn, keyReturn, valueReturn string) {
	profile := flag.String("profile", "default", "set the aws profile to run the commands")
	region := flag.String("region", "us-east-1", "set the aws region to run the commands")
	key := flag.String("key", "default", "set the tag key to search")
	value := flag.String("value", "default", "set the tag value to search")
	flag.Parse()
	return *profile, *region, *key, *value
}

func main() {
	ctx := context.Background()
	profile, region, key, value := settingFlags()
	cfg, _ := MustLoadConfig(profile, region)
	client := ec2.NewFromConfig(cfg)

	withSnap, withoutSnap, _ := getInstancesWSnapshots(ctx, client, key, value)

	func(v, k []string) {
		red := color.New(color.FgRed).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()
		for i := 0; i < len(v); i++ {
			name, err := getInstanceNameByID(ctx, client, v[i])
			if err != nil {
				fmt.Printf("Erro: %v\n", err)
			}
			fmt.Printf("Possui Snapshot: -> %v\n", green(name))
		}
		for i := 0; i < len(k); i++ {
			name, err := getInstanceNameByID(ctx, client, k[i])
			if err != nil {
				fmt.Printf("Erro: %v\n", err)
			}
			fmt.Printf("Não possui Snapshot: -> %v\n", red(name))
		}
	}(withSnap, withoutSnap)

}
