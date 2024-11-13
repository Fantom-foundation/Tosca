use std::borrow::Cow;

use crate::interpreter::Interpreter;

pub trait Observer {
    fn pre_op(&mut self, interpreter: &Interpreter);

    fn post_op(&mut self, interpreter: &Interpreter);

    fn log(&mut self, message: Cow<str>);
}

pub struct NoOpObserver();

impl Observer for NoOpObserver {
    fn pre_op(&mut self, _interpreter: &Interpreter) {}

    fn post_op(&mut self, _interpreter: &Interpreter) {}

    fn log(&mut self, _message: Cow<str>) {}
}

pub struct LoggingObserver();

impl Observer for LoggingObserver {
    fn pre_op(&mut self, interpreter: &Interpreter) {
        println!(
            "pre opcode={:?} gas={} stack-size={}",
            interpreter.code_reader.get(),
            interpreter.gas_left.as_u64(),
            interpreter.stack.len()
        );
    }

    fn post_op(&mut self, interpreter: &Interpreter) {
        println!(
            "post gas={} stack-size={}",
            interpreter.gas_left.as_u64(),
            interpreter.stack.len()
        );
    }

    fn log(&mut self, message: Cow<str>) {
        println!("{message}");
    }
}

#[derive(Debug, Clone, Copy)]
pub enum ObserverType {
    NoOp,
    Logging,
}
