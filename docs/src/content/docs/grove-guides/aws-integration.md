---
title: AWS Integration and Authentication
description: A guide to using AWS SSO authentication with your Grove development environment.
---

Grove can handle AWS authentication for you, making it easy to work with AWS services from your development environment without manually managing credentials.

:::tip
Grove's AWS integration uses the official [AWS SDK for Go](https://aws.github.io/aws-sdk-go-v2/docs/) with no external dependencies. Authentication is session-only and always prompts for fresh credentials!
:::

## Setting up AWS SSO

Before using Grove's AWS integration, you'll need AWS SSO configured:

1. **Configure AWS CLI**: Make sure you have [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) installed and configured.
2. **Set up SSO**: Configure your AWS SSO settings in `~/.aws/config` ([AWS SSO configuration guide](https://docs.aws.amazon.com/cli/latest/userguide/sso-configure-profile-token.html)).
3. **Test authentication**: Verify you can authenticate with `aws sso login`.

Your `~/.aws/config` should look something like:

```ini
[default]
sso_start_url = https://your-org.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = DeveloperAccess
region = us-east-1
```

## Using AWS authentication with Grove

To enter your Grove environment with AWS authentication:

```bash
kanuka grove enter --auth
```

This will:

- Start your Grove development environment.
- Authenticate you with AWS SSO if needed.
- Set up AWS credentials for your session.
- Show authentication status and expiration time.

## Re-authenticating when credentials expire

When your AWS credentials expire, you can re-authenticate:

```bash
# Re-enter with fresh authentication
kanuka grove enter --auth

# Or authenticate without entering the environment
aws sso login
```

## Using AWS services in your environment

Once authenticated, you can use AWS services normally:

```bash
# Inside your Grove environment with --auth
aws s3 ls
aws ec2 describe-instances
aws lambda list-functions
```

## Session-only authentication

Grove's AWS authentication is session-only, meaning:

- You'll always be prompted to authenticate when using `--auth`.
- Authentication only lasts for the current shell session.
- Environment variables are cleared when entering the Grove shell.
- No persistent credentials are stored.

## Troubleshooting AWS integration

Common issues and solutions:

**"SSO session not found"**: Run `aws sso login` first.

**"Credentials expired"**: Re-run `kanuka grove enter --auth` or `aws sso login`.

**"Profile not found"**: Check your `~/.aws/config` file configuration.

**"Permission denied"**: Verify your SSO role has the necessary permissions.

## Security considerations

Grove's AWS integration:

- Never stores long-term credentials.
- Uses temporary tokens that expire automatically.
- Respects your existing AWS CLI configuration.
- Works with your organization's SSO policies.

## Next steps

To learn more about Grove's AWS integration, see the [development environments concepts](/concepts/grove-environments) and the [command reference](/reference/references).

Or, continue reading to learn about other Kānuka features.
