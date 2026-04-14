import * as pulumi from "@pulumi/pulumi";
import * as gcp from "@pulumi/gcp";

// ── Config ────────────────────────────────────────────────────────────────────

const config = new pulumi.Config();
const gcpConfig = new pulumi.Config("gcp");

const project = gcpConfig.require("project");
const enterpriseAutomationProjectId = "enterprise-automation-352103";
const region = config.get("region") ?? "europe-west4";
const imageTag = config.get("imageTag") ?? "latest";
const imageName = config.get("imageName") ?? "picoclaw-e2e-testing";

const awsRegion = config.get("awsRegion") ?? "ap-southeast-1";

const LITELLM_MODEL = "bedrock/global.anthropic.claude-haiku-4-5-20251001-v1:0";

// ── Artifact Registry (local — we have full IAM here) ────────────────────────

// Docker repo for the picoclaw image — CI pipeline pushes here.
const picoRepo = new gcp.artifactregistry.Repository("picoclaw-repo", {
    repositoryId: "picoclaw",
    location: region,
    project: enterpriseAutomationProjectId,
    format: "DOCKER",
});

// Remote repo proxying ghcr.io — Cloud Run Jobs rejects non-AR/GCR/DockerHub images.
const ghcrRemoteRepo = new gcp.artifactregistry.Repository("ghcr-remote", {
    repositoryId: "ghcr-remote",
    location: region,
    project: enterpriseAutomationProjectId,
    format: "DOCKER",
    mode: "REMOTE_REPOSITORY",
    remoteRepositoryConfig: {
        dockerRepository: {
            customRepository: {
                uri: "https://ghcr.io",
            },
        },
    },
});

const PICOCLAW_IMAGE = pulumi.interpolate`${region}-docker.pkg.dev/enterprise-automation-352103/container-repo/${imageName}:${imageTag}`;
const LITELLM_IMAGE = pulumi.interpolate`${region}-docker.pkg.dev/enterprise-automation-352103/${ghcrRemoteRepo.repositoryId}/berriai/litellm:main-latest`;

// ── Service Account ───────────────────────────────────────────────────────────

const sa = new gcp.serviceaccount.Account("picoclaw-e2e-sa", {
    accountId: "picoclaw-e2e-job",
    displayName: "Picoclaw E2E Test Job",
    project,
});

// Allow Cloud Run to invoke this job
new gcp.projects.IAMMember("sa-run-invoker", {
    project,
    role: "roles/run.invoker",
    member: pulumi.interpolate`serviceAccount:${sa.email}`,
});

// Grant SA access to all secrets in the project (avoids per-secret setIamPolicy)
new gcp.projects.IAMMember("sa-secret-accessor", {
    project,
    role: "roles/secretmanager.secretAccessor",
    member: pulumi.interpolate`serviceAccount:${sa.email}`,
});

// ── Secret Manager (created by Pulumi — Pulumi owns them and can set IAM) ────
// Delete any manually-created versions of these secrets before running pulumi up.
// Populate values after pulumi up:
//   gcloud secrets versions add PICOCLAW_AWS_ACCESS_KEY_ID --data-file=- <<< "YOUR_KEY"
//   gcloud secrets versions add PICOCLAW_AWS_SECRET_ACCESS_KEY --data-file=- <<< "YOUR_SECRET"
//   gcloud secrets versions add PICOCLAW_AWS_REGION_NAME --data-file=- <<< "ap-southeast-1"
//   gcloud secrets versions add PICOCLAW_IMAP_HOST --data-file=- <<< "imap.example.com"
//   gcloud secrets versions add PICOCLAW_IMAP_USER --data-file=- <<< "user@example.com"
//   gcloud secrets versions add PICOCLAW_IMAP_PASSWORD --data-file=- <<< "YOUR_PASSWORD"
//   gcloud secrets versions add PICOCLAW_IMAP_PORT --data-file=- <<< "993"

// config.json — mounted as a file at /home/picoclaw/.picoclaw/config.json
// Populate after pulumi up:
//   sed 's|http://litellm:4000|http://localhost:4000|g' docker/data/config.json | \
//     gcloud secrets versions add picoclaw-config --project=PROJECT --data-file=-
const configSecret = gcp.secretmanager.Secret.get(
    "picoclaw-config",
    `projects/${project}/secrets/PICOCLAW_CONFIG_FILE`,
);

