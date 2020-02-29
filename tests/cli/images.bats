#!/usr/bin/env bats

function cleanup {
    podman system prune --force --all || true
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
    cleanup
    label="dipod.is.awesome=yes"
    podman build \
        --label $label \
        --tag dipod-test \
        $BATS_TEST_DIRNAME/images-list-labels
    id=$(podman inspect dipod-test --format "{{.Id}}")

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
    run docker pull quay.io/buildah/stable
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "quay.io/buildah/stable:latest" ]]
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

@test "images: inspect basic" {
    # Arrange
    podman pull docker.io/library/ubuntu

    # Act
    run docker inspect ubuntu
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
    [[ "$(jq -r ".[0].RepoTags[0]" <<< $output)" == "docker.io/library/ubuntu:latest" ]]
    [[ "$(jq -r ".[0].RepoDigests[0]" <<< $output)" =~ "docker.io/library/ubuntu@sha256:" ]]
    [[ "$(jq -r ".[0].Parent" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].Comment" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].Created" <<< $output)" != "" ]]
    [[ "$(jq -r ".[0].Container" <<< $output)" != "" ]]

    [[ "$(jq -r ".[0].ContainerConfig.Hostname" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].ContainerConfig.Domainname" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].ContainerConfig.User" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].ContainerConfig.AttachStdin" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].ContainerConfig.AttachStdout" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].ContainerConfig.AttachStderr" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].ContainerConfig.Tty" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].ContainerConfig.OpenStdin" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].ContainerConfig.StdinOnce" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].ContainerConfig.Env[0]" <<< $output)" == "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" ]]
    [[ "$(jq -r ".[0].ContainerConfig.Cmd[0]" <<< $output)" == "/bin/bash" ]]
    [[ "$(jq -r ".[0].ContainerConfig.ArgsEscaped" <<< $output)" == "true" ]]
    [[ "$(jq -r ".[0].ContainerConfig.Image" <<< $output)" != "" ]]
    [[ "$(jq -r ".[0].ContainerConfig.Volumes" <<< $output)" == "null" ]]
    [[ "$(jq -r ".[0].ContainerConfig.WorkingDir" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].ContainerConfig.Entrypoint" <<< $output)" == "null" ]]
    [[ "$(jq -r ".[0].ContainerConfig.OnBuild" <<< $output)" == "null" ]]

    [[ "$(jq -r ".[0].DockerVersion" <<< $output)" != "" ]]
    [[ "$(jq -r ".[0].Author" <<< $output)" == "" ]]

    [[ "$(jq -r ".[0].Config.Hostname" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].Config.Domainname" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].Config.User" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].Config.AttachStdin" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].Config.AttachStdout" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].Config.AttachStderr" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].Config.Tty" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].Config.OpenStdin" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].Config.StdinOnce" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].Config.Env[0]" <<< $output)" == "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" ]]
    [[ "$(jq -r ".[0].Config.Cmd[0]" <<< $output)" == "/bin/bash" ]]
    [[ "$(jq -r ".[0].Config.ArgsEscaped" <<< $output)" == "true" ]]
    [[ "$(jq -r ".[0].Config.Image" <<< $output)" != "" ]]
    [[ "$(jq -r ".[0].Config.Volumes" <<< $output)" == "null" ]]
    [[ "$(jq -r ".[0].Config.WorkingDir" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].Config.Entrypoint" <<< $output)" == "null" ]]
    [[ "$(jq -r ".[0].Config.OnBuild" <<< $output)" == "null" ]]

    [[ "$(jq -r ".[0].Architecture" <<< $output)" == "amd64" ]]
    [[ "$(jq -r ".[0].Os" <<< $output)" == "linux" ]]
    [[ "$(jq -r ".[0].Size" <<< $output)" != "0" ]]
    [[ "$(jq -r ".[0].VirtualSize" <<< $output)" != "0" ]]
    [[ "$(jq -r ".[0].GraphDriver.Name" <<< $output)" == "overlay" ]]
    [[ "$(jq -r ".[0].RootFS.Type" <<< $output)" == "layers" ]]
}

