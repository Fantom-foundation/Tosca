// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

pipeline {
    agent {
        dockerfile {
            filename 'CI/Dockerfile.build'
            label 'quick'
        }
    }

    options {
        timestamps()
        timeout(time: 2, unit: 'HOURS')
    }

    stages {
        stage('Validate commit') {
            steps {
                script {
                    def CHANGE_REPO = sh(script: 'basename -s .git `git config --get remote.origin.url`', returnStdout: true).trim()
                    build job: '/Utils/Validate-Git-Commit', parameters: [
                        string(name: 'Repo', value: "${CHANGE_REPO}"),
                        string(name: 'Branch', value: "${env.CHANGE_BRANCH}"),
                        string(name: 'Commit', value: "${GIT_COMMIT}")
                    ]
                }
            }
        }

        stage('Checkout code') {
            steps {
                sh 'git submodule update --init --recursive'
            }
        }

        stage('Check License headers') {
            steps {
                sh 'cd scripts/license && ./add_license_header.sh --check'
            }
        }

        stage('Check Go sources formatting') {
            steps {
                sh 'gofmt -s -d go'
            }
        }

        stage('Lint Go sources') {
            steps {
                sh 'echo $PATH'
                sh 'echo $GOPATH'
                sh 'echo $GOCACHE'
                sh 'echo $GOTMPDIR'
                sh 'ls /tmp'
                withEnv(["PATH+GOPATH=${env.HOME}/go/bin"]) {
                    sh 'make lint-go'
                }
            }
        }

        stage('Check C++ sources formatting') {
            steps {
                sh 'find cpp/ -not -path "cpp/build/*" \\( -iname *.h -o -iname *.cc \\) | xargs clang-format --dry-run -Werror'
            }
        }

        stage('Build Go') {
            steps {
                sh 'make tosca-go'
            }
        }

        stage('Run Go tests') {
            steps {
                sh 'make test-go'
            }
        }

        stage('CT regression tests LFVM') {
            steps {
                sh 'go run ./go/ct/driver regressions lfvm'
            }
        }

        stage('CT regression tests evmzero') {
            steps {
                sh 'go run ./go/ct/driver regressions evmzero'
            }
        }

        stage('LFVM race condition tests') {
            steps {
                sh 'GORACE="halt_on_error=1" go test --race ./go/interpreter/lfvm/...'
            }
        }

        stage('Run C++ tests') {
            steps {
                sh 'make test-cpp'
            }
        }

        stage('Test C++ coverage support') {
            steps {
                sh 'make tosca-cpp-coverage'
                sh 'go test -v  -run ^TestDumpCppCoverageData$ ./go/lib/cpp/ --expect-coverage'
            }
        }

        stage('Test Rust coverage support') {
            steps {
                sh 'make tosca-rust-coverage'
                sh 'LLVM_PROFILE_FILE="/tmp/go/rust-%p-%m.profraw" go test -v  -run ^TestDumpRustCoverageData$ ./go/lib/rust/ --expect-coverage'
            }
        }
    }
}
