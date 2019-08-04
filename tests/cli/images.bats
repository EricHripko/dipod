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
    output=$(docker images --filter="reference=docker.io/library/ubuntu:latest" --format '{{json .}}')
    echo $output

    # Assert
    [[ "$(jq -r ".Repository" <<< $output)" == "ubuntu" ]]
    [[ "$(jq -r ".Tag" <<< $output)" == "latest" ]]
}

@test "images: list images by name only" {
    # Arrange
    cleanup
    podman pull docker.io/library/ubuntu:latest
    podman pull docker.io/library/ubuntu:cosmic

    # Act
    output=$(docker images --filter="reference=docker.io/library/ubuntu" --format '{{json .}}')
    echo $output

    # Assert
    echo "$(head -1 <<< $output | jq -r ".Repository")"
    [[ "$(head -1 <<< $output | jq -r ".Repository")" == "ubuntu" ]]
    [[ "$(head -1 <<< $output | jq -r ".Tag")" == "latest" ]]
    [[ "$(tail -1 <<< $output | jq -r ".Repository")" == "ubuntu" ]]
    [[ "$(tail -1 <<< $output | jq -r ".Tag")" == "cosmic" ]]
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
    output=$(docker images --filter="label=$label" --quiet)
    echo $output

    # Assert
    [[ $id =~ ^$output ]]
}

@test "images: list images with digests" {
    # Arrange
    cleanup
    podman pull docker.io/library/ubuntu:latest

    # Act
    output=$(docker images --filter="reference=docker.io/library/ubuntu" --format '{{json .}}' --digests)
    echo $output

    # Assert
    [[ ! -z "$(jq -r ".Digest" <<< $output)" ]]
}
