#!/usr/bin/env bash
source "$(dirname "${BASH_SOURCE}")/lib/init.sh"

os::test::junit::declare_suite_start "update/kube-api-inventory"
os::cmd::expect_success "go run -mod vendor ./test/extended/apiserver/inventory/write-kube-api-inventory"
os::test::junit::declare_suite_end
