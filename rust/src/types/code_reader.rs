use std::{
    cmp::min,
    mem::{self},
    ops::Deref,
};
#[cfg(feature = "jump-cache")]
use std::{
    num::NonZeroUsize,
    sync::{Arc, LazyLock, Mutex},
};

#[cfg(feature = "jump-cache")]
use lru::LruCache;
#[cfg(feature = "jump-cache")]
use nohash_hasher::BuildNoHashHasher;

use crate::types::{code_byte_type, u256, CodeByteType, FailStatus, Opcode};

/// This type represents a hash value in form of a u256.
/// Because it is already a hash value there is no need to hash it again when implementing Hash.
#[cfg(feature = "jump-cache")]
#[allow(non_camel_case_types)]
#[derive(Debug, PartialEq, Eq)]
struct u256Hash(u256);

#[cfg(feature = "jump-cache")]
impl std::hash::Hash for u256Hash {
    fn hash<H: std::hash::Hasher>(&self, state: &mut H) {
        state.write_u64(self.0.into_u64_with_overflow().0);
    }
}

#[cfg(feature = "jump-cache")]
const CACHE_SIZE: NonZeroUsize = unsafe { NonZeroUsize::new_unchecked(1 << 16) }; // taken from evmzero

#[cfg(feature = "jump-cache")]
type JumpCache = LazyLock<Mutex<LruCache<u256Hash, Arc<[CodeByteType]>, BuildNoHashHasher<u64>>>>;

// Mutex<LruCache<...>> is faster that quick_cache::Cache<...>
#[cfg(feature = "jump-cache")]
static JUMP_CACHE: JumpCache = LazyLock::new(|| {
    Mutex::new(LruCache::with_hasher(
        CACHE_SIZE,
        BuildNoHashHasher::default(),
    ))
});

#[derive(Debug)]
pub struct CodeReader<'a> {
    code: &'a [u8],
    #[cfg(feature = "jump-cache")]
    code_byte_types: Arc<[CodeByteType]>,
    #[cfg(not(feature = "jump-cache"))]
    code_byte_types: Box<[CodeByteType]>,
    pc: usize,
}

impl<'a> Deref for CodeReader<'a> {
    type Target = [u8];

    fn deref(&self) -> &Self::Target {
        self.code
    }
}

#[derive(Debug, PartialEq, Eq)]
pub enum GetOpcodeError {
    OutOfRange,
    Invalid,
}

impl<'a> CodeReader<'a> {
    #[allow(unused_variables)]
    pub fn new(code: &'a [u8], code_hash: Option<u256>, pc: usize) -> Self {
        #[cfg(feature = "jump-cache")]
        let code_byte_types = if code_hash.is_some_and(|h| h != u256::ZERO) {
            JUMP_CACHE
                .lock()
                .unwrap()
                .get_or_insert(u256Hash(code_hash.unwrap()), || {
                    Arc::from(compute_code_byte_types(code).as_slice())
                })
                .clone()
        } else {
            Arc::from(compute_code_byte_types(code).as_slice())
        };
        #[cfg(not(feature = "jump-cache"))]
        let code_byte_types = compute_code_byte_types(code).into_boxed_slice();

        Self {
            code,
            code_byte_types,
            pc,
        }
    }

    pub fn get(&self) -> Result<Opcode, GetOpcodeError> {
        if self.pc >= self.code.len() {
            Err(GetOpcodeError::OutOfRange)
        } else if self.code_byte_types[self.pc] == CodeByteType::DataOrInvalid {
            Err(GetOpcodeError::Invalid)
        } else {
            let op = self.code[self.pc];
            // SAFETY:
            // [Opcode] has repr(u8) and therefore the same memory layout as u8.
            // In get_code_byte_types this byte of the code was determined to be a valid opcode.
            // Therefore the value is a valid [Opcode].
            let op = unsafe { mem::transmute::<u8, Opcode>(op) };
            Ok(op)
        }
    }

    pub fn next(&mut self) {
        self.pc += 1;
    }

    pub fn try_jump(&mut self, dest: u256) -> Result<(), FailStatus> {
        let dest = u64::try_from(dest).map_err(|_| FailStatus::BadJumpDestination)? as usize;
        if dest >= self.code_byte_types.len()
            || self.code_byte_types[dest] != CodeByteType::JumpDest
        {
            return Err(FailStatus::BadJumpDestination);
        }
        self.pc = dest;

        Ok(())
    }

    pub fn get_push_data(&mut self, len: usize) -> u256 {
        assert!(len <= 32);

        let len = min(len, self.code.len() - self.pc);
        let mut data = u256::ZERO;
        data[32 - len..].copy_from_slice(&self.code[self.pc..self.pc + len]);
        self.pc += len;

        data
    }

    pub fn pc(&self) -> usize {
        self.pc
    }
}

fn compute_code_byte_types(code: &[u8]) -> Vec<CodeByteType> {
    let mut code_byte_types = vec![CodeByteType::DataOrInvalid; code.len()];

    let mut pc = 0;
    while pc < code.len() {
        let (code_byte_type, inc) = code_byte_type(code[pc]);
        code_byte_types[pc] = code_byte_type;
        pc += inc;
    }

    code_byte_types
}

