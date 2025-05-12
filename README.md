# Chaos CI Lib

[![Slack Channel](https://img.shields.io/badge/Slack-Join-purple)](https://slack.litmuschaos.io)
![GitHub Workflow](https://github.com/litmuschaos/chaos-ci-lib/actions/workflows/push.yml/badge.svg?branch=master)
[![Docker Pulls](https://img.shields.io/docker/pulls/litmuschaos/chaos-ci-lib.svg)](https://hub.docker.com/r/litmuschaos/chaos-ci-lib)
[![GitHub issues](https://img.shields.io/github/issues/litmuschaos/chaos-ci-lib)](https://github.com/litmuschaos/chaos-ci-lib/issues)
[![Twitter Follow](https://img.shields.io/twitter/follow/litmuschaos?style=social)](https://twitter.com/LitmusChaos)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/5297/badge)](https://bestpractices.coreinfrastructure.org/projects/5297)
[![Go Report Card](https://goreportcard.com/badge/github.com/litmuschaos/chaos-ci-lib)](https://goreportcard.com/report/github.com/litmuschaos/chaos-ci-lib)
[![YouTube Channel](https://img.shields.io/badge/YouTube-Subscribe-red)](https://www.youtube.com/channel/UCa57PMqmz_j0wnteRa9nCaw)
<br><br>

Chaos CI Lib is a central repository which contains different GO bdd tests implemented using the popular Ginkgo, Gomega test framework for running a number of litmuschaos experiments in different CI platforms that can be further used at remote places. The bdd can be used inside the job templates that can be used by the members who are using litmus experiments as part of their CI pipelines.

## Supported CI Platforms

Litmus supports CI plugin for the following CI platforms: 

<table style="width:50%">
  <tr>
    <th>CI Platform</th>
    <th>Chaos Template </th>
  </tr>
  <tr>
    <td>GitHub Actions</td>
    <td><a href="https://github.com/litmuschaos/github-chaos-actions">Click Here</a></td>
  </tr>  
  <tr>
    <td>GitLab Remote Templates</td>
    <td><a href="https://github.com/litmuschaos/gitlab-remote-templates">Click Here</a></td>
  </tr>
  <tr>
    <td>Spinnaker Plugin</td>
    <td><a href="https://github.com/litmuschaos/spinnaker-preconfigured-job-plugin">Click Here</a></td>
  </tr>
</table>

## Environment Variables

Chaos CI Lib uses standardized environment variables to configure environments, infrastructure, and probes. Below is the comprehensive list of supported environment variables.

### Litmus SDK & Authentication

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `LITMUS_ENDPOINT` | Litmus server endpoint URL | `""` | `https://chaos.example.com` |
| `LITMUS_USERNAME` | Username for Litmus authentication | `""` | `admin` |
| `LITMUS_PASSWORD` | Password for Litmus authentication | `""` | `litmus` |
| `LITMUS_PROJECT_ID` | ID of the Litmus project to use | `""` | `project-123` |

### Environment Management Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `CREATE_ENV` | Whether to create a new environment | `true` | `false` |
| `USE_EXISTING_ENV` | Whether to use an existing environment | `false` | `true` |
| `EXISTING_ENV_ID` | ID of existing environment (required if `USE_EXISTING_ENV=true`) | `""` | `env-123456` |
| `ENV_NAME` | Name for the new environment | `chaos-ci-env` | `my-k8s-env` |
| `ENV_TYPE` | Type of environment to create | `NON_PROD` | `PROD` |
| `ENV_DESCRIPTION` | Description of the environment | `CI Test Environment` | `Production Test Environment` |

### Infrastructure Management Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `INSTALL_INFRA` | Whether to install infrastructure | `true` | `false` |
| `USE_EXISTING_INFRA` | Whether to use existing infrastructure | `false` | `true` |
| `EXISTING_INFRA_ID` | ID of existing infrastructure (required if `USE_EXISTING_INFRA=true`) | `""` | `infra-123456` |
| `INFRA_NAME` | Name for the infrastructure | `ci-infra-{expName}` | `my-k8s-infra` |
| `INFRA_NAMESPACE` | Kubernetes namespace for infrastructure | `litmus` | `chaos-testing` |
| `INFRA_SCOPE` | Scope of infrastructure | `namespace` | `cluster` |
| `INFRA_SERVICE_ACCOUNT` | Service account for infrastructure | `litmus` | `chaos-runner` |
| `INFRA_DESCRIPTION` | Description of infrastructure | `CI Test Infrastructure` | `Production Test Infra` |
| `INFRA_PLATFORM_NAME` | Platform name | `others` | `gcp` |
| `INFRA_NS_EXISTS` | Whether namespace already exists | `false` | `true` |
| `INFRA_SA_EXISTS` | Whether service account already exists | `false` | `true` |
| `INFRA_SKIP_SSL` | Whether to skip SSL verification | `false` | `true` |
| `INFRA_NODE_SELECTOR` | Node selector for infrastructure | `""` | `disk=ssd` |
| `INFRA_TOLERATIONS` | Tolerations for infrastructure | `""` | `key=value:NoSchedule` |

### Probe Management Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `LITMUS_CREATE_PROBE` | Whether to create a probe | `false` | `true` |
| `LITMUS_PROBE_NAME` | Name of the probe | `http-probe` | `http-status-check` |
| `LITMUS_PROBE_TYPE` | Type of probe | `httpProbe` | `httpProbe` |
| `LITMUS_PROBE_MODE` | Mode of the probe | `SOT` | `Continuous` |
| `LITMUS_PROBE_URL` | URL for HTTP probe | `http://localhost:8080/health` | `http://app:8080/health` |
| `LITMUS_PROBE_TIMEOUT` | Timeout for probe | `30s` | `5s` |
| `LITMUS_PROBE_INTERVAL` | Interval for probe | `10s` | `5s` |
| `LITMUS_PROBE_ATTEMPTS` | Number of attempts for probe | `1` | `3` |
| `LITMUS_PROBE_RESPONSE_CODE` | Expected HTTP response code | `200` | `200` |

### Example Usage

To create a new environment and infrastructure:
```bash
# Authentication
export LITMUS_ENDPOINT="https://chaos.example.com"
export LITMUS_USERNAME="admin"
export LITMUS_PASSWORD="litmus"
export LITMUS_PROJECT_ID="project-123"

# Environment setup
export CREATE_ENV="true"
export ENV_NAME="test-environment"
export ENV_TYPE="NON_PROD"

# Infrastructure setup
export INSTALL_INFRA="true"
export INFRA_NAME="test-infra"
export INFRA_NAMESPACE="chaos-testing"
export INFRA_SCOPE="namespace"

# Optional probe setup
export LITMUS_CREATE_PROBE="true"
export LITMUS_PROBE_NAME="http-status-check"
export LITMUS_PROBE_TYPE="httpProbe"
export LITMUS_PROBE_URL="http://app:8080/health"
export LITMUS_PROBE_RESPONSE_CODE="200"
```

To use existing environment and infrastructure:
```bash
# Set environment variables for existing resources
export USE_EXISTING_ENV="true"
export EXISTING_ENV_ID="env-123456"
export USE_EXISTING_INFRA="true"
export EXISTING_INFRA_ID="infra-789012"
```

## How to get started?

Refer the [LitmusChaos Docs](https://docs.litmuschaos.io) and [Experiment Docs](https://litmuschaos.github.io/litmus/experiments/categories/contents/)

## How do I contribute?

You can contribute by raising issues, improving the documentation, contributing to the core framework and tooling, etc.

Head over to the [Contribution guide](CONTRIBUTING.md)
