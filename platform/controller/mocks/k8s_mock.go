// Package mocks provides mock implementations for testing the controller.
package mocks

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockClient is a mock implementation of client.Client for testing.
// It wraps a real client and allows intercepting calls for testing.
type MockClient struct {
	client.Client
	GetFunc    func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error
	ListFunc   func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
	CreateFunc func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error
	UpdateFunc func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error
	DeleteFunc func(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error
	PatchFunc  func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error
}

// Get retrieves an object.
func (m *MockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key, obj, opts...)
	}
	if m.Client != nil {
		return m.Client.Get(ctx, key, obj, opts...)
	}
	return nil
}

// List retrieves a list of objects.
func (m *MockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, list, opts...)
	}
	if m.Client != nil {
		return m.Client.List(ctx, list, opts...)
	}
	return nil
}

// Create creates an object.
func (m *MockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, obj, opts...)
	}
	if m.Client != nil {
		return m.Client.Create(ctx, obj, opts...)
	}
	return nil
}

// Update updates an object.
func (m *MockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, obj, opts...)
	}
	if m.Client != nil {
		return m.Client.Update(ctx, obj, opts...)
	}
	return nil
}

// Delete deletes an object.
func (m *MockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, obj, opts...)
	}
	if m.Client != nil {
		return m.Client.Delete(ctx, obj, opts...)
	}
	return nil
}

// Patch patches an object.
func (m *MockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if m.PatchFunc != nil {
		return m.PatchFunc(ctx, obj, patch, opts...)
	}
	if m.Client != nil {
		return m.Client.Patch(ctx, obj, patch, opts...)
	}
	return nil
}

// Scheme returns the scheme.
func (m *MockClient) Scheme() *runtime.Scheme {
	if m.Client != nil {
		return m.Client.Scheme()
	}
	return runtime.NewScheme()
}

// RESTMapper returns the REST mapper.
func (m *MockClient) RESTMapper() meta.RESTMapper {
	if m.Client != nil {
		return m.Client.RESTMapper()
	}
	return nil
}

// GroupVersionKindFor returns the GVK for an object.
func (m *MockClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	if m.Client != nil {
		return m.Client.GroupVersionKindFor(obj)
	}
	return schema.GroupVersionKind{}, nil
}

// IsObjectNamespaced returns whether the object is namespaced.
func (m *MockClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	if m.Client != nil {
		return m.Client.IsObjectNamespaced(obj)
	}
	return true, nil
}

// Status returns the status writer.
func (m *MockClient) Status() client.SubResourceWriter {
	if m.Client != nil {
		return m.Client.Status()
	}
	return &MockStatusWriter{}
}

// SubResource returns a subresource client.
func (m *MockClient) SubResource(subResource string) client.SubResourceClient {
	if m.Client != nil {
		return m.Client.SubResource(subResource)
	}
	return nil
}

// MockStatusWriter is a mock implementation of client.SubResourceWriter.
type MockStatusWriter struct {
	UpdateFunc func(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error
	PatchFunc  func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error
}

// Create is not implemented.
func (m *MockStatusWriter) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	return nil
}

// Update updates the status.
func (m *MockStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, obj, opts...)
	}
	return nil
}

// Patch patches the status.
func (m *MockStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	if m.PatchFunc != nil {
		return m.PatchFunc(ctx, obj, patch, opts...)
	}
	return nil
}
