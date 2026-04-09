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

const awsAccessKeySecret = gcp.secretmanager.Secret.get(
    "picoclaw-aws-access-key-id",
    `projects/${project}/secrets/PICOCLAW_AWS_ACCESS_KEY_ID`,
);

const awsSecretKeySecret = gcp.secretmanager.Secret.get(
    "picoclaw-aws-secret-access-key",
    `projects/${project}/secrets/PICOCLAW_AWS_SECRET_ACCESS_KEY`,
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
    template: {
        template: {
            serviceAccount: sa.email,
            maxRetries: 0,
            timeout: "3600s",
            volumes: [{
                name: "workspace",
                gcs: {
                    bucket: bucket.name,
                    readOnly: false,
                },
            }],
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
                        { name: "AWS_REGION_NAME", value: awsRegion },
                        awsAccessKeyRef,
                        awsSecretKeyRef,
                    ],
                },
                // ── Main container: picoclaw ──────────────────────────────
                {
                    name: "picoclaw",
                    image: PICOCLAW_IMAGE,
                    volumeMounts: [{
                        name: "workspace",
                        mountPath: "/home/picoclaw/.picoclaw/workspace",
                    }],
                    resources: {
                        limits: {
                            memory: "4Gi",
                            cpu: "2000m",
                        },
                    },
                    envs: [
                        // JOB_TYPE: run-all | run | generate
                        { name: "JOB_TYPE", value: "run-all" },
                        // JOB_SPEC: only used when JOB_TYPE=run
                        { name: "JOB_SPEC", value: "" },
                        // JOB_PROMPT: only used when JOB_TYPE=generate
                        { name: "JOB_PROMPT", value: "" },
                        { name: "RESULTS_BUCKET", value: bucket.name },
                        { name: "LITELLM_BASE_URL", value: "http://0.0.0.0:4000" },
                        { name: "AWS_REGION_NAME", value: awsRegion },
                        awsAccessKeyRef,
                        awsSecretKeyRef,
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
--container=picoclaw --update-env-vars=JOB_TYPE=generate,JOB_PROMPT="<paste prompt here>"`;
