# Campaign Lottery Platform

An original marketing campaign and lottery platform scaffold.

## Scope

- User session and guest login flow
- Campaign and prize pool modeling
- Lottery draw flow with stock and probability control
- Admin login and overview endpoints
- MySQL schema for users, campaigns, draws, and prize fulfillment

## Repository layout

- backend: standalone Go API service
- sql: MySQL initialization scripts

## Current status

- Independent Go API is available
- MySQL + Redis persistence is wired for sessions, campaigns, draws, and fulfillment tasks
- Admin CRUD endpoints are available for campaigns, prizes, and fulfillment task status updates
- Database schema and deployment assets are prepared for server rollout

## Admin API

- GET /api/v1/admin/campaigns
- POST /api/v1/admin/campaigns
- PUT /api/v1/admin/campaigns/{campaignID}
- DELETE /api/v1/admin/campaigns/{campaignID}
- GET /api/v1/admin/campaigns/{campaignID}/prizes
- POST /api/v1/admin/campaigns/{campaignID}/prizes
- PUT /api/v1/admin/prizes/{prizeID}
- DELETE /api/v1/admin/prizes/{prizeID}
- GET /api/v1/admin/fulfillment-tasks
- PATCH /api/v1/admin/fulfillment-tasks/{taskID}

## Deployment

- systemd unit: deploy/systemd/campaign-lottery-platform.service
- nginx vhost fragment: deploy/nginx/campaign-lottery-platform.conf
- release helper: scripts/release_82.sh