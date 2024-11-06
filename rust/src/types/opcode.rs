const STOP: u8 = 0x00;
const ADD: u8 = 0x01;
const MUL: u8 = 0x02;
const SUB: u8 = 0x03;
const DIV: u8 = 0x04;
const SDIV: u8 = 0x05;
const MOD: u8 = 0x06;
const SMOD: u8 = 0x07;
const ADDMOD: u8 = 0x08;
const MULMOD: u8 = 0x09;
const EXP: u8 = 0x0A;
const SIGNEXTEND: u8 = 0x0B;
const LT: u8 = 0x10;
const GT: u8 = 0x11;
const SLT: u8 = 0x12;
const SGT: u8 = 0x13;
const EQ: u8 = 0x14;
const ISZERO: u8 = 0x15;
const AND: u8 = 0x16;
const OR: u8 = 0x17;
const XOR: u8 = 0x18;
const NOT: u8 = 0x19;
const BYTE: u8 = 0x1A;
const SHL: u8 = 0x1B;
const SHR: u8 = 0x1C;
const SAR: u8 = 0x1D;
const SHA3: u8 = 0x20;
const ADDRESS: u8 = 0x30;
const BALANCE: u8 = 0x31;
const ORIGIN: u8 = 0x32;
const CALLER: u8 = 0x33;
const CALLVALUE: u8 = 0x34;
const CALLDATALOAD: u8 = 0x35;
const CALLDATASIZE: u8 = 0x36;
const CALLDATACOPY: u8 = 0x37;
const CODESIZE: u8 = 0x38;
const CODECOPY: u8 = 0x39;
const GASPRICE: u8 = 0x3A;
const EXTCODESIZE: u8 = 0x3B;
const EXTCODECOPY: u8 = 0x3C;
const RETURNDATASIZE: u8 = 0x3D;
const RETURNDATACOPY: u8 = 0x3E;
const EXTCODEHASH: u8 = 0x3F;
const BLOCKHASH: u8 = 0x40;
const COINBASE: u8 = 0x41;
const TIMESTAMP: u8 = 0x42;
const NUMBER: u8 = 0x43;
const PREVRANDAO: u8 = 0x44;
const GASLIMIT: u8 = 0x45;
const CHAINID: u8 = 0x46;
const SELFBALANCE: u8 = 0x47;
const BASEFEE: u8 = 0x48;
const BLOBHASH: u8 = 0x49;
const BLOBBASEFEE: u8 = 0x4A;
const POP: u8 = 0x50;
const MLOAD: u8 = 0x51;
const MSTORE: u8 = 0x52;
const MSTORE8: u8 = 0x53;
const SLOAD: u8 = 0x54;
const SSTORE: u8 = 0x55;
const JUMP: u8 = 0x56;
const JUMPI: u8 = 0x57;
const PC: u8 = 0x58;
const MSIZE: u8 = 0x59;
const GAS: u8 = 0x5A;
const JUMPDEST: u8 = 0x5B;
const TLOAD: u8 = 0x5C;
const TSTORE: u8 = 0x5D;
const MCOPY: u8 = 0x5E;
const PUSH0: u8 = 0x5F;
const PUSH1: u8 = 0x60;
const PUSH2: u8 = 0x61;
const PUSH3: u8 = 0x62;
const PUSH4: u8 = 0x63;
const PUSH5: u8 = 0x64;
const PUSH6: u8 = 0x65;
const PUSH7: u8 = 0x66;
const PUSH8: u8 = 0x67;
const PUSH9: u8 = 0x68;
const PUSH10: u8 = 0x69;
const PUSH11: u8 = 0x6A;
const PUSH12: u8 = 0x6B;
const PUSH13: u8 = 0x6C;
const PUSH14: u8 = 0x6D;
const PUSH15: u8 = 0x6E;
const PUSH16: u8 = 0x6F;
const PUSH17: u8 = 0x70;
const PUSH18: u8 = 0x71;
const PUSH19: u8 = 0x72;
const PUSH20: u8 = 0x73;
const PUSH21: u8 = 0x74;
const PUSH22: u8 = 0x75;
const PUSH23: u8 = 0x76;
const PUSH24: u8 = 0x77;
const PUSH25: u8 = 0x78;
const PUSH26: u8 = 0x79;
const PUSH27: u8 = 0x7A;
const PUSH28: u8 = 0x7B;
const PUSH29: u8 = 0x7C;
const PUSH30: u8 = 0x7D;
const PUSH31: u8 = 0x7E;
const PUSH32: u8 = 0x7F;
const DUP1: u8 = 0x80;
const DUP2: u8 = 0x81;
const DUP3: u8 = 0x82;
const DUP4: u8 = 0x83;
const DUP5: u8 = 0x84;
const DUP6: u8 = 0x85;
const DUP7: u8 = 0x86;
const DUP8: u8 = 0x87;
const DUP9: u8 = 0x88;
const DUP10: u8 = 0x89;
const DUP11: u8 = 0x8A;
const DUP12: u8 = 0x8B;
const DUP13: u8 = 0x8C;
const DUP14: u8 = 0x8D;
const DUP15: u8 = 0x8E;
const DUP16: u8 = 0x8F;
const SWAP1: u8 = 0x90;
const SWAP2: u8 = 0x91;
const SWAP3: u8 = 0x92;
const SWAP4: u8 = 0x93;
const SWAP5: u8 = 0x94;
const SWAP6: u8 = 0x95;
const SWAP7: u8 = 0x96;
const SWAP8: u8 = 0x97;
const SWAP9: u8 = 0x98;
const SWAP10: u8 = 0x99;
const SWAP11: u8 = 0x9A;
const SWAP12: u8 = 0x9B;
const SWAP13: u8 = 0x9C;
const SWAP14: u8 = 0x9D;
const SWAP15: u8 = 0x9E;
const SWAP16: u8 = 0x9F;
const LOG0: u8 = 0xA0;
const LOG1: u8 = 0xA1;
const LOG2: u8 = 0xA2;
const LOG3: u8 = 0xA3;
const LOG4: u8 = 0xA4;
const CREATE: u8 = 0xF0;
const CALL: u8 = 0xF1;
const CALLCODE: u8 = 0xF2;
const RETURN: u8 = 0xF3;
const DELEGATECALL: u8 = 0xF4;
const CREATE2: u8 = 0xF5;
const STATICCALL: u8 = 0xFA;
const REVERT: u8 = 0xFD;
const INVALID: u8 = 0xFE;
const SELFDESTRUCT: u8 = 0xFF;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u8)]
pub enum Opcode {
    Stop = STOP,
    Add = ADD,
    Mul = MUL,
    Sub = SUB,
    Div = DIV,
    SDiv = SDIV,
    Mod = MOD,
    SMod = SMOD,
    AddMod = ADDMOD,
    MulMod = MULMOD,
    Exp = EXP,
    SignExtend = SIGNEXTEND,
    Lt = LT,
    Gt = GT,
    SLt = SLT,
    SGt = SGT,
    Eq = EQ,
    IsZero = ISZERO,
    And = AND,
    Or = OR,
    Xor = XOR,
    Not = NOT,
    Byte = BYTE,
    Shl = SHL,
    Shr = SHR,
    Sar = SAR,
    Sha3 = SHA3,
    #[cfg(feature = "needs-fn-ptr-conversion")]
    NoOp = SHA3 + 1,
    #[cfg(feature = "needs-fn-ptr-conversion")]
    SkipNoOps = SHA3 + 2,
    Address = ADDRESS,
    Balance = BALANCE,
    Origin = ORIGIN,
    Caller = CALLER,
    CallValue = CALLVALUE,
    CallDataLoad = CALLDATALOAD,
    CallDataSize = CALLDATASIZE,
    CallDataCopy = CALLDATACOPY,
    CodeSize = CODESIZE,
    CodeCopy = CODECOPY,
    GasPrice = GASPRICE,
    ExtCodeSize = EXTCODESIZE,
    ExtCodeCopy = EXTCODECOPY,
    ReturnDataSize = RETURNDATASIZE,
    ReturnDataCopy = RETURNDATACOPY,
    ExtCodeHash = EXTCODEHASH,
    BlockHash = BLOCKHASH,
    Coinbase = COINBASE,
    Timestamp = TIMESTAMP,
    Number = NUMBER,
    PrevRandao = PREVRANDAO,
    GasLimit = GASLIMIT,
    ChainId = CHAINID,
    SelfBalance = SELFBALANCE,
    BaseFee = BASEFEE,
    BlobHash = BLOBHASH,
    BlobBaseFee = BLOBBASEFEE,
    Pop = POP,
    MLoad = MLOAD,
    MStore = MSTORE,
    MStore8 = MSTORE8,
    SLoad = SLOAD,
    SStore = SSTORE,
    Jump = JUMP,
    JumpI = JUMPI,
    Pc = PC,
    MSize = MSIZE,
    Gas = GAS,
    JumpDest = JUMPDEST,
    TLoad = TLOAD,
    TStore = TSTORE,
    MCopy = MCOPY,
    Push0 = PUSH0,
    Push1 = PUSH1,
    Push2 = PUSH2,
    Push3 = PUSH3,
    Push4 = PUSH4,
    Push5 = PUSH5,
    Push6 = PUSH6,
    Push7 = PUSH7,
    Push8 = PUSH8,
    Push9 = PUSH9,
    Push10 = PUSH10,
    Push11 = PUSH11,
    Push12 = PUSH12,
    Push13 = PUSH13,
    Push14 = PUSH14,
    Push15 = PUSH15,
    Push16 = PUSH16,
    Push17 = PUSH17,
    Push18 = PUSH18,
    Push19 = PUSH19,
    Push20 = PUSH20,
    Push21 = PUSH21,
    Push22 = PUSH22,
    Push23 = PUSH23,
    Push24 = PUSH24,
    Push25 = PUSH25,
    Push26 = PUSH26,
    Push27 = PUSH27,
    Push28 = PUSH28,
    Push29 = PUSH29,
    Push30 = PUSH30,
    Push31 = PUSH31,
    Push32 = PUSH32,
    Dup1 = DUP1,
    Dup2 = DUP2,
    Dup3 = DUP3,
    Dup4 = DUP4,
    Dup5 = DUP5,
    Dup6 = DUP6,
    Dup7 = DUP7,
    Dup8 = DUP8,
    Dup9 = DUP9,
    Dup10 = DUP10,
    Dup11 = DUP11,
    Dup12 = DUP12,
    Dup13 = DUP13,
    Dup14 = DUP14,
    Dup15 = DUP15,
    Dup16 = DUP16,
    Swap1 = SWAP1,
    Swap2 = SWAP2,
    Swap3 = SWAP3,
    Swap4 = SWAP4,
    Swap5 = SWAP5,
    Swap6 = SWAP6,
    Swap7 = SWAP7,
    Swap8 = SWAP8,
    Swap9 = SWAP9,
    Swap10 = SWAP10,
    Swap11 = SWAP11,
    Swap12 = SWAP12,
    Swap13 = SWAP13,
    Swap14 = SWAP14,
    Swap15 = SWAP15,
    Swap16 = SWAP16,
    Log0 = LOG0,
    Log1 = LOG1,
    Log2 = LOG2,
    Log3 = LOG3,
    Log4 = LOG4,
    Create = CREATE,
    Call = CALL,
    CallCode = CALLCODE,
    Return = RETURN,
    DelegateCall = DELEGATECALL,
    Create2 = CREATE2,
    StaticCall = STATICCALL,
    Revert = REVERT,
    Invalid = INVALID,
    SelfDestruct = SELFDESTRUCT,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CodeByteType {
    JumpDest,
    #[cfg(feature = "needs-fn-ptr-conversion")]
    Push,
    Opcode,
    DataOrInvalid,
}

pub fn code_byte_type(code_byte: u8) -> (CodeByteType, usize) {
    match code_byte {
        STOP | ADD | MUL | SUB | DIV | SDIV | MOD | SMOD | ADDMOD | MULMOD | EXP | SIGNEXTEND
        | LT | GT | SLT | SGT | EQ | ISZERO | AND | OR | XOR | NOT | BYTE | SHL | SHR | SAR
        | SHA3 | ADDRESS | BALANCE | ORIGIN | CALLER | CALLVALUE | CALLDATALOAD | CALLDATASIZE
        | CALLDATACOPY | CODESIZE | CODECOPY | GASPRICE | EXTCODESIZE | EXTCODECOPY
        | RETURNDATASIZE | RETURNDATACOPY | EXTCODEHASH | BLOCKHASH | COINBASE | TIMESTAMP
        | NUMBER | PREVRANDAO | GASLIMIT | CHAINID | SELFBALANCE | BASEFEE | BLOBHASH
        | BLOBBASEFEE | POP | MLOAD | MSTORE | MSTORE8 | SLOAD | SSTORE | JUMP | JUMPI | PC
        | MSIZE | GAS | TLOAD | TSTORE | MCOPY | PUSH0 | DUP1 | DUP2 | DUP3 | DUP4 | DUP5
        | DUP6 | DUP7 | DUP8 | DUP9 | DUP10 | DUP11 | DUP12 | DUP13 | DUP14 | DUP15 | DUP16
        | SWAP1 | SWAP2 | SWAP3 | SWAP4 | SWAP5 | SWAP6 | SWAP7 | SWAP8 | SWAP9 | SWAP10
        | SWAP11 | SWAP12 | SWAP13 | SWAP14 | SWAP15 | SWAP16 | LOG0 | LOG1 | LOG2 | LOG3
        | LOG4 | CREATE | CALL | CALLCODE | RETURN | DELEGATECALL | CREATE2 | STATICCALL
        | REVERT | INVALID | SELFDESTRUCT => (CodeByteType::Opcode, 0),
        PUSH1..=PUSH32 => (
            #[cfg(not(feature = "needs-fn-ptr-conversion"))]
            CodeByteType::Opcode,
            #[cfg(feature = "needs-fn-ptr-conversion")]
            CodeByteType::Push,
            (code_byte - Opcode::Push1 as u8 + 1) as usize,
        ),
        JUMPDEST => (CodeByteType::JumpDest, 0),
        _ => (CodeByteType::DataOrInvalid, 0),
    }
}
