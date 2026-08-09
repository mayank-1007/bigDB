# BigDB C++ port

This directory contains a C++ recreation of the BigDB storage engine project structure and core behavior.

## Build

```bash
cmake -S . -B build
cmake --build build
```

## Run

```bash
./build/bigdb -data-dir ./data -put-key user:1 -put-value alice
./build/bigdb -data-dir ./data -get-key user:1
```
