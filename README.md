## AI middleware written in go

## find port blocked

lsof -iTCP:9000 -sTCP:LISTEN

## kill

kill [process id]

## generate AZURE_CREDENTIALS

az login
az account show --query id --output tsv
az ad sp create-for-rbac \
 --name "github-actions-ai-go-staging" \
 --role contributor \
 --scopes /subscriptions/YOUR_SUBSCRIPTION_ID \
 --sdk-auth

## before committing to develop branch

`make migration-files`

This generate the db migration files. If you are not intenting to merge to staging branch, no need to run this.


## grant permissions to the web user paix_web

-- Grant usage on the schema
GRANT USAGE ON SCHEMA public TO paix_web;

-- Grant select, insert, update, delete on all existing tables
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO paix_web;

-- Grant usage/select on all sequences (needed for serial/identity columns)
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO paix_web;

-- Ensure future tables also have these permissions
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO paix_web;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON SEQUENCES TO paix_web;

## deploy fc
if there is a change of fc_user in RAM, call 
`aliyun configure` with the latest ak and sk
use `aliyun configure list` to get list of users used

## https for localhost
brew install ngrok
ngrok config add-authtoken 3IOt9OL9Yep2k9AeK4oMseHiHHm_3hhgur73nfUDX4aHvknh2
ngrok http 9000