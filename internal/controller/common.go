package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

func doNotRequeue() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func requeueWithError(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}
