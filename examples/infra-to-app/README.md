# Infrastructure to Application (GitLab CI/CD)

Provision a database, deploy an application, verify it works — all from one pipeline managed by formae.

## What You Get

- Deploy pipeline (`.gitlab-ci.yml` — provisions Azure PostgreSQL, deploys Miniflux, verifies feeds)
- Destroy pipeline (tears down container + database)
- Azure OIDC variables for authentication (masked)
- PostgreSQL password variable (masked)
- Project configuration variables

## Prerequisites

1. formae CLI installed with the GitLab and Azure plugins
2. GitLab project with CI/CD enabled
3. Azure subscription with OIDC configured for GitLab CI/CD
4. ACR (Azure Container Registry) with Miniflux image

## Environment Variables

Export these before running `formae apply`:

```bash
export AZURE_CLIENT_ID=<your-azure-client-id>
export AZURE_TENANT_ID=<your-azure-tenant-id>
export AZURE_SUBSCRIPTION_ID=<your-azure-subscription-id>
export ACR_PASSWORD=<your-acr-password>
# Optional — defaults to ChangeMe123!
export POSTGRES_PASSWORD=<your-postgres-password>
# Required — GitLab auth
export GITLAB_TOKEN=<your-gitlab-pat>
```

## Deploy

```bash
cd examples/infra-to-app
formae apply --mode reconcile --yes main.pkl
```

Then trigger the deploy pipeline in the GitLab UI (Run pipeline on main).

## Tear Down

Trigger the destroy pipeline in the GitLab UI.

To remove the GitLab resources:

```bash
formae destroy --query='stack:infra-to-app'
```
