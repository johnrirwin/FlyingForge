# NAT Gateway Cost Optimization Runbook

This repository now uses a split pattern for feed ingestion vs feed serving:

- API/web ECS services serve cached feed data only.
- A dedicated scheduled one-shot ECS task refreshes feed data once daily.

This keeps core AWS service traffic off NAT where possible and removes unnecessary repeated internet egress.

## What changed

- Removed manual refresh button from the news UI.
- `/api/refresh` is disabled by default (`ENABLE_MANUAL_REFRESH=false`).
- Removed feed prefetch-on-startup for the API server process.
- Added daily scheduled refresh at 4:00 PM in `America/New_York`:
  - Terraform var: `news_refresh_schedule_expression = "cron(0 16 * * ? *)"`
  - Terraform var: `news_refresh_schedule_timezone = "America/New_York"`
- Added scheduled ECS task definition:
  - Container command: `./flyingforge -refresh-once`
  - Uses Redis cache backend to publish refreshed feed payloads.
- Added VPC endpoints so AWS service calls bypass NAT:
  - S3 gateway endpoint (for ECR layer bucket)
  - Interface endpoints: ECR API, ECR DKR, CloudWatch Logs, Secrets Manager, STS, Rekognition, optional KMS

## Deploy

```bash
cd terraform
terraform plan \
  -var="google_client_id=<redacted>" \
  -var="google_client_secret=<redacted>" \
  -var="encryption_key=<redacted>"
terraform apply
```

## Verify

1. Confirm endpoints exist:

```bash
aws ec2 describe-vpc-endpoints \
  --filters "Name=vpc-id,Values=<vpc-id>" \
  --query "VpcEndpoints[].{Id:VpcEndpointId,Service:ServiceName,State:State}"
```

2. Confirm scheduler exists:

```bash
aws scheduler get-schedule --name flyingforge-news-refresh-4pm
```

3. Force ECS deployments:

```bash
aws ecs update-service --cluster flyingforge-cluster --service flyingforge-server --force-new-deployment
aws ecs update-service --cluster flyingforge-cluster --service flyingforge-web --force-new-deployment
```

4. Validate runtime behavior:
   - API and web tasks start normally.
   - News endpoint returns cached items.
   - Scheduled refresh logs appear in `/ecs/flyingforge-news-refresh`.
   - CloudWatch app logs still appear in `/ecs/flyingforge-server` and `/ecs/flyingforge-web`.

5. Cost checks (24-72h):
   - `NatGateway-Bytes` should trend down.
   - `NatGateway-Hours` remains until NAT architecture changes.
   - `VpcEndpoint-Hours` and `VpcEndpoint-Bytes` appear.

## Rollback

1. Re-enable manual refresh path if needed:

```hcl
# task env:
ENABLE_MANUAL_REFRESH=true
```

2. Disable scheduled refresh if needed:

```hcl
enable_scheduled_news_refresh = false
```

3. Disable endpoint creation if needed:

```hcl
enable_vpc_endpoints = false
```

4. Apply Terraform and redeploy ECS services.
