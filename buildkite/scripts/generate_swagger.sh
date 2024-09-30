#!/bin/bash
set -euo pipefail

# Call generate_swagger.sh
generate_swagger() {
    ../scripts/generate_swagger.sh
}

upload_swagger() {
    cd docs
    buildkite-agent artifact upload "swagger.json"
}

main() {
    generate_swagger
    upload_swagger
}

main
