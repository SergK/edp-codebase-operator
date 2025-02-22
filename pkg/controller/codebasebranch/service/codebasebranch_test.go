package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/bndr/gojenkins"
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCodebaseBranchService_TriggerReleaseJob(t *testing.T) {
	cb := v1alpha1.CodebaseBranch{
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: "codebase",
			ReleaseJobParams: map[string]string{
				"codebaseName": "RELEASE_NAME",
				"fromCommit":   "COMMIT_ID",
				"gitServer":    "GIT_SERVER",
			},
		},
		Status: v1alpha1.CodebaseBranchStatus{
			Status: model.StatusInit,
		},
	}
	c := v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name: "codebase",
		},
	}
	secret := coreV1.Secret{}
	js := jenkinsApi.Jenkins{}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, &cb, &js, &jenkinsApi.JenkinsList{}, &c)
	cl := fake.NewClientBuilder().WithRuntimeObjects(&cb, &js, &secret, &c).Build()
	svc := CodebaseBranchServiceProvider{
		Client: cl,
	}

	httpmock.Activate()
	httpmock.RegisterResponder("GET", "http://jenkins.:8080/api/json",
		httpmock.NewStringResponder(200, ""))

	jrsp := gojenkins.JobResponse{}

	httpmock.RegisterResponder("GET", "http://jenkins.:8080/job/codebase/job/Create-release-codebase/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jrsp))
	httpmock.RegisterResponder("GET", "http://jenkins.:8080/crumbIssuer/api/json/api/json",
		httpmock.NewStringResponder(404, ""))

	buildRsp := httpmock.NewStringResponse(200, "")
	buildRsp.Header.Add("Location", "/1")

	httpmock.RegisterResponder("POST", "http://jenkins.:8080/job/codebase/job/Create-release-codebase/build",
		func(request *http.Request) (*http.Response, error) {
			return buildRsp, nil
		})

	if err := svc.TriggerReleaseJob(&cb); err != nil {
		t.Fatalf("%+v", err)
	}
}

func TestCodebaseBranchService_TriggerDeletionJob(t *testing.T) {
	cb := v1alpha1.CodebaseBranch{
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: "codebase",
		},
	}

	c := v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name: "codebase",
		},
	}
	secret := coreV1.Secret{}
	js := jenkinsApi.Jenkins{}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, &cb, &js, &jenkinsApi.JenkinsList{}, &c)
	cl := fake.NewClientBuilder().WithRuntimeObjects(&cb, &js, &secret, &c).Build()
	svc := CodebaseBranchServiceProvider{
		Client: cl,
	}

	httpmock.Activate()
	httpmock.RegisterResponder("GET", "http://jenkins.:8080/api/json",
		httpmock.NewStringResponder(200, ""))

	jrsp := gojenkins.JobResponse{
		InQueue: false,
		Color:   jenkinsJobSuccessStatus,
		LastBuild: gojenkins.JobBuild{
			Number: 10,
		},
	}
	brsp := gojenkins.BuildResponse{Building: false}

	httpmock.RegisterResponder("GET", "http://jenkins.:8080/job/codebase/job/Delete-release-codebase/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jrsp))
	httpmock.RegisterResponder("GET", "http://jenkins.:8080/job/codebase/job/Delete-release-codebase/10/api/json?depth=1",
		httpmock.NewJsonResponderOrPanic(200, &brsp))
	httpmock.RegisterResponder("GET", "http://jenkins.:8080/crumbIssuer/api/json/api/json",
		httpmock.NewStringResponder(404, ""))

	buildRsp := httpmock.NewStringResponse(200, "")
	buildRsp.Header.Add("Location", "/1")

	httpmock.RegisterResponder("POST", "http://jenkins.:8080/job/codebase/job/Delete-release-codebase/build",
		func(request *http.Request) (*http.Response, error) {
			return buildRsp, nil
		})

	if err := svc.TriggerDeletionJob(&cb); err != nil {
		t.Fatalf("%+v", err)
	}
}

