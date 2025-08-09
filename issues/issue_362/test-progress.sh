#!/bin/bash
for i in {1..100}; do
    printf "\rProgress: [%-50s] %d%%" $(head -c $((i/2)) < /dev/zero | tr '\0' '=') $i
    sleep 0.05
done
printf "\n"
echo "Complete!"
