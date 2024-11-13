use benchmarks::RunArgs;

fn main() {
    let (mut args, expected) = RunArgs::ffi_overhead(1);
    //let (mut args, expected) = RunArgs::inc(1);
    //let (mut args, expected) = RunArgs::fib(20);
    //let (mut args, expected) = RunArgs::sha3(1000);
    //let (mut args, expected) = RunArgs::arithmetic(280);
    //let (mut args, expected) = RunArgs::memory(10000);
    //let (mut args, expected) = RunArgs::jumpdest_analysis(0);
    //let (mut args, expected) = RunArgs::stop_analysis(0);
    //let (mut args, expected) = RunArgs::push1_analysis(0);
    //let (mut args, expected) = RunArgs::push32_analysis(0);
    const ITERATIONS: usize = 200_000_000;
    for _ in 0..ITERATIONS {
        assert_eq!(benchmarks::run(&mut args), expected);
    }
}
