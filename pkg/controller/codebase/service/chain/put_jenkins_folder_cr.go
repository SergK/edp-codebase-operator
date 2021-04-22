package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
	"time"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/platform"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	jenkinsv1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type PutJenkinsFolder struct {
	next   handler.CodebaseHandler
	client client.Client
}

func (h PutJenkinsFolder) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)

	gs, err := util.GetGitServer(h.client, c.Spec.GitServer, c.Namespace)
	if err != nil {
		return err
	}

	log.Info("GIT server has been retrieved", "name", gs.Name)
	path := getRepositoryPath(c.Name, string(c.Spec.Strategy), c.Spec.GitUrlPath)
	sshLink := generateSshLink(path, gs)
	jpm := map[string]string{
		"PARAM":                    "true",
		"NAME":                     c.Name,
		"BUILD_TOOL":               strings.ToLower(c.Spec.BuildTool),
		"DEFAULT_BRANCH":           c.Spec.DefaultBranch,
		"GIT_SERVER_CR_NAME":       gs.Name,
		"GIT_SERVER_CR_VERSION":    "v2",
		"GIT_CREDENTIALS_ID":       gs.NameSshKeySecret,
		"REPOSITORY_PATH":          sshLink,
		"JIRA_INTEGRATION_ENABLED": strconv.FormatBool(isJiraIntegrationEnabled(c.Spec.JiraServer)),
		"PLATFORM_TYPE":            platform.GetPlatformType(),
	}

	jc, err := json.Marshal(jpm)
	if err != nil {
		return errors.Wrapf(err, "Can't marshal parameters %v into json string", jpm)
	}

	rLog.Info("start creating jenkins folder...")
	if err := h.putJenkinsFolder(c, string(jc)); err != nil {
		setFailedFields(c, v1alpha1.PutJenkinsFolder, err.Error())
		return err
	}
	rLog.Info("end creating jenkins folder...")
	return nextServeOrNil(h.next, c)
}

func (h PutJenkinsFolder) putJenkinsFolder(c *v1alpha1.Codebase, jc string) error {
	jfn := fmt.Sprintf("%v-%v", c.Name, "codebase")
	jfr, err := h.getJenkinsFolder(jfn, c.Namespace)
	if err != nil {
		return err
	}

	if jfr != nil {
		log.Info("jenkins folder already exists in cluster", "name", jfn)
		return nil
	}

	jf := &jenkinsv1alpha1.JenkinsFolder{
		TypeMeta: metav1.TypeMeta{
			APIVersion: util.V2APIVersion,
			Kind:       util.JenkinsFolderKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      jfn,
			Namespace: c.Namespace,
			Labels: map[string]string{
				util.CodebaseLabelKey: c.Name,
			},
		},
		Spec: jenkinsv1alpha1.JenkinsFolderSpec{
			Job: &jenkinsv1alpha1.Job{
				Name:   fmt.Sprintf("job-provisions/job/ci/job/%v", *c.Spec.JobProvisioning),
				Config: jc,
			},
		},
		Status: jenkinsv1alpha1.JenkinsFolderStatus{
			Available:       false,
			LastTimeUpdated: time.Time{},
			Status:          util.StatusInProgress,
		},
	}
	if err := h.client.Create(context.TODO(), jf); err != nil {
		return errors.Wrapf(err, "couldn't create jenkins folder %v", "name")
	}
	return nil
}

func (h PutJenkinsFolder) getJenkinsFolder(name, namespace string) (*jenkinsv1alpha1.JenkinsFolder, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	i := &jenkinsv1alpha1.JenkinsFolder{}
	if err := h.client.Get(context.TODO(), nsn, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get instance by owner %v", name)
	}
	return i, nil
}

func newTrue() *bool {
	b := true
	return &b
}

func getRepositoryPath(codebaseName, strategy string, gitUrlPath *string) string {
	if strategy == consts.ImportStrategy {
		return *gitUrlPath
	}
	return "/" + codebaseName
}

func generateSshLink(repoPath string, gs *model.GitServer) string {
	l := fmt.Sprintf("ssh://%v@%v:%v%v", gs.GitUser, gs.GitHost, gs.SshPort, repoPath)
	log.Info("generated SSH link", "link", l)
	return l
}

func isJiraIntegrationEnabled(server *string) bool {
	if server != nil {
		return true
	}
	return false
}
