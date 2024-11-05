#[cfg(any(
    feature = "opcode-fn-ptr-conversion",
    feature = "opcode-fn-ptr-conversion-inline"
))]
use std::cmp::min;
#[cfg(all(feature = "code-analysis-cache", feature = "thread-local-cache"))]
use std::rc::Rc;
#[cfg(all(feature = "code-analysis-cache", not(feature = "thread-local-cache")))]
use std::sync::Arc;

#[cfg(feature = "code-analysis-cache")]
use nohash_hasher::BuildNoHashHasher;

#[cfg(feature = "code-analysis-cache")]
use crate::types::Cache;
#[cfg(all(feature = "code-analysis-cache", feature = "thread-local-cache"))]
use crate::types::LocalKeyExt;
use crate::types::{code_byte_type, u256, CodeByteType};
#[cfg(all(
    not(feature = "opcode-fn-ptr-conversion"),
    feature = "opcode-fn-ptr-conversion-inline"
))]
use crate::types::{op_fn_data::OP_FN_DATA_SIZE, Opcode};
#[cfg(any(
    feature = "opcode-fn-ptr-conversion",
    feature = "opcode-fn-ptr-conversion-inline"
))]
use crate::types::{OpFnData, PcMap};

/// This type represents a hash value in form of a u256.
/// Because it is already a hash value there is no need to hash it again when implementing Hash.
#[cfg(feature = "code-analysis-cache")]
#[allow(non_camel_case_types)]
#[derive(Debug, PartialEq, Eq)]
struct u256Hash(u256);

#[cfg(feature = "code-analysis-cache")]
impl std::hash::Hash for u256Hash {
    fn hash<H: std::hash::Hasher>(&self, state: &mut H) {
        state.write_u64(self.0.into_u64_with_overflow().0);
    }
}

#[cfg(not(feature = "code-analysis-cache"))]
pub type AnalysisContainer<T> = T;
#[cfg(all(feature = "code-analysis-cache", not(feature = "thread-local-cache")))]
pub type AnalysisContainer<T> = Arc<T>;
#[cfg(all(feature = "code-analysis-cache", feature = "thread-local-cache"))]
pub type AnalysisContainer<T> = Rc<T>;

#[cfg(all(
    not(feature = "opcode-fn-ptr-conversion"),
    not(feature = "opcode-fn-ptr-conversion-inline"),
))]
pub type AnalysisItem = CodeByteType;
#[cfg(any(
    feature = "opcode-fn-ptr-conversion",
    feature = "opcode-fn-ptr-conversion-inline"
))]
pub type AnalysisItem = OpFnData;

#[cfg(feature = "code-analysis-cache")]
const CACHE_SIZE: usize = 1 << 16; // value taken from evmzero

#[cfg(feature = "code-analysis-cache")]
type CodeAnalysisCache =
    Cache<CACHE_SIZE, u256Hash, AnalysisContainer<CodeAnalysis>, BuildNoHashHasher<u64>>;

#[cfg(all(feature = "code-analysis-cache", not(feature = "thread-local-cache")))]
static CODE_ANALYSIS_CACHE: CodeAnalysisCache = CodeAnalysisCache::new();

#[cfg(all(feature = "code-analysis-cache", feature = "thread-local-cache"))]
thread_local! {
    static CODE_ANALYSIS_CACHE: CodeAnalysisCache = CodeAnalysisCache::new();
}

#[derive(Debug)]
pub struct CodeAnalysis {
    pub analysis: Vec<AnalysisItem>,
    #[cfg(any(
        feature = "opcode-fn-ptr-conversion",
        feature = "opcode-fn-ptr-conversion-inline"
    ))]
    pub pc_map: PcMap,
}

impl CodeAnalysis {
    #[allow(unused_variables)]
    pub fn new(code: &[u8], code_hash: Option<u256>) -> AnalysisContainer<Self> {
        #[cfg(feature = "code-analysis-cache")]
        match code_hash {
            Some(code_hash) if code_hash != u256::ZERO => CODE_ANALYSIS_CACHE
                .get_or_insert(u256Hash(code_hash), || {
                    AnalysisContainer::new(Self::analyze_code(code))
                }),
            _ => AnalysisContainer::new(Self::analyze_code(code)),
        }
        #[cfg(not(feature = "code-analysis-cache"))]
        Self::analyze_code(code)
    }

