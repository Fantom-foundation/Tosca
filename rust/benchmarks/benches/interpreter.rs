/// USAGE:
/// cargo bench --package benchmarks --profile profiling [--features <feature1,feature2,...>]
use std::time::Duration;

use benchmarks::RunArgs;
use criterion::{criterion_group, criterion_main, Criterion};

fn criterion_benchmark(c: &mut Criterion) {
    let (mut args, expected) = RunArgs::ffi_overhead(1);
    c.bench_function("ffi_overhead", |b| {
        b.iter(|| assert_eq!(benchmarks::run(&mut args), expected))
    });
    let (mut args, expected) = RunArgs::inc(1);
    c.bench_function("inc/1", |b| {
        b.iter(|| assert_eq!(benchmarks::run(&mut args), expected))
    });
    let (mut args, expected) = RunArgs::fib(20);
    c.bench_function("fib/20", |b| {
        b.iter(|| assert_eq!(benchmarks::run(&mut args), expected))
    });
    let (mut args, expected) = RunArgs::sha3(1000);
    c.bench_function("sha3/1000", |b| {
        b.iter(|| assert_eq!(benchmarks::run(&mut args), expected))
    });
    let (mut args, expected) = RunArgs::arithmetic(280);
    c.bench_function("arithmetic/280", |b| {
        b.iter(|| assert_eq!(benchmarks::run(&mut args), expected))
    });
    let (mut args, expected) = RunArgs::memory(10000);
    c.bench_function("memory/10000", |b| {
        b.iter(|| assert_eq!(benchmarks::run(&mut args), expected))
    });
    let (mut args, expected) = RunArgs::jumpdest_analysis(0x6000);
    c.bench_function("analysis/jumpdest", |b| {
        b.iter(|| assert_eq!(benchmarks::run(&mut args), expected))
    });
    let (mut args, expected) = RunArgs::stop_analysis(0x6000);
    c.bench_function("analysis/stop", |b| {
        b.iter(|| assert_eq!(benchmarks::run(&mut args), expected))
    });
    let (mut args, expected) = RunArgs::push1_analysis(0x6000);
    c.bench_function("analysis/push1", |b| {
        b.iter(|| assert_eq!(benchmarks::run(&mut args), expected))
    });
    let (mut args, expected) = RunArgs::push32_analysis(0x6000);
    c.bench_function("analysis/push32", |b| {
        b.iter(|| assert_eq!(benchmarks::run(&mut args), expected))
    });
}

criterion_group!(
    name = benches;
    config = Criterion::default()
        .warm_up_time(Duration::from_secs(5))
        .measurement_time(Duration::from_secs(20))
        .sample_size(100);
    targets = criterion_benchmark
);
criterion_main!(benches);
