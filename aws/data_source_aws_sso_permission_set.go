package aws

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssoadmin"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/keyvaluetags"
)

func dataSourceAwsSsoPermissionSet() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsSsoPermissionSetRead,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"created_date": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"instance_arn": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(10, 1224),
					validation.StringMatch(regexp.MustCompile(`^arn:aws:sso:::instance/(sso)?ins-[a-zA-Z0-9-.]{16}$`), "must match arn:aws:sso:::instance/(sso)?ins-[a-zA-Z0-9-.]{16}"),
				),
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 32),
					validation.StringMatch(regexp.MustCompile(`^[\w+=,.@-]+$`), "must match [\\w+=,.@-]"),
				),
			},

			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"session_duration": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"relay_state": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"inline_policy": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"managed_policy_arns": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"tags": tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsSsoPermissionSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ssoadminconn
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	instanceArn := d.Get("instance_arn").(string)
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Reading AWS SSO Permission Sets")
	resp, err := conn.ListPermissionSets(&ssoadmin.ListPermissionSetsInput{
		InstanceArn: aws.String(instanceArn),
		MaxResults:  aws.Int64(100),
	})
	if err != nil {
		return fmt.Errorf("Error getting AWS SSO Permission Sets: %s", err)
	}
	if resp == nil || len(resp.PermissionSets) == 0 {
		log.Printf("[DEBUG] No AWS SSO Permission Sets found")
		d.SetId("")
		return nil
	}

	// TODO: paging (if resp.NextToken != nil)
	var permissionSetArn string
	var permissionSet *ssoadmin.PermissionSet
	for _, permissionSetArns := range resp.PermissionSets {
		permissionSetArn = aws.StringValue(permissionSetArns)
		log.Printf("[DEBUG] Reading AWS SSO Permission Set: %v", permissionSetArn)
		permissionSetResp, permissionSetErr := conn.DescribePermissionSet(&ssoadmin.DescribePermissionSetInput{
			InstanceArn:      aws.String(instanceArn),
			PermissionSetArn: aws.String(permissionSetArn),
		})
		if permissionSetErr != nil {
			return fmt.Errorf("Error getting AWS SSO Permission Set: %s", permissionSetErr)
		}
		if aws.StringValue(permissionSetResp.PermissionSet.Name) == name {
			permissionSet = permissionSetResp.PermissionSet
			break
		}
	}

	if permissionSet == nil {
		log.Printf("[DEBUG] AWS SSO Permission Set %v not found", name)
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] Found AWS SSO Permission Set: %s", permissionSet)

	log.Printf("[DEBUG] Getting Inline Policy for AWS SSO Permission Set")
	inlinePolicyResp, inlinePolicyErr := conn.GetInlinePolicyForPermissionSet(&ssoadmin.GetInlinePolicyForPermissionSetInput{
		InstanceArn:      aws.String(instanceArn),
		PermissionSetArn: aws.String(permissionSetArn),
	})
	if inlinePolicyErr != nil {
		return fmt.Errorf("Error getting Inline Policy for AWS SSO Permission Set: %s", inlinePolicyErr)
	}

	log.Printf("[DEBUG] Getting Managed Policies for AWS SSO Permission Set")
	managedPoliciesResp, managedPoliciesErr := conn.ListManagedPoliciesInPermissionSet(&ssoadmin.ListManagedPoliciesInPermissionSetInput{
		InstanceArn:      aws.String(instanceArn),
		PermissionSetArn: aws.String(permissionSetArn),
	})
	if managedPoliciesErr != nil {
		return fmt.Errorf("Error getting Managed Policies for AWS SSO Permission Set: %s", managedPoliciesErr)
	}
	var managedPolicyArns []string
	for _, managedPolicy := range managedPoliciesResp.AttachedManagedPolicies {
		managedPolicyArns = append(managedPolicyArns, aws.StringValue(managedPolicy.Arn))
	}

	tags, err := keyvaluetags.SsoListTags(conn, permissionSetArn, instanceArn)
	if err != nil {
		return fmt.Errorf("Error listing tags for ASW SSO Permission Set (%s): %s", permissionSetArn, err)
	}

	d.SetId(permissionSetArn)
	d.Set("arn", permissionSetArn)
	d.Set("created_date", permissionSet.CreatedDate.Format(time.RFC3339))
	d.Set("instance_arn", instanceArn)
	d.Set("name", permissionSet.Name)
	d.Set("description", permissionSet.Description)
	d.Set("session_duration", permissionSet.SessionDuration)
	d.Set("relay_state", permissionSet.RelayState)
	d.Set("inline_policy", inlinePolicyResp.InlinePolicy)
	d.Set("managed_policy_arns", managedPolicyArns)
	if err := d.Set("tags", tags.IgnoreAws().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}

	return nil
}
