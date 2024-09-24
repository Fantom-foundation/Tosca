/// USAGE:
/// cargo bench --package benchmarks --profile profiling [--features <feature1,feature2,...>]
use std::time::Duration;

use benchmarks::RunArgs;
use criterion::{criterion_group, criterion_main, Criterion};

fn criterion_benchmark(c: &mut Criterion) {
    let args = RunArgs::ffi_overhead();
    c.bench_function("ffi_overhead", |b| b.iter(|| benchmarks::run(&args)));
}

criterion_group!(
    name = benches;
    config = Criterion::default()
        .warm_up_time(Duration::from_secs(5))
        .measurement_time(Duration::from_secs(20))
        .sample_size(1000);
    targets = criterion_benchmark
);
criterion_main!(benches);
