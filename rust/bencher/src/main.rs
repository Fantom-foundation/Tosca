use std::{
    cmp::Ordering,
    collections::HashMap,
    error::Error,
    fs,
    process::{Command, Output},
};

use chrono::Utc;
use clap::Parser;
use serde::Deserialize;

#[derive(Parser, Debug)]
#[command(version, about, long_about = None)]
/// Run Go interpreter benchmarks for evmzero, lfvm and geth as well as for evmrs with different
/// feature sets.
///
/// Requirements:
/// go install golang.org/x/perf/cmd/benchstat@latest
struct Args {
    #[arg(long)]
    evmrs_only: bool,
    #[arg(long, default_value = "20")]
    runs: u8,
    #[arg(long, default_value = "1h")]
    timeout: String,
    #[arg(long, default_value = "^Benchmark[a-zA-Z]+")]
    benchmark: String,
    #[arg(long, default_value = "performance")]
    baseline: String,
    /// evmrs features to benchmark. These features are relative to baseline. If they start with
    /// `~` they are subtracted and other wise added to baseline.
    features: Vec<String>,
}

#[derive(Deserialize)]
struct CargoToml {
    features: HashMap<String, Vec<String>>,
}

const NON_RUST_INTERPRETERS: [&str; 3] = ["evmzero", "lfvm", "geth"];

fn main() {
    if let Err(err) = run() {
        println!("{err}");
    }
}

fn run() -> Result<(), Box<dyn Error>> {
    let args = Args::parse();
    let datetime = Utc::now().format("%Y-%m-%dT%H:%M").to_string();
    let git_ref = get_git_ref()?;
    let baseline_features = get_baseline_features(&args)?;
    let feature_lists: Vec<_> = args
        .features
        .iter()
        .map(|features| {
            (
                format!("{},{features}", args.baseline),
                build_feature_list(&baseline_features, features),
            )
        })
        .chain(Some((args.baseline.clone(), baseline_features.join(","))))
        .collect();
    let mut feature_map = String::new();
    for (features, features_expanded) in &feature_lists {
        feature_map += &format!("{features} = {features_expanded}\n");
    }

    println!("planned runs:");
    if !args.evmrs_only {
        for interpreter in NON_RUST_INTERPRETERS {
            println!("{interpreter}");
        }
    }
    for (features, feature_expanded) in &feature_lists {
        println!("evmrs with features: {features} = {feature_expanded}");
    }

    let output_dir = format!("output/benches/{datetime}#{git_ref}#{}", args.runs);
    fs::create_dir_all(&output_dir)
        .map_err(|e| format!("Failed to create output directory: {e}"))?;

    fs::write(format!("{output_dir}/feature-map"), &feature_map)?;

    println!("running make ...");
    make_build()?;

    if !args.evmrs_only {
        for interpreter in NON_RUST_INTERPRETERS {
            println!("running {interpreter} ...");
            let output_file = format!("{output_dir}/{interpreter}");
            run_go_bench(interpreter, &args, &output_file)?;
        }
    }

    for (features, features_expanded) in &feature_lists {
        println!("running evmrs with features: {features} ...");
        cargo_build(features_expanded)?;
        let output_file = format!("{output_dir}/evmrs#{features}");
        run_go_bench("evmrs", &args, &output_file)?;
    }

    run_benchstat(&output_dir)
}

fn get_git_ref() -> Result<String, Box<dyn Error>> {
    let output = Command::new("git")
        .args(["rev-parse", "--short=7", "HEAD"])
        .output()?;
    check_success("git rev-parse", &output)?;
    Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
}

fn get_baseline_features(args: &Args) -> Result<Vec<String>, Box<dyn Error>> {
    let cargo_toml: CargoToml = toml::from_str(&fs::read_to_string("Cargo.toml")?)?;
    cargo_toml
        .features
        .get(&args.baseline)
        .cloned()
        .ok_or_else(|| format!("No feature '{}' in Cargo.toml", args.baseline).into())
}

fn build_feature_list(baseline_features: &[String], features: &str) -> String {
    let features = features.split(',').collect::<Vec<_>>();
    let neg_features: Vec<_> = features
        .iter()
        .copied()
        .filter(|f| f.starts_with('~'))
        .map(|f| &f[1..])
        .collect();
    let pos_features = features.iter().copied().filter(|f| !f.starts_with('~'));
    baseline_features
        .iter()
        .map(AsRef::as_ref)
        .filter(|f| !neg_features.contains(f))
        .chain(pos_features)
        .collect::<Vec<_>>()
        .join(",")
}

fn make_build() -> Result<(), Box<dyn Error>> {
    let output = Command::new("make").current_dir("..").output()?;
    check_success("make", &output)
}

fn cargo_build(features: &str) -> Result<(), Box<dyn Error>> {
    let output = Command::new("cargo").arg("clean").output()?;
    check_success("cargo clean", &output)?;
    let output = Command::new("cargo")
        .args(["build", "--lib", "--release", "--features", features])
        .output()?;
    check_success("cargo build", &output)
}

fn run_go_bench(interpreter: &str, args: &Args, output_file: &str) -> Result<(), Box<dyn Error>> {
    let output = Command::new("taskset")
        .args([
            "--cpu-list",
            "0",
            "go",
            "test",
            "../go/integration_test/interpreter",
            "--run",
            "none",
            "--bench",
            &format!("^{}/./{}$", args.benchmark, interpreter),
            "--timeout",
            &args.timeout,
            "--count",
            &args.runs.to_string(),
        ])
        .output()?;

    check_success("go test", &output)?;
    let bench_output = String::from_utf8_lossy(&output.stdout).replace(interpreter, "");
    fs::write(output_file, bench_output)?;
    Ok(())
}

fn run_benchstat(output_dir: &str) -> Result<(), Box<dyn Error>> {
    let mut files: Vec<_> = fs::read_dir(output_dir)?
        .map(|d| d.unwrap().file_name())
        .collect();
    // sort evmzero (baseline), evmrs..., geth, lfvm
    files.sort_by(|n1, n2| {
        if n1 == "evmzero" {
            Ordering::Less
        } else if n2 == "evmzero" {
            Ordering::Greater
        } else {
            n1.cmp(n2)
        }
    });
    let output = Command::new("benchstat")
        .args(files)
        .current_dir(output_dir)
        .output()?;

    check_success("benchstat", &output)?;
    fs::write(format!("{output_dir}/comparison"), &output.stdout)?;
    println!("{}", String::from_utf8_lossy(&output.stdout));
    Ok(())
}

fn check_success(command: &str, output: &Output) -> Result<(), Box<dyn Error>> {
    if !output.status.success() {
        return Err(format!(
            "{command} failed:\n{}\n{}",
            String::from_utf8_lossy(&output.stdout),
            String::from_utf8_lossy(&output.stderr)
        )
        .into());
    }
    Ok(())
}
