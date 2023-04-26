# Kyma Environments Cleanup Job
**Important!**  
**The job should run only in the development environment. Make sure you are connected to the development Kubernetes cluster before applying the job.**

Kyma Environments Cleanup Job removes Kyma Environments which are older than 24h. The job is scheduled to run every day at midnight (according to the local time defined in the system).

Directory contents:

| File                               | Description                                                                                 |
|------------------------------------|---------------------------------------------------------------------------------------------|
| kyma-environments-cleanup-job.yaml | Kyma Environments Cleanup CronJob manifest. Should not be applied directly into the cluster |
| apply.sh                           | Shell script for applying the Kyma Environments Cleanup CronJob into the cluster            |

The manifest contains three placeholders for values which are set by the shell script:
- `$SCRIPT_BROKER_URL` 
- `$SCRIPT_DOMAIN`
- `$SCRIPT_CLOUDSQL_PROXY_COMMAND`

The values are derived from Kyma Environment Broker Deployment which should be running in the cluster prior to the CronJob application.

Run `apply.sh` script to apply the CronJob into the cluster.