const awsAccessKeySecret = gcp.secretmanager.Secret.get(
    "picoclaw-aws-access-key-id",
    `projects/${project}/secrets/PICOCLAW_AWS_ACCESS_KEY_ID`,
);

const awsSecretKeySecret = gcp.secretmanager.Secret.get(
    "picoclaw-aws-secret-access-key",
    `projects/${project}/secrets/PICOCLAW_AWS_SECRET_ACCESS_KEY`,
);

const picoclawAwsRegionNameSecret = gcp.secretmanager.Secret.get(
    "picoclaw-aws-region-name",
    `projects/${project}/secrets/PICOCLAW_AWS_REGION_NAME`,
);

const imapHostSecret = gcp.secretmanager.Secret.get(
    "picoclaw-imap-host",
    `projects/${project}/secrets/PICOCLAW_IMAP_HOST`,
);

const imapUserSecret = gcp.secretmanager.Secret.get(
    "picoclaw-imap-user",
    `projects/${project}/secrets/PICOCLAW_IMAP_USER`,
);

const imapPasswordSecret = gcp.secretmanager.Secret.get(
    "picoclaw-imap-password",
    `projects/${project}/secrets/PICOCLAW_IMAP_PASSWORD`,
);

const imapPortSecret = gcp.secretmanager.Secret.get(
    "picoclaw-imap-port",
    `projects/${project}/secrets/PICOCLAW_IMAP_PORT`,
);

const awsAccessKeyRef = {
    name: "AWS_ACCESS_KEY_ID" as const,
    valueSource: {
        secretKeyRef: {
            secret: awsAccessKeySecret.secretId,
            version: "latest",
        },
    },
};

const awsSecretKeyRef = {
    name: "AWS_SECRET_ACCESS_KEY" as const,
    valueSource: {
        secretKeyRef: {
            secret: awsSecretKeySecret.secretId,
            version: "latest",
        },
    },
};

const awsRegionNameRef = {
    name: "AWS_REGION_NAME" as const,
    valueSource: {
        secretKeyRef: {
            secret: picoclawAwsRegionNameSecret.secretId,
            version: "latest",
        },
    },
};

const imapHostRef = {
    name: "IMAP_HOST" as const,
    valueSource: {
        secretKeyRef: {
            secret: imapHostSecret.secretId,
            version: "latest",
        },
    },
};

const imapUserRef = {
    name: "IMAP_USER" as const,
    valueSource: {
        secretKeyRef: {
            secret: imapUserSecret.secretId,
            version: "latest",
        },
    },
};

const imapPasswordRef = {
    name: "IMAP_PASSWORD" as const,
    valueSource: {
        secretKeyRef: {
            secret: imapPasswordSecret.secretId,
            version: "latest",
        },
    },
};

const imapPortRef = {
    name: "IMAP_PORT" as const,
    valueSource: {
        secretKeyRef: {
            secret: imapPortSecret.secretId,
            version: "latest",
        },
    },
};

// Grant SA read access to all AR repos in the project
new gcp.projects.IAMMember("sa-ar-reader", {
    project,
    role: "roles/artifactregistry.reader",
    member: pulumi.interpolate`serviceAccount:${sa.email}`,
});

// ── Storage Bucket ────────────────────────────────────────────────────────────

const bucket = new gcp.storage.Bucket("picoclaw-e2e-testing", {
    project,
    location: region,
    uniformBucketLevelAccess: true,
    labels: {
        "do-not-delete": "true",
    }
});

new gcp.storage.BucketIAMMember("sa-bucket-writer", {
    bucket: bucket.name,
    role: "roles/storage.objectAdmin",
    member: pulumi.interpolate`serviceAccount:${sa.email}`,
});

// ── Cloud Run Job ─────────────────────────────────────────────────────────────
//
// Two containers per task:
//   1. litellm  — LiteLLM proxy in front of AWS Bedrock (port 4000 on localhost)
//   2. picoclaw — runs entrypoint-job.sh; waits for LiteLLM before starting
//
// Control the job via JOB_TYPE env var at execution time:
//   run-all  (default) — run all Playwright tests in dependency order
//   run      + JOB_SPEC=tests/...spec.ts — run a single test file
//   generate + JOB_PROMPT="..."          — generate a test via the AI agent

