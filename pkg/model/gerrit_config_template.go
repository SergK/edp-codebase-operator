package model

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
)

type ConfigGoTemplating struct {
	Lang         string             `json:"lang"`
	Route        *v1alpha1.Route    `json:"route"`
	Database     *v1alpha1.Database `json:"database"`
	Name         string
	PlatformType string
	DnsWildcard  string
	Framework    string
}
