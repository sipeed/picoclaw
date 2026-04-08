# Pulumi GCP TypeScript Template

A minimal Google Cloud Storage bucket example using Pulumi and TypeScript. This template helps you get started quickly with a basic Pulumi program on GCP.

## Overview

This template provisions a Google Cloud Storage bucket in the `US` region and exports its URL. It demonstrates how to use the Pulumi GCP provider with TypeScript.

## Providers

- `@pulumi/pulumi`
- `@pulumi/gcp`

## Resources Created

- **Storage Bucket** (`gcp.storage.Bucket`)

## Outputs

- `bucketName` – The URL of the created Storage Bucket.

## When to Use

Use this template when you:
- Want a quick, minimal example of provisioning GCP resources with Pulumi.
- Are exploring Pulumi and TypeScript on Google Cloud.
- Need a starting point for building more complex GCP infrastructure in TypeScript.

## Prerequisites

- Node.js installed on your machine.
- Pulumi CLI installed.
- A Google Cloud project.
- GCP credentials configured (for example, via `gcloud auth login` or the `GOOGLE_APPLICATION_CREDENTIALS` environment variable).

## Getting Started

Create a new Pulumi project from this template:
```bash
pulumi new gcp-typescript
```
Follow the interactive prompts to set:
- Project name and description.
- `gcp:project` (the target Google Cloud project ID).

## Project Layout

```
.
├── Pulumi.yaml         # Pulumi project definition and template metadata
├── index.ts            # Entry point for the Pulumi program
├── package.json        # Node.js dependencies and metadata
└── tsconfig.json       # TypeScript compiler configuration
```

## Configuration

This template recognizes the following configuration values:

- `gcp:project` – The Google Cloud project where resources will be deployed.

Set this value in your stack with:
```bash
pulumi config set gcp:project YOUR_PROJECT_ID
```

## Next Steps

- Customize the storage bucket (e.g., change location, storage class, access policies).
- Add more GCP resources such as Compute Engine instances, Pub/Sub topics, or Firestore databases.
- Explore the full Pulumi GCP provider documentation:
  https://www.pulumi.com/docs/reference/pkg/gcp/
- Learn more about Pulumi with TypeScript:
  https://www.pulumi.com/docs/get-started/typescript/

## Getting Help

If you run into issues or have questions, check out:
- Pulumi Documentation: https://www.pulumi.com/docs/
- Community Slack: https://slack.pulumi.com/
- GitHub Issues: https://github.com/pulumi/pulumi/issues