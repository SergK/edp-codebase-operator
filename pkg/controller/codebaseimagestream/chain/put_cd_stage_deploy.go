package chain

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PutCDStageDeploy struct {
	client client.Client
	log    logr.Logger
}

type cdStageDeployCommand struct {
	Name      string
	Namespace string
	Pipeline  string
	Stage     string
	Tag       jenkinsApi.Tag
}

const dateLayout = "2006-01-02T15:04:05"

func (h PutCDStageDeploy) ServeRequest(imageStream *codebaseApi.CodebaseImageStream) error {
	log := h.log.WithValues("name", imageStream.Name)
	log.Info("creating/updating CDStageDeploy.")
	if err := h.handleCodebaseImageStreamEnvLabels(imageStream); err != nil {
		return errors.Wrapf(err, "couldn't handle %v codebase image stream", imageStream.Name)
	}
	log.Info("creating/updating CDStageDeploy has been finished.")
	return nil
}

func (h PutCDStageDeploy) handleCodebaseImageStreamEnvLabels(imageStream *codebaseApi.CodebaseImageStream) error {
	if imageStream.ObjectMeta.Labels == nil || len(imageStream.ObjectMeta.Labels) == 0 {
		h.log.Info("codebase image stream doesnt contain env labels. skip CDStageDeploy creating...")
		return nil
	}

	var labelValueRegexp = regexp.MustCompile("^[-A-Za-z0-9_.]+/[-A-Za-z0-9_.]+$")

	for envLabel := range imageStream.ObjectMeta.Labels {
		if errs := validateCbis(imageStream, envLabel, labelValueRegexp); len(errs) != 0 {
			return errors.New(strings.Join(errs, "; "))
		}
		if err := h.putCDStageDeploy(envLabel, imageStream.Namespace, imageStream.Spec); err != nil {
			return err
		}
	}
	return nil
}

func validateCbis(imageStream *codebaseApi.CodebaseImageStream, envLabel string, labelValueRegexp *regexp.Regexp) []string {
	var errs []string

	if len(imageStream.Spec.Codebase) == 0 {
		errs = append(errs, "codebase is not defined in spec ")
	}
	if len(imageStream.Spec.Tags) == 0 {
		errs = append(errs, "tags are not defined in spec ")
	}

	if !labelValueRegexp.MatchString(envLabel) {
		errs = append(errs, "Label must be in format cd-pipeline-name/stage-name")
	}
	return errs
}

func (h PutCDStageDeploy) putCDStageDeploy(envLabel, namespace string, spec codebaseApi.CodebaseImageStreamSpec) error {
	name := generateCdStageDeployName(envLabel, spec.Codebase)
	stageDeploy, err := h.getCDStageDeploy(name, namespace)
	if err != nil {
		return errors.Wrapf(err, "couldn't get %v cd stage deploy", name)
	}

	if stageDeploy != nil {
		h.log.Info("CDStageDeploy already exists. skip creating.", "name", stageDeploy.Name)
		return &util.CDStageDeployHasNotBeenProcessed{
			Message: fmt.Sprintf("%v has not been processed for previous version of application yet", name),
		}
	}

	cdsd, err := getCreateCommand(envLabel, name, namespace, spec.Codebase, spec.Tags)
	if err != nil {
		return errors.Wrapf(err, "couldn't construct command to create %v cd stage deploy", name)
	}
	if err := h.create(cdsd); err != nil {
		return errors.Wrapf(err, "couldn't create %v cd stage deploy", name)
	}
	return nil
}

func generateCdStageDeployName(env, codebase string) string {
	env = strings.Replace(env, "/", "-", -1)
	return fmt.Sprintf("%v-%v", env, codebase)
}

func (h PutCDStageDeploy) getCDStageDeploy(name, namespace string) (*codebaseApi.CDStageDeploy, error) {
	h.log.Info("getting cd stage deploy", "name", name)
	i := &codebaseApi.CDStageDeploy{}
	nn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	if err := h.client.Get(context.TODO(), nn, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return i, nil
}

func getCreateCommand(envLabel, name, namespace, codebase string, tags []codebaseApi.Tag) (*cdStageDeployCommand, error) {
	env := strings.Split(envLabel, "/")

	lastTag, err := getLastTag(tags)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't create cdStageDeployCommand with name %v", name)
	}

	return &cdStageDeployCommand{
		Name:      name,
		Namespace: namespace,
		Pipeline:  env[0],
		Stage:     env[1],
		Tag: jenkinsApi.Tag{
			Codebase: codebase,
			Tag:      lastTag.Name,
		},
	}, nil
}

func getLastTag(tags []codebaseApi.Tag) (codebaseApi.Tag, error) {
	var (
		latestTag     codebaseApi.Tag
		latestTagTime = time.Time{}
	)
	for i, s := range tags {
		if current, err := time.Parse(dateLayout, tags[i].Created); err == nil {
			if current.After(latestTagTime) {
				latestTagTime = current
				latestTag = s
			}
		}
	}
	if latestTag.Name == "" {
		return latestTag, errors.New("There are no valid tags")
	}
	return latestTag, nil
}

func (h PutCDStageDeploy) create(command *cdStageDeployCommand) error {
	log := h.log.WithValues("name", command.Name)
	log.Info("cd stage deploy is not present in cluster. start creating...")

	stageDeploy := &codebaseApi.CDStageDeploy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: util.V2APIVersion,
			Kind:       util.CDStageDeployKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      command.Name,
			Namespace: command.Namespace,
		},
		Spec: codebaseApi.CDStageDeploySpec{
			Pipeline: command.Pipeline,
			Stage:    command.Stage,
			Tag:      command.Tag,
		},
	}
	if err := h.client.Create(context.TODO(), stageDeploy); err != nil {
		return err
	}
	log.Info("cd stage deploy has been created.")
	return nil
}
