package handler

import "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"

type ImageStreamTagHandler interface {
	ServeRequest(jira *v1alpha1.ImageStreamTag) error
}
