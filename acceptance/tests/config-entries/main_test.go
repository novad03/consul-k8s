// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package config_entries

import (
	"os"
	"testing"

	testSuite "github.com/hashicorp/consul-k8s/acceptance/framework/suite"
)

var suite testSuite.Suite

func TestMain(m *testing.M) {
	suite = testSuite.NewSuite(m)
	os.Exit(suite.Run())
}