    #[cfg(all(
        not(feature = "opcode-fn-ptr-conversion"),
        not(feature = "opcode-fn-ptr-conversion-inline")
    ))]
    fn analyze_code(code: &[u8]) -> Self {
        let mut code_byte_types = vec![CodeByteType::DataOrInvalid; code.len()];

        let mut pc = 0;
        while let Some(op) = code.get(pc).copied() {
            let (code_byte_type, data) = code_byte_type(op);
            code_byte_types[pc] = code_byte_type;
            pc += 1 + data;
        }

        CodeAnalysis {
            analysis: code_byte_types,
        }
    }
    #[cfg(feature = "opcode-fn-ptr-conversion")]
    fn analyze_code(code: &[u8]) -> Self {
        let mut analysis = Vec::with_capacity(code.len());
        // +32+1 because if last op is push32 we need mapping from after converted to after code+32
        let mut pc_map = PcMap::new(code.len() + 32 + 1);

        let mut pc = 0;
        let mut no_ops = 0;
        while let Some(op) = code.get(pc).copied() {
            let (code_byte_type, data_len) = code_byte_type(op);

            pc += 1;
            match code_byte_type {
                CodeByteType::JumpDest => {
                    pc_map.add_mapping(pc - 1, analysis.len());
                    if no_ops > 0 {
                        analysis.extend(OpFnData::skip_no_ops_iter(no_ops));
                    }
                    no_ops = 0;
                    analysis.push(OpFnData::jump_dest());
                    pc_map.add_mapping(pc - 1, analysis.len() - 1);
                }
                CodeByteType::Push => {
                    let mut data = u256::ZERO;
                    let avail = min(data_len, code.len() - pc);
                    data[32 - data_len..32 - data_len + avail]
                        .copy_from_slice(&code[pc..pc + avail]);
                    analysis.push(OpFnData::func(op, data));
                    pc_map.add_mapping(pc - 1, analysis.len() - 1);

                    no_ops += data_len;
                    pc += data_len;
                }
                CodeByteType::Opcode => {
                    analysis.push(OpFnData::func(op, u256::ZERO));
                    pc_map.add_mapping(pc - 1, analysis.len() - 1);
                }
                CodeByteType::DataOrInvalid => {
                    // This should only be the case if an invalid opcode was not preceded by a push.
                    // In this case we don't care what the data contains.
                    analysis.push(OpFnData::data(u256::ZERO));
                    pc_map.add_mapping(pc - 1, analysis.len() - 1);
                }
            };
        }

        pc_map.add_mapping(pc, analysis.len()); // in case pc points past code (this is valid)

        CodeAnalysis { analysis, pc_map }
    }
    #[cfg(all(
        not(feature = "opcode-fn-ptr-conversion"),
        feature = "opcode-fn-ptr-conversion-inline"
    ))]
    fn analyze_code(code: &[u8]) -> Self {
        let mut analysis = Vec::with_capacity(code.len());
        // +32+1 because if last op is push32 we need mapping from after converted to after code+32
        let mut pc_map = PcMap::new(code.len() + 32 + 1);

        let mut pc = 0;
        let mut no_ops = 0;
        while let Some(op) = code.get(pc).copied() {
            let (code_byte_type, mut data_len) = code_byte_type(op);

            pc += 1;
            match code_byte_type {
                CodeByteType::JumpDest => {
                    pc_map.add_mapping(pc - 1, analysis.len());
                    match no_ops {
                        0 => (),
                        1 => analysis.push(OpFnData::func(Opcode::NoOp as u8)),
                        2.. => analysis.extend(OpFnData::skip_no_ops_iter(no_ops)),
                    }
                    no_ops = 0;
                    analysis.push(OpFnData::jump_dest());
                    pc_map.add_mapping(pc - 1, analysis.len() - 1);
                }
                CodeByteType::Push => {
                    analysis.push(OpFnData::func(op));
                    pc_map.add_mapping(pc - 1, analysis.len() - 1);

                    let mut capped_inc = data_len.rem_euclid(OP_FN_DATA_SIZE);
                    if capped_inc == 0 {
                        capped_inc = OP_FN_DATA_SIZE;
                    }
                    let data = copy_push_data(code, pc, capped_inc);
                    analysis.push(OpFnData::data(data));
                    no_ops += capped_inc - 1;
                    data_len -= capped_inc;
                    pc += capped_inc;

                    while data_len > 0 {
                        let capped_inc = min(data_len, OP_FN_DATA_SIZE);
                        let data = copy_push_data(code, pc, capped_inc);
                        analysis.push(OpFnData::data(data));
                        no_ops += capped_inc - 1;
                        data_len -= capped_inc;
                        pc += capped_inc;
                    }
                }
                CodeByteType::Opcode => {
                    analysis.push(OpFnData::func(op));
                    pc_map.add_mapping(pc - 1, analysis.len() - 1);
                }
                CodeByteType::DataOrInvalid => {
                    // This should only be the case if an invalid opcode was not preceded by a push.
                    // In this case we don't care what the data contains.
                    analysis.push(OpFnData::data([0; OP_FN_DATA_SIZE]));
                    pc_map.add_mapping(pc - 1, analysis.len() - 1);
                }
            };
        }

        pc_map.add_mapping(pc, analysis.len()); // in case pc points past code (this is valid)

        CodeAnalysis { analysis, pc_map }
    }
}

