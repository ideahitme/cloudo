package awsclient

import (
	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"gopkg.in/alecthomas/kingpin.v2"
)

const name = "cloudo"

type AWSClient struct {
	opt      *options
	vpcID    *string
	subnetID *string
	igID     *string //internet gateway
	rtID     *string //route table id
	sgID     *string //security group id
}

type options struct {
	vpcCIDR   string
	instances uint8
	region    string
	profile   string
}

var ( //defaults
	defVPCCIDR      = "10.0.0.0/24"
	defInstanceSize = "1"
	defRegion       = "eu-central-1"
	defProfile      = "yerken_tussupbekov"
)

func ReadFlags(app *kingpin.Application) {
	o := options{}
	aws := app.Command("aws", "Amazon Web Services")
	aws.Flag("region", "AWS Region").Default(defRegion).StringVar(&o.region)
	aws.Flag("profile", "AWS Profile").Default(defProfile).StringVar(&o.profile)

	create := aws.Command("create", "Provision CoreOS instances with separated VPC, Subnet, Security group with open ssh access and attached IG")
	create.Flag("vpc-cidr", "CIDR block for the VPC").Default(defVPCCIDR).StringVar(&o.vpcCIDR)
	create.Flag("instances", "Number of instances").Default(defInstanceSize).Uint8Var(&o.instances)
	create.Action(func(ctx *kingpin.ParseContext) error {
		client := New(&o)
		if err := client.CreateNetwork(); err != nil {
			logrus.Fatalln("Error while creating a network:", err)
			return err
		}
		return nil
	})
}

func New(o *options) *AWSClient {
	client := &AWSClient{
		opt: o,
	}
	return client
}

func (client *AWSClient) CreateNetwork() error {
	sess, err := session.NewSessionWithOptions(session.Options{
		Profile: client.opt.profile,
		Config: aws.Config{
			Region: &client.opt.region,
		},
	})

	if err != nil {
		return err
	}

	ec2Client := ec2.New(sess)
	if err = client.createVPC(ec2Client); err != nil {
		return err
	}
	if err = client.createSubnet(ec2Client); err != nil {
		return err
	}
	if err = client.createInternetGateway(ec2Client); err != nil {
		return err
	}
	if err = client.createRouteTable(ec2Client); err != nil {
		return err
	}
	if err = client.createSecurityGroup(ec2Client); err != nil {
		return err
	}
	return nil
}
