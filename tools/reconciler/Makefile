NAME = reconciler
NAMESPACE = reconciler

.PHONY: deploy-reconciler
deploy-reconciler:
	kubectl create namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	helm template $(NAME) --namespace $(NAMESPACE) --set global.mothership_reconciler.enabled=true \
		--set global.mothership_reconciler.authentication=false --set global.mothership_reconciler.expose=false \
		--set global.component_reconcilers.enabled=true --set global.component_reconcilers.authentication=false \
		--set compass.enabled=false --set mothership-reconciler.db.encryptionKey=5bc19d3a2032fb8795cd86e08b473a351631505d2522991266b1fb85f89bad5f \
		--set mothership-reconciler.db.serviceHost=$(NAME)-postgresql --set mothership-reconciler.db.reconcilerUsername=postgres \
		--set mothership-reconciler.db.reconcilerPassword=test --set mothership-reconciler.options.verbose=true \
		--set postgresql.postgresqlPassword=test --set postgresql.persistence.enabled=true \
		--set global.mothership_reconciler.auditlog.useTestConfig=true --set global.mothership_reconciler.auditlog.persistence.enabled=false \
		--set postgresql.initdbScriptsConfigMap=$(NAME)-postgresql-db-init \
		--set migratorJobs.enabled=false ../../resources/kcp > reconciler.yaml
	kubectl apply -f reconciler.yaml
	rm reconciler.yaml
