package webhook

import (
	"context"
	"fmt"

	"github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

type RemoteSecretValidator struct{}

func (a *RemoteSecretValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	rs, ok := obj.(*v1beta1.RemoteSecret)
	if !ok {
		return fmt.Errorf("expected a RemoteSecret but got a %T", obj)
	}
	return validateUniqueTargets(rs)
}

func (a *RemoteSecretValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	rs, ok := newObj.(*v1beta1.RemoteSecret)
	if !ok {
		return fmt.Errorf("expected a RemoteSecret but got a %T", newObj)
	}
	return validateUniqueTargets(rs)
}

func (a *RemoteSecretValidator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

func validateUniqueTargets(rs *v1beta1.RemoteSecret) error {
	targets := make(map[v1beta1.RemoteSecretTarget]string)
	for _, t := range rs.Spec.Targets {
		targets[t] = ""
	}
	if len(targets) != len(rs.Spec.Targets) {
		return fmt.Errorf("Targets are not unique in %s: %s", rs.Name, rs.Spec.Targets)
	}
	return nil
}
