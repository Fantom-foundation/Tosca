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
    agent { label 'quick' }

    options {
        timestamps()
        timeout(time: 2, unit: 'HOURS')
    }

    environment {
        GOROOT = '/usr/lib/go-1.21/'
        CC = 'gcc'
        CXX = 'g++'
    }

    stages {
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
                sh "$GOROOT/bin/gofmt -s -d go"
            }
        }

        stage('Lint Go sources') {
            steps {
                sh 'go vet ./go/...'
                sh 'go install honnef.co/go/tools/cmd/staticcheck@latest'
                sh "$HOME/go/bin/staticcheck ./go/..."
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

        stage('Run C++ tests') {
            steps {
                sh 'make test-cpp'
            }
        }

        stage('Test C++ coverage support') {
            steps {
                sh 'make tosca-cpp-coverage'
                sh 'go test -v  -run ^TestDumpCppCoverageData$ ./go/ct/common/ --expect-coverage'
            }
        }
    }
}
