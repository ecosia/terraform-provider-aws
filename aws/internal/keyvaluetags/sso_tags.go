// +build !generate

package keyvaluetags

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssoadmin"
)

// Custom SSO tag service functions using the same format as generated code.

// SsoListTags lists sso service tags.
// The identifier is typically the Amazon Resource Name (ARN), although
// it may also be a different identifier depending on the service.
func SsoListTags(conn *ssoadmin.SSOAdmin, identifier string, instanceArn string) (KeyValueTags, error) {
	input := &ssoadmin.ListTagsForResourceInput{
		InstanceArn: aws.String(instanceArn),
		ResourceArn: aws.String(identifier),
	}

	output, err := conn.ListTagsForResource(input)

	if err != nil {
		return New(nil), err
	}

	return SsoKeyValueTags(output.Tags), nil
}

// SsoUpdateTags updates sso service tags.
// The identifier is typically the Amazon Resource Name (ARN), although
// it may also be a different identifier depending on the service.
func SsoUpdateTags(conn *ssoadmin.SSOAdmin, identifier string, instanceArn string, oldTagsMap interface{}, newTagsMap interface{}) error {
	oldTags := New(oldTagsMap)
	newTags := New(newTagsMap)

	if removedTags := oldTags.Removed(newTags); len(removedTags) > 0 {
		input := &ssoadmin.UntagResourceInput{
			InstanceArn: aws.String(instanceArn),
			ResourceArn: aws.String(identifier),
			TagKeys:     aws.StringSlice(removedTags.IgnoreAws().Keys()),
		}

		_, err := conn.UntagResource(input)

		if err != nil {
			return fmt.Errorf("error untagging resource (%s): %w", identifier, err)
		}
	}

	if updatedTags := oldTags.Updated(newTags); len(updatedTags) > 0 {
		input := &ssoadmin.TagResourceInput{
			InstanceArn: aws.String(instanceArn),
			ResourceArn: aws.String(identifier),
			Tags:        updatedTags.IgnoreAws().SsoTags(),
		}

		_, err := conn.TagResource(input)

		if err != nil {
			return fmt.Errorf("error tagging resource (%s): %w", identifier, err)
		}
	}

	return nil
}

// SsoTags returns sso service tags.
func (tags KeyValueTags) SsoTags() []*ssoadmin.Tag {
	result := make([]*ssoadmin.Tag, 0, len(tags))

	for k, v := range tags.Map() {
		tag := &ssoadmin.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		}

		result = append(result, tag)
	}

	return result
}

// SsoKeyValueTags creates KeyValueTags from sso service tags.
func SsoKeyValueTags(tags []*ssoadmin.Tag) KeyValueTags {
	m := make(map[string]*string, len(tags))

	for _, tag := range tags {
		m[aws.StringValue(tag.Key)] = tag.Value
	}

	return New(m)
}
