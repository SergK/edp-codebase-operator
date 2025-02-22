package util

import (
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetStringP(val string) *string {
	return &val
}

func GetPointerStringP(val *string) *string {
	return val
}

func GetWorkDir(codebaseName, namespace string) string {
	value, ok := os.LookupEnv("WORKING_DIR")
	if !ok {
		return fmt.Sprintf("/home/codebase-operator/edp/%v/%v/%v/%v", namespace, codebaseName, "templates", codebaseName)
	}
	return fmt.Sprintf("%v/codebase-operator/edp/%v/%v/%v/%v", value, namespace, codebaseName, "templates", codebaseName)
}

func GetAssetsDir() string {
	value, ok := os.LookupEnv("ASSETS_DIR")
	if !ok {
		panic("env variable ASSETS_DIR is missing")
	}
	return value
}

func GetOwnerReference(ownerKind string, ors []metav1.OwnerReference) (*metav1.OwnerReference, error) {
	log.V(2).Info("finding owner", "kind", ownerKind)
	if len(ors) == 0 {
		return nil, fmt.Errorf("resource doesn't have %v owner reference", ownerKind)
	}
	for _, o := range ors {
		if o.Kind == ownerKind {
			return &o, nil
		}
	}
	return nil, fmt.Errorf("resource doesn't have %v owner reference", ownerKind)
}
