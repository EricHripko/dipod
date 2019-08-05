#!/usr/bin/env bats

function cleanup {
    podman rmi -f docker.io/library/ubuntu:latest || true
    podman rmi -f docker.io/library/ubuntu:cosmic || true
    podman rmi -f dipod-test || true
}

@test "images: list by name and tag" {
    # Arrange
    cleanup
    podman pull docker.io/library/ubuntu:latest

    # Act
    run docker images --filter="reference=docker.io/library/ubuntu:latest" --format '{{json .}}'

    # Assert
    [[ "$status" -eq 0 ]]
    [[ "$(jq -r ".Repository" <<< $output)" == "ubuntu" ]]
    [[ "$(jq -r ".Tag" <<< $output)" == "latest" ]]
}

@test "images: list by name only" {
    # Arrange
    cleanup
    podman pull docker.io/library/ubuntu:latest
    podman pull docker.io/library/ubuntu:cosmic

    # Act
    run docker images --filter="reference=docker.io/library/ubuntu" --format '{{json .}}'
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
    [[ "$(jq -r ".Repository" <<< "${lines[0]}")" == "ubuntu" ]]
    [[ "$(jq -r ".Tag" <<< "${lines[0]}")" == "latest" ]]
    [[ "$(jq -r ".Repository" <<< "${lines[1]}")" == "ubuntu" ]]
    [[ "$(jq -r ".Tag" <<< "${lines[1]}")" == "cosmic" ]]
}

@test "images: list by label" {
    # Arrange
    label="dipod.is.awesome=yes"
    podman rmi dipod-test || true
    podman build \
        --label $label \
        --tag dipod-test \
        $BATS_TEST_DIRNAME/images-list-labels
    id=$(podman inspect dipod-test -f "{{.Id}}")

    # Act
    run docker images --filter="label=$label" --quiet

    # Assert
    [[ "$status" -eq 0 ]]
    [[ $id =~ ^$output ]]
}

@test "images: list with digests" {
    # Arrange
    cleanup
    podman pull docker.io/library/ubuntu:latest

    # Act
    run docker images --filter="reference=docker.io/library/ubuntu" --format '{{json .}}' --digests
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
    [[ ! -z "$(jq -r ".Digest" <<< $output)" ]]
}

@test "images: pull from DockerHub" {
    # Arrange/Act
    run docker pull ubuntu
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "docker.io/library/ubuntu:latest" ]]
}

@test "images: pull from another registry" {
    # Arrange/Act
    run docker pull quay.io/openshift-pipeline/buildah
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "quay.io/openshift-pipeline/buildah:latest" ]]
}

@test "images: pull by tag" {
    # Arrange/Act
    run docker pull ubuntu:cosmic
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "docker.io/library/ubuntu:cosmic" ]]
}

@test "images: pull failed" {
    # Arrange/Act
    run docker pull does_not_exist:probably
    echo $output

    # Assert
    [[ "$status" -ne 0 ]]
}
