package awsstorage

import (
	"errors"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/smithy-go"
	"strings"
)

const (
	secretMarkedForDeletionMsg    = "marked for deletion"
	secretScheduledForDeletionMsg = "scheduled for deletion"
)

// Returns true if the error matches all these conditions:
//   - err is of type awserr.Error
//   - Error.Code() matches code
//   - Error.Message() contains message
func isAWSErr(err error, code string, message string) bool {
	var awsError smithy.APIError
	if errors.As(err, &awsError) {
		return awsError.ErrorCode() == code && strings.Contains(awsError.ErrorMessage(), message)
	}
	return false
}

func isAwsNotFoundError(err error) bool {
	return isAWSErr(err, secretsmanager.ErrCodeResourceNotFoundException, "")
}

func isAwsScheduledForDeletionError(err error) bool {
	return isAWSErr(err, secretsmanager.ErrCodeInvalidRequestException, secretScheduledForDeletionMsg)
}

func isAwsSecretMarkedForDeletionError(err error) bool {
	return isAWSErr(err, secretsmanager.ErrCodeInvalidRequestException, secretMarkedForDeletionMsg)
}
func isAwsInvalidRequestError(err error) bool {
	return isAWSErr(err, secretsmanager.ErrCodeInvalidRequestException, "")
}

func isAwsResourceExistsError(err error) bool {
	return isAWSErr(err, secretsmanager.ErrCodeResourceExistsException, "")
}
