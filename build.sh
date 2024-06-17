#!/usr/bin/env bash

package_name="fastcig-client"
platforms=("darwin/arm64" "darwin/amd64" "linux/386" "linux/amd64" "windows/386" "windows/amd64")

mkdir -p build
rm -rf build/*

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    output_name=$package_name'-'$GOOS'-'$GOARCH
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi
    echo "building " $output_name

    env GOOS=$GOOS GOARCH=$GOARCH go build -o $output_name app
    if [ $? -ne 0 ]; then
        echo 'An error has occurred! Aborting the script execution...'
        exit 1
    fi
    mv $output_name build/
done

cd build

shasum -a 256 * > checksum.txt
