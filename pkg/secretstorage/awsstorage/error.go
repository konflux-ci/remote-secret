//
// Copyright (c) 2023 Red Hat, Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package awsstorage

import (
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/smithy-go"
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