@test "images: inspect advanced" {
    # Arrange
    podman pull docker.io/library/node:carbon-onbuild

    # Act
    run docker inspect node:carbon-onbuild
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
    [[ "$(jq -r ".[0].RepoTags[0]" <<< $output)" == "docker.io/library/node:carbon-onbuild" ]]
    [[ "$(jq -r ".[0].RepoDigests[0]" <<< $output)" =~ "docker.io/library/node@sha256:" ]]
    [[ "$(jq -r ".[0].Parent" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].Comment" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].Created" <<< $output)" != "" ]]
    [[ "$(jq -r ".[0].Container" <<< $output)" != "" ]]

    [[ "$(jq -r ".[0].ContainerConfig.Hostname" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].ContainerConfig.Domainname" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].ContainerConfig.User" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].ContainerConfig.AttachStdin" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].ContainerConfig.AttachStdout" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].ContainerConfig.AttachStderr" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].ContainerConfig.Tty" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].ContainerConfig.OpenStdin" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].ContainerConfig.StdinOnce" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].ContainerConfig.Env[0]" <<< $output)" == "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" ]]
    [[ "$(jq -r ".[0].ContainerConfig.Env[1]" <<< $output)" =~ "NODE_VERSION=8" ]]
    [[ "$(jq -r ".[0].ContainerConfig.Env[2]" <<< $output)" =~ "YARN_VERSION=" ]]
    [[ "$(jq -r ".[0].ContainerConfig.Cmd[0]" <<< $output)" == "npm" ]]
    [[ "$(jq -r ".[0].ContainerConfig.Cmd[1]" <<< $output)" == "start" ]]
    [[ "$(jq -r ".[0].ContainerConfig.ArgsEscaped" <<< $output)" == "true" ]]
    [[ "$(jq -r ".[0].ContainerConfig.Image" <<< $output)" != "" ]]
    [[ "$(jq -r ".[0].ContainerConfig.Volumes" <<< $output)" == "null" ]]
    [[ "$(jq -r ".[0].ContainerConfig.WorkingDir" <<< $output)" == "/usr/src/app" ]]
    [[ "$(jq -r ".[0].ContainerConfig.Entrypoint[0]" <<< $output)" == "docker-entrypoint.sh" ]]

    [[ "$(jq -r ".[0].DockerVersion" <<< $output)" != "" ]]
    [[ "$(jq -r ".[0].Author" <<< $output)" == "" ]]

    [[ "$(jq -r ".[0].Config.Hostname" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].Config.Domainname" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].Config.User" <<< $output)" == "" ]]
    [[ "$(jq -r ".[0].Config.AttachStdin" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].Config.AttachStdout" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].Config.AttachStderr" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].Config.Tty" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].Config.OpenStdin" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].Config.StdinOnce" <<< $output)" == "false" ]]
    [[ "$(jq -r ".[0].Config.Env[0]" <<< $output)" == "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" ]]
    [[ "$(jq -r ".[0].Config.Env[1]" <<< $output)" =~ "NODE_VERSION=8" ]]
    [[ "$(jq -r ".[0].Config.Env[2]" <<< $output)" =~ "YARN_VERSION=" ]]
    [[ "$(jq -r ".[0].Config.Cmd[0]" <<< $output)" == "npm" ]]
    [[ "$(jq -r ".[0].Config.Cmd[1]" <<< $output)" == "start" ]]
    [[ "$(jq -r ".[0].Config.ArgsEscaped" <<< $output)" == "true" ]]
    [[ "$(jq -r ".[0].Config.Image" <<< $output)" != "" ]]
    [[ "$(jq -r ".[0].Config.Volumes" <<< $output)" == "null" ]]
    [[ "$(jq -r ".[0].Config.WorkingDir" <<< $output)" == "/usr/src/app" ]]
    [[ "$(jq -r ".[0].Config.Entrypoint[0]" <<< $output)" == "docker-entrypoint.sh" ]]

    [[ "$(jq -r ".[0].Architecture" <<< $output)" == "amd64" ]]
    [[ "$(jq -r ".[0].Os" <<< $output)" == "linux" ]]
    [[ "$(jq -r ".[0].Size" <<< $output)" != "0" ]]
    [[ "$(jq -r ".[0].VirtualSize" <<< $output)" != "0" ]]
    [[ "$(jq -r ".[0].GraphDriver.Name" <<< $output)" == "overlay" ]]
    [[ "$(jq -r ".[0].RootFS.Type" <<< $output)" == "layers" ]]
}

@test "images: inspect metadata" {
    # Arrange
    cleanup
    tag=dipod-inspect
    podman build --tag $tag $BATS_TEST_DIRNAME/images-inspect

    # Act
    run docker inspect dipod-inspect
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
    [[ "$(jq -r ".[0].RepoTags[0]" <<< $output)" == "localhost/dipod-inspect:latest" ]]
    [[ "$(jq -r ".[0].RepoDigests[0]" <<< $output)" =~ "localhost/dipod-inspect@sha256:" ]]

    [[ "$(jq -r ".[0].ContainerConfig.User" <<< $output)" == "dipod" ]]
    [[ "$(jq -r ".[0].ContainerConfig.StopSignal" <<< $output)" == "SIGKILL" ]]
    [[ "$(jq -cr ".[0].ContainerConfig.Volumes" <<< $output)" == '{"/data":{}}' ]]
    [[ "$(jq -cr ".[0].ContainerConfig.Labels" <<< $output)" == '{"dipod.is.awesome":"yes"}' ]]
    [[ "$(jq -cr ".[0].ContainerConfig.ExposedPorts" <<< $output)" == '{"80/tcp":{}}' ]]
}

