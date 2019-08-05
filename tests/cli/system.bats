#!/usr/bin/env bats

@test "system: version" {
    # Arrange/Act
    run docker version --format '{{json .Server}}'
    echo $output

    # Assert
    [[ "$(jq -r ".Version" <<< $output)" =~ dipod$ ]]
    [[ "$(jq -r ".ApiVersion" <<< $output)" == "1.26" ]]
    [[ "$(jq -r ".MinAPIVersion" <<< $output)" == "1.26" ]]
    [[ "$(jq -r ".Os" <<< $output)" == "linux" ]]
    [[ "$(jq -r ".Arch" <<< $output)" == "amd64" ]]

    component=$(jq -r ".Components[0]" <<< $output)
    [[ "$(jq -r ".Name" <<< $component)" == "Engine" ]]
    [[ "$(jq -r ".Version" <<< $component)" =~ dipod$ ]]
    [[ "$(jq -r ".Details.ApiVersion" <<< $component)" == "1.26" ]]
    [[ "$(jq -r ".Details.MinAPIVersion" <<< $component)" == "1.26" ]]
    [[ "$(jq -r ".Details.Os" <<< $component)" == "linux" ]]
    [[ "$(jq -r ".Details.Arch" <<< $component)" == "amd64" ]]
}

@test "system: info" {
    # Arrange/Act
    run docker system info --format '{{json .}}'
    echo $output

    # Assert
    [[ "$(jq -r ".Driver" <<< $output)" == "overlay" ]]
    [[ "$(jq -r ".MemoryLimit" <<< $output)" == "true" ]]
    [[ "$(jq -r ".SwapLimit" <<< $output)" == "true" ]]
    [[ "$(jq -r ".KernelMemory" <<< $output)" == "true" ]]
    [[ "$(jq -r ".CpuCfsPeriod" <<< $output)" == "true" ]]
    [[ "$(jq -r ".CpuCfsQuota" <<< $output)" == "true" ]]
    [[ "$(jq -r ".CPUShares" <<< $output)" == "true" ]]
    [[ "$(jq -r ".CPUSet" <<< $output)" == "true" ]]
    [[ "$(jq -r ".IPv4Forwarding" <<< $output)" == "true" ]]
    [[ "$(jq -r ".BridgeNfIptables" <<< $output)" == "true" ]]
    [[ "$(jq -r ".BridgeNfIp6tables" <<< $output)" == "true" ]]
    [[ "$(jq -r ".OomKillDisable" <<< $output)" == "true" ]]
    [[ "$(jq -r ".CgroupDriver" <<< $output)" == "podman" ]]
    [[ "$(jq -r ".OSType" <<< $output)" == "linux" ]]
    [[ "$(jq -r ".Architecture" <<< $output)" == "amd64" ]]
    [[ "$(jq -r ".DockerRootDir" <<< $output)" == "/var/run/containers/storage" ]]
}