#[cfg(test)]
mod tests {
    use crate::types::{
        code_reader::{compute_code_byte_types, CodeReader, GetOpcodeError},
        u256, CodeByteType, FailStatus, Opcode,
    };

    #[test]
    fn compute_code_byte_types_single_byte() {
        assert_eq!(
            *compute_code_byte_types(&[Opcode::Add as u8]),
            [CodeByteType::Opcode]
        );
        assert_eq!(
            *compute_code_byte_types(&[Opcode::Push2 as u8]),
            [CodeByteType::Opcode]
        );
        assert_eq!(
            *compute_code_byte_types(&[Opcode::JumpDest as u8]),
            [CodeByteType::JumpDest]
        );
        assert_eq!(
            *compute_code_byte_types(&[0xc0]),
            [CodeByteType::DataOrInvalid]
        );
    }

    #[test]
    fn compute_byte_types_jumpdest() {
        assert_eq!(
            *compute_code_byte_types(&[Opcode::JumpDest as u8, Opcode::Add as u8]),
            [CodeByteType::JumpDest, CodeByteType::Opcode]
        );
        assert_eq!(
            *compute_code_byte_types(&[Opcode::JumpDest as u8, 0xc0]),
            [CodeByteType::JumpDest, CodeByteType::DataOrInvalid]
        );
    }

    #[test]
    fn compute_code_byte_types_push_with_data() {
        assert_eq!(
            *compute_code_byte_types(&[Opcode::Push1 as u8, Opcode::Add as u8, Opcode::Add as u8]),
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::Opcode,
            ]
        );
        assert_eq!(
            *compute_code_byte_types(&[Opcode::Push1 as u8, Opcode::Add as u8, 0xc0]),
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
            ]
        );
        assert_eq!(
            *compute_code_byte_types(&[
                Opcode::Push1 as u8,
                Opcode::Add as u8,
                0xc0,
                Opcode::Add as u8
            ]),
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
                CodeByteType::Opcode,
            ]
        );
        assert_eq!(
            *compute_code_byte_types(&[
                Opcode::Push2 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
            ]),
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
                CodeByteType::Opcode,
            ]
        );
        assert_eq!(
            *compute_code_byte_types(&[
                Opcode::Push2 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
                0xc0
            ]),
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
            ]
        );
    }

    #[test]
    fn code_reader_internals() {
        let code = [Opcode::Add as u8, Opcode::Add as u8, 0xc0];
        let pc = 1;
        let code_reader = CodeReader::new(&code, None, pc);
        assert_eq!(*code_reader, code);
        assert_eq!(code_reader.len(), code.len());
        assert_eq!(code_reader.pc(), pc);
    }

    #[test]
    fn code_reader_get() {
        let mut code_reader =
            CodeReader::new(&[Opcode::Add as u8, Opcode::Add as u8, 0xc0], None, 0);
        assert_eq!(code_reader.get(), Ok(Opcode::Add));
        code_reader.next();
        assert_eq!(code_reader.get(), Ok(Opcode::Add));
        code_reader.next();
        assert_eq!(code_reader.get(), Err(GetOpcodeError::Invalid));
        code_reader.next();
        assert_eq!(code_reader.get(), Err(GetOpcodeError::OutOfRange));
    }

    #[test]
    fn code_reader_try_jump() {
        let mut code_reader = CodeReader::new(
            &[
                Opcode::Push1 as u8,
                Opcode::JumpDest as u8,
                Opcode::JumpDest as u8,
            ],
            None,
            0,
        );
        assert_eq!(
            code_reader.try_jump(1u8.into()),
            Err(FailStatus::BadJumpDestination)
        );
        assert_eq!(code_reader.try_jump(2u8.into()), Ok(()));
        assert_eq!(
            code_reader.try_jump(3u8.into()),
            Err(FailStatus::BadJumpDestination)
        );
        assert_eq!(
            code_reader.try_jump(u256::MAX),
            Err(FailStatus::BadJumpDestination)
        );
    }

    #[test]
    fn code_reader_get_push_data() {
        let mut code_reader = CodeReader::new(&[0xff; 32], None, 0);
        assert_eq!(code_reader.get_push_data(0u8.into()), u256::ZERO);

        let mut code_reader = CodeReader::new(&[0xff; 32], None, 0);
        assert_eq!(code_reader.get_push_data(1u8.into()), 0xffu8.into());

        let mut code_reader = CodeReader::new(&[0xff; 32], None, 0);
        assert_eq!(code_reader.get_push_data(32u8.into()), u256::MAX);

        let mut code_reader = CodeReader::new(&[0xff; 32], None, 31);
        assert_eq!(code_reader.get_push_data(32u8.into()), 0xffu8.into());

        let mut code_reader = CodeReader::new(&[0xff; 32], None, 32);
        assert_eq!(code_reader.get_push_data(32u8.into()), u256::ZERO);
    }
}
