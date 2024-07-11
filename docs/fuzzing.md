# Fuzzing Tosca

This project includes Fuzzing tests, using the Go Lang infrastructure. 

## Develop Fuzzing tests

Go provides fuzzing support in the form of special unit tests. This is well described [here](https://go.dev/doc/security/fuzz/).

Important things to keep in mind:
- Fuzzing test functions are found in file named *_test.go and are named starting with the word Fuzz
- Fuzzing interface is not freely typed, there is only a number of types supported and the arguments of the fuzzing kernel need to be keep in the same types and order as used in the corpus seed definition. 
- Fuzzing unit tests are unit tests, the corpus (initial seed + found and committed offending data) will be tested along with any other unit tests defined in the file, or in the tested paths.

## Fuzzing in the nightly build

The nightly build will execute fuzzing unit tests. Eventually may find an error, and a corpus file will be generated in such cases. This File is the input data required to reproduce the bug.

Final form of this tooling is still a work in progress... (one script committed in this repo, maybe to be migrated to lascala, manual configuration in [jenkins](https://scala.fantom.network/job/Tosca/job/Experimental/job/Test-Fuzzer/))

## Maintain Fuzzing tests

Whenever a failure is found, a corpus file will be generated. This file can be committed to the repository, at the same location where it is generated to serve as a regression test. Alternatively, the file can be used to craft an unit test.

Once the corpus file (named XXXXX) is found in the correct testdata folder (`go/ct/testdata/fuzz/FuzzFunctionName`), the test can be repeated in isolation with the command `go test -run=FuzzFunctionName/XXXXX ./go/ct`

Debugging is possible using vscode: add a [launch configuration](https://code.visualstudio.com/docs/editor/debugging) as follows:
```json
    {
        "name": "FuzzFunctionName XXXXX",
        "type": "go",
        "request": "launch",
        "mode": "test",
        "program": "${workspaceFolder}/go/ct",
        "args": [
            "test",
            "-run=FuzzFunctionName/XXXXX",
        ]
    }
```