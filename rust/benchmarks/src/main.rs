use benchmarks::RunArgs;
use clap::{Parser, ValueEnum};

#[derive(Parser, Debug)]
#[command(version, about, long_about = None)]
struct Args {
    runs: u64,
    benchmark: Benchmark,
}

#[derive(Debug, Clone, Copy, ValueEnum)]
enum Benchmark {
    FfiOverhead,
    Inc1,
    Fib20,
    Sha1000,
    Arithmetic280,
    Memory10000,
    JumpdestAnalysis,
    StopAnalysis,
    Push1Analysis,
    Push32Analysis,
    All,
    AllShort,
}

fn main() {
    let args = Args::parse();

    let benches: Vec<fn() -> (RunArgs, u32)> = match args.benchmark {
        Benchmark::FfiOverhead => vec![|| RunArgs::ffi_overhead(1)],
        Benchmark::Inc1 => vec![|| RunArgs::inc(1)],
        Benchmark::Fib20 => vec![|| RunArgs::fib(20)],
        Benchmark::Sha1000 => vec![|| RunArgs::sha3(1000)],
        Benchmark::Arithmetic280 => vec![|| RunArgs::arithmetic(280)],
        Benchmark::Memory10000 => vec![|| RunArgs::memory(10000)],
        Benchmark::JumpdestAnalysis => vec![|| RunArgs::jumpdest_analysis(0x6000)],
        Benchmark::StopAnalysis => vec![|| RunArgs::stop_analysis(0x6000)],
        Benchmark::Push1Analysis => vec![|| RunArgs::push1_analysis(0x6000)],
        Benchmark::Push32Analysis => vec![|| RunArgs::push32_analysis(0x6000)],
        Benchmark::All => vec![
            || RunArgs::ffi_overhead(1),
            || RunArgs::inc(1),
            || RunArgs::fib(20),
            || RunArgs::sha3(1000),
            || RunArgs::arithmetic(280),
            || RunArgs::memory(10000),
            || RunArgs::jumpdest_analysis(0x6000),
            || RunArgs::stop_analysis(0x6000),
            || RunArgs::push1_analysis(0x6000),
            || RunArgs::push32_analysis(0x6000),
        ],
        Benchmark::AllShort => vec![
            || RunArgs::ffi_overhead(1),
            || RunArgs::inc(1),
            || RunArgs::fib(1),
            || RunArgs::sha3(1),
            || RunArgs::arithmetic(1),
            || RunArgs::memory(1),
            || RunArgs::jumpdest_analysis(100),
            || RunArgs::stop_analysis(100),
            || RunArgs::push1_analysis(100),
            || RunArgs::push32_analysis(100),
        ],
    };

    for bench_fn in benches {
        let (mut run_args, expected) = bench_fn();
        for _ in 0..args.runs {
            assert_eq!(benchmarks::run(&mut run_args), expected);
        }
    }
}
