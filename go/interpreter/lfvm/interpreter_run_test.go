// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package lfvm

// type opDef struct {
// 	stackPops, stackPushes int
// 	introducedIn           tosca.Revision
// }

// var opDefinitions map[OpCode]opDef = map[OpCode]opDef{
// 	BASEFEE:     {introducedIn: tosca.R10_London},
// 	PUSH0:       {introducedIn: tosca.R12_Shanghai},
// 	BLOBHASH:    {introducedIn: tosca.R13_Cancun},
// 	BLOBBASEFEE: {introducedIn: tosca.R13_Cancun},
// 	TLOAD:       {introducedIn: tosca.R13_Cancun},
// 	TSTORE:      {introducedIn: tosca.R13_Cancun},
// 	MCOPY:       {introducedIn: tosca.R13_Cancun},
// }

// var jumpOpCodes = []OpCode{
// 	JUMP, JUMPI,
// 	JUMP_TO,
// 	POP_JUMP, PUSH2_JUMP, PUSH2_JUMPI,
// 	SWAP2_SWAP1_POP_JUMP, ISZERO_PUSH2_JUMPI,
// }

// func testInterpreter_Run_EveryOpCodeExecutes(t *testing.T) {

// 	for _, op := range allOpCodes() {
// 		t.Run(op.String(), func(t *testing.T) {

// 			if slices.Contains(jumpOpCodes, op) {
// 				t.Skip("Jump opcodes are not tested here")
// 			}

// 			if !isValidOpCode(op) {
// 				t.Skip("Invalid opcode", op)
// 			}

// 			opProperties, ok := opDefinitions[op]
// 			if !ok {
// 				opProperties = opDef{}
// 				// t.Errorf("op %v not defined in opDefinitions", op)
// 			}

// 			forEachRevision(t, opProperties, func(t *testing.T, revision tosca.Revision) {

// 				ctrl := gomock.NewController(t)
// 				mock := tosca.NewMockRunContext(ctrl)
// 				mockAll(mock)

// 				ctxt := makeContext(
// 					op,
// 					make([]byte, 0),
// 					mock,
// 					10_000,
// 					revision,
// 				)

// 				// Run testing code
// 				vanillaRunner{}.run(&ctxt)

// 				if got := ctxt.status; isError(got) {
// 					t.Errorf("execution failed: %v", got)
// 				}

// 				// // Check gas consumption
// 				// if want, got := test.gasConsumed, test.gasStart-ctxt.gas; want != got {
// 				// 	t.Errorf("execution failed: gas consumption is %v, wanted %v", got, want)
// 				// }

// 				// // Check gas refund
// 				// if want, got := test.gasRefund, ctxt.refund; want != got {
// 				// 	t.Errorf("execution failed: gas refund is %v, wanted %v", got, want)
// 				// }

// 			})
// 		})
// 	}
// }

// // //////////////////////////////////////////////////////////////////////////////
// // utils

// func isError(status status) bool {
// 	return status == statusFailed
// }

// func mockAll(mock *tosca.MockRunContext) {
// 	mock.EXPECT().AccessAccount(gomock.Any()).Return(tosca.WarmAccess).AnyTimes()
// 	mock.EXPECT().GetBalance(gomock.Any()).AnyTimes()
// 	mock.EXPECT().IsAddressInAccessList(gomock.Any()).AnyTimes()
// 	mock.EXPECT().GetCodeSize(gomock.Any()).AnyTimes()
// 	mock.EXPECT().GetCode(gomock.Any()).AnyTimes()
// 	mock.EXPECT().AccountExists(gomock.Any()).AnyTimes()
// 	mock.EXPECT().AccessStorage(gomock.Any(), gomock.Any()).AnyTimes()
// 	mock.EXPECT().GetStorage(gomock.Any(), gomock.Any()).AnyTimes()
// 	mock.EXPECT().SetStorage(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
// 	mock.EXPECT().Call(gomock.Any(), gomock.Any()).AnyTimes()
// 	mock.EXPECT().EmitLog(gomock.Any()).AnyTimes()
// 	mock.EXPECT().SelfDestruct(gomock.Any(), gomock.Any()).AnyTimes()
// 	mock.EXPECT().GetTransientStorage(gomock.Any(), gomock.Any()).AnyTimes()
// 	mock.EXPECT().SetTransientStorage(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

// }

// func makeContext(op OpCode, data []byte, runContext tosca.RunContext, gas tosca.Gas, revision tosca.Revision) context {

// 	// Create execution context.
// 	ctx := context{
// 		params: tosca.Parameters{
// 			BlockParameters: tosca.BlockParameters{
// 				Revision: revision,
// 			},
// 			Gas:   gas,
// 			Input: data,
// 		},
// 		context: runContext,
// 		gas:     gas,
// 		stack:   NewStack(),
// 		memory:  NewMemory(),
// 		status:  statusRunning,
// 		code:    generateCodeFor(op),
// 	}

// 	ctx.stack.stackPointer = 21

// 	return ctx
// }

// func test_generateCodeForOps(t *testing.T) {
// 	tests := map[OpCode]int{
// 		PUSH1: 0,
// 		PUSH2: 0,
// 		PUSH3: 1,
// 		PUSH4: 1,
// 		PUSH5: 2,
// 		PUSH6: 2,
// 	}
// 	for op, test := range tests {
// 		code := generateCodeFor(op)
// 		if len(code) != test+1 {
// 			t.Errorf("%v generation: expected %d instructions, got %d", op, test+1, len(code))
// 		}
// 	}
// }

// func generateCodeFor(op OpCode) Code {
// 	code := []Instruction{{op, 0}}
// 	for _, op := range append(op.decompose(), op) {
// 		if PUSH3 <= op && op <= PUSH32 {
// 			dataOpsRequired := (int(op) - int(PUSH3) + 1) / 2
// 			for i := 0; i < dataOpsRequired; i++ {
// 				code = append(code, Instruction{DATA, 0})
// 			}
// 		}
// 	}
// 	return code
// }

// func isValidOpCode(op OpCode) bool {
// 	re := regexp.MustCompile(`^op\(0x[0-9A-Fa-f]+\)$`)
// 	return !re.MatchString(op.String()) && op != INVALID
// }

// func forEachRevision(
// 	t *testing.T, opDef opDef,
// 	f func(t *testing.T, revision tosca.Revision)) {
// 	for revision := tosca.R07_Istanbul; revision <= newestSupportedRevision; revision++ {
// 		if revision < opDef.introducedIn {
// 			continue
// 		}
// 		t.Run(revision.String(), func(t *testing.T) {
// 			f(t, revision)
// 		})
// 	}
// }
