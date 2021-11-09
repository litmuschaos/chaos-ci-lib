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

## How to get started?

Refer the [LitmusChaos Docs](https://docs.litmuschaos.io) and [Experiment Docs](https://litmuschaos.github.io/litmus/experiments/categories/contents/)

## How do I contribute?

You can contribute by raising issues, improving the documentation, contributing to the core framework and tooling, etc.

Head over to the [Contribution guide](CONTRIBUTING.md)
