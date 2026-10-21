#!/bin/bash

# Chess Engine Benchmark and Performance Report Generator
# Usage: ./benchmark.sh [options]

set -e

# Configuration
BENCHMARK_OUTPUT="benchmark_results.txt"
REPORT_OUTPUT="performance_report.html"
DATE=$(date +"%Y-%m-%d %H:%M:%S")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH=$(git branch --show-current 2>/dev/null || echo "unknown")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Help function
show_help() {
    echo "Chess Engine Benchmark Script"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help          Show this help message"
    echo "  -o, --output DIR    Output directory (default: .reports)"
    echo "  -r, --runs N        Number of benchmark runs (default: 3)"
    echo "  -t, --tests PATTERN Run specific test pattern (default: all)"
    echo "  -c, --compare FILE  Compare with previous results"
    echo "  --cpu-profile       Enable CPU profiling"
    echo "  --mem-profile       Enable memory profiling"
    echo "  --integration       Run integration tests before benchmarks"
    echo ""
    echo "Examples:"
    echo "  $0                          # Run all benchmarks"
    echo "  $0 -r 5 -t BenchmarkMove    # Run move benchmarks 5 times"
    echo "  $0 --integration            # Run integration tests + benchmarks"
    echo "  $0 -c old_results.txt       # Compare with previous results"
}

# Default values
OUTPUT_DIR=".reports"
RUNS=3
TEST_PATTERN="Benchmark"
COMPARE_FILE=""
CPU_PROFILE=false
MEM_PROFILE=false
RUN_INTEGRATION=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -r|--runs)
            RUNS="$2"
            shift 2
            ;;
        -t|--tests)
            TEST_PATTERN="$2"
            shift 2
            ;;
        -c|--compare)
            COMPARE_FILE="$2"
            shift 2
            ;;
        --cpu-profile)
            CPU_PROFILE=true
            shift
            ;;
        --mem-profile)
            MEM_PROFILE=true
            shift
            ;;
        --integration)
            RUN_INTEGRATION=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

echo -e "${BLUE}=== Chess Engine Performance Benchmark ===${NC}"
echo -e "Date: ${YELLOW}$DATE${NC}"
echo -e "Git Branch: ${YELLOW}$GIT_BRANCH${NC}"
echo -e "Git Commit: ${YELLOW}$GIT_COMMIT${NC}"
echo -e "Output Directory: ${YELLOW}$OUTPUT_DIR${NC}"
echo ""

# Function to run integration tests
run_integration_tests() {
    echo -e "${BLUE}Running Integration Tests...${NC}"
    if go test -v ./pkg/chess -run "TestGameIntegration|TestGameWithTimerIntegration|TestCastlingIntegration|TestEnPassantIntegration" > "$OUTPUT_DIR/integration_results.txt" 2>&1; then
        echo -e "${GREEN}✓ Integration tests passed${NC}"
    else
        echo -e "${RED}✗ Integration tests failed${NC}"
        echo "Check $OUTPUT_DIR/integration_results.txt for details"
        exit 1
    fi
}

# Function to run benchmarks
run_benchmarks() {
    echo -e "${BLUE}Running Benchmarks...${NC}"
    
    local bench_flags=""
    
    # Add profiling flags if requested
    if [ "$CPU_PROFILE" = true ]; then
        bench_flags="$bench_flags -cpuprofile=$OUTPUT_DIR/cpu.prof"
    fi
    
    if [ "$MEM_PROFILE" = true ]; then
        bench_flags="$bench_flags -memprofile=$OUTPUT_DIR/mem.prof"
    fi
    
    # Run benchmarks multiple times for statistical significance
    echo "Running $RUNS benchmark iterations..."
    for i in $(seq 1 $RUNS); do
        echo -e "  ${YELLOW}Iteration $i/$RUNS${NC}"
        go test -bench="$TEST_PATTERN" -benchmem -count=1 $bench_flags ./pkg/chess >> "$OUTPUT_DIR/$BENCHMARK_OUTPUT" 2>&1
    done
    
    echo -e "${GREEN}✓ Benchmarks completed${NC}"
}

