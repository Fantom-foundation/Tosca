use benchmarks::RunArgs;

fn main() {
    let args = RunArgs::ffi_overhead();
    const ITERATIONS: usize = 200_000_000;
    for _ in 0..ITERATIONS {
        benchmarks::run(&args);
    }
}
