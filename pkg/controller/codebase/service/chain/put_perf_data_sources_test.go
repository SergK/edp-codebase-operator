package chain

import (
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1alpha1"
	perfApi "github.com/epam/edp-perf-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	fakeName = "fake-name"
)

func TestPutPerfDataSourcesChain_SkipCreatingPerfDataSource(t *testing.T) {
	sources := PutPerfDataSources{
		client: nil,
	}
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-name",
		},
		Spec: v1alpha1.CodebaseSpec{},
	}
	assert.NoError(t, sources.ServeRequest(c))
}

func TestPutPerfDataSourcesChain_JenkinsAndSonarDataSourcesShouldBeCreated(t *testing.T) {
	pdss := &perfApi.PerfDataSourceSonar{}
	pdsj := &perfApi.PerfDataSourceJenkins{}
	pdsg := &perfApi.PerfDataSourceGitLab{}
	ecJenkins := &edpCompApi.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jenkins",
			Namespace: fakeNamespace,
		},
		Spec: edpCompApi.EDPComponentSpec{},
	}

	ecSonar := &edpCompApi.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sonar",
			Namespace: fakeNamespace,
		},
		Spec: edpCompApi.EDPComponentSpec{},
	}

	gs := &v1alpha1.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.GitServerSpec{
			GitHost: fakeName,
		},
	}

	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			DefaultBranch: fakeName,
			GitUrlPath:    util.GetStringP("/fake"),
			GitServer:     fakeName,
			Perf: &v1alpha1.Perf{
				Name:        fakeName,
				DataSources: []string{"Jenkins", "Sonar", "GitLab"},
			},
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, gs)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, pdsj, pdss, pdsg, ecJenkins, ecSonar)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(pdsj, pdss, pdsg, ecJenkins, ecSonar, gs).Build()

	assert.NoError(t, PutPerfDataSources{client: fakeCl}.ServeRequest(c))
}

func TestPutPerfDataSourcesChain_ShouldNotFoundEdpComponent(t *testing.T) {

	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Perf: &v1alpha1.Perf{
				Name:        fakeName,
				DataSources: []string{"Jenkins", "Sonar"},
			},
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	assert.Error(t, PutPerfDataSources{client: fakeCl}.ServeRequest(c))
}
