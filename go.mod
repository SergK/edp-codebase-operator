module github.com/epam/edp-codebase-operator/v2

go 1.14

replace (
	git.apache.org/thrift.git => github.com/apache/thrift v0.12.0
	github.com/kubernetes-incubator/reference-docs => github.com/kubernetes-sigs/reference-docs v0.0.0-20170929004150-fcf65347b256
	github.com/markbates/inflect => github.com/markbates/inflect v1.0.4
	github.com/openshift/api => github.com/openshift/api v0.0.0-20210416130433-86964261530c
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47
	k8s.io/api => k8s.io/api v0.20.7-rc.0
)

require (
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/andygrunwald/go-jira v1.12.0
	github.com/bndr/gojenkins v0.2.1-0.20181125150310-de43c03cf849
	github.com/dchest/uniuri v0.0.0-20160212164326-8902c56451e9
	github.com/epam/edp-cd-pipeline-operator/v2 v2.3.0-58.0.20210726142624-e26cea43163f
	github.com/epam/edp-common v0.0.0-20211025102907-fa4104d4d65f // indirect
	github.com/epam/edp-component-operator v0.1.1-0.20210712140516-09b8bb3a4cff
	github.com/epam/edp-jenkins-operator/v2 v2.3.0-130.0.20210719110425-d2d190f7bff9
	github.com/epam/edp-perf-operator/v2 v2.0.0-20210719113600-816c452ccbb0
	github.com/go-git/go-billy/v5 v5.3.1
	github.com/go-git/go-git/v5 v5.4.3-0.20210630082519-b4368b2a2ca4
	github.com/go-logr/logr v0.4.0
	github.com/go-openapi/spec v0.19.5
	github.com/jarcoal/httpmock v1.0.8
	github.com/lib/pq v1.8.0
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/trivago/tgo v1.0.1
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b
	gopkg.in/resty.v1 v1.12.0
	gopkg.in/src-d/go-git.v4 v4.10.0
	k8s.io/api v0.21.0-rc.0
	k8s.io/apimachinery v0.21.0-rc.0
	k8s.io/client-go v0.20.2
	k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7
	sigs.k8s.io/controller-runtime v0.8.3
)