#[cfg(all(
    not(feature = "opcode-fn-ptr-conversion"),
    feature = "opcode-fn-ptr-conversion-inline"
))]
fn copy_push_data(src: &[u8], src_start: usize, len: usize) -> [u8; OP_FN_DATA_SIZE] {
    let src = &src[min(src.len(), src_start)..];
    let len = min(len, OP_FN_DATA_SIZE);
    let mut data = [0; OP_FN_DATA_SIZE];
    let copy = min(len, src.len());
    data[OP_FN_DATA_SIZE - len..OP_FN_DATA_SIZE - len + copy].copy_from_slice(&src[..copy]);
    data
}

#[cfg(test)]
mod tests {
    #[cfg(all(
        not(feature = "opcode-fn-ptr-conversion"),
        not(feature = "opcode-fn-ptr-conversion-inline")
    ))]
    use crate::types::CodeByteType;
    #[cfg(all(
        not(feature = "opcode-fn-ptr-conversion"),
        feature = "opcode-fn-ptr-conversion-inline"
    ))]
    use crate::types::{op_fn_data::OP_FN_DATA_SIZE, OpFnData};
    #[cfg(feature = "opcode-fn-ptr-conversion")]
    use crate::types::{u256, OpFnData};
    use crate::types::{CodeAnalysis, Opcode};

    #[cfg(all(
        not(feature = "opcode-fn-ptr-conversion"),
        not(feature = "opcode-fn-ptr-conversion-inline")
    ))]
    #[test]
    fn analyze_code_single_byte() {
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::Add as u8]).analysis,
            [CodeByteType::Opcode]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::Push2 as u8]).analysis,
            [CodeByteType::Opcode]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::JumpDest as u8]).analysis,
            [CodeByteType::JumpDest]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[0xc0]).analysis,
            [CodeByteType::DataOrInvalid]
        );
    }

    #[cfg(feature = "opcode-fn-ptr-conversion")]
    #[test]
    fn analyze_code_single_byte() {
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::Add as u8]).analysis,
            [OpFnData::func(Opcode::Add as u8, u256::ZERO)]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::Push2 as u8]).analysis,
            [OpFnData::func(Opcode::Push2 as u8, u256::ZERO)]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::JumpDest as u8]).analysis,
            [OpFnData::jump_dest()]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[0xc0]).analysis,
            [OpFnData::data(u256::ZERO)]
        );
    }
    #[cfg(all(
        not(feature = "opcode-fn-ptr-conversion"),
        feature = "opcode-fn-ptr-conversion-inline"
    ))]
    #[test]
    fn analyze_code_single_byte() {
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::Add as u8]).analysis,
            [OpFnData::func(Opcode::Add as u8)]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::Push2 as u8]).analysis,
            [
                OpFnData::func(Opcode::Push2 as u8),
                OpFnData::data([0; OP_FN_DATA_SIZE])
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::JumpDest as u8]).analysis,
            [OpFnData::jump_dest()]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[0xc0]).analysis,
            [OpFnData::data([0; OP_FN_DATA_SIZE])]
        );
    }

    #[cfg(all(
        not(feature = "opcode-fn-ptr-conversion"),
        not(feature = "opcode-fn-ptr-conversion-inline")
    ))]
    #[test]
    fn analyze_code_jumpdest() {
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::JumpDest as u8, Opcode::Add as u8]).analysis,
            [CodeByteType::JumpDest, CodeByteType::Opcode]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::JumpDest as u8, 0xc0]).analysis,
            [CodeByteType::JumpDest, CodeByteType::DataOrInvalid]
        );
    }

    #[cfg(feature = "opcode-fn-ptr-conversion")]
    #[test]
    fn analyze_code_jumpdest() {
        use crate::u256;

        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::JumpDest as u8, Opcode::Add as u8]).analysis,
            [
                OpFnData::jump_dest(),
                OpFnData::func(Opcode::Add as u8, u256::ZERO)
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::JumpDest as u8, 0xc0]).analysis,
            [OpFnData::jump_dest(), OpFnData::data(u256::ZERO)]
        );
    }
    #[cfg(all(
        not(feature = "opcode-fn-ptr-conversion"),
        feature = "opcode-fn-ptr-conversion-inline"
    ))]
    #[test]
    fn analyze_code_jumpdest() {
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::JumpDest as u8, Opcode::Add as u8]).analysis,
            [OpFnData::jump_dest(), OpFnData::func(Opcode::Add as u8)]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::JumpDest as u8, 0xc0]).analysis,
            [OpFnData::jump_dest(), OpFnData::data([0; OP_FN_DATA_SIZE])]
        );
    }
    #[cfg(all(
        not(feature = "opcode-fn-ptr-conversion"),
        not(feature = "opcode-fn-ptr-conversion-inline")
    ))]
    #[test]
    fn analyze_code_push_with_data() {
        assert_eq!(
            CodeAnalysis::analyze_code(&[
                Opcode::Push1 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8
            ])
            .analysis,
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::Opcode,
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::Push1 as u8, Opcode::Add as u8, 0xc0]).analysis,
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[
                Opcode::Push1 as u8,
                Opcode::Add as u8,
                0xc0,
                Opcode::Add as u8
            ])
            .analysis,
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
                CodeByteType::Opcode,
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[
                Opcode::Push2 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
            ])
            .analysis,
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
                CodeByteType::Opcode,
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[
                Opcode::Push2 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
                0xc0
            ])
            .analysis,
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
            ]
        );
    }

    #[cfg(feature = "opcode-fn-ptr-conversion")]
    #[test]
    fn analyze_code_push_with_data() {
        use crate::u256;

        assert_eq!(
            CodeAnalysis::analyze_code(&[
                Opcode::Push1 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8
            ])
            .analysis,
            [
                OpFnData::func(Opcode::Push1 as u8, (Opcode::Add as u8).into()),
                OpFnData::func(Opcode::Add as u8, u256::ZERO),
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::Push1 as u8, Opcode::Add as u8, 0xc0]).analysis,
            [
                OpFnData::func(Opcode::Push1 as u8, (Opcode::Add as u8).into()),
                OpFnData::data(u256::ZERO),
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[
                Opcode::Push1 as u8,
                Opcode::Add as u8,
                0xc0,
                Opcode::Add as u8
            ])
            .analysis,
            [
                OpFnData::func(Opcode::Push1 as u8, (Opcode::Add as u8).into()),
                OpFnData::data(u256::ZERO),
                OpFnData::func(Opcode::Add as u8, u256::ZERO),
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[
                Opcode::Push2 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
            ])
            .analysis,
            [
                OpFnData::func(
                    Opcode::Push2 as u8,
                    (((Opcode::Add as u8 as u64) << 8) + Opcode::Add as u8 as u64).into()
                ),
                OpFnData::func(Opcode::Add as u8, u256::ZERO),
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[
                Opcode::Push2 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
                0xc0
            ])
            .analysis,
            [
                OpFnData::func(
                    Opcode::Push2 as u8,
                    (((Opcode::Add as u8 as u64) << 8) + Opcode::Add as u8 as u64).into()
                ),
                OpFnData::data(u256::ZERO),
            ]
        );
        let mut code = [0; 23];
        code[0] = Opcode::Push21 as u8;
        code[1] = 1;
        code[21] = 2;
        code[22] = Opcode::Add as u8;
        assert_eq!(
            CodeAnalysis::analyze_code(&code).analysis,
            [
                OpFnData::func(
                    Opcode::Push21 as u8,
                    (u256::ONE << u256::from(8 * 20u8)) + u256::from(2u8)
                ),
                OpFnData::func(Opcode::Add as u8, u256::ZERO),
            ]
        );
    }
    #[cfg(all(
        not(feature = "opcode-fn-ptr-conversion"),
        feature = "opcode-fn-ptr-conversion-inline"
    ))]
    #[test]
    fn analyze_code_push_with_data() {
        assert_eq!(
            CodeAnalysis::analyze_code(&[
                Opcode::Push1 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8
            ])
            .analysis,
            [
                OpFnData::func(Opcode::Push1 as u8,),
                OpFnData::data([0, 0, 0, Opcode::Add as u8]),
                OpFnData::func(Opcode::Add as u8),
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::Push1 as u8, Opcode::Add as u8, 0xc0]).analysis,
            [
                OpFnData::func(Opcode::Push1 as u8,),
                OpFnData::data([0, 0, 0, Opcode::Add as u8]),
                OpFnData::data([0; OP_FN_DATA_SIZE]),
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[
                Opcode::Push1 as u8,
                Opcode::Add as u8,
                0xc0,
                Opcode::Add as u8
            ])
            .analysis,
            [
                OpFnData::func(Opcode::Push1 as u8,),
                OpFnData::data([0, 0, 0, Opcode::Add as u8]),
                OpFnData::data([0; OP_FN_DATA_SIZE]),
                OpFnData::func(Opcode::Add as u8),
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[
                Opcode::Push2 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
            ])
            .analysis,
            [
                OpFnData::func(Opcode::Push2 as u8,),
                OpFnData::data([0, 0, Opcode::Add as u8, Opcode::Add as u8]),
                OpFnData::func(Opcode::Add as u8),
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[
                Opcode::Push2 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
                0xc0
            ])
            .analysis,
            [
                OpFnData::func(Opcode::Push2 as u8,),
                OpFnData::data([0, 0, Opcode::Add as u8, Opcode::Add as u8]),
                OpFnData::data([0; OP_FN_DATA_SIZE]),
            ]
        );
        let mut code = [1; 23];
        code[0] = Opcode::Push21 as u8;
        code[1] = 2;
        code[21] = 3;
        code[22] = Opcode::Add as u8;
        assert_eq!(
            CodeAnalysis::analyze_code(&code).analysis,
            [
                OpFnData::func(Opcode::Push21 as u8),
                OpFnData::data([0, 0, 0, 2]),
                OpFnData::data([1, 1, 1, 1]),
                OpFnData::data([1, 1, 1, 1]),
                OpFnData::data([1, 1, 1, 1]),
                OpFnData::data([1, 1, 1, 1]),
                OpFnData::data([1, 1, 1, 3]),
                OpFnData::func(Opcode::Add as u8),
            ]
        );
    }

    #[cfg(all(
        not(feature = "opcode-fn-ptr-conversion"),
        feature = "opcode-fn-ptr-conversion-inline"
    ))]
    #[test]
    fn copy_push_data() {
        assert_eq!(super::copy_push_data(&[], 0, 0), [0; OP_FN_DATA_SIZE]);
        assert_eq!(super::copy_push_data(&[], 0, 1), [0; OP_FN_DATA_SIZE]);
        assert_eq!(super::copy_push_data(&[], 1, 0), [0; OP_FN_DATA_SIZE]);
        assert_eq!(super::copy_push_data(&[], 1, 1), [0; OP_FN_DATA_SIZE]);
        assert_eq!(super::copy_push_data(&[1, 2, 3], 0, 2), [0, 0, 1, 2]);
        assert_eq!(super::copy_push_data(&[1, 2, 3], 1, 2), [0, 0, 2, 3]);
        assert_eq!(super::copy_push_data(&[1, 2, 3], 1, 3), [0, 2, 3, 0]);
        assert_eq!(super::copy_push_data(&[1, 2, 3], 1, 1), [0, 0, 0, 2]);
        assert_eq!(
            super::copy_push_data(&[1, 2, 3, 4, 5, 6, 7, 8, 9, 10], 1, 4),
            [2, 3, 4, 5]
        );
        assert_eq!(
            super::copy_push_data(&[1, 2, 3, 4, 5, 6, 7, 8, 9, 10], 1, 100),
            [2, 3, 4, 5]
        );
    }
}
