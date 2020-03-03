package settings

import (
	"bytes"
	"context"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	ClientSet "github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	jenkinsApi "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsOperatorSpec "github.com/epmd-edp/jenkins-operator/v2/pkg/service/jenkins/spec"
	"github.com/pkg/errors"
	"html/template"
	"io/ioutil"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

func CreateWorkdir(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(path, 0744)
		}
	}
	return nil
}

func GetUserSettingsConfigMap(clientSet ClientSet.ClientSet, namespace string) (*model.UserSettings, error) {
	userSettings, err := clientSet.CoreClient.ConfigMaps(namespace).Get("edp-config", metav1.GetOptions{})
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to get user settings configmap: %v", err)
		log.Println(errorMsg)
		return nil, errors.New(errorMsg)
	}
	vcsIntegrationEnabled, err := strconv.ParseBool(userSettings.Data["vcs_integration_enabled"])
	if err != nil {
		errorMsg := fmt.Sprint(err)
		log.Println(errorMsg)
		return nil, errors.New(errorMsg)
	}
	perfIntegrationEnabled, err := strconv.ParseBool(userSettings.Data["perf_integration_enabled"])
	if err != nil {
		errorMsg := fmt.Sprint(err)
		log.Println(errorMsg)
		return nil, errors.New(errorMsg)
	}
	return &model.UserSettings{
		DnsWildcard:            userSettings.Data["dns_wildcard"],
		EdpName:                userSettings.Data["edp_name"],
		EdpVersion:             userSettings.Data["edp_version"],
		PerfIntegrationEnabled: perfIntegrationEnabled,
		VcsGroupNameUrl:        userSettings.Data["vcs_group_name_url"],
		VcsIntegrationEnabled:  vcsIntegrationEnabled,
		VcsSshPort:             userSettings.Data["vcs_ssh_port"],
		VcsToolName:            model.VCSTool(userSettings.Data["vcs_tool_name"]),
	}, nil
}

func GetGerritSettingsConfigMap(clientSet ClientSet.ClientSet, namespace string) (*model.GerritSettings, error) {
	gerritSettings, err := clientSet.CoreClient.ConfigMaps(namespace).Get("gerrit", metav1.GetOptions{})
	sshPort, err := strconv.ParseInt(gerritSettings.Data["sshPort"], 10, 64)
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to get Gerrit settings configmap: %v", err)
		log.Println(errorMsg)
		return nil, errors.New(errorMsg)
	}
	return &model.GerritSettings{
		Config:            gerritSettings.Data["config"],
		ReplicationConfig: gerritSettings.Data["replication.config"],
		SshPort:           int32(sshPort),
	}, nil
}

func GetJenkins(k8sClient client.Client, namespace string) (*jenkinsApi.Jenkins, error) {
	options := client.ListOptions{Namespace: namespace}
	jenkinsList := &jenkinsApi.JenkinsList{}

	err := k8sClient.List(context.TODO(), &options, jenkinsList)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get Jenkins CRs in namespace %v", namespace)
	}

	if len(jenkinsList.Items) == 0 {
		return nil, fmt.Errorf("jenkins installation is not found in namespace %v", namespace)
	}

	return &jenkinsList.Items[0], nil
}

func GetJenkinsCreds(jenkins jenkinsApi.Jenkins, clientSet ClientSet.ClientSet, namespace string) (string, string, error) {
	annotationKey := fmt.Sprintf("%v/%v", jenkinsOperatorSpec.EdpAnnotationsPrefix, jenkinsOperatorSpec.JenkinsTokenAnnotationSuffix)
	jenkinsTokenSecretName := jenkins.Annotations[annotationKey]
	jenkinsTokenSecret, err := clientSet.CoreClient.Secrets(namespace).Get(jenkinsTokenSecretName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return "", "", errors.Wrapf(err, "Secret %v in not found", jenkinsTokenSecretName)
		}
		return "", "", errors.Wrapf(err, "Getting secret %v failed", jenkinsTokenSecretName)
	}
	return string(jenkinsTokenSecret.Data["password"]), string(jenkinsTokenSecret.Data["username"]), nil
}

