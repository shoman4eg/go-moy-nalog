//-----------------------------------------------------------------
//
//  This code is copyright 2022 by G5 Software.
//  Any unauthorized usage, either in part or in whole of this code
//  is strictly prohibited. Violators WILL be prosecuted to the
//  maximum extent allowed by law.
//
//-----------------------------------------------------------------

//go:build tools
// +build tools

package tools

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "mvdan.cc/gofumpt"
)

// tools
//go:generate go build -o ../bin/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint
//go:generate go build -o ../bin/gofumports mvdan.cc/gofumpt