func TestCodebaseBranchService_TriggerDeletionJobFailed(t *testing.T) {
	cb := v1alpha1.CodebaseBranch{
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: "codebase",
		},
	}

	c := v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name: "codebase",
		},
	}
	secret := coreV1.Secret{}
	js := jenkinsApi.Jenkins{}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, &cb, &js, &jenkinsApi.JenkinsList{}, &c)
	cl := fake.NewClientBuilder().WithRuntimeObjects(&cb, &js, &secret, &c).Build()
	svc := CodebaseBranchServiceProvider{
		Client: cl,
	}

	httpmock.Activate()
	httpmock.RegisterResponder("GET", "http://jenkins.:8080/api/json",
		httpmock.NewStringResponder(200, ""))

	jrsp := gojenkins.JobResponse{
		InQueue: false,
		Color:   "red",
		LastBuild: gojenkins.JobBuild{
			Number: 10,
		},
	}
	brsp := gojenkins.BuildResponse{Building: false}

	httpmock.RegisterResponder("GET", "http://jenkins.:8080/job/codebase/job/Delete-release-codebase/api/json",
		httpmock.NewJsonResponderOrPanic(200, &jrsp))
	httpmock.RegisterResponder("GET", "http://jenkins.:8080/job/codebase/job/Delete-release-codebase/10/api/json?depth=1",
		httpmock.NewJsonResponderOrPanic(200, &brsp))
	httpmock.RegisterResponder("GET", "http://jenkins.:8080/crumbIssuer/api/json/api/json",
		httpmock.NewStringResponder(404, ""))

	buildRsp := httpmock.NewStringResponse(200, "")
	buildRsp.Header.Add("Location", "/1")

	httpmock.RegisterResponder("POST", "http://jenkins.:8080/job/codebase/job/Delete-release-codebase/build",
		func(request *http.Request) (*http.Response, error) {
			return buildRsp, nil
		})

	err := svc.TriggerDeletionJob(&cb)
	assert.Error(t, err)
	if errors.Cause(err) != JobFailedError(err.Error()) {
		t.Fatal("wrong error returned")
	}
}

func TestCodebaseBranchServiceProvider_AppendVersionToTheHistorySlice(t *testing.T) {
	version := "0-0-1-SNAPSHOT"
	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			Version: &version,
		},
		Status: v1alpha1.CodebaseBranchStatus{
			VersionHistory: []string{"0-0-0-SNAPSHOT"},
		},
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, cb)
	fakeCl := fake.NewClientBuilder().WithRuntimeObjects(cb).Build()

	s := &CodebaseBranchServiceProvider{
		Client: fakeCl,
	}
	if err := s.AppendVersionToTheHistorySlice(cb); err != nil {
		t.Errorf("unexpected error")
	}

	cbResp := &v1alpha1.CodebaseBranch{}
	err := fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		cbResp)
	assert.NoError(t, err)
	assert.Equal(t, len(cbResp.Status.VersionHistory), 2)
	assert.Equal(t, cbResp.Status.VersionHistory[1], version)
}

func TestCodebaseBranchServiceProvider_ResetBranchBuildCounter(t *testing.T) {
	b := "100"
	zb := "0"
	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Status: v1alpha1.CodebaseBranchStatus{
			Build: &b,
		},
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, cb)
	fakeCl := fake.NewClientBuilder().WithRuntimeObjects(cb).Build()

	s := &CodebaseBranchServiceProvider{
		Client: fakeCl,
	}
	if err := s.ResetBranchBuildCounter(cb); err != nil {
		t.Errorf("unexpected error")
	}

	cbResp := &v1alpha1.CodebaseBranch{}
	err := fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		cbResp)
	assert.NoError(t, err)
	assert.Equal(t, cbResp.Status.Build, &zb)
}

func TestCodebaseBranchServiceProvider_ResetBranchSuccessBuildCounter(t *testing.T) {
	b := "100"
	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Status: v1alpha1.CodebaseBranchStatus{
			LastSuccessfulBuild: &b,
		},
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, cb)
	fakeCl := fake.NewClientBuilder().WithRuntimeObjects(cb).Build()

	s := &CodebaseBranchServiceProvider{
		Client: fakeCl,
	}
	if err := s.ResetBranchSuccessBuildCounter(cb); err != nil {
		t.Errorf("unexpected error")
	}

	cbResp := &v1alpha1.CodebaseBranch{}
	err := fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		cbResp)
	assert.NoError(t, err)
	assert.Nil(t, cbResp.Status.LastSuccessfulBuild)
}
