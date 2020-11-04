package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ssoadmin"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAwsSsoInstance() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsSsoInstanceRead,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"identity_store_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsSsoInstanceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ssoadminconn

	log.Printf("[DEBUG] Reading AWS SSO Instances")
	instances := []*ssoadmin.InstanceMetadata{}
	err := conn.ListInstancesPages(&ssoadmin.ListInstancesInput{}, func(page *ssoadmin.ListInstancesOutput, lastPage bool) bool {
		if page != nil && len(page.Instances) != 0 {
			instances = append(instances, page.Instances...)
		}
		return !lastPage
	})
	if err != nil {
		return fmt.Errorf("Error getting AWS SSO Instances: %s", err)
	}

	if len(instances) == 0 {
		log.Printf("[DEBUG] No AWS SSO Instance found")
		d.SetId("")
		return nil
	}

	if len(instances) > 1 {
		return fmt.Errorf("Found multiple AWS SSO Instances. Not sure which one to use. %s", instances)
	}

	instance := instances[0]
	log.Printf("[DEBUG] Received AWS SSO Instance: %s", instance)

	id, idErr := dataSourceAwsSsoInstanceID(aws.StringValue(instance.InstanceArn), aws.StringValue(instance.IdentityStoreId))
	if idErr != nil {
		return idErr
	}
	d.SetId(id)

	d.Set("arn", instance.InstanceArn)
	d.Set("identity_store_id", instance.IdentityStoreId)

	return nil
}

func dataSourceAwsSsoInstanceID(instanceArn string, identityStoreId string) (string, error) {
	// arn:${Partition}:sso:::instance/${InstanceId}
	iArn, err := arn.Parse(instanceArn)
	if err != nil {
		return "", err
	}
	iArnResourceParts := strings.Split(iArn.Resource, "/")
	if len(iArnResourceParts) != 2 || iArnResourceParts[0] != "instance" || iArnResourceParts[1] == "" {
		return "", fmt.Errorf("Unexpected format of ARN (%s), expected arn:${Partition}:sso:::instance/${InstanceId}", instanceArn)
	}
	instanceID := iArnResourceParts[1]

	vars := []string{
		instanceID,
		identityStoreId,
	}
	return strings.Join(vars, "/"), nil
}
