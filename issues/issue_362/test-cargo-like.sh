#!/bin/bash
echo "=== Simulating Cargo Watch Behavior ==="

simulate_cargo_run() {
    local iteration=$1
    
    # Clear screen like cargo watch does
    printf "\033[2J\033[H"
    
    echo "Running \`cargo run\`"
    echo "   Compiling myproject v0.1.0 (/path/to/project)"
    
    # Simulate compilation time
    sleep 1
    
    if [ $((iteration % 2)) -eq 0 ]; then
        echo "    Finished dev [unoptimized + debuginfo] target(s) in 0.45s"
        echo "     Running \`target/debug/myproject\`"
        echo ""
        echo "Hello, world! (iteration $iteration)"
    else
        echo "error[E0308]: mismatched types"
        echo " --> src/main.rs:4:13"
        echo "  |"
        echo "4 |     let x = \"hello\""
        echo "  |             ^^^^^^^ expected integer, found string"
        echo ""
        echo "error: could not compile \`myproject\` due to previous error"
    fi
    
    echo ""
    echo "Waiting for file changes... (Ctrl+C to stop)"
}

# Simulate file changes triggering rebuilds
for i in {1..5}; do
    simulate_cargo_run $i
    sleep 3
done
