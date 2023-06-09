// Copyright 2018 Palantir Technologies, Inc.
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

package main

import (
	"fmt"
	"os"

	"github.com/palantir/go-baseapp/pkg/errfmt"
	"github.com/ridge/bulldozer/cmd"
	_ "golang.org/x/crypto/x509roots/fallback"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		if cmd.IsDebugMode() {
			fmt.Fprint(os.Stderr, errfmt.Print(err)+"\n")
		} else {
			fmt.Fprint(os.Stderr, err.Error()+"\n")
		}
		os.Exit(-1)
	}
}
