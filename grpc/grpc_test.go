// Copyright Greg Weber and PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0

package grpc_test

import (
	"fmt"
	"testing"

	"github.com/gregwebs/errcode"
	"github.com/gregwebs/errcode/grpc"
	"google.golang.org/grpc/codes"
)

// Test setting the HTTP code
type GRPCError struct{}

func (e GRPCError) Error() string { return "error" }

const grpcCodeStr = "input.grpc"

var codeAborted = grpc.SetCode(errcode.InvalidInputCode.Child(grpcCodeStr), codes.Aborted)

func (e GRPCError) Code() errcode.Code {
	return codeAborted
}

func (e GRPCError) WrapError(apply func(err error) error) {
	panic("WrapError not implemented")
}

func TestGrpcErrorCode(t *testing.T) {
	err := GRPCError{}
	AssertGRPCCode(t, err, codes.Aborted)
}

func TestWrapAsGrpc(t *testing.T) {
	err := grpc.WrapAsGRPC(errcode.NewInternalErr(fmt.Errorf("wrap me up")))
	AssertGRPCCode(t, err, codes.Internal)
}

func AssertGRPCCode(t *testing.T, code errcode.ErrorCode, grpcCode codes.Code) {
	t.Helper()
	expected := grpc.GetCode(code.Code())
	if expected != grpcCode {
		t.Errorf("excpected HTTP Code %v but got %v", grpcCode, expected)
	}
}
