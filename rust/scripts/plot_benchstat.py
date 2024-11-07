# USAGE: benchstat ... | python3 plot_benchstat.py

import sys
import matplotlib.pyplot as plt
import matplotlib
import re

def parse_benchstat_output(benchstat_output):
    lines = benchstat_output.strip().splitlines()
    lines = lines[4:] # skip config lines
    
    version_headers = [col.strip() for col in lines[0].split("│")[1:-1]]
    lines = lines[2:]
    
    # match number followed by n (nano), µ (micro), m (milli), or s (seconds)
    time_pattern = r'([\d\.]+)(n|µ|m|s)'
    
    data = {}
    for line in lines:
        # split at at least 2 whitespace chars
        parts = re.split(r'\s{2,}', line.strip())
        
        benchmark_name = parts[0]
        
        benchmark_data = {}
        i = 0
        for result in parts[1:]:
            match = re.search(time_pattern, result)
            if match:
                time_value = float(match.group(1))
                time_unit = match.group(2)

                if time_unit == 'µ':
                    time_value *= 1e3
                elif time_unit == 'm':
                    time_value *= 1e6
                elif time_unit == 's':
                    time_value *= 1e9

                interpreter = version_headers[i] if i < len(version_headers) else f"Version {i + 1}"
                
                benchmark_data[interpreter] = time_value
                i = i + 1
        data[benchmark_name] = benchmark_data
    
    return data

def generate_colors(num_colors):
    color_map = matplotlib.colormaps["tab20"]
    return [color_map(i / num_colors) for i in range(num_colors)]

def plot_benchmarks(data):
    benchmarks = list(data.keys())
    interpreters = list(data[benchmarks[0]].keys())

    colors = generate_colors(len(interpreters))

    for benchmark in benchmarks:
        fig, ax = plt.subplots(figsize=(10, 15))

        times = [data[benchmark][interpreter] for interpreter in interpreters]

        bars = ax.bar(interpreters, times, color=colors[:len(interpreters)])
        for (bar, time) in zip(bars, times):
            height = ax.get_ylim()[1]
            if time > 1e9:
                label = f'{time/1e9:.3f}s'
            elif time > 1e6:
                label = f'{time/1e6:.3f}ms'
            elif time > 1e3:
                label = f'{time/1e3:.3f}µs'
            else:
                label = f'{time:.3f}ns'
            ax.text(
                bar.get_x() + bar.get_width() / 2,
                time + height * 0.01,
                label,
                ha='center',
                va='bottom',
                fontsize=10,
                rotation=90
            )
        handles = [plt.Line2D([0], [0], color=color, lw=4) for color in colors]
        ax.legend(handles, interpreters, title='Interpreter', loc='upper center', bbox_to_anchor=(0.5, -0.05))

        ax.set_xlabel('Interpreter')
        ax.set_ylabel('Time (ns/op)')
        ax.set_title(benchmark)

        ax.grid(True)

        plt.xticks([])
        plt.tight_layout()
        plt.savefig(f'{benchmark.replace("/", "_")}.png')
        plt.close(fig)

def main():
    benchstat_output = sys.stdin.read()
    parsed_data = parse_benchstat_output(benchstat_output)
    plot_benchmarks(parsed_data)

if __name__ == "__main__":
    main()