@test "images: inspect not found" {
    # Arrange/Act
    run docker inspect does_not_exist:probably
    echo $output

    # Assert
    [[ "$status" -eq 1 ]]
}

@test "images: image history" {
    # Arrange
    podman pull docker.io/library/ubuntu

    # Act
    run docker history ubuntu --format "{{json .}}"
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
    [[ "${#lines[@]}" -gt 0 ]]
}

@test "images: image history fail" {
    # Arrange/Act
    run docker history does_not_exist:probably
    echo $output

    # Assert
    [[ "$status" -eq 1 ]]
}

@test "images: image tag name only" {
    # Arrange
    cleanup
    target=dipod-tag-test
    podman pull docker.io/library/ubuntu

    # Act
    run docker tag ubuntu $target
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
}

@test "images: image tag name with tag" {
    # Arrange
    cleanup
    target=dipod-tag-test:test
    podman pull docker.io/library/ubuntu

    # Act
    run docker tag ubuntu $target
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
}

@test "images: image tag source not found" {
    # Arrange
    cleanup
    target=dipod-tag-test

    # Act
    run docker tag does_not_exist:probably $target
    echo $output

    # Assert
    [[ "$status" -ne 0 ]]
}

@test "images: image remove" {
    # Arrange
    cleanup
    image=docker.io/library/ubuntu
    podman pull $image
    podman tag $image dipod-test

    # Act
    run docker rmi dipod-test
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
}

@test "images: image remove source not found" {
    # Arrange/Act
    run docker rmi does_not_exist:probably $target
    echo $output

    # Assert
    [[ "$status" -ne 0 ]]
}

@test "images: image search" {
    skip "DockerHub is flaky"

    # Arrange
    image=ubuntu

    # Act
    run docker search $image --format="{{json .}}"
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
    [[ "$(jq -r ".Name" <<< "${lines[0]}")" =~ "ubuntu" ]]
}

@test "images: image search by stars" {
    skip "DockerHub is flaky"

    # Arrange
    image=ubuntu
    stars=5000

    # Act
    run docker search $image --filter=stars=$stars --format="{{json .}}"
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
    [[ "$(jq -r ".Name" <<< "${lines[0]}")" =~ "ubuntu" ]]
    [[ "$(jq -r ".StarCount" <<< "${lines[0]}")" -gt $stars ]]
}

@test "images: image search by official status" {
    skip "DockerHub is flaky"

    # Arrange
    image=ubuntu

    # Act
    run docker search $image --filter=is-official=true --format="{{json .}}"
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
    [[ "$(jq -r ".Name" <<< "${lines[0]}")" =~ "ubuntu" ]]
    [[ "$(jq -r ".IsOfficial" <<< "${lines[0]}")" == "true" ]]
}

@test "images: image search by automation" {
    skip "DockerHub is flaky"

    # Arrange
    image=ubuntu

    # Act
    run docker search $image --filter=is-automated=true --format="{{json .}}"
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
    [[ "$(jq -r ".Name" <<< "${lines[0]}")" =~ "ubuntu" ]]
    [[ "$(jq -r ".IsAutomated" <<< "${lines[0]}")" == "true" ]]
}

@test "images: image save single" {
    # Arrange
    image=docker.io/ubuntu
    podman pull $image

    # Act
    run docker save $image --output ubuntu.tar
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
}

@test "images: image save multiple tags" {
    # Arrange
    image=docker.io/ubuntu
    image=docker.io/ubuntu
    podman pull $image:latest
    podman pull $image:xenial

    # Act
    run docker save $image:latest $image:xenial --output ubuntu.tar
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
}

@test "images: image save multiple images" {
    # Arrange
    image1=docker.io/ubuntu
    image2=docker.io/node
    podman pull $image1
    podman pull $image2

    # Act
    run docker save $image1 $image2 --output multiple.tar
    echo $output

    # Assert
    [[ "$status" -ne 0 ]]
}

@test "images: prune dangling" {
    # Arrange
    unused=docker.io/library/ubuntu
    podman pull $unused
    untagged=$(podman build $BATS_TEST_DIRNAME/images-list-labels --quiet)

    # Act
    run docker image prune --force
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
    [[ "$(podman images --filter="reference=$unused" --quiet)" != "" ]]
    [[ "$(podman images --filter="reference=$untagged" --quiet)" == "" ]]
}

@test "images: prune all" {
    # Arrange
    unused=docker.io/library/ubuntu
    podman pull $unused
    untagged=$(podman build $BATS_TEST_DIRNAME/images-list-labels --quiet)

    # Act
    run docker image prune --force --all
    echo $output

    # Assert
    [[ "$status" -eq 0 ]]
    [[ "$(podman images --filter="reference=$unused" --quiet)" == "" ]]
    [[ "$(podman images --filter="reference=$untagged" --quiet)" == "" ]]
}