# Function to generate HTML report
generate_html_report() {
    echo -e "${BLUE}Generating HTML Report...${NC}"
    
    cat > "$OUTPUT_DIR/$REPORT_OUTPUT" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Chess Engine Performance Report</title>
    <meta charset="UTF-8">
    <style>
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
            margin: 0; 
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container { 
            max-width: 1200px; 
            margin: 0 auto; 
            background: white; 
            padding: 30px; 
            border-radius: 10px; 
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .header { 
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px; 
            border-radius: 10px; 
            margin: -30px -30px 30px -30px;
        }
        .header h1 { margin: 0 0 20px 0; font-size: 2.5em; }
        .header p { margin: 5px 0; opacity: 0.9; }
        
        .section { margin: 30px 0; }
        .section h2 { 
            color: #333; 
            border-bottom: 3px solid #667eea; 
            padding-bottom: 10px; 
            margin-bottom: 20px;
        }
        
        .benchmark-table { 
            width: 100%; 
            border-collapse: collapse; 
            margin: 20px 0;
            background: white;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .benchmark-table th { 
            background: #667eea;
            color: white;
            padding: 15px 12px; 
            text-align: left; 
            font-weight: 600;
            text-transform: uppercase;
            font-size: 0.9em;
            letter-spacing: 0.5px;
        }
        .benchmark-table td { 
            padding: 12px; 
            border-bottom: 1px solid #eee;
            transition: background-color 0.2s;
        }
        .benchmark-table tr:hover td { 
            background-color: #f8f9ff;
        }
        .benchmark-table tr:last-child td { 
            border-bottom: none; 
        }
        
        .performance-good { color: #28a745; font-weight: bold; }
        .performance-medium { color: #ffc107; font-weight: bold; }
        .performance-poor { color: #dc3545; font-weight: bold; }
        
        .metrics { 
            display: grid; 
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); 
            gap: 20px; 
            margin: 20px 0;
        }
        .metric { 
            background: linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%);
            padding: 20px; 
            border-radius: 10px;
            border-left: 4px solid #667eea;
        }
        .metric h4 { margin: 0 0 10px 0; color: #333; }
        .metric p { margin: 0; color: #666; line-height: 1.5; }
        
        .system-info { 
            background: #f8f9fa; 
            padding: 20px; 
            border-radius: 8px; 
            font-family: 'Courier New', monospace;
            font-size: 0.9em;
            line-height: 1.6;
            border-left: 4px solid #6c757d;
        }
        
        .test-results { 
            background: #f8f9fa; 
            padding: 20px; 
            border-radius: 8px; 
            font-family: 'Courier New', monospace;
            font-size: 0.9em;
            max-height: 400px;
            overflow-y: auto;
            border-left: 4px solid #28a745;
        }
        
        .footer { 
            margin-top: 50px; 
            padding-top: 20px; 
            border-top: 2px solid #eee; 
            text-align: center; 
            color: #666;
            font-style: italic;
        }
        
        .status-badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 20px;
            font-size: 0.8em;
            font-weight: bold;
            text-transform: uppercase;
        }
        .status-good { background: #d4edda; color: #155724; }
        .status-medium { background: #fff3cd; color: #856404; }
        .status-poor { background: #f8d7da; color: #721c24; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>♔ Chess Engine Performance Report</h1>
            <p><strong>Generated:</strong> $DATE</p>
            <p><strong>Git Branch:</strong> $GIT_BRANCH</p>
            <p><strong>Git Commit:</strong> $GIT_COMMIT</p>
            <p><strong>Benchmark Runs:</strong> $RUNS</p>
        </div>

        <div class="section">
            <h2>📊 System Information</h2>
            <div class="system-info">$(go version)
$(uname -a)
CPU: $(grep -m1 "model name" /proc/cpuinfo | cut -d: -f2 | xargs)
Memory: $(free -h | grep "Mem:" | awk '{print $2}')
            </div>
        </div>

        <div class="section">
            <h2>🚀 Benchmark Results</h2>
EOF

    # Parse benchmark results and add to HTML
    if [ -f "$OUTPUT_DIR/$BENCHMARK_OUTPUT" ]; then
        echo "            <table class=\"benchmark-table\">" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                <thead>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                    <tr>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                        <th>Benchmark Name</th>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                        <th>Iterations</th>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                        <th>Time per Op</th>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                        <th>Memory per Op</th>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                        <th>Allocs per Op</th>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                        <th>Performance</th>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                    </tr>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                </thead>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                <tbody>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        
        # Parse and format benchmark data
        grep "^Benchmark" "$OUTPUT_DIR/$BENCHMARK_OUTPUT" | while read -r line; do
            name=$(echo "$line" | awk '{print $1}' | sed 's/Benchmark//')
            iterations=$(echo "$line" | awk '{print $2}')
            time_per_op=$(echo "$line" | awk '{print $3}')
            
            # Extract memory and allocations if available
            memory_per_op="N/A"
            allocs_per_op="N/A"
            if [[ $line == *"B/op"* ]]; then
                memory_per_op=$(echo "$line" | grep -o '[0-9.]* B/op' | head -1)
            fi
            if [[ $line == *"allocs/op"* ]]; then
                allocs_per_op=$(echo "$line" | grep -o '[0-9.]* allocs/op' | head -1)
            fi
            
            # Determine performance category based on time
            performance_class="performance-good"
            performance_badge="status-good"
            performance_text="Excellent"
            
            if [[ $time_per_op == *"µs"* ]]; then
                time_val=$(echo "$time_per_op" | sed 's/µs//')
                if (( $(echo "$time_val > 1000" | bc -l) 2>/dev/null || [[ ${time_val%.*} -gt 1000 ]] )); then
                    performance_class="performance-medium"
                    performance_badge="status-medium"
                    performance_text="Good"
                fi
            elif [[ $time_per_op == *"ms"* ]]; then
                performance_class="performance-poor"
                performance_badge="status-poor"
                performance_text="Needs Optimization"
            elif [[ $time_per_op == *"ns"* ]]; then
                time_val=$(echo "$time_per_op" | sed 's/ns//')
                if (( $(echo "$time_val > 10000" | bc -l) 2>/dev/null || [[ ${time_val%.*} -gt 10000 ]] )); then
                    performance_class="performance-medium"
                    performance_badge="status-medium"
                    performance_text="Good"
                fi
            fi
            
            echo "                    <tr>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
            echo "                        <td><strong>$name</strong></td>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
            echo "                        <td>$iterations</td>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
            echo "                        <td class=\"$performance_class\">$time_per_op</td>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
            echo "                        <td>$memory_per_op</td>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
            echo "                        <td>$allocs_per_op</td>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
            echo "                        <td><span class=\"status-badge $performance_badge\">$performance_text</span></td>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
            echo "                    </tr>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        done
        
        echo "                </tbody>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "            </table>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        
        echo "        </div>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        
        # Add detailed metrics section
        echo "        <div class=\"section\">" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "            <h2>📈 Performance Metrics</h2>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "            <div class=\"metrics\">" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        
        # Calculate some basic statistics
        total_benchmarks=$(grep -c "^Benchmark" "$OUTPUT_DIR/$BENCHMARK_OUTPUT")
        fastest_time=$(grep "^Benchmark" "$OUTPUT_DIR/$BENCHMARK_OUTPUT" | awk '{print $3}' | sort -n | head -1)
        slowest_time=$(grep "^Benchmark" "$OUTPUT_DIR/$BENCHMARK_OUTPUT" | awk '{print $3}' | sort -n | tail -1)
        
        echo "                <div class=\"metric\">" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                    <h4>Total Benchmarks</h4>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                    <p>$total_benchmarks benchmark functions executed</p>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                </div>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        
        echo "                <div class=\"metric\">" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                    <h4>Fastest Operation</h4>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                    <p class=\"performance-good\">$fastest_time</p>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                </div>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        
        echo "                <div class=\"metric\">" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                    <h4>Slowest Operation</h4>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                    <p class=\"performance-medium\">$slowest_time</p>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "                </div>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        
        echo "            </div>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        
        # Add raw results section
        echo "            <h3>📋 Raw Benchmark Output</h3>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "            <div class=\"test-results\">" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        tail -50 "$OUTPUT_DIR/$BENCHMARK_OUTPUT" | sed 's/</\&lt;/g' | sed 's/>/\&gt;/g' >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "            </div>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
    fi

    # Add comparison if compare file provided
    if [ -n "$COMPARE_FILE" ] && [ -f "$COMPARE_FILE" ]; then
        echo "    <h3>Comparison with Previous Results</h3>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "    <p>Comparing current results with: $COMPARE_FILE</p>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        # Add comparison logic here (you could use benchcmp or similar tools)
    fi

    # Add integration test results if available
    if [ -f "$OUTPUT_DIR/integration_results.txt" ]; then
        echo "    <h2>Integration Test Results</h2>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "    <pre>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        cat "$OUTPUT_DIR/integration_results.txt" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
        echo "    </pre>" >> "$OUTPUT_DIR/$REPORT_OUTPUT"
    fi

    cat >> "$OUTPUT_DIR/$REPORT_OUTPUT" << EOF
    
    <h2>Performance Analysis</h2>
    <div class="metrics">
        <div class="metric">
            <h4>Move Generation</h4>
            <p>Critical for game responsiveness</p>
        </div>
        <div class="metric">
            <h4>Board Operations</h4>
            <p>Foundation performance metrics</p>
        </div>
        <div class="metric">
            <h4>Timer Operations</h4>
            <p>Real-time game requirements</p>
        </div>
    </div>

    <h2>Recommendations</h2>
    <ul>
        <li>Monitor move generation performance - should be < 1ms for real-time play</li>
        <li>Board copy operations should be optimized as they're used frequently</li>
        <li>Timer operations should have minimal overhead</li>
        <li>Memory allocations should be minimized in hot paths</li>
    </ul>

    <footer>
        <hr>
        <p><em>Generated by Chess Engine Benchmark Script</em></p>
    </footer>
</body>
</html>
EOF

    echo -e "${GREEN}✓ HTML report generated: $OUTPUT_DIR/$REPORT_OUTPUT${NC}"
}

# Function to analyze performance trends
analyze_performance() {
    if [ -f "$OUTPUT_DIR/$BENCHMARK_OUTPUT" ]; then
        echo -e "${BLUE}Performance Analysis:${NC}"
        
        # Extract key metrics
        local move_gen_ns=$(grep "BenchmarkMoveGeneration" "$OUTPUT_DIR/$BENCHMARK_OUTPUT" | tail -1 | awk '{print $3}')
        local game_state_ns=$(grep "BenchmarkGameStateUpdate" "$OUTPUT_DIR/$BENCHMARK_OUTPUT" | tail -1 | awk '{print $3}')
        local board_copy_ns=$(grep "BenchmarkBoardCopy" "$OUTPUT_DIR/$BENCHMARK_OUTPUT" | tail -1 | awk '{print $3}')
        
        echo "  Move Generation: $move_gen_ns ns/op"
        echo "  Game State Update: $game_state_ns ns/op" 
        echo "  Board Copy: $board_copy_ns ns/op"
        
        # Simple performance categorization
        if [ -n "$move_gen_ns" ] && [ "$move_gen_ns" -lt 1000000 ]; then
            echo -e "  Move Generation: ${GREEN}GOOD${NC} (< 1ms)"
        elif [ -n "$move_gen_ns" ] && [ "$move_gen_ns" -lt 10000000 ]; then
            echo -e "  Move Generation: ${YELLOW}ACCEPTABLE${NC} (< 10ms)"
        else
            echo -e "  Move Generation: ${RED}NEEDS OPTIMIZATION${NC} (> 10ms)"
        fi
    fi
}

# Main execution
main() {
    # Run integration tests if requested
    if [ "$RUN_INTEGRATION" = true ]; then
        run_integration_tests
    fi
    
    # Run benchmarks
    run_benchmarks
    
    # Analyze performance
    analyze_performance
    
    # Generate HTML report
    generate_html_report
    
    echo ""
    echo -e "${GREEN}=== Benchmark Complete ===${NC}"
    echo -e "Results saved to: ${YELLOW}$OUTPUT_DIR/$BENCHMARK_OUTPUT${NC}"
    echo -e "HTML report: ${YELLOW}$OUTPUT_DIR/$REPORT_OUTPUT${NC}"
    
    if [ "$CPU_PROFILE" = true ] || [ "$MEM_PROFILE" = true ]; then
        echo -e "Profile files saved to: ${YELLOW}$OUTPUT_DIR/${NC}"
        echo "Analyze with: go tool pprof [cpu.prof|mem.prof]"
    fi
}

# Run main function
main
