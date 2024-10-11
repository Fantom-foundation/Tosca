#[cfg(all(feature = "jump-cache", feature = "thread-local-cache"))]
use std::rc::Rc;
#[cfg(all(feature = "jump-cache", not(feature = "thread-local-cache")))]
use std::sync::Arc;
use std::{
    cmp::min,
    mem::{self},
    ops::Deref,
};

#[cfg(feature = "jump-cache")]
use nohash_hasher::BuildNoHashHasher;

#[cfg(feature = "jump-cache")]
use crate::types::Cache;
use crate::types::{code_byte_type, u256, CodeByteType, FailStatus, Opcode};

#[cfg(feature = "jump-cache")]
const CACHE_SIZE: usize = 1 << 16; // value taken from evmzero

#[cfg(all(feature = "jump-cache", not(feature = "thread-local-cache"),))]
pub type JumpCache = Cache<CACHE_SIZE, u256, Arc<[CodeByteType]>, BuildNoHashHasher<u64>>;
#[cfg(all(feature = "jump-cache", feature = "thread-local-cache",))]
pub type JumpCache = Cache<CACHE_SIZE, u256, Rc<[CodeByteType]>, BuildNoHashHasher<u64>>;

#[cfg(all(feature = "jump-cache", not(feature = "thread-local-cache"),))]
static JUMP_CACHE: JumpCache = JumpCache::new();

#[cfg(all(feature = "jump-cache", feature = "thread-local-cache"))]
thread_local! {
    static JUMP_CACHE: JumpCache = JumpCache::new();
}

#[derive(Debug, Clone, Copy)]
pub struct PushLen(usize);

impl PushLen {
    pub const fn new(len: usize) -> Self {
        assert!(len > 0 && len <= 32);
        Self(len)
    }

    pub const fn value(self) -> usize {
        self.0
    }
}

#[derive(Debug)]
pub struct CodeReader<'a> {
    code: &'a [u8],
    #[cfg(not(feature = "jump-cache"))]
    code_byte_types: Box<[CodeByteType]>,
    #[cfg(all(feature = "jump-cache", not(feature = "thread-local-cache")))]
    code_byte_types: Arc<[CodeByteType]>,
    #[cfg(all(feature = "jump-cache", feature = "thread-local-cache"))]
    code_byte_types: Rc<[CodeByteType]>,
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
        let code_byte_type = {
            // NOTE: The HashMap only stores the hash of the code, not the code itself (because
            // this would require an allocation). This means a hash conflict
            // will result in a wrong jump analysis.
            match code_hash {
                #[cfg(not(feature = "thread-local-cache"))]
                Some(code_hash) if code_hash != u256::ZERO => JUMP_CACHE
                    .get_or_insert(code_hash, || {
                        Arc::from(compute_code_byte_types(code).as_slice())
                    }),
                #[cfg(feature = "thread-local-cache")]
                Some(code_hash) if code_hash != u256::ZERO => JUMP_CACHE.with(|cache| {
                    cache.get_or_insert(code_hash, || {
                        Rc::from(compute_code_byte_types(code).as_slice())
                    })
                }),
                #[cfg(not(feature = "thread-local-cache"))]
                _ => Arc::from(compute_code_byte_types(code).as_slice()),
                #[cfg(feature = "thread-local-cache")]
                _ => Rc::from(compute_code_byte_types(code).as_slice()),
            }
        };
        #[cfg(not(feature = "jump-cache"))]
        let code_byte_type = compute_code_byte_types(code).into_boxed_slice();

        Self {
            code,
            code_byte_types: code_byte_type,
            pc,
        }
    }

    pub fn get(&self) -> Result<Opcode, GetOpcodeError> {
        if self.pc >= self.code.len() {
            Err(GetOpcodeError::OutOfRange)
        } else if {
            #[cfg(not(feature = "no-bounds-checks"))]
            {
                self.code_byte_types[self.pc]
            }
            #[cfg(feature = "no-bounds-checks")]
            // SAFETY:
            // self.code and self.code_byte_types have the same length. Because self.pc <
            // self.code.len() this also holds for self.code_byte_types.
            unsafe {
                *self.code_byte_types.get_unchecked(self.pc)
            }
        } == CodeByteType::DataOrInvalid
        {
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

    pub fn get_push_data(&mut self, push_len: PushLen) -> u256 {
        let len = min(push_len.value(), self.code.len().saturating_sub(self.pc));
        let mut data = u256::ZERO;
        if len > 0 {
            #[cfg(not(feature = "no-bounds-checks"))]
            let dest = &mut data[32 - len..];
            #[cfg(feature = "no-bounds-checks")]
            // SAFETY:
            // Because push_len <= 32 so is len, which means the index is always in bounds.
            let dest = unsafe { data.get_unchecked_mut(32 - len..) };
            #[cfg(not(feature = "no-bounds-checks"))]
            let src = &self.code[self.pc..self.pc + len];
            #[cfg(feature = "no-bounds-checks")]
            // SAFETY:
            // - len > 0
            // - because push_len <= 32 so is len
            // - self.pc + len will not overflow because self.code can never be that large because
            //   we would run out of memory before. Therefore, self.pc < self.pc + len.
            // - len <= self.code.len().saturating_sub(self.pc) which also means self.pc + len <=
            //   self.code.len()
            // Therefore, the index is always withing bounds.
            let src = unsafe { self.code.get_unchecked(self.pc..self.pc + len) };
            dest.copy_from_slice(src);
        }
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
        u256, CodeByteType, FailStatus, Opcode, PushLen,
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
    #[should_panic]
    fn push_len_new_0() {
        PushLen::new(0);
    }

    #[test]
    #[should_panic]
    fn push_len_new_33() {
        PushLen::new(33);
    }

    #[test]
    fn push_len_new() {
        for i in 1..=32 {
            assert_eq!(PushLen::new(i).value(), i);
        }
    }

    #[test]
    fn code_reader_get_push_data() {
        let mut code_reader = CodeReader::new(&[0xff; 32], None, 0);
        assert_eq!(code_reader.get_push_data(PushLen::new(1)), 0xffu8.into());

        let mut code_reader = CodeReader::new(&[0xff; 32], None, 0);
        assert_eq!(code_reader.get_push_data(PushLen::new(32)), u256::MAX);

        let mut code_reader = CodeReader::new(&[0xff; 32], None, 31);
        assert_eq!(code_reader.get_push_data(PushLen::new(32)), 0xffu8.into());

        let mut code_reader = CodeReader::new(&[0xff; 32], None, 32);
        assert_eq!(code_reader.get_push_data(PushLen::new(32)), u256::ZERO);

        let mut code_reader = CodeReader::new(&[0xff; 32], None, 33);
        assert_eq!(code_reader.get_push_data(PushLen::new(32)), u256::ZERO);
    }
}
