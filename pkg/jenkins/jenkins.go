package jenkins

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bndr/gojenkins"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	jenkinsApi "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsOperatorSpec "github.com/epmd-edp/jenkins-operator/v2/pkg/service/jenkins/spec"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("jenkins-client")

type JenkinsClient struct {
	Jenkins *gojenkins.Jenkins
}

type JobNotFoundError string

func (j JobNotFoundError) Error() string {
	return string(j)
}

func Init(url string, username string, token string) (*JenkinsClient, error) {
	log.Info("initializing new Jenkins client", "url", url, "username", username)
	jenkins, err := gojenkins.CreateJenkins(&http.Client{}, url, username, token).Init()
	if err != nil {
		return nil, err
	}
	return &JenkinsClient{
		Jenkins: jenkins,
	}, nil
}

func (c JenkinsClient) GetJob(name string, delay time.Duration, retryCount int) bool {
	for i := 0; i < retryCount; i++ {
		_, err := c.Jenkins.GetJob(name)
		if err == nil {
			return true
		}
		log.Info("Job is currently doesn't exist", "name", name, "delay", delay, "attempts lasts", retryCount-i)
		time.Sleep(delay)
	}
	return false
}

func (c JenkinsClient) TriggerDeletionJob(branchName string, fromCommit string, appName string) error {
	jobName := fmt.Sprintf("%v/job/Delete-release-%v", appName, appName)
	log.Info("Trying to trigger jenkins job", "name", jobName)

	if c.GetJob(jobName, time.Second, 1) {
		_, err := c.Jenkins.BuildJob(jobName, map[string]string{
			"RELEASE_NAME": branchName,
			"COMMIT_ID":    fromCommit,
		})
		if err != nil {
			return errors.Wrap(err, "unable to build job")
		}

		return nil
	}

	return JobNotFoundError("deletion job not found")
}

func (c JenkinsClient) TriggerReleaseJob(branchName string, fromCommit string, appName string) error {
	jobName := fmt.Sprintf("%v/job/Create-release-%v", appName, appName)
	log.Info("Trying to trigger jenkins job", "name", jobName)

	if c.GetJob(jobName, time.Second, 60) {
		_, err := c.Jenkins.BuildJob(jobName, map[string]string{
			"RELEASE_NAME": branchName,
			"COMMIT_ID":    fromCommit,
		})
		return err
	}
	return errors.New(fmt.Sprintf("Couldn't trigger %v job", jobName))
}

func (c JenkinsClient) GetJobStatus(name string, delay time.Duration, retryCount int) (string, error) {
	time.Sleep(delay)
	for i := 0; i < retryCount; i++ {
		isQueued, qErr := c.IsJobQueued(name)
		isRunning, rErr := c.IsJobRunning(name)
		if qErr != nil || rErr != nil {
			job, err := c.Jenkins.GetJob(name)
			if err != nil {
				return "", errors.Wrap(err, "job not found")
			}
			if job.Raw.Color == "notbuilt" {
				log.Info("Job didn't start yet", "name", name, "delay", delay, "attempts lasts", retryCount-i)
				time.Sleep(delay)
				continue
			}
		}
		if (isRunning != nil && *isRunning) || (isQueued != nil && *isQueued) {
			log.Info("Job is running", "name", name, "delay", delay, "attempts lasts", retryCount-i)
			time.Sleep(delay)
		} else {
			job, err := c.Jenkins.GetJob(name)
			if err != nil {
				return "", err
			}

			return job.Raw.Color, nil
		}
	}

	return "", errors.Errorf("Job %v has not been finished after specified delay", name)
}

func (c JenkinsClient) IsJobQueued(name string) (*bool, error) {
	job, err := c.Jenkins.GetJob(name)
	if err != nil {
		return nil, err
	}

	isQueued, err := job.IsQueued()
	if err != nil {
		return nil, err
	}

	return &isQueued, nil
}

func (c JenkinsClient) IsJobRunning(name string) (*bool, error) {
	job, err := c.Jenkins.GetJob(name)
	if err != nil {
		return nil, err
	}

	isRunning, err := job.IsRunning()
	if err != nil {
		return nil, err
	}

	return &isRunning, nil
}

func GetJenkins(c client.Client, namespace string) (*jenkinsApi.Jenkins, error) {
	options := client.ListOptions{Namespace: namespace}
	jenkinsList := &jenkinsApi.JenkinsList{}

	err := c.List(context.TODO(), &options, jenkinsList)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get Jenkins CRs in namespace %v", namespace)
	}

	if len(jenkinsList.Items) == 0 {
		return nil, fmt.Errorf("jenkins installation is not found in namespace %v", namespace)
	}

	return &jenkinsList.Items[0], nil
}

func GetJenkinsCreds(client client.Client, jenkins jenkinsApi.Jenkins, namespace string) (string, string, error) {
	annotationKey := fmt.Sprintf("%v/%v", jenkinsOperatorSpec.EdpAnnotationsPrefix, jenkinsOperatorSpec.JenkinsTokenAnnotationSuffix)
	jenkinsTokenSecretName := jenkins.Annotations[annotationKey]

	jenkinsTokenSecret, err := util.GetSecretData(client, jenkinsTokenSecretName, namespace)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return "", "", errors.Wrapf(err, "Secret %v in not found", jenkinsTokenSecretName)
		}
		return "", "", errors.Wrapf(err, "Getting secret %v failed", jenkinsTokenSecretName)
	}
	return string(jenkinsTokenSecret.Data["password"]), string(jenkinsTokenSecret.Data["username"]), nil
}

func GetJenkinsUrl(jenkins jenkinsApi.Jenkins, namespace string) string {
	log.Info("creating Jenkins url")
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
