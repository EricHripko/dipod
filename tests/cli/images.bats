#!/usr/bin/env bats

function cleanup {
    podman rmi $(podman images --filter="reference=docker.io/library/ubuntu" -q) 
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
    [[ "$(head -1 <<< $output | jq -r ".Repository")" == "ubuntu" ]]
    [[ "$(head -1 <<< $output | jq -r ".Tag")" == "latest" ]]
    [[ "$(tail -1 <<< $output | jq -r ".Repository")" == "ubuntu" ]]
    [[ "$(tail -1 <<< $output | jq -r ".Tag")" == "cosmic" ]]
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
