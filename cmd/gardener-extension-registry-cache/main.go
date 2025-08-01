// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/gardener/gardener/pkg/logger"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/gardener/gardener-extension-registry-cache/cmd/gardener-extension-registry-cache/app"
)

func main() {
	logf.SetLogger(logger.MustNewZapLogger(logger.InfoLevel, logger.FormatJSON))

	ctx := signals.SetupSignalHandler()
	if err := app.NewServiceControllerCommand().ExecuteContext(ctx); err != nil {
		logf.Log.Error(err, "Error executing the main controller command")
		os.Exit(1)
	}
}
