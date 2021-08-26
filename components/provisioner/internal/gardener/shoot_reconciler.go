package gardener

import (
	"context"

	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/util/retry"

	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewReconciler(
	mgr ctrl.Manager,
	dbsFactory dbsession.Factory,
	auditLogConfigurator AuditLogConfigurator) *Reconciler {
	return &Reconciler{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		log:    logrus.WithField("Component", "ShootReconciler"),

		dbsFactory:           dbsFactory,
		auditLogConfigurator: auditLogConfigurator,
	}
}

type Reconciler struct {
	client     client.Client
	scheme     *runtime.Scheme
	dbsFactory dbsession.Factory

	log *logrus.Entry

	auditLogConfigurator AuditLogConfigurator
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithField("Shoot", req.NamespacedName)
	log.Infof("Reconciling Shoot")

	var shoot gardener_types.Shoot
	if err := r.client.Get(ctx, req.NamespacedName, &shoot); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		log.Errorf("Unable to get shoot: %s", err)
		return ctrl.Result{}, err
	}

	shouldReconcile, err := r.shouldReconcileShoot(shoot)
	if err != nil {
		log.Errorf("Failed to verify if shoot should be reconciled: %s", err.Error())
		return ctrl.Result{}, err
	}
	if !shouldReconcile {
		log.Debugf("Gardener cluster not found in database, shoot will be ignored")
		return ctrl.Result{}, nil
	}
	runtimeId := getRuntimeId(shoot)
	log = log.WithField("RuntimeId", runtimeId)

	seedName := getSeedName(shoot)

	if r.auditLogConfigurator.CanEnableAuditLogsForShoot(seedName) {
		err := r.enableAuditLogs(log, &shoot, seedName)
		if err != nil {
			log.Errorf("Failed to enable audit logs for %s shoot: %s", shoot.Name, err.Error())
		}
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) shouldReconcileShoot(shoot gardener_types.Shoot) (bool, error) {
	session := r.dbsFactory.NewReadSession()

	_, err := session.GetGardenerClusterByName(shoot.Name)
	if err != nil {
		if err.Code() == dberrors.CodeNotFound {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (r *Reconciler) updateShoot(modifiedShoot *gardener_types.Shoot) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		return r.client.Update(context.Background(), modifiedShoot)
	})
}

func (r *Reconciler) enableAuditLogs(logger logrus.FieldLogger, shoot *gardener_types.Shoot, seedName string) error {
	logger.Info("Enabling audit logs")

	seedKey := types.NamespacedName{Name: seedName, Namespace: ""}

	var seed gardener_types.Seed
	if err := r.client.Get(context.Background(), seedKey, &seed); err != nil {
		logger.Errorf("Cannot get %s seed: %s", seedName, err.Error())
		return err
	}

	annotated, err := r.auditLogConfigurator.SetAuditLogAnnotation(shoot, seed)
	if err != nil {
		logger.Errorf("Cannot enable audit logs: %s", err.Error())
		return nil
	}
	if !annotated {
		logger.Debugf("Audit Log Tenant did not change, skipping update of cluster")
		return nil
	}

	logger.Infof("Modifying Audit Log Tenant")
	if err := r.updateShoot(shoot); err != nil {
		logger.Errorf("Failed to update shoot: %s", err.Error())
		return err
	}
	return nil
}

func getSeedName(shoot gardener_types.Shoot) string {
	if shoot.Spec.SeedName != nil {
		return *shoot.Spec.SeedName
	}

	return ""
}
