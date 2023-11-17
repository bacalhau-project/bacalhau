
# Executor plugins 

Executors are responsible for running code in Bacalhau, and this directory is the home of the builtin Bacalhau plugins for Docker and WebAssembly jobs. 


## WIP: Building plugins 

To build the default plugins once they are implemented here, you can use the provided Makefile.

```sh 
# Build 
make build

# Cleanup
make clean 
```

This will run the build, and clean, tasks respectively for each plugin that lives in this directory.
