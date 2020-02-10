package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/pkg/errors"
)

var (
	profile        = flag.String("p", "", "Profile to use")
	tagValue       = flag.String("t", "", "Tag to search for")
	instanceValue  = flag.String("i", "", "Instance to search for")
	quiet          = flag.Bool("q", false, "Only show the IP addresses")
	showTerminated = flag.Bool("showTerminated", false, "Show terminated instances")
)

func run() error {
	flag.Parse()

	if *tagValue == "" && *instanceValue == "" {
		return errors.New("Must specify tag or instance to search for")
	}

	cfg, err := external.LoadDefaultAWSConfig(external.WithSharedConfigProfile(*profile))
	if err != nil {
		return errors.Wrap(err, "failed to load aws config")
	}

	client := ec2.New(cfg)

	var input *ec2.DescribeInstancesInput
	if *tagValue != "" {
		input = &ec2.DescribeInstancesInput{
			Filters: []ec2.Filter{
				ec2.Filter{Name: aws.String("tag-value"), Values: []string{
					fmt.Sprintf("*%v*", *tagValue),
				}},
			},
		}
	} else {
		if !strings.HasPrefix(*instanceValue, "i-") {
			*instanceValue = fmt.Sprintf("i-%s", *instanceValue)
		}

		input = &ec2.DescribeInstancesInput{
			InstanceIds: []string{
				*instanceValue,
			},
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)
	resp, err := client.DescribeInstancesRequest(input).Send(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to describe instances")
	}

	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			var nameTag string
			for _, tag := range instance.Tags {
				if *tag.Key == "Name" {
					nameTag = *tag.Value
					break
				}
			}

			if nameTag == "" {
				nameTag = "<Unknown>"
			}

			var privateIPAddress string
			if instance.PrivateIpAddress == nil {
				privateIPAddress = "<unknown>"
			} else {
				privateIPAddress = *instance.PrivateIpAddress
			}

			if !*showTerminated {
				if instance.State.Name == ec2.InstanceStateNameTerminated {
					continue
				}
			}

			if !*quiet {
				fmt.Fprintf(w, "%v\t\t %v\t%v\t%v\n", nameTag, *instance.InstanceId, privateIPAddress, instance.State.Name)
			} else {
				fmt.Fprintf(w, "%v\n", privateIPAddress)
			}
		}
	}

	return w.Flush()
}

func main() {
	if err := run(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