func GetJenkinsUrl(jenkins jenkinsApi.Jenkins, namespace string) string {
	key := fmt.Sprintf("%v/%v", jenkinsOperatorSpec.EdpAnnotationsPrefix, "externalUrl")
	url := jenkins.Annotations[key]
	basePath := ""
	if len(jenkins.Spec.BasePath) > 0 {
		basePath = fmt.Sprintf("/%v", jenkins.Spec.BasePath)
	}
	if len(url) == 0 {
		return fmt.Sprintf("http://jenkins.%s:8080%v", namespace, basePath)
	}
	return url
}

func GetVcsCredentials(clientSet ClientSet.ClientSet, namespace string) (string, string, error) {
	vcsAutouserSecret, err := clientSet.CoreClient.Secrets(namespace).Get("vcs-autouser", metav1.GetOptions{})
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to get VCS credentials: %v", err)
		log.Println(errorMsg)
		return "", "", errors.New(errorMsg)
	}
	return string(vcsAutouserSecret.Data["ssh-privatekey"]), string(vcsAutouserSecret.Data["username"]), nil
}

func GetGerritCredentials(clientSet ClientSet.ClientSet, namespace string) (string, string, error) {
	vcsAutouserSecret, err := clientSet.CoreClient.Secrets(namespace).Get("gerrit-project-creator", metav1.GetOptions{})
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to get gerrit credentials: %v", err)
		log.Println(errorMsg)
		return "", "", errors.New(errorMsg)
	}
	return string(vcsAutouserSecret.Data["id_rsa"]), string(vcsAutouserSecret.Data["id_rsa.pub"]), nil
}

func CreateGerritPrivateKey(privateKey string, path string) error {
	err := ioutil.WriteFile(path, []byte(privateKey), 400)
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to write the Gerrit ssh key: %v", err)
		log.Println(errorMsg)
		return errors.New(errorMsg)
	}
	return nil
}

func CreateSshConfig(codebaseSettings model.CodebaseSettings) error {
	var config bytes.Buffer
	sshPath := "/home/codebase-operator/.ssh"
	if _, err := os.Stat(sshPath); os.IsNotExist(err) {
		err = os.MkdirAll(sshPath, 0744)
		if err != nil {
			return err
		}
	}

	tmpl, err := template.New("config.tmpl").ParseFiles("/usr/local/bin/templates/ssh/config.tmpl")
	if err != nil {
		return err
	}

	err = tmpl.Execute(&config, codebaseSettings)
	if err != nil {
		log.Printf("execute: %v", err)
		return err
	}

	f, err := os.OpenFile(sshPath+"/config", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(f, config.String())
	if err != nil {
		return err
	}

	defer f.Close()
	return nil
}

func GetVcsBasicAuthConfig(clientSet ClientSet.ClientSet, namespace string, secretName string) (string, string, error) {
	vcsCredentialsSecret, err := clientSet.CoreClient.Secrets(namespace).Get(secretName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) || k8serrors.IsForbidden(err) {
		return "", "", err
	}
	err = DeleteTempVcsSecret(clientSet, namespace, secretName)
	if err != nil {
		return "", "", err
	}
	return string(vcsCredentialsSecret.Data["username"]), string(vcsCredentialsSecret.Data["password"]), nil
}

func DeleteTempVcsSecret(clientSet ClientSet.ClientSet, namespace string, secretName string) error {
	log.Println("Start deleting temp secret with VCS credentials")
	err := clientSet.CoreClient.Secrets(namespace).Delete(secretName, &metav1.DeleteOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		errorMsg := fmt.Sprintf("Unable to delete temp secret: %v", err)
		log.Println(errorMsg)
		return errors.New(errorMsg)
	}
	log.Println("Temp secret with VCS credentials has been deleted")
	return nil
}
