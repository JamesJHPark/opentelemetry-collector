// Copyright The OpenTelemetry Authors
//
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

package configauth

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/collector/client"
)

func TestDefaultUnaryInterceptorAuthSucceeded(t *testing.T) {
	// prepare
	handlerCalled := false
	authCalled := false
	authFunc := func(context.Context, map[string][]string) (context.Context, error) {
		authCalled = true
		ctx := client.NewContext(context.Background(), client.Info{
			Addr: &net.IPAddr{IP: net.IPv4(1, 2, 3, 4)},
		})

		return ctx, nil
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		cl := client.FromContext(ctx)
		assert.Equal(t, "1.2.3.4", cl.Addr.String())
		return nil, nil
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "some-auth-data"))

	// test
	res, err := DefaultGRPCUnaryServerInterceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler, authFunc)

	// verify
	assert.Nil(t, res)
	assert.NoError(t, err)
	assert.True(t, authCalled)
	assert.True(t, handlerCalled)
}

func TestDefaultUnaryInterceptorAuthFailure(t *testing.T) {
	// prepare
	authCalled := false
	expectedErr := fmt.Errorf("not authenticated")
	authFunc := func(context.Context, map[string][]string) (context.Context, error) {
		authCalled = true
		return context.Background(), expectedErr
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		assert.FailNow(t, "the handler should not have been called on auth failure!")
		return nil, nil
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "some-auth-data"))

	// test
	res, err := DefaultGRPCUnaryServerInterceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler, authFunc)

	// verify
	assert.Nil(t, res)
	assert.Equal(t, expectedErr, err)
	assert.True(t, authCalled)
}

func TestDefaultUnaryInterceptorMissingMetadata(t *testing.T) {
	// prepare
	authFunc := func(context.Context, map[string][]string) (context.Context, error) {
		assert.FailNow(t, "the auth func should not have been called!")
		return context.Background(), nil
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		assert.FailNow(t, "the handler should not have been called!")
		return nil, nil
	}

	// test
	res, err := DefaultGRPCUnaryServerInterceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler, authFunc)

	// verify
	assert.Nil(t, res)
	assert.Equal(t, errMetadataNotFound, err)
}

func TestDefaultStreamInterceptorAuthSucceeded(t *testing.T) {
	// prepare
	handlerCalled := false
	authCalled := false
	authFunc := func(context.Context, map[string][]string) (context.Context, error) {
		authCalled = true
		ctx := client.NewContext(context.Background(), client.Info{
			Addr: &net.IPAddr{IP: net.IPv4(1, 2, 3, 4)},
		})
		return ctx, nil
	}
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		// ensure that the client information is propagated down to the underlying stream
		cl := client.FromContext(stream.Context())
		assert.Equal(t, "1.2.3.4", cl.Addr.String())
		handlerCalled = true
		return nil
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "some-auth-data"))
	streamServer := &mockServerStream{
		ctx: ctx,
	}

	// test
	err := DefaultGRPCStreamServerInterceptor(nil, streamServer, &grpc.StreamServerInfo{}, handler, authFunc)

	// verify
	assert.NoError(t, err)
	assert.True(t, authCalled)
	assert.True(t, handlerCalled)
}

func TestDefaultStreamInterceptorAuthFailure(t *testing.T) {
	// prepare
	authCalled := false
	expectedErr := fmt.Errorf("not authenticated")
	authFunc := func(context.Context, map[string][]string) (context.Context, error) {
		authCalled = true
		return context.Background(), expectedErr
	}
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		assert.FailNow(t, "the handler should not have been called on auth failure!")
		return nil
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "some-auth-data"))
	streamServer := &mockServerStream{
		ctx: ctx,
	}

	// test
	err := DefaultGRPCStreamServerInterceptor(nil, streamServer, &grpc.StreamServerInfo{}, handler, authFunc)

	// verify
	assert.Equal(t, expectedErr, err)
	assert.True(t, authCalled)
}

func TestDefaultStreamInterceptorMissingMetadata(t *testing.T) {
	// prepare
	authFunc := func(context.Context, map[string][]string) (context.Context, error) {
		assert.FailNow(t, "the auth func should not have been called!")
		return context.Background(), nil
	}
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		assert.FailNow(t, "the handler should not have been called!")
		return nil
	}
	streamServer := &mockServerStream{
		ctx: context.Background(),
	}

	// test
	err := DefaultGRPCStreamServerInterceptor(nil, streamServer, &grpc.StreamServerInfo{}, handler, authFunc)

	// verify
	assert.Equal(t, errMetadataNotFound, err)
}

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}