const e2eJob = new gcp.cloudrunv2.Job("picoclaw-e2e-job", {
    name: "picoclaw-e2e",
    location: region,
    project,
    labels: {
        "do-not-delete": "true",
    },
    template: {
        template: {
            serviceAccount: sa.email,
            maxRetries: 0,
            timeout: "3600s",
            volumes: [
                {
                    name: "workspace",
                    gcs: {
                        bucket: bucket.name,
                        readOnly: false,
                    },
                },
                {
                    name: "config",
                    secret: {
                        secret: configSecret.secretId,
                        items: [{
                            version: "latest",
                            path: "config.json",
                        }],
                    },
                },
            ],
            containers: [
                // ── Sidecar: LiteLLM ──────────────────────────────────────
                {
                    name: "litellm",
                    image: LITELLM_IMAGE,
                    args: ["--model", LITELLM_MODEL, "--port", "4000"],
                    resources: {
                        limits: {
                            memory: "2Gi",
                            cpu: "1000m",
                        },
                    },
                    envs: [
                        awsRegionNameRef,
                        awsAccessKeyRef,
                        awsSecretKeyRef,
                    ],
                },
                // ── Main container: picoclaw ──────────────────────────────
                {
                    name: "picoclaw",
                    image: PICOCLAW_IMAGE,
                    volumeMounts: [
                        {
                            name: "workspace",
                            mountPath: "/home/picoclaw/.picoclaw/workspace",
                        },
                        {
                            name: "config",
                            mountPath: "/home/picoclaw/.picoclaw-config",
                        },
                    ],
                    resources: {
                        limits: {
                            memory: "4Gi",
                            cpu: "2000m",
                        },
                    },
                    envs: [
                        // JOB_TYPE: run-all | run | autofix | generate | prompt
                        { name: "JOB_TYPE", value: "run-all" },
                        // ENVIRONMENT: UAT | PREVIEW-PROD (maps to BASE_URL in entrypoint-job.sh)
                        { name: "ENVIRONMENT", value: "UAT" },
                        // JOB_SPEC: used when JOB_TYPE=run or autofix
                        { name: "JOB_SPEC", value: "" },
                        // used when JOB_TYPE=generate
                        { name: "JOB_AREA", value: "" },
                        { name: "JOB_TEST_FILE", value: "" },
                        { name: "JOB_STEPS", value: "" },
                        { name: "JOB_EXPECTED_RESULT", value: "" },
                        // used when JOB_TYPE=prompt
                        { name: "JOB_PROMPT", value: "" },
                        { name: "RESULTS_BUCKET", value: bucket.name },
                        { name: "LITELLM_BASE_URL", value: "http://0.0.0.0:4000" },
                        awsRegionNameRef,
                        awsAccessKeyRef,
                        awsSecretKeyRef,
                        imapHostRef,
                        imapUserRef,
                        imapPasswordRef,
                        imapPortRef,
                    ],
                },
            ],
        },
    },
});

// ── Outputs ───────────────────────────────────────────────────────────────────

export const bucketName = bucket.name;
export const jobName = e2eJob.name;
export const jobLocation = e2eJob.location;
export const serviceAccountEmail = sa.email;
export const picoRegistryUrl = pulumi.interpolate`${region}-docker.pkg.dev/${project}/${picoRepo.repositoryId}`;

// Note: --container=picoclaw is required for multi-container jobs.
// Without it, gcloud crashes with: 'NoneType' object has no attribute 'template'

export const runAllCommand = pulumi.interpolate
    `gcloud run jobs execute ${e2eJob.name} --region=${region} --project=${project} \
--container=picoclaw --update-env-vars=JOB_TYPE=run-all`;

export const runSingleTestCommand = pulumi.interpolate
    `gcloud run jobs execute ${e2eJob.name} --region=${region} --project=${project} \
--container=picoclaw --update-env-vars=JOB_TYPE=run,JOB_SPEC=tests/flow-designer/create-new-flow-model-node-parser.spec.ts`;

export const generateTestCommand = pulumi.interpolate
    `gcloud run jobs execute ${e2eJob.name} --region=${region} --project=${project} \
--container=picoclaw --update-env-vars=JOB_TYPE=generate --update-env-vars=JOB_AREA=flow-designer --update-env-vars=JOB_TEST_FILE=<test-file> --update-env-vars="JOB_STEPS=<steps>" --update-env-vars="JOB_EXPECTED_RESULT=<expected>"`;

export const autofixTestCommand = pulumi.interpolate
    `gcloud run jobs execute ${e2eJob.name} --region=${region} --project=${project} \
--container=picoclaw --update-env-vars=JOB_TYPE=autofix --update-env-vars=JOB_SPEC=tests/flow-designer/create-new-flow-custom-node.spec.ts`;
