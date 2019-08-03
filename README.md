# dipod
[![Build Status](https://travis-ci.org/EricHripko/dipod.svg?branch=master)](https://travis-ci.org/EricHripko/dipod)
[![Go Report](https://goreportcard.com/badge/github.com/EricHripko/dipod)](https://goreportcard.com/report/github.com/EricHripko/dipod)

`dipod` is a Docker Interface for PODman. Why? Because it's easier to proxy than to rewrite Docker-reliant software.
`dipod` also eases migration of workflows and CI pipelines away from Docker.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Build from Source](#build-from-source)
- [Running Tests](#running-tests)

## Prerequisites
- [Go v1.12 or later](https://golang.org/doc/install) to build
- [varlink-enabled podman](https://github.com/containers/libpod/blob/master/install.md) to run the backend
- `docker-cli` to interact with proxy
- `bats` and `jq` to run tests

## Build from Source
- Clone the repo:  
  `git clone https://github.com/EricHripko/dipod.git`
- Go inside:  
  `cd dipod`
- Build:
  `go build cmd/dipod/dipod.go`
- Start the proxy:  
  `./dipod`

## Run Tests
- Ensure that `dipod` is started
- Run CLI tests:
  `bats tests/cli`
