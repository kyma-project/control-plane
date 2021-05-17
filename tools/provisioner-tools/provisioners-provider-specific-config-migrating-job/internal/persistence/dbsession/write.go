package dbsession

import (
	"database/sql"
	"fmt"
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/persistence/dberrors"

	"github.com/gocraft/dbr/v2"
)

type writeSession struct {
	session     *dbr.Session
	transaction *dbr.Tx
}

func (ws writeSession) UpdateProviderSpecificConfig(id string, providerSpecificConfig string) dberrors.Error {
	res, err := ws.update("gardener_config").
		Where(dbr.Eq("Id", id)).
		Set("provider_specific_config", providerSpecificConfig).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to update provider_specific_config for gardener shoot cluster '%s': %s", id, err)
	}

	return ws.updateSucceeded(res, fmt.Sprintf("Failed to update provider_specific_config for gardener shoot cluster '%s' state: %s", id, err))
}

func (ws writeSession) RollbackUnlessCommitted() {
	ws.transaction.RollbackUnlessCommitted()
}

func (ws writeSession) Commit() dberrors.Error {
	err := ws.transaction.Commit()
	if err != nil {
		return dberrors.Internal("Failed to commit transaction: %s", err)
	}

	return nil
}

func (ws writeSession) update(table string) *dbr.UpdateStmt {
	if ws.transaction != nil {
		return ws.transaction.Update(table)
	}

	return ws.session.Update(table)
}

func (ws writeSession) updateSucceeded(result sql.Result, errorMsg string) dberrors.Error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dberrors.Internal("Failed to get number of rows affected: %s", err)
	}

	if rowsAffected == 0 {
		return dberrors.NotFound(errorMsg)
	}

	return nil
}
