package awsclient

import (
	"errors"

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
		return client.Create()
	})
}

func New(o *options) *AWSClient {
	client := &AWSClient{
		opt: o,
	}
	return client
}

func (client *AWSClient) Create() error {
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
	return nil
}

func (client *AWSClient) createVPC(ec2Client *ec2.EC2) error {
	logrus.Infoln("Creating VPC")
	vpcInput := &ec2.CreateVpcInput{
		CidrBlock: &client.opt.vpcCIDR,
	}

	request, output := ec2Client.CreateVpcRequest(vpcInput)

	if err := request.Send(); err != nil {
		logrus.Errorf("Error creating VPC %v", err)
		return err
	}
	logrus.Infof("VPC (CIDR: %s) is created with id: %s", client.opt.vpcCIDR, *output.Vpc.VpcId)
	client.vpcID = output.Vpc.VpcId

	logrus.Infof("Tagging VPC %s with creator=%s tag", *client.vpcID, name)
	if err := client.tagResource(client.vpcID, ec2Client); err != nil {
		return err
	}

	logrus.Infoln("Enabling VPC dns-support")
	dnsInput := &ec2.ModifyVpcAttributeInput{
		VpcId: client.vpcID,
		EnableDnsSupport: &ec2.AttributeBooleanValue{
			Value: aws.Bool(true),
		},
	}

	if _, err := ec2Client.ModifyVpcAttribute(dnsInput); err != nil {
		return err
	}

	logrus.Infoln("Enabling VPC dns-hostnames")
	dnsInput = &ec2.ModifyVpcAttributeInput{
		VpcId: client.vpcID,
		EnableDnsHostnames: &ec2.AttributeBooleanValue{
			Value: aws.Bool(true),
		},
	}

	if _, err := ec2Client.ModifyVpcAttribute(dnsInput); err != nil {
		return err
	}

	logrus.Infoln("VPC is successfully configured. DHCP option set is automatically created by AWS")
	return nil
}

func (client *AWSClient) createSubnet(ec2Client *ec2.EC2) error {
	logrus.Infof("Creating Subnet for %s", *client.vpcID)
	zones, err := client.getAvailableZones(ec2Client)
	if err != nil {
		return err
	}
	if len(zones) == 0 {
		return errors.New("No zones are available for subnet creation")
	}
	logrus.Infoln("Available zones: ", zones)
	logrus.Infof("Picking: %s", zones[0])
	zone := zones[0]

	subnetInput := &ec2.CreateSubnetInput{
		AvailabilityZone: aws.String(zone),
		VpcId:            client.vpcID,
		CidrBlock:        aws.String(client.opt.vpcCIDR),
	}
	output, err := ec2Client.CreateSubnet(subnetInput)
	if err != nil {
		return err
	}
	client.subnetID = output.Subnet.SubnetId
	logrus.Infof("Subnet (CIDR: %s) is created with id: %s", client.opt.vpcCIDR, *output.Subnet.SubnetId)
	logrus.Infof("Tagging Subnet %s with creator=%s tag", *client.subnetID, name)
	if err := client.tagResource(client.subnetID, ec2Client); err != nil {
		return err
	}
	logrus.Infoln("Subnet is successfully configured")
	return nil
}

func (client *AWSClient) createInternetGateway(ec2Client *ec2.EC2) error {
	logrus.Infoln("Creating Internet Gateway")
	igInput := &ec2.CreateInternetGatewayInput{}
	output, err := ec2Client.CreateInternetGateway(igInput)
	if err != nil {
		return err
	}
	client.igID = output.InternetGateway.InternetGatewayId
	logrus.Infof("IG (%s) created. Attaching to VPC %s ...", *client.igID, *client.vpcID)

	attachInput := &ec2.AttachInternetGatewayInput{
		InternetGatewayId: client.igID,
		VpcId:             client.vpcID,
	}
	if _, err = ec2Client.AttachInternetGateway(attachInput); err != nil {
		return err
	}
	logrus.Infof("Tagging IG %s with creator=%s tag", *client.subnetID, name)
	client.tagResource(client.igID, ec2Client)
	return nil
}

func (client *AWSClient) getAvailableZones(ec2Client *ec2.EC2) ([]string, error) {
	zonesInput := &ec2.DescribeAvailabilityZonesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("region-name"),
				Values: []*string{aws.String(client.opt.region)},
			},
		},
	}
	output, err := ec2Client.DescribeAvailabilityZones(zonesInput)
	if err != nil {
		return nil, err
	}
	var zones []string
	for _, zone := range output.AvailabilityZones {
		if *zone.State == "available" {
			zones = append(zones, *zone.ZoneName)
		}
	}
	return zones, nil
}

func (client *AWSClient) tagResource(resourceID *string, ec2Client *ec2.EC2) error {
	tagInput := &ec2.CreateTagsInput{
		Resources: []*string{
			resourceID,
		},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   aws.String("creator"),
				Value: aws.String(name),
			},
		},
	}
	_, err := ec2Client.CreateTags(tagInput)
	return err
}
