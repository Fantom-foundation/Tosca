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

    parameters {
        choice(
            name: 'ENTRY_POINT',
            choices: ['FuzzLfvm', 'FuzzGeth', 'FuzzDifferentialLfvmVsGeth'],
            description: 'Selects which fuzzer test fucntion to start.')
    }

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
        stage('Build') {
            steps {
                sh 'make'
            }
        }

        stage('Fuzz LFVM') {
            steps {
                sh "go test -fuzz=$ENTRY_POINT ./go/ct"
            }
        }
    }

    post {
        always {
            archiveArtifacts artifacts: "go/ct/testdata/fuzz/$ENTRY_POINT/*"
        }
    }
}
