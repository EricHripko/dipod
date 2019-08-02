#!/usr/bin/env bats

@test "system: version" {
    # Arrange/Act
    output=$(docker version --format '{{json .Server}}')
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
