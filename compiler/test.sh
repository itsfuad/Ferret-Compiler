# Navigate to compiler directory
cd compiler

# Show the structure of key modules
find internal -name "*.go" | grep -E "(import|config|registry)" | head -20

# Show key files content
cat internal/registry/config.go
cat internal/registry/lockfile.go  
cat internal/frontend/parser/imports.go