pipeline {
    agent { label 'quick' }

    options {
        timestamps ()
        timeout(time: 2, unit: 'HOURS')
    }

    environment {
        GOROOT = '/usr/lib/go-1.21/'
        CC = 'gcc'
        CXX = 'g++'
    }

    stages {
        stage('Check Go sources formatting') {
            steps {
                sh 'diff=`${GOROOT}/bin/gofmt -s -d .` && echo "$diff" && test -z "$diff"'
            }
        }

        stage('Check C++ sources formatting') {
            steps {
                sh 'find cpp/ -not -path "cpp/build/*" \\( -iname *.h -o -iname *.cc \\) | xargs clang-format --dry-run -Werror'
            }
        }

        stage('Build Go') {
            steps {
                sh 'git submodule update --init --recursive'
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
    }
}
