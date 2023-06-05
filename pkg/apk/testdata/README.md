# testdata

This directory contains useful artifacts for tests.

Notably:

* `APKINDEX.tar.gz` - a copy of `https://dl-cdn.alpinelinux.org/alpine/v3.16/main/aarch64/APKINDEX.tar.gz`.
* `alpine-baselayout-3.2.0.-r23.apk` - a copy of that apk from `https://dl-cdn.alpinelinux.org/alpine/v3.16/main/aarch64/alpine-baselayout-3.2.0.-r23.apk`. It should not be read, only used to validate bytes.
* `cache/APKINDEX.tar.gz` - `https://dl-cdn.alpinelinux.org/alpine/v3.17/main/aarch64/APKINDEX.tar.gz`. Note that it is from 3.17. It really only serves the prupose of being a valid `APKINDEX.tar.gz` but different from the one in the root of this directory, so we can compare which one is read.
* `alpine-baselayout-3.2.0-r23.apk` - a copy of the apk from `https://dl-cdn.alpinelinux.org/alpine/v3.17/main/aarch64/alpine-baselayout-3.4.0.-0.apk`. Note that the versions are different. It should not be read, only used to validate that this one is read versus the one in the root of this directory.
