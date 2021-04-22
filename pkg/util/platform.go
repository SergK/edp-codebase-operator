package util

import (
	"context"
	"fmt"
	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-component-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

const (
	watchNamespaceEnvVar   = "WATCH_NAMESPACE"
	debugModeEnvVar        = "DEBUG_MODE"
	inClusterNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

func GetUserSettings(client client.Client, namespace string) (*model.UserSettings, error) {
	us := &v1.ConfigMap{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      "edp-config",
	}, us)
	if err != nil {
		return nil, err
	}
	vcsIntegrationEnabled, err := strconv.ParseBool(us.Data["vcs_integration_enabled"])
	if err != nil {
		return nil, err
	}
	perfIntegrationEnabled, err := strconv.ParseBool(us.Data["perf_integration_enabled"])
	if err != nil {
		return nil, err
	}
	return &model.UserSettings{
		DnsWildcard:            us.Data["dns_wildcard"],
		EdpName:                us.Data["edp_name"],
		EdpVersion:             us.Data["edp_version"],
		PerfIntegrationEnabled: perfIntegrationEnabled,
		VcsGroupNameUrl:        us.Data["vcs_group_name_url"],
		VcsIntegrationEnabled:  vcsIntegrationEnabled,
		VcsSshPort:             us.Data["vcs_ssh_port"],
		VcsToolName:            model.VCSTool(us.Data["vcs_tool_name"]),
	}, nil
}

func GetGerritPort(c client.Client, namespace string) (*int32, error) {
	gs, err := getGitServerCR(c, "gerrit", namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "an error has occurred while getting %v Git Server CR", "gerrit")
	}
	return getInt32P(gs.Spec.SshPort), nil
}

func getInt32P(val int32) *int32 {
	return &val
}

func GetVcsBasicAuthConfig(c client.Client, namespace string, secretName string) (string, string, error) {
	log.Info("Start getting secret", "name", secretName)
	secret := &coreV1.Secret{}
	err := c.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      secretName,
	}, secret)
	if k8serrors.IsNotFound(err) || k8serrors.IsForbidden(err) {
		return "", "", err
	}
	return string(secret.Data["username"]), string(secret.Data["password"]), nil
}

func GetGitServer(c client.Client, name, namespace string) (*model.GitServer, error) {
	gitReq, err := getGitServerCR(c, name, namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "an error has occurred while getting %v Git Server CR", name)
	}

	gs, err := model.ConvertToGitServer(*gitReq)
	if err != nil {
		return nil, errors.Wrapf(err, "an error has occurred while converting request %v Git Server to DTO",
			name)
	}
	return gs, nil
}

func getGitServerCR(c client.Client, name, namespace string) (*edpv1alpha1.GitServer, error) {
	log.Info("Start fetching GitServer resource from k8s", "name", name, "namespace", namespace)
	instance := &edpv1alpha1.GitServer{}
	if err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, instance); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, errors.Wrapf(err, "Git Server %v doesn't exist in k8s.", name)
		}
		return nil, err
	}
	log.Info("Git Server instance has been received", "name", name)
	return instance, nil
}

func GetSecret(c client.Client, secretName, namespace string) (*v1.Secret, error) {
	log.Info("Start fetching Secret resource from k8s", "secret name", secretName, "namespace", namespace)
	secret := &coreV1.Secret{}
	err := c.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      secretName,
	}, secret)
	if k8serrors.IsNotFound(err) || k8serrors.IsForbidden(err) {
		return nil, err
	}
	log.Info("Secret has been fetched", "secret name", secretName, "namespace", namespace)
	return secret, nil
}

func GetCodebase(client client.Client, name, namespace string) (*edpv1alpha1.Codebase, error) {
	instance := &edpv1alpha1.Codebase{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, instance)

	if err != nil {
		return nil, err
	}

	return instance, nil
}

func GetSecretData(client client.Client, name, namespace string) (*coreV1.Secret, error) {
	s := &coreV1.Secret{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func GetEdpComponent(c client.Client, name, namespace string) (*v1alpha1.EDPComponent, error) {
	ec := &v1alpha1.EDPComponent{}
	err := c.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, ec)
	if err != nil {
		return nil, err
	}
	return ec, nil
}

// GetWatchNamespace returns the namespace the operator should be watching for changes
func GetWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}
	return ns, nil
}

// GetDebugMode returns the debug mode value
func GetDebugMode() (bool, error) {
	mode, found := os.LookupEnv(debugModeEnvVar)
	if !found {
		return false, nil
	}

	b, err := strconv.ParseBool(mode)
	if err != nil {
		return false, err
	}
	return b, nil
}

// Check whether the operator is running in cluster or locally
func RunningInCluster() bool {
	_, err := os.Stat(inClusterNamespacePath)
	if os.IsNotExist(err) {
		return false
	}
	return true
}
