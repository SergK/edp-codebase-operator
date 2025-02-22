package chain

import (
	"context"
	"fmt"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Cleaner struct {
	next   handler.CodebaseHandler
	client client.Client
}

func (h Cleaner) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase_name", c.Name)
	rLog.Info("start cleaning data...")
	if err := h.tryToClean(c); err != nil {
		setFailedFields(c, v1alpha1.CleanData, err.Error())
		return err
	}
	rLog.Info("end cleaning data...")
	return nextServeOrNil(h.next, c)
}

func (h Cleaner) tryToClean(c *v1alpha1.Codebase) error {
	s := fmt.Sprintf("repository-codebase-%v-temp", c.Name)
	if err := h.deleteSecret(s, c.Namespace); err != nil {
		return errors.Wrapf(err, "unable to delete secret %v", s)
	}
	wd := util.GetWorkDir(c.Name, c.Namespace)
	if err := deleteWorkDirectory(wd); err != nil {
		return err
	}
	return nil
}

func (h Cleaner) deleteSecret(secretName, namespace string) error {
	log.Info("start deleting secret", "name", secretName)
	if err := h.client.Delete(context.TODO(), &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
	}); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("secret doesn't exist. skip deleting", "name", secretName)
			return nil
		}
		return err
	}
	log.Info("end deleting secret", "name", secretName)
	return nil
}

func deleteWorkDirectory(dir string) error {
	if err := util.RemoveDirectory(dir); err != nil {
		return errors.Wrapf(err, "couldn't delete directory %v", dir)
	}
	log.Info("directory was cleaned", "path", dir)
	return nil
}
