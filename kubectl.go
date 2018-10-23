/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	goflag "flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/spf13/pflag"

	utilflag "k8s.io/apiserver/pkg/util/flag"
	// "k8s.io/kubernetes/pkg/kubectl/cmd"
	"k8s.io/kubernetes/pkg/kubectl/util/logs"

	// Import to initialize client auth plugins.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func kubectl(args []string) string {
	os.Args = args //TODO investigate: k8s code is ignoring the passed args

	var b bytes.Buffer
	if err := kubectlCmd(args, nil, &b, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return ""
	}
	s := b.String()
	return s
}

func kubectlCmd(args []string, in io.Reader, out, errout io.Writer) error {
	rand.Seed(time.Now().UTC().UnixNano())

	// command := cmd.NewDefaultKubectlCommand()
	command := NewDefaultKubectlCommandWithArgs(&defaultPluginHandler{}, args, in, out, errout)

	// TODO: once we switch everything over to Cobra commands, we can go back to calling
	// utilflag.InitFlags() (by removing its pflag.Parse() call). For now, we have to set the
	// normalize func and add the go flag set by hand.
	pflag.CommandLine.SetNormalizeFunc(utilflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	// utilflag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		// fmt.Fprintf(os.Stderr, "%v\n", err)
		//os.Exit(1)
		return err
	}
	return nil
}
