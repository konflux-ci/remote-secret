package controllers

// Caution. Changing may lead to undesired effects in other projects that uses remote-secret.
const (
	UploadSecretLabel          = "appstudio.redhat.com/upload-secret"     //#nosec G101 -- false positive, this is not a token
	RemoteSecretNameAnnotation = "appstudio.redhat.com/remotesecret-name" //#nosec G101 -- false positive, this is not a token
	TargetNamespaceAnnotation  = "appstudio.redhat.com/remotesecret-target-namespace"
)
