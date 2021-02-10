package chain

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jirafixversion/chain/handler"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("jira_fix_version_handler")

func CreateDefChain(jiraClient *jira.Client, client client.Client) handler.JiraFixVersionHandler {
	return PutFixVersion{
		next: SetFixVersion{
			client: *jiraClient,
			next: DeleteFixVersionCr{
				jc: nil,
				c:  client,
			},
		},
		client: *jiraClient,
	}
}

func nextServeOrNil(next handler.JiraFixVersionHandler, version *v1alpha1.JiraFixVersion) error {
	if next != nil {
		return next.ServeRequest(version)
	}
	log.Info("handling of JiraFixVersion has been finished", "name", version.Name)
	return nil
}
