package jiraserver

import (
	"context"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/adapter"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/dto"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jiraserver/chain"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const statusError = "error"

func NewReconcileJiraServer(client client.Client, scheme *runtime.Scheme, log logr.Logger) *ReconcileJiraServer {
	return &ReconcileJiraServer{
		client: client,
		scheme: scheme,
		log:    log.WithName("jira-server"),
	}
}

type ReconcileJiraServer struct {
	client client.Client
	scheme *runtime.Scheme
	log    logr.Logger
}

func (r *ReconcileJiraServer) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject := e.ObjectOld.(*codebaseApi.JiraServer)
			newObject := e.ObjectNew.(*codebaseApi.JiraServer)
			if oldObject.Status != newObject.Status {
				return false
			}
			return true
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.JiraServer{}, builder.WithPredicates(p)).
		Complete(r)
}

func (r *ReconcileJiraServer) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.V(2).Info("Reconciling JiraServer")

	i := &codebaseApi.JiraServer{}
	if err := r.client.Get(ctx, request.NamespacedName, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	defer r.updateStatus(ctx, i)

	c, err := r.initJiraClient(*i)
	if err != nil {
		i.Status.Available = false
		return reconcile.Result{}, err
	}

	jiraHandler := chain.CreateDefChain(c, r.client)
	if err := jiraHandler.ServeRequest(i); err != nil {
		i.Status.Status = statusError
		i.Status.DetailedMessage = err.Error()
		return reconcile.Result{}, err
	}
	log.Info("Reconciling JiraServer has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileJiraServer) updateStatus(ctx context.Context, instance *codebaseApi.JiraServer) {
	instance.Status.LastTimeUpdated = time.Now()
	err := r.client.Status().Update(ctx, instance)
	if err != nil {
		_ = r.client.Update(ctx, instance)
	}
}

func (r *ReconcileJiraServer) initJiraClient(jira codebaseApi.JiraServer) (jira.Client, error) {
	s, err := util.GetSecretData(r.client, jira.Spec.CredentialName, jira.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't get secret %v", jira.Spec.CredentialName)
	}
	user := string(s.Data["username"])
	pwd := string(s.Data["password"])
	c, err := new(adapter.GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer(jira.Spec.ApiUrl, user, pwd))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create Jira client")
	}
	return c, nil
}
