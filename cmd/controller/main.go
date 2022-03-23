package main

import (
	"flag"

	"github.com/vdemeester/tekton-task-group/pkg/apis/taskgroup/v1alpha1"
	"github.com/vdemeester/tekton-task-group/pkg/reconciler/taskgroup"
	corev1 "k8s.io/api/core/v1"
	filteredinformerfactory "knative.dev/pkg/client/injection/kube/informers/factory/filtered"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"
)

const (
	// ControllerLogKey is the name of the logger for the controller cmd
	ControllerLogKey = "tekton-pipelines-controller"
)

func main() {
	namespace := flag.String("namespace", corev1.NamespaceAll, "Namespace to restrict informer to. Optional, defaults to all namespaces.")

	// This parses flags.
	cfg := injection.ParseAndGetRESTConfigOrDie()

	ctx := injection.WithNamespaceScope(signals.NewContext(), *namespace)
	ctx = filteredinformerfactory.WithSelectors(ctx, v1alpha1.ManagedByLabelKey)
	sharedmain.MainWithConfig(ctx, ControllerLogKey, cfg,
		taskgroup.NewController(),
	)
}
