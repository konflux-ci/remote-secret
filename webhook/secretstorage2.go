package webhook

import "context"

// TODO REMOVE IT

// +kubebuilder:object:generate=false
type SecretStorage2 interface {
	// Initialize initializes the connection to the underlying data store, etc.
	Initialize(ctx context.Context) error
	// Store stores the provided data under given id
	Store(ctx context.Context, id SecretID2, data []byte) error
	// Get retrieves the data under the given id. A NotFoundError is returned if the data is not found.
	Get(ctx context.Context, id SecretID2) ([]byte, error)
	// Delete deletes the data of given id. A NotFoundError is returned if there is no such data.
	Delete(ctx context.Context, id SecretID2) error
}

type SecretID2 struct {
	Name      string
	Namespace string
}
