#!/usr/bin/env bats

function cleanup {
    podman rmi -f docker.io/library/ubuntu:latest || true
    podman rmi -f docker.io/library/ubuntu:cosmic || true
    podman rmi -f dipod-test || true
}

@test "images: list images by name and tag" {
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

@test "images: list images by name only" {
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

@test "images: list images by label" {
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

@test "images: list images with digests" {
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
